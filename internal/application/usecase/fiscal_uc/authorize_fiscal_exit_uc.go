package fiscal_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
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
	// CustomerRepo é opcional e resolve a condição de pagamento da nota: é o que
	// faz o título nascer parcelado como foi vendido, e com o cliente dono.
	CustomerRepo customerrepo.CustomerRepository
	// BeneficiamentoGuard confere, antes de transmitir, se a nota de beneficiamento
	// já baixou o saldo de material do cliente. Opcional: sem ele a autorização
	// segue como antes, e notas de beneficiamento passam sem essa conferência.
	//
	// A trava existe porque a alternativa é a pior possível: material do cliente que
	// saiu fiscalmente pela SEFAZ e continua aparecendo como presente no estoque de
	// terceiros. A nota é criada em rascunho e o saldo é baixado antes; aqui só se
	// confirma que isso aconteceu.
	BeneficiamentoGuard BeneficiamentoBaixaGuard
}

// BeneficiamentoBaixaGuard é a conferência da baixa de material de terceiro.
type BeneficiamentoBaixaGuard interface {
	ConferirBaixaDaNota(ctx context.Context, fiscalExitID int64) error
}

// OrigemBeneficiamento marca a saída criada pelo faturamento do beneficiamento.
const OrigemBeneficiamento = "BENEFICIAMENTO"

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
	if exit.Status != entity.ExitStatusDraft && exit.Status != entity.ExitStatusAwaitingAuthorization {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("NF-e deve estar em rascunho para autorizar, status atual: %s", exit.Status))
	}

	// Beneficiamento: o material do cliente tem de estar baixado ANTES de a nota
	// ser transmitida. Transmitir sem a baixa deixaria material que saiu
	// fiscalmente aparecendo como presente no estoque de terceiros.
	if uc.BeneficiamentoGuard != nil && exit.SourceType != nil && *exit.SourceType == OrigemBeneficiamento {
		if err := uc.BeneficiamentoGuard.ConferirBaixaDaNota(ctx, id); err != nil {
			return nil, err
		}
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
		return nil, errorsuc.NewValidationError("o token da Focus NF-e não está configurado — acesse Configurações Fiscais para informá-lo antes de autorizar a nota")
	}

	focusCli := focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente)
	focusCli.WithLogger(func(endpoint, method, reqBody, respBody string, statusCode, durationMs int) {
		_ = uc.Repo.SaveFocusLog(ctx, id, endpoint, method, reqBody, respBody, statusCode, durationMs)
	})

	ref := fmt.Sprintf("%d%d", id, time.Now().UnixNano()%1000000)
	if len(ref) > 50 {
		ref = ref[:50]
	}

	// A condição de pagamento resolvida aqui serve aos três: a duplicata da
	// NF-e, a forma de pagamento declarada à SEFAZ e o título do contas a
	// receber. Um cálculo só, um resultado só.
	plano := resolverPlanoDaNota(ctx, exit, uc.CustomerRepo, uc.SalesOrderRepo)
	payload := montarPayloadNFe(exit, items, cfg, plano)

	if exit.Status == entity.ExitStatusDraft {
		if _, err = uc.Repo.UpdateExitStatus(ctx, id, entity.ExitStatusAwaitingAuthorization); err != nil {
			return nil, fmt.Errorf("registrando NF-e como aguardando autorização: %w", err)
		}
	}

	focusResp, err := focusCli.EmitirNFe(ctx, ref, payload)
	if err != nil {
		_, _ = uc.Repo.UpdateExitStatus(ctx, id, entity.ExitStatusRejected)
		return nil, errorsuc.NewExternalServiceError("Focus NF-e", err.Error())
	}

	updated, err := uc.Repo.UpdateExitAuthorization(ctx, id, focusResp.ChaveNFe, focusResp.Protocolo, ref, focusResp.PathXML, focusResp.PathDANFE)
	if err != nil {
		return nil, err
	}

	// Títulos do contas a receber, um por parcela da condição de pagamento.
	//
	// Antes a nota gerava SEMPRE um único título vencendo em 30 dias, sem
	// cliente: uma venda em 28/56/84 entrava no financeiro como uma parcela só,
	// a entrada de 30% não existia e o título não tinha dono — então aging,
	// extrato por cliente e limite de crédito ficavam cegos.
	if uc.FinancialRepo != nil {
		uc.gerarTitulos(ctx, exit, id, plano, userID)
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

		// ⚠️ A unidade era FIXA em "UN" e o código do produto vinha de `item_code`,
		// que serializa "0" quando nulo. A NF-e de retorno do beneficiamento tem
		// linhas em KG com o código DO CLIENTE — com os valores fixos, a nota
		// descrevia outra mercadoria. A linha agora pode trazer os dois (migração
		// 000374); na falta deles o comportamento antigo é preservado, para não
		// mudar nota que já era emitida assim.
		unidade := "UN"
		if it.UnidadeComercial != nil && strings.TrimSpace(*it.UnidadeComercial) != "" {
			unidade = strings.ToUpper(strings.TrimSpace(*it.UnidadeComercial))
		}
		codigoProduto := fmt.Sprintf("%d", safeInt64(it.ItemCode))
		if it.CodigoProduto != nil && strings.TrimSpace(*it.CodigoProduto) != "" {
			codigoProduto = strings.TrimSpace(*it.CodigoProduto)
		}

		nfeIt := focusnfe.NFEItem{
			NumeroItem:                     i + 1,
			CodigoProduto:                  codigoProduto,
			Descricao:                      desc,
			CodigoNCM:                      ncm,
			CFOP:                           cfop,
			UnidadeComercial:               unidade,
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

// gerarTitulos grava um título do contas a receber por parcela da condição de
// pagamento da nota.
//
// O valor parcelado é o TOTAL da nota (produtos + IPI + ICMS-ST + frete + seguro
// − desconto), não o valor dos produtos: é esse o número que o cliente paga e o
// que o boleto tem de fechar com a NF-e.
func (uc *AuthorizeFiscalExitUseCase) gerarTitulos(
	ctx context.Context,
	exit *entity.FiscalExit,
	exitID int64,
	plano PlanoDaNota,
	userID uuid.UUID,
) {
	parcelas := plano.Parcelas
	if len(parcelas) == 0 {
		parcelas = []customerentity.ParcelaCalculada{{
			Numero: 1, Valor: decimal.NewFromFloat(exit.ValorTotal),
			Vencimento: exit.DataEmissao.AddDate(0, 0, 30), Descricao: "30 dias",
		}}
	}
	total := int32(len(parcelas))

	var clienteID *int64
	if uc.CustomerRepo != nil && exit.CustomerCode != nil {
		if cliente, err := uc.CustomerRepo.GetCustomerByCode(ctx, *exit.CustomerCode); err == nil && cliente != nil {
			id := cliente.ID
			clienteID = &id
		}
	}

	for _, p := range parcelas {
		numDoc := fmt.Sprintf("NF-%d/%d", exit.NumeroNF, p.Numero)
		if total == 1 {
			numDoc = fmt.Sprintf("NF-%d", exit.NumeroNF)
		}
		forma := formaPagamentoNFe(p)
		cr := &financialEntity.ContaReceber{
			NumeroDocumento: &numDoc,
			ClienteID:       clienteID,
			FiscalExitID:    &exitID,
			SalesOrderID:    exit.SalesOrderCode,
			DataLancamento:  time.Now(),
			DataEmissao:     exit.DataEmissao,
			DataVencimento:  p.Vencimento,
			ValorBruto:      p.Valor,
			Desconto:        decimal.Zero,
			Juros:           decimal.Zero,
			Multa:           decimal.Zero,
			ValorRecebido:   decimal.Zero,
			ParcelaNumero:   int32(p.Numero),
			ParcelaTotal:    total,
			FormaPagamento:  &forma,
			Status:          financialEntity.ContaReceberStatusPendente,
			IsActive:        true,
			CriadoPor:       userID,
		}
		_, _ = uc.FinancialRepo.CreateContaReceber(ctx, cr)
	}
}
