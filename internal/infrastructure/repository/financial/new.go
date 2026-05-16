package financial

import (
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FinancialRepositoryPG struct {
	pool *pgxpool.Pool
}

var _ repository.FinancialRepository = (*FinancialRepositoryPG)(nil)

func NewFinancialRepositoryPG(pool *pgxpool.Pool) repository.FinancialRepository {
	return &FinancialRepositoryPG{pool: pool}
}
