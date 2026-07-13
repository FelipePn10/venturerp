package third_party_service

// The list queries in this repository are intentionally assembled dynamically:
// the API exposes independent optional ranges and pgx/sqlc cannot represent an
// optional ORDER BY safely. Every value remains a positional parameter and the
// order column is selected from a closed allow-list.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	domain "github.com/FelipePn10/panossoerp/internal/domain/third_party_service"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Repo struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) domain.Repository { return &Repo{db: db} }

func (r *Repo) CreatePrice(ctx context.Context, p *domain.Price, reason string) (*domain.Price, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return r.savePrice(ctx, p, reason, "CREATE", true)
}
func (r *Repo) UpdatePrice(ctx context.Context, p *domain.Price, reason string) (*domain.Price, error) {
	if p.ID <= 0 {
		return nil, errors.New("price id is required")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return r.savePrice(ctx, p, reason, "UPDATE", false)
}
func (r *Repo) savePrice(ctx context.Context, p *domain.Price, reason, action string, create bool) (*domain.Price, error) {
	eid, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(reason) == "" {
		return nil, errors.New("change reason is required")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	saved, err := r.savePriceTx(ctx, tx, eid, p, reason, action, create)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return saved, nil
}

func (r *Repo) savePriceTx(ctx context.Context, tx pgx.Tx, eid int64, p *domain.Price, reason, action string, create bool) (*domain.Price, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, errors.New("change reason is required")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	var err error
	var valid bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM operations WHERE id=$1 AND origin IN ('EXTERNA','TERCEIROS')) AND EXISTS(SELECT 1 FROM suppliers WHERE code=$2 AND is_active) AND EXISTS(SELECT 1 FROM items WHERE code=$3)`, p.OperationID, p.SupplierCode, p.ItemCode).Scan(&valid); err != nil || !valid {
		if err == nil {
			err = errors.New("external operation or active supplier not found")
		}
		return nil, err
	}
	if p.Preferred {
		_, err = tx.Exec(ctx, `UPDATE third_party_service_prices SET preferred=FALSE,updated_at=NOW() WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND operation_id=$4 AND reference_date=$5 AND preferred`, eid, p.ItemCode, p.Mask, p.OperationID, p.ReferenceDate)
		if err != nil {
			return nil, err
		}
	}
	if create {
		err = tx.QueryRow(ctx, `INSERT INTO third_party_service_prices(enterprise_id,item_code,mask,supplier_code,operation_id,uom,reference_date,preferred,unit_price,conversion_factor,freight_type,freight_value,tax_percent,formula,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NULLIF($14,''),$15) RETURNING id,created_at,updated_at`, eid, p.ItemCode, p.Mask, p.SupplierCode, p.OperationID, p.UOM, p.ReferenceDate, p.Preferred, p.UnitPrice, p.ConversionFactor, p.FreightType, p.FreightValue, p.TaxPercent, p.Formula, p.CreatedBy).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	} else {
		var tag pgconnTag
		tag, err = tx.Exec(ctx, `UPDATE third_party_service_prices SET item_code=$3,mask=$4,supplier_code=$5,operation_id=$6,uom=$7,reference_date=$8,preferred=$9,unit_price=$10,conversion_factor=$11,freight_type=$12,freight_value=$13,tax_percent=$14,formula=NULLIF($15,''),updated_at=NOW() WHERE id=$2 AND enterprise_id=$1`, eid, p.ID, p.ItemCode, p.Mask, p.SupplierCode, p.OperationID, p.UOM, p.ReferenceDate, p.Preferred, p.UnitPrice, p.ConversionFactor, p.FreightType, p.FreightValue, p.TaxPercent, p.Formula)
		if err == nil && tag.RowsAffected() == 0 {
			err = domain.ErrNotFound
		}
	}
	if err != nil {
		return nil, err
	}
	p.EnterpriseID = eid
	if _, err = tx.Exec(ctx, `DELETE FROM third_party_service_price_rules WHERE price_id=$1 AND enterprise_id=$2`, p.ID, eid); err != nil {
		return nil, err
	}
	for _, rule := range p.Rules {
		if strings.TrimSpace(rule.Characteristic) == "" {
			return nil, errors.New("rule characteristic is required")
		}
		if _, err = tx.Exec(ctx, `INSERT INTO third_party_service_price_rules(enterprise_id,price_id,characteristic,answer) VALUES($1,$2,$3,$4)`, eid, p.ID, strings.TrimSpace(rule.Characteristic), rule.Answer); err != nil {
			return nil, err
		}
	}
	snapshot, _ := json.Marshal(p)
	if _, err = tx.Exec(ctx, `INSERT INTO third_party_service_price_history(enterprise_id,price_id,action,reason,snapshot,changed_by) VALUES($1,$2,$3,$4,$5,$6)`, eid, p.ID, action, reason, snapshot, p.CreatedBy); err != nil {
		return nil, err
	}
	return p, nil
}

type pgconnTag interface{ RowsAffected() int64 }

func (r *Repo) DeletePrice(ctx context.Context, id int64, reason string, by uuid.UUID) error {
	eid, e := tenant.ID(ctx)
	if e != nil {
		return e
	}
	if strings.TrimSpace(reason) == "" {
		return errors.New("change reason is required")
	}
	tx, e := r.db.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if e = r.deletePriceTx(ctx, tx, eid, id, reason, by); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

func (r *Repo) deletePriceTx(ctx context.Context, tx pgx.Tx, eid, id int64, reason string, by uuid.UUID) error {
	if strings.TrimSpace(reason) == "" {
		return errors.New("change reason is required")
	}
	p, e := r.getPrice(ctx, tx, eid, id)
	if e != nil {
		return e
	}
	snap, _ := json.Marshal(p)
	tag, e := tx.Exec(ctx, `UPDATE third_party_service_prices SET is_active=FALSE,preferred=FALSE,updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, id, eid)
	if e != nil || tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	_, e = tx.Exec(ctx, `INSERT INTO third_party_service_price_history(enterprise_id,price_id,action,reason,snapshot,changed_by) VALUES($1,$2,'DELETE',$3,$4,$5)`, eid, id, reason, snap, by)
	if e != nil {
		return e
	}
	return nil
}
func (r *Repo) GetPrice(ctx context.Context, id int64) (*domain.Price, error) {
	eid, e := tenant.ID(ctx)
	if e != nil {
		return nil, e
	}
	return r.getPrice(ctx, r.db, eid, id)
}

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (r *Repo) getPrice(ctx context.Context, q rowQuerier, eid, id int64) (*domain.Price, error) {
	p := &domain.Price{}
	e := q.QueryRow(ctx, priceSelect+` WHERE p.enterprise_id=$1 AND p.id=$2`, eid, id).Scan(priceScan(p)...)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if e != nil {
		return nil, e
	}
	rows, e := q.Query(ctx, `SELECT id,characteristic,answer FROM third_party_service_price_rules WHERE enterprise_id=$1 AND price_id=$2 ORDER BY id`, eid, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var x domain.PriceRule
		if e = rows.Scan(&x.ID, &x.Characteristic, &x.Answer); e != nil {
			return nil, e
		}
		p.Rules = append(p.Rules, x)
	}
	return p, rows.Err()
}

const priceSelect = `SELECT p.id,p.enterprise_id,p.item_code,p.mask,p.supplier_code,p.operation_id,p.uom,p.reference_date,p.preferred,p.unit_price,p.conversion_factor,p.freight_type,p.freight_value,p.tax_percent,COALESCE(p.formula,''),p.is_active,p.created_by,p.created_at,p.updated_at,
	COALESCE((SELECT i.pdm_description_technique FROM items i WHERE i.code=p.item_code),''),
	COALESCE((SELECT s.name FROM suppliers s WHERE s.code=p.supplier_code),''),
	COALESCE((SELECT o.name FROM operations o WHERE o.id=p.operation_id),'') FROM third_party_service_prices p`

func priceScan(p *domain.Price) []any {
	return []any{&p.ID, &p.EnterpriseID, &p.ItemCode, &p.Mask, &p.SupplierCode, &p.OperationID, &p.UOM, &p.ReferenceDate, &p.Preferred, &p.UnitPrice, &p.ConversionFactor, &p.FreightType, &p.FreightValue, &p.TaxPercent, &p.Formula, &p.IsActive, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.ItemDescription, &p.SupplierName, &p.OperationName}
}
func (r *Repo) ListPrices(ctx context.Context, f domain.PriceFilter) ([]domain.Price, error) {
	eid, e := tenant.ID(ctx)
	if e != nil {
		return nil, e
	}
	q := priceSelect + ` WHERE p.enterprise_id=$1 AND p.is_active`
	args := []any{eid}
	add := func(cond string, v any) { args = append(args, v); q += fmt.Sprintf(cond, len(args)) }
	if f.ItemFrom != nil {
		add(` AND p.item_code >= $%d`, *f.ItemFrom)
	}
	if f.ItemTo != nil {
		add(` AND p.item_code <= $%d`, *f.ItemTo)
	}
	if f.SupplierFrom != nil {
		add(` AND p.supplier_code >= $%d`, *f.SupplierFrom)
	}
	if f.SupplierTo != nil {
		add(` AND p.supplier_code <= $%d`, *f.SupplierTo)
	}
	if strings.TrimSpace(f.ItemSearch) != "" {
		add(` AND EXISTS(SELECT 1 FROM items i WHERE i.code=p.item_code AND (i.code::text ILIKE '%%'||$%[1]d||'%%' OR i.pdm_description_technique ILIKE '%%'||$%[1]d||'%%' OR COALESCE(i.complement,'') ILIKE '%%'||$%[1]d||'%%'))`, strings.TrimSpace(f.ItemSearch))
	}
	if strings.TrimSpace(f.SupplierSearch) != "" {
		add(` AND EXISTS(SELECT 1 FROM suppliers s WHERE s.code=p.supplier_code AND (s.code::text ILIKE '%%'||$%[1]d||'%%' OR s.name ILIKE '%%'||$%[1]d||'%%'))`, strings.TrimSpace(f.SupplierSearch))
	}
	if f.ClassificationMaskCode != nil || len(f.ClassificationCodes) > 0 {
		args = append(args, f.ClassificationMaskCode, f.ClassificationCodes)
		maskArg, codesArg := len(args)-1, len(args)
		q += fmt.Sprintf(` AND EXISTS(SELECT 1 FROM item_classification_assignments a JOIN item_classifications c ON c.id=a.classification_id AND c.is_active JOIN item_classification_masks m ON m.id=c.mask_id AND m.is_active WHERE a.enterprise_id=p.enterprise_id AND a.item_code=p.item_code AND ($%d::bigint IS NULL OR m.code=$%d) AND (cardinality($%d::text[])=0 OR c.code=ANY($%d::text[])))`, maskArg, maskArg, codesArg, codesArg)
	}
	if f.OperationID != nil {
		add(` AND p.operation_id = $%d`, *f.OperationID)
	}
	if f.Mask != nil {
		add(` AND p.mask = $%d`, *f.Mask)
	}
	if f.Preferred != nil {
		add(` AND p.preferred = $%d`, *f.Preferred)
	}
	switch strings.ToUpper(f.PriceType) {
	case "WITH_PRICE":
		q += ` AND p.unit_price>0`
	case "WITHOUT_PRICE":
		q += ` AND p.unit_price=0`
	}
	if f.ReferenceDate != nil {
		args = append(args, *f.ReferenceDate)
		n := len(args)
		q += fmt.Sprintf(` AND (p.reference_date >= $%d OR p.reference_date=(SELECT MAX(x.reference_date) FROM third_party_service_prices x WHERE x.enterprise_id=p.enterprise_id AND x.item_code=p.item_code AND x.mask=p.mask AND x.supplier_code=p.supplier_code AND x.operation_id=p.operation_id AND x.is_active AND x.reference_date < $%d))`, n, n)
	}
	order := map[string]string{"ITEM": "p.item_code", "SUPPLIER": "p.supplier_code", "OPERATION": "p.operation_id", "DATE": "p.reference_date"}[strings.ToUpper(f.OrderBy)]
	if order == "" {
		order = "p.item_code,p.operation_id,p.supplier_code,p.reference_date DESC"
	}
	q += ` ORDER BY ` + order
	lim := f.Limit
	if lim <= 0 || lim > 500 {
		lim = 100
	}
	q += fmt.Sprintf(` LIMIT %d OFFSET %d`, lim, max(f.Offset, 0))
	rows, e := r.db.Query(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Price{}
	for rows.Next() {
		var p domain.Price
		if e = rows.Scan(priceScan(&p)...); e != nil {
			return nil, e
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (r *Repo) ResolvePrice(ctx context.Context, item int64, mask string, supplier, op int64, at time.Time, attrs map[string]string) (*domain.Price, error) {
	eid, e := tenant.ID(ctx)
	if e != nil {
		return nil, e
	}
	attrs, e = r.formulaAttributes(ctx, item, attrs)
	if e != nil {
		return nil, e
	}
	rows, e := r.db.Query(ctx, priceSelect+` WHERE p.enterprise_id=$1 AND p.item_code=$2 AND p.operation_id=$3 AND ($4::bigint=0 OR p.supplier_code=$4) AND p.is_active AND (p.mask=$5 OR p.mask='') AND p.reference_date<= $6 ORDER BY (p.mask=$5) DESC,(p.supplier_code=$4) DESC,p.preferred DESC,p.reference_date DESC,p.id`, eid, item, op, supplier, mask, at)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var selected *domain.Price
	for rows.Next() {
		var p domain.Price
		if e = rows.Scan(priceScan(&p)...); e != nil {
			return nil, e
		}
		full, e := r.GetPrice(ctx, p.ID)
		if e != nil {
			return nil, e
		}
		valid := true
		for _, rule := range full.Rules {
			got, ok := attrs[strings.ToUpper(strings.TrimSpace(rule.Characteristic))]
			if !ok || (rule.Answer != nil && got != *rule.Answer) {
				valid = false
				break
			}
		}
		if valid {
			if selected == nil {
				selected = full
				continue
			}
			sameRank := (selected.Mask == mask) == (full.Mask == mask) && (selected.SupplierCode == supplier) == (full.SupplierCode == supplier) && selected.Preferred == full.Preferred && selected.ReferenceDate.Equal(full.ReferenceDate)
			if sameRank {
				return nil, errors.New("ambiguous third-party price rules: more than one price matches with the same priority")
			}
			break
		}
	}
	if selected == nil {
		return nil, domain.ErrNotFound
	}
	if strings.TrimSpace(selected.Formula) != "" {
		vars := make(map[string]decimal.Decimal, len(attrs))
		for name, raw := range attrs {
			if parsed, parseErr := decimal.NewFromString(raw); parseErr == nil {
				vars[name] = parsed
			}
		}
		calculated, formulaErr := domain.EvaluateFormula(selected.Formula, vars)
		if formulaErr != nil {
			return nil, formulaErr
		}
		if calculated.IsNegative() {
			return nil, errors.New("formula price cannot be negative")
		}
		selected.UnitPrice = calculated
	}
	return selected, nil
}

func (r *Repo) formulaAttributes(ctx context.Context, item int64, supplied map[string]string) (map[string]string, error) {
	out := map[string]string{"ITEM_CODE": fmt.Sprint(item)}
	var pdm, weight, dimensions []byte
	if err := r.db.QueryRow(ctx, `SELECT pdm_attributes,engineering_weight,COALESCE(engineering_dimensions,'{}'::jsonb) FROM items WHERE code=$1`, item).Scan(&pdm, &weight, &dimensions); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("item not found")
		}
		return nil, err
	}
	var attributes []struct{ Name, Value string }
	if json.Unmarshal(pdm, &attributes) == nil {
		for _, a := range attributes {
			out[strings.ToUpper(strings.TrimSpace(a.Name))] = strings.TrimSpace(a.Value)
		}
	}
	var w struct {
		Gross, Net json.Number
		Unit       string
	}
	if json.Unmarshal(weight, &w) == nil {
		out["WEIGHT_GROSS"], out["WEIGHT_NET"] = w.Gross.String(), w.Net.String()
	}
	var d struct{ Length, Width, Height json.Number }
	if json.Unmarshal(dimensions, &d) == nil {
		out["LENGTH"], out["WIDTH"], out["HEIGHT"] = d.Length.String(), d.Width.String(), d.Height.String()
	}
	for k, v := range supplied {
		out[strings.ToUpper(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}
	return out, nil
}
func (r *Repo) ResolveConversionFactor(ctx context.Context, item int64, mask, from string) (*decimal.Decimal, error) {
	eid, tenantErr := tenant.ID(ctx)
	if tenantErr != nil {
		return nil, tenantErr
	}
	var stock string
	if e := r.db.QueryRow(ctx, `SELECT warehouse_unit_of_measurement::text FROM items WHERE code=$1`, item).Scan(&stock); e != nil {
		return nil, e
	}
	from = strings.ToUpper(strings.TrimSpace(from))
	if from == stock {
		v := decimal.NewFromInt(1)
		return &v, nil
	}
	var factor decimal.Decimal
	e := r.db.QueryRow(ctx, `SELECT factor FROM item_unit_conversions WHERE item_code=$1 AND (mask=$2 OR mask='') AND from_uom=$3 AND to_uom=$4 AND is_active ORDER BY (mask=$2) DESC LIMIT 1`, item, mask, from, stock).Scan(&factor)
	if e == nil {
		return &factor, nil
	}
	e = r.db.QueryRow(ctx, `SELECT factor FROM item_unit_conversions WHERE item_code=$1 AND (mask=$2 OR mask='') AND from_uom=$3 AND to_uom=$4 AND is_active ORDER BY (mask=$2) DESC LIMIT 1`, item, mask, stock, from).Scan(&factor)
	if e == nil {
		v := decimal.NewFromInt(1).Div(factor)
		return &v, nil
	}
	e = r.db.QueryRow(ctx, `SELECT factor FROM global_unit_conversions WHERE enterprise_id=$1 AND from_uom=$2 AND to_uom=$3 AND is_active`, eid, from, stock).Scan(&factor)
	if e == nil {
		return &factor, nil
	}
	e = r.db.QueryRow(ctx, `SELECT factor FROM global_unit_conversions WHERE enterprise_id=$1 AND from_uom=$2 AND to_uom=$3 AND is_active`, eid, stock, from).Scan(&factor)
	if e == nil {
		v := decimal.NewFromInt(1).Div(factor)
		return &v, nil
	}
	return nil, domain.ErrNotFound
}
func (r *Repo) History(ctx context.Context, id int64) ([]domain.History, error) {
	eid, e := tenant.ID(ctx)
	if e != nil {
		return nil, e
	}
	rows, e := r.db.Query(ctx, `SELECT id,price_id,action,reason,snapshot,changed_by,changed_at FROM third_party_service_price_history WHERE enterprise_id=$1 AND price_id=$2 ORDER BY changed_at DESC`, eid, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.History{}
	for rows.Next() {
		var v domain.History
		if e = rows.Scan(&v.ID, &v.PriceID, &v.Action, &v.Reason, &v.Snapshot, &v.ChangedBy, &v.ChangedAt); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (r *Repo) Readjust(ctx context.Context, ids []int64, pct decimal.Decimal, ref time.Time, reason string, by uuid.UUID) ([]domain.Price, error) {
	if len(ids) == 0 || ref.IsZero() || strings.TrimSpace(reason) == "" {
		return nil, errors.New("ids, reference_date and reason are required")
	}
	factor := decimal.NewFromInt(1).Add(pct.Div(decimal.NewFromInt(100)))
	if !factor.IsPositive() {
		return nil, errors.New("readjustment cannot make price negative")
	}
	eid, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	out := []domain.Price{}
	for _, id := range ids {
		p, e := r.getPrice(ctx, tx, eid, id)
		if e != nil {
			return nil, e
		}
		p.ID = 0
		p.ReferenceDate = ref
		p.UnitPrice = p.UnitPrice.Mul(factor)
		p.FreightValue = p.FreightValue.Mul(factor)
		p.CreatedBy = by
		p.Rules = append([]domain.PriceRule(nil), p.Rules...)
		v, e := r.savePriceTx(ctx, tx, eid, p, reason, "READJUST", true)
		if e != nil {
			return nil, e
		}
		out = append(out, *v)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}
func (r *Repo) CopyMove(ctx context.Context, ids []int64, supplier, operation int64, move bool, ref time.Time, reason string, by uuid.UUID) ([]domain.Price, error) {
	if len(ids) == 0 || supplier <= 0 || operation <= 0 || ref.IsZero() || strings.TrimSpace(reason) == "" {
		return nil, errors.New("ids, destination, reference_date and reason are required")
	}
	eid, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	out := []domain.Price{}
	for _, id := range ids {
		p, e := r.getPrice(ctx, tx, eid, id)
		if e != nil {
			return nil, e
		}
		p.ID = 0
		p.SupplierCode = supplier
		p.OperationID = operation
		p.ReferenceDate = ref
		p.CreatedBy = by
		v, e := r.savePriceTx(ctx, tx, eid, p, reason, map[bool]string{true: "MOVE", false: "COPY"}[move], true)
		if e != nil {
			return nil, e
		}
		out = append(out, *v)
		if move {
			if e = r.deletePriceTx(ctx, tx, eid, id, "moved: "+reason, by); e != nil {
				return nil, e
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repo) CreateOrdersForProduction(ctx context.Context, productionID int64, by uuid.UUID) ([]domain.ServiceOrder, error) {
	eid, e := tenant.ID(ctx)
	if e != nil {
		return nil, e
	}
	tx, e := r.db.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('third_party_service_orders:' || $1::bigint::text,0))`, eid); e != nil {
		return nil, e
	}
	rows, e := tx.Query(ctx, `SELECT po.item_code,po.mask,po.planned_qty,COALESCE(po.start_date,CURRENT_DATE),COALESCE(po.end_date,CURRENT_DATE),ro.id,ro.operation_id,
		COALESCE((SELECT supplier.code FROM suppliers supplier WHERE supplier.id=COALESCE(ro.supplier_id,op.supplier_id) OR supplier.code=COALESCE(ro.supplier_id,op.supplier_id) ORDER BY (supplier.id=COALESCE(ro.supplier_id,op.supplier_id)) DESC LIMIT 1),COALESCE(ro.supplier_id,op.supplier_id)),
		COALESCE(ro.service_item_code,op.service_item_code),COALESCE(ro.lead_time_days,op.lead_time_days,0),COALESCE(ro.third_party_remittance,op.third_party_remittance,'DEMAND_ITEMS') FROM production_orders po JOIN manufacturing_routes mr ON mr.id=COALESCE(po.route_id,(SELECT id FROM manufacturing_routes x WHERE x.item_code=po.item_code AND x.is_standard AND x.is_active ORDER BY id LIMIT 1)) JOIN route_operations ro ON ro.route_id=mr.id AND ro.is_active JOIN operations op ON op.id=ro.operation_id AND op.origin IN ('EXTERNA','TERCEIROS') WHERE po.id=$1 AND po.enterprise_id=$2`, productionID, eid)
	if e != nil {
		return nil, e
	}
	out := []domain.ServiceOrder{}
	for rows.Next() {
		var o domain.ServiceOrder
		var lead int
		var rem string
		if e = rows.Scan(&o.ItemCode, &o.Mask, &o.Quantity, &o.StartDate, &o.DueDate, &o.RouteOperationID, &o.OperationID, &o.SupplierCode, &o.ServiceItemCode, &lead, &rem); e != nil {
			return nil, e
		}
		o.ProductionOrderID = productionID
		o.EnterpriseID = eid
		o.CreatedBy = by
		o.UOM = "UN"
		o.Status = "FIRM"
		o.RemittanceType = map[string]string{"ORDER_ITEM": "ORDER_ITEM", "DEMAND_ITEMS": "DEMAND_ITEMS", "NONE": "NONE"}[rem]
		if o.RemittanceType == "" {
			o.RemittanceType = "DEMAND_ITEMS"
		}
		if lead > 0 {
			o.DueDate = o.StartDate.AddDate(0, 0, lead)
		}
		out = append(out, o)
	}
	if e = rows.Err(); e != nil {
		rows.Close()
		return nil, e
	}
	rows.Close()
	for i := range out {
		o := &out[i]
		e = tx.QueryRow(ctx, `INSERT INTO third_party_service_orders(code,enterprise_id,production_order_id,route_operation_id,operation_id,item_code,mask,supplier_code,service_item_code,uom,quantity,start_date,due_date,status,remittance_type,created_by) VALUES((SELECT COALESCE(MAX(code),0)+1 FROM third_party_service_orders WHERE enterprise_id=$1),$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) ON CONFLICT(enterprise_id,production_order_id,route_operation_id) DO UPDATE SET updated_at=NOW() RETURNING id,code,created_at,updated_at`, eid, productionID, o.RouteOperationID, o.OperationID, o.ItemCode, o.Mask, o.SupplierCode, o.ServiceItemCode, o.UOM, o.Quantity, o.StartDate, o.DueDate, o.Status, o.RemittanceType, by).Scan(&o.ID, &o.Code, &o.CreatedAt, &o.UpdatedAt)
		if e != nil {
			return nil, e
		}
	}
	if _, e = tx.Exec(ctx, `INSERT INTO third_party_service_order_history(enterprise_id,service_order_id,event_type,new_status,actor_id)
		SELECT $1,o.id,'CREATE',o.status,$3 FROM third_party_service_orders o WHERE o.enterprise_id=$1 AND o.production_order_id=$2
		AND NOT EXISTS(SELECT 1 FROM third_party_service_order_history h WHERE h.enterprise_id=$1 AND h.service_order_id=o.id AND h.event_type='CREATE')`, eid, productionID, by); e != nil {
		return nil, e
	}
	return out, tx.Commit(ctx)
}
func (r *Repo) LinkRequisitionToProduction(ctx context.Context, productionID, requisitionCode int64) error {
	eid, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if productionID <= 0 || requisitionCode <= 0 {
		return errors.New("production_order_id and purchase_requisition_code are required")
	}
	tag, err := r.db.Exec(ctx, `UPDATE third_party_service_orders
		SET purchase_requisition_code=$3,updated_at=NOW()
		WHERE enterprise_id=$1 AND production_order_id=$2`, eid, productionID, requisitionCode)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func orderScan(o *domain.ServiceOrder) []any {
	return []any{&o.ID, &o.Code, &o.EnterpriseID, &o.ProductionOrderID, &o.RouteOperationID, &o.OperationID, &o.ItemCode, &o.Mask, &o.SupplierCode, &o.ServiceItemCode, &o.UOM, &o.Quantity, &o.FulfilledQuantity, &o.StartDate, &o.DueDate, &o.Status, &o.PurchaseRequisitionCode, &o.PurchaseOrderCode, &o.RemittanceType, &o.Kanban, &o.Notes, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt, &o.ItemDescription, &o.SupplierName, &o.OperationName}
}

const orderSelect = `SELECT service_order.id,service_order.code,service_order.enterprise_id,service_order.production_order_id,service_order.route_operation_id,service_order.operation_id,service_order.item_code,service_order.mask,service_order.supplier_code,service_order.service_item_code,service_order.uom,service_order.quantity,service_order.fulfilled_quantity,service_order.start_date,service_order.due_date,service_order.status,service_order.purchase_requisition_code,service_order.purchase_order_code,service_order.remittance_type,service_order.kanban,COALESCE(service_order.notes,''),service_order.created_by,service_order.created_at,service_order.updated_at,
	COALESCE((SELECT i.pdm_description_technique FROM items i WHERE i.code=service_order.item_code),''),
	COALESCE((SELECT s.name FROM suppliers s WHERE s.code=service_order.supplier_code),''),
	COALESCE((SELECT o.name FROM operations o WHERE o.id=service_order.operation_id),'') FROM third_party_service_orders service_order`

func (r *Repo) ListOrders(ctx context.Context, f domain.OrderFilter) ([]domain.ServiceOrder, error) {
	eid, e := tenant.ID(ctx)
	if e != nil {
		return nil, e
	}
	q := orderSelect + ` WHERE service_order.enterprise_id=$1`
	args := []any{eid}
	add := func(c string, v any) { args = append(args, v); q += fmt.Sprintf(c, len(args)) }
	if f.PlanCode != nil {
		add(` AND EXISTS(SELECT 1 FROM production_orders po JOIN planned_orders pl ON (pl.id=po.planned_order_id OR pl.code=po.planned_order_id) AND pl.enterprise_id=service_order.enterprise_id WHERE po.id=service_order.production_order_id AND po.enterprise_id=service_order.enterprise_id AND pl.plan_code=$%d)`, *f.PlanCode)
	}
	if f.ItemFrom != nil {
		add(` AND item_code >= $%d`, *f.ItemFrom)
	}
	if f.ItemTo != nil {
		add(` AND item_code <= $%d`, *f.ItemTo)
	}
	if f.ProductionOrderID != nil {
		add(` AND production_order_id=$%d`, *f.ProductionOrderID)
	}
	if f.ServiceOrderCode != nil {
		add(` AND code=$%d`, *f.ServiceOrderCode)
	}
	if f.OperationID != nil {
		add(` AND operation_id=$%d`, *f.OperationID)
	}
	if f.SupplierCode != nil {
		add(` AND supplier_code=$%d`, *f.SupplierCode)
	}
	if f.PurchaseOrderCode != nil {
		add(` AND purchase_order_code=$%d`, *f.PurchaseOrderCode)
	}
	if len(f.ProductionOrderIDs) > 0 {
		add(` AND production_order_id=ANY($%d)`, f.ProductionOrderIDs)
	}
	if len(f.ServiceOrderCodes) > 0 {
		add(` AND code=ANY($%d)`, f.ServiceOrderCodes)
	}
	if len(f.OperationIDs) > 0 {
		add(` AND operation_id=ANY($%d)`, f.OperationIDs)
	}
	if len(f.SupplierCodes) > 0 {
		add(` AND supplier_code=ANY($%d)`, f.SupplierCodes)
	}
	if len(f.PurchaseOrderCodes) > 0 {
		add(` AND purchase_order_code=ANY($%d)`, f.PurchaseOrderCodes)
	}
	if strings.TrimSpace(f.ItemSearch) != "" {
		add(` AND EXISTS(SELECT 1 FROM items i WHERE i.code=service_order.item_code AND (i.code::text ILIKE '%%'||$%[1]d||'%%' OR i.pdm_description_technique ILIKE '%%'||$%[1]d||'%%'))`, strings.TrimSpace(f.ItemSearch))
	}
	if strings.TrimSpace(f.SupplierSearch) != "" {
		add(` AND EXISTS(SELECT 1 FROM suppliers s WHERE s.code=service_order.supplier_code AND (s.code::text ILIKE '%%'||$%[1]d||'%%' OR s.name ILIKE '%%'||$%[1]d||'%%'))`, strings.TrimSpace(f.SupplierSearch))
	}
	if f.ClassificationMaskCode != nil || len(f.ClassificationCodes) > 0 {
		args = append(args, f.ClassificationMaskCode, f.ClassificationCodes)
		maskArg, codesArg := len(args)-1, len(args)
		q += fmt.Sprintf(` AND EXISTS(SELECT 1 FROM item_classification_assignments a JOIN item_classifications c ON c.id=a.classification_id AND c.is_active JOIN item_classification_masks m ON m.id=c.mask_id AND m.is_active WHERE a.enterprise_id=service_order.enterprise_id AND a.item_code=service_order.item_code AND ($%d::bigint IS NULL OR m.code=$%d) AND (cardinality($%d::text[])=0 OR c.code=ANY($%d::text[])))`, maskArg, maskArg, codesArg, codesArg)
	}
	if f.From != nil {
		add(` AND start_date >= $%d`, *f.From)
	}
	if f.To != nil {
		add(` AND due_date <= $%d`, *f.To)
	}
	if f.EmittedFrom != nil {
		add(` AND created_at >= $%d`, *f.EmittedFrom)
	}
	if f.EmittedTo != nil {
		add(` AND created_at < $%d + INTERVAL '1 day'`, *f.EmittedTo)
	}
	if f.DeliveryFrom != nil {
		add(` AND due_date >= $%d`, *f.DeliveryFrom)
	}
	if f.DeliveryTo != nil {
		add(` AND due_date <= $%d`, *f.DeliveryTo)
	}
	if f.OnlyKanban {
		q += ` AND kanban`
	}
	switch strings.ToUpper(f.Position) {
	case "ATTENDED":
		q += ` AND fulfilled_quantity>=quantity`
	case "PENDING":
		q += ` AND fulfilled_quantity<quantity`
	}
	if len(f.Statuses) > 0 {
		add(` AND status=ANY($%d)`, f.Statuses)
	}
	lim := f.Limit
	if lim <= 0 || lim > 500 {
		lim = 100
	}
	order := map[string]string{"ITEM": "item_code,code", "SUPPLIER": "supplier_code,due_date,code", "OPERATION": "operation_id,due_date,code", "PRODUCTION_ORDER": "production_order_id,code", "SERVICE_ORDER": "code", "DUE_DATE": "due_date,code"}[strings.ToUpper(f.OrderBy)]
	if order == "" {
		order = "due_date,code"
	}
	combinedWithPlan := f.PlanCode != nil
	queryLimit, queryOffset := lim, max(f.Offset, 0)
	if combinedWithPlan {
		queryLimit, queryOffset = 500, 0
	}
	q += fmt.Sprintf(` ORDER BY %s LIMIT %d OFFSET %d`, order, queryLimit, queryOffset)
	rows, e := r.db.Query(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.ServiceOrder{}
	for rows.Next() {
		var o domain.ServiceOrder
		if e = rows.Scan(orderScan(&o)...); e != nil {
			return nil, e
		}
		out = append(out, o)
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	rows.Close()
	if combinedWithPlan {
		planned, plannedErr := r.listPlannedOrders(ctx, eid, f)
		if plannedErr != nil {
			return nil, plannedErr
		}
		out = append(out, planned...)
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].DueDate.Equal(out[j].DueDate) {
				return out[i].Code < out[j].Code
			}
			return out[i].DueDate.Before(out[j].DueDate)
		})
		start := min(max(f.Offset, 0), len(out))
		end := min(start+lim, len(out))
		out = out[start:end]
	}
	return out, nil
}

// listPlannedOrders exposes the MRP's enriched SERVICO suggestions in the same
// consultation used for firm service orders. Planned rows have ID and
// production_order_id equal to zero and carry planned_suggestion_code instead;
// they cannot receive operational movements until their production OF exists.
func (r *Repo) listPlannedOrders(ctx context.Context, eid int64, f domain.OrderFilter) ([]domain.ServiceOrder, error) {
	if f.PlanCode == nil || f.ProductionOrderID != nil || len(f.ProductionOrderIDs) > 0 || f.PurchaseOrderCode != nil || len(f.PurchaseOrderCodes) > 0 || strings.EqualFold(f.Position, "ATTENDED") {
		return nil, nil
	}
	if len(f.Statuses) > 0 {
		found := false
		for _, status := range f.Statuses {
			found = found || strings.EqualFold(status, "PLANNED")
		}
		if !found {
			return nil, nil
		}
	}
	q := `SELECT suggestion.code,COALESCE(suggestion.order_number,suggestion.code),suggestion.plan_code,
		suggestion.route_operation_id,suggestion.operation_id,suggestion.item_code,suggestion.mask,
		suggestion.supplier_code,suggestion.service_item_code,
		COALESCE((SELECT item.warehouse_unit_of_measurement::text FROM items item WHERE item.code=suggestion.item_code),'UN'),
		suggestion.quantity,COALESCE(suggestion.start_date,suggestion.need_date),suggestion.need_date,
		COALESCE(suggestion.remittance_type,'DEMAND_ITEMS'),COALESCE(suggestion.notes,''),suggestion.created_at,
		COALESCE((SELECT item.pdm_description_technique FROM items item WHERE item.code=suggestion.item_code),''),
		COALESCE((SELECT supplier.name FROM suppliers supplier WHERE supplier.code=suggestion.supplier_code),''),
		COALESCE((SELECT operation.name FROM operations operation WHERE operation.id=suggestion.operation_id),'')
		FROM mrp_planned_suggestions suggestion
		WHERE suggestion.enterprise_id=$1 AND suggestion.plan_code=$2 AND suggestion.order_type='SERVICO'`
	args := []any{eid, *f.PlanCode}
	add := func(condition string, value any) { args = append(args, value); q += fmt.Sprintf(condition, len(args)) }
	if f.ItemFrom != nil {
		add(` AND suggestion.item_code >= $%d`, *f.ItemFrom)
	}
	if f.ItemTo != nil {
		add(` AND suggestion.item_code <= $%d`, *f.ItemTo)
	}
	if f.ServiceOrderCode != nil {
		add(` AND COALESCE(suggestion.order_number,suggestion.code)=$%d`, *f.ServiceOrderCode)
	}
	if len(f.ServiceOrderCodes) > 0 {
		add(` AND COALESCE(suggestion.order_number,suggestion.code)=ANY($%d)`, f.ServiceOrderCodes)
	}
	if f.OperationID != nil {
		add(` AND suggestion.operation_id=$%d`, *f.OperationID)
	}
	if len(f.OperationIDs) > 0 {
		add(` AND suggestion.operation_id=ANY($%d)`, f.OperationIDs)
	}
	if f.SupplierCode != nil {
		add(` AND suggestion.supplier_code=$%d`, *f.SupplierCode)
	}
	if len(f.SupplierCodes) > 0 {
		add(` AND suggestion.supplier_code=ANY($%d)`, f.SupplierCodes)
	}
	if strings.TrimSpace(f.ItemSearch) != "" {
		add(` AND EXISTS(SELECT 1 FROM items item WHERE item.code=suggestion.item_code AND (item.code::text ILIKE '%%'||$%[1]d||'%%' OR item.pdm_description_technique ILIKE '%%'||$%[1]d||'%%'))`, strings.TrimSpace(f.ItemSearch))
	}
	if strings.TrimSpace(f.SupplierSearch) != "" {
		add(` AND EXISTS(SELECT 1 FROM suppliers supplier WHERE supplier.code=suggestion.supplier_code AND (supplier.code::text ILIKE '%%'||$%[1]d||'%%' OR supplier.name ILIKE '%%'||$%[1]d||'%%'))`, strings.TrimSpace(f.SupplierSearch))
	}
	if f.ClassificationMaskCode != nil || len(f.ClassificationCodes) > 0 {
		args = append(args, f.ClassificationMaskCode, f.ClassificationCodes)
		maskArg, codesArg := len(args)-1, len(args)
		q += fmt.Sprintf(` AND EXISTS(SELECT 1 FROM item_classification_assignments assignment JOIN item_classifications classification ON classification.id=assignment.classification_id AND classification.is_active JOIN item_classification_masks mask ON mask.id=classification.mask_id AND mask.is_active WHERE assignment.enterprise_id=suggestion.enterprise_id AND assignment.item_code=suggestion.item_code AND ($%d::bigint IS NULL OR mask.code=$%d) AND (cardinality($%d::text[])=0 OR classification.code=ANY($%d::text[])))`, maskArg, maskArg, codesArg, codesArg)
	}
	if f.From != nil {
		add(` AND COALESCE(suggestion.start_date,suggestion.need_date) >= $%d`, *f.From)
	}
	if f.To != nil {
		add(` AND suggestion.need_date <= $%d`, *f.To)
	}
	if f.EmittedFrom != nil {
		add(` AND suggestion.created_at >= $%d`, *f.EmittedFrom)
	}
	if f.EmittedTo != nil {
		add(` AND suggestion.created_at < $%d + INTERVAL '1 day'`, *f.EmittedTo)
	}
	if f.DeliveryFrom != nil {
		add(` AND suggestion.need_date >= $%d`, *f.DeliveryFrom)
	}
	if f.DeliveryTo != nil {
		add(` AND suggestion.need_date <= $%d`, *f.DeliveryTo)
	}
	if f.OnlyKanban {
		q += ` AND EXISTS(SELECT 1 FROM kanban_cards card WHERE card.enterprise_id=suggestion.enterprise_id AND card.item_code=suggestion.item_code)`
	}
	q += ` ORDER BY suggestion.need_date,COALESCE(suggestion.order_number,suggestion.code) LIMIT 500`
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orders := []domain.ServiceOrder{}
	for rows.Next() {
		var order domain.ServiceOrder
		var suggestionCode, planCode int64
		if err = rows.Scan(&suggestionCode, &order.Code, &planCode, &order.RouteOperationID, &order.OperationID, &order.ItemCode, &order.Mask, &order.SupplierCode, &order.ServiceItemCode, &order.UOM, &order.Quantity, &order.StartDate, &order.DueDate, &order.RemittanceType, &order.Notes, &order.CreatedAt, &order.ItemDescription, &order.SupplierName, &order.OperationName); err != nil {
			return nil, err
		}
		order.PlannedSuggestionCode, order.PlanCode = &suggestionCode, &planCode
		order.EnterpriseID, order.Status, order.UpdatedAt = eid, "PLANNED", order.CreatedAt
		orders = append(orders, order)
	}
	return orders, rows.Err()
}
func (r *Repo) GetOrder(ctx context.Context, id int64) (*domain.ServiceOrder, error) {
	eid, e := tenant.ID(ctx)
	if e != nil {
		return nil, e
	}
	o := &domain.ServiceOrder{}
	e = r.db.QueryRow(ctx, orderSelect+` WHERE id=$1 AND enterprise_id=$2`, id, eid).Scan(orderScan(o)...)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return o, e
}
func (r *Repo) UpdateOrderStatus(ctx context.Context, id int64, status string, req, po *int64, by uuid.UUID) (*domain.ServiceOrder, error) {
	eid, e := tenant.ID(ctx)
	if e != nil {
		return nil, e
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	allowed := map[string]bool{"PLANNED": true, "FIRM": true, "RELEASED_WITH_PO": true, "RELEASED_WITHOUT_PO": true, "COMPLETED": true, "CANCELLED": true}
	if !allowed[status] {
		return nil, errors.New("invalid service order status")
	}
	tx, e := r.db.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	var current string
	if e = tx.QueryRow(ctx, `SELECT status FROM third_party_service_orders WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, id, eid).Scan(&current); errors.Is(e, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	} else if e != nil {
		return nil, e
	}
	transitions := map[string]map[string]bool{"PLANNED": {"FIRM": true, "CANCELLED": true}, "FIRM": {"RELEASED_WITH_PO": true, "RELEASED_WITHOUT_PO": true, "CANCELLED": true}, "RELEASED_WITH_PO": {"COMPLETED": true, "CANCELLED": true}, "RELEASED_WITHOUT_PO": {"COMPLETED": true, "CANCELLED": true}, "COMPLETED": {}, "CANCELLED": {}}
	if status != current && !transitions[current][status] {
		return nil, fmt.Errorf("invalid service order transition %s -> %s", current, status)
	}
	if status == "RELEASED_WITH_PO" && po == nil {
		return nil, errors.New("purchase_order_code is required")
	}
	tag, e := tx.Exec(ctx, `UPDATE third_party_service_orders SET status=$3,purchase_requisition_code=COALESCE($4,purchase_requisition_code),purchase_order_code=COALESCE($5,purchase_order_code),updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, id, eid, status, req, po)
	if e != nil {
		return nil, e
	}
	if tag.RowsAffected() == 0 {
		return nil, domain.ErrNotFound
	}
	if status != current {
		if _, e = tx.Exec(ctx, `INSERT INTO third_party_service_order_history(enterprise_id,service_order_id,event_type,previous_status,new_status,actor_id) VALUES($1,$2,'STATUS_CHANGE',$3,$4,$5)`, eid, id, current, status, by); e != nil {
			return nil, e
		}
	}
	o := &domain.ServiceOrder{}
	if e = tx.QueryRow(ctx, orderSelect+` WHERE id=$1 AND enterprise_id=$2`, id, eid).Scan(orderScan(o)...); e != nil {
		return nil, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	return o, nil
}
func (r *Repo) AddMovement(ctx context.Context, id int64, v domain.Movement) (*domain.Movement, error) {
	eid, e := tenant.ID(ctx)
	if e != nil {
		return nil, e
	}
	v.MovementType = strings.ToUpper(strings.TrimSpace(v.MovementType))
	if !map[string]bool{"REMITTANCE": true, "RETURN": true, "RECEIPT": true, "ADJUSTMENT": true}[v.MovementType] || !v.Quantity.IsPositive() || v.OccurredAt.IsZero() || strings.TrimSpace(v.IdempotencyKey) == "" {
		return nil, errors.New("invalid movement")
	}
	tx, e := r.db.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	var existing domain.Movement
	e = tx.QueryRow(ctx, `SELECT id,service_order_id,movement_type,quantity,occurred_at,COALESCE(reference_type,''),COALESCE(reference_code,''),COALESCE(notes,''),created_by,COALESCE(idempotency_key,''),warehouse_id,COALESCE(lot,'') FROM third_party_service_movements WHERE enterprise_id=$1 AND idempotency_key=$2`, eid, v.IdempotencyKey).Scan(movementScan(&existing)...)
	if e == nil {
		if existing.ServiceOrderID != id || existing.MovementType != v.MovementType || !existing.Quantity.Equal(v.Quantity) {
			return nil, errors.New("idempotency key already used with a different movement")
		}
		return &existing, nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return nil, e
	}
	var qty, done decimal.Decimal
	var status, remittanceType string
	e = tx.QueryRow(ctx, `SELECT quantity,fulfilled_quantity,status,remittance_type FROM third_party_service_orders WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, id, eid).Scan(&qty, &done, &status, &remittanceType)
	if e != nil {
		return nil, domain.ErrNotFound
	}
	if status == "COMPLETED" || status == "CANCELLED" {
		return nil, errors.New("terminal service order does not accept movements")
	}
	if v.MovementType != "ADJUSTMENT" && status != "RELEASED_WITH_PO" && status != "RELEASED_WITHOUT_PO" {
		return nil, errors.New("service order must be released before logistical movements")
	}
	var remitted, returned decimal.Decimal
	if e = tx.QueryRow(ctx, `SELECT
		COALESCE(SUM(quantity) FILTER(WHERE movement_type='REMITTANCE'),0),
		COALESCE(SUM(quantity) FILTER(WHERE movement_type='RETURN'),0)
		FROM third_party_service_movements WHERE enterprise_id=$1 AND service_order_id=$2`, eid, id).Scan(&remitted, &returned); e != nil {
		return nil, e
	}
	if v.MovementType == "REMITTANCE" {
		if remittanceType == "NONE" {
			return nil, errors.New("service order configured without remittance")
		}
		if remitted.Add(v.Quantity).GreaterThan(qty) {
			return nil, errors.New("remittance exceeds service order quantity")
		}
	}
	if v.MovementType == "RETURN" && returned.Add(v.Quantity).GreaterThan(remitted) {
		return nil, errors.New("return exceeds remitted quantity")
	}
	if v.MovementType == "RECEIPT" {
		done = done.Add(v.Quantity)
		if done.GreaterThan(qty) {
			return nil, errors.New("movement exceeds pending quantity")
		}
	}
	e = tx.QueryRow(ctx, `INSERT INTO third_party_service_movements(enterprise_id,service_order_id,movement_type,quantity,occurred_at,reference_type,reference_code,notes,created_by,idempotency_key,warehouse_id,lot) VALUES($1,$2,$3,$4,$5,NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),$9,$10,$11,NULLIF($12,'')) RETURNING id`, eid, id, v.MovementType, v.Quantity, v.OccurredAt, v.ReferenceType, v.ReferenceCode, v.Notes, v.CreatedBy, v.IdempotencyKey, v.WarehouseID, v.Lot).Scan(&v.ID)
	if e != nil {
		return nil, e
	}
	v.ServiceOrderID = id
	nextStatus := status
	if done.GreaterThanOrEqual(qty) {
		nextStatus = "COMPLETED"
	}
	if _, e = tx.Exec(ctx, `UPDATE third_party_service_orders SET fulfilled_quantity=$3,status=$4,updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, id, eid, done, nextStatus); e != nil {
		return nil, e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO third_party_service_order_history(enterprise_id,service_order_id,event_type,previous_status,new_status,quantity,reference_type,reference_code,actor_id,occurred_at) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,''),NULLIF($8,''),$9,$10)`, eid, id, v.MovementType, status, nextStatus, v.Quantity, v.ReferenceType, v.ReferenceCode, v.CreatedBy, v.OccurredAt); e != nil {
		return nil, e
	}
	return &v, tx.Commit(ctx)
}
func (r *Repo) ListMovements(ctx context.Context, id int64) ([]domain.Movement, error) {
	eid, e := tenant.ID(ctx)
	if e != nil {
		return nil, e
	}
	rows, e := r.db.Query(ctx, `SELECT m.id,m.service_order_id,m.movement_type,m.quantity,m.occurred_at,COALESCE(m.reference_type,''),COALESCE(m.reference_code,''),COALESCE(m.notes,''),m.created_by,COALESCE(m.idempotency_key,''),m.warehouse_id,COALESCE(m.lot,'') FROM third_party_service_movements m JOIN third_party_service_orders o ON o.id=m.service_order_id AND o.enterprise_id=$1 WHERE m.enterprise_id=$1 AND m.service_order_id=$2 ORDER BY m.occurred_at,m.id`, eid, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Movement{}
	for rows.Next() {
		var v domain.Movement
		if e = rows.Scan(movementScan(&v)...); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func movementScan(v *domain.Movement) []any {
	return []any{&v.ID, &v.ServiceOrderID, &v.MovementType, &v.Quantity, &v.OccurredAt, &v.ReferenceType, &v.ReferenceCode, &v.Notes, &v.CreatedBy, &v.IdempotencyKey, &v.WarehouseID, &v.Lot}
}

func (r *Repo) UpsertGlobalConversion(ctx context.Context, v domain.GlobalConversion) (*domain.GlobalConversion, error) {
	eid, e := tenant.ID(ctx)
	if e != nil {
		return nil, e
	}
	v.FromUOM = strings.ToUpper(strings.TrimSpace(v.FromUOM))
	v.ToUOM = strings.ToUpper(strings.TrimSpace(v.ToUOM))
	if v.FromUOM == "" || v.ToUOM == "" || v.FromUOM == v.ToUOM || !v.Factor.IsPositive() {
		return nil, errors.New("invalid global conversion")
	}
	v.EnterpriseID = eid
	e = r.db.QueryRow(ctx, `INSERT INTO global_unit_conversions(enterprise_id,from_uom,to_uom,factor,created_by) VALUES($1,$2,$3,$4,$5) ON CONFLICT(enterprise_id,from_uom,to_uom) DO UPDATE SET factor=EXCLUDED.factor,is_active=TRUE,updated_at=NOW() RETURNING id,is_active,created_at,updated_at`, eid, v.FromUOM, v.ToUOM, v.Factor, v.CreatedBy).Scan(&v.ID, &v.IsActive, &v.CreatedAt, &v.UpdatedAt)
	return &v, e
}
func (r *Repo) ListGlobalConversions(ctx context.Context) ([]domain.GlobalConversion, error) {
	eid, e := tenant.ID(ctx)
	if e != nil {
		return nil, e
	}
	rows, e := r.db.Query(ctx, `SELECT id,enterprise_id,from_uom,to_uom,factor,is_active,created_by,created_at,updated_at FROM global_unit_conversions WHERE enterprise_id=$1 AND is_active ORDER BY from_uom,to_uom`, eid)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.GlobalConversion{}
	for rows.Next() {
		var v domain.GlobalConversion
		if e = rows.Scan(&v.ID, &v.EnterpriseID, &v.FromUOM, &v.ToUOM, &v.Factor, &v.IsActive, &v.CreatedBy, &v.CreatedAt, &v.UpdatedAt); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *Repo) DeleteGlobalConversion(ctx context.Context, id int64) error {
	eid, e := tenant.ID(ctx)
	if e != nil {
		return e
	}
	tag, e := r.db.Exec(ctx, `UPDATE global_unit_conversions SET is_active=FALSE,updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, id, eid)
	if e == nil && tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return e
}
func (r *Repo) OrderHistory(ctx context.Context, id int64) ([]domain.OrderHistory, error) {
	eid, e := tenant.ID(ctx)
	if e != nil {
		return nil, e
	}
	rows, e := r.db.Query(ctx, `SELECT h.id,h.service_order_id,h.event_type,h.previous_status,h.new_status,h.quantity,COALESCE(h.reference_type,''),COALESCE(h.reference_code,''),h.actor_id,h.occurred_at FROM third_party_service_order_history h JOIN third_party_service_orders o ON o.id=h.service_order_id AND o.enterprise_id=$1 WHERE h.enterprise_id=$1 AND h.service_order_id=$2 ORDER BY h.occurred_at,h.id`, eid, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.OrderHistory{}
	for rows.Next() {
		var v domain.OrderHistory
		if e = rows.Scan(&v.ID, &v.ServiceOrderID, &v.EventType, &v.PreviousStatus, &v.NewStatus, &v.Quantity, &v.ReferenceType, &v.ReferenceCode, &v.ActorID, &v.OccurredAt); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
