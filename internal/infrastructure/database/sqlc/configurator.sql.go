package sqlc

// Hand-written data access for the Product Configurator (Fase 1). Kept out of
// sqlc codegen (like production_order_operations.sql.go / tool_serials.sql.go)
// so the multi-entity model stays under our control.

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type cfgScanner interface{ Scan(...any) error }

// ─── cfg_sets ─────────────────────────────────────────────────────────────────

type DBCfgSet struct {
	ID          int64
	Description string
	IsActive    bool
	CreatedAt   pgtype.Timestamptz
	UpdatedAt   pgtype.Timestamptz
	CreatedBy   pgtype.UUID
	VariableQty int64
}

const cfgSetCols = `s.id, s.description, s.is_active, s.created_at, s.updated_at, s.created_by,
	(SELECT COUNT(*) FROM cfg_variables v WHERE v.set_id = s.id) AS variable_qty`

func scanCfgSet(sc cfgScanner) (DBCfgSet, error) {
	var i DBCfgSet
	err := sc.Scan(&i.ID, &i.Description, &i.IsActive, &i.CreatedAt, &i.UpdatedAt, &i.CreatedBy, &i.VariableQty)
	return i, err
}

func (q *Queries) CreateCfgSet(ctx context.Context, description string, createdBy pgtype.UUID) (DBCfgSet, error) {
	const sql = `WITH ins AS (
		INSERT INTO cfg_sets (description, created_by) VALUES ($1,$2) RETURNING *
	) SELECT ins.id, ins.description, ins.is_active, ins.created_at, ins.updated_at, ins.created_by, 0
	FROM ins`
	return scanCfgSet(q.db.QueryRow(ctx, sql, description, createdBy))
}

func (q *Queries) UpdateCfgSet(ctx context.Context, id int64, description string, isActive bool) (DBCfgSet, error) {
	const sql = `WITH upd AS (
		UPDATE cfg_sets SET description=$2, is_active=$3, updated_at=NOW() WHERE id=$1 RETURNING *
	) SELECT upd.id, upd.description, upd.is_active, upd.created_at, upd.updated_at, upd.created_by,
		(SELECT COUNT(*) FROM cfg_variables v WHERE v.set_id = upd.id) FROM upd`
	return scanCfgSet(q.db.QueryRow(ctx, sql, id, description, isActive))
}

func (q *Queries) GetCfgSet(ctx context.Context, id int64) (DBCfgSet, error) {
	return scanCfgSet(q.db.QueryRow(ctx, `SELECT `+cfgSetCols+` FROM cfg_sets s WHERE s.id=$1`, id))
}

func (q *Queries) ListCfgSets(ctx context.Context, onlyActive bool) ([]DBCfgSet, error) {
	const sql = `SELECT ` + cfgSetCols + ` FROM cfg_sets s
		WHERE ($1::BOOLEAN = FALSE OR s.is_active = TRUE) ORDER BY s.description`
	rows, err := q.db.Query(ctx, sql, onlyActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBCfgSet
	for rows.Next() {
		s, err := scanCfgSet(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (q *Queries) DeactivateCfgSet(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `UPDATE cfg_sets SET is_active=FALSE, updated_at=NOW() WHERE id=$1`, id)
	return err
}

// ─── cfg_variables ────────────────────────────────────────────────────────────

type DBCfgVariable struct {
	ID                 int64
	SetID              int64
	Code               string
	Description        string
	MaskComposition    string
	IsActive           bool
	IsSpecial          bool
	IncludeDescription bool
	SpecialData        pgtype.Text
	Marketing          bool
	CreatedAt          pgtype.Timestamptz
	UpdatedAt          pgtype.Timestamptz
	CreatedBy          pgtype.UUID
}

const cfgVarCols = `id, set_id, code, description, mask_composition, is_active, is_special,
	include_description, special_data, marketing, created_at, updated_at, created_by`

func scanCfgVariable(sc cfgScanner) (DBCfgVariable, error) {
	var i DBCfgVariable
	err := sc.Scan(&i.ID, &i.SetID, &i.Code, &i.Description, &i.MaskComposition, &i.IsActive,
		&i.IsSpecial, &i.IncludeDescription, &i.SpecialData, &i.Marketing,
		&i.CreatedAt, &i.UpdatedAt, &i.CreatedBy)
	return i, err
}

type CreateCfgVariableParams struct {
	SetID              int64
	Code               string
	Description        string
	MaskComposition    string
	IsSpecial          bool
	IncludeDescription bool
	SpecialData        pgtype.Text
	Marketing          bool
	CreatedBy          pgtype.UUID
}

func (q *Queries) CreateCfgVariable(ctx context.Context, a CreateCfgVariableParams) (DBCfgVariable, error) {
	const sql = `INSERT INTO cfg_variables
		(set_id, code, description, mask_composition, is_special, include_description, special_data, marketing, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING ` + cfgVarCols
	return scanCfgVariable(q.db.QueryRow(ctx, sql, a.SetID, a.Code, a.Description, a.MaskComposition,
		a.IsSpecial, a.IncludeDescription, a.SpecialData, a.Marketing, a.CreatedBy))
}

type UpdateCfgVariableParams struct {
	ID                 int64
	Code               string
	Description        string
	MaskComposition    string
	IsActive           bool
	IsSpecial          bool
	IncludeDescription bool
	SpecialData        pgtype.Text
	Marketing          bool
}

func (q *Queries) UpdateCfgVariable(ctx context.Context, a UpdateCfgVariableParams) (DBCfgVariable, error) {
	const sql = `UPDATE cfg_variables SET code=$2, description=$3, mask_composition=$4, is_active=$5,
		is_special=$6, include_description=$7, special_data=$8, marketing=$9, updated_at=NOW()
		WHERE id=$1 RETURNING ` + cfgVarCols
	return scanCfgVariable(q.db.QueryRow(ctx, sql, a.ID, a.Code, a.Description, a.MaskComposition, a.IsActive,
		a.IsSpecial, a.IncludeDescription, a.SpecialData, a.Marketing))
}

func (q *Queries) GetCfgVariable(ctx context.Context, id int64) (DBCfgVariable, error) {
	return scanCfgVariable(q.db.QueryRow(ctx, `SELECT `+cfgVarCols+` FROM cfg_variables WHERE id=$1`, id))
}

func (q *Queries) ListCfgVariablesBySet(ctx context.Context, setID int64, onlyActive bool) ([]DBCfgVariable, error) {
	const sql = `SELECT ` + cfgVarCols + ` FROM cfg_variables
		WHERE set_id=$1 AND ($2::BOOLEAN = FALSE OR is_active = TRUE) ORDER BY code`
	rows, err := q.db.Query(ctx, sql, setID, onlyActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBCfgVariable
	for rows.Next() {
		v, err := scanCfgVariable(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (q *Queries) DeactivateCfgVariable(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `UPDATE cfg_variables SET is_active=FALSE, updated_at=NOW() WHERE id=$1`, id)
	return err
}

// ─── cfg_variable_languages ───────────────────────────────────────────────────

type DBCfgVariableLanguage struct {
	ID          int64
	VariableID  int64
	Language    string
	Country     pgtype.Text
	Translation string
}

func (q *Queries) UpsertCfgVariableLanguage(ctx context.Context, variableID int64, language string, country pgtype.Text, translation string) (DBCfgVariableLanguage, error) {
	const sql = `INSERT INTO cfg_variable_languages (variable_id, language, country, translation)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (variable_id, language) DO UPDATE SET country=EXCLUDED.country, translation=EXCLUDED.translation
		RETURNING id, variable_id, language, country, translation`
	var i DBCfgVariableLanguage
	err := q.db.QueryRow(ctx, sql, variableID, language, country, translation).
		Scan(&i.ID, &i.VariableID, &i.Language, &i.Country, &i.Translation)
	return i, err
}

func (q *Queries) ListCfgVariableLanguages(ctx context.Context, variableID int64) ([]DBCfgVariableLanguage, error) {
	rows, err := q.db.Query(ctx, `SELECT id, variable_id, language, country, translation
		FROM cfg_variable_languages WHERE variable_id=$1 ORDER BY language`, variableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBCfgVariableLanguage
	for rows.Next() {
		var i DBCfgVariableLanguage
		if err := rows.Scan(&i.ID, &i.VariableID, &i.Language, &i.Country, &i.Translation); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (q *Queries) DeleteCfgVariableLanguage(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `DELETE FROM cfg_variable_languages WHERE id=$1`, id)
	return err
}

// ─── cfg_characteristics ──────────────────────────────────────────────────────

type DBCfgCharacteristic struct {
	ID                int64
	Code              string
	Description       string
	CharType          string
	IsActive          bool
	SetID             pgtype.Int8
	DefaultVariableID pgtype.Int8
	Mask              pgtype.Text
	IsSpecial         bool
	AffectsPrice      bool
	ControlsGoals     bool
	ReceivingType     string
	FieldSource       pgtype.Text
	Formula           pgtype.Text
	IsRequired        bool
	NumMin            pgtype.Numeric
	NumMax            pgtype.Numeric
	NumMultiple       pgtype.Numeric
	OptionTrue        pgtype.Text
	OptionFalse       pgtype.Text
	CreatedAt         pgtype.Timestamptz
	UpdatedAt         pgtype.Timestamptz
	CreatedBy         pgtype.UUID
	// denormalized
	SetDescription     pgtype.Text
	DefaultVariableStr pgtype.Text
}

const cfgCharCols = `c.id, c.code, c.description, c.char_type, c.is_active, c.set_id, c.default_variable_id,
	c.mask, c.is_special, c.affects_price, c.controls_goals, c.receiving_type, c.field_source, c.formula,
	c.is_required, c.num_min, c.num_max, c.num_multiple, c.option_true, c.option_false,
	c.created_at, c.updated_at, c.created_by,
	s.description AS set_description, dv.code AS default_variable_code`

const cfgCharFrom = `FROM cfg_characteristics c
	LEFT JOIN cfg_sets s ON s.id = c.set_id
	LEFT JOIN cfg_variables dv ON dv.id = c.default_variable_id`

// cfgCharReturn selects an inserted/updated characteristic straight from the
// data-modifying CTE (`alias`), joining the lookup tables. Selecting from the
// base table would miss the CTE's own write (Postgres CTE snapshot semantics).
func cfgCharReturn(alias string) string {
	return `SELECT ` + alias + `.id, ` + alias + `.code, ` + alias + `.description, ` + alias + `.char_type, ` +
		alias + `.is_active, ` + alias + `.set_id, ` + alias + `.default_variable_id, ` + alias + `.mask, ` +
		alias + `.is_special, ` + alias + `.affects_price, ` + alias + `.controls_goals, ` + alias + `.receiving_type, ` +
		alias + `.field_source, ` + alias + `.formula, ` + alias + `.is_required, ` + alias + `.num_min, ` +
		alias + `.num_max, ` + alias + `.num_multiple, ` + alias + `.option_true, ` + alias + `.option_false, ` +
		alias + `.created_at, ` + alias + `.updated_at, ` + alias + `.created_by, s.description, dv.code
	FROM ` + alias + `
	LEFT JOIN cfg_sets s ON s.id = ` + alias + `.set_id
	LEFT JOIN cfg_variables dv ON dv.id = ` + alias + `.default_variable_id`
}

func scanCfgCharacteristic(sc cfgScanner) (DBCfgCharacteristic, error) {
	var i DBCfgCharacteristic
	err := sc.Scan(&i.ID, &i.Code, &i.Description, &i.CharType, &i.IsActive, &i.SetID, &i.DefaultVariableID,
		&i.Mask, &i.IsSpecial, &i.AffectsPrice, &i.ControlsGoals, &i.ReceivingType, &i.FieldSource, &i.Formula,
		&i.IsRequired, &i.NumMin, &i.NumMax, &i.NumMultiple, &i.OptionTrue, &i.OptionFalse,
		&i.CreatedAt, &i.UpdatedAt, &i.CreatedBy, &i.SetDescription, &i.DefaultVariableStr)
	return i, err
}

type CfgCharacteristicParams struct {
	ID                int64
	Code              string
	Description       string
	CharType          string
	IsActive          bool
	SetID             pgtype.Int8
	DefaultVariableID pgtype.Int8
	Mask              pgtype.Text
	IsSpecial         bool
	AffectsPrice      bool
	ControlsGoals     bool
	ReceivingType     string
	FieldSource       pgtype.Text
	Formula           pgtype.Text
	IsRequired        bool
	NumMin            pgtype.Numeric
	NumMax            pgtype.Numeric
	NumMultiple       pgtype.Numeric
	OptionTrue        pgtype.Text
	OptionFalse       pgtype.Text
	CreatedBy         pgtype.UUID
}

func (q *Queries) CreateCfgCharacteristic(ctx context.Context, a CfgCharacteristicParams) (DBCfgCharacteristic, error) {
	sql := `WITH ins AS (
		INSERT INTO cfg_characteristics
		(code, description, char_type, set_id, default_variable_id, mask, is_special, affects_price,
		 controls_goals, receiving_type, field_source, formula, is_required, num_min, num_max, num_multiple,
		 option_true, option_false, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		RETURNING *
	) ` + cfgCharReturn("ins")
	return scanCfgCharacteristic(q.db.QueryRow(ctx, sql, a.Code, a.Description, a.CharType, a.SetID,
		a.DefaultVariableID, a.Mask, a.IsSpecial, a.AffectsPrice, a.ControlsGoals, a.ReceivingType,
		a.FieldSource, a.Formula, a.IsRequired, a.NumMin, a.NumMax, a.NumMultiple, a.OptionTrue,
		a.OptionFalse, a.CreatedBy))
}

func (q *Queries) UpdateCfgCharacteristic(ctx context.Context, a CfgCharacteristicParams) (DBCfgCharacteristic, error) {
	sql := `WITH upd AS (
		UPDATE cfg_characteristics SET code=$2, description=$3, char_type=$4, is_active=$5, set_id=$6,
		 default_variable_id=$7, mask=$8, is_special=$9, affects_price=$10, controls_goals=$11,
		 receiving_type=$12, field_source=$13, formula=$14, is_required=$15, num_min=$16, num_max=$17,
		 num_multiple=$18, option_true=$19, option_false=$20, updated_at=NOW()
		WHERE id=$1 RETURNING *
	) ` + cfgCharReturn("upd")
	return scanCfgCharacteristic(q.db.QueryRow(ctx, sql, a.ID, a.Code, a.Description, a.CharType, a.IsActive,
		a.SetID, a.DefaultVariableID, a.Mask, a.IsSpecial, a.AffectsPrice, a.ControlsGoals, a.ReceivingType,
		a.FieldSource, a.Formula, a.IsRequired, a.NumMin, a.NumMax, a.NumMultiple, a.OptionTrue, a.OptionFalse))
}

func (q *Queries) GetCfgCharacteristic(ctx context.Context, id int64) (DBCfgCharacteristic, error) {
	return scanCfgCharacteristic(q.db.QueryRow(ctx, `SELECT `+cfgCharCols+` `+cfgCharFrom+` WHERE c.id=$1`, id))
}

func (q *Queries) ListCfgCharacteristics(ctx context.Context, onlyActive bool) ([]DBCfgCharacteristic, error) {
	const sql = `SELECT ` + cfgCharCols + ` ` + cfgCharFrom + `
		WHERE ($1::BOOLEAN = FALSE OR c.is_active = TRUE) ORDER BY c.code`
	rows, err := q.db.Query(ctx, sql, onlyActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBCfgCharacteristic
	for rows.Next() {
		c, err := scanCfgCharacteristic(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (q *Queries) DeactivateCfgCharacteristic(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `UPDATE cfg_characteristics SET is_active=FALSE, updated_at=NOW() WHERE id=$1`, id)
	return err
}

// ─── cfg_characteristic_languages ─────────────────────────────────────────────

type DBCfgCharacteristicLanguage struct {
	ID               int64
	CharacteristicID int64
	Language         string
	Description      string
	Mask             pgtype.Text
}

func (q *Queries) UpsertCfgCharacteristicLanguage(ctx context.Context, charID int64, language, description string, mask pgtype.Text) (DBCfgCharacteristicLanguage, error) {
	const sql = `INSERT INTO cfg_characteristic_languages (characteristic_id, language, description, mask)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (characteristic_id, language) DO UPDATE SET description=EXCLUDED.description, mask=EXCLUDED.mask
		RETURNING id, characteristic_id, language, description, mask`
	var i DBCfgCharacteristicLanguage
	err := q.db.QueryRow(ctx, sql, charID, language, description, mask).
		Scan(&i.ID, &i.CharacteristicID, &i.Language, &i.Description, &i.Mask)
	return i, err
}

func (q *Queries) ListCfgCharacteristicLanguages(ctx context.Context, charID int64) ([]DBCfgCharacteristicLanguage, error) {
	rows, err := q.db.Query(ctx, `SELECT id, characteristic_id, language, description, mask
		FROM cfg_characteristic_languages WHERE characteristic_id=$1 ORDER BY language`, charID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBCfgCharacteristicLanguage
	for rows.Next() {
		var i DBCfgCharacteristicLanguage
		if err := rows.Scan(&i.ID, &i.CharacteristicID, &i.Language, &i.Description, &i.Mask); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (q *Queries) DeleteCfgCharacteristicLanguage(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `DELETE FROM cfg_characteristic_languages WHERE id=$1`, id)
	return err
}

// ─── cfg_item_characteristics ─────────────────────────────────────────────────

type DBCfgItemCharacteristic struct {
	ID                int64
	ItemCode          int64
	CharacteristicID  int64
	Sequence          int32
	DefaultVariableID pgtype.Int8
	ParentID          pgtype.Int8
	IsSpecial         bool
	IsDrawing         bool
	IsLoad            bool
	Formula           pgtype.Text
	CreatedAt         pgtype.Timestamptz
	UpdatedAt         pgtype.Timestamptz
	// denormalized
	CharCode string
	CharName string
	CharType string
	CharMask pgtype.Text
}

const cfgItemCharCols = `ic.id, ic.item_code, ic.characteristic_id, ic.sequence, ic.default_variable_id,
	ic.parent_id, ic.is_special, ic.is_drawing, ic.is_load, ic.formula, ic.created_at, ic.updated_at,
	c.code, c.description, c.char_type, c.mask`

// cfgItemCharReturn selects an inserted/updated item-characteristic straight
// from the data-modifying CTE (`alias`), joining the characteristic.
func cfgItemCharReturn(alias string) string {
	return `SELECT ` + alias + `.id, ` + alias + `.item_code, ` + alias + `.characteristic_id, ` + alias + `.sequence, ` +
		alias + `.default_variable_id, ` + alias + `.parent_id, ` + alias + `.is_special, ` + alias + `.is_drawing, ` +
		alias + `.is_load, ` + alias + `.formula, ` + alias + `.created_at, ` + alias + `.updated_at,
		c.code, c.description, c.char_type, c.mask
	FROM ` + alias + ` JOIN cfg_characteristics c ON c.id = ` + alias + `.characteristic_id`
}

func scanCfgItemCharacteristic(sc cfgScanner) (DBCfgItemCharacteristic, error) {
	var i DBCfgItemCharacteristic
	err := sc.Scan(&i.ID, &i.ItemCode, &i.CharacteristicID, &i.Sequence, &i.DefaultVariableID, &i.ParentID,
		&i.IsSpecial, &i.IsDrawing, &i.IsLoad, &i.Formula, &i.CreatedAt, &i.UpdatedAt,
		&i.CharCode, &i.CharName, &i.CharType, &i.CharMask)
	return i, err
}

type CfgItemCharacteristicParams struct {
	ID                int64
	ItemCode          int64
	CharacteristicID  int64
	Sequence          int32
	DefaultVariableID pgtype.Int8
	ParentID          pgtype.Int8
	IsSpecial         bool
	IsDrawing         bool
	IsLoad            bool
	Formula           pgtype.Text
}

func (q *Queries) AddCfgItemCharacteristic(ctx context.Context, a CfgItemCharacteristicParams) (DBCfgItemCharacteristic, error) {
	sql := `WITH ins AS (
		INSERT INTO cfg_item_characteristics
		(item_code, characteristic_id, sequence, default_variable_id, parent_id, is_special, is_drawing, is_load, formula)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING *
	) ` + cfgItemCharReturn("ins")
	return scanCfgItemCharacteristic(q.db.QueryRow(ctx, sql, a.ItemCode, a.CharacteristicID, a.Sequence,
		a.DefaultVariableID, a.ParentID, a.IsSpecial, a.IsDrawing, a.IsLoad, a.Formula))
}

func (q *Queries) UpdateCfgItemCharacteristic(ctx context.Context, a CfgItemCharacteristicParams) (DBCfgItemCharacteristic, error) {
	sql := `WITH upd AS (
		UPDATE cfg_item_characteristics SET sequence=$2, default_variable_id=$3, parent_id=$4,
		 is_special=$5, is_drawing=$6, is_load=$7, formula=$8, updated_at=NOW()
		WHERE id=$1 RETURNING *
	) ` + cfgItemCharReturn("upd")
	return scanCfgItemCharacteristic(q.db.QueryRow(ctx, sql, a.ID, a.Sequence, a.DefaultVariableID, a.ParentID,
		a.IsSpecial, a.IsDrawing, a.IsLoad, a.Formula))
}

func (q *Queries) GetCfgItemCharacteristic(ctx context.Context, id int64) (DBCfgItemCharacteristic, error) {
	const sql = `SELECT ` + cfgItemCharCols + ` FROM cfg_item_characteristics ic
		JOIN cfg_characteristics c ON c.id = ic.characteristic_id WHERE ic.id=$1`
	return scanCfgItemCharacteristic(q.db.QueryRow(ctx, sql, id))
}

func (q *Queries) ListCfgItemCharacteristics(ctx context.Context, itemCode int64) ([]DBCfgItemCharacteristic, error) {
	const sql = `SELECT ` + cfgItemCharCols + ` FROM cfg_item_characteristics ic
		JOIN cfg_characteristics c ON c.id = ic.characteristic_id
		WHERE ic.item_code=$1 ORDER BY ic.sequence`
	rows, err := q.db.Query(ctx, sql, itemCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBCfgItemCharacteristic
	for rows.Next() {
		ic, err := scanCfgItemCharacteristic(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ic)
	}
	return out, rows.Err()
}

func (q *Queries) RemoveCfgItemCharacteristic(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `DELETE FROM cfg_item_characteristics WHERE id=$1`, id)
	return err
}

// Default answers (ESCOLHA_MULT): replace the whole set atomically.
func (q *Queries) ReplaceCfgItemCharDefaultAnswers(ctx context.Context, itemCharID int64, variableIDs []int64) error {
	if _, err := q.db.Exec(ctx, `DELETE FROM cfg_item_char_default_answers WHERE item_characteristic_id=$1`, itemCharID); err != nil {
		return err
	}
	for _, v := range variableIDs {
		if _, err := q.db.Exec(ctx, `INSERT INTO cfg_item_char_default_answers (item_characteristic_id, variable_id)
			VALUES ($1,$2) ON CONFLICT DO NOTHING`, itemCharID, v); err != nil {
			return err
		}
	}
	return nil
}

func (q *Queries) ListCfgItemCharDefaultAnswers(ctx context.Context, itemCharID int64) ([]int64, error) {
	rows, err := q.db.Query(ctx, `SELECT variable_id FROM cfg_item_char_default_answers
		WHERE item_characteristic_id=$1 ORDER BY variable_id`, itemCharID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ItemHasGeneratedMask reports whether the item already has a generated mask —
// used to guard edits/deletes of its characteristics.
func (q *Queries) ItemHasGeneratedMask(ctx context.Context, itemCode int64) (bool, error) {
	var exists bool
	err := q.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM item_masks WHERE item_code=$1)`, itemCode).Scan(&exists)
	return exists, err
}

// ItemHasStructureFormula reports whether the item has a structure line with a
// loss formula — the second guard condition from the spec ("ou não possuir uma
// fórmula no cadastro de estruturas").
func (q *Queries) ItemHasStructureFormula(ctx context.Context, itemCode int64) (bool, error) {
	var exists bool
	err := q.db.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM item_structures WHERE parent_code=$1 AND loss_formula IS NOT NULL AND loss_formula <> '')`,
		itemCode).Scan(&exists)
	return exists, err
}

// ListItemsByCharacteristic returns the item codes that use a characteristic
// (Botão Itens Vinculados).
func (q *Queries) ListItemsByCharacteristic(ctx context.Context, characteristicID int64) ([]int64, error) {
	rows, err := q.db.Query(ctx, `SELECT DISTINCT item_code FROM cfg_item_characteristics
		WHERE characteristic_id=$1 ORDER BY item_code`, characteristicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ─── mask bridge (item_masks + cfg_item_mask_answers) ─────────────────────────

// PersistCfgItemMask writes the generated mask to item_masks (consumed by
// structure/sales/MRP) and returns the mask id for the rich answers.
func (q *Queries) PersistCfgItemMask(ctx context.Context, itemCode int64, mask, hash string, createdBy pgtype.UUID) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, `INSERT INTO item_masks (item_code, mask, mask_hash, created_by, created_at)
		VALUES ($1,$2,$3,$4,NOW()) RETURNING id`, itemCode, mask, hash, createdBy).Scan(&id)
	return id, err
}

func (q *Queries) InsertCfgItemMaskAnswer(ctx context.Context, maskID, charID int64, variableID pgtype.Int8, value string, position int32) error {
	_, err := q.db.Exec(ctx, `INSERT INTO cfg_item_mask_answers
		(mask_id, characteristic_id, variable_id, answer_value, position) VALUES ($1,$2,$3,$4,$5)`,
		maskID, charID, variableID, value, position)
	return err
}

// ─── cfg_characteristic_receiving_items (Botão Itens do Tipo Recebimento) ─────

type DBCfgCharReceivingItem struct {
	ID                 int64
	CharacteristicID   int64
	VariableID         pgtype.Int8
	ReceivingType      string
	ItemCode           pgtype.Int8
	ClassificationCode pgtype.Int8
	// denormalized
	VariableCode pgtype.Text
}

func (q *Queries) AddCfgCharReceivingItem(ctx context.Context, charID int64, variableID pgtype.Int8, receivingType string, itemCode, classificationCode pgtype.Int8) (DBCfgCharReceivingItem, error) {
	const sql = `WITH ins AS (
		INSERT INTO cfg_characteristic_receiving_items
		(characteristic_id, variable_id, receiving_type, item_code, classification_code)
		VALUES ($1,$2,$3,$4,$5) RETURNING *
	) SELECT ins.id, ins.characteristic_id, ins.variable_id, ins.receiving_type, ins.item_code,
		ins.classification_code, v.code
	FROM ins LEFT JOIN cfg_variables v ON v.id = ins.variable_id`
	var i DBCfgCharReceivingItem
	err := q.db.QueryRow(ctx, sql, charID, variableID, receivingType, itemCode, classificationCode).
		Scan(&i.ID, &i.CharacteristicID, &i.VariableID, &i.ReceivingType, &i.ItemCode, &i.ClassificationCode, &i.VariableCode)
	return i, err
}

func (q *Queries) ListCfgCharReceivingItems(ctx context.Context, charID int64) ([]DBCfgCharReceivingItem, error) {
	const sql = `SELECT r.id, r.characteristic_id, r.variable_id, r.receiving_type, r.item_code,
		r.classification_code, v.code
	FROM cfg_characteristic_receiving_items r
	LEFT JOIN cfg_variables v ON v.id = r.variable_id
	WHERE r.characteristic_id=$1 ORDER BY r.id`
	rows, err := q.db.Query(ctx, sql, charID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBCfgCharReceivingItem
	for rows.Next() {
		var i DBCfgCharReceivingItem
		if err := rows.Scan(&i.ID, &i.CharacteristicID, &i.VariableID, &i.ReceivingType, &i.ItemCode, &i.ClassificationCode, &i.VariableCode); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (q *Queries) DeleteCfgCharReceivingItem(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `DELETE FROM cfg_characteristic_receiving_items WHERE id=$1`, id)
	return err
}
