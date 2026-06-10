package fiscal_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	financialEntity "github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	financialRepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	salesentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AuthorizeFiscalExitUseCase struct {
	Repo          repository.FiscalRepository
	FinancialRepo financialRepo.FinancialRepository
	Auth          ports.AuthService
	// StockRepo is optional. When set, authorizing the exit posts an OUT stock
	// movement per item (warehouse resolved from the linked sales order line).
	StockRepo stockrepo.StockRepository
	// SalesOrderRepo is optional. When set together with a linked sales order,
	// authorizing the exit marks the order as invoiced and resolves the
	// warehouse for the stock write-down, and active reservations are consumed.
	SalesOrderRepo salesrepo.SalesOrderRepository
}

func (uc *AuthorizeFiscalExitUseCase) Execute(ctx context.Context, id int64) (*response.FiscalExitResponse, error) {
	if !uc.Auth.CanAuthorizeFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	exit, err := uc.Repo.GetExitByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if exit.Status != entity.ExitStatusDraft {
		return nil, fmt.Errorf("NF-e deve estar em rascunho para autorizar, status atual: %s", exit.Status)
	}

	items, err := uc.Repo.GetExitItems(ctx, id)
	if err != nil {
		return nil, err
	}

	cfg, err := uc.Repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}

	if cfg.FocusNfeToken == nil || *cfg.FocusNfeToken == "" {
		return nil, fmt.Errorf("token Focus NF-e não configurado — acesse Configurações Fiscais")
	}

	focusCli := focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente)
	focusCli.WithLogger(func(endpoint, method, reqBody, respBody string, statusCode, durationMs int) {
		_ = uc.Repo.SaveFocusLog(ctx, id, endpoint, method, reqBody, respBody, statusCode, durationMs)
	})

	ref := fmt.Sprintf("%d%d", id, time.Now().UnixNano()%1000000)
	if len(ref) > 50 {
		ref = ref[:50]
	}

	cnpjDest := ""
	if exit.CnpjDestinatario != nil {
		cnpjDest = *exit.CnpjDestinatario
	}
	razaoDest := "Destinatário"
	if exit.RazaoSocialDestinatario != nil {
		razaoDest = *exit.RazaoSocialDestinatario
	}
	ufDest := "PR"
	if exit.UFDestinatario != nil {
		ufDest = *exit.UFDestinatario
	}

	nfeItems := buildFocusItems(items, cfg)

	localDestino := 1
	if ufDest != cfg.UFEmpresa {
		localDestino = 2
	}

	consumidorFinal := 0
	indicadorIE := 1
	if exit.IEDestinatario == nil || *exit.IEDestinatario == "" || *exit.IEDestinatario == "ISENTO" {
		consumidorFinal = 1
		indicadorIE = 9
	}

	payload := focusnfe.NFEPayload{
		NaturezaOperacao:  exit.NaturezaOperacao,
		DataEmissao:       exit.DataEmissao.Format("2006-01-02T15:04:05-03:00"),
		TipoDocumento:     1,
		LocalDestino:      localDestino,
		FinalidadeEmissao: 1,
		ConsumidorFinal:   consumidorFinal,
		PresencaComprador: 4,
		Emitente: focusnfe.NFEEmitente{
			CNPJ:             cfg.CnpjEmpresa,
			Nome:             cfg.RazaoSocial,
			Logradouro:       cfg.Logradouro,
			Numero:           cfg.Numero,
			Bairro:           cfg.Bairro,
			Municipio:        cfg.Municipio,
			UF:               cfg.UFEmpresa,
			CEP:              cfg.CEP,
			Telefone:         derefStr(cfg.Telefone),
			RegimeTributario: 3,
		},
		Destinatario: focusnfe.NFEDestinatario{
			CNPJCPF:     cnpjDest,
			Nome:        razaoDest,
			UF:          ufDest,
			IndicadorIE: indicadorIE,
			IE:          exit.IEDestinatario,
		},
		Items: nfeItems,
		FormaPagamento: []focusnfe.NFEFormaPagamento{
			{FormaPagamento: "01", Valor: exit.ValorTotal},
		},
	}

	focusResp, err := focusCli.EmitirNFe(ctx, ref, payload)
	if err != nil {
		_, _ = uc.Repo.UpdateExitStatus(ctx, id, entity.ExitStatusRejected)
		return nil, fmt.Errorf("Focus NF-e: %w", err)
	}

	updated, err := uc.Repo.UpdateExitAuthorization(ctx, id, focusResp.ChaveNFe, focusResp.Protocolo, ref)
	if err != nil {
		return nil, err
	}

	// Auto-gerar Conta a Receber baseado no valor total
	if uc.FinancialRepo != nil {
		numDoc := fmt.Sprintf("NF-%d", exit.NumeroNF)
		cr := &financialEntity.ContaReceber{
			NumeroDocumento: &numDoc,
			FiscalExitID:    &id,
			DataLancamento:  time.Now(),
			DataEmissao:     exit.DataEmissao,
			DataVencimento:  exit.DataEmissao.AddDate(0, 0, 30),
			ValorBruto:      decimal.NewFromFloat(exit.ValorTotal),
			Desconto:        decimal.Zero,
			Juros:           decimal.Zero,
			Multa:           decimal.Zero,
			ValorRecebido:   decimal.Zero,
			ParcelaNumero:   1,
			ParcelaTotal:    1,
			Status:          financialEntity.ContaReceberStatusPendente,
			IsActive:        true,
			CriadoPor:       userID,
		}
		if exit.CnpjDestinatario != nil {
			// clienteID would be set if client lookup exists; for now leave nil
		}
		_, _ = uc.FinancialRepo.CreateContaReceber(ctx, cr)
	}

	// Stock write-down + sales order settlement. Best-effort: a failure here does
	// not undo an already-authorized NF-e (which lives at SEFAZ), but is surfaced
	// via the returned exit being authorized regardless.
	uc.settleStockAndOrder(ctx, exit, items, userID)

	return toFiscalExitResponse(updated), nil
}

// settleStockAndOrder posts the OUT movements for the exit items, consumes the
// sales order reservations and flags the linked order as invoiced.
func (uc *AuthorizeFiscalExitUseCase) settleStockAndOrder(
	ctx context.Context,
	exit *entity.FiscalExit,
	items []*entity.FiscalExitItem,
	userID uuid.UUID,
) {
	// Build item_code -> warehouse from the linked sales order lines.
	warehouseByItem := map[int64]int64{}
	if uc.SalesOrderRepo != nil && exit.SalesOrderCode != nil {
		if soItems, err := uc.SalesOrderRepo.ListItems(ctx, *exit.SalesOrderCode); err == nil {
			for _, si := range soItems {
				if si.WarehouseCode != nil {
					warehouseByItem[si.ItemCode] = *si.WarehouseCode
				}
			}
		}
	}

	if uc.StockRepo != nil {
		for _, it := range items {
			if it.ItemCode == nil {
				continue
			}
			wh, ok := warehouseByItem[*it.ItemCode]
			if !ok {
				continue // no resolvable warehouse; skip silently
			}
			refType := stockentity.ReferenceTypeNFExit
			refCode := exit.ID
			mov := &stockentity.StockMovement{
				ItemCode:      *it.ItemCode,
				WarehouseID:   wh,
				MovementType:  stockentity.MovementTypeOut,
				Quantity:      it.Quantity,
				UnitPrice:     it.UnitPrice,
				TotalPrice:    it.TotalPrice,
				ReferenceType: &refType,
				ReferenceCode: &refCode,
				CreatedBy:     userID,
			}
			_, _ = uc.StockRepo.CreateMovement(ctx, mov)
		}

		// Consume any active reservations tied to the sales order.
		if exit.SalesOrderCode != nil {
			if reservations, err := uc.StockRepo.ListActiveReservations(ctx); err == nil {
				for _, r := range reservations {
					if r.ReferenceType == stockentity.ReferenceTypeSalesOrder && r.ReferenceCode == *exit.SalesOrderCode {
						_ = uc.StockRepo.ConsumeReservation(ctx, r.ID)
					}
				}
			}
		}
	}

	// Flag the sales order as invoiced.
	if uc.SalesOrderRepo != nil && exit.SalesOrderCode != nil {
		_ = uc.SalesOrderRepo.ChangeStatus(ctx, *exit.SalesOrderCode, salesentity.SalesOrderStatusInvoiced)
	}
}

func buildFocusItems(items []*entity.FiscalExitItem, cfg *entity.FiscalConfig) []focusnfe.NFEItem {
	result := make([]focusnfe.NFEItem, 0, len(items))
	for i, it := range items {
		ncm := ""
		if it.Ncm != nil {
			ncm = *it.Ncm
		}
		cfop := it.Cfop
		desc := fmt.Sprintf("Produto %d", safeInt64(it.ItemCode))
		if it.Description != nil {
			desc = *it.Description
		}
		cstICMS := "00"
		if it.CstICMS != nil {
			cstICMS = *it.CstICMS
		}
		cstIPI := "50"
		if it.CstIPI != nil {
			cstIPI = *it.CstIPI
		}
		cstPIS := "01"
		if it.CstPIS != nil {
			cstPIS = *it.CstPIS
		}
		cstCOFINS := "01"
		if it.CstCOFINS != nil {
			cstCOFINS = *it.CstCOFINS
		}

		origem := 0
		if o := it.OrigemMercadoria; len(o) > 0 {
			switch o {
			case "1":
				origem = 1
			case "2":
				origem = 2
			case "3":
				origem = 3
			case "4":
				origem = 4
			case "5":
				origem = 5
			case "6":
				origem = 6
			case "7":
				origem = 7
			}
		}

		nfeIt := focusnfe.NFEItem{
			NumeroItem:                     i + 1,
			CodigoProduto:                  fmt.Sprintf("%d", safeInt64(it.ItemCode)),
			Descricao:                      desc,
			CodigoNCM:                      ncm,
			CFOP:                           cfop,
			UnidadeComercial:               "UN",
			QuantidadeComercial:            it.Quantity,
			ValorUnitarioComercial:         it.UnitPrice,
			ValorBruto:                     it.TotalPrice,
			CodigoSituacaoTributariaICMS:   cstICMS,
			ModalidadeBaseCalculoICMS:      3,
			ValorBaseCalculoICMS:           it.BaseICMS,
			AliquotaICMS:                   it.AliqICMS * 100,
			ValorICMS:                      it.ValorICMS,
			CodigoSituacaoTributariaIPI:    cstIPI,
			AliquotaIPI:                    it.AliqIPI * 100,
			ValorIPI:                       it.ValorIPI,
			CodigoSituacaoTributariaPIS:    cstPIS,
			AliquotaPIS:                    it.AliqPIS * 100,
			ValorPIS:                       it.ValorPIS,
			CodigoSituacaoTributariaCOFINS: cstCOFINS,
			AliquotaCOFINS:                 it.AliqCOFINS * 100,
			ValorCOFINS:                    it.ValorCOFINS,
			OrigemMercadoria:               origem,
		}

		// Diferimento parcial CST 51
		if cstICMS == "51" && it.ValorICMSDiferido > 0 {
			pct := cfg.IcmsDiferimentoPercentual * 100
			nfeIt.PercentualDiferimento = &pct
			nfeIt.ValorICMSDiferido = &it.ValorICMSDiferido
		}

		// Substituição Tributária (CST 10/70) — populated when the engine computed ST
		if it.ValorICMSST > 0 || it.BaseICMSST > 0 {
			modST := 4 // 4 = MVA (margem de valor agregado)
			mvaPct := it.MVA * 100
			aliqST := it.AliqICMSST * 100
			baseST := it.BaseICMSST
			valorST := it.ValorICMSST
			nfeIt.ModalidadeBaseCalculoICMSST = &modST
			nfeIt.PercentualMVAICMSST = &mvaPct
			nfeIt.BaseCalculoICMSST = &baseST
			nfeIt.AliquotaICMSST = &aliqST
			nfeIt.ValorICMSST = &valorST
		}

		result = append(result, nfeIt)
	}
	return result
}

func safeInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

var _ = uuid.UUID{}
