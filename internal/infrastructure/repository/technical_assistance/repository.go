package technical_assistance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryPGX struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *RepositoryPGX {
	return &RepositoryPGX{pool: pool}
}

func (r *RepositoryPGX) NextCallNumber(ctx context.Context, enterpriseCode int64) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(call_number), 0) + 1 FROM technical_assistance_calls WHERE enterprise_code=$1`, enterpriseCode).Scan(&n)
	return n, err
}

func (r *RepositoryPGX) CreateDefectGroup(ctx context.Context, g *entity.DefectGroup) (*entity.DefectGroup, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO technical_assistance_defect_groups (description, is_active, created_by)
		VALUES ($1,$2,$3)
		RETURNING code, description, is_active, created_at, updated_at, created_by`,
		g.Description, trueOrDefault(g.IsActive), pgutil.ToPgUUID(g.CreatedBy))
	return scanDefectGroup(row)
}

func (r *RepositoryPGX) ListDefectGroups(ctx context.Context, onlyActive bool) ([]*entity.DefectGroup, error) {
	q := `SELECT code, description, is_active, created_at, updated_at, created_by FROM technical_assistance_defect_groups`
	if onlyActive {
		q += ` WHERE is_active`
	}
	q += ` ORDER BY description`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.DefectGroup
	for rows.Next() {
		v, err := scanDefectGroup(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) CreateDefectReason(ctx context.Context, d *entity.DefectReason) (*entity.DefectReason, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO technical_assistance_defect_reasons
		(group_code, description, allows_complement, generates_revenue, requires_return_note,
		 generates_sales_order, generates_production_order, is_replacement, is_service, available_web, is_active, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING code, group_code, description, allows_complement, generates_revenue, requires_return_note,
		          generates_sales_order, generates_production_order, is_replacement, is_service, available_web,
		          is_active, created_at, updated_at, created_by`,
		d.GroupCode, d.Description, d.AllowsComplement, d.GeneratesRevenue, d.RequiresReturnNote,
		d.GeneratesSalesOrder, d.GeneratesProductionOrder, d.IsReplacement, d.IsService, d.AvailableWeb,
		trueOrDefault(d.IsActive), pgutil.ToPgUUID(d.CreatedBy))
	return scanDefectReason(row)
}

func (r *RepositoryPGX) ListDefectReasons(ctx context.Context, groupCode *int64, onlyActive bool) ([]*entity.DefectReason, error) {
	conds := []string{}
	args := []any{}
	if groupCode != nil {
		args = append(args, *groupCode)
		conds = append(conds, fmt.Sprintf("group_code=$%d", len(args)))
	}
	if onlyActive {
		conds = append(conds, "is_active")
	}
	q := `SELECT code, group_code, description, allows_complement, generates_revenue, requires_return_note,
	             generates_sales_order, generates_production_order, is_replacement, is_service, available_web,
	             is_active, created_at, updated_at, created_by
	      FROM technical_assistance_defect_reasons`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY description"
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.DefectReason
	for rows.Next() {
		v, err := scanDefectReason(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) GetDefectReason(ctx context.Context, code int64) (*entity.DefectReason, error) {
	row := r.pool.QueryRow(ctx, `SELECT code, group_code, description, allows_complement, generates_revenue, requires_return_note,
		generates_sales_order, generates_production_order, is_replacement, is_service, available_web,
		is_active, created_at, updated_at, created_by
		FROM technical_assistance_defect_reasons WHERE code=$1`, code)
	return scanDefectReason(row)
}

func (r *RepositoryPGX) CreateWarrantyResponsible(ctx context.Context, w *entity.WarrantyResponsible) (*entity.WarrantyResponsible, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO technical_assistance_warranty_responsibles
		(name, employee_code, customer_code, email, phone, is_active, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING code, name, employee_code, customer_code, email, phone, is_active, created_at, updated_at, created_by`,
		w.Name, w.EmployeeCode, w.CustomerCode, w.Email, w.Phone, trueOrDefault(w.IsActive), pgutil.ToPgUUID(w.CreatedBy))
	return scanWarrantyResponsible(row)
}

func (r *RepositoryPGX) ListWarrantyResponsibles(ctx context.Context, onlyActive bool) ([]*entity.WarrantyResponsible, error) {
	q := `SELECT code, name, employee_code, customer_code, email, phone, is_active, created_at, updated_at, created_by FROM technical_assistance_warranty_responsibles`
	if onlyActive {
		q += ` WHERE is_active`
	}
	q += ` ORDER BY name`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.WarrantyResponsible
	for rows.Next() {
		v, err := scanWarrantyResponsible(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) CreateCall(ctx context.Context, c *entity.Call) (*entity.Call, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO technical_assistance_calls
		(call_number, enterprise_code, customer_code, consumer_name, consumer_document, technical_assistant_code,
		 warranty_responsible_code, status, priority, opened_at, promised_date, subject, description,
		 return_note_required, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING code, call_number, enterprise_code, customer_code, consumer_name, consumer_document,
		          technical_assistant_code, warranty_responsible_code, status, priority, opened_at, promised_date,
		          attended_at, closed_at, subject, description, diagnosis, solution, return_note_required,
		          sales_order_code, production_order_id, service_invoice_number, close_reason,
		          is_active, created_at, updated_at, created_by`,
		c.CallNumber, c.EnterpriseCode, c.CustomerCode, c.ConsumerName, c.ConsumerDocument,
		c.TechnicalAssistantCode, c.WarrantyResponsibleCode, string(c.Status), c.Priority, c.OpenedAt,
		datePtr(c.PromisedDate), c.Subject, c.Description, c.ReturnNoteRequired, pgutil.ToPgUUID(c.CreatedBy))
	return scanCall(row)
}

func (r *RepositoryPGX) UpdateCall(ctx context.Context, c *entity.Call) (*entity.Call, error) {
	row := r.pool.QueryRow(ctx, `UPDATE technical_assistance_calls SET
		status=$2, priority=$3, promised_date=$4, attended_at=$5, closed_at=$6, subject=$7, description=$8,
		diagnosis=$9, solution=$10, return_note_required=$11, sales_order_code=$12, production_order_id=$13,
		service_invoice_number=$14, close_reason=$15, updated_at=NOW()
		WHERE code=$1
		RETURNING code, call_number, enterprise_code, customer_code, consumer_name, consumer_document,
		          technical_assistant_code, warranty_responsible_code, status, priority, opened_at, promised_date,
		          attended_at, closed_at, subject, description, diagnosis, solution, return_note_required,
		          sales_order_code, production_order_id, service_invoice_number, close_reason,
		          is_active, created_at, updated_at, created_by`,
		c.Code, string(c.Status), c.Priority, datePtr(c.PromisedDate), timePtr(c.AttendedAt), timePtr(c.ClosedAt),
		c.Subject, c.Description, c.Diagnosis, c.Solution, c.ReturnNoteRequired, c.SalesOrderCode,
		c.ProductionOrderID, c.ServiceInvoiceNumber, c.CloseReason)
	return scanCall(row)
}

func (r *RepositoryPGX) GetCall(ctx context.Context, code int64) (*entity.Call, error) {
	row := r.pool.QueryRow(ctx, callSelect()+` WHERE code=$1`, code)
	call, err := scanCall(row)
	if err != nil {
		return nil, err
	}
	call.Items, _ = r.ListCallItems(ctx, code)
	call.ReturnNotes, _ = r.ListReturnNotes(ctx, code)
	return call, nil
}

func (r *RepositoryPGX) ListCalls(ctx context.Context, filter repository.CallFilter) ([]*entity.Call, error) {
	conds := []string{}
	args := []any{}
	if filter.Status != nil {
		args = append(args, string(*filter.Status))
		conds = append(conds, fmt.Sprintf("status=$%d", len(args)))
	}
	if filter.CustomerCode != nil {
		args = append(args, *filter.CustomerCode)
		conds = append(conds, fmt.Sprintf("customer_code=$%d", len(args)))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		conds = append(conds, fmt.Sprintf("opened_at >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		conds = append(conds, fmt.Sprintf("opened_at < $%d", len(args)))
	}
	if filter.OnlyActive {
		conds = append(conds, "is_active")
	}
	q := callSelect()
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY opened_at DESC, code DESC"
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.Call
	for rows.Next() {
		v, err := scanCall(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) AddCallItem(ctx context.Context, it *entity.CallItem) (*entity.CallItem, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO technical_assistance_call_items
		(call_code, sequence, item_code, mask, serial_number, quantity, defect_reason_code, defect_complement,
		 purchase_invoice_number, purchase_invoice_date, warranty_days, warranty_until, in_warranty,
		 generates_revenue, requested_action, status, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		RETURNING code, call_code, sequence, item_code, mask, serial_number, quantity, defect_reason_code,
		          defect_complement, purchase_invoice_number, purchase_invoice_date, warranty_days, warranty_until,
		          in_warranty, generates_revenue, requested_action, status, notes, created_at, updated_at`,
		it.CallCode, it.Sequence, it.ItemCode, it.Mask, it.SerialNumber, it.Quantity, it.DefectReasonCode,
		it.DefectComplement, it.PurchaseInvoiceNumber, datePtr(it.PurchaseInvoiceDate), it.WarrantyDays,
		datePtr(it.WarrantyUntil), it.InWarranty, it.GeneratesRevenue, it.RequestedAction, it.Status, it.Notes)
	return scanCallItem(row)
}

func (r *RepositoryPGX) ListCallItems(ctx context.Context, callCode int64) ([]*entity.CallItem, error) {
	rows, err := r.pool.Query(ctx, itemSelect()+` WHERE call_code=$1 ORDER BY sequence`, callCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.CallItem
	for rows.Next() {
		v, err := scanCallItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) AddReturnNote(ctx context.Context, n *entity.ReturnNote) (*entity.ReturnNote, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO technical_assistance_return_notes
		(call_code, note_number, note_series, emission_date, customer_code, operation_type, access_key, total_value, notes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING code, call_code, note_number, note_series, emission_date, customer_code, operation_type,
		          access_key, total_value, notes, created_at, created_by`,
		n.CallCode, n.NoteNumber, n.NoteSeries, n.EmissionDate, n.CustomerCode, n.OperationType, n.AccessKey,
		n.TotalValue, n.Notes, pgutil.ToPgUUID(n.CreatedBy))
	return scanReturnNote(row)
}

func (r *RepositoryPGX) ListReturnNotes(ctx context.Context, callCode int64) ([]*entity.ReturnNote, error) {
	rows, err := r.pool.Query(ctx, returnNoteSelect()+` WHERE call_code=$1 ORDER BY emission_date DESC, code DESC`, callCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.ReturnNote
	for rows.Next() {
		v, err := scanReturnNote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) AddOrderLink(ctx context.Context, l *entity.OrderLink) (*entity.OrderLink, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO technical_assistance_order_links
		(call_code, call_item_code, generated_type, sales_order_code, production_order_id, created_by, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING code, call_code, call_item_code, generated_type, sales_order_code, production_order_id,
		          generated_at, created_by, notes`,
		l.CallCode, l.CallItemCode, l.GeneratedType, l.SalesOrderCode, l.ProductionOrderID, pgutil.ToPgUUID(l.CreatedBy), l.Notes)
	return scanOrderLink(row)
}

func (r *RepositoryPGX) ListOrderLinks(ctx context.Context, callCode int64) ([]*entity.OrderLink, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, call_code, call_item_code, generated_type, sales_order_code, production_order_id,
		generated_at, created_by, notes FROM technical_assistance_order_links WHERE call_code=$1 ORDER BY generated_at DESC`, callCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entity.OrderLink
	for rows.Next() {
		v, err := scanOrderLink(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *RepositoryPGX) Report(ctx context.Context, f repository.ReportFilter) (*repository.Report, error) {
	conds := []string{"c.is_active"}
	args := []any{}
	if f.From != nil {
		args = append(args, *f.From)
		conds = append(conds, fmt.Sprintf("c.opened_at >= $%d", len(args)))
	}
	if f.To != nil {
		args = append(args, *f.To)
		conds = append(conds, fmt.Sprintf("c.opened_at < $%d", len(args)))
	}
	if f.CustomerCode != nil {
		args = append(args, *f.CustomerCode)
		conds = append(conds, fmt.Sprintf("c.customer_code=$%d", len(args)))
	}
	if f.Status != nil {
		args = append(args, string(*f.Status))
		conds = append(conds, fmt.Sprintf("c.status=$%d", len(args)))
	}
	q := `SELECT COUNT(DISTINCT c.code),
		COUNT(DISTINCT c.code) FILTER (WHERE c.status IN ('PENDING','IN_ANALYSIS','WAITING_RETURN','WAITING_ORDER')),
		COUNT(DISTINCT c.code) FILTER (WHERE c.status='ATTENDED'),
		COUNT(DISTINCT c.code) FILTER (WHERE c.status='CLOSED'),
		COUNT(DISTINCT c.code) FILTER (WHERE c.status='CANCELLED'),
		COUNT(i.code) FILTER (WHERE i.in_warranty),
		COUNT(i.code) FILTER (WHERE i.generates_revenue),
		COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(c.attended_at, c.closed_at) - c.opened_at))/3600)
			FILTER (WHERE c.attended_at IS NOT NULL OR c.closed_at IS NOT NULL), 0)
		FROM technical_assistance_calls c
		LEFT JOIN technical_assistance_call_items i ON i.call_code=c.code
		WHERE ` + strings.Join(conds, " AND ")
	out := &repository.Report{}
	err := r.pool.QueryRow(ctx, q, args...).Scan(&out.TotalCalls, &out.PendingCalls, &out.AttendedCalls,
		&out.ClosedCalls, &out.CancelledCalls, &out.InWarrantyItems, &out.RevenueItems, &out.AverageLeadHours)
	return out, err
}

func scanDefectGroup(row pgx.Row) (*entity.DefectGroup, error) {
	var v entity.DefectGroup
	err := row.Scan(&v.Code, &v.Description, &v.IsActive, &v.CreatedAt, &v.UpdatedAt, &v.CreatedBy)
	return &v, err
}

func scanDefectReason(row pgx.Row) (*entity.DefectReason, error) {
	var v entity.DefectReason
	err := row.Scan(&v.Code, &v.GroupCode, &v.Description, &v.AllowsComplement, &v.GeneratesRevenue,
		&v.RequiresReturnNote, &v.GeneratesSalesOrder, &v.GeneratesProductionOrder, &v.IsReplacement,
		&v.IsService, &v.AvailableWeb, &v.IsActive, &v.CreatedAt, &v.UpdatedAt, &v.CreatedBy)
	return &v, err
}

func scanWarrantyResponsible(row pgx.Row) (*entity.WarrantyResponsible, error) {
	var v entity.WarrantyResponsible
	err := row.Scan(&v.Code, &v.Name, &v.EmployeeCode, &v.CustomerCode, &v.Email, &v.Phone, &v.IsActive, &v.CreatedAt, &v.UpdatedAt, &v.CreatedBy)
	return &v, err
}

func scanCall(row pgx.Row) (*entity.Call, error) {
	var v entity.Call
	var status string
	err := row.Scan(&v.Code, &v.CallNumber, &v.EnterpriseCode, &v.CustomerCode, &v.ConsumerName, &v.ConsumerDocument,
		&v.TechnicalAssistantCode, &v.WarrantyResponsibleCode, &status, &v.Priority, &v.OpenedAt, &v.PromisedDate,
		&v.AttendedAt, &v.ClosedAt, &v.Subject, &v.Description, &v.Diagnosis, &v.Solution, &v.ReturnNoteRequired,
		&v.SalesOrderCode, &v.ProductionOrderID, &v.ServiceInvoiceNumber, &v.CloseReason, &v.IsActive,
		&v.CreatedAt, &v.UpdatedAt, &v.CreatedBy)
	v.Status = entity.CallStatus(status)
	return &v, err
}

func scanCallItem(row pgx.Row) (*entity.CallItem, error) {
	var v entity.CallItem
	err := row.Scan(&v.Code, &v.CallCode, &v.Sequence, &v.ItemCode, &v.Mask, &v.SerialNumber, &v.Quantity,
		&v.DefectReasonCode, &v.DefectComplement, &v.PurchaseInvoiceNumber, &v.PurchaseInvoiceDate,
		&v.WarrantyDays, &v.WarrantyUntil, &v.InWarranty, &v.GeneratesRevenue, &v.RequestedAction,
		&v.Status, &v.Notes, &v.CreatedAt, &v.UpdatedAt)
	return &v, err
}

func scanReturnNote(row pgx.Row) (*entity.ReturnNote, error) {
	var v entity.ReturnNote
	err := row.Scan(&v.Code, &v.CallCode, &v.NoteNumber, &v.NoteSeries, &v.EmissionDate, &v.CustomerCode,
		&v.OperationType, &v.AccessKey, &v.TotalValue, &v.Notes, &v.CreatedAt, &v.CreatedBy)
	return &v, err
}

func scanOrderLink(row pgx.Row) (*entity.OrderLink, error) {
	var v entity.OrderLink
	err := row.Scan(&v.Code, &v.CallCode, &v.CallItemCode, &v.GeneratedType, &v.SalesOrderCode,
		&v.ProductionOrderID, &v.GeneratedAt, &v.CreatedBy, &v.Notes)
	return &v, err
}

func callSelect() string {
	return `SELECT code, call_number, enterprise_code, customer_code, consumer_name, consumer_document,
		technical_assistant_code, warranty_responsible_code, status, priority, opened_at, promised_date,
		attended_at, closed_at, subject, description, diagnosis, solution, return_note_required,
		sales_order_code, production_order_id, service_invoice_number, close_reason,
		is_active, created_at, updated_at, created_by FROM technical_assistance_calls`
}

func itemSelect() string {
	return `SELECT code, call_code, sequence, item_code, mask, serial_number, quantity, defect_reason_code,
		defect_complement, purchase_invoice_number, purchase_invoice_date, warranty_days, warranty_until,
		in_warranty, generates_revenue, requested_action, status, notes, created_at, updated_at
		FROM technical_assistance_call_items`
}

func returnNoteSelect() string {
	return `SELECT code, call_code, note_number, note_series, emission_date, customer_code, operation_type,
		access_key, total_value, notes, created_at, created_by FROM technical_assistance_return_notes`
}

func trueOrDefault(v bool) bool {
	return v
}

func datePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func timePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}
