package fiscal_uc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	financialRepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	salesentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

type CancelFiscalExitUseCase struct {
	// Efeitos cancela títulos e devolve o pedido como pendência reprocessável.
	Efeitos       *EfeitosNFe
	Repo          repository.FiscalRepository
	FinancialRepo financialRepo.FinancialRepository
	Auth          ports.AuthService
	// Estorno do beneficiamento. Nulo em ambiente que não usa o módulo; o
	// cancelamento segue funcionando como antes.
	BeneficiamentoEstorno EstornoDeBeneficiamento
	// Faturamento devolve ao pedido de venda o que a nota cancelada havia
	// atendido; o pedido faturado volta a ficar em aberto para refaturar.
	Faturamento repository.SalesOrderInvoicingRepository
	Pedidos     salesrepo.SalesOrderRepository
	// Contabil devolve o estoque baixado e estorna a contabilização da venda.
	Contabil SaidaContabil
	// Devolucao desfaz a devolução de compra cancelada.
	Devolucao *DevolucaoCompraUseCase
	// DevolucaoVenda recusa cancelar a NF-e de entrada de devolução de venda já
	// efetivada no estoque e no financeiro.
	DevolucaoVenda interface {
		PodeCancelarNF(ctx context.Context, exit *entity.FiscalExit) error
	}
}

// EstornoDeBeneficiamento devolve ao saldo do cliente o que a nota cancelada
// baixou. Interface aqui, implementação no módulo de material de terceiro: o
// fiscal não precisa conhecer o razão do beneficiamento, só que existe algo a
// desfazer. Tem de ser idempotente — o cancelamento pode ser repetido.
type EstornoDeBeneficiamento interface {
	EstornarNotaCancelada(ctx context.Context, fiscalExitID int64, usuario, motivo string) error
}

type CancelFiscalExitParams struct {
	ID     int64
	Motivo string
}

func (uc *CancelFiscalExitUseCase) Execute(ctx context.Context, params CancelFiscalExitParams) (*response.FiscalExitResponse, error) {
	if !uc.Auth.CanCancelFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	params.Motivo = strings.TrimSpace(params.Motivo)
	if n := len([]rune(params.Motivo)); n < 15 || n > 255 {
		return nil, errorsuc.NewValidationError("a justificativa do cancelamento deve ter de 15 a 255 caracteres (exigência da SEFAZ)")
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	exit, err := uc.Repo.GetExitByID(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	if exit.Status != entity.ExitStatusAuthorized {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("apenas NF-e autorizadas podem ser canceladas, status atual: %s", exit.Status))
	}

	// 24-hour window check
	if time.Since(exit.DataEmissao) > 24*time.Hour {
		return nil, errorsuc.NewValidationError("prazo de cancelamento expirado: NF-e emitida há mais de 24 horas")
	}

	// Devolução que já mexeu em título pago depois não pode ser cancelada na
	// SEFAZ: a checagem vem antes, para a nota não ficar cancelada lá com o
	// financeiro valendo aqui.
	if uc.Devolucao != nil {
		if err := uc.Devolucao.PodeDesfazer(ctx, exit); err != nil {
			return nil, err
		}
	}
	if uc.DevolucaoVenda != nil {
		if err := uc.DevolucaoVenda.PodeCancelarNF(ctx, exit); err != nil {
			return nil, err
		}
	}

	// Call Focus NF-e API
	cfg, err := uc.Repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}

	if cfg.FocusNfeToken != nil && *cfg.FocusNfeToken != "" && exit.FocusRef != nil {
		cli := focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente)
		cli.WithLogger(func(endpoint, method, reqBody, respBody string, statusCode, durationMs int) {
			if errLog := uc.Repo.SaveFocusLog(ctx, params.ID, endpoint, method, reqBody, respBody, statusCode, durationMs); errLog != nil {
				// A chamada à Focus já aconteceu: não dá para desfazê-la, mas a falha
				// de registrar a comunicação não pode sumir.
				slog.Error("falha ao gravar o registro da comunicação com a Focus", "documento", params.ID, "endpoint", endpoint, "erro", errLog)
			}
		})
		if _, err := cli.CancelarNFe(ctx, *exit.FocusRef, params.Motivo); err != nil {
			return nil, errorsuc.NewExternalServiceError("Focus NF-e (cancelamento)", err.Error())
		}
	}

	updated, err := uc.Repo.CancelExitWithMotivo(ctx, params.ID, params.Motivo, userID)
	if err != nil {
		return nil, err
	}

	// Títulos e pedido: a nota já está cancelada na SEFAZ, então viram pendência
	// (como na autorização) — se falharem, nada deles fica pela metade e o
	// reprocesso automático/manual conclui. Antes a falha era ignorada e o título
	// de uma nota cancelada continuava a receber.
	var avisosEfeitos []string
	if uc.Efeitos != nil && exit.FiscalEntryID == nil {
		if err := uc.Efeitos.Registrar(ctx, params.ID, entity.PendenciaCancelamento, userID); err != nil {
			slog.Error("NF-e cancelada sem pendência de efeitos registrada", "nota", params.ID, "erro", err)
			avisosEfeitos = append(avisosEfeitos, "a nota foi cancelada, mas a pendência dos títulos/pedido não pôde ser registrada — títulos e pedido NÃO foram atualizados: "+err.Error())
		} else if err := uc.Efeitos.Executar(ctx, params.ID, entity.PendenciaCancelamento, userID); err != nil {
			avisosEfeitos = append(avisosEfeitos, "nota cancelada; o cancelamento dos títulos e a volta do pedido ficaram PENDENTES e serão reprocessados automaticamente: "+err.Error())
		}
	} else if uc.FinancialRepo != nil {
		if err := uc.FinancialRepo.CancelContasReceberByFiscalExit(ctx, params.ID); err != nil {
			return nil, fmt.Errorf("a NF-e %d foi cancelada, mas os títulos a receber não foram cancelados: %w", exit.NumeroNF, err)
		}
	}
	// A mercadoria não saiu: o estoque volta pelo mesmo custo e a venda é
	// estornada na contabilidade.
	var avisos []string
	if exit.FiscalEntryID != nil && uc.Devolucao != nil {
		avisos = uc.Devolucao.Desfazer(ctx, exit, userID)
	} else {
		avisos = desfazerSaida(ctx, uc.Contabil, exit, userID)
	}

	// O que a nota atendeu do pedido volta a ficar pendente. Sem isso o pedido
	// continuaria "faturado" com a nota cancelada e não daria para refaturar.
	// (Com Efeitos configurado isso já foi feito acima, na pendência.)
	if uc.Efeitos == nil && uc.Faturamento != nil && exit.SalesOrderCode != nil {
		ligada, _, err := uc.Faturamento.RegistrarFaturamento(ctx, params.ID, true)
		if err != nil {
			return nil, fmt.Errorf("a NF-e %d foi cancelada, mas o pedido %d não voltou a ficar pendente: %w",
				exit.NumeroNF, *exit.SalesOrderCode, err)
		}
		if ligada && uc.Pedidos != nil {
			pedido, err := uc.Pedidos.GetByCode(ctx, *exit.SalesOrderCode)
			if err != nil {
				return nil, fmt.Errorf("a NF-e %d foi cancelada, mas o pedido %d não pôde ser lido: %w", exit.NumeroNF, *exit.SalesOrderCode, err)
			}
			if pedido != nil && pedido.Status == salesentity.SalesOrderStatusInvoiced {
				if err := uc.Pedidos.ChangeStatus(ctx, *exit.SalesOrderCode, salesentity.SalesOrderStatusOrder); err != nil {
					return nil, fmt.Errorf("a NF-e %d foi cancelada, mas o pedido %d não voltou para pedido: %w", exit.NumeroNF, *exit.SalesOrderCode, err)
				}
			}
		}
	}

	// Estorno do material de terceiro, DEPOIS do cancelamento: cancelar na SEFAZ
	// pode falhar, e devolver saldo de uma nota que continua válida seria inventar
	// material no pátio. A falha aqui não é engolida — se o saldo do cliente não
	// voltar, alguém precisa saber, porque o sistema estaria afirmando que o
	// material voltou ao cliente. O estorno é idempotente e a mensagem diz como
	// concluir.
	if uc.BeneficiamentoEstorno != nil {
		motivoEstorno := fmt.Sprintf("cancelamento da NF-e %d: %s", exit.NumeroNF, params.Motivo)
		if err := uc.BeneficiamentoEstorno.EstornarNotaCancelada(ctx, params.ID, userID.String(), motivoEstorno); err != nil {
			return nil, fmt.Errorf(
				"a NF-e %d foi cancelada, mas o saldo de material de terceiro NÃO foi devolvido ao cliente: %w. "+
					"Refaça o estorno pela tela de beneficiamento — ele não devolve duas vezes",
				exit.NumeroNF, err)
		}
	}

	resp := toFiscalExitResponse(updated)
	resp.Warnings = append(resp.Warnings, avisosEfeitos...)
	resp.Warnings = append(resp.Warnings, avisos...)
	return resp, nil
}
