package fiscal_uc

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

type ConsultarNFeUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
	// Efeitos: quando a consulta descobre a nota autorizada (a resposta da
	// emissão se perdeu), estoque, títulos e pedido entram por aqui — antes só a
	// situação mudava e os efeitos nunca aconteciam.
	Efeitos *EfeitosNFe
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
	// Avisos: efeitos que ficaram pendentes (reprocessados automaticamente).
	Avisos []string `json:"avisos,omitempty"`
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
		return nil, errorsuc.NewValidationError(fmt.Sprintf("NF-e %d não possui referência Focus — não foi enviada para autorização", exitID))
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
		if errLog := uc.Repo.SaveFocusLog(ctx, exitID, endpoint, method, reqBody, respBody, statusCode, durationMs); errLog != nil {
			// A chamada à Focus já aconteceu: não dá para desfazê-la, mas a falha
			// de registrar a comunicação não pode sumir.
			slog.Error("falha ao gravar o registro da comunicação com a Focus", "documento", exitID, "endpoint", endpoint, "erro", errLog)
		}
	})

	resp, err := cli.ConsultarNFe(ctx, *exit.FocusRef)
	if err != nil {
		return nil, fmt.Errorf("consultando NF-e no Focus: %w", err)
	}

	var avisos []string
	// Sync local status if Focus shows a terminal state
	if resp != nil {
		switch resp.Status {
		case "autorizado":
			if exit.Status != entity.ExitStatusAuthorized {
				if _, errSituacao := uc.Repo.UpdateExitStatus(ctx, exitID, entity.ExitStatusAuthorized); errSituacao != nil {
					return nil, fmt.Errorf("gravando a situação autorizada da NF-e: %w", errSituacao)
				}
				if uc.Efeitos != nil && exit.FiscalEntryID == nil {
					avisos = append(avisos, uc.aplicarEfeitos(ctx, exitID)...)
				}
			}
		case "cancelado":
			if exit.Status != entity.ExitStatusCancelled {
				if _, errSituacao := uc.Repo.UpdateExitStatus(ctx, exitID, entity.ExitStatusCancelled); errSituacao != nil {
					return nil, fmt.Errorf("gravando a situação cancelada da NF-e: %w", errSituacao)
				}
			}
		case "rejeitado", "erro_autorizacao":
			if exit.Status != entity.ExitStatusRejected {
				if _, errSituacao := uc.Repo.UpdateExitStatus(ctx, exitID, entity.ExitStatusRejected); errSituacao != nil {
					return nil, fmt.Errorf("gravando a situação rejeitada da NF-e: %w", errSituacao)
				}
			}
		}
	}

	result := &ConsultarNFeResult{
		Avisos:   avisos,
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

func (uc *ListCartasCorrecaoUseCase) Execute(ctx context.Context, exitID int64) ([]*response.CartaCorrecaoResponse, error) {
	if !uc.Auth.CanGetFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListCartasCorrecao(ctx, exitID)
	if err != nil {
		return nil, err
	}
	return toCartaCorrecaoResponses(list), nil
}

// aplicarEfeitos registra e roda a pendência da autorização descoberta pela
// consulta; o que falhar volta como aviso (a pendência é reprocessada).
func (uc *ConsultarNFeUseCase) aplicarEfeitos(ctx context.Context, exitID int64) []string {
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return []string{"a nota está autorizada, mas os efeitos não puderam ser registrados sem o usuário da sessão"}
	}
	if err := uc.Efeitos.Registrar(ctx, exitID, entity.PendenciaAutorizacao, userID); err != nil {
		slog.Error("NF-e autorizada (consulta) sem pendência de efeitos registrada", "nota", exitID, "erro", err)
		return []string{"a nota está autorizada, mas a pendência dos efeitos não pôde ser registrada — estoque e financeiro NÃO foram atualizados: " + err.Error()}
	}
	if err := uc.Efeitos.Executar(ctx, exitID, entity.PendenciaAutorizacao, userID); err != nil {
		return []string{"nota autorizada; estoque, títulos e pedido ficaram PENDENTES e serão reprocessados automaticamente: " + err.Error()}
	}
	return nil
}
