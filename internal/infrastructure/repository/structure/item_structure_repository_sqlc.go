package structure

import (
	"context"
	"fmt"

	maskservice "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service"
	maskvo "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *ItemStructureRepositorySQLC) Create(
	ctx context.Context,
	s *entity.ItemStructure,
) (*entity.ItemStructure, error) {

	row, err := r.q.CreateStructureComponent(ctx, sqlc.CreateStructureComponentParams{
		ParentCode:        s.ParentCode,
		ChildCode:         s.ChildCode,
		ParentMask:        stringPtrToPgText(s.ParentMask),
		Quantity:          s.Quantity,
		UnitOfMeasurement: sqlc.UnitOfMeasurementEnum(s.UnitOfMeasurement),
		Health:            sqlc.HealthEnum(s.Health),
		LossPercentage:    s.LossPercentage,
		Sequence:          int32(s.Sequence),
		Notes:             stringPtrToPgText(s.Notes),
		CreatedBy:         pgutil.ToPgUUID(s.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating structure: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *ItemStructureRepositorySQLC) Update(
	ctx context.Context,
	s *entity.ItemStructure,
) (*entity.ItemStructure, error) {

	row, err := r.q.UpdateStructureComponent(ctx, sqlc.UpdateStructureComponentParams{
		ParentCode:        s.ParentCode,
		ChildCode:         s.ChildCode,
		ParentMask:        stringPtrToPgText(s.ParentMask),
		Quantity:          s.Quantity,
		UnitOfMeasurement: sqlc.UnitOfMeasurementEnum(s.UnitOfMeasurement),
		Health:            sqlc.HealthEnum(s.Health),
		LossPercentage:    s.LossPercentage,
		Sequence:          int32(s.Sequence),
		Notes:             stringPtrToPgText(s.Notes),
	})
	if err != nil {
		return nil, fmt.Errorf("updating structure: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *ItemStructureRepositorySQLC) Delete(
	ctx context.Context,
	id int64,
) error {
	return r.q.DeactivateStructureComponent(ctx, id)
}

func (r *ItemStructureRepositorySQLC) GetByID(
	ctx context.Context,
	id int64,
) (*entity.ItemStructure, error) {

	row, err := r.q.GetStructureComponentByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching structure: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *ItemStructureRepositorySQLC) GetAllDirectChildren(
	ctx context.Context,
	parentCode int64,
) ([]*entity.ItemStructure, error) {

	rows, err := r.q.GetAllDirectChildren(ctx, parentCode)
	if err != nil {
		return nil, fmt.Errorf("fetching children: %w", err)
	}

	return mapDirectChildrenRows(rows), nil
}

func (r *ItemStructureRepositorySQLC) GetDirectChildrenForMask(
	ctx context.Context,
	parentCode int64,
	mask string,
) ([]*entity.ItemStructure, error) {

	rows, err := r.q.GetDirectChildrenForMask(ctx, sqlc.GetDirectChildrenForMaskParams{
		ParentCode: parentCode,
		ParentMask: pgutil.ToPgTextFromString(mask),
	})
	if err != nil {
		return nil, fmt.Errorf("fetching children by mask: %w", err)
	}

	return mapDirectChildrenWithMask(rows), nil
}

func (r *ItemStructureRepositorySQLC) ItemExists(
	ctx context.Context,
	itemCode int64,
) (bool, error) {
	return r.q.ItemExists(ctx, itemCode)
}

func (r *ItemStructureRepositorySQLC) HasCyclicReference(
	ctx context.Context,
	parentCode, childCode int64,
) (bool, error) {

	return r.q.HasCyclicReference(ctx, sqlc.HasCyclicReferenceParams{
		StartCode:  parentCode,
		TargetCode: childCode,
	})
}

func (r *ItemStructureRepositorySQLC) SequenceExists(
	ctx context.Context,
	parentCode int64,
	sequence int,
) (bool, error) {

	return r.q.SequenceExists(ctx, sqlc.SequenceExistsParams{
		ParentCode: parentCode,
		Sequence:   int32(sequence),
	})
}

func (r *ItemStructureRepositorySQLC) GetItemCodeAndDesc(
	ctx context.Context,
	itemCode int64,
) (int64, string, error) {

	row, err := r.q.GetItemCodeAndDescription(ctx, itemCode)
	if err != nil {
		return 0, "", fmt.Errorf("fetching item: %w", err)
	}

	return row.Code, row.Description, nil
}

func (r *ItemStructureRepositorySQLC) LoadBOMForRoots(
	ctx context.Context,
	rootCodes []int64,
) (map[int64][]*entity.ItemStructure, error) {

	edges, err := r.q.LoadBOMForRoots(ctx, rootCodes)
	if err != nil {
		return nil, fmt.Errorf("loading BOM for roots: %w", err)
	}

	adjacency := make(map[int64][]*entity.ItemStructure, len(edges))
	for _, e := range edges {
		child := &entity.ItemStructure{
			ParentCode:     e.ParentCode,
			ChildCode:      e.ChildCode,
			Quantity:       pgutil.FromPgNumericToFloat64(e.Quantity),
			LossPercentage: pgutil.FromPgNumericToFloat64(e.LossPercentage),
			IsActive:       true,
		}
		if e.ParentMask.Valid {
			v := e.ParentMask.String
			child.ParentMask = &v
		}
		adjacency[e.ParentCode] = append(adjacency[e.ParentCode], child)
	}
	return adjacency, nil
}

func (r *ItemStructureRepositorySQLC) GetMaskAnswersByItemAndValue(
	ctx context.Context,
	itemCode int64,
	maskValue string,
) ([]maskvo.MaskAnswer, error) {
	return nil, fmt.Errorf("not implemented: use StructureQueryRepository")
}

func (r *ItemStructureRepositorySQLC) GetItemQuestions(
	ctx context.Context,
	itemCode int64,
) ([]maskservice.ItemQuestion, error) {
	return nil, fmt.Errorf("not implemented: use StructureQueryRepository")
}

func rowToEntity(row sqlc.ItemStructure) *entity.ItemStructure {
	e := &entity.ItemStructure{
		ID:                row.ID,
		ParentCode:        row.ParentCode,
		ChildCode:         row.ChildCode,
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

	return e
}

func mapDirectChildrenRows(rows []sqlc.GetAllDirectChildrenRow) []*entity.ItemStructure {
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

func mapDirectChildrenWithMask(rows []sqlc.GetDirectChildrenForMaskRow) []*entity.ItemStructure {
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

func stringPtrToPgText(v *string) pgtype.Text {
	if v == nil {
		return pgtype.Text{}
	}
	return pgutil.ToPgText(*v)
}
