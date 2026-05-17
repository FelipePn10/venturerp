package enterprise_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/enterprise/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/enterprise/repository"
)

type CreateEnterpriseUseCase struct {
	Repo repository.EnterpriseRepository
	Auth ports.AuthService
}

func NewCreateEnterpriseUseCase(
	repo repository.EnterpriseRepository,
	auth ports.AuthService,
) *CreateEnterpriseUseCase {
	return &CreateEnterpriseUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func (uc *CreateEnterpriseUseCase) Execute(
	ctx context.Context,
	enterprise *entity.Enterprise,
) (*entity.Enterprise, error) {
	if !uc.Auth.CanCreateEnterprise(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	created, err := uc.Repo.Create(ctx, enterprise)
	if err != nil {
		return nil, err
	}

	return created, nil
}
