package employee

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
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
	dto request.CreateEmployeeDTO,
) (*response.EmployeeResponse, error) {
	if !uc.Auth.CanCreateEmployee(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	e, err := entity.NewEmployee(dto.Code, dto.Name, dto.Role, dto.ParticipatesBudget, dto.TechnicalAssistant, dto.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("building employee: %w", err)
	}

	created, err := uc.Repo.Create(ctx, e)
	if err != nil {
		return nil, err
	}
	return toEmployeeResponse(created), nil
}
