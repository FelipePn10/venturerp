package employee

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/employee/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/employee/repository"
)

type ListEmployeesUseCase struct {
	Repo repository.EmployeeRepository
	Auth ports.AuthService
}

func (uc *ListEmployeesUseCase) Execute(ctx context.Context) ([]*entity.Employee, error) {
	if !uc.Auth.CanCreateEmployee(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.List(ctx)
}

type ListEmployeesByRoleUseCase struct {
	Repo repository.EmployeeRepository
	Auth ports.AuthService
}

func (uc *ListEmployeesByRoleUseCase) Execute(ctx context.Context, role string) ([]*entity.Employee, error) {
	if !uc.Auth.CanCreateEmployee(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListByRole(ctx, role)
}
