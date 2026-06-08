package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// ─── inspection plans ────────────────────────────────────────────────────────

const createInspectionPlan = `INSERT INTO inspection_plans
(item_code, route_operation_id, point_type, description, sample_size, acceptance_level, instructions, is_active, created_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,TRUE,$8)
RETURNING id, item_code, route_operation_id, point_type, description, sample_size, acceptance_level, instructions, is_active, created_at, updated_at, created_by`

type CreateInspectionPlanParams struct {
	ItemCode         int64
	RouteOperationID pgtype.Int8
	PointType        InspectionPointType
	Description      string
	SampleSize       float64
	AcceptanceLevel  float64
	Instructions     pgtype.Text
	CreatedBy        pgtype.UUID
}

func (q *Queries) CreateInspectionPlan(ctx context.Context, arg CreateInspectionPlanParams) (InspectionPlan, error) {
	row := q.db.QueryRow(ctx, createInspectionPlan,
		arg.ItemCode, arg.RouteOperationID, arg.PointType, arg.Description,
		arg.SampleSize, arg.AcceptanceLevel, arg.Instructions, arg.CreatedBy,
	)
	var i InspectionPlan
	err := row.Scan(&i.ID, &i.ItemCode, &i.RouteOperationID, &i.PointType,
		&i.Description, &i.SampleSize, &i.AcceptanceLevel, &i.Instructions,
		&i.IsActive, &i.CreatedAt, &i.UpdatedAt, &i.CreatedBy)
	return i, err
}

const getInspectionPlanByID = `SELECT id, item_code, route_operation_id, point_type, description, sample_size, acceptance_level, instructions, is_active, created_at, updated_at, created_by FROM inspection_plans WHERE id = $1`

func (q *Queries) GetInspectionPlanByID(ctx context.Context, id int64) (InspectionPlan, error) {
	row := q.db.QueryRow(ctx, getInspectionPlanByID, id)
	var i InspectionPlan
	err := row.Scan(&i.ID, &i.ItemCode, &i.RouteOperationID, &i.PointType,
		&i.Description, &i.SampleSize, &i.AcceptanceLevel, &i.Instructions,
		&i.IsActive, &i.CreatedAt, &i.UpdatedAt, &i.CreatedBy)
	return i, err
}

const listPlansByItem = `SELECT id, item_code, route_operation_id, point_type, description, sample_size, acceptance_level, instructions, is_active, created_at, updated_at, created_by FROM inspection_plans WHERE item_code = $1 AND is_active = TRUE ORDER BY point_type, id`

func (q *Queries) ListPlansByItem(ctx context.Context, itemCode int64) ([]InspectionPlan, error) {
	rows, err := q.db.Query(ctx, listPlansByItem, itemCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []InspectionPlan
	for rows.Next() {
		var i InspectionPlan
		if err := rows.Scan(&i.ID, &i.ItemCode, &i.RouteOperationID, &i.PointType,
			&i.Description, &i.SampleSize, &i.AcceptanceLevel, &i.Instructions,
			&i.IsActive, &i.CreatedAt, &i.UpdatedAt, &i.CreatedBy); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const deactivateInspectionPlan = `UPDATE inspection_plans SET is_active=FALSE, updated_at=NOW() WHERE id=$1`

func (q *Queries) DeactivateInspectionPlan(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, deactivateInspectionPlan, id)
	return err
}

// ─── characteristics ─────────────────────────────────────────────────────────

const addCharacteristic = `INSERT INTO inspection_plan_characteristics (plan_id,name,nominal,tolerance_upper,tolerance_lower,unit,is_critical)
VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id,plan_id,name,nominal,tolerance_upper,tolerance_lower,unit,is_critical`

type AddCharacteristicParams struct {
	PlanID         int64
	Name           string
	Nominal        pgtype.Float8
	ToleranceUpper pgtype.Float8
	ToleranceLower pgtype.Float8
	Unit           pgtype.Text
	IsCritical     bool
}

func (q *Queries) AddCharacteristic(ctx context.Context, arg AddCharacteristicParams) (InspectionPlanCharacteristic, error) {
	row := q.db.QueryRow(ctx, addCharacteristic,
		arg.PlanID, arg.Name, arg.Nominal, arg.ToleranceUpper, arg.ToleranceLower, arg.Unit, arg.IsCritical)
	var i InspectionPlanCharacteristic
	err := row.Scan(&i.ID, &i.PlanID, &i.Name, &i.Nominal, &i.ToleranceUpper, &i.ToleranceLower, &i.Unit, &i.IsCritical)
	return i, err
}

const listCharacteristics = `SELECT id,plan_id,name,nominal,tolerance_upper,tolerance_lower,unit,is_critical FROM inspection_plan_characteristics WHERE plan_id=$1`

func (q *Queries) ListCharacteristics(ctx context.Context, planID int64) ([]InspectionPlanCharacteristic, error) {
	rows, err := q.db.Query(ctx, listCharacteristics, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []InspectionPlanCharacteristic
	for rows.Next() {
		var i InspectionPlanCharacteristic
		if err := rows.Scan(&i.ID, &i.PlanID, &i.Name, &i.Nominal, &i.ToleranceUpper, &i.ToleranceLower, &i.Unit, &i.IsCritical); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ─── quality records ─────────────────────────────────────────────────────────

const createQualityRecord = `INSERT INTO quality_records
(plan_id,production_order_id,lot,item_code,inspected_qty,approved_qty,rejected_qty,result,inspector_id,inspected_at,notes,created_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),$10,$11)
RETURNING id,plan_id,production_order_id,lot,item_code,inspected_qty,approved_qty,rejected_qty,result,inspector_id,inspected_at,notes,created_at,created_by`

type CreateQualityRecordParams struct {
	PlanID            int64
	ProductionOrderID pgtype.Int8
	Lot               pgtype.Text
	ItemCode          int64
	InspectedQty      float64
	ApprovedQty       float64
	RejectedQty       float64
	Result            InspectionResultEnum
	InspectorID       pgtype.Int8
	Notes             pgtype.Text
	CreatedBy         pgtype.UUID
}

func (q *Queries) CreateQualityRecord(ctx context.Context, arg CreateQualityRecordParams) (QualityRecord, error) {
	row := q.db.QueryRow(ctx, createQualityRecord,
		arg.PlanID, arg.ProductionOrderID, arg.Lot, arg.ItemCode,
		arg.InspectedQty, arg.ApprovedQty, arg.RejectedQty,
		arg.Result, arg.InspectorID, arg.Notes, arg.CreatedBy,
	)
	var i QualityRecord
	err := row.Scan(&i.ID, &i.PlanID, &i.ProductionOrderID, &i.Lot, &i.ItemCode,
		&i.InspectedQty, &i.ApprovedQty, &i.RejectedQty, &i.Result,
		&i.InspectorID, &i.InspectedAt, &i.Notes, &i.CreatedAt, &i.CreatedBy)
	return i, err
}

const addMeasurement = `INSERT INTO quality_measurements (record_id,characteristic_id,measured_value,is_conformant) VALUES ($1,$2,$3,$4)`

type AddMeasurementParams struct {
	RecordID         int64
	CharacteristicID int64
	MeasuredValue    float64
	IsConformant     bool
}

func (q *Queries) AddMeasurement(ctx context.Context, arg AddMeasurementParams) error {
	_, err := q.db.Exec(ctx, addMeasurement, arg.RecordID, arg.CharacteristicID, arg.MeasuredValue, arg.IsConformant)
	return err
}

const getQualityRecordByID = `SELECT id,plan_id,production_order_id,lot,item_code,inspected_qty,approved_qty,rejected_qty,result,inspector_id,inspected_at,notes,created_at,created_by FROM quality_records WHERE id=$1`

func (q *Queries) GetQualityRecordByID(ctx context.Context, id int64) (QualityRecord, error) {
	row := q.db.QueryRow(ctx, getQualityRecordByID, id)
	var i QualityRecord
	err := row.Scan(&i.ID, &i.PlanID, &i.ProductionOrderID, &i.Lot, &i.ItemCode,
		&i.InspectedQty, &i.ApprovedQty, &i.RejectedQty, &i.Result,
		&i.InspectorID, &i.InspectedAt, &i.Notes, &i.CreatedAt, &i.CreatedBy)
	return i, err
}

const listRecordsByOrder = `SELECT id,plan_id,production_order_id,lot,item_code,inspected_qty,approved_qty,rejected_qty,result,inspector_id,inspected_at,notes,created_at,created_by FROM quality_records WHERE production_order_id=$1 ORDER BY created_at DESC`

func (q *Queries) ListRecordsByOrder(ctx context.Context, orderID int64) ([]QualityRecord, error) {
	rows, err := q.db.Query(ctx, listRecordsByOrder, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []QualityRecord
	for rows.Next() {
		var i QualityRecord
		if err := rows.Scan(&i.ID, &i.PlanID, &i.ProductionOrderID, &i.Lot, &i.ItemCode,
			&i.InspectedQty, &i.ApprovedQty, &i.RejectedQty, &i.Result,
			&i.InspectorID, &i.InspectedAt, &i.Notes, &i.CreatedAt, &i.CreatedBy); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const listRecordsByItem = `SELECT id,plan_id,production_order_id,lot,item_code,inspected_qty,approved_qty,rejected_qty,result,inspector_id,inspected_at,notes,created_at,created_by FROM quality_records WHERE item_code=$1 ORDER BY created_at DESC LIMIT 100`

func (q *Queries) ListRecordsByItem(ctx context.Context, itemCode int64) ([]QualityRecord, error) {
	rows, err := q.db.Query(ctx, listRecordsByItem, itemCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []QualityRecord
	for rows.Next() {
		var i QualityRecord
		if err := rows.Scan(&i.ID, &i.PlanID, &i.ProductionOrderID, &i.Lot, &i.ItemCode,
			&i.InspectedQty, &i.ApprovedQty, &i.RejectedQty, &i.Result,
			&i.InspectorID, &i.InspectedAt, &i.Notes, &i.CreatedAt, &i.CreatedBy); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ─── non-conformances ─────────────────────────────────────────────────────────

const createNC = `INSERT INTO non_conformances
(code,quality_record_id,production_order_id,item_code,lot,nonconform_qty,description,severity,is_open,created_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,TRUE,$9)
RETURNING id,code,quality_record_id,production_order_id,item_code,lot,nonconform_qty,description,severity,root_cause,corrective_action,disposition,disposed_at,disposed_by,is_open,created_at,updated_at,created_by`

type CreateNCParams struct {
	Code              int64
	QualityRecordID   pgtype.Int8
	ProductionOrderID pgtype.Int8
	ItemCode          int64
	Lot               pgtype.Text
	NonConformQty     float64
	Description       string
	Severity          NcSeverityEnum
	CreatedBy         pgtype.UUID
}

func (q *Queries) CreateNC(ctx context.Context, arg CreateNCParams) (NonConformance, error) {
	row := q.db.QueryRow(ctx, createNC,
		arg.Code, arg.QualityRecordID, arg.ProductionOrderID,
		arg.ItemCode, arg.Lot, arg.NonConformQty, arg.Description, arg.Severity, arg.CreatedBy,
	)
	var i NonConformance
	err := row.Scan(&i.ID, &i.Code, &i.QualityRecordID, &i.ProductionOrderID, &i.ItemCode,
		&i.Lot, &i.NonconformQty, &i.Description, &i.Severity,
		&i.RootCause, &i.CorrectiveAction, &i.Disposition, &i.DisposedAt, &i.DisposedBy,
		&i.IsOpen, &i.CreatedAt, &i.UpdatedAt, &i.CreatedBy)
	return i, err
}

const getNCByID = `SELECT id,code,quality_record_id,production_order_id,item_code,lot,nonconform_qty,description,severity,root_cause,corrective_action,disposition,disposed_at,disposed_by,is_open,created_at,updated_at,created_by FROM non_conformances WHERE id=$1`

func (q *Queries) GetNCByID(ctx context.Context, id int64) (NonConformance, error) {
	row := q.db.QueryRow(ctx, getNCByID, id)
	var i NonConformance
	err := row.Scan(&i.ID, &i.Code, &i.QualityRecordID, &i.ProductionOrderID, &i.ItemCode,
		&i.Lot, &i.NonconformQty, &i.Description, &i.Severity,
		&i.RootCause, &i.CorrectiveAction, &i.Disposition, &i.DisposedAt, &i.DisposedBy,
		&i.IsOpen, &i.CreatedAt, &i.UpdatedAt, &i.CreatedBy)
	return i, err
}

const listOpenNCs = `SELECT id,code,quality_record_id,production_order_id,item_code,lot,nonconform_qty,description,severity,root_cause,corrective_action,disposition,disposed_at,disposed_by,is_open,created_at,updated_at,created_by FROM non_conformances WHERE is_open=TRUE ORDER BY severity,created_at`

func (q *Queries) ListOpenNCs(ctx context.Context) ([]NonConformance, error) {
	rows, err := q.db.Query(ctx, listOpenNCs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNCs(rows)
}

const listNCsByItem = `SELECT id,code,quality_record_id,production_order_id,item_code,lot,nonconform_qty,description,severity,root_cause,corrective_action,disposition,disposed_at,disposed_by,is_open,created_at,updated_at,created_by FROM non_conformances WHERE item_code=$1 ORDER BY created_at DESC`

func (q *Queries) ListNCsByItem(ctx context.Context, itemCode int64) ([]NonConformance, error) {
	rows, err := q.db.Query(ctx, listNCsByItem, itemCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNCs(rows)
}

const dispositionNC = `UPDATE non_conformances SET disposition=$2, disposed_at=NOW(), disposed_by=$3, is_open=FALSE, updated_at=NOW() WHERE id=$1`

type DispositionNCParams struct {
	ID          int64
	Disposition NcDispositionEnum
	DisposedBy  pgtype.UUID
}

func (q *Queries) DispositionNC(ctx context.Context, arg DispositionNCParams) error {
	_, err := q.db.Exec(ctx, dispositionNC, arg.ID, arg.Disposition, arg.DisposedBy)
	return err
}

const nextNCCode = `SELECT COALESCE(MAX(code),0)+1 FROM non_conformances`

func (q *Queries) NextNCCode(ctx context.Context) (int64, error) {
	row := q.db.QueryRow(ctx, nextNCCode)
	var n int64
	return n, row.Scan(&n)
}

func scanNCs(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]NonConformance, error) {
	var items []NonConformance
	for rows.Next() {
		var i NonConformance
		if err := rows.Scan(&i.ID, &i.Code, &i.QualityRecordID, &i.ProductionOrderID, &i.ItemCode,
			&i.Lot, &i.NonconformQty, &i.Description, &i.Severity,
			&i.RootCause, &i.CorrectiveAction, &i.Disposition, &i.DisposedAt, &i.DisposedBy,
			&i.IsOpen, &i.CreatedAt, &i.UpdatedAt, &i.CreatedBy); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
