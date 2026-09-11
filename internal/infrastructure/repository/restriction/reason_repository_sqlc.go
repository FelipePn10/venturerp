package restriction

import (
	"context"
	"errors"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"

	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
)

type RestrictionReasonRepositorySQLC struct {
	q *sqlc.Queries
}

func NewRestrictionReasonRepositorySQLC(q *sqlc.Queries) *RestrictionReasonRepositorySQLC {
	return &RestrictionReasonRepositorySQLC{q: q}
}

func (r *RestrictionReasonRepositorySQLC) Create(
	ctx context.Context,
	re *entity.RestrictionReason,
) (*entity.RestrictionReason, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateRestrictionReason(ctx, sqlc.CreateRestrictionReasonParams{
		Description:  re.Description,
		Situation:    re.Situation,
		EnterpriseID: empresa,
	})
	if err != nil {
		return nil, fmt.Errorf("creating restriction reason: %w", err)
	}
	return reasonRowToEntity(row), nil
}

func (r *RestrictionReasonRepositorySQLC) GetByCode(
	ctx context.Context,
	code int64,
) (*entity.RestrictionReason, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetRestrictionReasonByCode(ctx, sqlc.GetRestrictionReasonByCodeParams{Code: code, EnterpriseID: empresa})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("motivo de restrição %d não encontrado", code))
		}
		return nil, fmt.Errorf("fetching restriction reason: %w", err)
	}
	return reasonRowToEntity(row), nil
}

func (r *RestrictionReasonRepositorySQLC) List(
	ctx context.Context,
) ([]*entity.RestrictionReason, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListRestrictionReasons(ctx, empresa)
	if err != nil {
		return nil, fmt.Errorf("listing restriction reasons: %w", err)
	}
	out := make([]*entity.RestrictionReason, 0, len(rows))
	for _, row := range rows {
		out = append(out, reasonRowToEntity(row))
	}
	return out, nil
}

func (r *RestrictionReasonRepositorySQLC) Update(
	ctx context.Context,
	re *entity.RestrictionReason,
) (*entity.RestrictionReason, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateRestrictionReason(ctx, sqlc.UpdateRestrictionReasonParams{
		Code:         re.Code,
		Description:  re.Description,
		Situation:    re.Situation,
		EnterpriseID: empresa,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("motivo de restrição %d não encontrado", re.Code))
		}
		return nil, fmt.Errorf("updating restriction reason: %w", err)
	}
	return reasonRowToEntity(row), nil
}

func (r *RestrictionReasonRepositorySQLC) Delete(ctx context.Context, code int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	linhas, err := r.q.DeleteRestrictionReason(ctx, sqlc.DeleteRestrictionReasonParams{Code: code, EnterpriseID: empresa})
	if err != nil {
		return err
	}
	if linhas == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("motivo de restrição %d não encontrado", code))
	}
	return nil
}

func reasonRowToEntity(row sqlc.RestrictionReason) *entity.RestrictionReason {
	return &entity.RestrictionReason{
		ID:          row.ID,
		Code:        row.Code,
		Description: row.Description,
		Situation:   row.Situation,
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:   pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}
