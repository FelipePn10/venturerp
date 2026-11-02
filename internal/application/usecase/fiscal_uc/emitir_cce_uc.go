package fiscal_uc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

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

	params.TextoCorrecao = strings.TrimSpace(params.TextoCorrecao)
	if n := len([]rune(params.TextoCorrecao)); n < 15 || n > 1000 {
		return nil, errorsuc.NewValidationError("o texto da carta de correção deve ter de 15 a 1000 caracteres (exigência da SEFAZ)")
	}

	exit, err := uc.Repo.GetExitByID(ctx, params.FiscalExitID)
	if err != nil {
		return nil, err
	}

	if exit.Status != entity.ExitStatusAuthorized {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("carta de correção só vale para nota autorizada; esta está %s", situacaoDaNota(exit.Status)))
	}

	if exit.FocusRef == nil {
		return nil, errorsuc.NewValidationError("esta nota não foi transmitida à SEFAZ por este sistema, então não há o que corrigir por carta de correção")
	}

	cfg, err := uc.Repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}

	if cfg.FocusNfeToken == nil || *cfg.FocusNfeToken == "" {
		return nil, errorsuc.NewValidationError(tokenFocusAusente)
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
		return nil, erroDaEmissora(ctx, "a carta de correção", params.FiscalExitID, err)
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
