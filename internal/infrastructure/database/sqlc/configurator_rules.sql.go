package sqlc

// Hand-written data access for the configurator rule engines (Fase 5):
// equivalent-variable rules (parent→child) and configured-item rules.

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type cfgRuleScanner interface{ Scan(...any) error }

// ─── cfg_equivalent_rules ─────────────────────────────────────────────────────

type DBCfgEquivalentRule struct {
	ID                     int64
	ParentItemCode         int64
	ParentUom              pgtype.Text
	ChildItemCode          int64
	ChildSeq               pgtype.Int4
	ParentCharacteristicID int64
	ParentOperator         string
	ParentVariableID       pgtype.Int8
	ChildCharacteristicID  int64
	ChildOperator          string
	ChildVariableID        pgtype.Int8
	Formula                pgtype.Text
	IsActive               bool
	CreatedAt              pgtype.Timestamptz
	CreatedBy              pgtype.UUID
	// denormalized
	ParentVariableCode pgtype.Text
	ChildVariableCode  pgtype.Text
}

const cfgEquivCols = `r.id, r.parent_item_code, r.parent_uom, r.child_item_code, r.child_seq,
	r.parent_characteristic_id, r.parent_operator, r.parent_variable_id,
	r.child_characteristic_id, r.child_operator, r.child_variable_id,
	r.formula, r.is_active, r.created_at, r.created_by, pv.code, cv.code`

const cfgEquivFrom = `FROM cfg_equivalent_rules r
	LEFT JOIN cfg_variables pv ON pv.id = r.parent_variable_id
	LEFT JOIN cfg_variables cv ON cv.id = r.child_variable_id`

func scanEquivRule(s cfgRuleScanner) (DBCfgEquivalentRule, error) {
	var i DBCfgEquivalentRule
	err := s.Scan(&i.ID, &i.ParentItemCode, &i.ParentUom, &i.ChildItemCode, &i.ChildSeq,
		&i.ParentCharacteristicID, &i.ParentOperator, &i.ParentVariableID,
		&i.ChildCharacteristicID, &i.ChildOperator, &i.ChildVariableID,
		&i.Formula, &i.IsActive, &i.CreatedAt, &i.CreatedBy, &i.ParentVariableCode, &i.ChildVariableCode)
	return i, err
}

type CfgEquivalentRuleParams struct {
	ID                     int64
	ParentItemCode         int64
	ParentUom              pgtype.Text
	ChildItemCode          int64
	ChildSeq               pgtype.Int4
	ParentCharacteristicID int64
	ParentOperator         string
	ParentVariableID       pgtype.Int8
	ChildCharacteristicID  int64
	ChildOperator          string
	ChildVariableID        pgtype.Int8
	Formula                pgtype.Text
	CreatedBy              pgtype.UUID
}

func (q *Queries) CreateCfgEquivalentRule(ctx context.Context, a CfgEquivalentRuleParams) (DBCfgEquivalentRule, error) {
	const ins = `WITH r AS (
		INSERT INTO cfg_equivalent_rules
		(parent_item_code, parent_uom, child_item_code, child_seq, parent_characteristic_id, parent_operator,
		 parent_variable_id, child_characteristic_id, child_operator, child_variable_id, formula, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING *
	) SELECT ` + cfgEquivCols + ` FROM r
		LEFT JOIN cfg_variables pv ON pv.id = r.parent_variable_id
		LEFT JOIN cfg_variables cv ON cv.id = r.child_variable_id`
	return scanEquivRule(q.db.QueryRow(ctx, ins, a.ParentItemCode, a.ParentUom, a.ChildItemCode, a.ChildSeq,
		a.ParentCharacteristicID, a.ParentOperator, a.ParentVariableID, a.ChildCharacteristicID, a.ChildOperator,
		a.ChildVariableID, a.Formula, a.CreatedBy))
}

func (q *Queries) UpdateCfgEquivalentRule(ctx context.Context, a CfgEquivalentRuleParams) (DBCfgEquivalentRule, error) {
	const upd = `WITH r AS (
		UPDATE cfg_equivalent_rules SET parent_item_code=$2, parent_uom=$3, child_item_code=$4, child_seq=$5,
		 parent_characteristic_id=$6, parent_operator=$7, parent_variable_id=$8, child_characteristic_id=$9,
		 child_operator=$10, child_variable_id=$11, formula=$12, updated_at=NOW()
		WHERE id=$1 RETURNING *
	) SELECT ` + cfgEquivCols + ` FROM r
		LEFT JOIN cfg_variables pv ON pv.id = r.parent_variable_id
		LEFT JOIN cfg_variables cv ON cv.id = r.child_variable_id`
	return scanEquivRule(q.db.QueryRow(ctx, upd, a.ID, a.ParentItemCode, a.ParentUom, a.ChildItemCode, a.ChildSeq,
		a.ParentCharacteristicID, a.ParentOperator, a.ParentVariableID, a.ChildCharacteristicID, a.ChildOperator,
		a.ChildVariableID, a.Formula))
}

func (q *Queries) GetCfgEquivalentRule(ctx context.Context, id int64) (DBCfgEquivalentRule, error) {
	return scanEquivRule(q.db.QueryRow(ctx, `SELECT `+cfgEquivCols+` `+cfgEquivFrom+` WHERE r.id=$1`, id))
}

func (q *Queries) ListCfgEquivalentRulesByParent(ctx context.Context, parentItemCode int64, onlyActive bool) ([]DBCfgEquivalentRule, error) {
	const sql = `SELECT ` + cfgEquivCols + ` ` + cfgEquivFrom + `
		WHERE r.parent_item_code=$1 AND ($2::BOOLEAN = FALSE OR r.is_active = TRUE) ORDER BY r.id`
	rows, err := q.db.Query(ctx, sql, parentItemCode, onlyActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBCfgEquivalentRule
	for rows.Next() {
		r, err := scanEquivRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Queries) DeactivateCfgEquivalentRule(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `UPDATE cfg_equivalent_rules SET is_active=FALSE, updated_at=NOW() WHERE id=$1`, id)
	return err
}

// ─── cfg_item_rules ───────────────────────────────────────────────────────────

type DBCfgItemRule struct {
	ID          int64
	ItemCode    int64
	TargetTable string
	TargetField string
	Content     pgtype.Text
	Formula     pgtype.Text
	Description pgtype.Text
	Situation   string
	CreatedAt   pgtype.Timestamptz
	CreatedBy   pgtype.UUID
}

const cfgItemRuleCols = `id, item_code, target_table, target_field, content, formula, description, situation, created_at, created_by`

func scanItemRule(s cfgRuleScanner) (DBCfgItemRule, error) {
	var i DBCfgItemRule
	err := s.Scan(&i.ID, &i.ItemCode, &i.TargetTable, &i.TargetField, &i.Content, &i.Formula,
		&i.Description, &i.Situation, &i.CreatedAt, &i.CreatedBy)
	return i, err
}

type CfgItemRuleParams struct {
	ID          int64
	ItemCode    int64
	TargetTable string
	TargetField string
	Content     pgtype.Text
	Formula     pgtype.Text
	Description pgtype.Text
	Situation   string
	CreatedBy   pgtype.UUID
}

func (q *Queries) CreateCfgItemRule(ctx context.Context, a CfgItemRuleParams) (DBCfgItemRule, error) {
	const sql = `INSERT INTO cfg_item_rules
		(item_code, target_table, target_field, content, formula, description, situation, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING ` + cfgItemRuleCols
	return scanItemRule(q.db.QueryRow(ctx, sql, a.ItemCode, a.TargetTable, a.TargetField, a.Content,
		a.Formula, a.Description, a.Situation, a.CreatedBy))
}

func (q *Queries) UpdateCfgItemRule(ctx context.Context, a CfgItemRuleParams) (DBCfgItemRule, error) {
	const sql = `UPDATE cfg_item_rules SET target_table=$2, target_field=$3, content=$4, formula=$5,
		description=$6, situation=$7, updated_at=NOW() WHERE id=$1 RETURNING ` + cfgItemRuleCols
	return scanItemRule(q.db.QueryRow(ctx, sql, a.ID, a.TargetTable, a.TargetField, a.Content,
		a.Formula, a.Description, a.Situation))
}

func (q *Queries) GetCfgItemRule(ctx context.Context, id int64) (DBCfgItemRule, error) {
	return scanItemRule(q.db.QueryRow(ctx, `SELECT `+cfgItemRuleCols+` FROM cfg_item_rules WHERE id=$1`, id))
}

func (q *Queries) ListCfgItemRulesByItem(ctx context.Context, itemCode int64, onlyActive bool) ([]DBCfgItemRule, error) {
	const sql = `SELECT ` + cfgItemRuleCols + ` FROM cfg_item_rules
		WHERE item_code=$1 AND ($2::BOOLEAN = FALSE OR situation = 'ACTIVE') ORDER BY id`
	rows, err := q.db.Query(ctx, sql, itemCode, onlyActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBCfgItemRule
	for rows.Next() {
		r, err := scanItemRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Queries) DeleteCfgItemRule(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `DELETE FROM cfg_item_rules WHERE id=$1`, id)
	return err
}

// ─── cfg_item_rule_conditions ─────────────────────────────────────────────────

type DBCfgItemRuleCondition struct {
	ID               int64
	RuleID           int64
	CharacteristicID int64
	Operator         string
	VariableID       pgtype.Int8
	Sequence         int32
	// denormalized
	VariableCode pgtype.Text
}

func (q *Queries) AddCfgItemRuleCondition(ctx context.Context, ruleID, charID int64, operator string, variableID pgtype.Int8, sequence int32) (DBCfgItemRuleCondition, error) {
	const sql = `WITH c AS (
		INSERT INTO cfg_item_rule_conditions (rule_id, characteristic_id, operator, variable_id, sequence)
		VALUES ($1,$2,$3,$4,$5) RETURNING *
	) SELECT c.id, c.rule_id, c.characteristic_id, c.operator, c.variable_id, c.sequence, v.code
		FROM c LEFT JOIN cfg_variables v ON v.id = c.variable_id`
	var i DBCfgItemRuleCondition
	err := q.db.QueryRow(ctx, sql, ruleID, charID, operator, variableID, sequence).
		Scan(&i.ID, &i.RuleID, &i.CharacteristicID, &i.Operator, &i.VariableID, &i.Sequence, &i.VariableCode)
	return i, err
}

func (q *Queries) ListCfgItemRuleConditions(ctx context.Context, ruleID int64) ([]DBCfgItemRuleCondition, error) {
	const sql = `SELECT c.id, c.rule_id, c.characteristic_id, c.operator, c.variable_id, c.sequence, v.code
		FROM cfg_item_rule_conditions c LEFT JOIN cfg_variables v ON v.id = c.variable_id
		WHERE c.rule_id=$1 ORDER BY c.sequence, c.id`
	rows, err := q.db.Query(ctx, sql, ruleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBCfgItemRuleCondition
	for rows.Next() {
		var i DBCfgItemRuleCondition
		if err := rows.Scan(&i.ID, &i.RuleID, &i.CharacteristicID, &i.Operator, &i.VariableID, &i.Sequence, &i.VariableCode); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (q *Queries) DeleteCfgItemRuleConditionsByRule(ctx context.Context, ruleID int64) error {
	_, err := q.db.Exec(ctx, `DELETE FROM cfg_item_rule_conditions WHERE rule_id=$1`, ruleID)
	return err
}
