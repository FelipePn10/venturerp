package structure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/valueobject"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *ItemStructureRepositorySQLC) Create(
	ctx context.Context,
	s *entity.ItemStructure,
) (*entity.ItemStructure, error) {
	row, err := r.q.CreateStructureComponent(ctx, sqlc.CreateStructureComponentParams{
		ParentCode:        s.ParentCode,
		ChildCode:         s.ChildCode,
		ParentMask:        toNullString(s.ParentMask),
		Quantity:          s.Quantity,
		UnitOfMeasurement: sqlc.UnitOfMeasurementEnum(s.UnitOfMeasurement),
		Health:            sqlc.HealthEnum(s.Health),
		LossPercentage:    s.LossPercentage,
		Position:          int32(s.Sequence),
		Notes:             toNullString(s.Notes),
		CreatedBy:         s.CreatedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("creating structural component: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *ItemStructureRepositorySQLC) Update(
	ctx context.Context,
	s *entity.ItemStructure,
) (*entity.ItemStructure, error) {
	row, err := r.q.UpdateStructureComponent(ctx, sqlc.UpdateStructureComponentParams{
		ID:                s.ID,
		Quantity:          s.Quantity,
		UnitOfMeasurement: sqlc.UnitOfMeasurementEnum(s.UnitOfMeasurement),
		Health:            sqlc.HealthEnum(s.Health),
		LossPercentage:    s.LossPercentage,
		Position:          int32(s.Sequence),
		Notes:             toNullString(s.Notes),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("component %d not found or inactiveo", s.ID)
		}
		return nil, fmt.Errorf("updating component %d: %w", s.ID, err)
	}
	return rowToEntity(row), nil
}

func (r *ItemStructureRepositorySQLC) Delete(ctx context.Context, code int64) error {
	if err := r.q.DeactivateStructureComponent(ctx, code); err != nil {
		return fmt.Errorf("disabling component %d: %w", code, err)
	}
	return nil
}

func (r *ItemStructureRepositorySQLC) GetByID(ctx context.Context, id int64) (*entity.ItemStructure, error) {
	row, err := r.q.GetStructureComponentByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("component %d not found", id)
		}
		return nil, fmt.Errorf("searching for component %d: %w", id, err)
	}
	return rowToEntity(row), nil
}

func (r *ItemStructureRepositorySQLC) GetAllDirectChildren(
	ctx context.Context,
	parentCode int64,
) ([]*response.StructureComponentResponse, error) {

	rows, err := r.q.GetAllDirectChildren(ctx, parentCode)
	if err != nil {
		return nil, fmt.Errorf("searching for children of item %d: %w", parentCode, err)
	}

	result := make([]*response.StructureComponentResponse, 0, len(rows))

	for _, row := range rows {

		var notes *string
		if row.Notes.Valid {
			notes = &row.Notes.String
		}

		var parentMask *string
		if row.ParentMask.Valid {
			parentMask = &row.ParentMask.String
		}

		result = append(result, &response.StructureComponentResponse{
			ID:               row.ID,
			ParentItemCode:   row.ParentCode,
			ChildItemCode:    row.ChildCode,
			ChildDescription: row.ChildDescription,

			ParentMask: parentMask,
			IsGeneric:  !row.ParentMask.Valid,

			Quantity:          row.Quantity,
			EffectiveQuantity: row.Quantity * (1 + row.LossPercentage/100),

			UnitOfMeasurement: types.TypeUnitOfMeasurementItem(row.UnitOfMeasurement),
			Health:            types.Health(row.Health),

			LossPercentage: row.LossPercentage,
			Position:       int(row.Position),

			Notes: notes,

			IsActive:  row.IsActive,
			CreatedBy: row.CreatedBy,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}

	return result, nil
}

func (r *ItemStructureRepositorySQLC) GetGenericChildren(
	ctx context.Context,
	parentCode int64,
) ([]*entity.ItemStructure, error) {
	rows, err := r.q.GetGenericChildren(ctx, parentCode)
	if err != nil {
		return nil, fmt.Errorf("searching for generic children of the item %d: %w", parentCode, err)
	}
	return rowsToEntities(rows), nil
}

func (r *ItemStructureRepositorySQLC) GetDirectChildrenForMask(
	ctx context.Context,
	parentCode int64,
	mask string,
) ([]*entity.ItemStructure, error) {
	rows, err := r.q.GetDirectChildrenForMask(ctx, sqlc.GetDirectChildrenForMaskParams{
		ParentCode: parentCode,
		ParentMask: sql.NullString{String: mask, Valid: mask != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("searching for children of item %d for mask '%s': %w", parentCode, mask, err)
	}
	return rowsToEntities(rows), nil
}

func (r *ItemStructureRepositorySQLC) ItemExists(ctx context.Context, itemCode int64) (bool, error) {
	exists, err := r.q.ItemExists(ctx, itemCode)
	if err != nil {
		return false, fmt.Errorf("checking for the item's existence %d: %w", itemCode, err)
	}
	return exists, nil
}

func (r *ItemStructureRepositorySQLC) HasCyclicReference(
	ctx context.Context,
	parentCode, childCode int64,
) (bool, error) {

	hasCycle, err := r.q.HasCyclicReference(ctx, sqlc.HasCyclicReferenceParams{
		StartCode:  childCode,  // $1
		TargetCode: parentCode, // $2
	})
	if err != nil {
		return false, fmt.Errorf(
			"checking cycle between parent=%d and child=%d: %w",
			parentCode,
			childCode,
			err,
		)
	}

	return hasCycle, nil
}

func (r *ItemStructureRepositorySQLC) GetItemCodeAndDesc(
	ctx context.Context,
	itemCode int64,
) (int64, string, error) {

	row, err := r.q.GetItemCodeAndDescription(ctx, itemCode)
	if err != nil {
		return 0, "", err
	}

	return row.Code, row.Description, nil
}

// GetMaskAnswersByItemAndValue retorna as respostas de uma máscara específica.
// O campo OptionID (int64) é mapeado para o value object MaskAnswer.
// A resolução do valor textual da opção (se necessário para montar a máscara
// do filho) deve ser feita via join ou lookup na camada de aplicação quando
// necessário — aqui devolvemos o OptionID que é suficiente para identificar
// a resposta e propagar a máscara.
func (r *ItemStructureRepositorySQLC) GetMaskAnswersByItemAndValue(
	ctx context.Context,
	itemCode int64,
	maskValue string,
) ([]valueobject.MaskAnswer, error) {
	rows, err := r.q.GetItemMaskAnswersByValue(ctx, sqlc.GetItemMaskAnswersByValueParams{
		ItemCode: itemCode,
		Mask:     maskValue,
	})
	if err != nil {
		return nil, fmt.Errorf("searching for answers from the '%s' mask of the item %d: %w", maskValue, itemCode, err)
	}

	answers := make([]valueobject.MaskAnswer, 0, len(rows))
	for _, row := range rows {
		answers = append(answers, valueobject.MaskAnswer{
			QuestionID: row.QuestionID,
			Position:   row.Position,
			OptionID:   row.OptionID,
		})
	}
	return answers, nil
}

func (r *ItemStructureRepositorySQLC) GetItemQuestions(
	ctx context.Context,
	itemID int64,
) ([]valueobject.ItemQuestion, error) {
	rows, err := r.q.GetItemQuestions(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("searching for questions from the item %d: %w", itemID, err)
	}

	questions := make([]valueobject.ItemQuestion, 0, len(rows))
	for _, row := range rows {
		questions = append(questions, valueobject.ItemQuestion{
			QuestionID: row.QuestionID,
			Position:   row.Position,
		})
	}
	return questions, nil
}

// Mappers internos básicos

func rowToEntity(row sqlc.ItemStructure) *entity.ItemStructure {
	e := &entity.ItemStructure{
		ID:                row.ID,
		Quantity:          row.Quantity,
		UnitOfMeasurement: types.TypeUnitOfMeasurementItem(row.UnitOfMeasurement),
		LossPercentage:    row.LossPercentage,
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
	return e
}

func rowsToEntities(rows []sqlc.ItemStructure) []*entity.ItemStructure {
	out := make([]*entity.ItemStructure, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out
}

func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}
