package modifier

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/modifier/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

func (r *repositoryModifierSQLC) Create(
	ctx context.Context,
	modifier *entity.Modifier,
) (*entity.Modifier, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	params := sqlc.CreateModifierParams{
		Description:  modifier.Description,
		EnterpriseID: tenantID,
		CreatedBy:    pgutil.ToPgUUID(modifier.CreatedBy),
	}

	dbModifier, err := r.q.CreateModifier(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create modifier: %w", err)
	}

	return modifierToEntity(dbModifier), nil
}

func (r *repositoryModifierSQLC) GetByID(ctx context.Context, id int) (*entity.Modifier, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetModifierByID(ctx, sqlc.GetModifierByIDParams{EnterpriseID: tenantID, ID: int64(id)})
	if err != nil {
		return nil, fmt.Errorf("get modifier by id %d: %w", id, err)
	}
	return modifierToEntity(row), nil
}

func (r *repositoryModifierSQLC) List(ctx context.Context) ([]*entity.Modifier, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListModifiers(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list modifiers: %w", err)
	}
	out := make([]*entity.Modifier, 0, len(rows))
	for _, row := range rows {
		out = append(out, modifierToEntity(row))
	}
	return out, nil
}

func (r *repositoryModifierSQLC) Update(ctx context.Context, modifier *entity.Modifier) (*entity.Modifier, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateModifier(ctx, sqlc.UpdateModifierParams{
		EnterpriseID: tenantID, ID: int64(modifier.ID), Description: modifier.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("update modifier %d: %w", modifier.ID, err)
	}
	return modifierToEntity(row), nil
}

func modifierToEntity(dbModifier sqlc.Modifier) *entity.Modifier {
	return &entity.Modifier{
		ID:          int(dbModifier.ID),
		Description: dbModifier.Description,
		CreatedBy:   pgutil.FromPgUUID(dbModifier.CreatedBy),
	}
}
