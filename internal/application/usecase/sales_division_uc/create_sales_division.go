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

type CreateSalesDivisionUseCase struct {
	Repo repository.SalesDivisionRepository
	Auth ports.AuthService
}

// normalizeAnalysis validates a commercial/financial analysis enum value,
// defaulting empty to FREE and returning a descriptive 422 for unknown values.
func normalizeAnalysis(value, field string) (entity.SalesDivisionAnalysis, error) {
	if value == "" {
		return entity.AnalysisFree, nil
	}
	a := entity.SalesDivisionAnalysis(value)
	switch a {
	case entity.AnalysisFree, entity.AnalysisBlockAlways, entity.AnalysisAlwaysAnalyze:
		return a, nil
	default:
		return "", errorsuc.NewValidationError("invalid " + field + " value: must be FREE, BLOCK_ALWAYS or ALWAYS_ANALYZE")
	}
}

func (uc *CreateSalesDivisionUseCase) Execute(
	ctx context.Context,
	dto request.CreateSalesDivisionDTO,
) (*response.SalesDivisionResponse, error) {
	if !uc.Auth.CanCreateSalesDivision(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}

	sd, err := entity.NewSalesDivision(
		dto.Code,
		dto.Description,
		entity.SalesDivisionAnalysis(dto.CommercialAnalysis),
		entity.SalesDivisionAnalysis(dto.FinancialAnalysis),
		dto.IsTechnicalAssistance,
		dto.ConsiderDeliveryPromise,
		dto.ConsiderMRP,
		dto.AllowOutsideLimits,
		dto.MinimumDeliveryDays,
		dto.FinancialDelayDays,
		dto.PISPercentage,
		dto.CofinsPercentage,
		dto.ParentDivisionID,
		userID,
	)
	if err != nil {
		// Domain validation (invalid enum, missing description, ...) is a client
		// error -> 422, not a 500.
		return nil, errorsuc.NewValidationError(err.Error())
	}

	created, err := uc.Repo.Create(ctx, sd)
	if err != nil {
		return nil, err
	}
	return toSalesDivisionResponse(created), nil
}
