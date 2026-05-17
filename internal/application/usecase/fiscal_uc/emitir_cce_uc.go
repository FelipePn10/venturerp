package fiscal_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

type EmitirCCeUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

type EmitirCCeParams struct {
	FiscalExitID  int64
	TextoCorrecao string
}

type CCeResult struct {
	FiscalExitID  int64  `json:"fiscal_exit_id"`
	TextoCorrecao string `json:"texto_correcao"`
	Status        string `json:"status"`
	NumeroSeq     int    `json:"numero_seq"`
}

func (uc *EmitirCCeUseCase) Execute(ctx context.Context, params EmitirCCeParams) (*CCeResult, error) {
	if !uc.Auth.CanCreateFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	if len(params.TextoCorrecao) < 15 {
		return nil, fmt.Errorf("texto de correção deve ter pelo menos 15 caracteres")
	}

	exit, err := uc.Repo.GetExitByID(ctx, params.FiscalExitID)
	if err != nil {
		return nil, err
	}

	if exit.Status != entity.ExitStatusAuthorized {
		return nil, fmt.Errorf("CC-e só pode ser emitida para NF-e autorizada, status: %s", exit.Status)
	}

	if exit.FocusRef == nil {
		return nil, fmt.Errorf("NF-e sem referência Focus — não é possível emitir CC-e")
	}

	cfg, err := uc.Repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}

	if cfg.FocusNfeToken == nil || *cfg.FocusNfeToken == "" {
		return nil, fmt.Errorf("token Focus NF-e não configurado")
	}

	cli := focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente)
	cli.WithLogger(func(endpoint, method, reqBody, respBody string, statusCode, durationMs int) {
		_ = uc.Repo.SaveFocusLog(ctx, params.FiscalExitID, endpoint, method, reqBody, respBody, statusCode, durationMs)
	})

	resp, err := cli.EmitirCCe(ctx, *exit.FocusRef, params.TextoCorrecao)
	if err != nil {
		return nil, fmt.Errorf("Focus CC-e: %w", err)
	}

	userID, _ := uc.Auth.UserID(ctx)
	seq, err := uc.Repo.SaveCartaCorrecao(ctx, params.FiscalExitID, params.TextoCorrecao, *exit.FocusRef, userID)
	if err != nil {
		return nil, err
	}

	status := "enviada"
	if s, ok := resp["status"].(string); ok {
		status = s
	}

	return &CCeResult{
		FiscalExitID:  params.FiscalExitID,
		TextoCorrecao: params.TextoCorrecao,
		Status:        status,
		NumeroSeq:     seq,
	}, nil
}
