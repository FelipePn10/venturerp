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

	return &entity.Group{
		ID:           int32(dbGroup.ID),
		Code:         int(dbGroup.Code),
		Description:  dbGroup.Description,
		EnterpriseID: int(dbGroup.EnterpriseID),
		CreatedBy:    pgutil.FromPgUUID(dbGroup.CreatedBy),
	}, nil
}
