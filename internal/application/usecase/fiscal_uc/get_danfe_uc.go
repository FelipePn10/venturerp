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

// GetDANFEUseCase returns the DANFE PDF URL and XML URL for an authorized NF-e.
// If danfe_path is already persisted in fiscal_exits it is used directly.
// Otherwise the status is refreshed from FocusNFE so the paths can be stored.
type GetDANFEUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

type DANFEResult struct {
	ExitID   int64  `json:"exit_id"`
	DanfeURL string `json:"danfe_url"`
	XMLURL   string `json:"xml_url"`
	Status   string `json:"status"`
}

func (uc *GetDANFEUseCase) Execute(ctx context.Context, exitID int64) (*DANFEResult, error) {
	if !uc.Auth.CanGetFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	exit, err := uc.Repo.GetExitByID(ctx, exitID)
	if err != nil {
		return nil, err
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
		if errLog := uc.Repo.SaveFocusLog(ctx, exitID, endpoint, method, reqBody, respBody, statusCode, durationMs); errLog != nil {
			// A chamada à Focus já aconteceu: não dá para desfazê-la, mas a falha
			// de registrar a comunicação não pode sumir.
			slog.Error("falha ao gravar o registro da comunicação com a Focus", "documento", exitID, "endpoint", endpoint, "erro", errLog)
		}
	})

	// If paths are already persisted, use them without an extra FocusNFE call.
	if exit.DanfePath != nil && *exit.DanfePath != "" {
		return &DANFEResult{
			ExitID:   exitID,
			DanfeURL: cli.DocumentURL(*exit.DanfePath),
			XMLURL: func() string {
				if exit.XmlPath != nil {
					return cli.DocumentURL(*exit.XmlPath)
				}
				return ""
			}(),
			Status: string(exit.Status),
		}, nil
	}

	// Paths not stored — refresh from FocusNFE and persist.
	if exit.FocusRef == nil {
		return nil, errorsuc.NewValidationError("esta nota não foi transmitida à SEFAZ por este sistema, então não tem DANFE")
	}

	resp, err := cli.ConsultarNFe(ctx, *exit.FocusRef)
	if err != nil {
		return nil, erroDaEmissora(ctx, "a consulta do DANFE", exitID, err)
	}
	if resp == nil || (resp.PathDANFE == "" && resp.PathXML == "") {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o DANFE ainda não está disponível: a nota está %s", situacaoDaNota(exit.Status)))
	}

	// Sem persistir na nota cancelada: gravar a autorização a devolveria para
	// AUTORIZADA só por alguém ter pedido o DANFE.
	if exit.Status != entity.ExitStatusAuthorized {
		return &DANFEResult{
			ExitID:   exitID,
			DanfeURL: cli.DocumentURL(resp.PathDANFE),
			XMLURL:   cli.DocumentURL(resp.PathXML),
			Status:   resp.Status,
		}, nil
	}

	// Persist so subsequent calls skip the network round-trip.
	if _, errAutorizacao := uc.Repo.UpdateExitAuthorization(ctx,
		exitID,
		derefStr(exit.ChaveAcesso),
		derefStr(exit.Protocolo),
		*exit.FocusRef,
		resp.PathXML,
		resp.PathDANFE,
	); errAutorizacao != nil {
		return nil, fmt.Errorf("gravando a autorização da NF-e: %w", errAutorizacao)
	}

	return &DANFEResult{
		ExitID:   exitID,
		DanfeURL: cli.DocumentURL(resp.PathDANFE),
		XMLURL:   cli.DocumentURL(resp.PathXML),
		Status:   resp.Status,
	}, nil
}
