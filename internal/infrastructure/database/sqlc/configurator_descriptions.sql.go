package sqlc

// Hand-written data access for the configurator description types and configured
// item descriptions (Fase 4).

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type cfgDescScanner interface{ Scan(...any) error }

// ─── cfg_description_types ────────────────────────────────────────────────────

type DBCfgDescriptionType struct {
	ID          int64
	Code        string
	Description string
	Kind        string
	IsActive    bool
	CreatedAt   pgtype.Timestamptz
	CreatedBy   pgtype.UUID
}

const cfgDescTypeCols = `id, code, description, kind, is_active, created_at, created_by`

func scanCfgDescType(s cfgDescScanner) (DBCfgDescriptionType, error) {
	var i DBCfgDescriptionType
	err := s.Scan(&i.ID, &i.Code, &i.Description, &i.Kind, &i.IsActive, &i.CreatedAt, &i.CreatedBy)
	return i, err
}

func (q *Queries) CreateCfgDescriptionType(ctx context.Context, code, description, kind string, createdBy pgtype.UUID) (DBCfgDescriptionType, error) {
	const sql = `INSERT INTO cfg_description_types (code, description, kind, created_by)
		VALUES ($1,$2,$3,$4) RETURNING ` + cfgDescTypeCols
	return scanCfgDescType(q.db.QueryRow(ctx, sql, code, description, kind, createdBy))
}

func (q *Queries) UpdateCfgDescriptionType(ctx context.Context, id int64, code, description, kind string, isActive bool) (DBCfgDescriptionType, error) {
	const sql = `UPDATE cfg_description_types SET code=$2, description=$3, kind=$4, is_active=$5, updated_at=NOW()
		WHERE id=$1 RETURNING ` + cfgDescTypeCols
	return scanCfgDescType(q.db.QueryRow(ctx, sql, id, code, description, kind, isActive))
}

func (q *Queries) GetCfgDescriptionType(ctx context.Context, id int64) (DBCfgDescriptionType, error) {
	return scanCfgDescType(q.db.QueryRow(ctx, `SELECT `+cfgDescTypeCols+` FROM cfg_description_types WHERE id=$1`, id))
}

func (q *Queries) ListCfgDescriptionTypes(ctx context.Context, onlyActive bool) ([]DBCfgDescriptionType, error) {
	const sql = `SELECT ` + cfgDescTypeCols + ` FROM cfg_description_types
		WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE) ORDER BY code`
	rows, err := q.db.Query(ctx, sql, onlyActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBCfgDescriptionType
	for rows.Next() {
		t, err := scanCfgDescType(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (q *Queries) DeactivateCfgDescriptionType(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `UPDATE cfg_description_types SET is_active=FALSE, updated_at=NOW() WHERE id=$1`, id)
	return err
}

// ─── cfg_item_descriptions (header) ───────────────────────────────────────────

type DBCfgItemDescription struct {
	ID                int64
	ItemCode          int64
	DescriptionTypeID int64
	CreatedAt         pgtype.Timestamptz
	CreatedBy         pgtype.UUID
}

func (q *Queries) CreateCfgItemDescription(ctx context.Context, itemCode, typeID int64, createdBy pgtype.UUID) (DBCfgItemDescription, error) {
	const sql = `INSERT INTO cfg_item_descriptions (item_code, description_type_id, created_by)
		VALUES ($1,$2,$3) RETURNING id, item_code, description_type_id, created_at, created_by`
	var i DBCfgItemDescription
	err := q.db.QueryRow(ctx, sql, itemCode, typeID, createdBy).
		Scan(&i.ID, &i.ItemCode, &i.DescriptionTypeID, &i.CreatedAt, &i.CreatedBy)
	return i, err
}

func (q *Queries) GetCfgItemDescription(ctx context.Context, id int64) (DBCfgItemDescription, error) {
	var i DBCfgItemDescription
	err := q.db.QueryRow(ctx, `SELECT id, item_code, description_type_id, created_at, created_by
		FROM cfg_item_descriptions WHERE id=$1`, id).
		Scan(&i.ID, &i.ItemCode, &i.DescriptionTypeID, &i.CreatedAt, &i.CreatedBy)
	return i, err
}

// GetCfgItemDescriptionByItemType finds the header for an (item, type) pair.
func (q *Queries) GetCfgItemDescriptionByItemType(ctx context.Context, itemCode, typeID int64) (DBCfgItemDescription, error) {
	var i DBCfgItemDescription
	err := q.db.QueryRow(ctx, `SELECT id, item_code, description_type_id, created_at, created_by
		FROM cfg_item_descriptions WHERE item_code=$1 AND description_type_id=$2`, itemCode, typeID).
		Scan(&i.ID, &i.ItemCode, &i.DescriptionTypeID, &i.CreatedAt, &i.CreatedBy)
	return i, err
}

func (q *Queries) ListCfgItemDescriptionsByItem(ctx context.Context, itemCode int64) ([]DBCfgItemDescription, error) {
	rows, err := q.db.Query(ctx, `SELECT id, item_code, description_type_id, created_at, created_by
		FROM cfg_item_descriptions WHERE item_code=$1 ORDER BY description_type_id`, itemCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBCfgItemDescription
	for rows.Next() {
		var i DBCfgItemDescription
		if err := rows.Scan(&i.ID, &i.ItemCode, &i.DescriptionTypeID, &i.CreatedAt, &i.CreatedBy); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (q *Queries) DeleteCfgItemDescription(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `DELETE FROM cfg_item_descriptions WHERE id=$1`, id)
	return err
}

// ─── cfg_item_description_lines (grade) ───────────────────────────────────────

type DBCfgItemDescriptionLine struct {
	ID                   int64
	ItemDescriptionID    int64
	ItemCharacteristicID int64
	OrderIndex           int32
	ShowCharacteristic   bool
	ShowMask             bool
	DescType             string
	Text                 string
	LineBreak            bool
	// denormalized for reads / render
	Sequence         int32
	CharacteristicID int64
	CharCode         string
	CharDescription  string
	CharMask         pgtype.Text
}

const cfgDescLineCols = `l.id, l.item_description_id, l.item_characteristic_id, l.order_index,
	l.show_characteristic, l.show_mask, l.desc_type, l.text, l.line_break,
	ic.sequence, ic.characteristic_id, c.code, c.description, c.mask`

const cfgDescLineFrom = `FROM cfg_item_description_lines l
	JOIN cfg_item_characteristics ic ON ic.id = l.item_characteristic_id
	JOIN cfg_characteristics c ON c.id = ic.characteristic_id`

func scanCfgDescLine(s cfgDescScanner) (DBCfgItemDescriptionLine, error) {
	var i DBCfgItemDescriptionLine
	err := s.Scan(&i.ID, &i.ItemDescriptionID, &i.ItemCharacteristicID, &i.OrderIndex,
		&i.ShowCharacteristic, &i.ShowMask, &i.DescType, &i.Text, &i.LineBreak,
		&i.Sequence, &i.CharacteristicID, &i.CharCode, &i.CharDescription, &i.CharMask)
	return i, err
}

func (q *Queries) InsertCfgItemDescriptionLine(ctx context.Context, headerID, itemCharID int64, orderIndex int32, showChar, showMask bool, descType, text string, lineBreak bool) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, `INSERT INTO cfg_item_description_lines
		(item_description_id, item_characteristic_id, order_index, show_characteristic, show_mask, desc_type, text, line_break)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (item_description_id, item_characteristic_id) DO NOTHING
		RETURNING id`, headerID, itemCharID, orderIndex, showChar, showMask, descType, text, lineBreak).Scan(&id)
	return id, err
}

func (q *Queries) UpdateCfgItemDescriptionLine(ctx context.Context, id int64, orderIndex int32, showChar, showMask bool, descType, text string, lineBreak bool) error {
	_, err := q.db.Exec(ctx, `UPDATE cfg_item_description_lines
		SET order_index=$2, show_characteristic=$3, show_mask=$4, desc_type=$5, text=$6, line_break=$7
		WHERE id=$1`, id, orderIndex, showChar, showMask, descType, text, lineBreak)
	return err
}

func (q *Queries) ListCfgItemDescriptionLines(ctx context.Context, headerID int64) ([]DBCfgItemDescriptionLine, error) {
	const sql = `SELECT ` + cfgDescLineCols + ` ` + cfgDescLineFrom + `
		WHERE l.item_description_id=$1 ORDER BY l.order_index, ic.sequence`
	rows, err := q.db.Query(ctx, sql, headerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBCfgItemDescriptionLine
	for rows.Next() {
		l, err := scanCfgDescLine(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (q *Queries) DeleteCfgItemDescriptionLine(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `DELETE FROM cfg_item_description_lines WHERE id=$1`, id)
	return err
}

func (q *Queries) DeleteCfgItemDescriptionLinesByHeader(ctx context.Context, headerID int64) error {
	_, err := q.db.Exec(ctx, `DELETE FROM cfg_item_description_lines WHERE item_description_id=$1`, headerID)
	return err
}
