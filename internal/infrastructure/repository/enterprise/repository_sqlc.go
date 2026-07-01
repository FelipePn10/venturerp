package enterprise

import (
	"context"
	"errors"
	"fmt"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/enterprise/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgconn"
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
		// code has a UNIQUE constraint: surface a typed conflict so the handler
		// returns 409 instead of a generic 500 on duplicate codes.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, errorsuc.NewConflictError(fmt.Sprintf("enterprise with code %d already exists", enterprise.Code))
		}
		return nil, fmt.Errorf("create enterprise: %w", err)
	}

	return toEnterpriseEntity(dbEnterprise), nil
}

func (r *repositoryEnterpriseSQLC) GetByCode(ctx context.Context, code int) (*entity.Enterprise, error) {
	row, err := r.q.GetEnterpriseByCode(ctx, int32(code))
	if err != nil {
		return nil, fmt.Errorf("get enterprise by code %d: %w", code, err)
	}
	return toEnterpriseEntity(row), nil
}

func (r *repositoryEnterpriseSQLC) List(ctx context.Context) ([]*entity.Enterprise, error) {
	rows, err := r.q.ListEnterprises(ctx)
	if err != nil {
		return nil, fmt.Errorf("list enterprises: %w", err)
	}
	out := make([]*entity.Enterprise, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEnterpriseEntity(row))
	}
	return out, nil
}

func toEnterpriseEntity(row sqlc.Enterprise) *entity.Enterprise {
	return &entity.Enterprise{
		ID:        int(row.ID),
		Code:      int(row.Code),
		Name:      row.Name,
		CreatedBy: pgutil.FromPgUUID(row.CreatedBy),
	}
}
