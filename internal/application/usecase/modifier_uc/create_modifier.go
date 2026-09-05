package modifier_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
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
) (*response.ModifierResponse, error) {
	if !uc.Auth.CanCreateModifier(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	// O autor sai do usuário autenticado, como no cadastro de grupo. Confiar no
	// created_by do corpo deixava o campo em branco quando a tela não o enviava
	// — e o banco recusava o registro por chave estrangeira, com uma mensagem
	// que não dizia nada ao usuário.
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	modifier.CreatedBy = actor

	created, err := uc.Repo.Create(ctx, modifier)
	if err != nil {
		return nil, err
	}

	return toModifierResponse(created), nil
}
