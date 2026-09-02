package structure_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
)

type DeleteStructureComponentUseCase struct {
	Repo  repository.ItemStructureRepository
	Auth  ports.AuthService
	Items any
}

func NewDeleteStructureComponentUseCase(repo repository.ItemStructureRepository, auth ports.AuthService, items any) *DeleteStructureComponentUseCase {
	return &DeleteStructureComponentUseCase{Repo: repo, Auth: auth, Items: items}
}

func (uc *DeleteStructureComponentUseCase) Execute(ctx context.Context, parent, child request.TextCode, mask *string) error {
	if !uc.Auth.UpdateStructure(ctx) {
		return errorsuc.ErrUnauthorized
	}
	parentCode, err := resolveItemCode(ctx, uc.Items, parent)
	if err != nil {
		return err
	}
	childCode, err := resolveItemCode(ctx, uc.Items, child)
	if err != nil {
		return err
	}
	return uc.Repo.DeleteByCodes(ctx, parentCode, childCode, mask)
}
