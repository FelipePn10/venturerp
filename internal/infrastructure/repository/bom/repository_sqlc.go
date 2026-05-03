package bom

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/FelipePn10/panossoerp/internal/domain/bom/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *repositoryBomSQLC) Create(
	ctx context.Context,
	bom *entity.Bom,
) (*entity.Bom, error) {

	params := sqlc.CreateBomParams{
		ProductID: bom.ProductId,
		BomType:   bom.BomType,
		Mask:      bom.MaskID,
		Version:   int32(bom.Version),
		ValidFrom: pgtype.Date{},
		Status:    bom.Status,
	}

	_, err := r.q.CreateBom(ctx, params)
	if err != nil {
		return nil, err
	}

	return bom, nil
}
