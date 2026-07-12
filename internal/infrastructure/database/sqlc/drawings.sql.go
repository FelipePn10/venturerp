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
	EnterpriseID int64
}

func (q *Queries) DrawingBelongsToEnterprise(ctx context.Context, drawingID, enterpriseID int64) (bool, error) {
	var exists bool
	err := q.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM drawings WHERE id=$1 AND enterprise_id=$2)`, drawingID, enterpriseID).Scan(&exists)
	return exists, err
}

func (q *Queries) DrawingRevisionBelongsToEnterprise(ctx context.Context, revisionID, enterpriseID int64) (bool, error) {
	var exists bool
	err := q.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM drawing_revisions revision JOIN drawings drawing ON drawing.id=revision.drawing_id WHERE revision.id=$1 AND drawing.enterprise_id=$2)`, revisionID, enterpriseID).Scan(&exists)
	return exists, err
}

func (q *Queries) DrawingDistributionBelongsToEnterprise(ctx context.Context, distributionID, enterpriseID int64) (bool, error) {
	var exists bool
	err := q.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM drawing_revision_distributions distribution JOIN drawing_revisions revision ON revision.id=distribution.revision_id JOIN drawings drawing ON drawing.id=revision.drawing_id WHERE distribution.id=$1 AND drawing.enterprise_id=$2)`, distributionID, enterpriseID).Scan(&exists)
	return exists, err
}

func (q *Queries) DrawingCharacteristicBelongsToEnterprise(ctx context.Context, linkID, enterpriseID int64) (bool, error) {
	var exists bool
	err := q.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM drawing_characteristics characteristic JOIN drawings drawing ON drawing.id=characteristic.drawing_id WHERE characteristic.id=$1 AND drawing.enterprise_id=$2)`, linkID, enterpriseID).Scan(&exists)
	return exists, err
}

func (q *Queries) CreateDrawingForEnterprise(ctx context.Context, a DrawingParams) (DBDrawing, error) {
	const sql = `INSERT INTO drawings
		(code,digit,format,model,item_code,description,uom,weight,material_spec,creation_date,created_by,enterprise_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING ` + drawingCols
	return scanDrawing(q.db.QueryRow(ctx, sql, a.Code, a.Digit, a.Format, a.Model, a.ItemCode, a.Description,
		a.Uom, a.Weight, a.MaterialSpec, a.CreationDate, a.CreatedBy, a.EnterpriseID))
}

func (q *Queries) UpdateDrawingForEnterprise(ctx context.Context, a DrawingParams) (DBDrawing, error) {
	const sql = `UPDATE drawings SET code=$3,digit=$4,format=$5,model=$6,item_code=$7,description=$8,
		uom=$9,weight=$10,material_spec=$11,creation_date=$12,updated_at=NOW()
		WHERE id=$1 AND enterprise_id=$2 RETURNING ` + drawingCols
	return scanDrawing(q.db.QueryRow(ctx, sql, a.ID, a.EnterpriseID, a.Code, a.Digit, a.Format, a.Model,
		a.ItemCode, a.Description, a.Uom, a.Weight, a.MaterialSpec, a.CreationDate))
}

func (q *Queries) GetDrawingForEnterprise(ctx context.Context, id, enterpriseID int64) (DBDrawing, error) {
	return scanDrawing(q.db.QueryRow(ctx, `SELECT `+drawingCols+` FROM drawings WHERE id=$1 AND enterprise_id=$2`, id, enterpriseID))
}

func (q *Queries) ListDrawingsForEnterprise(ctx context.Context, enterpriseID int64, onlyActive bool, search string) ([]DBDrawing, error) {
	rows, err := q.db.Query(ctx, `SELECT `+drawingCols+` FROM drawings WHERE enterprise_id=$1
		AND ($2::boolean=FALSE OR is_active) AND ($3='' OR code ILIKE '%'||$3||'%' OR description ILIKE '%'||$3||'%') ORDER BY code,digit`, enterpriseID, onlyActive, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DBDrawing
	for rows.Next() {
		drawing, err := scanDrawing(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, drawing)
	}
	return out, rows.Err()
}

func (q *Queries) DeactivateDrawingForEnterprise(ctx context.Context, id, enterpriseID int64) error {
	_, err := q.db.Exec(ctx, `UPDATE drawings SET is_active=FALSE,updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, id, enterpriseID)
	return err
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

func (q *Queries) AddDrawingRevisionForEnterprise(ctx context.Context, enterpriseID int64, a DrawingRevisionParams, updatedBy pgtype.UUID) (DBDrawingRevision, error) {
	const sql = `WITH scoped AS (
		SELECT drawing.*,LEFT(drawing.code,20)||drawing.digit||drawing.format AS prefix
		FROM drawings drawing WHERE drawing.id=$1 AND drawing.enterprise_id=$2 FOR UPDATE
	), previous AS (
		SELECT revision.revision,scoped.prefix||revision.revision AS composite
		FROM scoped JOIN drawing_revisions revision ON revision.drawing_id=scoped.id AND revision.is_current
		ORDER BY revision.created_at DESC,revision.id DESC LIMIT 1
	), clear_current AS (
		UPDATE drawing_revisions SET is_current=FALSE WHERE drawing_id=(SELECT id FROM scoped) AND $11
	), inserted AS (
		INSERT INTO drawing_revisions(drawing_id,revision,start_date,end_date,material_spec,reason,approved_by,approval_date,is_current)
		SELECT id,$3,$4,$5,$6,$7,$8,$9,$11 FROM scoped
		RETURNING id,drawing_id,revision,start_date,end_date,material_spec,reason,approved_by,approval_date,is_current,created_at
	), replicated AS (
		UPDATE item_engineering_drawings engineering
		SET drawing_code=(SELECT prefix FROM scoped)||$3,updated_at=NOW(),updated_by=$10
		WHERE $11 AND engineering.enterprise_id=$2 AND engineering.item_code=(SELECT item_code FROM scoped)
		AND engineering.drawing_code=(SELECT composite FROM previous)
		AND EXISTS(SELECT 1 FROM manufacturing_item_parameters parameter WHERE parameter.enterprise_id=$2 AND parameter.parameter_8_replicate_drawing_revision)
	)
	SELECT id,drawing_id,revision,start_date,end_date,material_spec,reason,approved_by,approval_date,is_current,created_at FROM inserted`
	return scanDrawingRev(q.db.QueryRow(ctx, sql, a.DrawingID, enterpriseID, a.Revision, a.StartDate, a.EndDate,
		a.MaterialSpec, a.Reason, a.ApprovedBy, a.ApprovalDate, updatedBy, a.IsCurrent))
}

func (q *Queries) UpdateDrawingRevision(ctx context.Context, a DrawingRevisionParams) (DBDrawingRevision, error) {
	const sql = `UPDATE drawing_revisions SET revision=$2, start_date=$3, end_date=$4, material_spec=$5,
		reason=$6, approved_by=$7, approval_date=$8, is_current=$9 WHERE id=$1 RETURNING ` + drawingRevCols
	return scanDrawingRev(q.db.QueryRow(ctx, sql, a.ID, a.Revision, a.StartDate, a.EndDate, a.MaterialSpec,
		a.Reason, a.ApprovedBy, a.ApprovalDate, a.IsCurrent))
}

func (q *Queries) UpdateDrawingRevisionForEnterprise(ctx context.Context, enterpriseID int64, a DrawingRevisionParams, updatedBy pgtype.UUID) (DBDrawingRevision, error) {
	const sql = `WITH scoped AS (
		SELECT revision.*,drawing.item_code,LEFT(drawing.code,20)||drawing.digit||drawing.format AS prefix
		FROM drawing_revisions revision JOIN drawings drawing ON drawing.id=revision.drawing_id
		WHERE revision.id=$1 AND drawing.enterprise_id=$2 FOR UPDATE
	), updated AS (
		UPDATE drawing_revisions revision SET revision=$3,start_date=$4,end_date=$5,material_spec=$6,
			reason=$7,approved_by=$8,approval_date=$9,is_current=$11
		FROM scoped WHERE revision.id=scoped.id
		RETURNING revision.id,revision.drawing_id,revision.revision,revision.start_date,revision.end_date,revision.material_spec,
			revision.reason,revision.approved_by,revision.approval_date,revision.is_current,revision.created_at
	), clear_current AS (
		UPDATE drawing_revisions SET is_current=FALSE WHERE drawing_id=(SELECT drawing_id FROM scoped) AND id<>$1 AND $11
	), replicated AS (
		UPDATE item_engineering_drawings engineering SET drawing_code=(SELECT prefix FROM scoped)||$3,updated_at=NOW(),updated_by=$10
		WHERE $11 AND engineering.enterprise_id=$2 AND engineering.item_code=(SELECT item_code FROM scoped)
		AND engineering.drawing_code=(SELECT prefix||revision FROM scoped)
		AND EXISTS(SELECT 1 FROM manufacturing_item_parameters parameter WHERE parameter.enterprise_id=$2 AND parameter.parameter_8_replicate_drawing_revision)
	)
	SELECT id,drawing_id,revision,start_date,end_date,material_spec,reason,approved_by,approval_date,is_current,created_at FROM updated`
	return scanDrawingRev(q.db.QueryRow(ctx, sql, a.ID, enterpriseID, a.Revision, a.StartDate, a.EndDate,
		a.MaterialSpec, a.Reason, a.ApprovedBy, a.ApprovalDate, updatedBy, a.IsCurrent))
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

type DBItemEngineeringDrawing struct {
	EnterpriseID int64
	ItemCode     int64
	Mask         string
	DrawingCode  string
}

func (q *Queries) UpsertItemEngineeringDrawing(ctx context.Context, enterpriseID, itemCode int64, mask, drawingCode string, updatedBy pgtype.UUID) (DBItemEngineeringDrawing, error) {
	const sql = `INSERT INTO item_engineering_drawings(enterprise_id,item_code,mask,drawing_code,updated_by)
		SELECT $1,$2,$3::varchar,$4,$5 WHERE EXISTS(SELECT 1 FROM items WHERE code=$2)
		AND ((NOT EXISTS(SELECT 1 FROM item_masks WHERE item_code=$2) AND $3::varchar='')
			OR EXISTS(SELECT 1 FROM item_masks WHERE item_code=$2 AND mask=$3::varchar))
		ON CONFLICT(enterprise_id,item_code,mask) DO UPDATE SET drawing_code=EXCLUDED.drawing_code,updated_at=NOW(),updated_by=EXCLUDED.updated_by
		RETURNING enterprise_id,item_code,mask,drawing_code`
	var result DBItemEngineeringDrawing
	err := q.db.QueryRow(ctx, sql, enterpriseID, itemCode, mask, drawingCode, updatedBy).Scan(&result.EnterpriseID, &result.ItemCode, &result.Mask, &result.DrawingCode)
	return result, err
}

func (q *Queries) GetItemEngineeringDrawing(ctx context.Context, enterpriseID, itemCode int64, mask string) (DBItemEngineeringDrawing, error) {
	var result DBItemEngineeringDrawing
	err := q.db.QueryRow(ctx, `SELECT enterprise_id,item_code,mask,drawing_code FROM item_engineering_drawings WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3`, enterpriseID, itemCode, mask).
		Scan(&result.EnterpriseID, &result.ItemCode, &result.Mask, &result.DrawingCode)
	return result, err
}

func (q *Queries) UpsertDrawingManufacturingParameters(ctx context.Context, enterpriseID int64, replicate bool, updatedBy pgtype.UUID) error {
	_, err := q.db.Exec(ctx, `INSERT INTO manufacturing_item_parameters(enterprise_id,parameter_8_replicate_drawing_revision,updated_by)
		VALUES($1,$2,$3) ON CONFLICT(enterprise_id) DO UPDATE SET parameter_8_replicate_drawing_revision=EXCLUDED.parameter_8_replicate_drawing_revision,updated_at=NOW(),updated_by=EXCLUDED.updated_by`, enterpriseID, replicate, updatedBy)
	return err
}

func (q *Queries) GetDrawingManufacturingParameters(ctx context.Context, enterpriseID int64) (bool, error) {
	var replicate bool
	err := q.db.QueryRow(ctx, `SELECT COALESCE((SELECT parameter_8_replicate_drawing_revision FROM manufacturing_item_parameters WHERE enterprise_id=$1),FALSE)`, enterpriseID).Scan(&replicate)
	return replicate, err
}
