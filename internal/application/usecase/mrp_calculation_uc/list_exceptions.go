package mrp_calculation_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
)

type ListMRPExceptionsUseCase struct {
	Repo repository.MRPCalculationRepository
	Auth ports.AuthService
	// Pegging, quando presente, diz em cada exceção de ordem planejada quais
	// pedidos/demandas dependem dela.
	Pegging *PeggingUseCase
}

func (uc *ListMRPExceptionsUseCase) Execute(
	ctx context.Context,
	planCode int64,
) ([]*response.MRPExceptionMessageResponse, error) {
	if !uc.Auth.CanListMRPExceptions(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListExceptionsByPlan(ctx, planCode)
	if err != nil {
		return nil, err
	}
	out := toMRPExceptionResponses(list)
	if uc.Pegging == nil {
		return out, nil
	}
	ordens := []int64{}
	for _, e := range list {
		if e.SourceCode != nil && e.SourceType != nil && *e.SourceType == "PLANNED_ORDER" {
			ordens = append(ordens, *e.SourceCode)
		}
	}
	afetados, err := uc.Pegging.AffectedBySupply(ctx, planCode, entity.PegSupplyFirmOrder, ordens)
	if err != nil {
		return nil, err
	}
	for i, e := range list {
		if e.SourceCode != nil {
			out[i].AffectedDemands = afetados[*e.SourceCode]
		}
	}
	return out, nil
}
