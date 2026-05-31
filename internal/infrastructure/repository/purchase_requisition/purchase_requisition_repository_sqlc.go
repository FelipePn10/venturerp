package purchase_requisition

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PurchaseRequisitionRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func New(q *sqlc.Queries, pool *pgxpool.Pool) domainrepo.PurchaseRequisitionRepository {
	return &PurchaseRequisitionRepositorySQLC{q: q, pool: pool}
}

func (r *PurchaseRequisitionRepositorySQLC) Create(ctx context.Context, req *entity.PurchaseRequisition) (*entity.PurchaseRequisition, error) {
	row, err := r.q.CreatePurchaseRequisition(ctx, sqlc.CreatePurchaseRequisitionParams{
		Code:                  req.Code,
		EnterpriseCode:        req.EnterpriseCode,
		RequestTypeCode:       req.RequestTypeCode,
		RequesterEmployeeCode: req.RequesterEmployeeCode,
		EmissionDate:          pgutil.ToPgDate(req.EmissionDate),
		Notes:                 pgutil.ToPgTextFromPtr(req.Notes),
		CreatedBy:             pgutil.ToPgUUID(req.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating purchase requisition: %w", err)
	}
	return reqToEntity(row), nil
}

func (r *PurchaseRequisitionRepositorySQLC) GetByCode(ctx context.Context, code int64) (*entity.PurchaseRequisition, error) {
	row, err := r.q.GetPurchaseRequisitionByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("purchase requisition %d not found: %w", code, err)
	}
	return reqToEntity(row), nil
}

func (r *PurchaseRequisitionRepositorySQLC) List(ctx context.Context, onlyOpen bool) ([]*entity.PurchaseRequisition, error) {
	rows, err := r.q.ListPurchaseRequisitions(ctx, onlyOpen)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.PurchaseRequisition, 0, len(rows))
	for _, row := range rows {
		out = append(out, reqToEntity(row))
	}
	return out, nil
}

func (r *PurchaseRequisitionRepositorySQLC) NextCode(ctx context.Context) (int64, error) {
	v, err := r.q.NextPurchaseRequisitionCode(ctx)
	return int64(v), err
}

func (r *PurchaseRequisitionRepositorySQLC) AddItem(ctx context.Context, item *entity.PurchaseRequisitionItem) (*entity.PurchaseRequisitionItem, error) {
	row, err := r.q.CreatePurchaseRequisitionItem(ctx, sqlc.CreatePurchaseRequisitionItemParams{
		RequisitionCode:   item.RequisitionCode,
		Sequence:          item.Sequence,
		ItemCode:          item.ItemCode,
		Quantity:          pgutil.ToPgNumericFromFloat64(item.Quantity),
		Uom:               pgutil.ToPgTextFromPtr(item.UOM),
		CostCenterCode:    item.CostCenterCode,
		AccountingAccount: pgutil.ToPgTextFromPtr(item.AccountingAccount),
		SuggestedPrice:    pgutil.ToPgNumericFromFloat64(item.SuggestedPrice),
		DeliveryDate:      pgutil.ToPgDateFromPtr(item.DeliveryDate),
		Application:       pgutil.ToPgTextFromPtr(item.Application),
		UtilizationType:   pgutil.ToPgTextFromPtr(item.UtilizationType),
	})
	if err != nil {
		return nil, fmt.Errorf("adding requisition item: %w", err)
	}
	return itemToEntity(row), nil
}

func (r *PurchaseRequisitionRepositorySQLC) ListItems(ctx context.Context, requisitionCode int64) ([]*entity.PurchaseRequisitionItem, error) {
	rows, err := r.q.ListPurchaseRequisitionItems(ctx, requisitionCode)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.PurchaseRequisitionItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, itemToEntity(row))
	}
	return out, nil
}

func (r *PurchaseRequisitionRepositorySQLC) GetItem(ctx context.Context, id int64) (*entity.PurchaseRequisitionItem, error) {
	row, err := r.q.GetPurchaseRequisitionItem(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("requisition item %d not found: %w", id, err)
	}
	return itemToEntity(row), nil
}

func (r *PurchaseRequisitionRepositorySQLC) RegisterAttendance(ctx context.Context, itemID int64, qty float64) (*entity.PurchaseRequisitionItem, error) {
	row, err := r.q.RegisterRequisitionItemAttendance(ctx, sqlc.RegisterRequisitionItemAttendanceParams{
		ID:          itemID,
		AttendedQty: pgutil.ToPgNumericFromFloat64(qty),
	})
	if err != nil {
		return nil, fmt.Errorf("registering attendance: %w", err)
	}
	return itemToEntity(row), nil
}

func (r *PurchaseRequisitionRepositorySQLC) UpdateStatus(ctx context.Context, code int64, status string) error {
	return r.q.UpdatePurchaseRequisitionStatus(ctx, sqlc.UpdatePurchaseRequisitionStatusParams{Code: code, Status: status})
}

// ─── Mappers ──────────────────────────────────────────────────────────────────

func reqToEntity(row sqlc.PurchaseRequisition) *entity.PurchaseRequisition {
	return &entity.PurchaseRequisition{
		ID:                    row.ID,
		Code:                  row.Code,
		EnterpriseCode:        row.EnterpriseCode,
		RequestTypeCode:       row.RequestTypeCode,
		RequesterEmployeeCode: row.RequesterEmployeeCode,
		EmissionDate:          pgutil.FromPgDate(row.EmissionDate),
		Status:                entity.RequisitionStatus(row.Status),
		Notes:                 pgutil.FromPgTextPtr(row.Notes),
		IsActive:              row.IsActive,
		CreatedAt:             pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy:             pgutil.FromPgUUID(row.CreatedBy),
		UpdatedAt:             pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}

func itemToEntity(row sqlc.PurchaseRequisitionItem) *entity.PurchaseRequisitionItem {
	return &entity.PurchaseRequisitionItem{
		ID:                row.ID,
		RequisitionCode:   row.RequisitionCode,
		Sequence:          row.Sequence,
		ItemCode:          row.ItemCode,
		Quantity:          pgutil.FromPgNumericToFloat64(row.Quantity),
		AttendedQty:       pgutil.FromPgNumericToFloat64(row.AttendedQty),
		CancelledQty:      pgutil.FromPgNumericToFloat64(row.CancelledQty),
		UOM:               pgutil.FromPgTextPtr(row.Uom),
		CostCenterCode:    row.CostCenterCode,
		AccountingAccount: pgutil.FromPgTextPtr(row.AccountingAccount),
		SuggestedPrice:    pgutil.FromPgNumericToFloat64(row.SuggestedPrice),
		DeliveryDate:      pgutil.FromPgDateToPtr(row.DeliveryDate),
		Application:       pgutil.FromPgTextPtr(row.Application),
		UtilizationType:   pgutil.FromPgTextPtr(row.UtilizationType),
		Status:            entity.RequisitionStatus(row.Status),
		IsActive:          row.IsActive,
		CreatedAt:         pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}
