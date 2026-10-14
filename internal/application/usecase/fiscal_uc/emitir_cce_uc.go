package fiscal_uc

import (
	"context"
	"fmt"
	"log/slog"

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
		return nil, errorsuc.NewValidationError("texto de correção deve ter pelo menos 15 caracteres")
	}

	exit, err := uc.Repo.GetExitByID(ctx, params.FiscalExitID)
	if err != nil {
		return nil, err
	}

	if exit.Status != entity.ExitStatusAuthorized {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("CC-e só pode ser emitida para NF-e autorizada, status: %s", exit.Status))
	}

	if exit.FocusRef == nil {
		return nil, errorsuc.NewValidationError("esta NF-e não foi autorizada pela Focus NF-e, então não há o que corrigir por carta de correção")
	}

	cfg, err := uc.Repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}

	if cfg.FocusNfeToken == nil || *cfg.FocusNfeToken == "" {
		return nil, errorsuc.NewValidationError("o token da Focus NF-e não está configurado — acesse Configurações Fiscais")
	}

	cli := focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente)
	cli.WithLogger(func(endpoint, method, reqBody, respBody string, statusCode, durationMs int) {
		if errLog := uc.Repo.SaveFocusLog(ctx, params.FiscalExitID, endpoint, method, reqBody, respBody, statusCode, durationMs); errLog != nil {
			// A chamada à Focus já aconteceu: não dá para desfazê-la, mas a falha
			// de registrar a comunicação não pode sumir.
			slog.Error("falha ao gravar o registro da comunicação com a Focus", "documento", params.FiscalExitID, "endpoint", endpoint, "erro", errLog)
		}
	})

	resp, err := cli.EmitirCCe(ctx, *exit.FocusRef, params.TextoCorrecao)
	if err != nil {
		return nil, errorsuc.NewExternalServiceError("Focus CC-e", err.Error())
	}

	userID, errUsuario := uc.Auth.UserID(ctx)
	if errUsuario != nil {
		return nil, errorsuc.ErrUnauthorized
	}
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
