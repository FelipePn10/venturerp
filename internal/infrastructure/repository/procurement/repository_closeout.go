package procurement

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/procurement/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ---- Receiving notices + divergences (FAVR) ----

func (r *Repository) CreateReceivingNotice(ctx context.Context, n *entity.ReceivingNotice) (*entity.ReceivingNotice, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning receiving notice tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO receiving_notices
			(enterprise_code, supplier_code, purchase_order_code, carrier_code, status, dock,
			 scheduled_at, arrived_at, invoice_number, blocked, notes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id, notice_number, status, blocked, created_at, updated_at`,
		n.EnterpriseCode, n.SupplierCode, n.PurchaseOrderCode, n.CarrierCode, n.Status, n.Dock,
		n.ScheduledAt, n.ArrivedAt, n.InvoiceNumber, n.Blocked, n.Notes, n.CreatedBy,
	).Scan(&n.ID, &n.NoticeNumber, &n.Status, &n.Blocked, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating receiving notice: %w", err)
	}
	for _, it := range n.Items {
		err = tx.QueryRow(ctx, `
			INSERT INTO receiving_notice_items
				(notice_id, purchase_order_item_code, item_code, mask, expected_qty, received_qty, unit, notes)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			RETURNING id`,
			n.ID, it.PurchaseOrderItemCode, it.ItemCode, it.Mask, it.ExpectedQty, it.ReceivedQty, it.Unit, it.Notes,
		).Scan(&it.ID)
		if err != nil {
			return nil, fmt.Errorf("creating receiving notice item: %w", err)
		}
		it.NoticeID = n.ID
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing receiving notice tx: %w", err)
	}
	return r.GetReceivingNotice(ctx, n.ID)
}

func (r *Repository) GetReceivingNotice(ctx context.Context, id int64) (*entity.ReceivingNotice, error) {
	var n entity.ReceivingNotice
	err := r.pool.QueryRow(ctx, `
		SELECT id, enterprise_code, notice_number, supplier_code, purchase_order_code, carrier_code, status,
		       dock, scheduled_at, arrived_at, invoice_number, blocked, notes, created_by, created_at, updated_at
		FROM receiving_notices WHERE id=$1`, id).Scan(
		&n.ID, &n.EnterpriseCode, &n.NoticeNumber, &n.SupplierCode, &n.PurchaseOrderCode, &n.CarrierCode, &n.Status,
		&n.Dock, &n.ScheduledAt, &n.ArrivedAt, &n.InvoiceNumber, &n.Blocked, &n.Notes, &n.CreatedBy, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("fetching receiving notice: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, notice_id, purchase_order_item_code, item_code, mask, expected_qty, received_qty, unit, notes
		FROM receiving_notice_items WHERE notice_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, fmt.Errorf("listing receiving notice items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var it entity.ReceivingNoticeItem
		if err := rows.Scan(&it.ID, &it.NoticeID, &it.PurchaseOrderItemCode, &it.ItemCode, &it.Mask,
			&it.ExpectedQty, &it.ReceivedQty, &it.Unit, &it.Notes); err != nil {
			return nil, fmt.Errorf("scanning receiving notice item: %w", err)
		}
		n.Items = append(n.Items, &it)
	}
	return &n, rows.Err()
}

func (r *Repository) ListReceivingNotices(ctx context.Context, status string) ([]*entity.ReceivingNotice, error) {
	query := `
		SELECT id, enterprise_code, notice_number, supplier_code, purchase_order_code, carrier_code, status,
		       dock, scheduled_at, arrived_at, invoice_number, blocked, notes, created_by, created_at, updated_at
		FROM receiving_notices`
	args := []any{}
	if status != "" {
		query += ` WHERE status=$1`
		args = append(args, status)
	}
	query += ` ORDER BY COALESCE(scheduled_at, created_at) DESC, id DESC LIMIT 200`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing receiving notices: %w", err)
	}
	defer rows.Close()
	var out []*entity.ReceivingNotice
	for rows.Next() {
		var n entity.ReceivingNotice
		if err := rows.Scan(&n.ID, &n.EnterpriseCode, &n.NoticeNumber, &n.SupplierCode, &n.PurchaseOrderCode,
			&n.CarrierCode, &n.Status, &n.Dock, &n.ScheduledAt, &n.ArrivedAt, &n.InvoiceNumber, &n.Blocked,
			&n.Notes, &n.CreatedBy, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning receiving notice: %w", err)
		}
		out = append(out, &n)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateReceivingNoticeStatus(ctx context.Context, id int64, status string, blocked bool) (*entity.ReceivingNotice, error) {
	_, err := r.pool.Exec(ctx, `
		UPDATE receiving_notices
		SET status=$2, blocked=$3,
		    arrived_at = CASE WHEN $2='ARRIVED' AND arrived_at IS NULL THEN NOW() ELSE arrived_at END,
		    updated_at=NOW()
		WHERE id=$1`, id, status, blocked)
	if err != nil {
		return nil, fmt.Errorf("updating receiving notice status: %w", err)
	}
	return r.GetReceivingNotice(ctx, id)
}

func (r *Repository) CreateReceivingDivergence(ctx context.Context, d *entity.ReceivingDivergence) (*entity.ReceivingDivergence, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO receiving_divergences
			(notice_id, purchase_order_code, purchase_order_item_code, supplier_code, item_code, mask,
			 divergence_type, expected_qty, actual_qty, expected_price, actual_price, resolution,
			 affects_supplier_score, notes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id, resolution, created_at`,
		d.NoticeID, d.PurchaseOrderCode, d.PurchaseOrderItemCode, d.SupplierCode, d.ItemCode, d.Mask,
		d.DivergenceType, d.ExpectedQty, d.ActualQty, d.ExpectedPrice, d.ActualPrice, d.Resolution,
		d.AffectsSupplierScore, d.Notes, d.CreatedBy,
	).Scan(&d.ID, &d.Resolution, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating receiving divergence: %w", err)
	}
	return d, nil
}

func (r *Repository) ListReceivingDivergences(ctx context.Context, supplierCode *int64, resolution string) ([]*entity.ReceivingDivergence, error) {
	query := baseDivergenceSelect()
	args := []any{}
	conds := []string{}
	if supplierCode != nil {
		args = append(args, *supplierCode)
		conds = append(conds, fmt.Sprintf("supplier_code=$%d", len(args)))
	}
	if resolution != "" {
		args = append(args, resolution)
		conds = append(conds, fmt.Sprintf("resolution=$%d", len(args)))
	}
	if len(conds) > 0 {
		query += " WHERE " + conds[0]
		for _, c := range conds[1:] {
			query += " AND " + c
		}
	}
	query += " ORDER BY created_at DESC, id DESC LIMIT 200"
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing receiving divergences: %w", err)
	}
	defer rows.Close()
	return scanDivergences(rows)
}

func (r *Repository) ResolveReceivingDivergence(ctx context.Context, id int64, resolution string) (*entity.ReceivingDivergence, error) {
	rows, err := r.pool.Query(ctx, `
		UPDATE receiving_divergences
		SET resolution=$2, resolved_at = CASE WHEN $2='PENDING' THEN NULL ELSE NOW() END
		WHERE id=$1
		RETURNING `+divergenceColumns(), id, resolution)
	if err != nil {
		return nil, fmt.Errorf("resolving receiving divergence: %w", err)
	}
	defer rows.Close()
	list, err := scanDivergences(rows)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, pgx.ErrNoRows
	}
	return list[0], nil
}

func baseDivergenceSelect() string {
	return "SELECT " + divergenceColumns() + " FROM receiving_divergences"
}

func divergenceColumns() string {
	return `id, notice_id, purchase_order_code, purchase_order_item_code, supplier_code, item_code, mask,
		divergence_type, expected_qty, actual_qty, expected_price, actual_price, resolution,
		affects_supplier_score, notes, created_by, created_at, resolved_at`
}

func scanDivergences(rows pgx.Rows) ([]*entity.ReceivingDivergence, error) {
	var out []*entity.ReceivingDivergence
	for rows.Next() {
		var d entity.ReceivingDivergence
		if err := rows.Scan(&d.ID, &d.NoticeID, &d.PurchaseOrderCode, &d.PurchaseOrderItemCode, &d.SupplierCode,
			&d.ItemCode, &d.Mask, &d.DivergenceType, &d.ExpectedQty, &d.ActualQty, &d.ExpectedPrice, &d.ActualPrice,
			&d.Resolution, &d.AffectsSupplierScore, &d.Notes, &d.CreatedBy, &d.CreatedAt, &d.ResolvedAt); err != nil {
			return nil, fmt.Errorf("scanning receiving divergence: %w", err)
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}

// ---- Supplier EDI (FEDS) ----

func (r *Repository) CreateEDIMessage(ctx context.Context, m *entity.SupplierEDIMessage) (*entity.SupplierEDIMessage, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning edi tx: %w", err)
	}
	defer tx.Rollback(ctx)

	payload := m.Payload
	if len(payload) == 0 || !json.Valid(payload) {
		payload = []byte(`{}`)
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO supplier_edi_messages
			(enterprise_code, supplier_code, direction, message_type, purchase_order_code, external_reference,
			 status, divergence_count, payload, notes, created_by, processed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id, status, created_at`,
		m.EnterpriseCode, m.SupplierCode, m.Direction, m.MessageType, m.PurchaseOrderCode, m.ExternalReference,
		m.Status, m.DivergenceCount, payload, m.Notes, m.CreatedBy, m.ProcessedAt,
	).Scan(&m.ID, &m.Status, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating edi message: %w", err)
	}
	for _, l := range m.Lines {
		err = tx.QueryRow(ctx, `
			INSERT INTO supplier_edi_lines
				(message_id, purchase_order_item_code, item_code, mask, confirmed_qty, confirmed_price, confirmed_date, divergence, notes)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			RETURNING id`,
			m.ID, l.PurchaseOrderItemCode, l.ItemCode, l.Mask, l.ConfirmedQty, l.ConfirmedPrice, l.ConfirmedDate, l.Divergence, l.Notes,
		).Scan(&l.ID)
		if err != nil {
			return nil, fmt.Errorf("creating edi line: %w", err)
		}
		l.MessageID = m.ID
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing edi tx: %w", err)
	}
	return r.GetEDIMessage(ctx, m.ID)
}

func (r *Repository) GetEDIMessage(ctx context.Context, id int64) (*entity.SupplierEDIMessage, error) {
	var m entity.SupplierEDIMessage
	err := r.pool.QueryRow(ctx, `
		SELECT id, enterprise_code, supplier_code, direction, message_type, purchase_order_code, external_reference,
		       status, divergence_count, payload, notes, created_by, created_at, processed_at
		FROM supplier_edi_messages WHERE id=$1`, id).Scan(
		&m.ID, &m.EnterpriseCode, &m.SupplierCode, &m.Direction, &m.MessageType, &m.PurchaseOrderCode, &m.ExternalReference,
		&m.Status, &m.DivergenceCount, &m.Payload, &m.Notes, &m.CreatedBy, &m.CreatedAt, &m.ProcessedAt)
	if err != nil {
		return nil, fmt.Errorf("fetching edi message: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, message_id, purchase_order_item_code, item_code, mask, confirmed_qty, confirmed_price, confirmed_date, divergence, notes
		FROM supplier_edi_lines WHERE message_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, fmt.Errorf("listing edi lines: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var l entity.SupplierEDILine
		if err := rows.Scan(&l.ID, &l.MessageID, &l.PurchaseOrderItemCode, &l.ItemCode, &l.Mask,
			&l.ConfirmedQty, &l.ConfirmedPrice, &l.ConfirmedDate, &l.Divergence, &l.Notes); err != nil {
			return nil, fmt.Errorf("scanning edi line: %w", err)
		}
		m.Lines = append(m.Lines, &l)
	}
	return &m, rows.Err()
}

func (r *Repository) ListEDIMessages(ctx context.Context, supplierCode *int64) ([]*entity.SupplierEDIMessage, error) {
	query := `
		SELECT id, enterprise_code, supplier_code, direction, message_type, purchase_order_code, external_reference,
		       status, divergence_count, payload, notes, created_by, created_at, processed_at
		FROM supplier_edi_messages`
	args := []any{}
	if supplierCode != nil {
		query += ` WHERE supplier_code=$1`
		args = append(args, *supplierCode)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT 200`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing edi messages: %w", err)
	}
	defer rows.Close()
	var out []*entity.SupplierEDIMessage
	for rows.Next() {
		var m entity.SupplierEDIMessage
		if err := rows.Scan(&m.ID, &m.EnterpriseCode, &m.SupplierCode, &m.Direction, &m.MessageType,
			&m.PurchaseOrderCode, &m.ExternalReference, &m.Status, &m.DivergenceCount, &m.Payload,
			&m.Notes, &m.CreatedBy, &m.CreatedAt, &m.ProcessedAt); err != nil {
			return nil, fmt.Errorf("scanning edi message: %w", err)
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

// ---- Import processes (FREC0203 / FIMP) ----

func (r *Repository) CreateImportProcess(ctx context.Context, p *entity.ImportProcess) (*entity.ImportProcess, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning import tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO import_processes
			(enterprise_code, supplier_code, purchase_order_code, reference, incoterm, currency,
			 exchange_rate, apportion_basis, status, notes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, process_number, status, created_at, updated_at`,
		p.EnterpriseCode, p.SupplierCode, p.PurchaseOrderCode, p.Reference, p.Incoterm, p.Currency,
		p.ExchangeRate, p.ApportionBasis, p.Status, p.Notes, p.CreatedBy,
	).Scan(&p.ID, &p.ProcessNumber, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating import process: %w", err)
	}
	for _, e := range p.Expenses {
		err = tx.QueryRow(ctx, `
			INSERT INTO import_expenses (process_id, expense_type, amount, in_item_cost, notes)
			VALUES ($1,$2,$3,$4,$5) RETURNING id`,
			p.ID, e.ExpenseType, e.Amount, e.InItemCost, e.Notes,
		).Scan(&e.ID)
		if err != nil {
			return nil, fmt.Errorf("creating import expense: %w", err)
		}
		e.ProcessID = p.ID
	}
	for _, it := range p.Items {
		err = tx.QueryRow(ctx, `
			INSERT INTO import_process_items
				(process_id, item_code, mask, quantity, weight, fob_unit_price, apportioned_expenses, landed_unit_cost, notes)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
			p.ID, it.ItemCode, it.Mask, it.Quantity, it.Weight, it.FobUnitPrice, it.ApportionedExpenses, it.LandedUnitCost, it.Notes,
		).Scan(&it.ID)
		if err != nil {
			return nil, fmt.Errorf("creating import process item: %w", err)
		}
		it.ProcessID = p.ID
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing import tx: %w", err)
	}
	return r.GetImportProcess(ctx, p.ID)
}

func (r *Repository) GetImportProcess(ctx context.Context, id int64) (*entity.ImportProcess, error) {
	var p entity.ImportProcess
	err := r.pool.QueryRow(ctx, `
		SELECT id, enterprise_code, process_number, supplier_code, purchase_order_code, reference, incoterm,
		       currency, exchange_rate, apportion_basis, status, notes, created_by, created_at, updated_at, nationalized_at
		FROM import_processes WHERE id=$1`, id).Scan(
		&p.ID, &p.EnterpriseCode, &p.ProcessNumber, &p.SupplierCode, &p.PurchaseOrderCode, &p.Reference, &p.Incoterm,
		&p.Currency, &p.ExchangeRate, &p.ApportionBasis, &p.Status, &p.Notes, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.NationalizedAt)
	if err != nil {
		return nil, fmt.Errorf("fetching import process: %w", err)
	}
	items, err := r.pool.Query(ctx, `
		SELECT id, process_id, item_code, mask, quantity, weight, fob_unit_price, apportioned_expenses, landed_unit_cost, notes
		FROM import_process_items WHERE process_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, fmt.Errorf("listing import process items: %w", err)
	}
	defer items.Close()
	for items.Next() {
		var it entity.ImportProcessItem
		if err := items.Scan(&it.ID, &it.ProcessID, &it.ItemCode, &it.Mask, &it.Quantity, &it.Weight,
			&it.FobUnitPrice, &it.ApportionedExpenses, &it.LandedUnitCost, &it.Notes); err != nil {
			return nil, fmt.Errorf("scanning import process item: %w", err)
		}
		p.Items = append(p.Items, &it)
	}
	if err := items.Err(); err != nil {
		return nil, err
	}
	exps, err := r.pool.Query(ctx, `
		SELECT id, process_id, expense_type, amount, in_item_cost, notes
		FROM import_expenses WHERE process_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, fmt.Errorf("listing import expenses: %w", err)
	}
	defer exps.Close()
	for exps.Next() {
		var e entity.ImportExpense
		if err := exps.Scan(&e.ID, &e.ProcessID, &e.ExpenseType, &e.Amount, &e.InItemCost, &e.Notes); err != nil {
			return nil, fmt.Errorf("scanning import expense: %w", err)
		}
		p.Expenses = append(p.Expenses, &e)
	}
	return &p, exps.Err()
}

func (r *Repository) ListImportProcesses(ctx context.Context, status string) ([]*entity.ImportProcess, error) {
	query := `
		SELECT id, enterprise_code, process_number, supplier_code, purchase_order_code, reference, incoterm,
		       currency, exchange_rate, apportion_basis, status, notes, created_by, created_at, updated_at, nationalized_at
		FROM import_processes`
	args := []any{}
	if status != "" {
		query += ` WHERE status=$1`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT 200`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing import processes: %w", err)
	}
	defer rows.Close()
	var out []*entity.ImportProcess
	for rows.Next() {
		var p entity.ImportProcess
		if err := rows.Scan(&p.ID, &p.EnterpriseCode, &p.ProcessNumber, &p.SupplierCode, &p.PurchaseOrderCode,
			&p.Reference, &p.Incoterm, &p.Currency, &p.ExchangeRate, &p.ApportionBasis, &p.Status, &p.Notes,
			&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.NationalizedAt); err != nil {
			return nil, fmt.Errorf("scanning import process: %w", err)
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// UpdateImportItemCosts persists recomputed landed costs for the given items.
func (r *Repository) UpdateImportItemCosts(ctx context.Context, items []*entity.ImportProcessItem) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning import cost tx: %w", err)
	}
	defer tx.Rollback(ctx)
	for _, it := range items {
		_, err = tx.Exec(ctx, `
			UPDATE import_process_items
			SET apportioned_expenses=$2, landed_unit_cost=$3
			WHERE id=$1`, it.ID, it.ApportionedExpenses, it.LandedUnitCost)
		if err != nil {
			return fmt.Errorf("updating import item cost: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) UpdateImportProcessStatus(ctx context.Context, id int64, status string) (*entity.ImportProcess, error) {
	_, err := r.pool.Exec(ctx, `
		UPDATE import_processes
		SET status=$2,
		    nationalized_at = CASE WHEN $2='NATIONALIZED' AND nationalized_at IS NULL THEN NOW() ELSE nationalized_at END,
		    updated_at=NOW()
		WHERE id=$1`, id, status)
	if err != nil {
		return nil, fmt.Errorf("updating import process status: %w", err)
	}
	return r.GetImportProcess(ctx, id)
}

// ---- Procurement parameters (FUTL0125) ----

func (r *Repository) UpsertParameter(ctx context.Context, p *entity.ProcurementParameter) (*entity.ProcurementParameter, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO procurement_parameters (enterprise_code, domain, param_key, param_value, value_type, description, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (enterprise_code, domain, param_key)
		DO UPDATE SET param_value=EXCLUDED.param_value, value_type=EXCLUDED.value_type,
		              description=EXCLUDED.description, updated_by=EXCLUDED.updated_by, updated_at=NOW()
		RETURNING id, updated_at`,
		p.EnterpriseCode, p.Domain, p.Key, p.Value, p.ValueType, p.Description, p.UpdatedBy,
	).Scan(&p.ID, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("upserting procurement parameter: %w", err)
	}
	return p, nil
}

func (r *Repository) ListParameters(ctx context.Context, enterpriseCode int64, domain string) ([]*entity.ProcurementParameter, error) {
	query := `
		SELECT id, enterprise_code, domain, param_key, param_value, value_type, description, updated_by, updated_at
		FROM procurement_parameters WHERE enterprise_code=$1`
	args := []any{enterpriseCode}
	if domain != "" {
		query += ` AND domain=$2`
		args = append(args, domain)
	}
	query += ` ORDER BY domain, param_key`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing procurement parameters: %w", err)
	}
	defer rows.Close()
	var out []*entity.ProcurementParameter
	for rows.Next() {
		var p entity.ProcurementParameter
		if err := rows.Scan(&p.ID, &p.EnterpriseCode, &p.Domain, &p.Key, &p.Value, &p.ValueType,
			&p.Description, &p.UpdatedBy, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning procurement parameter: %w", err)
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// GenerateItemSuppliersFromHistory (FFOR0204) creates item_preferred_suppliers
// links for every item purchased from the supplier that is not linked yet, in one
// statement. Returns how many links were created.
func (r *Repository) GenerateItemSuppliersFromHistory(ctx context.Context, supplierCode int64, actor uuid.UUID) (int, error) {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO item_preferred_suppliers (item_code, supplier_code, ranking, is_active, created_by)
		SELECT DISTINCT i.item_code, o.supplier_code, 99, TRUE, $2
		FROM purchase_order_items i
		JOIN purchase_orders o ON o.code = i.purchase_order_code
		WHERE o.supplier_code = $1
		ON CONFLICT (item_code, supplier_code) DO NOTHING`, supplierCode, actor)
	if err != nil {
		return 0, fmt.Errorf("generating item suppliers from history: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// ---- Supplier homologation (FAVF0203) ----

func (r *Repository) CreateHomologation(ctx context.Context, h *entity.SupplierHomologation) (*entity.SupplierHomologation, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO supplier_homologations (supplier_code, status, iqf_score, category, valid_until, notes, decided_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, decided_at`,
		h.SupplierCode, h.Status, h.IQFScore, h.Category, h.ValidUntil, h.Notes, h.DecidedBy,
	).Scan(&h.ID, &h.DecidedAt)
	if err != nil {
		return nil, fmt.Errorf("creating supplier homologation: %w", err)
	}
	return h, nil
}

func (r *Repository) ListHomologations(ctx context.Context, supplierCode int64) ([]*entity.SupplierHomologation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, supplier_code, status, iqf_score, category, valid_until, notes, decided_by, decided_at
		FROM supplier_homologations WHERE supplier_code=$1 ORDER BY decided_at DESC`, supplierCode)
	if err != nil {
		return nil, fmt.Errorf("listing supplier homologations: %w", err)
	}
	defer rows.Close()
	var out []*entity.SupplierHomologation
	for rows.Next() {
		var h entity.SupplierHomologation
		if err := rows.Scan(&h.ID, &h.SupplierCode, &h.Status, &h.IQFScore, &h.Category, &h.ValidUntil,
			&h.Notes, &h.DecidedBy, &h.DecidedAt); err != nil {
			return nil, fmt.Errorf("scanning supplier homologation: %w", err)
		}
		out = append(out, &h)
	}
	return out, rows.Err()
}
