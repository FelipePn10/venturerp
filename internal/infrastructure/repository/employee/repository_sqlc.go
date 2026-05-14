package employee

import (
	"context"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/employee/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
)

func (r *RepositoryEmployeeSQLC) Create(
	ctx context.Context,
	e *entity.Employee,
) (*entity.Employee, error) {
	row, err := r.q.CreateNewEmployee(ctx, sqlc.CreateNewEmployeeParams{
		Code:               e.Code,
		Name:               e.Name,
		Situation:          sqlc.SituationEnum(e.Situation),
		ParticipatesBudget: e.ParticipatesBudget,
		TechnicalAssistant: e.TechnicalAssistant,
		Role:               e.Role,
		CreatedBy:          pgutil.ToPgUUID(e.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating employee: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *RepositoryEmployeeSQLC) Update(
	ctx context.Context,
	e *entity.Employee,
) (*entity.Employee, error) {
	row, err := r.q.UpdateEmployee(ctx, sqlc.UpdateEmployeeParams{
		Code:               e.Code,
		Name:               e.Name,
		Situation:          sqlc.SituationEnum(e.Situation),
		ParticipatesBudget: e.ParticipatesBudget,
		TechnicalAssistant: e.TechnicalAssistant,
		Role:               e.Role,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("employee %d not found", e.Code)
		}
		return nil, fmt.Errorf("updating employee: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *RepositoryEmployeeSQLC) GetByCode(
	ctx context.Context,
	code int64,
) (*entity.Employee, error) {
	row, err := r.q.GetEmployeeByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("employee %d not found", code)
		}
		return nil, fmt.Errorf("fetching employee: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *RepositoryEmployeeSQLC) List(ctx context.Context) ([]*entity.Employee, error) {
	rows, err := r.q.ListEmployees(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing employees: %w", err)
	}
	return rowsToEntities(rows), nil
}

func (r *RepositoryEmployeeSQLC) ListByRole(
	ctx context.Context,
	role string,
) ([]*entity.Employee, error) {
	rows, err := r.q.ListEmployeesByRole(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("listing employees by role %s: %w", role, err)
	}
	return rowsToEntities(rows), nil
}

func (r *RepositoryEmployeeSQLC) Deactivate(ctx context.Context, code int64) error {
	if err := r.q.DeactivateEmployee(ctx, code); err != nil {
		return fmt.Errorf("deactivating employee %d: %w", code, err)
	}
	return nil
}

func rowToEntity(row sqlc.EmployeeLegacy) *entity.Employee {
	return &entity.Employee{
		ID:                 row.ID,
		Code:               row.Code,
		Name:               row.Name,
		Situation:          entity.EmployeeSituation(row.Situation),
		ParticipatesBudget: row.ParticipatesBudget,
		TechnicalAssistant: row.TechnicalAssistant,
		Role:               row.Role,
		CreatedAt:          pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:          pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:          pgutil.FromPgUUID(row.CreatedBy),
	}
}

func rowsToEntities(rows []sqlc.EmployeeLegacy) []*entity.Employee {
	out := make([]*entity.Employee, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out
}
