package group_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/group/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/group/repository"
)

// GetGroupUseCase fetches a single PDM group by its code.
type GetGroupUseCase struct {
	Repo repository.GroupRepository
	Auth ports.AuthService
}

func NewGetGroupUseCase(repo repository.GroupRepository, auth ports.AuthService) *GetGroupUseCase {
	return &GetGroupUseCase{Repo: repo, Auth: auth}
}

func (uc *GetGroupUseCase) Execute(ctx context.Context, code int) (*response.GroupResponse, error) {
	g, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toGroupResponse(g), nil
}

// ListGroupsUseCase returns all PDM groups.
type ListGroupsUseCase struct {
	Repo repository.GroupRepository
	Auth ports.AuthService
}

func NewListGroupsUseCase(repo repository.GroupRepository, auth ports.AuthService) *ListGroupsUseCase {
	return &ListGroupsUseCase{Repo: repo, Auth: auth}
}

func (uc *ListGroupsUseCase) Execute(ctx context.Context) ([]*response.GroupResponse, error) {
	items, err := uc.Repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*response.GroupResponse, 0, len(items))
	for _, g := range items {
		out = append(out, toGroupResponse(g))
	}
	return out, nil
}

// UpdateGroupUseCase changes a group's description/enterprise.
type UpdateGroupUseCase struct {
	Repo repository.GroupRepository
	Auth ports.AuthService
}

func NewUpdateGroupUseCase(repo repository.GroupRepository, auth ports.AuthService) *UpdateGroupUseCase {
	return &UpdateGroupUseCase{Repo: repo, Auth: auth}
}

func (uc *UpdateGroupUseCase) Execute(ctx context.Context, code int, description string, enterpriseID int) (*response.GroupResponse, error) {
	if !uc.Auth.CanCreateGroup(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if description == "" {
		return nil, errorsuc.NewValidationError("description is required")
	}
	updated, err := uc.Repo.Update(ctx, &entity.Group{
		Code:         code,
		Description:  description,
		EnterpriseID: enterpriseID,
	})
	if err != nil {
		return nil, err
	}
	return toGroupResponse(updated), nil
}
