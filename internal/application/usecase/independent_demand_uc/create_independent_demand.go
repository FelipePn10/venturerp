package independent_demand_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository"
)

type CreateIndependentDemandUseCase struct {
	Repo repository.IndependentDemandRepository
	Auth ports.AuthService
}

func (uc *CreateIndependentDemandUseCase) Execute(
	ctx context.Context,
	dto request.CreateIndependentDemandDTO,
) (*response.IndependentDemandResponse, error) {
	if !uc.Auth.CanCreateIndependentDemand(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	date, _ := time.Parse("2006-01-02", dto.DemandDate)
	demand := &entity.IndependentDemand{
		CodeDemand:     dto.CodeDemand,
		ItemCode:       dto.ItemCode,
		Mask:           dto.Mask,
		CostCenterCode: dto.CostCenterCode,
		Quantity:       dto.Quantity,
		DemandDate:     date,
		CreatedBy:      dto.CreatedBy,
	}
	created, err := uc.Repo.Create(ctx, demand)
	if err != nil {
		return nil, err
	}
	return toIndependentDemandResponse(created), nil
}
