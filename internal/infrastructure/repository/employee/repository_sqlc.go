package employee

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/employee/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *repositoryEmployeeSQLC) Create(
	ctx context.Context,
	employee *entity.Employee,
) (*entity.Employee, error) {
	params := sqlc.CreateEmployeeParams{
		EnterpriseID: int32(employee.EnterpriseID),
		Code:         int32(employee.Code),
		Description:  employee.Description,
		Name:         employee.Name,
	}

	dbEmployee, err := r.q.CreateEmployee(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create employee: %w", err)
	}

	return &entity.Employee{
		Code: int(dbEmployee.Code),
		Name: dbEmployee.Name,
	}, nil
}
