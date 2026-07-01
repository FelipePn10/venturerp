package sales_division_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_division/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_division/repository"
)

type UpdateSalesDivisionUseCase struct {
	Repo repository.SalesDivisionRepository
	Auth ports.AuthService
}

func (uc *UpdateSalesDivisionUseCase) Execute(
	ctx context.Context,
	code int64,
	dto request.UpdateSalesDivisionDTO,
) (*response.SalesDivisionResponse, error) {
	if !uc.Auth.CanUpdateSalesDivision(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	// commercial/financial analysis are constrained enums; validate here (empty
	// defaults to FREE) so an unknown value returns 422 instead of a raw DB enum
	// error. Valid values: FREE, BLOCK_ALWAYS, ALWAYS_ANALYZE.
	commercial, err := normalizeAnalysis(dto.CommercialAnalysis, "commercial_analysis")
	if err != nil {
		return nil, err
	}
	financial, err := normalizeAnalysis(dto.FinancialAnalysis, "financial_analysis")
	if err != nil {
		return nil, err
	}

	sd := &entity.SalesDivision{
		Code:                    code,
		Description:             dto.Description,
		CommercialAnalysis:      commercial,
		FinancialAnalysis:       financial,
		IsTechnicalAssistance:   dto.IsTechnicalAssistance,
		ConsiderDeliveryPromise: dto.ConsiderDeliveryPromise,
		ConsiderMRP:             dto.ConsiderMRP,
		AllowOutsideLimits:      dto.AllowOutsideLimits,
		MinimumDeliveryDays:     dto.MinimumDeliveryDays,
		FinancialDelayDays:      dto.FinancialDelayDays,
		PISPercentage:           dto.PISPercentage,
		CofinsPercentage:        dto.CofinsPercentage,
		ParentDivisionID:        dto.ParentDivisionID,
	}

	updated, err := uc.Repo.Update(ctx, sd)
	if err != nil {
		return nil, err
	}
	return toSalesDivisionResponse(updated), nil
}
