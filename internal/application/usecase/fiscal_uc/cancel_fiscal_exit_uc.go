package fiscal_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	financialRepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

type CancelFiscalExitUseCase struct {
	Repo          repository.FiscalRepository
	FinancialRepo financialRepo.FinancialRepository
	Auth          ports.AuthService
	// Estorno do beneficiamento. Nulo em ambiente que não usa o módulo; o
	// cancelamento segue funcionando como antes.
	BeneficiamentoEstorno EstornoDeBeneficiamento
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

	if len(params.Motivo) < 15 {
		return nil, fmt.Errorf("motivo do cancelamento deve ter pelo menos 15 caracteres")
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
		return nil, fmt.Errorf("apenas NF-e autorizadas podem ser canceladas, status atual: %s", exit.Status)
	}

	// 24-hour window check
	if time.Since(exit.DataEmissao) > 24*time.Hour {
		return nil, fmt.Errorf("prazo de cancelamento expirado: NF-e emitida há mais de 24 horas")
	}

	// Call Focus NF-e API
	cfg, err := uc.Repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}

	if cfg.FocusNfeToken != nil && *cfg.FocusNfeToken != "" && exit.FocusRef != nil {
		cli := focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente)
		cli.WithLogger(func(endpoint, method, reqBody, respBody string, statusCode, durationMs int) {
			_ = uc.Repo.SaveFocusLog(ctx, params.ID, endpoint, method, reqBody, respBody, statusCode, durationMs)
		})
		if _, err := cli.CancelarNFe(ctx, *exit.FocusRef, params.Motivo); err != nil {
			return nil, errorsuc.NewExternalServiceError("Focus NF-e (cancelamento)", err.Error())
		}
	}

	updated, err := uc.Repo.CancelExitWithMotivo(ctx, params.ID, params.Motivo, userID)
	if err != nil {
		return nil, err
	}

	// Revert associated Conta a Receber
	if uc.FinancialRepo != nil {
		_ = uc.FinancialRepo.CancelContasReceberByFiscalExit(ctx, params.ID)
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

	return toFiscalExitResponse(updated), nil
}
