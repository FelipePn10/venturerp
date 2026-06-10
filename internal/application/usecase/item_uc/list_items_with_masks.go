package item_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
)

type ListItemsWithMasksUseCase struct {
	Repo repository.ItemRepository
	Auth ports.AuthService
}

func NewListItemsWithMasksUseCase(repo repository.ItemRepository, auth ports.AuthService) *ListItemsWithMasksUseCase {
	return &ListItemsWithMasksUseCase{Repo: repo, Auth: auth}
}

func (uc *ListItemsWithMasksUseCase) Execute(ctx context.Context) ([]response.ItemWithMasksResponse, error) {
	if !uc.Auth.FindItemByCode(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListAllWithMasks(ctx)
	if err != nil {
		return nil, err
	}
	return toItemWithMasksResponses(list), nil
}
