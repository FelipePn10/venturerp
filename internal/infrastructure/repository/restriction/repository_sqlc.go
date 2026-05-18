package restriction

import (
	"context"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *RestrictionRepositorySQLC) Create(
	ctx context.Context,
	res *entity.Restriction,
) (*entity.Restriction, error) {
	row, err := r.q.CreateRestriction(ctx, sqlc.CreateRestrictionParams{
		Situation:            sqlc.RestrictionSituationEnum(res.Situation),
		CustomerCode:         res.CustomerCode,
		ItemCode:             res.ItemCode,
		ReasonCode:           res.ReasonCode,
		ClassificationType:   pgutil.ToPgTextFromPtr(res.ClassificationType),
		ClassificationOrigin: pgutil.ToPgTextFromPtr(res.ClassificationOrigin),
		DivisionID:           res.DivisionID,
		Weight:               int32(res.Weight),
		CreatedBy:            pgutil.ToPgUUID(res.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating restriction: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *RestrictionRepositorySQLC) Update(
	ctx context.Context,
	res *entity.Restriction,
) (*entity.Restriction, error) {
	row, err := r.q.UpdateRestriction(ctx, sqlc.UpdateRestrictionParams{
		Code:                 pgtype.Int8{Int64: res.Code, Valid: true},
		Situation:            sqlc.RestrictionSituationEnum(res.Situation),
		CustomerCode:         res.CustomerCode,
		ItemCode:             res.ItemCode,
		ReasonCode:           res.ReasonCode,
		ClassificationType:   pgutil.ToPgTextFromPtr(res.ClassificationType),
		ClassificationOrigin: pgutil.ToPgTextFromPtr(res.ClassificationOrigin),
		DivisionID:           res.DivisionID,
		Weight:               int32(res.Weight),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("restriction %d not found", res.Code)
		}
		return nil, fmt.Errorf("updating restriction: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *RestrictionRepositorySQLC) GetByCode(
	ctx context.Context,
	code int64,
) (*entity.Restriction, error) {
	row, err := r.q.GetRestrictionByCode(ctx, pgtype.Int8{Int64: code, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("restriction %d not found", code)
		}
		return nil, fmt.Errorf("fetching restriction: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *RestrictionRepositorySQLC) GetByItemCode(
	ctx context.Context,
	itemCode int64,
) ([]*entity.Restriction, error) {
	rows, err := r.q.GetRestrictionsByItemCode(ctx, &itemCode)
	if err != nil {
		return nil, fmt.Errorf("fetching restrictions for item %d: %w", itemCode, err)
	}
	out := make([]*entity.Restriction, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out, nil
}

func (r *RestrictionRepositorySQLC) GetByCustomerCode(
	ctx context.Context,
	customerCode int64,
) ([]*entity.Restriction, error) {
	rows, err := r.q.GetRestrictionsByCustomerCode(ctx, &customerCode)
	if err != nil {
		return nil, fmt.Errorf("fetching restrictions for customer %d: %w", customerCode, err)
	}
	out := make([]*entity.Restriction, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out, nil
}

func (r *RestrictionRepositorySQLC) List(ctx context.Context) ([]*entity.Restriction, error) {
	rows, err := r.q.ListRestrictions(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing restrictions: %w", err)
	}
	out := make([]*entity.Restriction, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out, nil
}

func (r *RestrictionRepositorySQLC) ListActive(ctx context.Context) ([]*entity.Restriction, error) {
	rows, err := r.q.ListActiveRestrictions(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing active restrictions: %w", err)
	}
	out := make([]*entity.Restriction, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out, nil
}

func (r *RestrictionRepositorySQLC) Deactivate(ctx context.Context, code int64) error {
	return r.q.DeactivateRestriction(ctx, pgtype.Int8{Int64: code, Valid: true})
}

func (r *RestrictionRepositorySQLC) ListRestrictedItemCodes(
	ctx context.Context,
	itemCodes []int64,
) (map[int64]struct{}, error) {
	if len(itemCodes) == 0 {
		return make(map[int64]struct{}), nil
	}
	rows, err := r.q.ListActiveRestrictionsByItems(ctx, itemCodes)
	if err != nil {
		return nil, fmt.Errorf("listing restricted items: %w", err)
	}
	out := make(map[int64]struct{}, len(rows))
	for _, row := range rows {
		if row.ItemCode != nil {
			out[*row.ItemCode] = struct{}{}
		}
	}
	return out, nil
}

func (r *RestrictionRepositorySQLC) AddDominant(
	ctx context.Context,
	d *entity.RestrictionDominant,
) (*entity.RestrictionDominant, error) {
	row, err := r.q.AddRestrictionDominant(ctx, sqlc.AddRestrictionDominantParams{
		RestrictionID: d.RestrictionID,
		QuestionID:    d.QuestionID,
		Operator:      sqlc.RestrictionOperatorEnum(d.Operator),
		ConditionType: sqlc.RestrictionConditionEnum(d.ConditionType),
		AnswerValue:   d.AnswerValue,
		Sequence:      int32(d.Sequence),
	})
	if err != nil {
		return nil, fmt.Errorf("adding restriction dominant: %w", err)
	}
	return &entity.RestrictionDominant{
		ID:            row.ID,
		RestrictionID: row.RestrictionID,
		QuestionID:    row.QuestionID,
		Operator:      entity.RestrictionOperator(row.Operator),
		ConditionType: entity.RestrictionCondition(row.ConditionType),
		AnswerValue:   row.AnswerValue,
		Sequence:      int(row.Sequence),
	}, nil
}

func (r *RestrictionRepositorySQLC) AddDeterminant(
	ctx context.Context,
	d *entity.RestrictionDeterminant,
) (*entity.RestrictionDeterminant, error) {
	row, err := r.q.AddRestrictionDeterminant(ctx, sqlc.AddRestrictionDeterminantParams{
		RestrictionID: d.RestrictionID,
		QuestionID:    d.QuestionID,
		Operator:      sqlc.RestrictionOperatorEnum(d.Operator),
		AnswerValue:   pgutil.ToPgTextFromPtr(d.AnswerValue),
	})
	if err != nil {
		return nil, fmt.Errorf("adding restriction determinant: %w", err)
	}
	return &entity.RestrictionDeterminant{
		ID:            row.ID,
		RestrictionID: row.RestrictionID,
		QuestionID:    row.QuestionID,
		Operator:      entity.RestrictionOperator(row.Operator),
		AnswerValue:   pgutil.FromPgTextPtr(row.AnswerValue),
	}, nil
}

func (r *RestrictionRepositorySQLC) DeleteDominant(ctx context.Context, id int64) error {
	return r.q.DeleteRestrictionDominant(ctx, id)
}

func (r *RestrictionRepositorySQLC) DeleteDeterminant(ctx context.Context, id int64) error {
	return r.q.DeleteRestrictionDeterminant(ctx, id)
}

func (r *RestrictionRepositorySQLC) GetDominants(
	ctx context.Context,
	restrictionID int64,
) ([]*entity.RestrictionDominant, error) {
	rows, err := r.q.GetRestrictionDominants(ctx, restrictionID)
	if err != nil {
		return nil, fmt.Errorf("fetching dominants: %w", err)
	}
	out := make([]*entity.RestrictionDominant, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.RestrictionDominant{
			ID:            row.ID,
			RestrictionID: row.RestrictionID,
			QuestionID:    row.QuestionID,
			Operator:      entity.RestrictionOperator(row.Operator),
			ConditionType: entity.RestrictionCondition(row.ConditionType),
			AnswerValue:   row.AnswerValue,
			Sequence:      int(row.Sequence),
		})
	}
	return out, nil
}

func (r *RestrictionRepositorySQLC) GetDeterminants(
	ctx context.Context,
	restrictionID int64,
) ([]*entity.RestrictionDeterminant, error) {
	rows, err := r.q.GetRestrictionDeterminants(ctx, restrictionID)
	if err != nil {
		return nil, fmt.Errorf("fetching determinants: %w", err)
	}
	out := make([]*entity.RestrictionDeterminant, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.RestrictionDeterminant{
			ID:            row.ID,
			RestrictionID: row.RestrictionID,
			QuestionID:    row.QuestionID,
			Operator:      entity.RestrictionOperator(row.Operator),
			AnswerValue:   pgutil.FromPgTextPtr(row.AnswerValue),
		})
	}
	return out, nil
}

func rowToEntity(row sqlc.Restriction) *entity.Restriction {
	e := &entity.Restriction{
		ID:           row.ID,
		Situation:    entity.RestrictionSituation(row.Situation),
		CustomerCode: row.CustomerCode,
		ItemCode:     row.ItemCode,
		ReasonCode:   row.ReasonCode,
		DivisionID:   row.DivisionID,
		Weight:       int(row.Weight),
		CreatedAt:    pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:    pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:    pgutil.FromPgUUID(row.CreatedBy),
	}
	if row.Code.Valid {
		e.Code = row.Code.Int64
	}
	e.ClassificationType = pgutil.FromPgTextPtr(row.ClassificationType)
	e.ClassificationOrigin = pgutil.FromPgTextPtr(row.ClassificationOrigin)
	return e
}
