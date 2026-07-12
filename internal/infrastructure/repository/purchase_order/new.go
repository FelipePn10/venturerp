package purchase_order

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type PurchaseOrderRepositorySQLC struct {
	db *pgxpool.Pool
}

func NewPurchaseOrderRepositorySQLC(pool *pgxpool.Pool) *PurchaseOrderRepositorySQLC {
	return &PurchaseOrderRepositorySQLC{db: pool}
}
