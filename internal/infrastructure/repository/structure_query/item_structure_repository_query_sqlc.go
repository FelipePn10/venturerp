package structure_query

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	maskservice "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service"
	maskvo "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	str "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/google/uuid"
)

func (r *StructureQueryRepositorySQLC) GetItemByCode(
	ctx context.Context,
	code int64,
) (*itementity.Item, error) {

	row, err := r.q.GetItemByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("fetching item %d: %w", code, err)
	}

	return &itementity.Item{
		Inherit: row.Inherit,
	}, nil
}

func (r *StructureQueryRepositorySQLC) GetDirectChildrenForMask(
	ctx context.Context,
	parentCode int64,
	mask string,
) ([]*str.ItemStructure, error) {

	rows, err := r.q.GetDirectChildrenForMask(ctx, sqlc.GetDirectChildrenForMaskParams{
		ParentCode: parentCode,
		ParentMask: pgutil.ToPgTextFromString(mask),
	})
	if err != nil {
		return nil, fmt.Errorf("fetching children of item %d: %w", parentCode, err)
	}

	return structureQueryRowsToEntities(rows), nil
}

func (r *StructureQueryRepositorySQLC) CreateMaskForItem(
	ctx context.Context,
	itemCode int64,
	mask string,
	answers []maskservice.ChildMaskAnswerInput,
	createdBy uuid.UUID,
) error {

	m, err := r.q.InsertItemtMask(ctx, sqlc.InsertItemtMaskParams{
		ItemCode:  itemCode,
		Mask:      mask,
		MaskHash:  maskHash(mask),
		CreatedBy: pgutil.ToPgUUID(createdBy),
	})
	if err != nil {
		return err
	}

	for _, a := range answers {
		err := r.q.InsertItemMaskAnswer(ctx, sqlc.InsertItemMaskAnswerParams{
			MaskID:     m.ID,
			QuestionID: a.QuestionID,
			OptionID:   a.OptionID,
			Position:   a.Position,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func structureQueryRowsToEntities(rows []sqlc.GetDirectChildrenForMaskRow) []*str.ItemStructure {
	out := make([]*str.ItemStructure, 0, len(rows))

	for _, row := range rows {
		e := &str.ItemStructure{
			ID:                row.ID,
			ParentCode:        row.ParentCode,
			ChildCode:         row.ChildCode,
			ChildDescription:  row.ChildDescription,
			Quantity:          row.Quantity,
			LossPercentage:    row.LossPercentage,
			UnitOfMeasurement: types.TypeUnitOfMeasurementItem(row.UnitOfMeasurement),
			Health:            types.Health(row.Health),
			Sequence:          int(row.Sequence),
			IsActive:          row.IsActive,
			CreatedBy:         pgutil.FromPgUUID(row.CreatedBy),
			CreatedAt:         pgutil.FromPgTimestamptz(row.CreatedAt),
			UpdatedAt:         pgutil.FromPgTimestamptz(row.UpdatedAt),
		}

		if row.ParentMask.Valid {
			v := row.ParentMask.String
			e.ParentMask = &v
		}

		if row.Notes.Valid {
			v := row.Notes.String
			e.Notes = &v
		}

		out = append(out, e)
	}

	return out
}

func (r *StructureQueryRepositorySQLC) GetMaskAnswersByItemAndValue(
	ctx context.Context,
	itemCode int64,
	mask string,
) ([]maskvo.MaskAnswer, error) {

	rows, err := r.q.GetMaskAnswersByItemAndValue(ctx, sqlc.GetMaskAnswersByItemAndValueParams{
		ItemCode: itemCode,
		Mask:     mask,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching mask answers for item %d: %w", itemCode, err)
	}

	out := make([]maskvo.MaskAnswer, 0, len(rows))

	for _, row := range rows {

		answer, err := maskvo.NewMaskAnswer(
			row.QuestionID,
			row.OptionID,
			int(row.Position),
			row.OptionValue,
		)
		if err != nil {
			return nil, fmt.Errorf("invalid mask answer from DB: %w", err)
		}

		out = append(out, answer)
	}

	return out, nil
}
func (r *StructureQueryRepositorySQLC) GetItemQuestions(
	ctx context.Context,
	itemCode int64,
) ([]maskservice.ItemQuestion, error) {

	rows, err := r.q.GetItemQuestions(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("fetching item questions for item %d: %w", itemCode, err)
	}

	out := make([]maskservice.ItemQuestion, 0, len(rows))

	for _, row := range rows {
		out = append(out, maskservice.ItemQuestion{
			QuestionID: row.QuestionID,
			Position:   row.Position,
		})
	}

	return out, nil
}

func maskHash(mask string) string {
	h := sha256.Sum256([]byte(mask))
	return hex.EncodeToString(h[:])[:8]
}
