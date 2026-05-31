package purchase_price

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/purchase_price/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_price/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PurchasePriceRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func New(q *sqlc.Queries, pool *pgxpool.Pool) domainrepo.PurchasePriceRepository {
	return &PurchasePriceRepositorySQLC{q: q, pool: pool}
}

func (r *PurchasePriceRepositorySQLC) CreateTable(ctx context.Context, t *entity.PurchasePriceTable) (*entity.PurchasePriceTable, error) {
	row, err := r.q.CreatePurchasePriceTable(ctx, sqlc.CreatePurchasePriceTableParams{
		Code:          t.Code,
		Description:   t.Description,
		CurrencyCode:  t.CurrencyCode,
		ValidityStart: pgutil.ToPgDateFromPtr(t.ValidityStart),
		ValidityEnd:   pgutil.ToPgDateFromPtr(t.ValidityEnd),
		CreatedBy:     pgutil.ToPgUUID(t.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating purchase price table: %w", err)
	}
	return tableToEntity(row), nil
}

func (r *PurchasePriceRepositorySQLC) UpdateTable(ctx context.Context, t *entity.PurchasePriceTable) (*entity.PurchasePriceTable, error) {
	row, err := r.q.UpdatePurchasePriceTable(ctx, sqlc.UpdatePurchasePriceTableParams{
		Code:          t.Code,
		Description:   t.Description,
		CurrencyCode:  t.CurrencyCode,
		ValidityStart: pgutil.ToPgDateFromPtr(t.ValidityStart),
		ValidityEnd:   pgutil.ToPgDateFromPtr(t.ValidityEnd),
		IsActive:      t.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("updating purchase price table: %w", err)
	}
	return tableToEntity(row), nil
}

func (r *PurchasePriceRepositorySQLC) GetTableByCode(ctx context.Context, code int64) (*entity.PurchasePriceTable, error) {
	row, err := r.q.GetPurchasePriceTableByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("purchase price table %d not found: %w", code, err)
	}
	return tableToEntity(row), nil
}

func (r *PurchasePriceRepositorySQLC) ListTables(ctx context.Context, onlyActive bool) ([]*entity.PurchasePriceTable, error) {
	rows, err := r.q.ListPurchasePriceTables(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.PurchasePriceTable, 0, len(rows))
	for _, row := range rows {
		out = append(out, tableToEntity(row))
	}
	return out, nil
}

func (r *PurchasePriceRepositorySQLC) NextTableCode(ctx context.Context) (int64, error) {
	v, err := r.q.NextPurchasePriceTableCode(ctx)
	return int64(v), err
}

func (r *PurchasePriceRepositorySQLC) AddItem(ctx context.Context, item *entity.PurchasePriceTableItem) (*entity.PurchasePriceTableItem, error) {
	row, err := r.q.CreatePurchasePriceTableItem(ctx, sqlc.CreatePurchasePriceTableItemParams{
		TableID:      item.TableID,
		ItemCode:     item.ItemCode,
		SupplierCode: item.SupplierCode,
		Uom:          pgutil.ToPgTextFromPtr(item.UOM),
		Price:        pgutil.ToPgNumericFromFloat64(item.Price),
		MinQty:       pgutil.ToPgNumericFromFloat64(item.MinQty),
	})
	if err != nil {
		return nil, fmt.Errorf("adding purchase price item: %w", err)
	}
	return itemToEntity(row), nil
}

func (r *PurchasePriceRepositorySQLC) ListItems(ctx context.Context, tableID int64) ([]*entity.PurchasePriceTableItem, error) {
	rows, err := r.q.ListPurchasePriceTableItems(ctx, tableID)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.PurchasePriceTableItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, itemToEntity(row))
	}
	return out, nil
}

func (r *PurchasePriceRepositorySQLC) DeleteItem(ctx context.Context, id int64) error {
	return r.q.DeletePurchasePriceTableItem(ctx, id)
}

func (r *PurchasePriceRepositorySQLC) GetItemPrice(ctx context.Context, tableCode, itemCode int64, supplierCode *int64) (*entity.PurchasePriceTableItem, error) {
	row, err := r.q.GetPurchasePrice(ctx, sqlc.GetPurchasePriceParams{
		Code:         tableCode,
		ItemCode:     itemCode,
		SupplierCode: supplierCode,
	})
	if err != nil {
		return nil, err
	}
	return itemToEntity(row), nil
}

// ─── Mappers ──────────────────────────────────────────────────────────────────

func tableToEntity(row sqlc.PurchasePriceTable) *entity.PurchasePriceTable {
	return &entity.PurchasePriceTable{
		ID:            row.ID,
		Code:          row.Code,
		Description:   row.Description,
		CurrencyCode:  row.CurrencyCode,
		ValidityStart: pgutil.FromPgDateToPtr(row.ValidityStart),
		ValidityEnd:   pgutil.FromPgDateToPtr(row.ValidityEnd),
		IsActive:      row.IsActive,
		CreatedAt:     pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy:     pgutil.FromPgUUID(row.CreatedBy),
		UpdatedAt:     pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}

func itemToEntity(row sqlc.PurchasePriceTableItem) *entity.PurchasePriceTableItem {
	return &entity.PurchasePriceTableItem{
		ID:           row.ID,
		TableID:      row.TableID,
		ItemCode:     row.ItemCode,
		SupplierCode: row.SupplierCode,
		UOM:          pgutil.FromPgTextPtr(row.Uom),
		Price:        pgutil.FromPgNumericToFloat64(row.Price),
		MinQty:       pgutil.FromPgNumericToFloat64(row.MinQty),
		IsActive:     row.IsActive,
		CreatedAt:    pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}
