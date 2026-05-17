package fiscal_uc

import (
	"context"
	"fmt"
	"time"

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
}

type CancelFiscalExitParams struct {
	ID     int64
	Motivo string
}

func (uc *CancelFiscalExitUseCase) Execute(ctx context.Context, params CancelFiscalExitParams) (*entity.FiscalExit, error) {
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
			return nil, fmt.Errorf("Focus NF-e cancelamento: %w", err)
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

	return updated, nil
}
