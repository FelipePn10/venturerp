package procurement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"

	"github.com/FelipePn10/panossoerp/internal/domain/procurement/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/procurement/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) domainrepo.Repository {
	return &Repository{pool: pool}
}

// empresa é o código da empresa da sessão — a convenção do módulo de compras.
// TODA consulta deste repositório filtra por ela: o módulo nasceu supondo uma
// empresa por banco e a sonda cruzada leu registros de uma empresa a partir de
// outra.
func empresa(ctx context.Context) (int64, error) {
	return tenant.Code(ctx)
}

func naoEncontrado(err error, msg string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(msg)
	}
	return err
}

func (r *Repository) CreateRecord(ctx context.Context, rec *entity.Record) (*entity.Record, error) {
	if len(rec.Payload) == 0 {
		rec.Payload = json.RawMessage(`{}`)
	}
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO procurement_records
			(record_type, status, supplier_code, purchase_order_code, purchase_order_item_code,
			 item_code, mask, warehouse_id, quantity, reference, payload, created_by, enterprise_code)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id, opened_at, closed_at, updated_at`,
		rec.RecordType, rec.Status, rec.SupplierCode, rec.PurchaseOrderCode, rec.PurchaseOrderItemCode,
		rec.ItemCode, rec.Mask, rec.WarehouseID, rec.Quantity, rec.Reference, rec.Payload, rec.CreatedBy, emp,
	).Scan(&rec.ID, &rec.OpenedAt, &rec.ClosedAt, &rec.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating procurement record: %w", err)
	}
	return rec, nil
}

func (r *Repository) GetRecord(ctx context.Context, id int64) (*entity.Record, error) {
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, baseRecordSelect()+` WHERE id=$1 AND enterprise_code=$2`, id, emp)
	if err != nil {
		return nil, fmt.Errorf("fetching procurement record: %w", err)
	}
	defer rows.Close()
	records, err := scanRecords(rows)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("registro de suprimentos %d não encontrado", id))
	}
	return records[0], nil
}

func (r *Repository) ListRecords(ctx context.Context, recordType, status string) ([]*entity.Record, error) {
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	args := []any{emp}
	conds := []string{"enterprise_code=$1"}
	if recordType != "" {
		args = append(args, recordType)
		conds = append(conds, fmt.Sprintf("record_type=$%d", len(args)))
	}
	if status != "" {
		args = append(args, status)
		conds = append(conds, fmt.Sprintf("status=$%d", len(args)))
	}
	query := baseRecordSelect() + " WHERE " + strings.Join(conds, " AND ") + ` ORDER BY opened_at DESC, id DESC`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing procurement records: %w", err)
	}
	defer rows.Close()
	return scanRecords(rows)
}

func (r *Repository) UpdateRecordStatus(ctx context.Context, id int64, status entity.RecordStatus) (*entity.Record, error) {
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE procurement_records
		SET status=$2,
		    closed_at = CASE WHEN $2 IN ('APPROVED','REJECTED','CLOSED','CANCELLED') THEN COALESCE(closed_at, NOW()) ELSE closed_at END,
		    updated_at=NOW()
		WHERE id=$1 AND enterprise_code=$3`, id, status, emp)
	if err != nil {
		return nil, fmt.Errorf("updating procurement record status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("registro de suprimentos %d não encontrado", id))
	}
	return r.GetRecord(ctx, id)
}

func (r *Repository) CreateSupplierScorecard(ctx context.Context, s *entity.SupplierScorecard) (*entity.SupplierScorecard, error) {
	s.OverallScore = (s.QualityScore * 0.40) + (s.DeliveryScore * 0.30) + (s.CommercialScore * 0.20) + (s.ServiceScore * 0.10)
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO supplier_scorecard_snapshots
			(supplier_code, period_start, period_end, quality_score, delivery_score, commercial_score,
			 service_score, overall_score, total_receipts, rejected_receipts, late_receipts, notes, created_by, enterprise_code)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, created_at`,
		s.SupplierCode, s.PeriodStart, s.PeriodEnd, s.QualityScore, s.DeliveryScore, s.CommercialScore,
		s.ServiceScore, s.OverallScore, s.TotalReceipts, s.RejectedReceipts, s.LateReceipts, s.Notes, s.CreatedBy, emp,
	).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating supplier scorecard: %w", err)
	}
	return s, nil
}

func (r *Repository) ListSupplierScorecards(ctx context.Context, supplierCode int64) ([]*entity.SupplierScorecard, error) {
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, supplier_code, period_start, period_end, quality_score, delivery_score, commercial_score,
		       service_score, overall_score, total_receipts, rejected_receipts, late_receipts, notes, created_at, created_by
		FROM supplier_scorecard_snapshots
		WHERE supplier_code=$1 AND enterprise_code=$2
		ORDER BY period_end DESC`, supplierCode, emp)
	if err != nil {
		return nil, fmt.Errorf("listing supplier scorecards: %w", err)
	}
	defer rows.Close()

	var out []*entity.SupplierScorecard
	for rows.Next() {
		var s entity.SupplierScorecard
		if err := rows.Scan(&s.ID, &s.SupplierCode, &s.PeriodStart, &s.PeriodEnd, &s.QualityScore, &s.DeliveryScore,
			&s.CommercialScore, &s.ServiceScore, &s.OverallScore, &s.TotalReceipts, &s.RejectedReceipts,
			&s.LateReceipts, &s.Notes, &s.CreatedAt, &s.CreatedBy); err != nil {
			return nil, fmt.Errorf("scanning supplier scorecard: %w", err)
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *Repository) CreateReceivingInspectionRoute(ctx context.Context, route *entity.ReceivingInspectionRoute) (*entity.ReceivingInspectionRoute, error) {
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	route.EnterpriseCode = emp
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning inspection route tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO receiving_inspection_routes
			(enterprise_code, basis, item_code, classification_code, mask, inspection_warehouse_id,
			 handling_type, storage_type, route_type, market_type, inspection_type, valid_from, valid_to, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, is_active, created_at, updated_at`,
		route.EnterpriseCode, route.Basis, route.ItemCode, route.ClassificationCode, route.Mask, route.InspectionWarehouseID,
		route.HandlingType, route.StorageType, route.RouteType, route.MarketType, route.InspectionType, route.ValidFrom, route.ValidTo, route.CreatedBy,
	).Scan(&route.ID, &route.IsActive, &route.CreatedAt, &route.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating receiving inspection route: %w", err)
	}

	for _, step := range route.Steps {
		err = tx.QueryRow(ctx, `
			INSERT INTO receiving_inspection_route_steps
				(route_id, sequence, inspection_name, kind, appointment_mode, is_required, emits_label,
				 instrument_group, sample_type, sample_unit, sample_qty, acceptance_qty, rejection_qty,
				 norm, reference, valid_to, nominal_value, min_value, max_value)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
			RETURNING id`,
			route.ID, step.Sequence, step.InspectionName, step.Kind, step.AppointmentMode, step.IsRequired, step.EmitsLabel,
			step.InstrumentGroup, step.SampleType, step.SampleUnit, step.SampleQty, step.AcceptanceQty, step.RejectionQty,
			step.Norm, step.Reference, step.ValidTo, step.NominalValue, step.MinValue, step.MaxValue,
		).Scan(&step.ID)
		if err != nil {
			return nil, fmt.Errorf("creating receiving inspection route step: %w", err)
		}
		step.RouteID = route.ID
		for _, attr := range step.Attributes {
			err = tx.QueryRow(ctx, `
				INSERT INTO receiving_inspection_step_attributes (step_id, description, is_approved)
				VALUES ($1,$2,$3)
				RETURNING id`,
				step.ID, attr.Description, attr.IsApproved,
			).Scan(&attr.ID)
			if err != nil {
				return nil, fmt.Errorf("creating receiving inspection attribute: %w", err)
			}
			attr.StepID = step.ID
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing inspection route tx: %w", err)
	}
	return r.GetReceivingInspectionRoute(ctx, route.ID)
}

func (r *Repository) GetReceivingInspectionRoute(ctx context.Context, id int64) (*entity.ReceivingInspectionRoute, error) {
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	route, err := r.getRoute(ctx, `WHERE id=$1 AND enterprise_code=$2`, id, emp)
	if err != nil {
		return nil, naoEncontrado(err, fmt.Sprintf("roteiro de inspeção %d não encontrado", id))
	}
	return route, nil
}

func (r *Repository) FindReceivingInspectionRoute(ctx context.Context, enterpriseCode int64, itemCode int64, mask string, classificationCode *string) (*entity.ReceivingInspectionRoute, error) {
	route, err := r.getRoute(ctx, `
		WHERE enterprise_code=$1
		  AND is_active=TRUE
		  AND basis='ITEM'
		  AND item_code=$2
		  AND (mask=$3 OR mask='')
		  AND valid_from <= CURRENT_DATE
		  AND (valid_to IS NULL OR valid_to >= CURRENT_DATE)
		ORDER BY CASE WHEN mask=$3 THEN 0 ELSE 1 END, valid_from DESC, id DESC
		LIMIT 1`, enterpriseCode, itemCode, mask)
	if err == nil {
		return route, nil
	}
	if classificationCode == nil || *classificationCode == "" {
		return nil, err
	}
	return r.getRoute(ctx, `
		WHERE enterprise_code=$1
		  AND is_active=TRUE
		  AND basis='CLASSIFICATION'
		  AND $2 LIKE classification_code || '%'
		  AND valid_from <= CURRENT_DATE
		  AND (valid_to IS NULL OR valid_to >= CURRENT_DATE)
		ORDER BY length(classification_code) DESC, valid_from DESC, id DESC
		LIMIT 1`, enterpriseCode, *classificationCode)
}

func (r *Repository) LinkReceivingInspectionQualityReport(ctx context.Context, enterpriseID, orderID, reportID int64, linkedBy uuid.UUID) (*entity.ReceivingInspectionQualityReport, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO quality_lot_supplier_reports
			(enterprise_id, lot_id, quality_report_id, linked_by)
		SELECT $1, o.id, q.id, $4
		FROM quality_inspection_lots o
		JOIN item_supplier_quality_reports q ON q.id=$3 AND q.enterprise_id=$1
		JOIN item_preferred_suppliers s ON s.id=q.item_supplier_id
			AND s.enterprise_id=$1 AND s.item_code=o.item_code
			AND (o.supplier_code IS NULL OR s.supplier_code=o.supplier_code)
		WHERE o.id=$2 AND o.enterprise_id=$1
		ON CONFLICT (lot_id, quality_report_id)
		DO UPDATE SET quality_report_id=EXCLUDED.quality_report_id
		RETURNING lot_id, quality_report_id, linked_at, linked_by`, enterpriseID, orderID, reportID, linkedBy)
	link := &entity.ReceivingInspectionQualityReport{EnterpriseID: enterpriseID}
	if err := row.Scan(&link.InspectionOrderID, &link.QualityReportID, &link.LinkedAt, &link.LinkedBy); err != nil {
		if err == pgx.ErrNoRows {
			return nil, errorsuc.NewNotFoundError("laudo incompatível com a empresa, o item ou o fornecedor da inspeção")
		}
		return nil, fmt.Errorf("vinculando laudo à inspeção: %w", err)
	}
	links, err := r.ListReceivingInspectionQualityReports(ctx, enterpriseID, orderID)
	if err != nil {
		return nil, err
	}
	for _, candidate := range links {
		if candidate.QualityReportID == reportID {
			return candidate, nil
		}
	}
	return nil, fmt.Errorf("laudo vinculado não pôde ser recuperado")
}

func (r *Repository) ListReceivingInspectionQualityReports(ctx context.Context, enterpriseID, orderID int64) ([]*entity.ReceivingInspectionQualityReport, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT l.enterprise_id,l.lot_id,l.quality_report_id,q.item_supplier_id,
			q.registered_on,q.status,q.report_file_name,q.report_content_type,q.notes,l.linked_at,l.linked_by
		FROM quality_lot_supplier_reports l
		JOIN item_supplier_quality_reports q ON q.id=l.quality_report_id AND q.enterprise_id=l.enterprise_id
		WHERE l.enterprise_id=$1 AND l.lot_id=$2
		ORDER BY q.registered_on DESC,q.id DESC`, enterpriseID, orderID)
	if err != nil {
		return nil, fmt.Errorf("listando laudos da inspeção: %w", err)
	}
	defer rows.Close()
	var out []*entity.ReceivingInspectionQualityReport
	for rows.Next() {
		link := &entity.ReceivingInspectionQualityReport{}
		if err := rows.Scan(&link.EnterpriseID, &link.InspectionOrderID, &link.QualityReportID, &link.ItemSupplierID,
			&link.RegisteredOn, &link.Status, &link.FileName, &link.ContentType, &link.Notes, &link.LinkedAt, &link.LinkedBy); err != nil {
			return nil, fmt.Errorf("lendo laudo da inspeção: %w", err)
		}
		out = append(out, link)
	}
	return out, rows.Err()
}

func (r *Repository) UnlinkReceivingInspectionQualityReport(ctx context.Context, enterpriseID, orderID, reportID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM quality_lot_supplier_reports
		WHERE enterprise_id=$1 AND lot_id=$2 AND quality_report_id=$3`, enterpriseID, orderID, reportID)
	if err != nil {
		return fmt.Errorf("desvinculando laudo da inspeção: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func baseRecordSelect() string {
	return `SELECT id, record_type, status, supplier_code, purchase_order_code, purchase_order_item_code,
		item_code, mask, warehouse_id, quantity, reference, payload, opened_at, closed_at, created_by, updated_at
		FROM procurement_records`
}

func (r *Repository) getRoute(ctx context.Context, where string, args ...any) (*entity.ReceivingInspectionRoute, error) {
	rows, err := r.pool.Query(ctx, baseInspectionRouteSelect()+" "+where, args...)
	if err != nil {
		return nil, fmt.Errorf("fetching receiving inspection route: %w", err)
	}
	defer rows.Close()
	route, err := scanOneRoute(rows)
	if err != nil {
		return nil, err
	}
	if err := r.loadRouteSteps(ctx, route); err != nil {
		return nil, err
	}
	return route, nil
}

func baseInspectionRouteSelect() string {
	return `SELECT id, enterprise_code, basis, item_code, classification_code, mask, inspection_warehouse_id,
		handling_type, storage_type, route_type, market_type, inspection_type, valid_from, valid_to,
		is_active, created_at, updated_at, created_by
		FROM receiving_inspection_routes`
}

func scanOneRoute(rows pgx.Rows) (*entity.ReceivingInspectionRoute, error) {
	if !rows.Next() {
		return nil, pgx.ErrNoRows
	}
	var route entity.ReceivingInspectionRoute
	if err := rows.Scan(&route.ID, &route.EnterpriseCode, &route.Basis, &route.ItemCode, &route.ClassificationCode,
		&route.Mask, &route.InspectionWarehouseID, &route.HandlingType, &route.StorageType, &route.RouteType,
		&route.MarketType, &route.InspectionType, &route.ValidFrom, &route.ValidTo, &route.IsActive,
		&route.CreatedAt, &route.UpdatedAt, &route.CreatedBy); err != nil {
		return nil, fmt.Errorf("scanning receiving inspection route: %w", err)
	}
	return &route, rows.Err()
}

func (r *Repository) loadRouteSteps(ctx context.Context, route *entity.ReceivingInspectionRoute) error {
	rows, err := r.pool.Query(ctx, `
		SELECT id, route_id, sequence, inspection_name, kind, appointment_mode, is_required, emits_label,
		       instrument_group, sample_type, sample_unit, sample_qty, acceptance_qty, rejection_qty,
		       norm, reference, valid_to, nominal_value, min_value, max_value
		FROM receiving_inspection_route_steps
		WHERE route_id=$1
		ORDER BY sequence`, route.ID)
	if err != nil {
		return fmt.Errorf("listing receiving inspection route steps: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var step entity.ReceivingInspectionRouteStep
		if err := rows.Scan(&step.ID, &step.RouteID, &step.Sequence, &step.InspectionName, &step.Kind, &step.AppointmentMode,
			&step.IsRequired, &step.EmitsLabel, &step.InstrumentGroup, &step.SampleType, &step.SampleUnit,
			&step.SampleQty, &step.AcceptanceQty, &step.RejectionQty, &step.Norm, &step.Reference, &step.ValidTo,
			&step.NominalValue, &step.MinValue, &step.MaxValue); err != nil {
			return fmt.Errorf("scanning receiving inspection route step: %w", err)
		}
		attrs, err := r.listStepAttributes(ctx, step.ID)
		if err != nil {
			return err
		}
		step.Attributes = attrs
		route.Steps = append(route.Steps, &step)
	}
	return rows.Err()
}

func (r *Repository) listStepAttributes(ctx context.Context, stepID int64) ([]*entity.ReceivingInspectionStepAttribute, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, step_id, description, is_approved
		FROM receiving_inspection_step_attributes
		WHERE step_id=$1
		ORDER BY id`, stepID)
	if err != nil {
		return nil, fmt.Errorf("listing receiving inspection step attributes: %w", err)
	}
	defer rows.Close()
	var out []*entity.ReceivingInspectionStepAttribute
	for rows.Next() {
		var attr entity.ReceivingInspectionStepAttribute
		if err := rows.Scan(&attr.ID, &attr.StepID, &attr.Description, &attr.IsApproved); err != nil {
			return nil, fmt.Errorf("scanning receiving inspection step attribute: %w", err)
		}
		out = append(out, &attr)
	}
	return out, rows.Err()
}

func scanRecords(rows pgx.Rows) ([]*entity.Record, error) {
	var out []*entity.Record
	for rows.Next() {
		var rec entity.Record
		if err := rows.Scan(&rec.ID, &rec.RecordType, &rec.Status, &rec.SupplierCode, &rec.PurchaseOrderCode,
			&rec.PurchaseOrderItemCode, &rec.ItemCode, &rec.Mask, &rec.WarehouseID, &rec.Quantity,
			&rec.Reference, &rec.Payload, &rec.OpenedAt, &rec.ClosedAt, &rec.CreatedBy, &rec.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning procurement record: %w", err)
		}
		if rec.Payload == nil {
			rec.Payload = json.RawMessage(`{}`)
		}
		out = append(out, &rec)
	}
	return out, rows.Err()
}

func ParseDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", value)
}

// ---- Approval limits (alçada de valores) ----

func (r *Repository) CreateApprovalLimit(ctx context.Context, limit *entity.ApprovalLimit) (*entity.ApprovalLimit, error) {
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	limit.EnterpriseCode = emp
	err = r.pool.QueryRow(ctx, `
		INSERT INTO purchase_approval_limits
			(enterprise_code, scope, scope_ref, currency, auto_approve_max, block_above,
			 valid_from, valid_to, notes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, is_active, created_at, updated_at`,
		limit.EnterpriseCode, limit.Scope, limit.ScopeRef, limit.Currency, limit.AutoApproveMax, limit.BlockAbove,
		limit.ValidFrom, limit.ValidTo, limit.Notes, limit.CreatedBy,
	).Scan(&limit.ID, &limit.IsActive, &limit.CreatedAt, &limit.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating approval limit: %w", err)
	}
	return limit, nil
}

func (r *Repository) ListApprovalLimits(ctx context.Context, enterpriseCode int64) ([]*entity.ApprovalLimit, error) {
	rows, err := r.pool.Query(ctx, baseApprovalLimitSelect()+`
		WHERE enterprise_code=$1
		ORDER BY scope, scope_ref NULLS FIRST, valid_from DESC`, enterpriseCode)
	if err != nil {
		return nil, fmt.Errorf("listing approval limits: %w", err)
	}
	defer rows.Close()
	return scanApprovalLimits(rows)
}

func (r *Repository) FindApprovalLimit(ctx context.Context, enterpriseCode int64, supplierRef, costCenterRef, categoryRef *string) (*entity.ApprovalLimit, error) {
	// Try the most specific scopes first, then GLOBAL.
	type probe struct {
		scope string
		ref   *string
	}
	probes := []probe{
		{"SUPPLIER", supplierRef},
		{"COST_CENTER", costCenterRef},
		{"CATEGORY", categoryRef},
		{"GLOBAL", nil},
	}
	for _, p := range probes {
		if p.scope != "GLOBAL" && (p.ref == nil || *p.ref == "") {
			continue
		}
		where := `
			WHERE enterprise_code=$1
			  AND is_active=TRUE
			  AND scope=$2
			  AND valid_from <= CURRENT_DATE
			  AND (valid_to IS NULL OR valid_to >= CURRENT_DATE)`
		args := []any{enterpriseCode, p.scope}
		if p.scope == "GLOBAL" {
			where += ` AND scope_ref IS NULL`
		} else {
			where += ` AND scope_ref=$3`
			args = append(args, *p.ref)
		}
		where += ` ORDER BY valid_from DESC, id DESC LIMIT 1`
		rows, err := r.pool.Query(ctx, baseApprovalLimitSelect()+where, args...)
		if err != nil {
			return nil, fmt.Errorf("finding approval limit: %w", err)
		}
		limits, err := scanApprovalLimits(rows)
		rows.Close()
		if err != nil {
			return nil, err
		}
		if len(limits) > 0 {
			return limits[0], nil
		}
	}
	return nil, pgx.ErrNoRows
}

func baseApprovalLimitSelect() string {
	return `SELECT id, enterprise_code, scope, scope_ref, currency, auto_approve_max, block_above,
		is_active, valid_from, valid_to, notes, created_by, created_at, updated_at
		FROM purchase_approval_limits`
}

func scanApprovalLimits(rows pgx.Rows) ([]*entity.ApprovalLimit, error) {
	var out []*entity.ApprovalLimit
	for rows.Next() {
		var l entity.ApprovalLimit
		if err := rows.Scan(&l.ID, &l.EnterpriseCode, &l.Scope, &l.ScopeRef, &l.Currency, &l.AutoApproveMax,
			&l.BlockAbove, &l.IsActive, &l.ValidFrom, &l.ValidTo, &l.Notes, &l.CreatedBy, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning approval limit: %w", err)
		}
		out = append(out, &l)
	}
	return out, rows.Err()
}

// ---- Supplier contracts ----

func (r *Repository) CreateSupplierContract(ctx context.Context, c *entity.SupplierContract) (*entity.SupplierContract, error) {
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	c.EnterpriseCode = emp
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning contract tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO supplier_contracts
			(enterprise_code, supplier_code, contract_number, description, status, currency,
			 valid_from, valid_to, price_index, notes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, status, created_at, updated_at`,
		c.EnterpriseCode, c.SupplierCode, c.ContractNumber, c.Description, c.Status, c.Currency,
		c.ValidFrom, c.ValidTo, c.PriceIndex, c.Notes, c.CreatedBy,
	).Scan(&c.ID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating supplier contract: %w", err)
	}
	for _, it := range c.Items {
		err = tx.QueryRow(ctx, `
			INSERT INTO supplier_contract_items
				(contract_id, item_code, mask, unit, contracted_qty, consumed_qty, unit_price, min_order_qty, notes)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			RETURNING id`,
			c.ID, it.ItemCode, it.Mask, it.Unit, it.ContractedQty, it.ConsumedQty, it.UnitPrice, it.MinOrderQty, it.Notes,
		).Scan(&it.ID)
		if err != nil {
			return nil, fmt.Errorf("creating supplier contract item: %w", err)
		}
		it.ContractID = c.ID
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing contract tx: %w", err)
	}
	return r.GetSupplierContract(ctx, c.ID)
}

func (r *Repository) GetSupplierContract(ctx context.Context, id int64) (*entity.SupplierContract, error) {
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	var c entity.SupplierContract
	err = r.pool.QueryRow(ctx, baseSupplierContractSelect()+` WHERE id=$1 AND enterprise_code=$2`, id, emp).Scan(
		&c.ID, &c.EnterpriseCode, &c.SupplierCode, &c.ContractNumber, &c.Description, &c.Status, &c.Currency,
		&c.ValidFrom, &c.ValidTo, &c.PriceIndex, &c.Notes, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, naoEncontrado(fmt.Errorf("fetching supplier contract: %w", err), fmt.Sprintf("contrato %d não encontrado", id))
	}
	items, err := r.listContractItems(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	c.Items = items
	return &c, nil
}

func (r *Repository) ListSupplierContracts(ctx context.Context, supplierCode int64, status string) ([]*entity.SupplierContract, error) {
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	query := baseSupplierContractSelect()
	args := []any{emp}
	conds := []string{"enterprise_code=$1"}
	if supplierCode > 0 {
		args = append(args, supplierCode)
		conds = append(conds, fmt.Sprintf("supplier_code=$%d", len(args)))
	}
	if status != "" {
		args = append(args, status)
		conds = append(conds, fmt.Sprintf("status=$%d", len(args)))
	}
	if len(conds) > 0 {
		query += " WHERE " + conds[0]
		for _, c := range conds[1:] {
			query += " AND " + c
		}
	}
	query += " ORDER BY valid_from DESC, id DESC"
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing supplier contracts: %w", err)
	}
	defer rows.Close()
	var out []*entity.SupplierContract
	for rows.Next() {
		var c entity.SupplierContract
		if err := rows.Scan(&c.ID, &c.EnterpriseCode, &c.SupplierCode, &c.ContractNumber, &c.Description, &c.Status,
			&c.Currency, &c.ValidFrom, &c.ValidTo, &c.PriceIndex, &c.Notes, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning supplier contract: %w", err)
		}
		out = append(out, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, c := range out {
		items, err := r.listContractItems(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		c.Items = items
	}
	return out, nil
}

func (r *Repository) UpdateSupplierContractStatus(ctx context.Context, id int64, status string) (*entity.SupplierContract, error) {
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE supplier_contracts SET status=$2, updated_at=NOW() WHERE id=$1 AND enterprise_code=$3`, id, status, emp)
	if err != nil {
		return nil, fmt.Errorf("updating supplier contract status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("contrato %d não encontrado", id))
	}
	return r.GetSupplierContract(ctx, id)
}

func (r *Repository) FindContractItem(ctx context.Context, contractID, itemCode int64, mask string) (*entity.SupplierContractItem, error) {
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	var it entity.SupplierContractItem
	err = r.pool.QueryRow(ctx, `
		SELECT ci.id, ci.contract_id, ci.item_code, ci.mask, ci.unit, ci.contracted_qty, ci.consumed_qty, ci.unit_price, ci.min_order_qty, ci.notes
		FROM supplier_contract_items ci
		JOIN supplier_contracts c ON c.id = ci.contract_id AND c.enterprise_code = $4
		WHERE ci.contract_id=$1 AND ci.item_code=$2 AND ci.mask=$3`, contractID, itemCode, mask, emp).Scan(
		&it.ID, &it.ContractID, &it.ItemCode, &it.Mask, &it.Unit, &it.ContractedQty, &it.ConsumedQty, &it.UnitPrice, &it.MinOrderQty, &it.Notes)
	if err != nil {
		return nil, fmt.Errorf("finding contract item: %w", err)
	}
	return &it, nil
}

// ConsumeContractItem atomically increases consumed_qty without exceeding the
// contracted balance, returning the updated line.
func (r *Repository) ConsumeContractItem(ctx context.Context, contractItemID int64, qty float64) (*entity.SupplierContractItem, error) {
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	var it entity.SupplierContractItem
	err = r.pool.QueryRow(ctx, `
		UPDATE supplier_contract_items
		SET consumed_qty = consumed_qty + $2
		WHERE id=$1 AND consumed_qty + $2 <= contracted_qty + 0.0001
		  AND EXISTS (SELECT 1 FROM supplier_contracts c WHERE c.id = supplier_contract_items.contract_id AND c.enterprise_code = $3)
		RETURNING id, contract_id, item_code, mask, unit, contracted_qty, consumed_qty, unit_price, min_order_qty, notes`,
		contractItemID, qty, emp).Scan(
		&it.ID, &it.ContractID, &it.ItemCode, &it.Mask, &it.Unit, &it.ContractedQty, &it.ConsumedQty, &it.UnitPrice, &it.MinOrderQty, &it.Notes)
	if err == pgx.ErrNoRows {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("o consumo de %.4f excede o saldo contratado do item de contrato %d", qty, contractItemID))
	}
	if err != nil {
		return nil, fmt.Errorf("consuming contract item: %w", err)
	}
	return &it, nil
}

func (r *Repository) listContractItems(ctx context.Context, contractID int64) ([]*entity.SupplierContractItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, contract_id, item_code, mask, unit, contracted_qty, consumed_qty, unit_price, min_order_qty, notes
		FROM supplier_contract_items
		WHERE contract_id=$1
		ORDER BY id`, contractID)
	if err != nil {
		return nil, fmt.Errorf("listing supplier contract items: %w", err)
	}
	defer rows.Close()
	var out []*entity.SupplierContractItem
	for rows.Next() {
		var it entity.SupplierContractItem
		if err := rows.Scan(&it.ID, &it.ContractID, &it.ItemCode, &it.Mask, &it.Unit, &it.ContractedQty,
			&it.ConsumedQty, &it.UnitPrice, &it.MinOrderQty, &it.Notes); err != nil {
			return nil, fmt.Errorf("scanning supplier contract item: %w", err)
		}
		out = append(out, &it)
	}
	return out, rows.Err()
}

func baseSupplierContractSelect() string {
	return `SELECT id, enterprise_code, supplier_code, contract_number, description, status, currency,
		valid_from, valid_to, price_index, notes, created_by, created_at, updated_at
		FROM supplier_contracts`
}

// ---- Supplier performance for IQF ----

func (r *Repository) AggregateSupplierPerformance(ctx context.Context, supplierCode int64, from, to time.Time) (*entity.SupplierPerformanceAggregate, error) {
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	agg := &entity.SupplierPerformanceAggregate{}
	empresaID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	// Qualidade: lotes de inspeção decididos do fornecedor no período (a
	// decisão marcada para não afetar o IQF fica de fora). Recusa é lote com
	// parte rejeitada, sucateada ou em retrabalho, ou recusado pela amostragem.
	err = r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE d.qtd_rejeitada + d.qtd_sucata + d.qtd_retrabalho > 0 OR d.aceito_amostragem = FALSE),
			COALESCE(SUM(l.quantidade), 0),
			COALESCE(SUM(d.qtd_rejeitada + d.qtd_sucata + d.qtd_retrabalho), 0)
		FROM quality_inspection_lots l
		JOIN quality_lot_decisions d ON d.lot_id = l.id AND d.enterprise_id = l.enterprise_id
		WHERE l.supplier_code = $1 AND l.enterprise_id = $4 AND d.afeta_iqf
		  AND l.created_at >= $2 AND l.created_at < ($3::date + INTERVAL '1 day')`,
		supplierCode, from, to, empresaID).Scan(&agg.TotalReceipts, &agg.RejectedReceipts, &agg.InspectedQty, &agg.RejectedQty)
	if err != nil {
		return nil, fmt.Errorf("aggregating supplier quality: %w", err)
	}
	// Delivery: late lines from purchase orders (received after the promised/delivery date).
	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM purchase_order_items i
		JOIN purchase_orders o ON o.code = i.purchase_order_code
		WHERE o.supplier_code = $1 AND o.enterprise_code = $4
		  AND i.received_qty > 0
		  AND COALESCE(i.promised_date, i.delivery_date, o.delivery_date) IS NOT NULL
		  AND i.updated_at::date > COALESCE(i.promised_date, i.delivery_date, o.delivery_date)::date
		  AND o.emission_date >= $2 AND o.emission_date < ($3::date + INTERVAL '1 day')`,
		supplierCode, from, to, emp).Scan(&agg.LateReceipts)
	if err != nil {
		return nil, fmt.Errorf("aggregating supplier delivery: %w", err)
	}
	return agg, nil
}

// ---- Consolidated purchase movement history ----

func (r *Repository) ListPurchaseMovementHistory(ctx context.Context, supplierCode *int64, itemCode *int64, limit int) ([]*entity.PurchaseMovementHistoryRow, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	emp, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	args := []any{emp}
	conds := []string{"o.enterprise_code=$1"}
	if supplierCode != nil {
		args = append(args, *supplierCode)
		conds = append(conds, fmt.Sprintf("o.supplier_code=$%d", len(args)))
	}
	if itemCode != nil {
		args = append(args, *itemCode)
		conds = append(conds, fmt.Sprintf("i.item_code=$%d", len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + conds[0]
		for _, c := range conds[1:] {
			where += " AND " + c
		}
	}
	args = append(args, limit)
	query := `
		SELECT o.supplier_code, o.code, o.order_number, i.item_code, i.mask,
		       i.requested_qty, i.received_qty, i.cancelled_qty, i.unit_price, i.status,
		       o.emission_date, o.delivery_date
		FROM purchase_order_items i
		JOIN purchase_orders o ON o.code = i.purchase_order_code` + where + `
		ORDER BY o.emission_date DESC, o.code DESC, i.sequence
		LIMIT $` + fmt.Sprintf("%d", len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing purchase movement history: %w", err)
	}
	defer rows.Close()
	var out []*entity.PurchaseMovementHistoryRow
	for rows.Next() {
		var h entity.PurchaseMovementHistoryRow
		if err := rows.Scan(&h.SupplierCode, &h.PurchaseOrderCode, &h.OrderNumber, &h.ItemCode, &h.Mask,
			&h.RequestedQty, &h.ReceivedQty, &h.CancelledQty, &h.UnitPrice, &h.Status, &h.EmissionDate, &h.DeliveryDate); err != nil {
			return nil, fmt.Errorf("scanning purchase movement history: %w", err)
		}
		out = append(out, &h)
	}
	return out, rows.Err()
}
