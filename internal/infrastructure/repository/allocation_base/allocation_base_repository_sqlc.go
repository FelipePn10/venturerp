package allocation_base

import (
	"context"
	"errors"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"

	"github.com/FelipePn10/panossoerp/internal/domain/allocation_base/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
)

// empresa devolve o identificador do tenant no formato que o sqlc espera para a
// coluna `enterprise_id` (anulável no esquema, sempre preenchida em uso).
func empresa(ctx context.Context) (*int64, error) {
	id, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (r *AllocationBaseRepositorySQLC) Create(
	ctx context.Context,
	ab *entity.AllocationBase,
) (*entity.AllocationBase, error) {

	tenantID, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateAllocationBase(ctx, sqlc.CreateAllocationBaseParams{
		Code:         ab.Code,
		Description:  ab.Description,
		Period:       ab.Period,
		Observation:  pgutil.ToPgTextFromPtr(ab.Observation),
		CreatedBy:    pgutil.ToPgUUID(ab.CreatedBy),
		EnterpriseID: tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("creating allocation base: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *AllocationBaseRepositorySQLC) AddItem(
	ctx context.Context,
	item *entity.AllocationBaseItem,
) (*entity.AllocationBaseItem, error) {

	tenantID, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.AddAllocationBaseItem(ctx, sqlc.AddAllocationBaseItemParams{
		AllocationBaseCode: item.AllocationBaseCode,
		CostCenterCode:     item.CostCenterCode,
		Amount:             item.Amount,
		Percentage:         item.Percentage,
		EnterpriseID:       tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// A linha só nasce se a base for da empresa: o INSERT ... SELECT não
			// devolve nada quando a base é de outra.
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("base de rateio %d não encontrada", item.AllocationBaseCode))
		}
		return nil, fmt.Errorf("adding allocation base item: %w", err)
	}

	return itemRowToEntity(row), nil
}

func (r *AllocationBaseRepositorySQLC) GetByCode(
	ctx context.Context,
	code int32,
) (*entity.AllocationBase, error) {

	tenantID, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetAllocationBaseByCode(ctx, sqlc.GetAllocationBaseByCodeParams{Code: code, EnterpriseID: tenantID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("base de rateio %d não encontrada", code))
		}
		return nil, fmt.Errorf("fetching allocation base: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *AllocationBaseRepositorySQLC) GetItems(
	ctx context.Context,
	baseCode int32,
) ([]*entity.AllocationBaseItem, error) {

	tenantID, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.GetAllocationBaseItems(ctx, sqlc.GetAllocationBaseItemsParams{AllocationBaseCode: baseCode, EnterpriseID: tenantID})
	if err != nil {
		return nil, fmt.Errorf("fetching allocation base items: %w", err)
	}

	return itemsToEntities(rows), nil
}

func (r *AllocationBaseRepositorySQLC) List(
	ctx context.Context,
) ([]*entity.AllocationBase, error) {

	tenantID, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListAllocationBases(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing allocation bases: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *AllocationBaseRepositorySQLC) Delete(
	ctx context.Context,
	code int32,
) error {

	// remove dependências primeiro (consistência referencial)
	if err := r.DeleteItems(ctx, code); err != nil {
		return err
	}

	tenantID, err := empresa(ctx)
	if err != nil {
		return err
	}
	return r.q.DeleteAllocationBase(ctx, sqlc.DeleteAllocationBaseParams{Code: code, EnterpriseID: tenantID})
}

func (r *AllocationBaseRepositorySQLC) DeleteItems(
	ctx context.Context,
	baseCode int32,
) error {
	tenantID, err := empresa(ctx)
	if err != nil {
		return err
	}
	return r.q.DeleteAllocationBaseItems(ctx, sqlc.DeleteAllocationBaseItemsParams{AllocationBaseCode: baseCode, EnterpriseID: tenantID})
}

func rowToEntity(row sqlc.AllocationBasis) *entity.AllocationBase {
	e := &entity.AllocationBase{
		Code:        row.Code,
		Description: row.Description,
		Period:      row.Period,
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:   pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:   pgutil.FromPgUUID(row.CreatedBy),
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
		CreatedAt:          pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func itemsToEntities(rows []sqlc.AllocationBaseItem) []*entity.AllocationBaseItem {
	out := make([]*entity.AllocationBaseItem, 0, len(rows))

	for _, row := range rows {
		out = append(out, itemRowToEntity(row))
	}

	return out
}
