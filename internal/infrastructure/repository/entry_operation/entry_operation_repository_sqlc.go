package entry_operation

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/entry_operation/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/entry_operation/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EntryOperationRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func New(q *sqlc.Queries, pool *pgxpool.Pool) domainrepo.EntryOperationRepository {
	return &EntryOperationRepositorySQLC{q: q, pool: pool}
}

// ─── State Groups ─────────────────────────────────────────────────────────────

func (r *EntryOperationRepositorySQLC) CreateStateGroup(ctx context.Context, g *entity.StateGroup) (*entity.StateGroup, error) {
	row, err := r.q.CreateStateGroup(ctx, sqlc.CreateStateGroupParams{
		Code:        g.Code,
		Description: g.Description,
		CreatedBy:   pgutil.ToPgUUID(g.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating state group: %w", err)
	}
	return stateGroupToEntity(row), nil
}

func (r *EntryOperationRepositorySQLC) GetStateGroupByCode(ctx context.Context, code int64) (*entity.StateGroup, error) {
	row, err := r.q.GetStateGroupByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("state group %d not found: %w", code, err)
	}
	g := stateGroupToEntity(row)
	if g.UFs, err = r.ListStateGroupUFs(ctx, code); err != nil {
		return nil, err
	}
	return g, nil
}

func (r *EntryOperationRepositorySQLC) ListStateGroups(ctx context.Context) ([]*entity.StateGroup, error) {
	rows, err := r.q.ListStateGroups(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.StateGroup, 0, len(rows))
	for _, row := range rows {
		out = append(out, stateGroupToEntity(row))
	}
	return out, nil
}

func (r *EntryOperationRepositorySQLC) NextStateGroupCode(ctx context.Context) (int64, error) {
	v, err := r.q.NextStateGroupCode(ctx)
	return int64(v), err
}

func (r *EntryOperationRepositorySQLC) AddStateGroupUF(ctx context.Context, stateGroupCode int64, uf string) error {
	return r.q.AddStateGroupUF(ctx, sqlc.AddStateGroupUFParams{StateGroupCode: stateGroupCode, Uf: uf})
}

func (r *EntryOperationRepositorySQLC) ListStateGroupUFs(ctx context.Context, stateGroupCode int64) ([]string, error) {
	return r.q.ListStateGroupUFs(ctx, stateGroupCode)
}

func (r *EntryOperationRepositorySQLC) UFInGroup(ctx context.Context, stateGroupCode int64, uf string) (bool, error) {
	return r.q.UFInStateGroup(ctx, sqlc.UFInStateGroupParams{StateGroupCode: stateGroupCode, Uf: uf})
}

// ─── Entry Operation Types ────────────────────────────────────────────────────

func (r *EntryOperationRepositorySQLC) CreateEntryOperation(ctx context.Context, o *entity.EntryOperationType) (*entity.EntryOperationType, error) {
	row, err := r.q.CreateEntryOperationType(ctx, sqlc.CreateEntryOperationTypeParams{
		Code:               o.Code,
		Description:        o.Description,
		InvoiceTypeCode:    o.InvoiceTypeCode,
		NatureOperation:    o.NatureOperation,
		ClassificationType: pgutil.ToPgTextFromPtr(o.ClassificationType),
		ClassificationCode: pgutil.ToPgTextFromPtr(o.ClassificationCode),
		StateGroupCode:     o.StateGroupCode,
		SupplierTypeCode:   o.SupplierTypeCode,
		CreatedBy:          pgutil.ToPgUUID(o.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating entry operation type: %w", err)
	}
	return entryOpToEntity(row), nil
}

func (r *EntryOperationRepositorySQLC) UpdateEntryOperation(ctx context.Context, o *entity.EntryOperationType) (*entity.EntryOperationType, error) {
	row, err := r.q.UpdateEntryOperationType(ctx, sqlc.UpdateEntryOperationTypeParams{
		Code:               o.Code,
		Description:        o.Description,
		InvoiceTypeCode:    o.InvoiceTypeCode,
		NatureOperation:    o.NatureOperation,
		ClassificationType: pgutil.ToPgTextFromPtr(o.ClassificationType),
		ClassificationCode: pgutil.ToPgTextFromPtr(o.ClassificationCode),
		StateGroupCode:     o.StateGroupCode,
		SupplierTypeCode:   o.SupplierTypeCode,
		IsActive:           o.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("updating entry operation type: %w", err)
	}
	return entryOpToEntity(row), nil
}

func (r *EntryOperationRepositorySQLC) GetEntryOperationByCode(ctx context.Context, code int64) (*entity.EntryOperationType, error) {
	row, err := r.q.GetEntryOperationTypeByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("entry operation type %d not found: %w", code, err)
	}
	return entryOpToEntity(row), nil
}

func (r *EntryOperationRepositorySQLC) ListEntryOperations(ctx context.Context, onlyActive bool) ([]*entity.EntryOperationType, error) {
	rows, err := r.q.ListEntryOperationTypes(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.EntryOperationType, 0, len(rows))
	for _, row := range rows {
		out = append(out, entryOpToEntity(row))
	}
	return out, nil
}

func (r *EntryOperationRepositorySQLC) NextEntryOperationCode(ctx context.Context) (int64, error) {
	v, err := r.q.NextEntryOperationTypeCode(ctx)
	return int64(v), err
}

// ─── Mappers ──────────────────────────────────────────────────────────────────

func stateGroupToEntity(row sqlc.StateGroup) *entity.StateGroup {
	return &entity.StateGroup{
		ID:          row.ID,
		Code:        row.Code,
		Description: row.Description,
		IsActive:    row.IsActive,
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy:   pgutil.FromPgUUID(row.CreatedBy),
	}
}

func entryOpToEntity(row sqlc.EntryOperationType) *entity.EntryOperationType {
	return &entity.EntryOperationType{
		ID:                 row.ID,
		Code:               row.Code,
		Description:        row.Description,
		InvoiceTypeCode:    row.InvoiceTypeCode,
		NatureOperation:    row.NatureOperation,
		ClassificationType: pgutil.FromPgTextPtr(row.ClassificationType),
		ClassificationCode: pgutil.FromPgTextPtr(row.ClassificationCode),
		StateGroupCode:     row.StateGroupCode,
		SupplierTypeCode:   row.SupplierTypeCode,
		IsActive:           row.IsActive,
		CreatedAt:          pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy:          pgutil.FromPgUUID(row.CreatedBy),
	}
}
