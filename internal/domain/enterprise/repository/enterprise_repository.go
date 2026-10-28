package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/enterprise/entity"
)

type EnterpriseRepository interface {
	Create(ctx context.Context, enterprise *entity.Enterprise) (*entity.Enterprise, error)
	// GetByCodeForUser e ListForUser só enxergam as empresas a que o usuário
	// está vinculado (user_enterprises): cada empresa é um mundo à parte.
	GetByCodeForUser(ctx context.Context, code int, userID uuid.UUID) (*entity.Enterprise, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]*entity.Enterprise, error)
	// Identificacao e AtualizarIdentificacao: a da empresa da sessão. A
	// alteração grava o nome da empresa e a identificação fiscal juntos.
	Identificacao(ctx context.Context) (*entity.DadosFiscais, error)
	AtualizarIdentificacao(ctx context.Context, d *entity.DadosFiscais, por uuid.UUID) error
}
