package overhead_allocation

import (
	"context"
	"errors"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"

	"github.com/FelipePn10/panossoerp/internal/domain/overhead_allocation/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
)

func (r *OverheadAllocationRepositorySQLC) Create(
	ctx context.Context,
	oa *entity.OverheadAllocation,
) (*entity.OverheadAllocation, error) {

	emp, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}

	var costCenterCode *int32
	if oa.CostCenterCode != 0 {
		v := int32(oa.CostCenterCode)
		costCenterCode = &v
	}

	var planAccountCode *int32
	if oa.PlanAccountCode != nil {
		v := int32(*oa.PlanAccountCode)
		planAccountCode = &v
	}

	row, err := r.q.CreateOverheadAllocation(
		ctx,
		sqlc.CreateOverheadAllocationParams{
			CostCenterCode:  costCenterCode,
			PlanAccountCode: planAccountCode,
			AccountCode:     pgutil.ToPgTextFromPtr(oa.AccountCode),
			PeriodStart:     pgutil.ToPgDate(oa.PeriodStart),
			PeriodEnd:       pgutil.ToPgDate(oa.PeriodEnd),
			AllocationType:  oa.AllocationType,
			BaseCode:        oa.BaseCode,
			CreatedBy:       pgutil.ToPgUUID(oa.CreatedBy),
			EnterpriseID:    emp,
		},
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError("centro de custo ou base de rateio não encontrados nesta empresa")
	}
	if err != nil {
		return nil, fmt.Errorf("creating overhead allocation: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *OverheadAllocationRepositorySQLC) AddTarget(
	ctx context.Context,
	target *entity.AllocationTarget,
) (*entity.AllocationTarget, error) {

	emp, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}

	var overheadCode *int64
	overheadCode = &target.OverheadCode

	var costCenterCode *int32
	v := int32(target.CostCenterCode)
	costCenterCode = &v

	row, err := r.q.AddAllocationTarget(
		ctx,
		sqlc.AddAllocationTargetParams{
			OverheadCode:   overheadCode,
			CostCenterCode: costCenterCode,
			Percentage:     pgutil.ToPgNumericFromFloat64(target.Percentage),
			Amount:         pgutil.ToPgNumericFromFloat64(target.Amount),
			EnterpriseID:   emp,
		},
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError("rateio ou centro de custo de destino não encontrados nesta empresa")
	}
	if err != nil {
		return nil, fmt.Errorf("adding allocation target: %w", err)
	}

	return targetRowToEntity(row), nil
}

func (r *OverheadAllocationRepositorySQLC) GetByCode(
	ctx context.Context,
	code int64,
) (*entity.OverheadAllocation, error) {

	emp, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetOverheadAllocationByCode(ctx, sqlc.GetOverheadAllocationByCodeParams{Code: code, EnterpriseID: emp})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("rateio de despesas indiretas %d não encontrado", code))
		}

		return nil, fmt.Errorf("fetching overhead allocation: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *OverheadAllocationRepositorySQLC) GetTargets(
	ctx context.Context,
	overheadCode int64,
) ([]*entity.AllocationTarget, error) {

	emp, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.GetAllocationTargets(ctx, sqlc.GetAllocationTargetsParams{OverheadCode: &overheadCode, EnterpriseID: emp})

	if err != nil {
		return nil, fmt.Errorf("fetching allocation targets: %w", err)
	}

	return targetsToEntities(rows), nil
}

func (r *OverheadAllocationRepositorySQLC) List(
	ctx context.Context,
) ([]*entity.OverheadAllocation, error) {

	emp, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListOverheadAllocations(ctx, emp)
	if err != nil {
		return nil, fmt.Errorf("listing overhead allocations: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *OverheadAllocationRepositorySQLC) ListByCostCenter(
	ctx context.Context,
	ccCode int64,
) ([]*entity.OverheadAllocation, error) {

	v := int32(ccCode)

	emp, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListOverheadAllocationsByCostCenter(ctx, sqlc.ListOverheadAllocationsByCostCenterParams{CostCenterCode: &v, EnterpriseID: emp})
	if err != nil {
		return nil, fmt.Errorf("listing overhead allocations by cost center: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *OverheadAllocationRepositorySQLC) Delete(
	ctx context.Context,
	id int64,
) error {

	emp, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if err := r.q.DeleteAllocationTargets(ctx, sqlc.DeleteAllocationTargetsParams{OverheadCode: &id, EnterpriseID: emp}); err != nil {
		return fmt.Errorf("deleting targets: %w", err)
	}
	n, err := r.q.DeleteOverheadAllocation(ctx, sqlc.DeleteOverheadAllocationParams{Code: id, EnterpriseID: emp})
	if err != nil {
		return fmt.Errorf("deleting overhead allocation: %w", err)
	}
	if n == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("rateio de despesas indiretas %d não encontrado", id))
	}
	return nil
}

func (r *OverheadAllocationRepositorySQLC) DeleteTargets(
	ctx context.Context,
	overheadCode int64,
) error {

	emp, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.q.DeleteAllocationTargets(ctx, sqlc.DeleteAllocationTargetsParams{OverheadCode: &overheadCode, EnterpriseID: emp})
}

func rowToEntity(
	row sqlc.OverheadAllocation,
) *entity.OverheadAllocation {

	e := &entity.OverheadAllocation{
		Code:           row.Code,
		AllocationType: row.AllocationType,
		PeriodStart:    pgutil.FromPgDate(row.PeriodStart),
		PeriodEnd:      pgutil.FromPgDate(row.PeriodEnd),
		CreatedAt:      pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:      pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:      pgutil.FromPgUUID(row.CreatedBy),
	}

	if row.CostCenterCode != nil {
		e.CostCenterCode = int64(*row.CostCenterCode)
	}

	if row.PlanAccountCode != nil {
		v := int64(*row.PlanAccountCode)
		e.PlanAccountCode = &v
	}

	if row.AccountCode.Valid {
		v := row.AccountCode.String
		e.AccountCode = &v
	}

	if row.BaseCode != nil {
		v := *row.BaseCode
		e.BaseCode = &v
	}

	return e
}

func rowsToEntities(
	rows []sqlc.OverheadAllocation,
) []*entity.OverheadAllocation {

	out := make([]*entity.OverheadAllocation, 0, len(rows))

	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}

	return out
}

func targetRowToEntity(
	row sqlc.OverheadAllocationTarget,
) *entity.AllocationTarget {

	e := &entity.AllocationTarget{
		Code:       row.Code,
		Percentage: pgutil.FromPgNumericToFloat64(row.Percentage),
		Amount:     pgutil.FromPgNumericToFloat64(row.Amount),
		CreatedAt:  pgutil.FromPgTimestamptz(row.CreatedAt),
	}

	if row.OverheadCode != nil {
		e.OverheadCode = *row.OverheadCode
	}

	if row.CostCenterCode != nil {
		e.CostCenterCode = int64(*row.CostCenterCode)
	}

	return e
}

func targetsToEntities(
	rows []sqlc.OverheadAllocationTarget,
) []*entity.AllocationTarget {

	out := make([]*entity.AllocationTarget, 0, len(rows))

	for _, row := range rows {
		out = append(out, targetRowToEntity(row))
	}

	return out
}
