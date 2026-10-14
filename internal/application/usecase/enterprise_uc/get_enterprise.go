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
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	e, err := uc.Repo.GetByCodeForUser(ctx, code, userID)
	if err != nil {
		return nil, err
	}
	return toEnterpriseResponse(e), nil
}

// ListEnterprisesUseCase devolve as empresas a que o usuário está vinculado.
// Antes listava todas as empresas do banco para qualquer usuário.
type ListEnterprisesUseCase struct {
	Repo repository.EnterpriseRepository
	Auth ports.AuthService
}

func (uc *ListEnterprisesUseCase) Execute(ctx context.Context) ([]*response.EnterpriseResponse, error) {
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	items, err := uc.Repo.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.EnterpriseResponse, 0, len(items))
	for _, e := range items {
		out = append(out, toEnterpriseResponse(e))
	}
	return out, nil
}
