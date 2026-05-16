package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type GetTaxAssessmentUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetTaxAssessmentUseCase) Execute(ctx context.Context, imposto, competencia string) (*entity.TaxAssessment, error) {
	if !uc.Auth.CanGetTaxAssessment(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetTaxAssessment(ctx, imposto, competencia)
}

func (uc *GetTaxAssessmentUseCase) List(ctx context.Context, competencia string) ([]*entity.TaxAssessment, error) {
	if !uc.Auth.CanListTaxAssessments(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListTaxAssessments(ctx, competencia)
}
