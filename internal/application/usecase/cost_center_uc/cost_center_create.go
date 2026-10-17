package cost_center_uc

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
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
	start, msg := datetime.Obrigatoria("data de início", dto.StartDate)
	if msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	}
	var end *time.Time
	if dto.EndDate != nil {
		e, msg := datetime.Obrigatoria("data de fim", *dto.EndDate)
		if msg != "" {
			return nil, errorsuc.NewValidationError(msg)
		}
		end = &e
	}
	if !tiposCentro[dto.Type] {
		return nil, errorsuc.NewValidationError("tipo inválido: use AUXILIARY, PRODUCTIVE, ADMINISTRATIVE ou COMMERCIAL")
	}
	if end != nil && end.Before(start) {
		return nil, errorsuc.NewValidationError("a data de fim não pode ser anterior à data de início")
	}
	if err := conferirCentroPai(ctx, uc.Repo, dto.Code, dto.ParentCode); err != nil {
		return nil, err
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
