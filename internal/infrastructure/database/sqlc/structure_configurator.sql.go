package sqlc

// Consultas do painel do configurador embutido na Estrutura de Produto
// (VENT0210): tudo o que o botão precisa carregar de uma vez, sem tela própria.

import "context"

// CfgItemMaskRow é uma configuração já gerada para o item.
type CfgItemMaskRow struct {
	ID       int64
	Mask     string
	MaskHash string
	// Answered indica que a máscara tem respostas gravadas (veio do configurador).
	Answered bool
}

// ListCfgItemMasks lista as configurações já geradas para o item, da mais
// recente para a mais antiga.
func (q *Queries) ListCfgItemMasks(ctx context.Context, itemCode int64) ([]CfgItemMaskRow, error) {
	const sql = `SELECT im.id, im.mask, im.mask_hash,
		EXISTS(SELECT 1 FROM cfg_item_mask_answers a WHERE a.mask_id = im.id) AS answered
		FROM item_masks im
		WHERE im.item_code = $1
		ORDER BY im.created_at DESC`
	rows, err := q.db.Query(ctx, sql, itemCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CfgItemMaskRow, 0)
	for rows.Next() {
		var i CfgItemMaskRow
		if err := rows.Scan(&i.ID, &i.Mask, &i.MaskHash, &i.Answered); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// CfgStructureFormulaRow é um componente da estrutura que calcula a quantidade
// por fórmula.
type CfgStructureFormulaRow struct {
	ChildCode        int64
	ChildDescription string
	Formula          string
	Rounding         string
	Scale            int16
	NominalQuantity  float64
	UnitOfMeasure    string
}

// ListStructureQuantityFormulas lista os componentes do item cuja quantidade vem
// de fórmula — é a lista que a tela usa para mostrar o que a configuração
// alimenta.
func (q *Queries) ListStructureQuantityFormulas(ctx context.Context, parentCode int64) ([]CfgStructureFormulaRow, error) {
	const sql = `SELECT s.child_code, COALESCE(i.pdm_description_technique,''), s.quantity_formula,
		s.quantity_rounding, s.quantity_scale, s.quantity, s.unit_of_measurement::text
		FROM item_structures s
		LEFT JOIN items i ON i.code = s.child_code
		WHERE s.parent_code = $1 AND s.is_active AND s.quantity_formula IS NOT NULL
		ORDER BY s.sequence, s.id`
	rows, err := q.db.Query(ctx, sql, parentCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CfgStructureFormulaRow, 0)
	for rows.Next() {
		var i CfgStructureFormulaRow
		if err := rows.Scan(&i.ChildCode, &i.ChildDescription, &i.Formula, &i.Rounding, &i.Scale,
			&i.NominalQuantity, &i.UnitOfMeasure); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
