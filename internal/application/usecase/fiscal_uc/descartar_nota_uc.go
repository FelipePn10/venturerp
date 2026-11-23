package fiscal_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

// DescartarNotaUseCase descarta a NF-e que nunca foi autorizada (rascunho ou
// rejeitada). Sem ele, a nota errada ficava viva prendendo o romaneio e o
// saldo do pedido, e o cancelamento só aceita nota autorizada.
type DescartarNotaUseCase struct {
	Auth  ports.AuthService
	Notas NotasDescartaveis
	// Romaneios desliga a nota do romaneio e da carga.
	Romaneios interface {
		ReabrirPelaNotaCancelada(ctx context.Context, exitID int64) error
	}
	// Beneficiamento devolve ao cliente o material baixado para a nota.
	Beneficiamento interface {
		EstornarNotaCancelada(ctx context.Context, fiscalExitID int64, userID, motivo string) error
	}
}

type NotasDescartaveis interface {
	GetExitByID(ctx context.Context, id int64) (*entity.FiscalExit, error)
	// DescartarRascunho marca como cancelada a nota em rascunho/rejeitada
	// (sem chave) e devolve false se ela mudou de situação no caminho.
	DescartarRascunho(ctx context.Context, id int64, motivo string, por uuid.UUID) (bool, error)
}

func (uc *DescartarNotaUseCase) Execute(ctx context.Context, id int64, motivo string) (*response.FiscalExitResponse, error) {
	if !uc.Auth.CanCancelFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	exit, err := uc.Notas.GetExitByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if exit.Status != entity.ExitStatusDraft && exit.Status != entity.ExitStatusRejected {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("só nota em rascunho ou rejeitada pode ser descartada; esta está %s", situacaoDaNota(exit.Status)))
	}
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		motivo = "rascunho descartado"
	}
	ok, err := uc.Notas.DescartarRascunho(ctx, id, "Descartada sem autorização: "+motivo, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errorsuc.NewConflictError("a nota mudou de situação enquanto era descartada: confira o status antes de tentar de novo")
	}
	if uc.Romaneios != nil {
		if err := uc.Romaneios.ReabrirPelaNotaCancelada(ctx, id); err != nil {
			return nil, fmt.Errorf("a NF-e %d foi descartada, mas o romaneio continua ligado a ela: %w", exit.NumeroNF, err)
		}
	}
	if uc.Beneficiamento != nil && exit.SourceType != nil && *exit.SourceType == OrigemBeneficiamento {
		if err := uc.Beneficiamento.EstornarNotaCancelada(ctx, id, userID.String(), "descarte da NF-e: "+motivo); err != nil {
			return nil, fmt.Errorf("a NF-e %d foi descartada, mas o material do cliente NÃO voltou ao saldo: %w. Refaça o estorno pela tela de beneficiamento", exit.NumeroNF, err)
		}
	}
	saida, err := uc.Notas.GetExitByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toFiscalExitResponse(saida), nil
}
