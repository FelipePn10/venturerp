package fiscal_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	financialEntity "github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	financialRepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AuthorizeFiscalExitUseCase struct {
	Repo          repository.FiscalRepository
	FinancialRepo financialRepo.FinancialRepository
	Auth          ports.AuthService
}

func (uc *AuthorizeFiscalExitUseCase) Execute(ctx context.Context, id int64) (*entity.FiscalExit, error) {
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

	nfeItems := buildFocusItems(items)

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

	return updated, nil
}

func buildFocusItems(items []*entity.FiscalExitItem) []focusnfe.NFEItem {
	result := make([]focusnfe.NFEItem, 0, len(items))
	for i, it := range items {
		ncm := ""
		if it.Ncm != nil {
			ncm = *it.Ncm
		}
		cfop := it.Cfop
		desc := fmt.Sprintf("Produto %d", it.ItemCode)
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
			AliquotaPIS:                    1.65,
			ValorPIS:                       it.ValorPIS,
			CodigoSituacaoTributariaCOFINS: cstCOFINS,
			AliquotaCOFINS:                 7.60,
			ValorCOFINS:                    it.ValorCOFINS,
			OrigemMercadoria:               origem,
		}

		// Diferimento parcial CST 51
		if cstICMS == "51" && it.ValorICMSDiferido > 0 {
			pct := 38.46
			nfeIt.PercentualDiferimento = &pct
			nfeIt.ValorICMSDiferido = &it.ValorICMSDiferido
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
