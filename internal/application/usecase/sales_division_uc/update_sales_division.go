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

	sd := &entity.SalesDivision{
		Code:                    code,
		Description:             dto.Description,
		CommercialAnalysis:      entity.SalesDivisionAnalysis(dto.CommercialAnalysis),
		FinancialAnalysis:       entity.SalesDivisionAnalysis(dto.FinancialAnalysis),
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
