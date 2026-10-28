package fiscal_uc

import (
	"context"
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	financialRepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
	"github.com/google/uuid"
)

type AuthorizeFiscalExitUseCase struct {
	// CFOPs: a autorização recusa CFOP que a SEFAZ recusaria.
	CFOPs CatalogoCFOP
	// Series: números de série das linhas, impressos no infAdProd do item.
	Series SeriesPorLinha
	// Efeitos aplica estoque, reservas, títulos e pedido como pendência
	// reprocessável (ver EfeitosNFe).
	Efeitos       *EfeitosNFe
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
	// Faturamento soma ao atendido do pedido o que a nota faturou. Com ele, o
	// pedido só vira "Faturado" quando todas as linhas foram atendidas — uma
	// nota parcial não fecha o pedido.
	Faturamento repository.SalesOrderInvoicingRepository
	// Contabil, quando presente, contabiliza a venda (receita, impostos e CMV).
	Contabil SaidaContabil
	// Devolucao efetiva a devolução de compra autorizada (estoque, abatimento
	// dos títulos, apuração e contabilidade) no lugar do faturamento da venda.
	Devolucao *DevolucaoCompraUseCase
	// DevolucaoVenda efetiva a devolução de venda quando a NF-e de entrada
	// emitida pela empresa é autorizada (no lugar dos efeitos da venda).
	DevolucaoVenda interface {
		AoAutorizar(ctx context.Context, exit *entity.FiscalExit, userID uuid.UUID) []string
	}
	// Reforma resolve alíquotas e cClassTrib da nota e guarda o retrato do grupo
	// IBS/CBS transmitido. Sem ele valem a lei e a tributação integral.
	Reforma repository.ReformaTributariaRepository
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
	// Rejeitada pode ser retransmitida depois de corrigida: a rejeição não usa
	// o número na SEFAZ (cada envio tem referência nova na Focus).
	if exit.Status != entity.ExitStatusDraft && exit.Status != entity.ExitStatusAwaitingAuthorization && exit.Status != entity.ExitStatusRejected {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("NF-e deve estar em rascunho ou rejeitada para autorizar, status atual: %s", exit.Status))
	}
	// Já enviada e sem resposta: reenviar criaria outra nota na Focus, e as duas
	// poderiam ser autorizadas. A consulta resolve (autoriza ou libera o reenvio).
	if exit.Status == entity.ExitStatusAwaitingAuthorization && exit.FocusRef != nil && strings.TrimSpace(*exit.FocusRef) != "" {
		return nil, errorsuc.NewConflictError(fmt.Sprintf("a NF-e %d já foi enviada e aguarda a resposta da SEFAZ: use \"Status\" antes de enviar de novo", id))
	}

	// Beneficiamento: o material do cliente tem de estar baixado ANTES de a nota
	// ser transmitida. Transmitir sem a baixa deixaria material que saiu
	// fiscalmente aparecendo como presente no estoque de terceiros.
	if uc.BeneficiamentoGuard != nil && exit.SourceType != nil && *exit.SourceType == OrigemBeneficiamento {
		if err := uc.BeneficiamentoGuard.ConferirBaixaDaNota(ctx, id); err != nil {
			return nil, err
		}
	}

	if uc.Devolucao != nil {
		if err := uc.Devolucao.PodeAutorizar(ctx, exit); err != nil {
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
	if err := recusarPendenciasTributarias(exit, items, cfg); err != nil {
		return nil, err
	}
	rt, err := resolverReformaDaNota(ctx, uc.Reforma, exit, items, cfg)
	if err != nil {
		return nil, err
	}

	focusCli := focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente)
	focusCli.WithLogger(func(endpoint, method, reqBody, respBody string, statusCode, durationMs int) {
		if errLog := uc.Repo.SaveFocusLog(ctx, id, endpoint, method, reqBody, respBody, statusCode, durationMs); errLog != nil {
			// A chamada à Focus já aconteceu: não dá para desfazê-la, mas a falha
			// de registrar a comunicação não pode sumir.
			slog.Error("falha ao gravar o registro da comunicação com a Focus", "documento", id, "endpoint", endpoint, "erro", errLog)
		}
	})

	ref := fmt.Sprintf("%d%d", id, time.Now().UnixNano()%1000000)
	if len(ref) > 50 {
		ref = ref[:50]
	}

	// A condição de pagamento resolvida aqui serve aos três: a duplicata da
	// NF-e, a forma de pagamento declarada à SEFAZ e o título do contas a
	// receber. Um cálculo só, um resultado só.
	plano, err := resolverPlanoDaNota(ctx, exit, uc.CustomerRepo, uc.SalesOrderRepo)
	if err != nil {
		return nil, err
	}
	payload, retrato := montarPayloadNFe(exit, items, cfg, plano, rt)
	if err := anexarSeries(ctx, uc.Series, id, items, &payload); err != nil {
		return nil, err
	}
	if uc.Devolucao != nil {
		if err := uc.Devolucao.ReferenciarItens(ctx, exit, items, &payload); err != nil {
			return nil, err
		}
	}
	if err := referenciarDevolucaoDeVenda(ctx, uc.Repo, exit, items, &payload); err != nil {
		return nil, err
	}
	if err := recusarCFOPs(ctx, uc.CFOPs, exit, items, &payload); err != nil {
		return nil, err
	}

	// Depósito de saída de cada item conferido ANTES de transmitir: autorizada,
	// a nota não volta, e a baixa de estoque precisa de depósito.
	if uc.Efeitos != nil && exit.FiscalEntryID == nil {
		if err := uc.Efeitos.PreConferirDepositos(ctx, exit, items); err != nil {
			return nil, err
		}
	}

	if exit.Status == entity.ExitStatusDraft || exit.Status == entity.ExitStatusRejected {
		if _, err = uc.Repo.UpdateExitStatus(ctx, id, entity.ExitStatusAwaitingAuthorization); err != nil {
			return nil, fmt.Errorf("registrando NF-e como aguardando autorização: %w", err)
		}
	}

	// A referência é gravada ANTES do envio: se a resposta não voltar, a nota
	// continua consultável (antes ficava sem referência e "não enviada").
	envio, guardaEnvio := uc.Repo.(registroEnvioFocus)
	if guardaEnvio {
		if err := envio.RegistrarEnvioFocus(ctx, id, ref); err != nil {
			return nil, err
		}
	}
	// O retrato do IBS/CBS também é gravado ANTES do envio: autorizada, a nota
	// precisa dizer o que transmitiu (apuração e conferência da CBS a partir de
	// 2027), e falhar aqui ainda não criou nada na SEFAZ.
	if uc.Reforma != nil && rt.Aplica {
		ibs, cbs := totaisDoRetrato(retrato)
		if err := uc.Reforma.GravarRetrato(ctx, id, retrato, ibs, cbs); err != nil {
			return nil, err
		}
	}

	focusResp, err := focusCli.EmitirNFe(ctx, ref, payload)
	if err != nil {
		var rejeicao *focusnfe.RejeicaoError
		var processando *focusnfe.EmProcessamentoError
		switch {
		case errors.As(err, &processando):
			// Não é rejeição: a nota fica aguardando e a consulta conclui.
			atual, errLer := uc.Repo.GetExitByID(ctx, id)
			if errLer != nil {
				return nil, errLer
			}
			resp := toFiscalExitResponse(atual)
			resp.Warnings = append(resp.Warnings, "a SEFAZ ainda está processando a NF-e — use \"Status\" em instantes; os títulos, o estoque e o pedido entram quando ela for autorizada")
			return resp, nil
		case errors.As(err, &rejeicao):
			if guardaEnvio {
				if errSit := envio.RegistrarRejeicao(ctx, id, rejeicao.Motivo); errSit != nil {
					return nil, fmt.Errorf("a SEFAZ rejeitou a NF-e (%s) e a rejeição não pôde ser gravada: %w", rejeicao.Motivo, errSit)
				}
			} else if _, errSit := uc.Repo.UpdateExitStatus(ctx, id, entity.ExitStatusRejected); errSit != nil {
				return nil, fmt.Errorf("a SEFAZ rejeitou a NF-e (%s) e a rejeição não pôde ser gravada: %w", rejeicao.Motivo, errSit)
			}
			// Rejeição é dado a corrigir, não falha do servidor: 422 com o motivo.
			return nil, errorsuc.NewValidationError("a SEFAZ rejeitou a NF-e: " + rejeicao.Motivo + ". Corrija e envie de novo")
		default:
			// Rede ou erro da Focus: não se sabe se a nota chegou. Ela fica
			// aguardando, com a referência gravada, para a consulta resolver.
			return nil, errorsuc.NewExternalServiceError("Focus NF-e",
				err.Error()+" — a nota ficou aguardando autorização; use \"Status\" antes de enviar de novo")
		}
	}

	// O número e a série são os que a SEFAZ autorizou: a Focus numera a nota.
	if n, convErr := strconv.ParseInt(strings.TrimSpace(focusResp.Numero), 10, 64); convErr == nil && n > 0 {
		if err := uc.Repo.UpdateExitNumbering(ctx, id, n, focusResp.Serie); err != nil {
			return nil, err
		}
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
	// Devolução de compra: não é venda — não gera título a receber do cliente
	// nem baixa pedido; o efeito é o inverso da nota de entrada.
	// Devolução de venda (NF-e de entrada própria): não é venda — não baixa
	// estoque nem gera título; efetiva a devolução.
	if updated.DevolucaoVendaID != nil {
		resp := toFiscalExitResponse(updated)
		if uc.DevolucaoVenda != nil {
			resp.Warnings = append(resp.Warnings, uc.DevolucaoVenda.AoAutorizar(ctx, updated, userID)...)
		}
		return resp, nil
	}
	if updated.FiscalEntryID != nil {
		resp := toFiscalExitResponse(updated)
		if uc.Devolucao != nil {
			resp.Warnings = append(resp.Warnings, uc.Devolucao.Efetivar(ctx, updated, userID)...)
		}
		return resp, nil
	}

	// Estoque, reservas, títulos e pedido: a nota já está autorizada na SEFAZ e
	// não volta, então os efeitos viram uma pendência registrada ANTES de rodar.
	// Se falharem, nada deles fica gravado, a pendência guarda o motivo e é
	// reprocessada (automático e pelo botão da nota). Antes cada falha era
	// ignorada e o estoque/financeiro ficava errado sem ninguém saber.
	resp := toFiscalExitResponse(updated)
	if uc.Efeitos != nil {
		if err := uc.Efeitos.Registrar(ctx, id, entity.PendenciaAutorizacao, userID); err != nil {
			resp.Warnings = append(resp.Warnings, "a nota foi autorizada, mas a pendência dos efeitos não pôde ser registrada — estoque e financeiro NÃO foram atualizados: "+err.Error())
			slog.Error("NF-e autorizada sem pendência de efeitos registrada", "nota", id, "erro", err)
			return resp, nil
		}
		if err := uc.Efeitos.Executar(ctx, id, entity.PendenciaAutorizacao, userID); err != nil {
			resp.Warnings = append(resp.Warnings, "nota autorizada; estoque, títulos e pedido ficaram PENDENTES e serão reprocessados automaticamente (há também o botão Reprocessar na nota): "+err.Error())
		}
	}
	resp.Warnings = append(resp.Warnings, contabilizarSaida(ctx, uc.Contabil, updated, uc.FinancialRepo != nil)...)
	return resp, nil
}

func buildFocusItems(items []*entity.FiscalExitItem, cfg *entity.FiscalConfig) []focusnfe.NFEItem {
	result := make([]focusnfe.NFEItem, 0, len(items))
	for i, it := range items {
		// O campo <NCM> do XML tem 8 dígitos: máscara faz a SEFAZ recusar a nota.
		ncm := ""
		if it.Ncm != nil {
			ncm = engine.NormalizarNCM(*it.Ncm)
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
			AliquotaICMS:                   percentualXML(it.AliqICMS),
			ValorICMS:                      it.ValorICMS,
			CodigoSituacaoTributariaIPI:    cstIPI,
			AliquotaIPI:                    percentualXML(it.AliqIPI),
			ValorIPI:                       it.ValorIPI,
			BaseIPI:                        it.BaseIPI,
			CodigoSituacaoTributariaPIS:    cstPIS,
			AliquotaPIS:                    percentualXML(it.AliqPIS),
			ValorPIS:                       it.ValorPIS,
			CodigoSituacaoTributariaCOFINS: cstCOFINS,
			AliquotaCOFINS:                 percentualXML(it.AliqCOFINS),
			ValorCOFINS:                    it.ValorCOFINS,
			OrigemMercadoria:               origem,
		}

		// Diferimento parcial CST 51 (a configuração guarda fração; aceita
		// porcentagem por engano, como o motor).
		// O percentual é o da linha (regra do cadastro); linha antiga, sem ele,
		// usa o da configuração, como antes.
		if cstICMS == "51" && it.ValorICMSDiferido > 0 {
			pct := it.ICMSDiferimentoPct
			if pct <= 0 {
				pct, _ = engine.FracaoDiferimento(cfg.IcmsDiferimentoPercentual).Mul(decimal.NewFromInt(100)).Float64()
			}
			nfeIt.PercentualDiferimento = &pct
			nfeIt.ValorICMSDiferido = &it.ValorICMSDiferido
		}
		if it.ICMSReducaoBasePct > 0 {
			red := it.ICMSReducaoBasePct
			nfeIt.PercentualReducaoBC = &red
		}
		if it.AliqCreditoSimples > 0 {
			aliq, valor := it.AliqCreditoSimples, it.ValorCreditoSimples
			nfeIt.AliquotaCreditoSimples, nfeIt.ValorCreditoSimples = &aliq, &valor
		}
		if it.ICMSUFDestBase > 0 {
			nfeIt.UFDest = &focusnfe.NFEItemUFDest{
				Base: it.ICMSUFDestBase, AliquotaInterna: it.ICMSUFDestAliqInterna, AliquotaInterestadual: it.ICMSAliqInterestadual,
				ValorDestino: it.ICMSUFDestValor, PercentualFCP: it.FCPUFDestPct, ValorFCP: it.FCPUFDestValor,
			}
		}
		if it.CodigoBeneficioFiscal != nil {
			nfeIt.CodigoBeneficioFiscal = *it.CodigoBeneficioFiscal
		}

		// Substituição Tributária (CST 10/70) — populated when the engine computed ST
		if it.ValorICMSST > 0 || it.BaseICMSST > 0 {
			modST := 4 // 4 = MVA (margem de valor agregado)
			mvaPct := percentualXML(it.MVA)
			aliqST := percentualXML(it.AliqICMSST)
			baseST := it.BaseICMSST
			valorST := it.ValorICMSST
			nfeIt.ModalidadeBaseCalculoICMSST = &modST
			nfeIt.PercentualMVAICMSST = &mvaPct
			nfeIt.BaseCalculoICMSST = &baseST
			nfeIt.AliquotaICMSST = &aliqST
			nfeIt.ValorICMSST = &valorST
			if it.ICMSSTReducaoBasePct > 0 {
				red := it.ICMSSTReducaoBasePct
				nfeIt.PercentualReducaoBCST = &red
			}
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

// registroEnvioFocus: repositórios que guardam a referência do envio e o motivo
// da rejeição (o FiscalRepositoryPG). Opcional para não quebrar os dublês.
type registroEnvioFocus interface {
	RegistrarEnvioFocus(ctx context.Context, id int64, ref string) error
	RegistrarRejeicao(ctx context.Context, id int64, motivo string) error
}

// percentualXML: a fração gravada (0,07) em percentual para o XML (7), com
// quatro casas. Multiplicar em float deixava 7.000000000000001 no payload.
func percentualXML(fracao float64) float64 {
	return decimal.NewFromFloat(fracao).Mul(decimal.NewFromInt(100)).Round(4).InexactFloat64()
}
