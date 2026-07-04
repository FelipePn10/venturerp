package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// StructureBOMEdge is a lightweight BOM row used for full-tree pre-loading.
// Written manually; will be absorbed by sqlc generate once the query is added.
type StructureBOMEdge struct {
	ParentCode         int64
	ChildCode          int64
	Quantity           pgtype.Numeric
	LossPercentage     pgtype.Numeric
	ParentMask         pgtype.Text
	IsCoproduct        bool
	IsFixedQty         bool
	SubstituteGroup    int16
	SubstitutePriority int16
}

// loadBOMForRoots fetches the complete BOM tree for a set of root items in a
// single recursive query. UNION (not UNION ALL) prevents infinite loops if a
// cycle accidentally exists in the data.
const loadBOMForRoots = `
WITH RECURSIVE bom_tree AS (
    SELECT parent_code, child_code, quantity, loss_percentage, parent_mask, is_coproduct, is_fixed_qty, substitute_group, substitute_priority
    FROM item_structures
    WHERE parent_code = ANY($1::bigint[]) AND is_active = TRUE

    UNION

    SELECT s.parent_code, s.child_code, s.quantity, s.loss_percentage, s.parent_mask, s.is_coproduct, s.is_fixed_qty, s.substitute_group, s.substitute_priority
    FROM item_structures s
    INNER JOIN bom_tree bt ON s.parent_code = bt.child_code
    WHERE s.is_active = TRUE
)
SELECT parent_code, child_code, quantity, loss_percentage, parent_mask, is_coproduct, is_fixed_qty, substitute_group, substitute_priority FROM bom_tree
`

func (q *Queries) LoadBOMForRoots(ctx context.Context, rootCodes []int64) ([]StructureBOMEdge, error) {
	rows, err := q.db.Query(ctx, loadBOMForRoots, rootCodes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []StructureBOMEdge
	for rows.Next() {
		var e StructureBOMEdge
		if err := rows.Scan(
			&e.ParentCode,
			&e.ChildCode,
			&e.Quantity,
			&e.LossPercentage,
			&e.ParentMask,
			&e.IsCoproduct,
			&e.IsFixedQty,
			&e.SubstituteGroup,
			&e.SubstitutePriority,
		); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}
