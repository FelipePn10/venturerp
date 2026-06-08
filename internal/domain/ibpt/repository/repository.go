package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/ibpt/entity"
)

type IBPTRepository interface {
	// BulkUpsert inserts or updates a batch of IBPT rates, returning how many
	// rows were written. Conflicts on (ncm, ex, uf, versao) update the rates.
	BulkUpsert(ctx context.Context, rates []*entity.IBPTRate) (int, error)
	// GetByNCM returns the most recent rate for an NCM in a UF (ex defaults to "0").
	GetByNCM(ctx context.Context, ncm, uf string) (*entity.IBPTRate, error)
}
