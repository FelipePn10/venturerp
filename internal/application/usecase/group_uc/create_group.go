package group_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/group/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/group/repository"
)

type CreateGroupUseCase struct {
	Repo repository.GroupRepository
	Auth ports.AuthService
}

func NewCreateGroupUseCase(
	repo repository.GroupRepository,
	auth ports.AuthService,
) *CreateGroupUseCase {
	return &CreateGroupUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func (uc *CreateGroupUseCase) Execute(
	ctx context.Context,
	group *entity.Group,
) (*response.GroupResponse, error) {
	if !uc.Auth.CanCreateGroup(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	created, err := uc.Repo.Create(ctx, group)
	if err != nil {
		return nil, err
	}
	return toGroupResponse(created), nil
}
