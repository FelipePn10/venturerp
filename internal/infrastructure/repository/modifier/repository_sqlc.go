package modifier

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/modifier/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *repositoryModifierSQLC) Create(
	ctx context.Context,
	modifier *entity.Modifier,
) (*entity.Modifier, error) {
	params := sqlc.CreateModifierParams{
		Description: modifier.Description,
		CreatedBy:   pgutil.ToPgUUID(modifier.CreatedBy),
	}

	dbModifier, err := r.q.CreateModifier(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create modifier: %w", err)
	}

	return &entity.Modifier{
		ID:          int(dbModifier.ID),
		Description: dbModifier.Description,
		CreatedBy:   pgutil.FromPgUUID(dbModifier.CreatedBy),
	}, nil
}
