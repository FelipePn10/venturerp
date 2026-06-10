package cost_center_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/repository"
)

type CreateCostCenterUseCase struct {
	Repo repository.CostCenterRepository
	Auth ports.AuthService
}

func (uc *CreateCostCenterUseCase) Execute(
	ctx context.Context,
	dto request.CreateCostCenterDTO,
) (*response.CostCenterResponse, error) {
	if !uc.Auth.CanCreateCostCenter(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	start, _ := time.Parse("2006-01-02", dto.StartDate)
	var end *time.Time
	if dto.EndDate != nil {
		e, _ := time.Parse("2006-01-02", *dto.EndDate)
		end = &e
	}
	cc := &entity.CostCenter{
		Code:        dto.Code,
		Description: dto.Description,
		ParentCode:  dto.ParentCode,
		Type:        dto.Type,
		IsRatio:     dto.IsRatio,
		StartDate:   start,
		EndDate:     end,
		CreatedBy:   dto.CreatedBy,
	}
	created, err := uc.Repo.Create(ctx, cc)
	if err != nil {
		return nil, err
	}
	return toCostCenterResponse(created), nil
}
