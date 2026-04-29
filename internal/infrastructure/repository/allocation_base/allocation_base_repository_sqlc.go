package allocation_base

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/allocation_base/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *AllocationBaseRepositorySQLC) Create(ctx context.Context, ab *entity.AllocationBase) (*entity.AllocationBase, error) {
	row, err := r.q.CreateAllocationBase(ctx, sqlc.CreateAllocationBaseParams{
		Code:        ab.Code,
		Description: ab.Description,
		Period:      ab.Period,
		Observation: toNullString(ab.Observation),
		CreatedBy:   ab.CreatedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("creating allocation base: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *AllocationBaseRepositorySQLC) AddItem(ctx context.Context, item *entity.AllocationBaseItem) (*entity.AllocationBaseItem, error) {
	row, err := r.q.AddAllocationBaseItem(ctx, sqlc.AddAllocationBaseItemParams{
		AllocationBaseCode: item.AllocationBaseCode,
		CostCenterCode:     item.CostCenterCode,
		Amount:             item.Amount,
		Percentage:         item.Percentage,
	})
	if err != nil {
		return nil, fmt.Errorf("adding allocation base item: %w", err)
	}

	return itemRowToEntity(row), nil
}

func (r *AllocationBaseRepositorySQLC) GetByCode(ctx context.Context, code int32) (*entity.AllocationBase, error) {
	row, err := r.q.GetAllocationBaseByCode(ctx, code)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("allocation base %d not found", code)
		}
		return nil, fmt.Errorf("fetching allocation base: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *AllocationBaseRepositorySQLC) GetItems(ctx context.Context, baseCode int32) ([]*entity.AllocationBaseItem, error) {
	rows, err := r.q.GetAllocationBaseItems(ctx, baseCode)
	if err != nil {
		return nil, fmt.Errorf("fetching allocation base items: %w", err)
	}

	return itemsToEntities(rows), nil
}

func (r *AllocationBaseRepositorySQLC) List(ctx context.Context) ([]*entity.AllocationBase, error) {
	rows, err := r.q.ListAllocationBases(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing allocation bases: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *AllocationBaseRepositorySQLC) Delete(ctx context.Context, code int32) error {
	if err := r.q.DeleteAllocationBaseItems(ctx, code); err != nil {
		return fmt.Errorf("deleting items: %w", err)
	}

	if err := r.q.DeleteAllocationBase(ctx, code); err != nil {
		return fmt.Errorf("deleting allocation base: %w", err)
	}

	return nil
}

func (r *AllocationBaseRepositorySQLC) DeleteItems(ctx context.Context, baseCode int32) error {
	return r.q.DeleteAllocationBaseItems(ctx, baseCode)
}

func rowToEntity(row sqlc.AllocationBasis) *entity.AllocationBase {
	e := &entity.AllocationBase{
		Code:        row.Code,
		Description: row.Description,
		Period:      row.Period,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		CreatedBy:   row.CreatedBy,
	}

	if row.Observation.Valid {
		v := row.Observation.String
		e.Observation = &v
	}

	return e
}

func rowsToEntities(rows []sqlc.AllocationBasis) []*entity.AllocationBase {
	out := make([]*entity.AllocationBase, 0, len(rows))

	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}

	return out
}

func itemRowToEntity(row sqlc.AllocationBaseItem) *entity.AllocationBaseItem {
	return &entity.AllocationBaseItem{
		AllocationBaseCode: row.AllocationBaseCode,
		CostCenterCode:     row.CostCenterCode,
		Amount:             row.Amount,
		Percentage:         row.Percentage,
		CreatedAt:          row.CreatedAt,
	}
}

func itemsToEntities(rows []sqlc.AllocationBaseItem) []*entity.AllocationBaseItem {
	out := make([]*entity.AllocationBaseItem, 0, len(rows))

	for _, row := range rows {
		out = append(out, itemRowToEntity(row))
	}

	return out
}

func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}

	return sql.NullString{
		String: *s,
		Valid:  true,
	}
}
