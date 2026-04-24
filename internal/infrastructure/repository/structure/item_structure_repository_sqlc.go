package structure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service"
	maskvo "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
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
		Sequence:          int32(s.Sequence),
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

	row, err := r.q.UpdateStructureComponent(
		ctx,
		sqlc.UpdateStructureComponentParams{
			ParentCode:        s.ParentCode,
			ChildCode:         s.ChildCode,
			ParentMask:        toNullString(s.ParentMask),
			Quantity:          s.Quantity,
			UnitOfMeasurement: sqlc.UnitOfMeasurementEnum(s.UnitOfMeasurement),
			LossPercentage:    s.LossPercentage,
			Sequence:          int32(s.Sequence),
			Health:            sqlc.HealthEnum(s.Health),
			Notes:             toNullString(s.Notes),
		},
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf(
				"component not found (parent=%d, child=%d, mask=%v) or inactive",
				s.ParentCode,
				s.ChildCode,
				s.ParentMask,
			)
		}
		return nil, fmt.Errorf(
			"updating component (parent=%d, child=%d): %w",
			s.ParentCode,
			s.ChildCode,
			err,
		)
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
) ([]*entity.ItemStructure, error) {

	rows, err := r.q.GetAllDirectChildren(ctx, parentCode)
	if err != nil {
		return nil, fmt.Errorf("searching for children of item %d: %w", parentCode, err)
	}

	result := make([]*entity.ItemStructure, 0, len(rows))

	for _, row := range rows {

		var notes *string
		if row.Notes.Valid {
			notes = &row.Notes.String
		}

		var parentMask *string
		if row.ParentMask.Valid {
			parentMask = &row.ParentMask.String
		}

		result = append(result, &entity.ItemStructure{
			ID:         row.ID,
			ParentCode: row.ParentCode,
			ChildCode:  row.ChildCode,

			// regra de domínio: inherit = se NÃO tem máscara (genérico)
			Inherit: !row.ParentMask.Valid,

			ParentMask:     parentMask,
			Quantity:       row.Quantity,
			LossPercentage: row.LossPercentage,

			UnitOfMeasurement: types.TypeUnitOfMeasurementItem(row.UnitOfMeasurement),
			Health:            types.Health(row.Health),

			Sequence: int(row.Sequence),

			Notes: notes,

			IsActive:  row.IsActive,
			CreatedBy: row.CreatedBy,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}

	return result, nil
}

//func (r *ItemStructureRepositorySQLC) GetGenericChildren(
//	ctx context.Context,
//	parentCode int64,
//) ([]*entity.ItemStructure, error) {
//	rows, err := r.q.GetGenericChildren(ctx, parentCode)
//	if err != nil {
//		return nil, fmt.Errorf("searching for generic children of the item %d: %w", parentCode, err)
//	}
//	return rowsToEntities(rows), nil
//}

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
	return rowsWithDescToEntities(rows), nil
}

func rowsWithDescToEntities(rows []sqlc.GetDirectChildrenForMaskRow) []*entity.ItemStructure {
	out := make([]*entity.ItemStructure, 0, len(rows))
	for _, row := range rows {
		e := &entity.ItemStructure{
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

func (r *ItemStructureRepositorySQLC) SequenceExists(
	ctx context.Context,
	parentCode int64,
	sequence int,
) (bool, error) {
	exists, err := r.q.SequenceExists(ctx, sqlc.SequenceExistsParams{
		ParentCode: parentCode,
		Sequence:   int32(sequence),
	})
	if err != nil {
		return false, fmt.Errorf("checking sequence %d for item %d: %w", sequence, parentCode, err)
	}
	return exists, nil
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
) ([]maskvo.MaskAnswer, error) {

	rows, err := r.q.GetItemMaskAnswersByValue(ctx, sqlc.GetItemMaskAnswersByValueParams{
		ItemCode: itemCode,
		Mask:     maskValue,
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching mask answers for item %d: %w", itemCode, err)
	}

	answers := make([]maskvo.MaskAnswer, 0, len(rows))

	for _, row := range rows {

		answer, err := maskvo.NewMaskAnswer(
			row.QuestionID,
			row.OptionID,
			int(row.Position),
			"", // valor não necessário aqui
		)
		if err != nil {
			return nil, err
		}

		answers = append(answers, answer)
	}

	return answers, nil
}
func (r *ItemStructureRepositorySQLC) GetItemQuestions(
	ctx context.Context,
	itemCode int64,
) ([]service.ItemQuestion, error) {

	rows, err := r.q.GetItemQuestions(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("error fetching questions for item %d: %w", itemCode, err)
	}

	questions := make([]service.ItemQuestion, 0, len(rows))

	for _, row := range rows {
		questions = append(questions, service.ItemQuestion{
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
		Sequence:          int(row.Sequence),
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
