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

type ConsultarNFeUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

type ConsultarNFeResult struct {
	ExitID    int64  `json:"exit_id"`
	FocusRef  string `json:"focus_ref"`
	Status    string `json:"status"`
	ChaveNFe  string `json:"chave_nfe,omitempty"`
	Protocolo string `json:"protocolo,omitempty"`
	Motivo    string `json:"motivo,omitempty"`
	// Document URLs (DANFE PDF and XML) served by Focus NF-e once authorized.
	DanfeURL string `json:"danfe_url,omitempty"`
	XMLURL   string `json:"xml_url,omitempty"`
}

func (uc *ConsultarNFeUseCase) Execute(ctx context.Context, exitID int64) (*ConsultarNFeResult, error) {
	if !uc.Auth.CanGetFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	exit, err := uc.Repo.GetExitByID(ctx, exitID)
	if err != nil {
		return nil, err
	}

	if exit.FocusRef == nil {
		return nil, fmt.Errorf("NF-e %d não possui referência Focus — não foi enviada para autorização", exitID)
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
		_ = uc.Repo.SaveFocusLog(ctx, exitID, endpoint, method, reqBody, respBody, statusCode, durationMs)
	})

	resp, err := cli.ConsultarNFe(ctx, *exit.FocusRef)
	if err != nil {
		return nil, fmt.Errorf("consultando NF-e no Focus: %w", err)
	}

	// Sync local status if Focus shows a terminal state
	if resp != nil {
		switch resp.Status {
		case "autorizado":
			if exit.Status != entity.ExitStatusAuthorized {
				_, _ = uc.Repo.UpdateExitStatus(ctx, exitID, entity.ExitStatusAuthorized)
			}
		case "cancelado":
			if exit.Status != entity.ExitStatusCancelled {
				_, _ = uc.Repo.UpdateExitStatus(ctx, exitID, entity.ExitStatusCancelled)
			}
		case "rejeitado", "erro_autorizacao":
			if exit.Status != entity.ExitStatusRejected {
				_, _ = uc.Repo.UpdateExitStatus(ctx, exitID, entity.ExitStatusRejected)
			}
		}
	}

	result := &ConsultarNFeResult{
		ExitID:   exitID,
		FocusRef: *exit.FocusRef,
	}
	if resp != nil {
		result.Status = resp.Status
		result.ChaveNFe = resp.ChaveNFe
		result.Protocolo = resp.Protocolo
		result.DanfeURL = cli.DocumentURL(resp.PathDANFE)
		result.XMLURL = cli.DocumentURL(resp.PathXML)
	}
	return result, nil
}

type ListCartasCorrecaoUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *ListCartasCorrecaoUseCase) Execute(ctx context.Context, exitID int64) ([]*entity.CartaCorrecao, error) {
	if !uc.Auth.CanGetFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListCartasCorrecao(ctx, exitID)
}
