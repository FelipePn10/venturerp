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

type UpdateIndependentDemandUseCase struct {
	Repo repository.IndependentDemandRepository
	Auth ports.AuthService
}

func (uc *UpdateIndependentDemandUseCase) Execute(
	ctx context.Context,
	dto request.UpdateIndependentDemandDTO,
) (*response.IndependentDemandResponse, error) {
	if !uc.Auth.CanUpdateIndependentDemand(ctx) {
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
	}

	updated, err := uc.Repo.Update(ctx, demand)
	if err != nil {
		return nil, err
	}
	return toIndependentDemandResponse(updated), nil
}
