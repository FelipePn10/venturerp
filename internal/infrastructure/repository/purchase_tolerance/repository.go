package purchase_tolerance

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_tolerance/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_tolerance/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) domainrepo.Repository { return &Repo{pool: pool} }

type scanner interface{ Scan(...any) error }

const cols = `id,enterprise_id,tolerance_type,applies_to,interval_min,interval_max,tolerance_value,value_type,supplier_code,action,is_active,created_at,updated_at,created_by`

func scan(s scanner) (*entity.Tolerance, error) {
	x := &entity.Tolerance{}
	err := s.Scan(&x.ID, &x.EnterpriseID, &x.ToleranceType, &x.AppliesTo, &x.IntervalMin, &x.IntervalMax, &x.ToleranceValue, &x.ValueType, &x.SupplierCode, &x.Action, &x.IsActive, &x.CreatedAt, &x.UpdatedAt, &x.CreatedBy)
	return x, err
}
func (r *Repo) Save(ctx context.Context, x *entity.Tolerance) (*entity.Tolerance, error) {
	if x.ID == 0 {
		return scan(r.pool.QueryRow(ctx, `INSERT INTO purchase_order_tolerances(enterprise_id,tolerance_type,applies_to,interval_min,interval_max,tolerance_value,value_type,supplier_code,action,is_active,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING `+cols, x.EnterpriseID, x.ToleranceType, x.AppliesTo, x.IntervalMin, x.IntervalMax, x.ToleranceValue, x.ValueType, x.SupplierCode, x.Action, x.IsActive, x.CreatedBy))
	}
	return scan(r.pool.QueryRow(ctx, `UPDATE purchase_order_tolerances SET tolerance_type=$3,applies_to=$4,interval_min=$5,interval_max=$6,tolerance_value=$7,value_type=$8,supplier_code=$9,action=$10,is_active=$11,updated_at=NOW() WHERE enterprise_id=$1 AND id=$2 RETURNING `+cols, x.EnterpriseID, x.ID, x.ToleranceType, x.AppliesTo, x.IntervalMin, x.IntervalMax, x.ToleranceValue, x.ValueType, x.SupplierCode, x.Action, x.IsActive))
}
func (r *Repo) List(ctx context.Context, e int64, supplier *int64) ([]*entity.Tolerance, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+cols+` FROM purchase_order_tolerances WHERE enterprise_id=$1 AND ($2::bigint IS NULL OR supplier_code=$2) ORDER BY tolerance_type,applies_to,supplier_code NULLS LAST,interval_min`, e, supplier)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.Tolerance{}
	for rows.Next() {
		x, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repo) Delete(ctx context.Context, e, id int64) error {
	tag, err := r.pool.Exec(ctx, `UPDATE purchase_order_tolerances SET is_active=FALSE,updated_at=NOW() WHERE enterprise_id=$1 AND id=$2`, e, id)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
func (r *Repo) Resolve(ctx context.Context, e int64, supplier *int64, t, a string, base decimal.Decimal) (*entity.Tolerance, error) {
	x, err := scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM purchase_order_tolerances r WHERE enterprise_id=$1 AND tolerance_type=$2 AND (applies_to=$3 OR applies_to='ALL') AND is_active AND interval_min<=$4 AND (interval_max IS NULL OR interval_max>=$4) AND ((EXISTS(SELECT 1 FROM purchase_order_tolerances s WHERE s.enterprise_id=$1 AND s.tolerance_type=$2 AND (s.applies_to=$3 OR s.applies_to='ALL') AND s.is_active AND s.supplier_code=$5 AND s.interval_min<=$4 AND (s.interval_max IS NULL OR s.interval_max>=$4)) AND r.supplier_code=$5) OR (NOT EXISTS(SELECT 1 FROM purchase_order_tolerances s WHERE s.enterprise_id=$1 AND s.tolerance_type=$2 AND (s.applies_to=$3 OR s.applies_to='ALL') AND s.is_active AND s.supplier_code=$5 AND s.interval_min<=$4 AND (s.interval_max IS NULL OR s.interval_max>=$4)) AND r.supplier_code IS NULL)) ORDER BY (r.applies_to=$3) DESC,r.interval_min DESC LIMIT 1`, e, t, a, base, supplier))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return x, err
}
