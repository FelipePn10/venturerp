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
	StartDate          pgtype.Date
	EndDate            pgtype.Date
	// A quantidade da linha está na unidade da ESTRUTURA; ConversionFactor leva
	// para a unidade de estoque. Sem ele o MRP planejava "2" para uma chapa
	// desenhada em m² e estocada em kg — reservava 2 kg onde a engenharia
	// pediu 2 m² (31,4 kg). O erro é multiplicativo e não aparece em lugar
	// nenhum: aparece no estoque, meses depois.
	ConversionFactor pgtype.Numeric
	// A quantidade também pode vir de fórmula (item configurado). Sem estes
	// campos `ResolvedQuantity` caía sempre na quantidade fixa e a explosão
	// ignorava a configuração.
	QuantityFormula  pgtype.Text
	QuantityRounding string
	QuantityScale    int16
}

// loadBOMForRoots fetches the complete BOM tree for a set of root items in a
// single recursive query. UNION (not UNION ALL) prevents infinite loops if a
// cycle accidentally exists in the data.
const loadBOMForRoots = `
WITH RECURSIVE bom_tree AS (
    SELECT parent_code, child_code, quantity, loss_percentage, parent_mask, is_coproduct, is_fixed_qty, substitute_group, substitute_priority, start_date, end_date, conversion_factor, quantity_formula, quantity_rounding, quantity_scale
    FROM item_structures
    WHERE parent_code = ANY($1::bigint[]) AND is_active = TRUE

    UNION

    SELECT s.parent_code, s.child_code, s.quantity, s.loss_percentage, s.parent_mask, s.is_coproduct, s.is_fixed_qty, s.substitute_group, s.substitute_priority, s.start_date, s.end_date, s.conversion_factor, s.quantity_formula, s.quantity_rounding, s.quantity_scale
    FROM item_structures s
    INNER JOIN bom_tree bt ON s.parent_code = bt.child_code
    WHERE s.is_active = TRUE
)
SELECT parent_code, child_code, quantity, loss_percentage, parent_mask, is_coproduct, is_fixed_qty, substitute_group, substitute_priority, start_date, end_date, conversion_factor, quantity_formula, quantity_rounding, quantity_scale FROM bom_tree
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
			&e.StartDate,
			&e.EndDate,
			&e.ConversionFactor,
			&e.QuantityFormula,
			&e.QuantityRounding,
			&e.QuantityScale,
		); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}
