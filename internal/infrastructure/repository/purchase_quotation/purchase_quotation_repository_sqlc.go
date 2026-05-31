package purchase_quotation

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PurchaseQuotationRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func New(q *sqlc.Queries, pool *pgxpool.Pool) domainrepo.PurchaseQuotationRepository {
	return &PurchaseQuotationRepositorySQLC{q: q, pool: pool}
}

func (r *PurchaseQuotationRepositorySQLC) Create(ctx context.Context, qt *entity.PurchaseQuotation) (*entity.PurchaseQuotation, error) {
	row, err := r.q.CreatePurchaseQuotation(ctx, sqlc.CreatePurchaseQuotationParams{
		Code:           qt.Code,
		EnterpriseCode: qt.EnterpriseCode,
		Notes:          pgutil.ToPgTextFromPtr(qt.Notes),
		CreatedBy:      pgutil.ToPgUUID(qt.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating quotation: %w", err)
	}
	return quotationToEntity(row), nil
}

func (r *PurchaseQuotationRepositorySQLC) GetByCode(ctx context.Context, code int64) (*entity.PurchaseQuotation, error) {
	row, err := r.q.GetPurchaseQuotationByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("quotation %d not found: %w", code, err)
	}
	return quotationToEntity(row), nil
}

func (r *PurchaseQuotationRepositorySQLC) List(ctx context.Context, onlyOpen bool) ([]*entity.PurchaseQuotation, error) {
	rows, err := r.q.ListPurchaseQuotations(ctx, onlyOpen)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.PurchaseQuotation, 0, len(rows))
	for _, row := range rows {
		out = append(out, quotationToEntity(row))
	}
	return out, nil
}

func (r *PurchaseQuotationRepositorySQLC) NextCode(ctx context.Context) (int64, error) {
	v, err := r.q.NextPurchaseQuotationCode(ctx)
	return int64(v), err
}

func (r *PurchaseQuotationRepositorySQLC) UpdateStatus(ctx context.Context, code int64, status string) error {
	return r.q.UpdatePurchaseQuotationStatus(ctx, sqlc.UpdatePurchaseQuotationStatusParams{Code: code, Status: status})
}

func (r *PurchaseQuotationRepositorySQLC) AddItem(ctx context.Context, item *entity.PurchaseQuotationItem) (*entity.PurchaseQuotationItem, error) {
	row, err := r.q.CreatePurchaseQuotationItem(ctx, sqlc.CreatePurchaseQuotationItemParams{
		QuotationCode: item.QuotationCode,
		Sequence:      item.Sequence,
		ItemCode:      item.ItemCode,
		Quantity:      pgutil.ToPgNumericFromFloat64(item.Quantity),
		Uom:           pgutil.ToPgTextFromPtr(item.UOM),
		DeliveryDate:  pgutil.ToPgDateFromPtr(item.DeliveryDate),
		SourceType:    string(item.SourceType),
		SourceCode:    item.SourceCode,
		SourceItemID:  item.SourceItemID,
		IsConfigured:  item.IsConfigured,
	})
	if err != nil {
		return nil, fmt.Errorf("adding quotation item: %w", err)
	}
	return itemToEntity(row), nil
}

func (r *PurchaseQuotationRepositorySQLC) ListItems(ctx context.Context, quotationCode int64) ([]*entity.PurchaseQuotationItem, error) {
	rows, err := r.q.ListPurchaseQuotationItems(ctx, quotationCode)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.PurchaseQuotationItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, itemToEntity(row))
	}
	return out, nil
}

func (r *PurchaseQuotationRepositorySQLC) GetItem(ctx context.Context, itemID int64) (*entity.PurchaseQuotationItem, error) {
	row, err := r.q.GetPurchaseQuotationItem(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("quotation item %d not found: %w", itemID, err)
	}
	return itemToEntity(row), nil
}

func (r *PurchaseQuotationRepositorySQLC) AddSupplier(ctx context.Context, s *entity.PurchaseQuotationSupplier) (*entity.PurchaseQuotationSupplier, error) {
	row, err := r.q.CreatePurchaseQuotationSupplier(ctx, sqlc.CreatePurchaseQuotationSupplierParams{
		QuotationCode: s.QuotationCode,
		SupplierCode:  s.SupplierCode,
	})
	if err != nil {
		return nil, fmt.Errorf("adding quotation supplier: %w", err)
	}
	return &entity.PurchaseQuotationSupplier{ID: row.ID, QuotationCode: row.QuotationCode, SupplierCode: row.SupplierCode, InvitedAt: pgutil.FromPgTimestamptz(row.InvitedAt)}, nil
}

func (r *PurchaseQuotationRepositorySQLC) ListSuppliers(ctx context.Context, quotationCode int64) ([]*entity.PurchaseQuotationSupplier, error) {
	rows, err := r.q.ListPurchaseQuotationSuppliers(ctx, quotationCode)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.PurchaseQuotationSupplier, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.PurchaseQuotationSupplier{ID: row.ID, QuotationCode: row.QuotationCode, SupplierCode: row.SupplierCode, InvitedAt: pgutil.FromPgTimestamptz(row.InvitedAt)})
	}
	return out, nil
}

func (r *PurchaseQuotationRepositorySQLC) UpsertPrice(ctx context.Context, p *entity.PurchaseQuotationPrice) (*entity.PurchaseQuotationPrice, error) {
	row, err := r.q.UpsertPurchaseQuotationPrice(ctx, sqlc.UpsertPurchaseQuotationPriceParams{
		QuotationItemID: p.QuotationItemID,
		SupplierCode:    p.SupplierCode,
		UnitPrice:       pgutil.ToPgNumericFromFloat64(p.UnitPrice),
		LeadTimeDays:    p.LeadTimeDays,
		PaymentTermCode: p.PaymentTermCode,
		Notes:           pgutil.ToPgTextFromPtr(p.Notes),
	})
	if err != nil {
		return nil, fmt.Errorf("upserting quotation price: %w", err)
	}
	return priceToEntity(row), nil
}

func (r *PurchaseQuotationRepositorySQLC) ListPricesByItem(ctx context.Context, quotationItemID int64) ([]*entity.PurchaseQuotationPrice, error) {
	rows, err := r.q.ListPurchaseQuotationPricesByItem(ctx, quotationItemID)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.PurchaseQuotationPrice, 0, len(rows))
	for _, row := range rows {
		out = append(out, priceToEntity(row))
	}
	return out, nil
}

func (r *PurchaseQuotationRepositorySQLC) ListSelectedPrices(ctx context.Context, quotationCode int64) ([]*entity.PurchaseQuotationPrice, error) {
	rows, err := r.q.ListSelectedQuotationPrices(ctx, quotationCode)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.PurchaseQuotationPrice, 0, len(rows))
	for _, row := range rows {
		out = append(out, priceToEntity(row))
	}
	return out, nil
}

// SelectPrice clears the selection for the item and selects the given price,
// atomically.
func (r *PurchaseQuotationRepositorySQLC) SelectPrice(ctx context.Context, priceID int64) (*entity.PurchaseQuotationPrice, error) {
	price, err := r.q.GetPurchaseQuotationPrice(ctx, priceID)
	if err != nil {
		return nil, fmt.Errorf("quotation price %d not found: %w", priceID, err)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := r.q.WithTx(tx)
	if err := qtx.ClearSelectionForItem(ctx, price.QuotationItemID); err != nil {
		return nil, fmt.Errorf("clearing selection: %w", err)
	}
	row, err := qtx.SetPriceSelected(ctx, priceID)
	if err != nil {
		return nil, fmt.Errorf("selecting price: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return priceToEntity(row), nil
}

// ─── Mappers ──────────────────────────────────────────────────────────────────

func quotationToEntity(row sqlc.PurchaseQuotation) *entity.PurchaseQuotation {
	return &entity.PurchaseQuotation{
		ID:             row.ID,
		Code:           row.Code,
		EnterpriseCode: row.EnterpriseCode,
		Status:         entity.QuotationStatus(row.Status),
		EmissionDate:   pgutil.FromPgDate(row.EmissionDate),
		Notes:          pgutil.FromPgTextPtr(row.Notes),
		IsActive:       row.IsActive,
		CreatedAt:      pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy:      pgutil.FromPgUUID(row.CreatedBy),
		UpdatedAt:      pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}

func itemToEntity(row sqlc.PurchaseQuotationItem) *entity.PurchaseQuotationItem {
	return &entity.PurchaseQuotationItem{
		ID:            row.ID,
		QuotationCode: row.QuotationCode,
		Sequence:      row.Sequence,
		ItemCode:      row.ItemCode,
		Quantity:      pgutil.FromPgNumericToFloat64(row.Quantity),
		UOM:           pgutil.FromPgTextPtr(row.Uom),
		DeliveryDate:  pgutil.FromPgDateToPtr(row.DeliveryDate),
		SourceType:    entity.QuotationSourceType(row.SourceType),
		SourceCode:    row.SourceCode,
		SourceItemID:  row.SourceItemID,
		IsConfigured:  row.IsConfigured,
		CreatedAt:     pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func priceToEntity(row sqlc.PurchaseQuotationPrice) *entity.PurchaseQuotationPrice {
	return &entity.PurchaseQuotationPrice{
		ID:              row.ID,
		QuotationItemID: row.QuotationItemID,
		SupplierCode:    row.SupplierCode,
		UnitPrice:       pgutil.FromPgNumericToFloat64(row.UnitPrice),
		LeadTimeDays:    row.LeadTimeDays,
		PaymentTermCode: row.PaymentTermCode,
		Notes:           pgutil.FromPgTextPtr(row.Notes),
		IsSelected:      row.IsSelected,
		CreatedAt:       pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}
