package group

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/group/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *repositoryGroupSQLC) Create(
	ctx context.Context,
	group *entity.Group,
) (*entity.Group, error) {

	params := sqlc.CreateGroupParams{
		Code:         int32(group.Code),
		Description:  group.Description,
		EnterpriseID: int64(group.EnterpriseID),
		CreatedBy:    pgutil.ToPgUUID(group.CreatedBy),
	}

	dbGroup, err := r.q.CreateGroup(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}

	return groupToEntity(dbGroup), nil
}

func (r *repositoryGroupSQLC) GetByCode(ctx context.Context, code int) (*entity.Group, error) {
	row, err := r.q.GetGroupByCode(ctx, int32(code))
	if err != nil {
		return nil, fmt.Errorf("get group by code %d: %w", code, err)
	}
	return groupToEntity(row), nil
}

func (r *repositoryGroupSQLC) List(ctx context.Context) ([]*entity.Group, error) {
	rows, err := r.q.ListGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	out := make([]*entity.Group, 0, len(rows))
	for _, row := range rows {
		out = append(out, groupToEntity(row))
	}
	return out, nil
}

func (r *repositoryGroupSQLC) Update(ctx context.Context, group *entity.Group) (*entity.Group, error) {
	row, err := r.q.UpdateGroup(ctx, sqlc.UpdateGroupParams{
		Code:         int32(group.Code),
		Description:  group.Description,
		EnterpriseID: int64(group.EnterpriseID),
	})
	if err != nil {
		return nil, fmt.Errorf("update group %d: %w", group.Code, err)
	}
	return groupToEntity(row), nil
}

func groupToEntity(dbGroup sqlc.Group) *entity.Group {
	return &entity.Group{
		ID:           int32(dbGroup.ID),
		Code:         int(dbGroup.Code),
		Description:  dbGroup.Description,
		EnterpriseID: int(dbGroup.EnterpriseID),
		CreatedBy:    pgutil.FromPgUUID(dbGroup.CreatedBy),
	}
}
