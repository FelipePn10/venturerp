package purchase_price

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/purchase_price/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_price/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// This repository intentionally uses pgx for the source UNION and transactional
// copy/apply operations; these queries are dynamic and cannot be expressed safely
// by the project's static sqlc query set without multiplying variants.
type PurchasePriceRepositorySQLC struct{ pool *pgxpool.Pool }

func New(_ *sqlc.Queries, pool *pgxpool.Pool) domainrepo.PurchasePriceRepository {
	return &PurchasePriceRepositorySQLC{pool: pool}
}

type rowScanner interface{ Scan(...any) error }

func scanTable(s rowScanner) (*entity.PurchasePriceTable, error) {
	t := &entity.PurchasePriceTable{}
	err := s.Scan(&t.ID, &t.EnterpriseID, &t.Code, &t.SupplierCode, &t.Description, &t.CurrencyCode, &t.ValidityStart, &t.ValidityEnd, &t.IsActive, &t.CreatedAt, &t.CreatedBy, &t.UpdatedAt)
	return t, err
}

const tableColumns = `id,enterprise_id,code,COALESCE(supplier_code,0),description,currency_code,validity_start,validity_end,is_active,created_at,created_by,updated_at`

func (r *PurchasePriceRepositorySQLC) CreateTable(ctx context.Context, t *entity.PurchasePriceTable) (*entity.PurchasePriceTable, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO purchase_price_tables(enterprise_id,code,supplier_code,description,currency_code,validity_start,validity_end,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING `+tableColumns, t.EnterpriseID, t.Code, t.SupplierCode, t.Description, t.CurrencyCode, t.ValidityStart, t.ValidityEnd, t.CreatedBy)
	x, err := scanTable(row)
	if err != nil {
		return nil, fmt.Errorf("creating purchase price table: %w", err)
	}
	return x, nil
}
func (r *PurchasePriceRepositorySQLC) UpdateTable(ctx context.Context, t *entity.PurchasePriceTable) (*entity.PurchasePriceTable, error) {
	row := r.pool.QueryRow(ctx, `UPDATE purchase_price_tables SET supplier_code=$3,description=$4,currency_code=$5,validity_start=$6,validity_end=$7,is_active=$8,updated_at=NOW() WHERE enterprise_id=$1 AND code=$2 RETURNING `+tableColumns, t.EnterpriseID, t.Code, t.SupplierCode, t.Description, t.CurrencyCode, t.ValidityStart, t.ValidityEnd, t.IsActive)
	x, err := scanTable(row)
	if err != nil {
		return nil, fmt.Errorf("updating purchase price table: %w", err)
	}
	return x, nil
}
func (r *PurchasePriceRepositorySQLC) GetTableByCode(ctx context.Context, e, code int64) (*entity.PurchasePriceTable, error) {
	x, err := scanTable(r.pool.QueryRow(ctx, `SELECT `+tableColumns+` FROM purchase_price_tables WHERE enterprise_id=$1 AND code=$2`, e, code))
	if err != nil {
		return nil, fmt.Errorf("purchase price table %d not found: %w", code, err)
	}
	return x, nil
}
func (r *PurchasePriceRepositorySQLC) ListTables(ctx context.Context, e int64, supplier *int64, active bool) ([]*entity.PurchasePriceTable, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+tableColumns+` FROM purchase_price_tables WHERE enterprise_id=$1 AND ($2::bigint IS NULL OR supplier_code=$2) AND (NOT $3 OR is_active) ORDER BY code`, e, supplier, active)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.PurchasePriceTable{}
	for rows.Next() {
		x, err := scanTable(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *PurchasePriceRepositorySQLC) NextTableCode(ctx context.Context, e int64) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(code),0)+1 FROM purchase_price_tables WHERE enterprise_id=$1`, e).Scan(&n)
	return n, err
}

func scanItem(s rowScanner) (*entity.PurchasePriceTableItem, error) {
	x := &entity.PurchasePriceTableItem{}
	err := s.Scan(&x.ID, &x.TableID, &x.ItemCode, &x.SupplierCode, &x.UOM, &x.Price, &x.MinQty, &x.UpdateReplacementValue, &x.IsActive, &x.CreatedAt, &x.UpdatedAt)
	return x, err
}

const itemColumns = `i.id,i.table_id,i.item_code,i.supplier_code,i.uom,i.price,i.min_qty,i.update_replacement_value,i.is_active,i.created_at,i.updated_at`

func (r *PurchasePriceRepositorySQLC) AddItem(ctx context.Context, e int64, it *entity.PurchasePriceTableItem) (*entity.PurchasePriceTableItem, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, `INSERT INTO purchase_price_table_items(table_id,item_code,supplier_code,uom,price,min_qty,update_replacement_value) SELECT $2,$3,$4,$5,$6,$7,$8 FROM purchase_price_tables t WHERE t.id=$2 AND t.enterprise_id=$1 ON CONFLICT (table_id,item_code,COALESCE(supplier_code,0)) DO UPDATE SET uom=EXCLUDED.uom,price=EXCLUDED.price,min_qty=EXCLUDED.min_qty,update_replacement_value=EXCLUDED.update_replacement_value,is_active=TRUE,updated_at=NOW() RETURNING id,table_id,item_code,supplier_code,uom,price,min_qty,update_replacement_value,is_active,created_at,updated_at`, e, it.TableID, it.ItemCode, it.SupplierCode, it.UOM, it.Price, it.MinQty, it.UpdateReplacementValue)
	saved, err := scanItem(row)
	if err != nil {
		return nil, fmt.Errorf("adding purchase price item: %w", err)
	}
	if it.Adjustments != nil {
		if err = replaceAdjustments(ctx, tx, saved.ID, it.Adjustments); err != nil {
			return nil, err
		}
		saved.Adjustments = it.Adjustments
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return saved, nil
}
func (r *PurchasePriceRepositorySQLC) ListItems(ctx context.Context, e, tableID int64) ([]*entity.PurchasePriceTableItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+itemColumns+` FROM purchase_price_table_items i JOIN purchase_price_tables t ON t.id=i.table_id WHERE t.enterprise_id=$1 AND i.table_id=$2 AND i.is_active ORDER BY i.item_code`, e, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.PurchasePriceTableItem{}
	for rows.Next() {
		x, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	for _, x := range out {
		x.Adjustments, err = r.listAdjustments(ctx, x.ID)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
func (r *PurchasePriceRepositorySQLC) DeleteItem(ctx context.Context, e, id int64) error {
	tag, err := r.pool.Exec(ctx, `UPDATE purchase_price_table_items i SET is_active=FALSE,updated_at=NOW() FROM purchase_price_tables t WHERE i.table_id=t.id AND t.enterprise_id=$1 AND i.id=$2`, e, id)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
func (r *PurchasePriceRepositorySQLC) GetItemPrice(ctx context.Context, e, tableCode, itemCode int64, supplier *int64) (*entity.PurchasePriceTableItem, error) {
	x, err := scanItem(r.pool.QueryRow(ctx, `SELECT `+itemColumns+` FROM purchase_price_table_items i JOIN purchase_price_tables t ON t.id=i.table_id WHERE t.enterprise_id=$1 AND t.code=$2 AND i.item_code=$3 AND i.is_active AND (i.supplier_code=$4 OR i.supplier_code IS NULL) AND (t.validity_start IS NULL OR t.validity_start<=CURRENT_DATE) AND (t.validity_end IS NULL OR t.validity_end>=CURRENT_DATE) ORDER BY (i.supplier_code=$4) DESC NULLS LAST LIMIT 1`, e, tableCode, itemCode, supplier))
	return x, err
}
func (r *PurchasePriceRepositorySQLC) IsPreferredSupplier(ctx context.Context, e, item, supplier int64) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM item_preferred_suppliers WHERE enterprise_id=$1 AND item_code=$2 AND supplier_code=$3 AND is_preferred AND is_active AND (valid_until IS NULL OR valid_until>=CURRENT_DATE))`, e, item, supplier).Scan(&ok)
	return ok, err
}

func (r *PurchasePriceRepositorySQLC) ListItemCandidates(ctx context.Context, e, code int64, mode, order string, classID *int64) ([]entity.ItemCandidate, error) {
	supplierOnly := mode == "SUPPLIER"
	orderSQL := "i.code"
	if order == "ALPHANUMERIC" {
		orderSQL = `COALESCE(NULLIF(s.supplier_description,''),NULLIF(i.pdm_description_technique,''),NULLIF(i.complement,''),i.code::text),i.code`
	}
	q := `SELECT i.code,COALESCE(NULLIF(i.pdm_description_technique,''),NULLIF(i.complement,''),i.code::text),s.supplier_item_code,s.supplier_description,s.uom FROM purchase_price_tables t JOIN items i ON TRUE LEFT JOIN LATERAL (SELECT x.id,x.supplier_item_code,x.supplier_description,x.uom FROM item_preferred_suppliers x WHERE x.enterprise_id=t.enterprise_id AND x.supplier_code=t.supplier_code AND x.item_code=i.code AND x.is_active AND (x.valid_until IS NULL OR x.valid_until>=CURRENT_DATE) ORDER BY x.is_preferred DESC,x.ranking,x.id LIMIT 1) s ON TRUE WHERE t.enterprise_id=$1 AND t.code=$2 AND (NOT $3 OR s.id IS NOT NULL) AND ($4::bigint IS NULL OR EXISTS(SELECT 1 FROM item_classification_assignments a WHERE a.enterprise_id=t.enterprise_id AND a.item_code=i.code AND a.classification_id=$4)) AND NOT EXISTS(SELECT 1 FROM purchase_price_table_items pi WHERE pi.table_id=t.id AND pi.item_code=i.code AND pi.is_active) ORDER BY ` + orderSQL
	rows, err := r.pool.Query(ctx, q, e, code, supplierOnly, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []entity.ItemCandidate{}
	for rows.Next() {
		var x entity.ItemCandidate
		if err = rows.Scan(&x.ItemCode, &x.InternalDescription, &x.SupplierItemCode, &x.SupplierDescription, &x.UOM); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func replaceAdjustments(ctx context.Context, tx pgx.Tx, id int64, a []*entity.PriceAdjustment) error {
	if _, err := tx.Exec(ctx, `DELETE FROM purchase_price_item_adjustments WHERE price_item_id=$1`, id); err != nil {
		return err
	}
	for _, x := range a {
		if _, err := tx.Exec(ctx, `INSERT INTO purchase_price_item_adjustments(price_item_id,sequence,adjustment_kind,calculation_type,value) VALUES($1,$2,$3,$4,$5)`, id, x.Sequence, x.Kind, x.CalculationType, x.Value); err != nil {
			return err
		}
	}
	return nil
}
func (r *PurchasePriceRepositorySQLC) ReplaceAdjustments(ctx context.Context, e, id int64, a []*entity.PriceAdjustment) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var ok bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM purchase_price_table_items i JOIN purchase_price_tables t ON t.id=i.table_id WHERE t.enterprise_id=$1 AND i.id=$2)`, e, id).Scan(&ok); err != nil || !ok {
		if err == nil {
			err = pgx.ErrNoRows
		}
		return err
	}
	if err = replaceAdjustments(ctx, tx, id, a); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *PurchasePriceRepositorySQLC) listAdjustments(ctx context.Context, id int64) ([]*entity.PriceAdjustment, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,price_item_id,sequence,adjustment_kind,calculation_type,value FROM purchase_price_item_adjustments WHERE price_item_id=$1 ORDER BY sequence`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.PriceAdjustment{}
	for rows.Next() {
		x := &entity.PriceAdjustment{}
		if err = rows.Scan(&x.ID, &x.PriceItemID, &x.Sequence, &x.Kind, &x.CalculationType, &x.Value); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *PurchasePriceRepositorySQLC) CopyAdjustments(ctx context.Context, e, src, dst int64, mode string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var count, tableCount int
	if err = tx.QueryRow(ctx, `SELECT COUNT(*),COUNT(DISTINCT i.table_id) FROM purchase_price_table_items i JOIN purchase_price_tables t ON t.id=i.table_id WHERE t.enterprise_id=$1 AND i.id=ANY($2::bigint[])`, e, []int64{src, dst}).Scan(&count, &tableCount); err != nil {
		return err
	}
	if count != 2 || tableCount != 1 {
		return pgx.ErrNoRows
	}
	if mode == "REPLACE" {
		if _, err = tx.Exec(ctx, `DELETE FROM purchase_price_item_adjustments WHERE price_item_id=$1`, dst); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO purchase_price_item_adjustments(price_item_id,sequence,adjustment_kind,calculation_type,value) SELECT $2,COALESCE((SELECT MAX(sequence) FROM purchase_price_item_adjustments WHERE price_item_id=$2),0)+ROW_NUMBER() OVER(ORDER BY sequence),adjustment_kind,calculation_type,value FROM purchase_price_item_adjustments WHERE price_item_id=$1 ORDER BY sequence`, src, dst)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PurchasePriceRepositorySQLC) ListSourcePrices(ctx context.Context, f domainrepo.SourceFilter) ([]entity.SourcePrice, error) {
	if f.TableCode != nil {
		var supplier int64
		if err := r.pool.QueryRow(ctx, `SELECT supplier_code FROM purchase_price_tables WHERE enterprise_id=$1 AND code=$2 AND is_active`, f.EnterpriseID, *f.TableCode).Scan(&supplier); err != nil {
			return nil, err
		}
		f.SupplierCode = &supplier
	}
	source := strings.ToUpper(f.Source)
	if source == "" {
		source = "BOTH"
	}
	if source != "BOTH" && source != "PURCHASE_ORDER" && source != "ENTRY_INVOICE" {
		return nil, fmt.Errorf("invalid source")
	}
	q := `SELECT source_type,source_id,document_code,document_date,supplier_code,item_code,uom,unit_price FROM (` +
		`SELECT 'PURCHASE_ORDER'::text source_type,poi.code source_id,po.order_number document_code,po.emission_date document_date,po.supplier_code,poi.item_code,COALESCE(poi.purchase_uom,poi.internal_uom,'') uom,poi.unit_price FROM purchase_order_items poi JOIN purchase_orders po ON po.code=poi.purchase_order_code JOIN enterprise e ON e.code=po.enterprise_code WHERE e.id=$1 AND po.supplier_code IS NOT NULL AND po.order_type<>'OSL' AND po.is_active AND poi.is_active AND poi.unit_price>0 AND po.emission_date BETWEEN $3 AND $4 ` +
		`UNION ALL SELECT 'ENTRY_INVOICE',fi.id,fe.numero_nf,fe.data_entrada,fe.supplier_code,fi.item_code,COALESCE(fi.uom,s.uom,i.warehouse_unit_of_measurement::text,''),fi.unit_price FROM fiscal_entry_items fi JOIN fiscal_entries fe ON fe.id=fi.fiscal_entry_id LEFT JOIN item_preferred_suppliers s ON s.enterprise_id=fe.enterprise_id AND s.supplier_code=fe.supplier_code AND s.item_code=fi.item_code AND s.is_active LEFT JOIN items i ON i.code=fi.item_code WHERE fe.enterprise_id=$1 AND fe.supplier_code IS NOT NULL AND fi.item_code IS NOT NULL AND fi.unit_price>0 AND fe.is_active AND fe.data_entrada BETWEEN $3 AND $4) x WHERE ($2::bigint IS NULL OR supplier_code=$2) AND ($5='BOTH' OR source_type=$5) ORDER BY document_date,document_code,item_code,source_type`
	rows, err := r.pool.Query(ctx, q, f.EnterpriseID, f.SupplierCode, f.Start, f.End, source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []entity.SourcePrice{}
	for rows.Next() {
		var x entity.SourcePrice
		if err = rows.Scan(&x.SourceType, &x.SourceID, &x.DocumentCode, &x.DocumentDate, &x.SupplierCode, &x.ItemCode, &x.UOM, &x.UnitPrice); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *PurchasePriceRepositorySQLC) ApplySourcePrices(ctx context.Context, e, code int64, overwrite bool, selections []domainrepo.ApplySourceSelection) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var tableID, supplier int64
	if err = tx.QueryRow(ctx, `SELECT id,supplier_code FROM purchase_price_tables WHERE enterprise_id=$1 AND code=$2 AND is_active FOR UPDATE`, e, code).Scan(&tableID, &supplier); err != nil {
		return 0, err
	}
	var applied int64
	for _, s := range selections {
		var item int64
		var uom string
		var price decimal.Decimal
		switch s.SourceType {
		case "PURCHASE_ORDER":
			err = tx.QueryRow(ctx, `SELECT poi.item_code,COALESCE(poi.purchase_uom,poi.internal_uom,''),poi.unit_price FROM purchase_order_items poi JOIN purchase_orders po ON po.code=poi.purchase_order_code JOIN enterprise en ON en.code=po.enterprise_code WHERE en.id=$1 AND po.supplier_code=$2 AND po.order_type<>'OSL' AND poi.code=$3 AND po.is_active AND poi.is_active AND poi.unit_price>0`, e, supplier, s.SourceID).Scan(&item, &uom, &price)
		case "ENTRY_INVOICE":
			err = tx.QueryRow(ctx, `SELECT fi.item_code,COALESCE(fi.uom,ips.uom,i.warehouse_unit_of_measurement::text,''),fi.unit_price FROM fiscal_entry_items fi JOIN fiscal_entries fe ON fe.id=fi.fiscal_entry_id LEFT JOIN item_preferred_suppliers ips ON ips.enterprise_id=fe.enterprise_id AND ips.supplier_code=fe.supplier_code AND ips.item_code=fi.item_code AND ips.is_active LEFT JOIN items i ON i.code=fi.item_code WHERE fe.enterprise_id=$1 AND fe.supplier_code=$2 AND fi.id=$3 AND fi.unit_price>0 AND fe.is_active`, e, supplier, s.SourceID).Scan(&item, &uom, &price)
		default:
			return 0, fmt.Errorf("invalid source_type %q", s.SourceType)
		}
		if err != nil {
			return 0, err
		}
		var tag pgconnTag
		if overwrite {
			tag, err = execTag(tx.Exec(ctx, `INSERT INTO purchase_price_table_items(table_id,item_code,supplier_code,uom,price) VALUES($1,$2,$3,NULLIF($4,''),$5) ON CONFLICT(table_id,item_code,COALESCE(supplier_code,0)) DO UPDATE SET uom=EXCLUDED.uom,price=EXCLUDED.price,is_active=TRUE,updated_at=NOW()`, tableID, item, supplier, uom, price))
		} else {
			tag, err = execTag(tx.Exec(ctx, `INSERT INTO purchase_price_table_items(table_id,item_code,supplier_code,uom,price) VALUES($1,$2,$3,NULLIF($4,''),$5) ON CONFLICT(table_id,item_code,COALESCE(supplier_code,0)) DO NOTHING`, tableID, item, supplier, uom, price))
		}
		if err != nil {
			return 0, err
		}
		applied += tag.RowsAffected()
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return applied, nil
}

type pgconnTag interface{ RowsAffected() int64 }

func execTag(tag pgconnTag, err error) (pgconnTag, error) { return tag, err }
