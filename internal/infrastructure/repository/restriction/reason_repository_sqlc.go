package restriction

import (
	"context"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
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
	row, err := r.q.CreateRestrictionReason(ctx, sqlc.CreateRestrictionReasonParams{
		Description: re.Description,
		Situation:   re.Situation,
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
	row, err := r.q.GetRestrictionReasonByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("restriction reason %d not found", code)
		}
		return nil, fmt.Errorf("fetching restriction reason: %w", err)
	}
	return reasonRowToEntity(row), nil
}

func (r *RestrictionReasonRepositorySQLC) List(
	ctx context.Context,
) ([]*entity.RestrictionReason, error) {
	rows, err := r.q.ListRestrictionReasons(ctx)
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
	row, err := r.q.UpdateRestrictionReason(ctx, sqlc.UpdateRestrictionReasonParams{
		Code:        re.Code,
		Description: re.Description,
		Situation:   re.Situation,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("restriction reason %d not found", re.Code)
		}
		return nil, fmt.Errorf("updating restriction reason: %w", err)
	}
	return reasonRowToEntity(row), nil
}

func (r *RestrictionReasonRepositorySQLC) Delete(ctx context.Context, code int64) error {
	return r.q.DeleteRestrictionReason(ctx, code)
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
