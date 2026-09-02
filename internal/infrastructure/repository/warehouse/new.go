package warehouse

import "github.com/jackc/pgx/v5/pgxpool"

type repositoryWarehouseSQLC struct {
	pool *pgxpool.Pool
}

func NewRepositoryQuestionSQLC(pool *pgxpool.Pool) *repositoryWarehouseSQLC {
	return &repositoryWarehouseSQLC{pool: pool}
}
