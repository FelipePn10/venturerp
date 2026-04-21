package structure_query

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	maskservice "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service"
	maskvo "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	str "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/google/uuid"
)

func (r *StructureQueryRepositorySQLC) GetItemByCode(ctx context.Context, code int64) (*itementity.Item, error) {
	row, err := r.q.GetItemByCode(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("item %d not found", code)
		}
		return nil, fmt.Errorf("fetching item %d: %w", code, err)
	}
	// Apenas Inherit é necessário para o resolver; outros campos ficam zerados.
	return &itementity.Item{Inherit: row.Inherit}, nil
}

// GetDirectChildrenForMask usa a query de structure.sql (já existente).
// mask="" → sql.NullString{Valid:false} → PostgreSQL recebe NULL → só filhos universais.
// mask="1.94M#1.94M" → retorna universais + específicos para essa máscara.
func (r *StructureQueryRepositorySQLC) GetDirectChildrenForMask(
	ctx context.Context,
	parentCode int64,
	mask string,
) ([]*str.ItemStructure, error) {
	rows, err := r.q.GetDirectChildrenForMask(ctx, sqlc.GetDirectChildrenForMaskParams{
		ParentCode: parentCode,
		ParentMask: sql.NullString{String: mask, Valid: mask != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("fetching children of item %d (mask=%q): %w", parentCode, mask, err)
	}
	return structureQueryRowsToEntities(rows), nil
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
			Sequence:          int(row.Position),
			IsActive:          row.IsActive,
			CreatedBy:         row.CreatedBy,
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
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

// GetMaskAnswersByItemAndValue usa a query de structure_query.sql, que inclui
// option_value via JOIN. Sem esse valor, PropagateMask gera máscaras incorretas.
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
		return nil, fmt.Errorf("fetching mask answers for item %d mask %q: %w", itemCode, mask, err)
	}

	answers := make([]maskvo.MaskAnswer, 0, len(rows))
	for _, row := range rows {
		a, err := maskvo.NewMaskAnswer(
			row.QuestionID,
			row.OptionID,
			int(row.Position),
			row.OptionValue, // campo gerado pelo SQLC a partir do JOIN com question_options
		)
		if err != nil {
			return nil, fmt.Errorf("building mask answer (question=%d option=%d): %w",
				row.QuestionID, row.OptionID, err)
		}
		answers = append(answers, a)
	}
	return answers, nil
}

// GetItemQuestions usa a query de structure.sql (corrigida para item_code).
func (r *StructureQueryRepositorySQLC) GetItemQuestions(
	ctx context.Context,
	itemCode int64,
) ([]maskservice.ItemQuestion, error) {
	rows, err := r.q.GetItemQuestions(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("fetching questions for item %d: %w", itemCode, err)
	}

	qs := make([]maskservice.ItemQuestion, 0, len(rows))
	for _, row := range rows {
		qs = append(qs, maskservice.ItemQuestion{
			QuestionID: row.QuestionID,
			Position:   row.Position,
		})
	}
	return qs, nil
}

// CreateMaskForItem cria automaticamente uma máscara propagada.
// As respostas chegam prontas do DeriveChildAnswers — sem reverse-lookup no banco.
func (r *StructureQueryRepositorySQLC) CreateMaskForItem(
	ctx context.Context,
	itemCode int64,
	mask string,
	answers []maskservice.ChildMaskAnswerInput,
	createdBy uuid.UUID,
) error {
	// InsertItemtMask — nome gerado pelo SQLC a partir de "-- name: InsertItemtMask :one"
	m, err := r.q.InsertItemtMask(ctx, sqlc.InsertItemtMaskParams{
		ItemCode:  itemCode,
		Mask:      mask,
		MaskHash:  maskHash(mask),
		CreatedBy: createdBy,
	})
	if err != nil {
		return fmt.Errorf("inserting mask %q for item %d: %w", mask, itemCode, err)
	}

	for _, a := range answers {
		err := r.q.InsertItemMaskAnswer(ctx, sqlc.InsertItemMaskAnswerParams{
			MaskID:     m.ID,
			QuestionID: a.QuestionID,
			OptionID:   a.OptionID,
			Position:   a.Position,
		})
		if err != nil {
			return fmt.Errorf("inserting answer (q=%d opt=%d) for mask %d: %w",
				a.QuestionID, a.OptionID, m.ID, err)
		}
	}
	return nil
}

func maskHash(mask string) string {
	h := sha256.Sum256([]byte(mask))
	return hex.EncodeToString(h[:])[:8]
}

func structureRowsToEntities(rows []sqlc.ItemStructure) []*str.ItemStructure {
	out := make([]*str.ItemStructure, 0, len(rows))
	for _, row := range rows {
		e := &str.ItemStructure{
			ID:                row.ID,
			ParentCode:        row.ParentCode,
			ChildCode:         row.ChildCode,
			Quantity:          row.Quantity,
			LossPercentage:    row.LossPercentage,
			UnitOfMeasurement: types.TypeUnitOfMeasurementItem(row.UnitOfMeasurement),
			Health:            types.Health(row.Health),
			Sequence:          int(row.Position),
			IsActive:          row.IsActive,
			CreatedBy:         row.CreatedBy,
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
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
