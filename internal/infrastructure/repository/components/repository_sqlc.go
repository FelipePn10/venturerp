package components

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/component/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *repositoryComponentsSQLC) Save(
	ctx context.Context,
	component *entity.Component,
) error {
	params := sqlc.CreateComponentParams{
		Name:      component.Name,
		GroupCode: component.GroupCode,
		Warehouse: &component.Warehouse,
		CreatedBy: pgutil.ToPgUUID(component.CreatedBy),
	}

	_, err := r.q.CreateComponent(ctx, params)
	if err != nil {
		return err
	}

	return nil
}
