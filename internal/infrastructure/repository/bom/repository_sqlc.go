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

	validFrom := pgtype.Date{}
	if !bom.ValidFrom.IsZero() {
		validFrom = pgtype.Date{Time: bom.ValidFrom, Valid: true}
	}

	params := sqlc.CreateBomParams{
		ProductID: bom.ProductId,
		BomType:   bom.BomType,
		Mask:      bom.MaskID,
		Version:   int32(bom.Version),
		ValidFrom: validFrom,
		Status:    bom.Status,
	}

	created, err := r.q.CreateBom(ctx, params)
	if err != nil {
		return nil, err
	}

	bom.ID = created.ID
	if created.ValidFrom.Valid {
		bom.ValidFrom = created.ValidFrom.Time
	}
	bom.CreatedAt = created.CreatedAt.Time
	return bom, nil
}
