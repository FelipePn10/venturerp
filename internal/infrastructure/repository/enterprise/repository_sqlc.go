package enterprise

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/enterprise/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *repositoryEnterpriseSQLC) Create(
	ctx context.Context,
	enterprise *entity.Enterprise,
) (*entity.Enterprise, error) {

	params := sqlc.CreateEnterpriseParams{
		Code:      int32(enterprise.Code),
		Name:      enterprise.Name,
		CreatedBy: pgutil.ToPgUUID(enterprise.CreatedBy),
	}
	// Note: ID is omitted — BIGSERIAL auto-generates it.

	dbEnterprise, err := r.q.CreateEnterprise(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create enterprise: %w", err)
	}

	return &entity.Enterprise{
		ID:        int(dbEnterprise.ID),
		Code:      int(dbEnterprise.Code),
		Name:      dbEnterprise.Name,
		CreatedBy: pgutil.FromPgUUID(dbEnterprise.CreatedBy),
	}, nil
}
