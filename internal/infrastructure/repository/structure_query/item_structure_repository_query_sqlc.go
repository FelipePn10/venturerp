package structure_query

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	maskservice "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service"
	maskvo "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject"
	str "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/formula"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/google/uuid"
)

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
			Inherit:           row.Inherit,
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

func (r *StructureQueryRepositorySQLC) ConsultChildren(
	ctx context.Context,
	parentCode int64,
	mask string,
	effectivenessDate *time.Time,
) ([]*str.ConsultRow, error) {

	rows, err := r.q.GetChildrenForConsult(ctx, sqlc.GetChildrenForConsultParams{
		ParentCode: parentCode,
		ParentMask: pgutil.ToPgText(mask),
		Column3:    pgutil.ToPgDateFromPtr(effectivenessDate),
	})
	if err != nil {
		return nil, fmt.Errorf("fetching consult children of item %d: %w", parentCode, err)
	}

	out := make([]*str.ConsultRow, 0, len(rows))
	for _, row := range rows {
		s := &str.ItemStructure{
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
			Inherit:           row.Inherit,
			CreatedBy:         pgutil.FromPgUUID(row.CreatedBy),
			CreatedAt:         pgutil.FromPgTimestamptz(row.CreatedAt),
			UpdatedAt:         pgutil.FromPgTimestamptz(row.UpdatedAt),
			StartDate:         pgutil.FromPgDateToPtr(row.StartDate),
			EndDate:           pgutil.FromPgDateToPtr(row.EndDate),
		}
		if row.ParentMask.Valid {
			v := row.ParentMask.String
			s.ParentMask = &v
		}
		if row.Notes.Valid {
			v := row.Notes.String
			s.Notes = &v
		}
		if row.LossFormula.Valid {
			v := row.LossFormula.String
			s.LossFormula = &v
		}
		out = append(out, &str.ConsultRow{
			ItemStructure: s,
			WarehouseCode: row.WarehouseCode,
			TypeStruct:    row.EngineeringTypeStruct,
		})
	}
	return out, nil
}

func (r *StructureQueryRepositorySQLC) GetMaskAnswersWithNames(
	ctx context.Context,
	itemCode int64,
	mask string,
) (map[string]float64, error) {

	rows, err := r.q.GetMaskAnswersWithNames(ctx, sqlc.GetMaskAnswersWithNamesParams{
		ItemCode: itemCode,
		Mask:     mask,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching mask answers with names for item %d: %w", itemCode, err)
	}

	vars := make(map[string]float64, len(rows))
	for _, row := range rows {
		if v, ok := formula.ParseOptionValue(row.OptionValue); ok {
			vars[row.QuestionName] = v
		}
	}
	return vars, nil
}

func (r *StructureQueryRepositorySQLC) GetLatestMaskForItem(
	ctx context.Context,
	itemCode int64,
) (string, error) {

	mask, err := r.q.GetProductMaskByItemCode(ctx, itemCode)
	if err != nil {
		return "", nil // sem máscara cadastrada = retorna vazio sem erro
	}
	return mask.Mask, nil
}

func (r *StructureQueryRepositorySQLC) GetWhereUsed(
	ctx context.Context,
	itemCode int64,
	levels int,
) ([]*str.WhereUsedRow, error) {

	rows, err := r.q.GetWhereUsed(ctx, sqlc.GetWhereUsedParams{
		ChildCode: itemCode,
		Column2:   int32(levels),
	})
	if err != nil {
		return nil, fmt.Errorf("fetching where-used for item %d: %w", itemCode, err)
	}

	out := make([]*str.WhereUsedRow, 0, len(rows))
	for _, row := range rows {
		wu := &str.WhereUsedRow{
			Level:             int(row.Level),
			ParentCode:        row.ParentCode,
			ChildCode:         row.ChildCode,
			ParentDescription: row.ParentDescription,
			Quantity:          pgutil.FromPgNumericToFloat64(row.Quantity),
			LossPercentage:    pgutil.FromPgNumericToFloat64(row.LossPercentage),
			Sequence:          int(row.Sequence),
		}
		if row.ParentMask.Valid {
			v := row.ParentMask.String
			wu.ParentMask = &v
		}
		out = append(out, wu)
	}
	return out, nil
}
