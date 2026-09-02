package sales_division

import (
	"context"
	"errors"
	"fmt"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_division/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
)

func (r *SalesDivisionRepositorySQLC) Create(
	ctx context.Context,
	sd *entity.SalesDivision,
) (*entity.SalesDivision, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateSalesDivision(ctx, sqlc.CreateSalesDivisionParams{
		Code:                    sd.Code,
		Description:             sd.Description,
		CommercialAnalysis:      sqlc.SalesDivisionAnalysisEnum(sd.CommercialAnalysis),
		FinancialAnalysis:       sqlc.SalesDivisionAnalysisEnum(sd.FinancialAnalysis),
		IsTechnicalAssistance:   sd.IsTechnicalAssistance,
		ConsiderDeliveryPromise: sd.ConsiderDeliveryPromise,
		ConsiderMrp:             sd.ConsiderMRP,
		AllowOutsideLimits:      sd.AllowOutsideLimits,
		AllowFreePaymentTerms:   sd.AllowFreePaymentTerms,
		MinimumDeliveryDays:     int32(sd.MinimumDeliveryDays),
		FinancialDelayDays:      int32(sd.FinancialDelayDays),
		PisPercentage:           pgutil.ToPgNumericFromFloat64(sd.PISPercentage),
		CofinsPercentage:        pgutil.ToPgNumericFromFloat64(sd.CofinsPercentage),
		ParentDivisionID:        sd.ParentDivisionID,
		CreatedBy:               pgutil.ToPgUUID(sd.CreatedBy),
		EnterpriseID:            &enterpriseID,
	})
	if err != nil {
		return nil, fmt.Errorf("creating sales division: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *SalesDivisionRepositorySQLC) Update(
	ctx context.Context,
	sd *entity.SalesDivision,
) (*entity.SalesDivision, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateSalesDivision(ctx, sqlc.UpdateSalesDivisionParams{
		Code:                    sd.Code,
		Description:             sd.Description,
		CommercialAnalysis:      sqlc.SalesDivisionAnalysisEnum(sd.CommercialAnalysis),
		FinancialAnalysis:       sqlc.SalesDivisionAnalysisEnum(sd.FinancialAnalysis),
		IsTechnicalAssistance:   sd.IsTechnicalAssistance,
		ConsiderDeliveryPromise: sd.ConsiderDeliveryPromise,
		ConsiderMrp:             sd.ConsiderMRP,
		AllowOutsideLimits:      sd.AllowOutsideLimits,
		AllowFreePaymentTerms:   sd.AllowFreePaymentTerms,
		MinimumDeliveryDays:     int32(sd.MinimumDeliveryDays),
		FinancialDelayDays:      int32(sd.FinancialDelayDays),
		PisPercentage:           pgutil.ToPgNumericFromFloat64(sd.PISPercentage),
		CofinsPercentage:        pgutil.ToPgNumericFromFloat64(sd.CofinsPercentage),
		ParentDivisionID:        sd.ParentDivisionID,
		EnterpriseID:            &enterpriseID,
	})
	if err != nil {
		return nil, fmt.Errorf("updating sales division: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *SalesDivisionRepositorySQLC) GetByCode(
	ctx context.Context,
	code int64,
) (*entity.SalesDivision, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetSalesDivisionByCode(ctx, sqlc.GetSalesDivisionByCodeParams{Code: code, EnterpriseID: &enterpriseID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError("divisão de vendas não encontrada")
		}
		return nil, fmt.Errorf("consultar divisão de vendas: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *SalesDivisionRepositorySQLC) List(
	ctx context.Context,
) ([]*entity.SalesDivision, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListSalesDivisions(ctx, &enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listar divisões de vendas: %w", err)
	}
	return rowsToEntities(rows), nil
}

func (r *SalesDivisionRepositorySQLC) ListActive(
	ctx context.Context,
) ([]*entity.SalesDivision, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListActiveSalesDivisions(ctx, &enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listar divisões de vendas ativas: %w", err)
	}
	return rowsToEntities(rows), nil
}

func (r *SalesDivisionRepositorySQLC) Delete(
	ctx context.Context,
	code int64,
) error {
	if r.pool == nil {
		return fmt.Errorf("exclusão de divisão de vendas não configurada")
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	command, err := r.pool.Exec(ctx, `DELETE FROM sales_divisions WHERE code=$1 AND enterprise_id=$2`, code, enterpriseID)
	if err != nil {
		return fmt.Errorf("deleting sales division %d: %w", code, err)
	}
	if command.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("divisão de vendas não encontrada")
	}
	return nil
}

func (r *SalesDivisionRepositorySQLC) HasReferences(ctx context.Context, code int64) (bool, error) {
	if r.pool == nil {
		return false, fmt.Errorf("verificação de vínculos da divisão não configurada")
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var linked bool
	err = r.pool.QueryRow(ctx, `
WITH division AS (SELECT id,code FROM sales_divisions WHERE code=$1 AND enterprise_id=$2)
SELECT EXISTS (
  SELECT 1 FROM sales_divisions child JOIN division d ON child.parent_division_id=d.id
  UNION ALL SELECT 1 FROM restrictions linked JOIN division d ON linked.division_code=d.id
  UNION ALL SELECT 1 FROM sales_order_demands linked JOIN division d ON linked.division_id=d.id
  UNION ALL SELECT 1 FROM sales_orders linked JOIN division d ON linked.sales_division_code=d.id
  UNION ALL SELECT 1 FROM sales_quotation_headers linked JOIN division d ON linked.sales_division_code=d.code
)`, code, enterpriseID).Scan(&linked)
	if err != nil {
		return false, fmt.Errorf("verificar vínculos da divisão de vendas: %w", err)
	}
	return linked, nil
}

func (r *SalesDivisionRepositorySQLC) SetActive(ctx context.Context, code int64, active bool) (*entity.SalesDivision, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("alteração de situação da divisão não configurada")
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `UPDATE sales_divisions SET is_active=$1,updated_at=NOW() WHERE code=$2 AND enterprise_id=$3 RETURNING id,code,description,commercial_analysis,financial_analysis,is_technical_assistance,consider_delivery_promise,consider_mrp,allow_outside_limits,allow_free_payment_terms,minimum_delivery_days,financial_delay_days,pis_percentage,cofins_percentage,parent_division_id,is_active,created_at,updated_at,created_by`, active, code, enterpriseID)
	var sd entity.SalesDivision
	var commercial, financial string
	if err := row.Scan(&sd.ID, &sd.Code, &sd.Description, &commercial, &financial, &sd.IsTechnicalAssistance, &sd.ConsiderDeliveryPromise, &sd.ConsiderMRP, &sd.AllowOutsideLimits, &sd.AllowFreePaymentTerms, &sd.MinimumDeliveryDays, &sd.FinancialDelayDays, &sd.PISPercentage, &sd.CofinsPercentage, &sd.ParentDivisionID, &sd.IsActive, &sd.CreatedAt, &sd.UpdatedAt, &sd.CreatedBy); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError("divisão de vendas não encontrada")
		}
		return nil, fmt.Errorf("alterar situação da divisão de vendas: %w", err)
	}
	sd.CommercialAnalysis = entity.SalesDivisionAnalysis(commercial)
	sd.FinancialAnalysis = entity.SalesDivisionAnalysis(financial)
	return &sd, nil
}

func rowToEntity(row sqlc.SalesDivision) *entity.SalesDivision {
	return &entity.SalesDivision{
		ID:                      row.ID,
		Code:                    row.Code,
		Description:             row.Description,
		CommercialAnalysis:      entity.SalesDivisionAnalysis(row.CommercialAnalysis),
		FinancialAnalysis:       entity.SalesDivisionAnalysis(row.FinancialAnalysis),
		IsTechnicalAssistance:   row.IsTechnicalAssistance,
		ConsiderDeliveryPromise: row.ConsiderDeliveryPromise,
		ConsiderMRP:             row.ConsiderMrp,
		AllowOutsideLimits:      row.AllowOutsideLimits,
		AllowFreePaymentTerms:   row.AllowFreePaymentTerms,
		MinimumDeliveryDays:     int(row.MinimumDeliveryDays),
		FinancialDelayDays:      int(row.FinancialDelayDays),
		PISPercentage:           pgutil.FromPgNumericToFloat64(row.PisPercentage),
		CofinsPercentage:        pgutil.FromPgNumericToFloat64(row.CofinsPercentage),
		ParentDivisionID:        row.ParentDivisionID,
		IsActive:                row.IsActive,
		CreatedAt:               pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:               pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:               pgutil.FromPgUUID(row.CreatedBy),
	}
}

func rowsToEntities(rows []sqlc.SalesDivision) []*entity.SalesDivision {
	out := make([]*entity.SalesDivision, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out
}
