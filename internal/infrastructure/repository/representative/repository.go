package representative

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/representative/entity"
	reprepo "github.com/FelipePn10/panossoerp/internal/domain/representative/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateType(ctx context.Context, t *entity.RepresentativeType) (*entity.RepresentativeType, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO public.representative_types (description,is_free,ignores_direct_billing,is_active) VALUES ($1,$2,$3,$4) RETURNING code,description,is_free,ignores_direct_billing,is_active,created_at,updated_at`, t.Description, t.IsFree, t.IgnoresDirectBilling, t.IsActive)
	return scanType(row)
}

func (r *Repository) UpdateType(ctx context.Context, t *entity.RepresentativeType) (*entity.RepresentativeType, error) {
	row := r.pool.QueryRow(ctx, `UPDATE public.representative_types SET description=$2,is_free=$3,ignores_direct_billing=$4,is_active=$5,updated_at=NOW() WHERE code=$1 RETURNING code,description,is_free,ignores_direct_billing,is_active,created_at,updated_at`, t.Code, t.Description, t.IsFree, t.IgnoresDirectBilling, t.IsActive)
	return scanType(row)
}

func (r *Repository) GetType(ctx context.Context, code int64) (*entity.RepresentativeType, error) {
	row := r.pool.QueryRow(ctx, `SELECT code,description,is_free,ignores_direct_billing,is_active,created_at,updated_at FROM public.representative_types WHERE code=$1`, code)
	return scanType(row)
}

func (r *Repository) ListTypes(ctx context.Context, onlyActive bool) ([]*entity.RepresentativeType, error) {
	sqlText := `SELECT code,description,is_free,ignores_direct_billing,is_active,created_at,updated_at FROM public.representative_types WHERE TRUE`
	if onlyActive {
		sqlText += ` AND is_active=TRUE`
	}
	sqlText += ` ORDER BY code`
	rows, err := r.pool.Query(ctx, sqlText)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.RepresentativeType{}
	for rows.Next() {
		t, err := scanType(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repository) Create(ctx context.Context, rep *entity.Representative) (*entity.Representative, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO public.representatives (
is_customer,customer_code,is_supplier,supplier_code,name,trade_name,type_code,category_code,register_date,core_number,document_number,
postal_code,city,state,full_address,street,street_number,complement,district,device_quantity,is_active
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21) RETURNING `+repColumns,
		rep.IsCustomer, rep.CustomerCode, rep.IsSupplier, rep.SupplierCode, rep.Name, rep.TradeName, rep.TypeCode, rep.CategoryCode, rep.RegisterDate, rep.CoreNumber, rep.DocumentNumber,
		rep.PostalCode, rep.City, rep.State, rep.FullAddress, rep.Street, rep.StreetNumber, rep.Complement, rep.District, rep.DeviceQuantity, rep.IsActive)
	return scanRep(row)
}

func (r *Repository) Update(ctx context.Context, rep *entity.Representative) (*entity.Representative, error) {
	row := r.pool.QueryRow(ctx, `UPDATE public.representatives SET
is_customer=$2,customer_code=$3,is_supplier=$4,supplier_code=$5,name=$6,trade_name=$7,type_code=$8,category_code=$9,register_date=$10,
core_number=$11,document_number=$12,postal_code=$13,city=$14,state=$15,full_address=$16,street=$17,street_number=$18,complement=$19,
district=$20,device_quantity=$21,is_active=$22,updated_at=NOW()
WHERE code=$1 RETURNING `+repColumns,
		rep.Code, rep.IsCustomer, rep.CustomerCode, rep.IsSupplier, rep.SupplierCode, rep.Name, rep.TradeName, rep.TypeCode, rep.CategoryCode, rep.RegisterDate,
		rep.CoreNumber, rep.DocumentNumber, rep.PostalCode, rep.City, rep.State, rep.FullAddress, rep.Street, rep.StreetNumber, rep.Complement,
		rep.District, rep.DeviceQuantity, rep.IsActive)
	return scanRep(row)
}

func (r *Repository) Get(ctx context.Context, code int64) (*entity.Representative, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+repColumns+` FROM public.representatives WHERE code=$1`, code)
	rep, err := scanRep(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("representative %d not found", code)
		}
		return nil, err
	}
	_ = r.loadChildren(ctx, rep)
	return rep, nil
}

func (r *Repository) List(ctx context.Context, filter reprepo.RepresentativeFilter) ([]*entity.Representative, error) {
	sqlText, args := listSQL(filter, `SELECT DISTINCT `+prefixedRepColumns("r")+` FROM public.representatives r`)
	sqlText += orderSQL(filter.SortBy)
	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.Representative{}
	for rows.Next() {
		rep, err := scanRep(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rep)
	}
	return out, rows.Err()
}

func (r *Repository) Block(ctx context.Context, code int64, reason string) error {
	_, err := r.pool.Exec(ctx, `UPDATE public.representatives SET blocked=TRUE, block_reason=$2, updated_at=NOW() WHERE code=$1`, code, reason)
	return err
}

func (r *Repository) Unblock(ctx context.Context, code int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE public.representatives SET blocked=FALSE, block_reason=NULL, updated_at=NOW() WHERE code=$1`, code)
	return err
}

func (r *Repository) AddEnterprise(ctx context.Context, row *entity.RepresentativeEnterprise) (*entity.RepresentativeEnterprise, error) {
	scan := r.pool.QueryRow(ctx, `INSERT INTO public.representative_enterprises (representative_code,enterprise_code,enterprise_name,commission_pattern_code,commission_pct,is_default,is_active)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (representative_code,enterprise_code) DO UPDATE SET enterprise_name=$3,commission_pattern_code=$4,commission_pct=$5,is_default=$6,is_active=$7,updated_at=NOW()
RETURNING id,representative_code,enterprise_code,enterprise_name,commission_pattern_code,commission_pct,is_default,is_active,created_at,updated_at`,
		row.RepresentativeCode, row.EnterpriseCode, row.EnterpriseName, row.CommissionPatternCode, row.CommissionPct, row.IsDefault, row.IsActive)
	return scanEnterprise(scan)
}

func (r *Repository) AddAccounting(ctx context.Context, row *entity.RepresentativeAccounting) (*entity.RepresentativeAccounting, error) {
	scan := r.pool.QueryRow(ctx, `INSERT INTO public.representative_accounting (representative_code,enterprise_code,event_type,debit_account_code,debit_cost_center_code,credit_account_code,credit_cost_center_code,history_code)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (representative_code,enterprise_code,event_type) DO UPDATE SET debit_account_code=$4,debit_cost_center_code=$5,credit_account_code=$6,credit_cost_center_code=$7,history_code=$8,updated_at=NOW()
RETURNING id,representative_code,enterprise_code,event_type,debit_account_code,debit_cost_center_code,credit_account_code,credit_cost_center_code,history_code,created_at,updated_at`,
		row.RepresentativeCode, row.EnterpriseCode, row.EventType, row.DebitAccountCode, row.DebitCostCenterCode, row.CreditAccountCode, row.CreditCostCenterCode, row.HistoryCode)
	return scanAccounting(scan)
}

func (r *Repository) AddRegion(ctx context.Context, row *entity.RepresentativeRegion) (*entity.RepresentativeRegion, error) {
	scan := r.pool.QueryRow(ctx, `INSERT INTO public.representative_regions (representative_code,enterprise_code,region_code,microregion_code,is_active) VALUES ($1,$2,$3,$4,$5) RETURNING id,representative_code,enterprise_code,region_code,microregion_code,is_active,created_at`, row.RepresentativeCode, row.EnterpriseCode, row.RegionCode, row.MicroregionCode, row.IsActive)
	return scanRegion(scan)
}

func (r *Repository) AddSegment(ctx context.Context, row *entity.RepresentativeSegment) (*entity.RepresentativeSegment, error) {
	scan := r.pool.QueryRow(ctx, `INSERT INTO public.representative_segments (representative_code,enterprise_code,microregion_code,market_segment_code,is_active) VALUES ($1,$2,$3,$4,$5) RETURNING id,representative_code,enterprise_code,microregion_code,market_segment_code,is_active,created_at`, row.RepresentativeCode, row.EnterpriseCode, row.MicroregionCode, row.MarketSegmentCode, row.IsActive)
	return scanSegment(scan)
}

func (r *Repository) AddSalesPlan(ctx context.Context, row *entity.RepresentativeSalesPlan) (*entity.RepresentativeSalesPlan, error) {
	scan := r.pool.QueryRow(ctx, `INSERT INTO public.representative_sales_plans (representative_code,enterprise_code,microregion_code,sales_plan_code,is_active) VALUES ($1,$2,$3,$4,$5) RETURNING id,representative_code,enterprise_code,microregion_code,sales_plan_code,is_active,created_at`, row.RepresentativeCode, row.EnterpriseCode, row.MicroregionCode, row.SalesPlanCode, row.IsActive)
	return scanSalesPlan(scan)
}

func (r *Repository) AddInterest(ctx context.Context, row *entity.RepresentativeInterest) (*entity.RepresentativeInterest, error) {
	scan := r.pool.QueryRow(ctx, `INSERT INTO public.representative_interests (representative_code,item_classification_code,is_active) VALUES ($1,$2,$3) ON CONFLICT (representative_code,item_classification_code) DO UPDATE SET is_active=$3 RETURNING id,representative_code,item_classification_code,is_active,created_at`, row.RepresentativeCode, row.ItemClassificationCode, row.IsActive)
	return scanInterest(scan)
}

func (r *Repository) AddPhone(ctx context.Context, row *entity.RepresentativePhone) (*entity.RepresentativePhone, error) {
	scan := r.pool.QueryRow(ctx, `INSERT INTO public.representative_phones (representative_code,ddi,ddd,phone,phone_type,ranking) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id,representative_code,ddi,ddd,phone,phone_type,ranking,created_at`, row.RepresentativeCode, row.DDI, row.DDD, row.Phone, row.PhoneType, row.Ranking)
	out, err := scanPhone(scan)
	if err == nil {
		_, err = r.pool.Exec(ctx, `UPDATE public.representatives SET main_phone=$2, updated_at=NOW() WHERE code=$1 AND (main_phone IS NULL OR $3 <= COALESCE((SELECT MIN(ranking) FROM public.representative_phones WHERE representative_code=$1 AND phone = main_phone), 999999))`, row.RepresentativeCode, row.Phone, row.Ranking)
	}
	return out, err
}

func (r *Repository) AddEmail(ctx context.Context, row *entity.RepresentativeEmail) (*entity.RepresentativeEmail, error) {
	scan := r.pool.QueryRow(ctx, `INSERT INTO public.representative_emails (representative_code,email,ranking) VALUES ($1,$2,$3) RETURNING id,representative_code,email,ranking,created_at`, row.RepresentativeCode, row.Email, row.Ranking)
	out, err := scanEmail(scan)
	if err == nil {
		_, err = r.pool.Exec(ctx, `UPDATE public.representatives SET main_email=$2, updated_at=NOW() WHERE code=$1 AND (main_email IS NULL OR $3 <= COALESCE((SELECT MIN(ranking) FROM public.representative_emails WHERE representative_code=$1 AND email = main_email), 999999))`, row.RepresentativeCode, row.Email, row.Ranking)
	}
	return out, err
}

func (r *Repository) AddCorrespondenceAddress(ctx context.Context, row *entity.RepresentativeCorrespondenceAddress) (*entity.RepresentativeCorrespondenceAddress, error) {
	scan := r.pool.QueryRow(ctx, `INSERT INTO public.representative_correspondence_addresses (representative_code,postal_code,city,state,full_address,street,street_number,complement,district,is_default) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id,representative_code,postal_code,city,state,full_address,street,street_number,complement,district,is_default,created_at,updated_at`, row.RepresentativeCode, row.PostalCode, row.City, row.State, row.FullAddress, row.Street, row.StreetNumber, row.Complement, row.District, row.IsDefault)
	return scanCorrespondence(scan)
}

func (r *Repository) AddContact(ctx context.Context, row *entity.RepresentativeContact) (*entity.RepresentativeContact, error) {
	scan := r.pool.QueryRow(ctx, `INSERT INTO public.representative_contacts (representative_code,contact_type_code,name,role,phone,email,notes,is_active) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id,representative_code,contact_type_code,name,role,phone,email,notes,is_active,created_at,updated_at`, row.RepresentativeCode, row.ContactTypeCode, row.Name, row.Role, row.Phone, row.Email, row.Notes, row.IsActive)
	return scanContact(scan)
}

func (r *Repository) Report(ctx context.Context, filter reprepo.RepresentativeFilter) ([]reprepo.RepresentativeReportRow, error) {
	sqlText, args := reportSQL(filter)
	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []reprepo.RepresentativeReportRow{}
	for rows.Next() {
		var row reprepo.RepresentativeReportRow
		var tradeName, typeDesc, state, city, phone, email sql.NullString
		var typeCode, debit, credit, history sql.NullInt64
		err := rows.Scan(&row.Code, &row.Name, &tradeName, &typeCode, &typeDesc, &state, &city, &phone, &email, &row.RegionCodes, &row.IsActive, &row.CommissionPct, &debit, &credit, &history)
		if err != nil {
			return nil, err
		}
		row.TradeName = strPtr(tradeName)
		row.TypeCode = intPtr(typeCode)
		row.TypeDescription = strPtr(typeDesc)
		row.State = strPtr(state)
		row.City = strPtr(city)
		row.MainPhone = strPtr(phone)
		row.MainEmail = strPtr(email)
		if filter.WithAccounts {
			row.DebitAccountCode = intPtr(debit)
			row.CreditAccountCode = intPtr(credit)
			row.GeneratedHistoryCode = intPtr(history)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func reportSQL(filter reprepo.RepresentativeFilter) (string, []any) {
	sqlText := `SELECT r.code,r.name,r.trade_name,r.type_code,rt.description,r.state,r.city,r.main_phone,r.main_email,
COALESCE(array_agg(DISTINCT rr.region_code) FILTER (WHERE rr.region_code IS NOT NULL), '{}') AS region_codes,
r.is_active,COALESCE(MAX(re.commission_pct),0),MAX(ra.debit_account_code),MAX(ra.credit_account_code),MAX(ra.history_code)
FROM public.representatives r
LEFT JOIN public.representative_types rt ON rt.code=r.type_code
LEFT JOIN public.representative_regions rr ON rr.representative_code=r.code AND rr.is_active=TRUE
LEFT JOIN public.representative_enterprises re ON re.representative_code=r.code AND re.is_active=TRUE
LEFT JOIN public.representative_accounting ra ON ra.representative_code=r.code AND ra.event_type='GENERATED'`
	sqlText += ` WHERE TRUE`
	args := []any{}
	add := func(clause string, arg any) {
		args = append(args, arg)
		sqlText += fmt.Sprintf(" AND "+clause, len(args))
	}
	if len(filter.Codes) > 0 {
		add("r.code = ANY($%d)", filter.Codes)
	}
	if filter.Description != nil {
		add("r.name ILIKE $%d", "%"+*filter.Description+"%")
	}
	if filter.TypeCode != nil {
		add("r.type_code=$%d", *filter.TypeCode)
	}
	if filter.State != nil {
		add("r.state=$%d", strings.ToUpper(*filter.State))
	}
	if filter.RegionCode != nil {
		add("rr.region_code=$%d", *filter.RegionCode)
	}
	switch filter.ActiveStatus {
	case "INACTIVE":
		sqlText += " AND r.is_active=FALSE"
	case "ALL":
	default:
		sqlText += " AND r.is_active=TRUE"
	}
	sqlText += ` GROUP BY r.code,rt.description`
	sqlText += orderSQL(filter.SortBy)
	return sqlText, args
}

func (r *Repository) FollowUp(ctx context.Context, filter reprepo.FollowUpFilter) ([]reprepo.RepresentativeFollowUp, error) {
	where, args := followWhere(filter, "r.code")
	sqlText := `WITH q AS (
 SELECT representative_code, customer_code, COUNT(*) quotation_count, COALESCE(SUM(total_net),0) total_quoted, MAX(emission_date) last_quotation_date
 FROM public.sales_quotations WHERE representative_code IS NOT NULL` + followDateWhere(filter, &args, "emission_date") + ` GROUP BY representative_code, customer_code
), o AS (
 SELECT representative_code, customer_code, COUNT(*) order_count, COALESCE(SUM(total_net),0) total_ordered, COALESCE(SUM(total_net * commission_pct / 100),0) commission_value, MAX(emission_date) last_order_date
 FROM public.sales_orders WHERE representative_code IS NOT NULL` + followDateWhere(filter, &args, "emission_date") + ` GROUP BY representative_code, customer_code
), c AS (
 SELECT COALESCE(q.representative_code,o.representative_code) representative_code, COALESCE(q.customer_code,o.customer_code) customer_code,
 COALESCE(q.quotation_count,0) quotation_count, COALESCE(o.order_count,0) order_count,
 COALESCE(q.total_quoted,0) total_quoted, COALESCE(o.total_ordered,0) total_ordered,
 q.last_quotation_date, o.last_order_date, COALESCE(o.commission_value,0) commission_value
 FROM q FULL JOIN o ON o.representative_code=q.representative_code AND o.customer_code=q.customer_code
)
SELECT r.code,r.name,COUNT(DISTINCT c.customer_code),COALESCE(SUM(c.quotation_count),0),COALESCE(SUM(c.order_count),0),
COALESCE(SUM(c.total_quoted),0),COALESCE(SUM(c.total_ordered),0),
CASE WHEN COALESCE(SUM(c.order_count),0)=0 THEN 0 ELSE COALESCE(SUM(c.total_ordered),0)/SUM(c.order_count) END,
COALESCE(SUM(c.total_ordered),0),COALESCE(SUM(c.commission_value),0),MAX(c.last_quotation_date),MAX(c.last_order_date)
FROM public.representatives r LEFT JOIN c ON c.representative_code=r.code ` + where + ` GROUP BY r.code ORDER BY r.name`
	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []reprepo.RepresentativeFollowUp{}
	for rows.Next() {
		var row reprepo.RepresentativeFollowUp
		var lastQ, lastO sql.NullTime
		if err := rows.Scan(&row.RepresentativeCode, &row.RepresentativeName, &row.CustomerCount, &row.QuotationCount, &row.OrderCount, &row.TotalQuoted, &row.TotalOrdered, &row.AverageTicket, &row.CommissionBase, &row.CommissionValue, &lastQ, &lastO); err != nil {
			return nil, err
		}
		row.LastQuotationDate = timePtr(lastQ)
		row.LastOrderDate = timePtr(lastO)
		row.Customers, _ = r.followCustomers(ctx, row.RepresentativeCode, filter)
		out = append(out, row)
	}
	return out, rows.Err()
}

const repColumns = `code,is_customer,customer_code,is_supplier,supplier_code,name,trade_name,type_code,category_code,register_date,core_number,document_number,postal_code,city,state,full_address,street,street_number,complement,district,main_phone,main_email,device_quantity,is_active,blocked,block_reason,created_at,updated_at`

func prefixedRepColumns(alias string) string {
	cols := strings.Split(repColumns, ",")
	for i, col := range cols {
		cols[i] = alias + "." + col
	}
	return strings.Join(cols, ",")
}

type scanner interface {
	Scan(dest ...any) error
}

func scanType(row scanner) (*entity.RepresentativeType, error) {
	var t entity.RepresentativeType
	err := row.Scan(&t.Code, &t.Description, &t.IsFree, &t.IgnoresDirectBilling, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	return &t, err
}

func scanRep(row scanner) (*entity.Representative, error) {
	var rep entity.Representative
	var customerCode, supplierCode, typeCode, categoryCode sql.NullInt64
	var tradeName, core, doc, postal, city, state, full, street, number, complement, district, phone, email, blockReason sql.NullString
	err := row.Scan(&rep.Code, &rep.IsCustomer, &customerCode, &rep.IsSupplier, &supplierCode, &rep.Name, &tradeName, &typeCode, &categoryCode, &rep.RegisterDate, &core, &doc, &postal, &city, &state, &full, &street, &number, &complement, &district, &phone, &email, &rep.DeviceQuantity, &rep.IsActive, &rep.Blocked, &blockReason, &rep.CreatedAt, &rep.UpdatedAt)
	if err != nil {
		return nil, err
	}
	rep.CustomerCode = intPtr(customerCode)
	rep.SupplierCode = intPtr(supplierCode)
	rep.TradeName = strPtr(tradeName)
	rep.TypeCode = intPtr(typeCode)
	rep.CategoryCode = intPtr(categoryCode)
	rep.CoreNumber = strPtr(core)
	rep.DocumentNumber = doc.String
	rep.PostalCode = strPtr(postal)
	rep.City = strPtr(city)
	rep.State = strPtr(state)
	rep.FullAddress = strPtr(full)
	rep.Street = strPtr(street)
	rep.StreetNumber = strPtr(number)
	rep.Complement = strPtr(complement)
	rep.District = strPtr(district)
	rep.MainPhone = strPtr(phone)
	rep.MainEmail = strPtr(email)
	rep.BlockReason = strPtr(blockReason)
	return &rep, nil
}

func scanEnterprise(row scanner) (*entity.RepresentativeEnterprise, error) {
	var out entity.RepresentativeEnterprise
	var name sql.NullString
	var pattern sql.NullInt64
	err := row.Scan(&out.ID, &out.RepresentativeCode, &out.EnterpriseCode, &name, &pattern, &out.CommissionPct, &out.IsDefault, &out.IsActive, &out.CreatedAt, &out.UpdatedAt)
	out.EnterpriseName = strPtr(name)
	out.CommissionPatternCode = intPtr(pattern)
	return &out, err
}

func scanAccounting(row scanner) (*entity.RepresentativeAccounting, error) {
	var out entity.RepresentativeAccounting
	var enterprise, debit, debitCC, credit, creditCC, history sql.NullInt64
	err := row.Scan(&out.ID, &out.RepresentativeCode, &enterprise, &out.EventType, &debit, &debitCC, &credit, &creditCC, &history, &out.CreatedAt, &out.UpdatedAt)
	out.EnterpriseCode = intPtr(enterprise)
	out.DebitAccountCode = intPtr(debit)
	out.DebitCostCenterCode = intPtr(debitCC)
	out.CreditAccountCode = intPtr(credit)
	out.CreditCostCenterCode = intPtr(creditCC)
	out.HistoryCode = intPtr(history)
	return &out, err
}

func scanRegion(row scanner) (*entity.RepresentativeRegion, error) {
	var out entity.RepresentativeRegion
	var enterprise, micro sql.NullInt64
	err := row.Scan(&out.ID, &out.RepresentativeCode, &enterprise, &out.RegionCode, &micro, &out.IsActive, &out.CreatedAt)
	out.EnterpriseCode = intPtr(enterprise)
	out.MicroregionCode = intPtr(micro)
	return &out, err
}

func scanSegment(row scanner) (*entity.RepresentativeSegment, error) {
	var out entity.RepresentativeSegment
	var enterprise, micro sql.NullInt64
	err := row.Scan(&out.ID, &out.RepresentativeCode, &enterprise, &micro, &out.MarketSegmentCode, &out.IsActive, &out.CreatedAt)
	out.EnterpriseCode = intPtr(enterprise)
	out.MicroregionCode = intPtr(micro)
	return &out, err
}

func scanSalesPlan(row scanner) (*entity.RepresentativeSalesPlan, error) {
	var out entity.RepresentativeSalesPlan
	var enterprise, micro sql.NullInt64
	err := row.Scan(&out.ID, &out.RepresentativeCode, &enterprise, &micro, &out.SalesPlanCode, &out.IsActive, &out.CreatedAt)
	out.EnterpriseCode = intPtr(enterprise)
	out.MicroregionCode = intPtr(micro)
	return &out, err
}

func scanInterest(row scanner) (*entity.RepresentativeInterest, error) {
	var out entity.RepresentativeInterest
	err := row.Scan(&out.ID, &out.RepresentativeCode, &out.ItemClassificationCode, &out.IsActive, &out.CreatedAt)
	return &out, err
}

func scanPhone(row scanner) (*entity.RepresentativePhone, error) {
	var out entity.RepresentativePhone
	var ddi, ddd sql.NullString
	err := row.Scan(&out.ID, &out.RepresentativeCode, &ddi, &ddd, &out.Phone, &out.PhoneType, &out.Ranking, &out.CreatedAt)
	out.DDI = strPtr(ddi)
	out.DDD = strPtr(ddd)
	return &out, err
}

func scanEmail(row scanner) (*entity.RepresentativeEmail, error) {
	var out entity.RepresentativeEmail
	err := row.Scan(&out.ID, &out.RepresentativeCode, &out.Email, &out.Ranking, &out.CreatedAt)
	return &out, err
}

func scanCorrespondence(row scanner) (*entity.RepresentativeCorrespondenceAddress, error) {
	var out entity.RepresentativeCorrespondenceAddress
	var postal, city, state, full, street, number, complement, district sql.NullString
	err := row.Scan(&out.ID, &out.RepresentativeCode, &postal, &city, &state, &full, &street, &number, &complement, &district, &out.IsDefault, &out.CreatedAt, &out.UpdatedAt)
	out.PostalCode = strPtr(postal)
	out.City = strPtr(city)
	out.State = strPtr(state)
	out.FullAddress = strPtr(full)
	out.Street = strPtr(street)
	out.StreetNumber = strPtr(number)
	out.Complement = strPtr(complement)
	out.District = strPtr(district)
	return &out, err
}

func scanContact(row scanner) (*entity.RepresentativeContact, error) {
	var out entity.RepresentativeContact
	var contactType sql.NullInt64
	var role, phone, email, notes sql.NullString
	err := row.Scan(&out.ID, &out.RepresentativeCode, &contactType, &out.Name, &role, &phone, &email, &notes, &out.IsActive, &out.CreatedAt, &out.UpdatedAt)
	out.ContactTypeCode = intPtr(contactType)
	out.Role = strPtr(role)
	out.Phone = strPtr(phone)
	out.Email = strPtr(email)
	out.Notes = strPtr(notes)
	return &out, err
}

func (r *Repository) loadChildren(ctx context.Context, rep *entity.Representative) error {
	load := func(sqlText string, fn func(pgx.Rows) error) error {
		rows, err := r.pool.Query(ctx, sqlText, rep.Code)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			if err := fn(rows); err != nil {
				return err
			}
		}
		return rows.Err()
	}
	_ = load(`SELECT id,representative_code,enterprise_code,enterprise_name,commission_pattern_code,commission_pct,is_default,is_active,created_at,updated_at FROM public.representative_enterprises WHERE representative_code=$1 ORDER BY enterprise_code`, func(rows pgx.Rows) error {
		v, e := scanEnterprise(rows)
		rep.Enterprises = append(rep.Enterprises, v)
		return e
	})
	_ = load(`SELECT id,representative_code,enterprise_code,event_type,debit_account_code,debit_cost_center_code,credit_account_code,credit_cost_center_code,history_code,created_at,updated_at FROM public.representative_accounting WHERE representative_code=$1 ORDER BY event_type`, func(rows pgx.Rows) error {
		v, e := scanAccounting(rows)
		rep.Accounting = append(rep.Accounting, v)
		return e
	})
	_ = load(`SELECT id,representative_code,enterprise_code,region_code,microregion_code,is_active,created_at FROM public.representative_regions WHERE representative_code=$1 ORDER BY region_code`, func(rows pgx.Rows) error { v, e := scanRegion(rows); rep.Regions = append(rep.Regions, v); return e })
	_ = load(`SELECT id,representative_code,enterprise_code,microregion_code,market_segment_code,is_active,created_at FROM public.representative_segments WHERE representative_code=$1 ORDER BY market_segment_code`, func(rows pgx.Rows) error { v, e := scanSegment(rows); rep.Segments = append(rep.Segments, v); return e })
	_ = load(`SELECT id,representative_code,enterprise_code,microregion_code,sales_plan_code,is_active,created_at FROM public.representative_sales_plans WHERE representative_code=$1 ORDER BY sales_plan_code`, func(rows pgx.Rows) error {
		v, e := scanSalesPlan(rows)
		rep.SalesPlans = append(rep.SalesPlans, v)
		return e
	})
	_ = load(`SELECT id,representative_code,item_classification_code,is_active,created_at FROM public.representative_interests WHERE representative_code=$1 ORDER BY item_classification_code`, func(rows pgx.Rows) error {
		v, e := scanInterest(rows)
		rep.Interests = append(rep.Interests, v)
		return e
	})
	_ = load(`SELECT id,representative_code,ddi,ddd,phone,phone_type,ranking,created_at FROM public.representative_phones WHERE representative_code=$1 ORDER BY ranking,id`, func(rows pgx.Rows) error { v, e := scanPhone(rows); rep.Phones = append(rep.Phones, v); return e })
	_ = load(`SELECT id,representative_code,email,ranking,created_at FROM public.representative_emails WHERE representative_code=$1 ORDER BY ranking,id`, func(rows pgx.Rows) error { v, e := scanEmail(rows); rep.Emails = append(rep.Emails, v); return e })
	_ = load(`SELECT id,representative_code,postal_code,city,state,full_address,street,street_number,complement,district,is_default,created_at,updated_at FROM public.representative_correspondence_addresses WHERE representative_code=$1 ORDER BY is_default DESC,id`, func(rows pgx.Rows) error {
		v, e := scanCorrespondence(rows)
		rep.CorrespondenceAddresses = append(rep.CorrespondenceAddresses, v)
		return e
	})
	_ = load(`SELECT id,representative_code,contact_type_code,name,role,phone,email,notes,is_active,created_at,updated_at FROM public.representative_contacts WHERE representative_code=$1 ORDER BY name`, func(rows pgx.Rows) error { v, e := scanContact(rows); rep.Contacts = append(rep.Contacts, v); return e })
	return nil
}

func listSQL(filter reprepo.RepresentativeFilter, selectFrom string) (string, []any) {
	sqlText := selectFrom
	if filter.RegionCode != nil {
		sqlText += ` LEFT JOIN public.representative_regions rr_filter ON rr_filter.representative_code=r.code`
	}
	sqlText += ` WHERE TRUE`
	args := []any{}
	add := func(clause string, arg any) {
		args = append(args, arg)
		sqlText += fmt.Sprintf(" AND "+clause, len(args))
	}
	if len(filter.Codes) > 0 {
		add("r.code = ANY($%d)", filter.Codes)
	}
	if filter.Description != nil {
		add("r.name ILIKE $%d", "%"+*filter.Description+"%")
	}
	if filter.TypeCode != nil {
		add("r.type_code=$%d", *filter.TypeCode)
	}
	if filter.State != nil {
		add("r.state=$%d", strings.ToUpper(*filter.State))
	}
	if filter.RegionCode != nil {
		add("rr_filter.region_code=$%d", *filter.RegionCode)
	}
	switch filter.ActiveStatus {
	case "INACTIVE":
		sqlText += " AND r.is_active=FALSE"
	case "ALL":
	default:
		sqlText += " AND r.is_active=TRUE"
	}
	return sqlText, args
}

func orderSQL(sortBy string) string {
	switch sortBy {
	case "NAME":
		return " ORDER BY r.name"
	case "STATE":
		return " ORDER BY r.state NULLS LAST, r.name"
	case "REGION":
		return " ORDER BY (SELECT MIN(region_code) FROM public.representative_regions WHERE representative_code=r.code AND is_active=TRUE) NULLS LAST, r.name"
	default:
		return " ORDER BY r.code"
	}
}

func followWhere(filter reprepo.FollowUpFilter, repExpr string) (string, []any) {
	args := []any{}
	where := " WHERE TRUE"
	if len(filter.RepresentativeCodes) > 0 {
		args = append(args, filter.RepresentativeCodes)
		where += fmt.Sprintf(" AND %s = ANY($%d)", repExpr, len(args))
	}
	return where, args
}

func followDateWhere(filter reprepo.FollowUpFilter, args *[]any, column string) string {
	sqlText := ""
	if filter.From != nil {
		*args = append(*args, *filter.From)
		sqlText += fmt.Sprintf(" AND %s >= $%d", column, len(*args))
	}
	if filter.To != nil {
		*args = append(*args, *filter.To)
		sqlText += fmt.Sprintf(" AND %s <= $%d", column, len(*args))
	}
	if len(filter.CustomerCodes) > 0 {
		*args = append(*args, filter.CustomerCodes)
		sqlText += fmt.Sprintf(" AND customer_code = ANY($%d)", len(*args))
	}
	return sqlText
}

func (r *Repository) followCustomers(ctx context.Context, representativeCode int64, filter reprepo.FollowUpFilter) ([]reprepo.RepresentativeCustomerFollowUp, error) {
	args := []any{representativeCode}
	extra := followDateWhere(filter, &args, "emission_date")
	rows, err := r.pool.Query(ctx, `WITH q AS (
 SELECT customer_code, COUNT(*) quotation_count, COALESCE(SUM(total_net),0) total_quoted, MAX(emission_date) last_quotation_date
 FROM public.sales_quotations WHERE representative_code=$1`+extra+` GROUP BY customer_code
), o AS (
 SELECT customer_code, COUNT(*) order_count, COALESCE(SUM(total_net),0) total_ordered, MAX(emission_date) last_order_date
 FROM public.sales_orders WHERE representative_code=$1`+extra+` GROUP BY customer_code
)
SELECT COALESCE(q.customer_code,o.customer_code),COALESCE(q.quotation_count,0),COALESCE(o.order_count,0),COALESCE(q.total_quoted,0),COALESCE(o.total_ordered,0),q.last_quotation_date,o.last_order_date
FROM q FULL JOIN o ON o.customer_code=q.customer_code ORDER BY 1`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []reprepo.RepresentativeCustomerFollowUp{}
	for rows.Next() {
		var row reprepo.RepresentativeCustomerFollowUp
		var lastQ, lastO sql.NullTime
		if err := rows.Scan(&row.CustomerCode, &row.QuotationCount, &row.OrderCount, &row.TotalQuoted, &row.TotalOrdered, &lastQ, &lastO); err != nil {
			return nil, err
		}
		row.LastQuotationDate = timePtr(lastQ)
		row.LastOrderDate = timePtr(lastO)
		out = append(out, row)
	}
	return out, rows.Err()
}

func strPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func intPtr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func timePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	return &v.Time
}
