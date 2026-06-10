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

type UpdateEmployeeUseCase struct {
	Repo repository.EmployeeRepository
	Auth ports.AuthService
}

func (uc *UpdateEmployeeUseCase) Execute(
	ctx context.Context,
	dto request.UpdateEmployeeDTO,
) (*response.EmployeeResponse, error) {
	if !uc.Auth.CanCreateEmployee(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	sit := entity.EmployeeSituation(dto.Situation)
	if sit == "" {
		sit = entity.EmployeeActive
	}
	if sit != entity.EmployeeActive && sit != entity.EmployeeInactive {
		return nil, fmt.Errorf("invalid situation: %s", dto.Situation)
	}

	e := &entity.Employee{
		Code:               dto.Code,
		Name:               dto.Name,
		Situation:          sit,
		ParticipatesBudget: dto.ParticipatesBudget,
		TechnicalAssistant: dto.TechnicalAssistant,
		Role:               dto.Role,
	}
	updated, err := uc.Repo.Update(ctx, e)
	if err != nil {
		return nil, err
	}
	return toEmployeeResponse(updated), nil
}
