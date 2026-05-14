package employee

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/employee/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/employee/repository"
)

type GetEmployeeUseCase struct {
	Repo repository.EmployeeRepository
	Auth ports.AuthService
}

func (uc *GetEmployeeUseCase) Execute(ctx context.Context, code int64) (*entity.Employee, error) {
	if !uc.Auth.CanCreateEmployee(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetByCode(ctx, code)
}
