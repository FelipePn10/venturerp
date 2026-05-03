package bomitem

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/bom_items/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *repositoryBomItemSQLC) Create(
	ctx context.Context,
	bomitems *entity.BomItems,
) (*entity.BomItems, error) {

	params := sqlc.CreateBomItemParams{
		BomID:         bomitems.BomID,
		ComponentID:   bomitems.ComponentID,
		Quantity:      pgutil.ToPgNumericFromString(bomitems.Quantity.String()),
		Uom:           pgutil.ToPgTextFromString(bomitems.Uom),
		ScrapPercent:  pgutil.ToPgNumericFromString(bomitems.Quantity.String()),
		OperationID:   bomitems.OperationID,
		MaskComponent: bomitems.MaskComponent,
	}

	if _, err := r.q.CreateBomItem(ctx, params); err != nil {
		return nil, err
	}

	return bomitems, nil
}
