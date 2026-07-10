package sqlc

// Hand-written data access for the Drawing register (Cadastro de Desenhos).

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type drawScanner interface{ Scan(...any) error }

// ─── drawings ─────────────────────────────────────────────────────────────────

type DBDrawing struct {
	ID           int64
	Code         string
	Digit        string
	Format       string
	Model        pgtype.Text
	ItemCode     pgtype.Int8
	Description  pgtype.Text
	Uom          pgtype.Text
	Weight       pgtype.Numeric
	MaterialSpec pgtype.Text
	CreationDate pgtype.Date
	IsActive     bool
	CreatedAt    pgtype.Timestamptz
	UpdatedAt    pgtype.Timestamptz
	CreatedBy    pgtype.UUID
}

const drawingCols = `id, code, digit, format, model, item_code, description, uom, weight,
	material_spec, creation_date, is_active, created_at, updated_at, created_by`

func scanDrawing(s drawScanner) (DBDrawing, error) {
	var i DBDrawing
	err := s.Scan(&i.ID, &i.Code, &i.Digit, &i.Format, &i.Model, &i.ItemCode, &i.Description, &i.Uom,
		&i.Weight, &i.MaterialSpec, &i.CreationDate, &i.IsActive, &i.CreatedAt, &i.UpdatedAt, &i.CreatedBy)
	return i, err
}

type DrawingParams struct {
	ID           int64
	Code         string
	Digit        string
	Format       string
	Model        pgtype.Text
	ItemCode     pgtype.Int8
	Description  pgtype.Text
	Uom          pgtype.Text
	Weight       pgtype.Numeric
	MaterialSpec pgtype.Text
	CreationDate pgtype.Date
	CreatedBy    pgtype.UUID
}

func (q *Queries) CreateDrawing(ctx context.Context, a DrawingParams) (DBDrawing, error) {
	const sql = `INSERT INTO drawings
		(code, digit, format, model, item_code, description, uom, weight, material_spec, creation_date, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING ` + drawingCols
	return scanDrawing(q.db.QueryRow(ctx, sql, a.Code, a.Digit, a.Format, a.Model, a.ItemCode, a.Description,
		a.Uom, a.Weight, a.MaterialSpec, a.CreationDate, a.CreatedBy))
}

func (q *Queries) UpdateDrawing(ctx context.Context, a DrawingParams) (DBDrawing, error) {
	const sql = `UPDATE drawings SET code=$2, digit=$3, format=$4, model=$5, item_code=$6, description=$7,
		uom=$8, weight=$9, material_spec=$10, creation_date=$11, updated_at=NOW()
		WHERE id=$1 RETURNING ` + drawingCols
	return scanDrawing(q.db.QueryRow(ctx, sql, a.ID, a.Code, a.Digit, a.Format, a.Model, a.ItemCode,
		a.Description, a.Uom, a.Weight, a.MaterialSpec, a.CreationDate))
}

func (q *Queries) GetDrawing(ctx context.Context, id int64) (DBDrawing, error) {
	return scanDrawing(q.db.QueryRow(ctx, `SELECT `+drawingCols+` FROM drawings WHERE id=$1`, id))
}

func (q *Queries) ListDrawings(ctx context.Context, onlyActive bool, search string) ([]DBDrawing, error) {
	const sql = `SELECT ` + drawingCols + ` FROM drawings
		WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
		  AND ($2::text = '' OR code ILIKE '%'||$2||'%' OR description ILIKE '%'||$2||'%')
		ORDER BY code, digit`
	rows, err := q.db.Query(ctx, sql, onlyActive, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBDrawing
	for rows.Next() {
		d, err := scanDrawing(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (q *Queries) DeactivateDrawing(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `UPDATE drawings SET is_active=FALSE, updated_at=NOW() WHERE id=$1`, id)
	return err
}

// ─── drawing_revisions ────────────────────────────────────────────────────────

type DBDrawingRevision struct {
	ID           int64
	DrawingID    int64
	Revision     string
	StartDate    pgtype.Date
	EndDate      pgtype.Date
	MaterialSpec pgtype.Text
	Reason       pgtype.Text
	ApprovedBy   pgtype.Text
	ApprovalDate pgtype.Date
	IsCurrent    bool
	CreatedAt    pgtype.Timestamptz
}

const drawingRevCols = `id, drawing_id, revision, start_date, end_date, material_spec, reason,
	approved_by, approval_date, is_current, created_at`

func scanDrawingRev(s drawScanner) (DBDrawingRevision, error) {
	var i DBDrawingRevision
	err := s.Scan(&i.ID, &i.DrawingID, &i.Revision, &i.StartDate, &i.EndDate, &i.MaterialSpec, &i.Reason,
		&i.ApprovedBy, &i.ApprovalDate, &i.IsCurrent, &i.CreatedAt)
	return i, err
}

type DrawingRevisionParams struct {
	ID           int64
	DrawingID    int64
	Revision     string
	StartDate    pgtype.Date
	EndDate      pgtype.Date
	MaterialSpec pgtype.Text
	Reason       pgtype.Text
	ApprovedBy   pgtype.Text
	ApprovalDate pgtype.Date
	IsCurrent    bool
}

func (q *Queries) AddDrawingRevision(ctx context.Context, a DrawingRevisionParams) (DBDrawingRevision, error) {
	const sql = `INSERT INTO drawing_revisions
		(drawing_id, revision, start_date, end_date, material_spec, reason, approved_by, approval_date, is_current)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING ` + drawingRevCols
	return scanDrawingRev(q.db.QueryRow(ctx, sql, a.DrawingID, a.Revision, a.StartDate, a.EndDate,
		a.MaterialSpec, a.Reason, a.ApprovedBy, a.ApprovalDate, a.IsCurrent))
}

func (q *Queries) UpdateDrawingRevision(ctx context.Context, a DrawingRevisionParams) (DBDrawingRevision, error) {
	const sql = `UPDATE drawing_revisions SET revision=$2, start_date=$3, end_date=$4, material_spec=$5,
		reason=$6, approved_by=$7, approval_date=$8, is_current=$9 WHERE id=$1 RETURNING ` + drawingRevCols
	return scanDrawingRev(q.db.QueryRow(ctx, sql, a.ID, a.Revision, a.StartDate, a.EndDate, a.MaterialSpec,
		a.Reason, a.ApprovedBy, a.ApprovalDate, a.IsCurrent))
}

func (q *Queries) ListDrawingRevisions(ctx context.Context, drawingID int64) ([]DBDrawingRevision, error) {
	rows, err := q.db.Query(ctx, `SELECT `+drawingRevCols+` FROM drawing_revisions
		WHERE drawing_id=$1 ORDER BY created_at`, drawingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBDrawingRevision
	for rows.Next() {
		r, err := scanDrawingRev(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Queries) DeleteDrawingRevision(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `DELETE FROM drawing_revisions WHERE id=$1`, id)
	return err
}

// SetCurrentDrawingRevision flips is_current to the given revision (others off).
func (q *Queries) SetCurrentDrawingRevision(ctx context.Context, drawingID, revisionID int64) error {
	if _, err := q.db.Exec(ctx, `UPDATE drawing_revisions SET is_current=(id=$2) WHERE drawing_id=$1`, drawingID, revisionID); err != nil {
		return err
	}
	return nil
}

// ─── drawing_revision_distributions ───────────────────────────────────────────

type DBDrawingDistribution struct {
	ID            int64
	RevisionID    int64
	Recipient     string
	DistributedAt pgtype.Date
	Notes         pgtype.Text
}

func (q *Queries) AddDrawingDistribution(ctx context.Context, revisionID int64, recipient string, at pgtype.Date, notes pgtype.Text) (DBDrawingDistribution, error) {
	const sql = `INSERT INTO drawing_revision_distributions (revision_id, recipient, distributed_at, notes)
		VALUES ($1,$2,$3,$4) RETURNING id, revision_id, recipient, distributed_at, notes`
	var i DBDrawingDistribution
	err := q.db.QueryRow(ctx, sql, revisionID, recipient, at, notes).
		Scan(&i.ID, &i.RevisionID, &i.Recipient, &i.DistributedAt, &i.Notes)
	return i, err
}

func (q *Queries) ListDrawingDistributions(ctx context.Context, revisionID int64) ([]DBDrawingDistribution, error) {
	rows, err := q.db.Query(ctx, `SELECT id, revision_id, recipient, distributed_at, notes
		FROM drawing_revision_distributions WHERE revision_id=$1 ORDER BY id`, revisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBDrawingDistribution
	for rows.Next() {
		var i DBDrawingDistribution
		if err := rows.Scan(&i.ID, &i.RevisionID, &i.Recipient, &i.DistributedAt, &i.Notes); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (q *Queries) DeleteDrawingDistribution(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `DELETE FROM drawing_revision_distributions WHERE id=$1`, id)
	return err
}

// ─── drawing_characteristics ──────────────────────────────────────────────────

type DBDrawingCharacteristic struct {
	ID               int64
	DrawingID        int64
	CharacteristicID int64
	Operator         string
	VariableID       pgtype.Int8
}

func (q *Queries) AddDrawingCharacteristic(ctx context.Context, drawingID, charID int64, operator string, variableID pgtype.Int8) (DBDrawingCharacteristic, error) {
	const sql = `INSERT INTO drawing_characteristics (drawing_id, characteristic_id, operator, variable_id)
		VALUES ($1,$2,$3,$4) RETURNING id, drawing_id, characteristic_id, operator, variable_id`
	var i DBDrawingCharacteristic
	err := q.db.QueryRow(ctx, sql, drawingID, charID, operator, variableID).
		Scan(&i.ID, &i.DrawingID, &i.CharacteristicID, &i.Operator, &i.VariableID)
	return i, err
}

func (q *Queries) ListDrawingCharacteristics(ctx context.Context, drawingID int64) ([]DBDrawingCharacteristic, error) {
	rows, err := q.db.Query(ctx, `SELECT id, drawing_id, characteristic_id, operator, variable_id
		FROM drawing_characteristics WHERE drawing_id=$1 ORDER BY id`, drawingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBDrawingCharacteristic
	for rows.Next() {
		var i DBDrawingCharacteristic
		if err := rows.Scan(&i.ID, &i.DrawingID, &i.CharacteristicID, &i.Operator, &i.VariableID); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (q *Queries) DeleteDrawingCharacteristic(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, `DELETE FROM drawing_characteristics WHERE id=$1`, id)
	return err
}
