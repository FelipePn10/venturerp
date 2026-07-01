package enterprise_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/enterprise/repository"
)

// GetEnterpriseUseCase fetches a single enterprise by its business code.
type GetEnterpriseUseCase struct {
	Repo repository.EnterpriseRepository
	Auth ports.AuthService
}

func (uc *GetEnterpriseUseCase) Execute(ctx context.Context, code int) (*response.EnterpriseResponse, error) {
	e, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toEnterpriseResponse(e), nil
}

// ListEnterprisesUseCase returns all enterprises ordered by code.
type ListEnterprisesUseCase struct {
	Repo repository.EnterpriseRepository
	Auth ports.AuthService
}

func (uc *ListEnterprisesUseCase) Execute(ctx context.Context) ([]*response.EnterpriseResponse, error) {
	items, err := uc.Repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*response.EnterpriseResponse, 0, len(items))
	for _, e := range items {
		out = append(out, toEnterpriseResponse(e))
	}
	return out, nil
}
