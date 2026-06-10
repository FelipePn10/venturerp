package employee

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/employee/repository"
)

type ListEmployeesUseCase struct {
	Repo repository.EmployeeRepository
	Auth ports.AuthService
}

func (uc *ListEmployeesUseCase) Execute(ctx context.Context) ([]*response.EmployeeResponse, error) {
	if !uc.Auth.CanCreateEmployee(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return toEmployeeResponses(list), nil
}

type ListEmployeesByRoleUseCase struct {
	Repo repository.EmployeeRepository
	Auth ports.AuthService
}

func (uc *ListEmployeesByRoleUseCase) Execute(ctx context.Context, role string) ([]*response.EmployeeResponse, error) {
	if !uc.Auth.CanCreateEmployee(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListByRole(ctx, role)
	if err != nil {
		return nil, err
	}
	return toEmployeeResponses(list), nil
}
