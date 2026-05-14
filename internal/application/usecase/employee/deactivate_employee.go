package employee

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/employee/repository"
)

type DeactivateEmployeeUseCase struct {
	Repo repository.EmployeeRepository
	Auth ports.AuthService
}

func (uc *DeactivateEmployeeUseCase) Execute(ctx context.Context, code int64) error {
	if !uc.Auth.CanCreateEmployee(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Deactivate(ctx, code)
}
