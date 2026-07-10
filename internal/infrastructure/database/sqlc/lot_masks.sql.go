package sqlc

// Hand-written data access for the Lot/Serial Mask register.

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type lotScanner interface{ Scan(...any) error }

// ─── lot_masks ────────────────────────────────────────────────────────────────

type DBLotMask struct {
	ID                 int64
	Application        string
	CustomerCode       pgtype.Int8
	ItemCode           pgtype.Int8
	ClassificationType pgtype.Text
	ClassificationCode pgtype.Int8
	ZeroOnYearChange   bool
	IsActive           bool
	Description        pgtype.Text
	CreatedAt          pgtype.Timestamptz
	UpdatedAt          pgtype.Timestamptz
	CreatedBy          pgtype.UUID
}

const lotMaskCols = `id, application, customer_code, item_code, classification_type, classification_code,
	zero_on_year_change, is_active, description, created_at, updated_at, created_by`

func scanLotMask(s lotScanner) (DBLotMask, error) {
	var i DBLotMask
	err := s.Scan(&i.ID, &i.Application, &i.CustomerCode, &i.ItemCode, &i.ClassificationType,
		&i.ClassificationCode, &i.ZeroOnYearChange, &i.IsActive, &i.Description,
		&i.CreatedAt, &i.UpdatedAt, &i.CreatedBy)
	return i, err
}

type LotMaskParams struct {
	ID                 int64
	Application        string
	CustomerCode       pgtype.Int8
	ItemCode           pgtype.Int8
	ClassificationType pgtype.Text
	ClassificationCode pgtype.Int8
	ZeroOnYearChange   bool
	Description        pgtype.Text
	CreatedBy          pgtype.UUID
}

func (q *Queries) CreateLotMask(ctx context.Context, a LotMaskParams) (DBLotMask, error) {
	const sql = `INSERT INTO lot_masks
		(application, customer_code, item_code, classification_type, classification_code, zero_on_year_change, description, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING ` + lotMaskCols
	return scanLotMask(q.db.QueryRow(ctx, sql, a.Application, a.CustomerCode, a.ItemCode, a.ClassificationType,
		a.ClassificationCode, a.ZeroOnYearChange, a.Description, a.CreatedBy))
}

func (q *Queries) UpdateLotMask(ctx context.Context, a LotMaskParams) (DBLotMask, error) {
	const sql = `UPDATE lot_masks SET application=$2, customer_code=$3, item_code=$4, classification_type=$5,
		classification_code=$6, zero_on_year_change=$7, description=$8, updated_at=NOW()
		WHERE id=$1 RETURNING ` + lotMaskCols
	return scanLotMask(q.db.QueryRow(ctx, sql, a.ID, a.Application, a.CustomerCode, a.ItemCode,
		a.ClassificationType, a.ClassificationCode, a.ZeroOnYearChange, a.Description))
}

func (q *Queries) GetLotMask(ctx context.Context, id int64) (DBLotMask, error) {
	return scanLotMask(q.db.QueryRow(ctx, `SELECT `+lotMaskCols+` FROM lot_masks WHERE id=$1`, id))
}

func (q *Queries) ListLotMasks(ctx context.Context, onlyActive bool) ([]DBLotMask, error) {
	const sql = `SELECT ` + lotMaskCols + ` FROM lot_masks
		WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE) ORDER BY application, id`
	rows, err := q.db.Query(ctx, sql, onlyActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBLotMask
	for rows.Next() {
		m, err := scanLotMask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (q *Queries) DeactivateLotMask(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `UPDATE lot_masks SET is_active=FALSE, updated_at=NOW() WHERE id=$1`, id)
	return err
}

// ResolveLotMask picks the most specific active mask for a context: customer+item
// beats item beats customer beats classification beats application-only. Returns
// the mask id (0 if none).
func (q *Queries) ResolveLotMask(ctx context.Context, application string, customerCode, itemCode, classificationCode pgtype.Int8) (int64, error) {
	const sql = `SELECT id FROM lot_masks
		WHERE is_active = TRUE AND application = $1
		  AND (customer_code IS NULL OR customer_code = $2)
		  AND (item_code IS NULL OR item_code = $3)
		  AND (classification_code IS NULL OR classification_code = $4)
		ORDER BY (customer_code IS NOT NULL)::int * 8
		       + (item_code IS NOT NULL)::int * 4
		       + (classification_code IS NOT NULL)::int * 2 DESC
		LIMIT 1`
	var id int64
	err := q.db.QueryRow(ctx, sql, application, customerCode, itemCode, classificationCode).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// ─── lot_mask_parts ───────────────────────────────────────────────────────────

type DBLotMaskPart struct {
	ID               int64
	LotMaskID        int64
	Sequence         int32
	PartType         string
	Value            string
	Size             int32
	DateFormat       pgtype.Text
	ZeroOnYearChange bool
	CurrentValue     string
	LastYear         pgtype.Int4
}

const lotPartCols = `id, lot_mask_id, sequence, part_type, value, size, date_format,
	zero_on_year_change, current_value, last_year`

func scanLotPart(s lotScanner) (DBLotMaskPart, error) {
	var i DBLotMaskPart
	err := s.Scan(&i.ID, &i.LotMaskID, &i.Sequence, &i.PartType, &i.Value, &i.Size, &i.DateFormat,
		&i.ZeroOnYearChange, &i.CurrentValue, &i.LastYear)
	return i, err
}

type LotMaskPartParams struct {
	ID               int64
	LotMaskID        int64
	Sequence         int32
	PartType         string
	Value            string
	Size             int32
	DateFormat       pgtype.Text
	ZeroOnYearChange bool
}

func (q *Queries) AddLotMaskPart(ctx context.Context, a LotMaskPartParams) (DBLotMaskPart, error) {
	const sql = `INSERT INTO lot_mask_parts
		(lot_mask_id, sequence, part_type, value, size, date_format, zero_on_year_change)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING ` + lotPartCols
	return scanLotPart(q.db.QueryRow(ctx, sql, a.LotMaskID, a.Sequence, a.PartType, a.Value, a.Size,
		a.DateFormat, a.ZeroOnYearChange))
}

func (q *Queries) UpdateLotMaskPart(ctx context.Context, a LotMaskPartParams) (DBLotMaskPart, error) {
	const sql = `UPDATE lot_mask_parts SET sequence=$2, part_type=$3, value=$4, size=$5, date_format=$6,
		zero_on_year_change=$7 WHERE id=$1 RETURNING ` + lotPartCols
	return scanLotPart(q.db.QueryRow(ctx, sql, a.ID, a.Sequence, a.PartType, a.Value, a.Size,
		a.DateFormat, a.ZeroOnYearChange))
}

func (q *Queries) ListLotMaskParts(ctx context.Context, lotMaskID int64) ([]DBLotMaskPart, error) {
	rows, err := q.db.Query(ctx, `SELECT `+lotPartCols+` FROM lot_mask_parts
		WHERE lot_mask_id=$1 ORDER BY sequence`, lotMaskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBLotMaskPart
	for rows.Next() {
		p, err := scanLotPart(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (q *Queries) DeleteLotMaskPart(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `DELETE FROM lot_mask_parts WHERE id=$1`, id)
	return err
}

// UpdateLotMaskPartState persists the sequence state after a generation.
func (q *Queries) UpdateLotMaskPartState(ctx context.Context, partID int64, currentValue string, lastYear int32) error {
	_, err := q.db.Exec(ctx, `UPDATE lot_mask_parts SET current_value=$2, last_year=$3 WHERE id=$1`,
		partID, currentValue, lastYear)
	return err
}
