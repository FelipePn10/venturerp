package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

// PendenciasNFeRepository guarda os efeitos de NF-e ainda não aplicados.
type PendenciasNFeRepository interface {
	// Registrar abre a pendência (nada acontece se já houver uma aberta).
	Registrar(ctx context.Context, exitID int64, etapa string, por uuid.UUID) error
	Resolver(ctx context.Context, exitID int64, etapa string) error
	RegistrarFalha(ctx context.Context, exitID int64, etapa, motivo string) error
	DaNota(ctx context.Context, exitID int64) ([]entity.PendenciaNFe, error)
	// Vencidas: fila de todas as empresas, para o processo em segundo plano.
	Vencidas(ctx context.Context, limite int) ([]entity.PendenciaVencida, error)
}
