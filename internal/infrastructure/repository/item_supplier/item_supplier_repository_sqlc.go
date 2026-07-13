package item_supplier

import (
	"context"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/item_supplier/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ItemSupplierRepositorySQLC struct{ pool *pgxpool.Pool }

func New(_ *sqlc.Queries, pool *pgxpool.Pool) domainrepo.ItemSupplierRepository {
	return &ItemSupplierRepositorySQLC{pool: pool}
}

type scanner interface{ Scan(...any) error }

const cols = `s.id,s.enterprise_id,s.item_code,s.supplier_code,s.mask,s.ranking,s.supplier_item_code,s.supplier_description,s.uom,s.xml_uom,s.conversion_factor,s.package_quantity,s.is_preferred,s.supplier_uf,s.classification_id,s.classification_date,s.classification_grade,s.direct_billing,s.third_party_order,s.ignore_avg_cost_addition,s.ecommerce,s.barcode,s.notes,s.valid_until,s.lead_time_days,s.is_active,s.created_at,s.created_by,s.updated_at`
const returningCols = `id,enterprise_id,item_code,supplier_code,mask,ranking,supplier_item_code,supplier_description,uom,xml_uom,conversion_factor,package_quantity,is_preferred,supplier_uf,classification_id,classification_date,classification_grade,direct_billing,third_party_order,ignore_avg_cost_addition,ecommerce,barcode,notes,valid_until,lead_time_days,is_active,created_at,created_by,updated_at`

func scan(s scanner) (*entity.ItemPreferredSupplier, error) {
	x := &entity.ItemPreferredSupplier{}
	err := s.Scan(&x.ID, &x.EnterpriseID, &x.ItemCode, &x.SupplierCode, &x.Mask, &x.Ranking, &x.SupplierItemCode, &x.SupplierDescription, &x.UOM, &x.XMLUOM, &x.ConversionFactor, &x.PackageQuantity, &x.IsPreferred, &x.SupplierUF, &x.ClassificationID, &x.ClassificationDate, &x.ClassificationGrade, &x.DirectBilling, &x.ThirdPartyOrder, &x.IgnoreAvgCostAddition, &x.Ecommerce, &x.Barcode, &x.Notes, &x.ValidUntil, &x.LeadTimeDays, &x.IsActive, &x.CreatedAt, &x.CreatedBy, &x.UpdatedAt)
	return x, err
}
func (r *ItemSupplierRepositorySQLC) Upsert(ctx context.Context, s *entity.ItemPreferredSupplier) (*entity.ItemPreferredSupplier, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if s.SupplierUF == nil {
		_ = tx.QueryRow(ctx, `SELECT a.uf FROM suppliers p JOIN supplier_addresses a ON a.supplier_id=p.id WHERE p.code=$1 ORDER BY a.is_default DESC,a.id LIMIT 1`, s.SupplierCode).Scan(&s.SupplierUF)
	}
	if s.ClassificationID == nil && s.ClassificationDate == nil {
		_ = tx.QueryRow(ctx, `SELECT classification_id,classification_date,classification_grade FROM item_preferred_suppliers WHERE enterprise_id=$1 AND item_code=$2 AND supplier_code=$3 AND is_active LIMIT 1`, s.EnterpriseID, s.ItemCode, s.SupplierCode).Scan(&s.ClassificationID, &s.ClassificationDate, &s.ClassificationGrade)
	}
	if s.IsPreferred {
		if _, err = tx.Exec(ctx, `UPDATE item_preferred_suppliers SET is_preferred=FALSE,ranking=GREATEST(ranking,2),updated_at=NOW() WHERE enterprise_id=$1 AND item_code=$2 AND supplier_code<>$3 AND is_active`, s.EnterpriseID, s.ItemCode, s.SupplierCode); err != nil {
			return nil, err
		}
	}
	row := tx.QueryRow(ctx, `INSERT INTO item_preferred_suppliers(enterprise_id,item_code,supplier_code,mask,ranking,supplier_item_code,supplier_description,uom,xml_uom,conversion_factor,package_quantity,is_preferred,supplier_uf,classification_id,classification_date,classification_grade,direct_billing,third_party_order,ignore_avg_cost_addition,ecommerce,barcode,notes,valid_until,lead_time_days,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25) ON CONFLICT(enterprise_id,item_code,supplier_code,mask) WHERE enterprise_id IS NOT NULL DO UPDATE SET ranking=EXCLUDED.ranking,supplier_item_code=EXCLUDED.supplier_item_code,supplier_description=EXCLUDED.supplier_description,uom=EXCLUDED.uom,xml_uom=EXCLUDED.xml_uom,conversion_factor=EXCLUDED.conversion_factor,package_quantity=EXCLUDED.package_quantity,is_preferred=EXCLUDED.is_preferred,supplier_uf=EXCLUDED.supplier_uf,classification_id=EXCLUDED.classification_id,classification_date=EXCLUDED.classification_date,classification_grade=EXCLUDED.classification_grade,direct_billing=EXCLUDED.direct_billing,third_party_order=EXCLUDED.third_party_order,ignore_avg_cost_addition=EXCLUDED.ignore_avg_cost_addition,ecommerce=EXCLUDED.ecommerce,barcode=EXCLUDED.barcode,notes=EXCLUDED.notes,valid_until=EXCLUDED.valid_until,lead_time_days=EXCLUDED.lead_time_days,is_active=TRUE,updated_at=NOW() RETURNING `+returningCols, s.EnterpriseID, s.ItemCode, s.SupplierCode, s.Mask, s.Ranking, s.SupplierItemCode, s.SupplierDescription, s.UOM, s.XMLUOM, s.ConversionFactor, s.PackageQuantity, s.IsPreferred, s.SupplierUF, s.ClassificationID, s.ClassificationDate, s.ClassificationGrade, s.DirectBilling, s.ThirdPartyOrder, s.IgnoreAvgCostAddition, s.Ecommerce, s.Barcode, s.Notes, s.ValidUntil, s.LeadTimeDays, s.CreatedBy)
	saved, err := scan(row)
	if err != nil {
		return nil, fmt.Errorf("upserting item supplier: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE item_preferred_suppliers SET classification_id=$4,classification_date=$5,classification_grade=$6,updated_at=NOW() WHERE enterprise_id=$1 AND item_code=$2 AND supplier_code=$3 AND id<>$7`, s.EnterpriseID, s.ItemCode, s.SupplierCode, s.ClassificationID, s.ClassificationDate, s.ClassificationGrade, saved.ID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return saved, nil
}
func (r *ItemSupplierRepositorySQLC) list(ctx context.Context, q string, args ...any) ([]*entity.ItemPreferredSupplier, error) {
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.ItemPreferredSupplier{}
	for rows.Next() {
		x, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *ItemSupplierRepositorySQLC) ListByItem(ctx context.Context, e, item int64) ([]*entity.ItemPreferredSupplier, error) {
	return r.list(ctx, `SELECT `+cols+` FROM item_preferred_suppliers s WHERE s.enterprise_id=$1 AND s.item_code=$2 AND s.is_active AND (s.valid_until IS NULL OR s.valid_until>=CURRENT_DATE) ORDER BY s.is_preferred DESC,s.ranking,s.supplier_code,s.mask`, e, item)
}
func (r *ItemSupplierRepositorySQLC) ListBySupplier(ctx context.Context, e, supplier int64) ([]*entity.ItemPreferredSupplier, error) {
	return r.list(ctx, `SELECT `+cols+` FROM item_preferred_suppliers s WHERE s.enterprise_id=$1 AND s.supplier_code=$2 AND s.is_active AND (s.valid_until IS NULL OR s.valid_until>=CURRENT_DATE) ORDER BY s.item_code,s.mask`, e, supplier)
}
func (r *ItemSupplierRepositorySQLC) GetPreferred(ctx context.Context, e, item int64) (*entity.ItemPreferredSupplier, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM item_preferred_suppliers s WHERE s.enterprise_id=$1 AND s.item_code=$2 AND s.is_active AND s.is_preferred AND (s.valid_until IS NULL OR s.valid_until>=CURRENT_DATE) ORDER BY s.ranking,s.supplier_code LIMIT 1`, e, item))
}
func (r *ItemSupplierRepositorySQLC) Delete(ctx context.Context, e, id int64) error {
	tag, err := r.pool.Exec(ctx, `UPDATE item_preferred_suppliers SET is_active=FALSE,updated_at=NOW() WHERE enterprise_id=$1 AND id=$2`, e, id)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
func (r *ItemSupplierRepositorySQLC) ItemAllowsConversionFactor(ctx context.Context, item int64) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM items WHERE code=$1 AND nature=0)`, item).Scan(&ok)
	return ok, err
}
func (r *ItemSupplierRepositorySQLC) CreateQualityReport(ctx context.Context, q *entity.QualityReport) (*entity.QualityReport, error) {
	x := &entity.QualityReport{}
	err := r.pool.QueryRow(ctx, `INSERT INTO item_supplier_quality_reports(enterprise_id,item_supplier_id,registered_on,status,report_file_name,report_content_type,report_content,notes,created_by) SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9 WHERE EXISTS(SELECT 1 FROM item_preferred_suppliers WHERE enterprise_id=$1 AND id=$2) RETURNING id,enterprise_id,item_supplier_id,registered_on,status,report_file_name,report_content_type,report_content,notes,created_at,created_by`, q.EnterpriseID, q.ItemSupplierID, q.RegisteredOn, q.Status, q.FileName, q.ContentType, q.Content, q.Notes, q.CreatedBy).Scan(&x.ID, &x.EnterpriseID, &x.ItemSupplierID, &x.RegisteredOn, &x.Status, &x.FileName, &x.ContentType, &x.Content, &x.Notes, &x.CreatedAt, &x.CreatedBy)
	if err != nil {
		return nil, err
	}
	return x, nil
}
func (r *ItemSupplierRepositorySQLC) ListQualityReports(ctx context.Context, e, link int64) ([]*entity.QualityReport, error) {
	rows, err := r.pool.Query(ctx, `SELECT q.id,q.enterprise_id,q.item_supplier_id,q.registered_on,q.status,q.report_file_name,q.report_content_type,q.report_content,q.notes,q.created_at,q.created_by FROM item_supplier_quality_reports q JOIN item_preferred_suppliers s ON s.id=q.item_supplier_id WHERE q.enterprise_id=$1 AND q.item_supplier_id=$2 AND s.enterprise_id=$1 ORDER BY q.registered_on DESC,q.id DESC`, e, link)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.QualityReport{}
	for rows.Next() {
		x := &entity.QualityReport{}
		if err = rows.Scan(&x.ID, &x.EnterpriseID, &x.ItemSupplierID, &x.RegisteredOn, &x.Status, &x.FileName, &x.ContentType, &x.Content, &x.Notes, &x.CreatedAt, &x.CreatedBy); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
