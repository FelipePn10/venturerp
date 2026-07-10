package sqlc

// cfg-based structure resolution — reads the configuration (item questions and
// mask answers) from the new cfg_* model, mapping characteristic_id → QuestionID
// and variable_id → OptionID. Used by the structure resolver as the primary path,
// (single source of truth after the legacy questions removal).

import "context"

// ItemUsesCfgConfigurator reports whether an item is configured via the new model
// (has cfg_item_characteristics). Governs cfg-primary vs legacy-fallback routing.
func (q *Queries) ItemUsesCfgConfigurator(ctx context.Context, itemCode int64) (bool, error) {
	var exists bool
	err := q.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cfg_item_characteristics WHERE item_code=$1)`, itemCode).Scan(&exists)
	return exists, err
}

type CfgStructItemQuestionRow struct {
	CharacteristicID int64
	Position         int32
}

func (q *Queries) CfgStructItemQuestions(ctx context.Context, itemCode int64) ([]CfgStructItemQuestionRow, error) {
	rows, err := q.db.Query(ctx, `SELECT characteristic_id, sequence
		FROM cfg_item_characteristics WHERE item_code=$1 ORDER BY sequence`, itemCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CfgStructItemQuestionRow
	for rows.Next() {
		var i CfgStructItemQuestionRow
		if err := rows.Scan(&i.CharacteristicID, &i.Position); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

type CfgStructMaskAnswerRow struct {
	CharacteristicID int64
	VariableID       int64
	AnswerValue      string
	Position         int32
}

// CfgStructMaskAnswers returns the choice answers of an item's mask (only rows
// with a variable — the ones that propagate down the BOM).
func (q *Queries) CfgStructMaskAnswers(ctx context.Context, itemCode int64, mask string) ([]CfgStructMaskAnswerRow, error) {
	const sql = `SELECT a.characteristic_id, a.variable_id, a.answer_value, a.position
		FROM item_masks im
		JOIN cfg_item_mask_answers a ON a.mask_id = im.id
		WHERE im.item_code=$1 AND im.mask=$2 AND a.variable_id IS NOT NULL
		ORDER BY a.position`
	rows, err := q.db.Query(ctx, sql, itemCode, mask)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CfgStructMaskAnswerRow
	for rows.Next() {
		var i CfgStructMaskAnswerRow
		if err := rows.Scan(&i.CharacteristicID, &i.VariableID, &i.AnswerValue, &i.Position); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

type CfgStructMaskAnswerNameRow struct {
	Code        string
	AnswerValue string
}

func (q *Queries) CfgStructMaskAnswersWithNames(ctx context.Context, itemCode int64, mask string) ([]CfgStructMaskAnswerNameRow, error) {
	const sql = `SELECT c.code, a.answer_value
		FROM item_masks im
		JOIN cfg_item_mask_answers a ON a.mask_id = im.id
		JOIN cfg_characteristics c ON c.id = a.characteristic_id
		WHERE im.item_code=$1 AND im.mask=$2
		ORDER BY a.position`
	rows, err := q.db.Query(ctx, sql, itemCode, mask)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CfgStructMaskAnswerNameRow
	for rows.Next() {
		var i CfgStructMaskAnswerNameRow
		if err := rows.Scan(&i.Code, &i.AnswerValue); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// GetCfgVariableMaskComposition returns a variable's mask composition (the value
// placed in the mask) — used when persisting a propagated cfg mask answer.
func (q *Queries) GetCfgVariableMaskComposition(ctx context.Context, variableID int64) (string, error) {
	var v string
	err := q.db.QueryRow(ctx, `SELECT mask_composition FROM cfg_variables WHERE id=$1`, variableID).Scan(&v)
	return v, err
}
