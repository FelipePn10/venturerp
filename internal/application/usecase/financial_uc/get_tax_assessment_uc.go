package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type GetTaxAssessmentUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetTaxAssessmentUseCase) Execute(ctx context.Context, imposto, competencia string) (*response.TaxAssessmentResponse, error) {
	if !uc.Auth.CanGetTaxAssessment(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	t, err := uc.Repo.GetTaxAssessment(ctx, imposto, competencia)
	if err != nil {
		return nil, err
	}
	return toTaxAssessmentResponse(t), nil
}

func (uc *GetTaxAssessmentUseCase) List(ctx context.Context, competencia string) ([]*response.TaxAssessmentResponse, error) {
	if !uc.Auth.CanListTaxAssessments(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListTaxAssessments(ctx, competencia)
	if err != nil {
		return nil, err
	}
	return toTaxAssessmentResponses(list), nil
}
