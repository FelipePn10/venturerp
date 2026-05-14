package sales_division_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_division/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_division/repository"
)

type CreateSalesDivisionUseCase struct {
	Repo repository.SalesDivisionRepository
	Auth ports.AuthService
}

func (uc *CreateSalesDivisionUseCase) Execute(
	ctx context.Context,
	dto request.CreateSalesDivisionDTO,
) (*entity.SalesDivision, error) {
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
		return nil, err
	}

	return uc.Repo.Create(ctx, sd)
}
