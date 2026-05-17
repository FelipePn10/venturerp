package modifier_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/modifier/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/modifier/repository"
)

type CreateModifierUseCase struct {
	Repo repository.ModifierRepository
	Auth ports.AuthService
}

func NewCreateModifierUseCase(
	repo repository.ModifierRepository,
	auth ports.AuthService,
) *CreateModifierUseCase {
	return &CreateModifierUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func (uc *CreateModifierUseCase) Execute(
	ctx context.Context,
	modifier *entity.Modifier,
) (*entity.Modifier, error) {
	if !uc.Auth.CanCreateModifier(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	created, err := uc.Repo.Create(ctx, modifier)
	if err != nil {
		return nil, err
	}

	return created, nil
}
