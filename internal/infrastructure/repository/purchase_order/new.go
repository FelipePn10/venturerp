package purchase_order

import (
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PurchaseOrderRepositorySQLC struct {
	db *pgxpool.Pool
}

func NewPurchaseOrderRepositorySQLC(pool *pgxpool.Pool) repository.PurchaseOrderRepository {
	return &PurchaseOrderRepositorySQLC{db: pool}
}
