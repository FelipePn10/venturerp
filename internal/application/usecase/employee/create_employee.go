package employee

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/employee/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/employee/repository"
)

type CreateEmployeeUseCase struct {
	Repo repository.EmployeeRepository
	Auth ports.AuthService
}

func (uc *CreateEmployeeUseCase) Execute(
	ctx context.Context,
	employee *entity.Employee,
) (*entity.Employee, error) {
	if !uc.Auth.CanCreateEmployee(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	created, err := uc.Repo.Create(ctx, employee)
	if err != nil {
		return nil, err
	}

	return created, nil
}
