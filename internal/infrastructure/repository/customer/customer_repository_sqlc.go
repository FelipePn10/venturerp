package customer

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqltypes"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func toPgInt2(v *int16) pgtype.Int2 {
	if v == nil {
		return pgtype.Int2{}
	}
	return pgtype.Int2{Int16: *v, Valid: true}
}

func fromPgInt2ToPtr(v pgtype.Int2) *int16 {
	if !v.Valid {
		return nil
	}
	return &v.Int16
}

type CustomerRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func New(q *sqlc.Queries, pool *pgxpool.Pool) domainrepo.CustomerRepository {
	return &CustomerRepositorySQLC{q: q, pool: pool}
}

// ─── Regions ─────────────────────────────────────────────────────────────────

func (r *CustomerRepositorySQLC) CreateRegion(ctx context.Context, reg *entity.Region) (*entity.Region, error) {
	row, err := r.q.CreateRegion(ctx, sqlc.CreateRegionParams{
		Code:        reg.Code,
		Description: reg.Description,
		Uf:          reg.UF,
		City:        reg.City,
		CreatedBy:   pgutil.ToPgUUID(reg.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating region: %w", err)
	}
	return regionToEntity(row), nil
}

func (r *CustomerRepositorySQLC) UpdateRegion(ctx context.Context, reg *entity.Region) (*entity.Region, error) {
	row, err := r.q.UpdateRegion(ctx, sqlc.UpdateRegionParams{
		ID:          reg.ID,
		Description: reg.Description,
		Uf:          reg.UF,
		City:        reg.City,
		IsActive:    reg.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("updating region: %w", err)
	}
	return regionToEntity(row), nil
}

func (r *CustomerRepositorySQLC) GetRegionByCode(ctx context.Context, code int64) (*entity.Region, error) {
	row, err := r.q.GetRegionByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("fetching region %d: %w", code, err)
	}
	return regionToEntity(row), nil
}

func (r *CustomerRepositorySQLC) ListRegions(ctx context.Context, onlyActive bool) ([]*entity.Region, error) {
	rows, err := r.q.ListRegions(ctx, onlyActive)
	if err != nil {
		return nil, fmt.Errorf("listing regions: %w", err)
	}
	out := make([]*entity.Region, 0, len(rows))
	for _, row := range rows {
		out = append(out, regionToEntity(row))
	}
	return out, nil
}

func (r *CustomerRepositorySQLC) NextRegionCode(ctx context.Context) (int64, error) {
	code, err := r.q.NextRegionCode(ctx)
	return int64(code), err
}

func regionToEntity(row sqlc.Region) *entity.Region {
	return &entity.Region{
		ID:          row.ID,
		Code:        row.Code,
		Description: row.Description,
		UF:          row.Uf,
		City:        row.City,
		IsActive:    row.IsActive,
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy:   pgutil.FromPgUUID(row.CreatedBy),
	}
}

// ─── Market Segments ──────────────────────────────────────────────────────────

func (r *CustomerRepositorySQLC) CreateMarketSegment(ctx context.Context, s *entity.MarketSegment) (*entity.MarketSegment, error) {
	row, err := r.q.CreateMarketSegment(ctx, sqlc.CreateMarketSegmentParams{
		Code:                  s.Code,
		Description:           s.Description,
		ParentID:              s.ParentID,
		HasPisCofinsRetention: s.HasPISCOFINSRetention,
		RetentionIndicator:    toPgInt2(s.RetentionIndicator),
	})
	if err != nil {
		return nil, fmt.Errorf("creating market segment: %w", err)
	}
	return segmentToEntity(row), nil
}

func (r *CustomerRepositorySQLC) UpdateMarketSegment(ctx context.Context, s *entity.MarketSegment) (*entity.MarketSegment, error) {
	row, err := r.q.UpdateMarketSegment(ctx, sqlc.UpdateMarketSegmentParams{
		ID:                    s.ID,
		Description:           s.Description,
		ParentID:              s.ParentID,
		HasPisCofinsRetention: s.HasPISCOFINSRetention,
		RetentionIndicator:    toPgInt2(s.RetentionIndicator),
		IsActive:              s.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("updating market segment: %w", err)
	}
	return segmentToEntity(row), nil
}

func (r *CustomerRepositorySQLC) GetMarketSegmentByCode(ctx context.Context, code int64) (*entity.MarketSegment, error) {
	row, err := r.q.GetMarketSegmentByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("fetching market segment %d: %w", code, err)
	}
	return segmentToEntity(row), nil
}

func (r *CustomerRepositorySQLC) ListMarketSegments(ctx context.Context, onlyActive bool) ([]*entity.MarketSegment, error) {
	rows, err := r.q.ListMarketSegments(ctx, onlyActive)
	if err != nil {
		return nil, fmt.Errorf("listing market segments: %w", err)
	}
	out := make([]*entity.MarketSegment, 0, len(rows))
	for _, row := range rows {
		out = append(out, segmentToEntity(row))
	}
	return out, nil
}

func (r *CustomerRepositorySQLC) NextMarketSegmentCode(ctx context.Context) (int64, error) {
	code, err := r.q.NextMarketSegmentCode(ctx)
	return int64(code), err
}

func segmentToEntity(row sqlc.MarketSegment) *entity.MarketSegment {
	return &entity.MarketSegment{
		ID:                    row.ID,
		Code:                  row.Code,
		Description:           row.Description,
		ParentID:              row.ParentID,
		HasPISCOFINSRetention: row.HasPisCofinsRetention,
		RetentionIndicator:    fromPgInt2ToPtr(row.RetentionIndicator),
		IsActive:              row.IsActive,
		CreatedAt:             pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── Customer Contact Types ───────────────────────────────────────────────────

func (r *CustomerRepositorySQLC) CreateContactType(ctx context.Context, ct *entity.CustomerContactType) (*entity.CustomerContactType, error) {
	row, err := r.q.CreateContactType(ctx, sqlc.CreateContactTypeParams{
		Code:        ct.Code,
		Description: ct.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("creating contact type: %w", err)
	}
	return contactTypeToEntity(row), nil
}

func (r *CustomerRepositorySQLC) UpdateContactType(ctx context.Context, ct *entity.CustomerContactType) (*entity.CustomerContactType, error) {
	row, err := r.q.UpdateContactType(ctx, sqlc.UpdateContactTypeParams{
		ID:          ct.ID,
		Description: ct.Description,
		IsActive:    ct.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("updating contact type: %w", err)
	}
	return contactTypeToEntity(row), nil
}

func (r *CustomerRepositorySQLC) GetContactTypeByCode(ctx context.Context, code int64) (*entity.CustomerContactType, error) {
	row, err := r.q.GetContactTypeByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("fetching contact type %d: %w", code, err)
	}
	return contactTypeToEntity(row), nil
}

func (r *CustomerRepositorySQLC) ListContactTypes(ctx context.Context, onlyActive bool) ([]*entity.CustomerContactType, error) {
	rows, err := r.q.ListContactTypes(ctx, onlyActive)
	if err != nil {
		return nil, fmt.Errorf("listing contact types: %w", err)
	}
	out := make([]*entity.CustomerContactType, 0, len(rows))
	for _, row := range rows {
		out = append(out, contactTypeToEntity(row))
	}
	return out, nil
}

func (r *CustomerRepositorySQLC) NextContactTypeCode(ctx context.Context) (int64, error) {
	code, err := r.q.NextContactTypeCode(ctx)
	return int64(code), err
}

func contactTypeToEntity(row sqlc.CustomerContactType) *entity.CustomerContactType {
	return &entity.CustomerContactType{
		ID:          row.ID,
		Code:        row.Code,
		Description: row.Description,
		IsActive:    row.IsActive,
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── Customer Types ───────────────────────────────────────────────────────────

func (r *CustomerRepositorySQLC) CreateCustomerType(ctx context.Context, ct *entity.CustomerType) (*entity.CustomerType, error) {
	row, err := r.q.CreateCustomerType(ctx, sqlc.CreateCustomerTypeParams{
		Code:         ct.Code,
		Description:  ct.Description,
		Category:     sqltypes.CustomerCategoryEnum(ct.Category),
		DeliveryDays: ct.DeliveryDays,
	})
	if err != nil {
		return nil, fmt.Errorf("creating customer type: %w", err)
	}
	return customerTypeToEntity(row), nil
}

func (r *CustomerRepositorySQLC) UpdateCustomerType(ctx context.Context, ct *entity.CustomerType) (*entity.CustomerType, error) {
	row, err := r.q.UpdateCustomerType(ctx, sqlc.UpdateCustomerTypeParams{
		ID:           ct.ID,
		Description:  ct.Description,
		Category:     sqltypes.CustomerCategoryEnum(ct.Category),
		DeliveryDays: ct.DeliveryDays,
		IsActive:     ct.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("updating customer type: %w", err)
	}
	return customerTypeToEntity(row), nil
}

func (r *CustomerRepositorySQLC) GetCustomerTypeByCode(ctx context.Context, code int64) (*entity.CustomerType, error) {
	row, err := r.q.GetCustomerTypeByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("fetching customer type %d: %w", code, err)
	}
	return customerTypeToEntity(row), nil
}

func (r *CustomerRepositorySQLC) ListCustomerTypes(ctx context.Context, onlyActive bool) ([]*entity.CustomerType, error) {
	rows, err := r.q.ListCustomerTypes(ctx, onlyActive)
	if err != nil {
		return nil, fmt.Errorf("listing customer types: %w", err)
	}
	out := make([]*entity.CustomerType, 0, len(rows))
	for _, row := range rows {
		out = append(out, customerTypeToEntity(row))
	}
	return out, nil
}

func customerTypeToEntity(row sqlc.CustomerType) *entity.CustomerType {
	return &entity.CustomerType{
		ID:           row.ID,
		Code:         row.Code,
		Description:  row.Description,
		Category:     entity.CustomerCategory(row.Category),
		DeliveryDays: row.DeliveryDays,
		IsActive:     row.IsActive,
		CreatedAt:    pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── Carriers ─────────────────────────────────────────────────────────────────

func (r *CustomerRepositorySQLC) CreateCarrier(ctx context.Context, c *entity.Carrier) (*entity.Carrier, error) {
	row, err := r.q.CreateCarrier(ctx, sqlc.CreateCarrierParams{
		Code:              c.Code,
		Description:       c.Description,
		BillingType:       sqltypes.CarrierBillingTypeEnum(c.BillingType),
		UsesCreditLimit:   c.UsesCreditLimit,
		ConsiderAvailable: c.ConsiderAvailable,
		PostponeDueDate:   c.PostponeDueDate,
		ReceiptDays:       c.ReceiptDays,
		PaymentDays:       c.PaymentDays,
	})
	if err != nil {
		return nil, fmt.Errorf("creating carrier: %w", err)
	}
	return carrierToEntity(row), nil
}

func (r *CustomerRepositorySQLC) UpdateCarrier(ctx context.Context, c *entity.Carrier) (*entity.Carrier, error) {
	row, err := r.q.UpdateCarrier(ctx, sqlc.UpdateCarrierParams{
		ID:                c.ID,
		Description:       c.Description,
		BillingType:       sqltypes.CarrierBillingTypeEnum(c.BillingType),
		UsesCreditLimit:   c.UsesCreditLimit,
		ConsiderAvailable: c.ConsiderAvailable,
		PostponeDueDate:   c.PostponeDueDate,
		ReceiptDays:       c.ReceiptDays,
		PaymentDays:       c.PaymentDays,
		IsActive:          c.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("updating carrier: %w", err)
	}
	return carrierToEntity(row), nil
}

func (r *CustomerRepositorySQLC) GetCarrierByCode(ctx context.Context, code int64) (*entity.Carrier, error) {
	row, err := r.q.GetCarrierByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("fetching carrier %d: %w", code, err)
	}
	return carrierToEntity(row), nil
}

func (r *CustomerRepositorySQLC) ListCarriers(ctx context.Context, onlyActive bool) ([]*entity.Carrier, error) {
	rows, err := r.q.ListCarriers(ctx, onlyActive)
	if err != nil {
		return nil, fmt.Errorf("listing carriers: %w", err)
	}
	out := make([]*entity.Carrier, 0, len(rows))
	for _, row := range rows {
		out = append(out, carrierToEntity(row))
	}
	return out, nil
}

func (r *CustomerRepositorySQLC) NextCarrierCode(ctx context.Context) (int64, error) {
	code, err := r.q.NextCarrierCode(ctx)
	return int64(code), err
}

func carrierToEntity(row sqlc.Carrier) *entity.Carrier {
	return &entity.Carrier{
		ID:                row.ID,
		Code:              row.Code,
		Description:       row.Description,
		BillingType:       entity.CarrierBillingType(row.BillingType),
		UsesCreditLimit:   row.UsesCreditLimit,
		ConsiderAvailable: row.ConsiderAvailable,
		PostponeDueDate:   row.PostponeDueDate,
		ReceiptDays:       row.ReceiptDays,
		PaymentDays:       row.PaymentDays,
		IsActive:          row.IsActive,
		CreatedAt:         pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── Carrier Groups ───────────────────────────────────────────────────────────

func (r *CustomerRepositorySQLC) CreateCarrierGroup(ctx context.Context, g *entity.CarrierGroup) (*entity.CarrierGroup, error) {
	row, err := r.q.CreateCarrierGroup(ctx, sqlc.CreateCarrierGroupParams{
		Code:        g.Code,
		Description: g.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("creating carrier group: %w", err)
	}
	return &entity.CarrierGroup{ID: row.ID, Code: row.Code, Description: row.Description, CreatedAt: pgutil.FromPgTimestamptz(row.CreatedAt)}, nil
}

func (r *CustomerRepositorySQLC) GetCarrierGroupByCode(ctx context.Context, code int64) (*entity.CarrierGroup, error) {
	row, err := r.q.GetCarrierGroupByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("fetching carrier group %d: %w", code, err)
	}
	return &entity.CarrierGroup{ID: row.ID, Code: row.Code, Description: row.Description, CreatedAt: pgutil.FromPgTimestamptz(row.CreatedAt)}, nil
}

func (r *CustomerRepositorySQLC) ListCarrierGroups(ctx context.Context) ([]*entity.CarrierGroup, error) {
	rows, err := r.q.ListCarrierGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing carrier groups: %w", err)
	}
	out := make([]*entity.CarrierGroup, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.CarrierGroup{ID: row.ID, Code: row.Code, Description: row.Description, CreatedAt: pgutil.FromPgTimestamptz(row.CreatedAt)})
	}
	return out, nil
}

func (r *CustomerRepositorySQLC) AddCarrierToGroup(ctx context.Context, groupID, carrierID int64) error {
	return r.q.AddCarrierToGroup(ctx, sqlc.AddCarrierToGroupParams{CarrierGroupID: groupID, CarrierID: carrierID})
}

func (r *CustomerRepositorySQLC) RemoveCarrierFromGroup(ctx context.Context, groupID, carrierID int64) error {
	return r.q.RemoveCarrierFromGroup(ctx, sqlc.RemoveCarrierFromGroupParams{CarrierGroupID: groupID, CarrierID: carrierID})
}

func (r *CustomerRepositorySQLC) NextCarrierGroupCode(ctx context.Context) (int64, error) {
	code, err := r.q.NextCarrierGroupCode(ctx)
	return int64(code), err
}

// ─── Payment Conditions ───────────────────────────────────────────────────────

func (r *CustomerRepositorySQLC) CreatePaymentCondition(ctx context.Context, pc *entity.PaymentCondition) (*entity.PaymentCondition, error) {
	row, err := r.q.CreatePaymentCondition(ctx, sqlc.CreatePaymentConditionParams{
		Code:         pc.Code,
		Description:  pc.Description,
		CarrierID:    pc.CarrierID,
		AnalysisType: sqltypes.PaymentAnalysisEnum(pc.AnalysisType),
		ParcelStart:  sqltypes.PaymentParcelStartEnum(pc.ParcelStart),
		Expenses:     pgutil.ToPgNumericFromFloat64(pc.Expenses),
		AverageTerm:  pc.AverageTerm,
		IsSpecial:    pc.IsSpecial,
		IsRevenue:    pc.IsRevenue,
		IsAtSight:    pc.IsAtSight,
	})
	if err != nil {
		return nil, fmt.Errorf("creating payment condition: %w", err)
	}
	return paymentCondToEntity(row), nil
}

func (r *CustomerRepositorySQLC) UpdatePaymentCondition(ctx context.Context, pc *entity.PaymentCondition) (*entity.PaymentCondition, error) {
	row, err := r.q.UpdatePaymentCondition(ctx, sqlc.UpdatePaymentConditionParams{
		ID:           pc.ID,
		Description:  pc.Description,
		CarrierID:    pc.CarrierID,
		AnalysisType: sqltypes.PaymentAnalysisEnum(pc.AnalysisType),
		ParcelStart:  sqltypes.PaymentParcelStartEnum(pc.ParcelStart),
		Expenses:     pgutil.ToPgNumericFromFloat64(pc.Expenses),
		AverageTerm:  pc.AverageTerm,
		IsSpecial:    pc.IsSpecial,
		IsRevenue:    pc.IsRevenue,
		IsAtSight:    pc.IsAtSight,
		IsActive:     pc.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("updating payment condition: %w", err)
	}
	return paymentCondToEntity(row), nil
}

func (r *CustomerRepositorySQLC) GetPaymentConditionByCode(ctx context.Context, code int64) (*entity.PaymentCondition, error) {
	row, err := r.q.GetPaymentConditionByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("fetching payment condition %d: %w", code, err)
	}
	return paymentCondToEntity(row), nil
}

func (r *CustomerRepositorySQLC) ListPaymentConditions(ctx context.Context, onlyActive bool) ([]*entity.PaymentCondition, error) {
	rows, err := r.q.ListPaymentConditions(ctx, onlyActive)
	if err != nil {
		return nil, fmt.Errorf("listing payment conditions: %w", err)
	}
	out := make([]*entity.PaymentCondition, 0, len(rows))
	for _, row := range rows {
		out = append(out, paymentCondToEntity(row))
	}
	return out, nil
}

func (r *CustomerRepositorySQLC) AddInstallment(ctx context.Context, inst *entity.PaymentInstallment) (*entity.PaymentInstallment, error) {
	row, err := r.q.AddInstallment(ctx, sqlc.AddInstallmentParams{
		PaymentConditionID: inst.PaymentConditionID,
		InstallmentNumber:  inst.InstallmentNumber,
		DueDays:            inst.DueDays,
		Description:        pgutil.ToPgTextFromPtr(inst.Description),
		DocumentType:       pgutil.ToPgTextFromPtr(inst.DocumentType),
		MovementType:       pgutil.ToPgTextFromPtr(inst.MovementType),
		CarrierID:          inst.CarrierID,
	})
	if err != nil {
		return nil, fmt.Errorf("adding installment: %w", err)
	}
	return installmentToEntity(row), nil
}

func (r *CustomerRepositorySQLC) ListInstallments(ctx context.Context, paymentConditionID int64) ([]*entity.PaymentInstallment, error) {
	rows, err := r.q.ListInstallments(ctx, paymentConditionID)
	if err != nil {
		return nil, fmt.Errorf("listing installments: %w", err)
	}
	out := make([]*entity.PaymentInstallment, 0, len(rows))
	for _, row := range rows {
		out = append(out, installmentToEntity(row))
	}
	return out, nil
}

func (r *CustomerRepositorySQLC) DeleteInstallment(ctx context.Context, id int64) error {
	return r.q.DeleteInstallment(ctx, id)
}

func (r *CustomerRepositorySQLC) NextPaymentConditionCode(ctx context.Context) (int64, error) {
	code, err := r.q.NextPaymentConditionCode(ctx)
	return int64(code), err
}

func paymentCondToEntity(row sqlc.PaymentCondition) *entity.PaymentCondition {
	return &entity.PaymentCondition{
		ID:           row.ID,
		Code:         row.Code,
		Description:  row.Description,
		CarrierID:    row.CarrierID,
		AnalysisType: entity.PaymentAnalysis(row.AnalysisType),
		ParcelStart:  entity.PaymentParcelStart(row.ParcelStart),
		Expenses:     pgutil.FromPgNumericToFloat64(row.Expenses),
		AverageTerm:  row.AverageTerm,
		IsSpecial:    row.IsSpecial,
		IsRevenue:    row.IsRevenue,
		IsAtSight:    row.IsAtSight,
		IsActive:     row.IsActive,
		CreatedAt:    pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func installmentToEntity(row sqlc.PaymentConditionInstallment) *entity.PaymentInstallment {
	return &entity.PaymentInstallment{
		ID:                 row.ID,
		PaymentConditionID: row.PaymentConditionID,
		InstallmentNumber:  row.InstallmentNumber,
		DueDays:            row.DueDays,
		Description:        pgutil.FromPgTextPtr(row.Description),
		DocumentType:       pgutil.FromPgTextPtr(row.DocumentType),
		MovementType:       pgutil.FromPgTextPtr(row.MovementType),
		CarrierID:          row.CarrierID,
		IsActive:           row.IsActive,
	}
}

// ─── Sales Tables ─────────────────────────────────────────────────────────────

func (r *CustomerRepositorySQLC) CreateSalesTable(ctx context.Context, st *entity.SalesTable) (*entity.SalesTable, error) {
	row, err := r.q.CreateSalesTable(ctx, sqlc.CreateSalesTableParams{
		Code:                       st.Code,
		Description:                st.Description,
		ValidityStart:              pgutil.ToPgDateFromPtr(st.ValidityStart),
		ValidityEnd:                pgutil.ToPgDateFromPtr(st.ValidityEnd),
		ToleranceMinPct:            pgutil.ToPgNumericFromFloat64(st.ToleranceMinPct),
		ToleranceMaxPct:            pgutil.ToPgNumericFromFloat64(st.ToleranceMaxPct),
		PriceFormation:             sqltypes.PriceFormationEnum(st.PriceFormation),
		DecimalPlaces:              st.DecimalPlaces,
		Composition:                sqltypes.TableCompositionEnum(st.Composition),
		TableType:                  sqltypes.TableTypeEnum(st.TableType),
		BaseDate:                   sqltypes.BaseDateEnum(st.BaseDate),
		AllowItemsBelowCent:        st.AllowItemsBelowCent,
		IcmsInterestadualPorDentro: st.ICMSInterestadualPorDentro,
		Observation:                pgutil.ToPgTextFromPtr(st.Observation),
	})
	if err != nil {
		return nil, fmt.Errorf("creating sales table: %w", err)
	}
	return salesTableToEntity(row), nil
}

func (r *CustomerRepositorySQLC) UpdateSalesTable(ctx context.Context, st *entity.SalesTable) (*entity.SalesTable, error) {
	row, err := r.q.UpdateSalesTable(ctx, sqlc.UpdateSalesTableParams{
		ID:                         st.ID,
		Description:                st.Description,
		ValidityStart:              pgutil.ToPgDateFromPtr(st.ValidityStart),
		ValidityEnd:                pgutil.ToPgDateFromPtr(st.ValidityEnd),
		ToleranceMinPct:            pgutil.ToPgNumericFromFloat64(st.ToleranceMinPct),
		ToleranceMaxPct:            pgutil.ToPgNumericFromFloat64(st.ToleranceMaxPct),
		PriceFormation:             sqltypes.PriceFormationEnum(st.PriceFormation),
		DecimalPlaces:              st.DecimalPlaces,
		IsActive:                   st.IsActive,
		Composition:                sqltypes.TableCompositionEnum(st.Composition),
		TableType:                  sqltypes.TableTypeEnum(st.TableType),
		BaseDate:                   sqltypes.BaseDateEnum(st.BaseDate),
		AllowItemsBelowCent:        st.AllowItemsBelowCent,
		IcmsInterestadualPorDentro: st.ICMSInterestadualPorDentro,
		Observation:                pgutil.ToPgTextFromPtr(st.Observation),
	})
	if err != nil {
		return nil, fmt.Errorf("updating sales table: %w", err)
	}
	return salesTableToEntity(row), nil
}

func (r *CustomerRepositorySQLC) GetSalesTableByCode(ctx context.Context, code int64) (*entity.SalesTable, error) {
	row, err := r.q.GetSalesTableByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("fetching sales table %d: %w", code, err)
	}
	return salesTableToEntity(row), nil
}

func (r *CustomerRepositorySQLC) GetSalesTableByID(ctx context.Context, id int64) (*entity.SalesTable, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, code, description, validity_start, validity_end, tolerance_min_pct,
		       tolerance_max_pct, price_formation, decimal_places, is_active,
		       created_at, composition, table_type, base_date, allow_items_below_cent,
		       icms_interestadual_por_dentro, observation
		FROM sales_tables
		WHERE id=$1`, id)
	var st entity.SalesTable
	var validityStart, validityEnd pgtype.Date
	var toleranceMin, toleranceMax pgtype.Numeric
	var formation, composition, tableType, baseDate string
	var createdAt pgtype.Timestamptz
	var observation pgtype.Text
	err := row.Scan(&st.ID, &st.Code, &st.Description, &validityStart, &validityEnd,
		&toleranceMin, &toleranceMax, &formation, &st.DecimalPlaces, &st.IsActive,
		&createdAt, &composition, &tableType, &baseDate, &st.AllowItemsBelowCent,
		&st.ICMSInterestadualPorDentro, &observation)
	if err != nil {
		return nil, fmt.Errorf("fetching sales table id %d: %w", id, err)
	}
	st.ValidityStart = pgutil.FromPgDateToPtr(validityStart)
	st.ValidityEnd = pgutil.FromPgDateToPtr(validityEnd)
	st.ToleranceMinPct = pgutil.FromPgNumericToFloat64(toleranceMin)
	st.ToleranceMaxPct = pgutil.FromPgNumericToFloat64(toleranceMax)
	st.PriceFormation = entity.PriceFormation(formation)
	st.Composition = entity.TableComposition(composition)
	st.TableType = entity.TableType(tableType)
	st.BaseDate = entity.BaseDate(baseDate)
	st.CreatedAt = pgutil.FromPgTimestamptz(createdAt)
	st.Observation = pgutil.FromPgTextPtr(observation)
	return &st, nil
}

func (r *CustomerRepositorySQLC) ListSalesTables(ctx context.Context, onlyActive bool) ([]*entity.SalesTable, error) {
	rows, err := r.q.ListSalesTables(ctx, onlyActive)
	if err != nil {
		return nil, fmt.Errorf("listing sales tables: %w", err)
	}
	out := make([]*entity.SalesTable, 0, len(rows))
	for _, row := range rows {
		out = append(out, salesTableToEntity(row))
	}
	return out, nil
}

func (r *CustomerRepositorySQLC) NextSalesTableCode(ctx context.Context) (int64, error) {
	code, err := r.q.NextSalesTableCode(ctx)
	return int64(code), err
}

func salesTableToEntity(row sqlc.SalesTable) *entity.SalesTable {
	return &entity.SalesTable{
		ID:                         row.ID,
		Code:                       row.Code,
		Description:                row.Description,
		ValidityStart:              pgutil.FromPgDateToPtr(row.ValidityStart),
		ValidityEnd:                pgutil.FromPgDateToPtr(row.ValidityEnd),
		ToleranceMinPct:            pgutil.FromPgNumericToFloat64(row.ToleranceMinPct),
		ToleranceMaxPct:            pgutil.FromPgNumericToFloat64(row.ToleranceMaxPct),
		PriceFormation:             entity.PriceFormation(row.PriceFormation),
		DecimalPlaces:              row.DecimalPlaces,
		IsActive:                   row.IsActive,
		Composition:                entity.TableComposition(row.Composition),
		TableType:                  entity.TableType(row.TableType),
		BaseDate:                   entity.BaseDate(row.BaseDate),
		AllowItemsBelowCent:        row.AllowItemsBelowCent,
		ICMSInterestadualPorDentro: row.IcmsInterestadualPorDentro,
		Observation:                pgutil.FromPgTextPtr(row.Observation),
		CreatedAt:                  pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── Sales Price Policies ────────────────────────────────────────────────────

func (r *CustomerRepositorySQLC) CreateSalesPricePolicy(ctx context.Context, p *entity.SalesPricePolicy) (*entity.SalesPricePolicy, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO sales_price_policies (
			code, description, cost_source, priority, sequence, policy_scope, policy_types,
			markup_pct, margin_pct, max_margin_pct, ideal_margin_pct, margin_step_pct,
			expenses_pct, taxes_pct, freight_pct, commission_pct, discount_pct,
			min_margin_pct, max_discount_pct, incidences_json, sales_table_id,
			validity_start, validity_end, observation
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20::jsonb,$21,$22,$23,$24)
		RETURNING id, code, description, cost_source, priority, sequence, policy_scope, policy_types,
			markup_pct, margin_pct, max_margin_pct, ideal_margin_pct, margin_step_pct,
			expenses_pct, taxes_pct, freight_pct, commission_pct, discount_pct,
			min_margin_pct, max_discount_pct, incidences_json::text, sales_table_id,
			validity_start, validity_end, is_active, observation, created_at, updated_at`,
		p.Code, p.Description, string(p.CostSource), p.Priority, p.Sequence, p.PolicyScope, p.PolicyTypes,
		p.MarkupPct, p.MarginPct, p.MaxMarginPct, p.IdealMarginPct, p.MarginStepPct,
		p.ExpensesPct, p.TaxesPct, p.FreightPct, p.CommissionPct, p.DiscountPct,
		p.MinMarginPct, p.MaxDiscountPct, p.IncidencesJSON, p.SalesTableID,
		pgutil.ToPgDateFromPtr(p.ValidityStart), pgutil.ToPgDateFromPtr(p.ValidityEnd), p.Observation,
	)
	created, err := scanSalesPricePolicy(row)
	if err != nil {
		return nil, fmt.Errorf("creating sales price policy: %w", err)
	}
	return created, nil
}

func (r *CustomerRepositorySQLC) UpdateSalesPricePolicy(ctx context.Context, p *entity.SalesPricePolicy) (*entity.SalesPricePolicy, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE sales_price_policies
		SET description=$2, cost_source=$3, priority=$4, sequence=$5, policy_scope=$6,
		    policy_types=$7, markup_pct=$8, margin_pct=$9, max_margin_pct=$10,
		    ideal_margin_pct=$11, margin_step_pct=$12, expenses_pct=$13, taxes_pct=$14,
		    freight_pct=$15, commission_pct=$16, discount_pct=$17, min_margin_pct=$18,
		    max_discount_pct=$19, incidences_json=$20::jsonb, sales_table_id=$21,
		    validity_start=$22, validity_end=$23, is_active=$24, observation=$25, updated_at=NOW()
		WHERE code=$1
		RETURNING id, code, description, cost_source, priority, sequence, policy_scope, policy_types,
			markup_pct, margin_pct, max_margin_pct, ideal_margin_pct, margin_step_pct,
			expenses_pct, taxes_pct, freight_pct, commission_pct, discount_pct,
			min_margin_pct, max_discount_pct, incidences_json::text, sales_table_id,
			validity_start, validity_end, is_active, observation, created_at, updated_at`,
		p.Code, p.Description, string(p.CostSource), p.Priority, p.Sequence, p.PolicyScope,
		p.PolicyTypes, p.MarkupPct, p.MarginPct, p.MaxMarginPct, p.IdealMarginPct,
		p.MarginStepPct, p.ExpensesPct, p.TaxesPct, p.FreightPct, p.CommissionPct,
		p.DiscountPct, p.MinMarginPct, p.MaxDiscountPct, p.IncidencesJSON, p.SalesTableID,
		pgutil.ToPgDateFromPtr(p.ValidityStart), pgutil.ToPgDateFromPtr(p.ValidityEnd),
		p.IsActive, p.Observation,
	)
	updated, err := scanSalesPricePolicy(row)
	if err != nil {
		return nil, fmt.Errorf("updating sales price policy %d: %w", p.Code, err)
	}
	return updated, nil
}

func (r *CustomerRepositorySQLC) GetSalesPricePolicyByCode(ctx context.Context, code int64) (*entity.SalesPricePolicy, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, code, description, cost_source, priority, sequence, policy_scope, policy_types,
			markup_pct, margin_pct, max_margin_pct, ideal_margin_pct, margin_step_pct,
			expenses_pct, taxes_pct, freight_pct, commission_pct, discount_pct,
			min_margin_pct, max_discount_pct, incidences_json::text, sales_table_id,
			validity_start, validity_end, is_active, observation, created_at, updated_at
		FROM sales_price_policies WHERE code=$1`, code)
	p, err := scanSalesPricePolicy(row)
	if err != nil {
		return nil, fmt.Errorf("fetching sales price policy %d: %w", code, err)
	}
	return p, nil
}

func (r *CustomerRepositorySQLC) ListSalesPricePolicies(ctx context.Context, onlyActive bool) ([]*entity.SalesPricePolicy, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, description, cost_source, priority, sequence, policy_scope, policy_types,
			markup_pct, margin_pct, max_margin_pct, ideal_margin_pct, margin_step_pct,
			expenses_pct, taxes_pct, freight_pct, commission_pct, discount_pct,
			min_margin_pct, max_discount_pct, incidences_json::text, sales_table_id,
			validity_start, validity_end, is_active, observation, created_at, updated_at
		FROM sales_price_policies
		WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
		ORDER BY code`, onlyActive)
	if err != nil {
		return nil, fmt.Errorf("listing sales price policies: %w", err)
	}
	defer rows.Close()
	out := make([]*entity.SalesPricePolicy, 0)
	for rows.Next() {
		p, err := scanSalesPricePolicy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *CustomerRepositorySQLC) NextSalesPricePolicyCode(ctx context.Context) (int64, error) {
	var code int64
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(code), 0) + 1 FROM sales_price_policies`).Scan(&code); err != nil {
		return 0, err
	}
	return code, nil
}

func scanSalesPricePolicy(row scannable) (*entity.SalesPricePolicy, error) {
	var p entity.SalesPricePolicy
	var source string
	var validityStart, validityEnd pgtype.Date
	err := row.Scan(&p.ID, &p.Code, &p.Description, &source, &p.Priority, &p.Sequence,
		&p.PolicyScope, &p.PolicyTypes, &p.MarkupPct, &p.MarginPct, &p.MaxMarginPct,
		&p.IdealMarginPct, &p.MarginStepPct, &p.ExpensesPct, &p.TaxesPct, &p.FreightPct,
		&p.CommissionPct, &p.DiscountPct, &p.MinMarginPct, &p.MaxDiscountPct,
		&p.IncidencesJSON, &p.SalesTableID, &validityStart, &validityEnd, &p.IsActive,
		&p.Observation, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.CostSource = entity.SalesCostSource(source)
	p.ValidityStart = pgutil.FromPgDateToPtr(validityStart)
	p.ValidityEnd = pgutil.FromPgDateToPtr(validityEnd)
	return &p, nil
}

// ─── Commercial Policies ─────────────────────────────────────────────────────

func (r *CustomerRepositorySQLC) CreateCommercialPolicy(ctx context.Context, p *entity.CommercialPolicy) (*entity.CommercialPolicy, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO commercial_policies (
			code, description, kind, choice_type, calc_type, percent_value, fixed_value, max_percent, max_value,
			min_gross_value, max_gross_value, min_quantity, max_quantity, priority, sequence,
			stackable, requires_approval, applies_on_net_value, allow_manual_change, allow_higher_values,
			used_in_commission, applies_to_items, subtract_commission_base, data_types_json,
			commission_discount_mode, customer_code, customer_type_id,
			market_segment_id, region_id, sales_table_id, payment_condition_id, carrier_id,
			item_code, item_mask, product_line_id, item_classification, rule_json,
			validity_start, validity_end, observation
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24::jsonb,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37::jsonb,$38,$39,$40)
		RETURNING id, code, description, kind, choice_type, calc_type, percent_value, fixed_value, max_percent, max_value,
			min_gross_value, max_gross_value, min_quantity, max_quantity, priority, sequence,
			stackable, requires_approval, applies_on_net_value, allow_manual_change, allow_higher_values,
			used_in_commission, applies_to_items, subtract_commission_base, data_types_json::text,
			commission_discount_mode, customer_code, customer_type_id,
			market_segment_id, region_id, sales_table_id, payment_condition_id, carrier_id,
			item_code, item_mask, product_line_id, item_classification, rule_json::text,
			validity_start, validity_end, is_active, observation, created_at, updated_at`,
		p.Code, p.Description, string(p.Kind), string(p.ChoiceType), string(p.CalcType), p.PercentValue, p.FixedValue, p.MaxPercent, p.MaxValue,
		p.MinGrossValue, p.MaxGrossValue, p.MinQuantity, p.MaxQuantity, p.Priority, p.Sequence,
		p.Stackable, p.RequiresApproval, p.AppliesOnNetValue, p.AllowManualChange, p.AllowHigherValues,
		p.UsedInCommission, p.AppliesToItems, p.SubtractCommissionBase, p.DataTypesJSON,
		p.CommissionDiscountMode, p.CustomerCode, p.CustomerTypeID,
		p.MarketSegmentID, p.RegionID, p.SalesTableID, p.PaymentConditionID, p.CarrierID,
		p.ItemCode, p.ItemMask, p.ProductLineID, p.ItemClassification, p.RuleJSON,
		pgutil.ToPgDateFromPtr(p.ValidityStart), pgutil.ToPgDateFromPtr(p.ValidityEnd), p.Observation)
	created, err := scanCommercialPolicy(row)
	if err != nil {
		return nil, fmt.Errorf("creating commercial policy: %w", err)
	}
	return created, nil
}

func (r *CustomerRepositorySQLC) UpdateCommercialPolicy(ctx context.Context, p *entity.CommercialPolicy) (*entity.CommercialPolicy, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE commercial_policies
		SET description=$2, kind=$3, choice_type=$4, calc_type=$5, percent_value=$6, fixed_value=$7,
			max_percent=$8, max_value=$9, min_gross_value=$10, max_gross_value=$11,
			min_quantity=$12, max_quantity=$13, priority=$14, sequence=$15,
			stackable=$16, requires_approval=$17, applies_on_net_value=$18,
			allow_manual_change=$19, allow_higher_values=$20, used_in_commission=$21,
			applies_to_items=$22, subtract_commission_base=$23, data_types_json=$24::jsonb,
			commission_discount_mode=$25, customer_code=$26, customer_type_id=$27,
			market_segment_id=$28, region_id=$29, sales_table_id=$30, payment_condition_id=$31,
			carrier_id=$32, item_code=$33, item_mask=$34, product_line_id=$35,
			item_classification=$36, rule_json=$37::jsonb, validity_start=$38,
			validity_end=$39, is_active=$40, observation=$41, updated_at=NOW()
		WHERE code=$1
		RETURNING id, code, description, kind, choice_type, calc_type, percent_value, fixed_value, max_percent, max_value,
			min_gross_value, max_gross_value, min_quantity, max_quantity, priority, sequence,
			stackable, requires_approval, applies_on_net_value, allow_manual_change, allow_higher_values,
			used_in_commission, applies_to_items, subtract_commission_base, data_types_json::text,
			commission_discount_mode, customer_code, customer_type_id,
			market_segment_id, region_id, sales_table_id, payment_condition_id, carrier_id,
			item_code, item_mask, product_line_id, item_classification, rule_json::text,
			validity_start, validity_end, is_active, observation, created_at, updated_at`,
		p.Code, p.Description, string(p.Kind), string(p.ChoiceType), string(p.CalcType), p.PercentValue, p.FixedValue,
		p.MaxPercent, p.MaxValue, p.MinGrossValue, p.MaxGrossValue, p.MinQuantity, p.MaxQuantity,
		p.Priority, p.Sequence, p.Stackable, p.RequiresApproval, p.AppliesOnNetValue,
		p.AllowManualChange, p.AllowHigherValues, p.UsedInCommission, p.AppliesToItems,
		p.SubtractCommissionBase, p.DataTypesJSON, p.CommissionDiscountMode,
		p.CustomerCode, p.CustomerTypeID, p.MarketSegmentID, p.RegionID, p.SalesTableID,
		p.PaymentConditionID, p.CarrierID, p.ItemCode, p.ItemMask, p.ProductLineID,
		p.ItemClassification, p.RuleJSON, pgutil.ToPgDateFromPtr(p.ValidityStart),
		pgutil.ToPgDateFromPtr(p.ValidityEnd), p.IsActive, p.Observation)
	updated, err := scanCommercialPolicy(row)
	if err != nil {
		return nil, fmt.Errorf("updating commercial policy %d: %w", p.Code, err)
	}
	return updated, nil
}

func (r *CustomerRepositorySQLC) GetCommercialPolicyByCode(ctx context.Context, code int64) (*entity.CommercialPolicy, error) {
	row := r.pool.QueryRow(ctx, commercialPolicySelect()+` WHERE code=$1`, code)
	p, err := scanCommercialPolicy(row)
	if err != nil {
		return nil, fmt.Errorf("fetching commercial policy %d: %w", code, err)
	}
	p.Lines, _ = r.ListCommercialPolicyLines(ctx, code)
	return p, nil
}

func (r *CustomerRepositorySQLC) ListCommercialPolicies(ctx context.Context, onlyActive bool, kind *entity.CommercialPolicyKind) ([]*entity.CommercialPolicy, error) {
	var kindArg *string
	if kind != nil {
		v := string(*kind)
		kindArg = &v
	}
	rows, err := r.pool.Query(ctx, commercialPolicySelect()+`
		WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
		  AND ($2::TEXT IS NULL OR kind = $2)
		ORDER BY priority, sequence, code`, onlyActive, kindArg)
	if err != nil {
		return nil, fmt.Errorf("listing commercial policies: %w", err)
	}
	defer rows.Close()
	out := make([]*entity.CommercialPolicy, 0)
	for rows.Next() {
		p, err := scanCommercialPolicy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, p := range out {
		p.Lines, _ = r.ListCommercialPolicyLines(ctx, p.Code)
	}
	return out, nil
}

func (r *CustomerRepositorySQLC) NextCommercialPolicyCode(ctx context.Context) (int64, error) {
	var code int64
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(code), 0) + 1 FROM commercial_policies`).Scan(&code); err != nil {
		return 0, err
	}
	return code, nil
}

func (r *CustomerRepositorySQLC) AddCommercialPolicyLine(ctx context.Context, line *entity.CommercialPolicyLine) (*entity.CommercialPolicyLine, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO commercial_policy_lines (
			policy_id, line_number, sequence_number, description, calc_type, percent_value,
			fixed_value, min_value, max_value, variables_json, validity_start, validity_end
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb,$11,$12)
		RETURNING id, policy_id, line_number, sequence_number, description, calc_type,
			percent_value, fixed_value, min_value, max_value, variables_json::text,
			validity_start, validity_end, is_active, created_at, updated_at`,
		line.PolicyID, line.LineNumber, line.SequenceNumber, line.Description,
		string(line.CalcType), line.PercentValue, line.FixedValue, line.MinValue,
		line.MaxValue, line.VariablesJSON, pgutil.ToPgDateFromPtr(line.ValidityStart),
		pgutil.ToPgDateFromPtr(line.ValidityEnd))
	created, err := scanCommercialPolicyLine(row)
	if err != nil {
		return nil, fmt.Errorf("adding commercial policy line: %w", err)
	}
	return created, nil
}

func (r *CustomerRepositorySQLC) ListCommercialPolicyLines(ctx context.Context, policyCode int64) ([]*entity.CommercialPolicyLine, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT l.id, l.policy_id, l.line_number, l.sequence_number, l.description, l.calc_type,
			l.percent_value, l.fixed_value, l.min_value, l.max_value, l.variables_json::text,
			l.validity_start, l.validity_end, l.is_active, l.created_at, l.updated_at
		FROM commercial_policy_lines l
		JOIN commercial_policies p ON p.id = l.policy_id
		WHERE p.code=$1
		ORDER BY l.line_number, l.sequence_number`, policyCode)
	if err != nil {
		return nil, fmt.Errorf("listing commercial policy lines: %w", err)
	}
	defer rows.Close()
	out := make([]*entity.CommercialPolicyLine, 0)
	for rows.Next() {
		line, err := scanCommercialPolicyLine(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, line)
	}
	return out, rows.Err()
}

func (r *CustomerRepositorySQLC) AddCommercialPolicySpecificItem(ctx context.Context, item *entity.CommercialPolicySpecificItem) (*entity.CommercialPolicySpecificItem, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO commercial_policy_specific_items (
			policy_id, item_code, item_mask, product_line_id, item_classification,
			validity_start, validity_end, block_discount, block_surcharge,
			ignore_item_policies, block_manual_change
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, policy_id, item_code, item_mask, product_line_id, item_classification,
			validity_start, validity_end, block_discount, block_surcharge,
			ignore_item_policies, block_manual_change, created_at`,
		item.PolicyID, item.ItemCode, item.ItemMask, item.ProductLineID, item.ItemClassification,
		pgutil.ToPgDateFromPtr(item.ValidityStart), pgutil.ToPgDateFromPtr(item.ValidityEnd),
		item.BlockDiscount, item.BlockSurcharge, item.IgnoreItemPolicies, item.BlockManualChange)
	created, err := scanCommercialPolicySpecificItem(row)
	if err != nil {
		return nil, fmt.Errorf("adding commercial policy specific item: %w", err)
	}
	return created, nil
}

func (r *CustomerRepositorySQLC) ListCommercialPolicySpecificItems(ctx context.Context, policyCode int64) ([]*entity.CommercialPolicySpecificItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT i.id, i.policy_id, i.item_code, i.item_mask, i.product_line_id, i.item_classification,
			i.validity_start, i.validity_end, i.block_discount, i.block_surcharge,
			i.ignore_item_policies, i.block_manual_change, i.created_at
		FROM commercial_policy_specific_items i
		JOIN commercial_policies p ON p.id = i.policy_id
		WHERE p.code=$1
		ORDER BY i.id`, policyCode)
	if err != nil {
		return nil, fmt.Errorf("listing commercial policy specific items: %w", err)
	}
	defer rows.Close()
	out := make([]*entity.CommercialPolicySpecificItem, 0)
	for rows.Next() {
		item, err := scanCommercialPolicySpecificItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func commercialPolicySelect() string {
	return `SELECT id, code, description, kind, choice_type, calc_type, percent_value, fixed_value, max_percent, max_value,
		min_gross_value, max_gross_value, min_quantity, max_quantity, priority, sequence,
		stackable, requires_approval, applies_on_net_value, allow_manual_change, allow_higher_values,
		used_in_commission, applies_to_items, subtract_commission_base, data_types_json::text,
		commission_discount_mode, customer_code, customer_type_id,
		market_segment_id, region_id, sales_table_id, payment_condition_id, carrier_id,
		item_code, item_mask, product_line_id, item_classification, rule_json::text,
		validity_start, validity_end, is_active, observation, created_at, updated_at
		FROM commercial_policies`
}

func scanCommercialPolicy(row scannable) (*entity.CommercialPolicy, error) {
	var p entity.CommercialPolicy
	var kind, choiceType, calcType string
	var validityStart, validityEnd pgtype.Date
	err := row.Scan(&p.ID, &p.Code, &p.Description, &kind, &choiceType, &calcType,
		&p.PercentValue, &p.FixedValue, &p.MaxPercent, &p.MaxValue,
		&p.MinGrossValue, &p.MaxGrossValue, &p.MinQuantity, &p.MaxQuantity,
		&p.Priority, &p.Sequence, &p.Stackable, &p.RequiresApproval,
		&p.AppliesOnNetValue, &p.AllowManualChange, &p.AllowHigherValues,
		&p.UsedInCommission, &p.AppliesToItems, &p.SubtractCommissionBase,
		&p.DataTypesJSON, &p.CommissionDiscountMode, &p.CustomerCode, &p.CustomerTypeID,
		&p.MarketSegmentID, &p.RegionID, &p.SalesTableID, &p.PaymentConditionID,
		&p.CarrierID, &p.ItemCode, &p.ItemMask, &p.ProductLineID,
		&p.ItemClassification, &p.RuleJSON, &validityStart, &validityEnd,
		&p.IsActive, &p.Observation, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.Kind = entity.CommercialPolicyKind(kind)
	p.ChoiceType = entity.CommercialPolicyChoiceType(choiceType)
	p.CalcType = entity.CommercialPolicyCalcType(calcType)
	p.ValidityStart = pgutil.FromPgDateToPtr(validityStart)
	p.ValidityEnd = pgutil.FromPgDateToPtr(validityEnd)
	return &p, nil
}

func scanCommercialPolicyLine(row scannable) (*entity.CommercialPolicyLine, error) {
	var line entity.CommercialPolicyLine
	var calcType string
	var validityStart, validityEnd pgtype.Date
	err := row.Scan(&line.ID, &line.PolicyID, &line.LineNumber, &line.SequenceNumber,
		&line.Description, &calcType, &line.PercentValue, &line.FixedValue,
		&line.MinValue, &line.MaxValue, &line.VariablesJSON, &validityStart,
		&validityEnd, &line.IsActive, &line.CreatedAt, &line.UpdatedAt)
	if err != nil {
		return nil, err
	}
	line.CalcType = entity.CommercialPolicyCalcType(calcType)
	line.ValidityStart = pgutil.FromPgDateToPtr(validityStart)
	line.ValidityEnd = pgutil.FromPgDateToPtr(validityEnd)
	return &line, nil
}

func scanCommercialPolicySpecificItem(row scannable) (*entity.CommercialPolicySpecificItem, error) {
	var item entity.CommercialPolicySpecificItem
	var validityStart, validityEnd pgtype.Date
	err := row.Scan(&item.ID, &item.PolicyID, &item.ItemCode, &item.ItemMask,
		&item.ProductLineID, &item.ItemClassification, &validityStart, &validityEnd,
		&item.BlockDiscount, &item.BlockSurcharge, &item.IgnoreItemPolicies,
		&item.BlockManualChange, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	item.ValidityStart = pgutil.FromPgDateToPtr(validityStart)
	item.ValidityEnd = pgutil.FromPgDateToPtr(validityEnd)
	return &item, nil
}

// ─── Invoice Types ────────────────────────────────────────────────────────────

func invoiceTypeImpostosNFe(it *entity.InvoiceType) sqlc.NullImpostosNfeEnum {
	if it.ImpostosNFe == nil {
		return sqlc.NullImpostosNfeEnum{}
	}
	return sqlc.NullImpostosNfeEnum{ImpostosNfeEnum: sqlc.ImpostosNfeEnum(*it.ImpostosNFe), Valid: true}
}

func (r *CustomerRepositorySQLC) CreateInvoiceType(ctx context.Context, it *entity.InvoiceType) (*entity.InvoiceType, error) {
	row, err := r.q.CreateInvoiceType(ctx, sqlc.CreateInvoiceTypeParams{
		Code:                    it.Code,
		Description:             it.Description,
		Type:                    sqltypes.InvoiceTypeEnum(it.Type),
		StockMovement:           sqltypes.InvoiceStockEnum(it.StockMovement),
		IcmsType:                sqltypes.InvoiceICMSTypeEnum(it.ICMSType),
		IcmsPct:                 pgutil.ToPgNumericFromFloat64(it.ICMSPct),
		IcmsReductionPct:        pgutil.ToPgNumericFromFloat64(it.ICMSReductionPct),
		IpiPct:                  pgutil.ToPgNumericFromFloat64(it.IPIPct),
		PisPct:                  pgutil.ToPgNumericFromFloat64(it.PISPct),
		CofinsPct:               pgutil.ToPgNumericFromFloat64(it.COFINSPct),
		IssqnPct:                pgutil.ToPgNumericFromFloat64(it.ISSQNPct),
		IrPct:                   pgutil.ToPgNumericFromFloat64(it.IRPct),
		CsllPct:                 pgutil.ToPgNumericFromFloat64(it.CSLLPct),
		InssPct:                 pgutil.ToPgNumericFromFloat64(it.INSSPct),
		GeneratesRevenue:        it.GeneratesRevenue,
		UpdatesInventory:        it.UpdatesInventory,
		GeneratesFinancialTitle: it.GeneratesFinancialTitle,
		ConsidersGoals:          it.ConsidersGoals,
		CalcSubstitutionTax:     it.CalcSubstitutionTax,
		CalcIcmsDeferral:        it.CalcICMSDeferral,
		CalcPisCofins:           it.CalcPISCOFINS,
		CalcDifal:               it.CalcDIFAL,
		RequiresSalesOrder:      it.RequiresSalesOrder,
		ListsFiscalBooks:        it.ListsFiscalBooks,
		ModelNf:                 pgutil.ToPgTextFromPtr(it.ModelNF),
		CstIcms:                 pgutil.ToPgTextFromPtr(it.CSTICMS),
		CsosnIcms:               pgutil.ToPgTextFromPtr(it.CSOSNTICMS),
		CstIpi:                  pgutil.ToPgTextFromPtr(it.CSTIPI),
		CstPis:                  pgutil.ToPgTextFromPtr(it.CSTPIS),
		CstCofins:               pgutil.ToPgTextFromPtr(it.CSTCOFINS),
		BaixaPedido:             it.BaixaPedido,
		GeraTituloDev:           it.GeraTituloDev,
		ExigeSuframa:            it.ExigeSuframa,
		IrPctPresumption:        pgutil.ToPgNumericFromFloat64(it.IRPctPresumption),
		CsllPctPresumption:      pgutil.ToPgNumericFromFloat64(it.CSLLPctPresumption),
		// Extended fields (migration 000126)
		DescriptionNf:            pgutil.ToPgTextFromPtr(it.DescriptionNF),
		ImpostosNfe:              invoiceTypeImpostosNFe(it),
		CfopID:                   it.CFOPId,
		DispositivoLegalIpiID:    it.DispositivoLegalIPIId,
		DispositivoLegalIcmsID:   it.DispositivoLegalICMSId,
		DispositivoLegalIcmsStID: it.DispositivoLegalICMSSTId,
		DispositivoLegalPisID:    it.DispositivoLegalPISId,
		DispositivoLegalCofinsID: it.DispositivoLegalCOFINSId,
		HierarchyIpi:             pgutil.ToPgTextFromPtr(it.HierarchyIPI),
		HierarchyIcms:            pgutil.ToPgTextFromPtr(it.HierarchyICMS),
		HierarchyIcmsSt:          pgutil.ToPgTextFromPtr(it.HierarchyICMSST),
		HierarchyPis:             pgutil.ToPgTextFromPtr(it.HierarchyPIS),
		HierarchyCofins:          pgutil.ToPgTextFromPtr(it.HierarchyCOFINS),
		IpiTransferSalesTableID:  it.IPITransferSalesTableId,
		ListaValorContabil:       it.ListaValorContabil,
		ListaRegistroSaida:       it.ListaRegistroSaida,
		ListaIcmsIpi:             it.ListaICMSIPI,
		SintegraSpedFiscal:       it.SintegraSpedFiscal,
		CalcFomentar:             it.CalcFomentar,
		ExcecaoFomentar:          it.ExcecaoFomentar,
		CompRessRetSt:            it.CompRessRetST,
		CalcReducao:              it.CalcReducao,
		ComplementoItens:         it.ComplementoItens,
		BuscaTipoNf:              it.BuscaTipoNF,
		IcmsStUltEntrada:         it.ICMSSTUltEntrada,
		SomenteConsultaLotes:     it.SomenteConsultaLotes,
		CalcImpIbpt:              it.CalcImpIBPT,
		CredPresumidoIcms:        it.CredPresumidoICMS,
		Ciap:                     it.CIAP,
		VlrAgregadoBaseSubst:     it.VlrAgregadoBaseSubst,
		ContratoFacon:            it.ContratoFacon,
		DescIcmsLicitacoes:       it.DescICMSLicitacoes,
		Sisdeclara:               it.Sisdeclara,
		CodClasTrib:              pgutil.ToPgTextFromPtr(it.CodClasTrib),
		CodClasTribTribReg:       pgutil.ToPgTextFromPtr(it.CodClasTribTribReg),
		CodMotivoRestCompIcmsSt:  pgutil.ToPgTextFromPtr(it.CodMotivoRestCompICMSST),
		CodBeneficioFiscal:       pgutil.ToPgTextFromPtr(it.CodBeneficioFiscal),
	})
	if err != nil {
		return nil, fmt.Errorf("creating invoice type: %w", err)
	}
	return invoiceTypeToEntity(row), nil
}

func (r *CustomerRepositorySQLC) UpdateInvoiceType(ctx context.Context, it *entity.InvoiceType) (*entity.InvoiceType, error) {
	existing, err := r.q.GetInvoiceTypeByCode(ctx, it.Code)
	if err != nil {
		return nil, fmt.Errorf("fetching invoice type for update: %w", err)
	}
	row, err := r.q.UpdateInvoiceType(ctx, sqlc.UpdateInvoiceTypeParams{
		ID:                      existing.ID,
		Description:             it.Description,
		Type:                    sqltypes.InvoiceTypeEnum(it.Type),
		StockMovement:           sqltypes.InvoiceStockEnum(it.StockMovement),
		IcmsType:                sqltypes.InvoiceICMSTypeEnum(it.ICMSType),
		IcmsPct:                 pgutil.ToPgNumericFromFloat64(it.ICMSPct),
		IcmsReductionPct:        pgutil.ToPgNumericFromFloat64(it.ICMSReductionPct),
		IpiPct:                  pgutil.ToPgNumericFromFloat64(it.IPIPct),
		PisPct:                  pgutil.ToPgNumericFromFloat64(it.PISPct),
		CofinsPct:               pgutil.ToPgNumericFromFloat64(it.COFINSPct),
		IssqnPct:                pgutil.ToPgNumericFromFloat64(it.ISSQNPct),
		IrPct:                   pgutil.ToPgNumericFromFloat64(it.IRPct),
		CsllPct:                 pgutil.ToPgNumericFromFloat64(it.CSLLPct),
		InssPct:                 pgutil.ToPgNumericFromFloat64(it.INSSPct),
		GeneratesRevenue:        it.GeneratesRevenue,
		UpdatesInventory:        it.UpdatesInventory,
		GeneratesFinancialTitle: it.GeneratesFinancialTitle,
		ConsidersGoals:          it.ConsidersGoals,
		CalcSubstitutionTax:     it.CalcSubstitutionTax,
		CalcIcmsDeferral:        it.CalcICMSDeferral,
		CalcPisCofins:           it.CalcPISCOFINS,
		CalcDifal:               it.CalcDIFAL,
		RequiresSalesOrder:      it.RequiresSalesOrder,
		ListsFiscalBooks:        it.ListsFiscalBooks,
		IsActive:                it.IsActive,
		ModelNf:                 pgutil.ToPgTextFromPtr(it.ModelNF),
		CstIcms:                 pgutil.ToPgTextFromPtr(it.CSTICMS),
		CsosnIcms:               pgutil.ToPgTextFromPtr(it.CSOSNTICMS),
		CstIpi:                  pgutil.ToPgTextFromPtr(it.CSTIPI),
		CstPis:                  pgutil.ToPgTextFromPtr(it.CSTPIS),
		CstCofins:               pgutil.ToPgTextFromPtr(it.CSTCOFINS),
		BaixaPedido:             it.BaixaPedido,
		GeraTituloDev:           it.GeraTituloDev,
		ExigeSuframa:            it.ExigeSuframa,
		IrPctPresumption:        pgutil.ToPgNumericFromFloat64(it.IRPctPresumption),
		CsllPctPresumption:      pgutil.ToPgNumericFromFloat64(it.CSLLPctPresumption),
		// Extended fields (migration 000126)
		DescriptionNf:            pgutil.ToPgTextFromPtr(it.DescriptionNF),
		ImpostosNfe:              invoiceTypeImpostosNFe(it),
		CfopID:                   it.CFOPId,
		DispositivoLegalIpiID:    it.DispositivoLegalIPIId,
		DispositivoLegalIcmsID:   it.DispositivoLegalICMSId,
		DispositivoLegalIcmsStID: it.DispositivoLegalICMSSTId,
		DispositivoLegalPisID:    it.DispositivoLegalPISId,
		DispositivoLegalCofinsID: it.DispositivoLegalCOFINSId,
		HierarchyIpi:             pgutil.ToPgTextFromPtr(it.HierarchyIPI),
		HierarchyIcms:            pgutil.ToPgTextFromPtr(it.HierarchyICMS),
		HierarchyIcmsSt:          pgutil.ToPgTextFromPtr(it.HierarchyICMSST),
		HierarchyPis:             pgutil.ToPgTextFromPtr(it.HierarchyPIS),
		HierarchyCofins:          pgutil.ToPgTextFromPtr(it.HierarchyCOFINS),
		IpiTransferSalesTableID:  it.IPITransferSalesTableId,
		ListaValorContabil:       it.ListaValorContabil,
		ListaRegistroSaida:       it.ListaRegistroSaida,
		ListaIcmsIpi:             it.ListaICMSIPI,
		SintegraSpedFiscal:       it.SintegraSpedFiscal,
		CalcFomentar:             it.CalcFomentar,
		ExcecaoFomentar:          it.ExcecaoFomentar,
		CompRessRetSt:            it.CompRessRetST,
		CalcReducao:              it.CalcReducao,
		ComplementoItens:         it.ComplementoItens,
		BuscaTipoNf:              it.BuscaTipoNF,
		IcmsStUltEntrada:         it.ICMSSTUltEntrada,
		SomenteConsultaLotes:     it.SomenteConsultaLotes,
		CalcImpIbpt:              it.CalcImpIBPT,
		CredPresumidoIcms:        it.CredPresumidoICMS,
		Ciap:                     it.CIAP,
		VlrAgregadoBaseSubst:     it.VlrAgregadoBaseSubst,
		ContratoFacon:            it.ContratoFacon,
		DescIcmsLicitacoes:       it.DescICMSLicitacoes,
		Sisdeclara:               it.Sisdeclara,
		CodClasTrib:              pgutil.ToPgTextFromPtr(it.CodClasTrib),
		CodClasTribTribReg:       pgutil.ToPgTextFromPtr(it.CodClasTribTribReg),
		CodMotivoRestCompIcmsSt:  pgutil.ToPgTextFromPtr(it.CodMotivoRestCompICMSST),
		CodBeneficioFiscal:       pgutil.ToPgTextFromPtr(it.CodBeneficioFiscal),
	})
	if err != nil {
		return nil, fmt.Errorf("updating invoice type: %w", err)
	}
	return invoiceTypeToEntity(row), nil
}

func (r *CustomerRepositorySQLC) GetInvoiceTypeByCode(ctx context.Context, code int64) (*entity.InvoiceType, error) {
	row, err := r.q.GetInvoiceTypeByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("fetching invoice type %d: %w", code, err)
	}
	return invoiceTypeToEntity(row), nil
}

func (r *CustomerRepositorySQLC) ListInvoiceTypes(ctx context.Context, onlyActive bool) ([]*entity.InvoiceType, error) {
	rows, err := r.q.ListInvoiceTypes(ctx, onlyActive)
	if err != nil {
		return nil, fmt.Errorf("listing invoice types: %w", err)
	}
	out := make([]*entity.InvoiceType, 0, len(rows))
	for _, row := range rows {
		out = append(out, invoiceTypeToEntity(row))
	}
	return out, nil
}

func (r *CustomerRepositorySQLC) NextInvoiceTypeCode(ctx context.Context) (int64, error) {
	code, err := r.q.NextInvoiceTypeCode(ctx)
	return int64(code), err
}

func invoiceTypeToEntity(row sqlc.InvoiceType) *entity.InvoiceType {
	var impostosNFe *entity.ImpostosNFe
	if row.ImpostosNfe.Valid {
		v := entity.ImpostosNFe(row.ImpostosNfe.ImpostosNfeEnum)
		impostosNFe = &v
	}
	return &entity.InvoiceType{
		ID:                      row.ID,
		Code:                    row.Code,
		Description:             row.Description,
		Type:                    entity.InvoiceTypeKind(row.Type),
		StockMovement:           entity.InvoiceStock(row.StockMovement),
		ICMSType:                entity.InvoiceICMSType(row.IcmsType),
		ICMSPct:                 pgutil.FromPgNumericToFloat64(row.IcmsPct),
		ICMSReductionPct:        pgutil.FromPgNumericToFloat64(row.IcmsReductionPct),
		IPIPct:                  pgutil.FromPgNumericToFloat64(row.IpiPct),
		PISPct:                  pgutil.FromPgNumericToFloat64(row.PisPct),
		COFINSPct:               pgutil.FromPgNumericToFloat64(row.CofinsPct),
		ISSQNPct:                pgutil.FromPgNumericToFloat64(row.IssqnPct),
		IRPct:                   pgutil.FromPgNumericToFloat64(row.IrPct),
		CSLLPct:                 pgutil.FromPgNumericToFloat64(row.CsllPct),
		INSSPct:                 pgutil.FromPgNumericToFloat64(row.InssPct),
		GeneratesRevenue:        row.GeneratesRevenue,
		UpdatesInventory:        row.UpdatesInventory,
		GeneratesFinancialTitle: row.GeneratesFinancialTitle,
		ConsidersGoals:          row.ConsidersGoals,
		CalcSubstitutionTax:     row.CalcSubstitutionTax,
		CalcICMSDeferral:        row.CalcIcmsDeferral,
		CalcPISCOFINS:           row.CalcPisCofins,
		CalcDIFAL:               row.CalcDifal,
		RequiresSalesOrder:      row.RequiresSalesOrder,
		ListsFiscalBooks:        row.ListsFiscalBooks,
		IsActive:                row.IsActive,
		ModelNF:                 pgutil.FromPgTextPtr(row.ModelNf),
		CSTICMS:                 pgutil.FromPgTextPtr(row.CstIcms),
		CSOSNTICMS:              pgutil.FromPgTextPtr(row.CsosnIcms),
		CSTIPI:                  pgutil.FromPgTextPtr(row.CstIpi),
		CSTPIS:                  pgutil.FromPgTextPtr(row.CstPis),
		CSTCOFINS:               pgutil.FromPgTextPtr(row.CstCofins),
		BaixaPedido:             row.BaixaPedido,
		GeraTituloDev:           row.GeraTituloDev,
		ExigeSuframa:            row.ExigeSuframa,
		IRPctPresumption:        pgutil.FromPgNumericToFloat64(row.IrPctPresumption),
		CSLLPctPresumption:      pgutil.FromPgNumericToFloat64(row.CsllPctPresumption),
		// Extended fields (migration 000126)
		DescriptionNF:            pgutil.FromPgTextPtr(row.DescriptionNf),
		ImpostosNFe:              impostosNFe,
		CFOPId:                   row.CfopID,
		DispositivoLegalIPIId:    row.DispositivoLegalIpiID,
		DispositivoLegalICMSId:   row.DispositivoLegalIcmsID,
		DispositivoLegalICMSSTId: row.DispositivoLegalIcmsStID,
		DispositivoLegalPISId:    row.DispositivoLegalPisID,
		DispositivoLegalCOFINSId: row.DispositivoLegalCofinsID,
		HierarchyIPI:             pgutil.FromPgTextPtr(row.HierarchyIpi),
		HierarchyICMS:            pgutil.FromPgTextPtr(row.HierarchyIcms),
		HierarchyICMSST:          pgutil.FromPgTextPtr(row.HierarchyIcmsSt),
		HierarchyPIS:             pgutil.FromPgTextPtr(row.HierarchyPis),
		HierarchyCOFINS:          pgutil.FromPgTextPtr(row.HierarchyCofins),
		IPITransferSalesTableId:  row.IpiTransferSalesTableID,
		ListaValorContabil:       row.ListaValorContabil,
		ListaRegistroSaida:       row.ListaRegistroSaida,
		ListaICMSIPI:             row.ListaIcmsIpi,
		SintegraSpedFiscal:       row.SintegraSpedFiscal,
		CalcFomentar:             row.CalcFomentar,
		ExcecaoFomentar:          row.ExcecaoFomentar,
		CompRessRetST:            row.CompRessRetSt,
		CalcReducao:              row.CalcReducao,
		ComplementoItens:         row.ComplementoItens,
		BuscaTipoNF:              row.BuscaTipoNf,
		ICMSSTUltEntrada:         row.IcmsStUltEntrada,
		SomenteConsultaLotes:     row.SomenteConsultaLotes,
		CalcImpIBPT:              row.CalcImpIbpt,
		CredPresumidoICMS:        row.CredPresumidoIcms,
		CIAP:                     row.Ciap,
		VlrAgregadoBaseSubst:     row.VlrAgregadoBaseSubst,
		ContratoFacon:            row.ContratoFacon,
		DescICMSLicitacoes:       row.DescIcmsLicitacoes,
		Sisdeclara:               row.Sisdeclara,
		CodClasTrib:              pgutil.FromPgTextPtr(row.CodClasTrib),
		CodClasTribTribReg:       pgutil.FromPgTextPtr(row.CodClasTribTribReg),
		CodMotivoRestCompICMSST:  pgutil.FromPgTextPtr(row.CodMotivoRestCompIcmsSt),
		CodBeneficioFiscal:       pgutil.FromPgTextPtr(row.CodBeneficioFiscal),
		CreatedAt:                pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── Tax Types ────────────────────────────────────────────────────────────────

func (r *CustomerRepositorySQLC) CreateTaxType(ctx context.Context, tt *entity.TaxType) (*entity.TaxType, error) {
	row, err := r.q.CreateTaxType(ctx, sqlc.CreateTaxTypeParams{
		Code:                          tt.Code,
		Description:                   tt.Description,
		IpiBaseTotalItems:             tt.IPIBaseTotalItems,
		IpiBaseSubtractDiscount:       tt.IPIBaseSubtractDiscount,
		IpiBaseAddFreight:             tt.IPIBaseAddFreight,
		IpiBaseAddExpenses:            tt.IPIBaseAddExpenses,
		IcmsBaseTotalItems:            tt.ICMSBaseTotalItems,
		IcmsBaseSubtractDiscount:      tt.ICMSBaseSubtractDiscount,
		IcmsBaseAddFreight:            tt.ICMSBaseAddFreight,
		IcmsBaseAddIpi:                tt.ICMSBaseAddIPI,
		IcmsBaseAddExpenses:           tt.ICMSBaseAddExpenses,
		PisCofinsBaseTotalItems:       tt.PISCOFINSBaseTotalItems,
		PisCofinsBaseSubtractDiscount: tt.PISCOFINSBaseSubtractDiscount,
		PisCofinsBaseAddFreight:       tt.PISCOFINSBaseAddFreight,
		PisCofinsBaseAddInsurance:     tt.PISCOFINSBaseAddInsurance,
		PisCofinsBaseAddExpenses:      tt.PISCOFINSBaseAddExpenses,
		CsllBaseTotalItems:            tt.CSLLBaseTotalItems,
		CsllBaseSubtractDiscount:      tt.CSLLBaseSubtractDiscount,
		CsllBaseAddFreight:            tt.CSLLBaseAddFreight,
		IrBaseTotalItems:              tt.IRBaseTotalItems,
		IrBaseSubtractDiscount:        tt.IRBaseSubtractDiscount,
		IrBaseAddFreight:              tt.IRBaseAddFreight,
		IsConsumer:                    tt.IsConsumer,
	})
	if err != nil {
		return nil, fmt.Errorf("creating tax type: %w", err)
	}
	return taxTypeToEntity(row), nil
}

func (r *CustomerRepositorySQLC) UpdateTaxType(ctx context.Context, tt *entity.TaxType) (*entity.TaxType, error) {
	return tt, nil
}

func (r *CustomerRepositorySQLC) GetTaxTypeByCode(ctx context.Context, code int64) (*entity.TaxType, error) {
	row, err := r.q.GetTaxTypeByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("fetching tax type %d: %w", code, err)
	}
	return taxTypeToEntity(row), nil
}

func (r *CustomerRepositorySQLC) ListTaxTypes(ctx context.Context, onlyActive bool) ([]*entity.TaxType, error) {
	rows, err := r.q.ListTaxTypes(ctx, onlyActive)
	if err != nil {
		return nil, fmt.Errorf("listing tax types: %w", err)
	}
	out := make([]*entity.TaxType, 0, len(rows))
	for _, row := range rows {
		out = append(out, taxTypeToEntity(row))
	}
	return out, nil
}

func (r *CustomerRepositorySQLC) NextTaxTypeCode(ctx context.Context) (int64, error) {
	code, err := r.q.NextTaxTypeCode(ctx)
	return int64(code), err
}

func taxTypeToEntity(row sqlc.TaxType) *entity.TaxType {
	return &entity.TaxType{
		ID:                            row.ID,
		Code:                          row.Code,
		Description:                   row.Description,
		IPIBaseTotalItems:             row.IpiBaseTotalItems,
		IPIBaseSubtractDiscount:       row.IpiBaseSubtractDiscount,
		IPIBaseAddFreight:             row.IpiBaseAddFreight,
		IPIBaseAddExpenses:            row.IpiBaseAddExpenses,
		ICMSBaseTotalItems:            row.IcmsBaseTotalItems,
		ICMSBaseSubtractDiscount:      row.IcmsBaseSubtractDiscount,
		ICMSBaseAddFreight:            row.IcmsBaseAddFreight,
		ICMSBaseAddIPI:                row.IcmsBaseAddIpi,
		ICMSBaseAddExpenses:           row.IcmsBaseAddExpenses,
		PISCOFINSBaseTotalItems:       row.PisCofinsBaseTotalItems,
		PISCOFINSBaseSubtractDiscount: row.PisCofinsBaseSubtractDiscount,
		PISCOFINSBaseAddFreight:       row.PisCofinsBaseAddFreight,
		PISCOFINSBaseAddInsurance:     row.PisCofinsBaseAddInsurance,
		PISCOFINSBaseAddExpenses:      row.PisCofinsBaseAddExpenses,
		CSLLBaseTotalItems:            row.CsllBaseTotalItems,
		CSLLBaseSubtractDiscount:      row.CsllBaseSubtractDiscount,
		CSLLBaseAddFreight:            row.CsllBaseAddFreight,
		IRBaseTotalItems:              row.IrBaseTotalItems,
		IRBaseSubtractDiscount:        row.IrBaseSubtractDiscount,
		IRBaseAddFreight:              row.IrBaseAddFreight,
		IsConsumer:                    row.IsConsumer,
		IsActive:                      row.IsActive,
		CreatedAt:                     pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── Customers ────────────────────────────────────────────────────────────────

func (r *CustomerRepositorySQLC) CreateCustomer(ctx context.Context, c *entity.Customer) (*entity.Customer, error) {
	row, err := r.q.CreateCustomer(ctx, sqlc.CreateCustomerParams{
		Code:                  c.Code,
		CorporateCode:         c.CorporateCode,
		IsCorporate:           c.IsCorporate,
		Name:                  c.Name,
		TradeName:             pgutil.ToPgTextFromPtr(c.TradeName),
		DocumentType:          sqltypes.DocumentTypeEnum(c.DocumentType),
		DocumentNumber:        c.DocumentNumber,
		StateRegistration:     pgutil.ToPgTextFromPtr(c.StateRegistration),
		MunicipalRegistration: pgutil.ToPgTextFromPtr(c.MunicipalRegistration),
		SuframaCode:           pgutil.ToPgTextFromPtr(c.SuframaCode),
		SuframaExpiry:         pgutil.ToPgDateFromPtr(c.SuframaExpiry),
		RegionID:              c.RegionID,
		MarketSegmentID:       c.MarketSegmentID,
		CustomerTypeID:        c.CustomerTypeID,
		PaymentConditionID:    c.PaymentConditionID,
		SalesTableID:          c.SalesTableID,
		CarrierID:             c.CarrierID,
		CarrierGroupID:        c.CarrierGroupID,
		InvoiceTypeID:         c.InvoiceTypeID,
		TaxTypeID:             c.TaxTypeID,
		PaymentCondVisibility: sqltypes.PaymentCondVisibilityEnum(c.PaymentCondVisibility),
		CreditLimit:           pgutil.ToPgNumericFromFloat64(c.CreditLimit),
		Website:               pgutil.ToPgTextFromPtr(c.Website),
		CreatedBy:             pgutil.ToPgUUID(c.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating customer: %w", err)
	}
	return customerToEntity(row), nil
}

func (r *CustomerRepositorySQLC) UpdateCustomer(ctx context.Context, c *entity.Customer) (*entity.Customer, error) {
	row, err := r.q.UpdateCustomer(ctx, sqlc.UpdateCustomerParams{
		ID:                    c.ID,
		Name:                  c.Name,
		TradeName:             pgutil.ToPgTextFromPtr(c.TradeName),
		StateRegistration:     pgutil.ToPgTextFromPtr(c.StateRegistration),
		MunicipalRegistration: pgutil.ToPgTextFromPtr(c.MunicipalRegistration),
		SuframaCode:           pgutil.ToPgTextFromPtr(c.SuframaCode),
		SuframaExpiry:         pgutil.ToPgDateFromPtr(c.SuframaExpiry),
		RegionID:              c.RegionID,
		MarketSegmentID:       c.MarketSegmentID,
		CustomerTypeID:        c.CustomerTypeID,
		PaymentConditionID:    c.PaymentConditionID,
		SalesTableID:          c.SalesTableID,
		CarrierID:             c.CarrierID,
		CarrierGroupID:        c.CarrierGroupID,
		InvoiceTypeID:         c.InvoiceTypeID,
		TaxTypeID:             c.TaxTypeID,
		PaymentCondVisibility: sqltypes.PaymentCondVisibilityEnum(c.PaymentCondVisibility),
		CreditLimit:           pgutil.ToPgNumericFromFloat64(c.CreditLimit),
		Website:               pgutil.ToPgTextFromPtr(c.Website),
		IsActive:              c.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("updating customer %d: %w", c.ID, err)
	}
	return customerToEntity(row), nil
}

func (r *CustomerRepositorySQLC) GetCustomerByCode(ctx context.Context, code int64) (*entity.Customer, error) {
	row, err := r.q.GetCustomerByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("fetching customer %d: %w", code, err)
	}
	return customerToEntity(row), nil
}

func (r *CustomerRepositorySQLC) GetCustomerByDocument(ctx context.Context, document string) (*entity.Customer, error) {
	row, err := r.q.GetCustomerByDocument(ctx, document)
	if err != nil {
		return nil, fmt.Errorf("fetching customer by document %s: %w", document, err)
	}
	return customerToEntity(row), nil
}

func (r *CustomerRepositorySQLC) ListCustomers(ctx context.Context, onlyActive bool) ([]*entity.Customer, error) {
	rows, err := r.q.ListCustomers(ctx, onlyActive)
	if err != nil {
		return nil, fmt.Errorf("listing customers: %w", err)
	}
	out := make([]*entity.Customer, 0, len(rows))
	for _, row := range rows {
		out = append(out, customerToEntity(row))
	}
	return out, nil
}

func (r *CustomerRepositorySQLC) ListEstablishments(ctx context.Context, corporateCode int64) ([]*entity.Customer, error) {
	rows, err := r.q.ListEstablishments(ctx, &corporateCode)
	if err != nil {
		return nil, fmt.Errorf("listing establishments for corporate %d: %w", corporateCode, err)
	}
	out := make([]*entity.Customer, 0, len(rows))
	for _, row := range rows {
		out = append(out, customerToEntity(row))
	}
	return out, nil
}

func (r *CustomerRepositorySQLC) BlockCustomer(ctx context.Context, code int64, reason string) error {
	return r.q.BlockCustomer(ctx, sqlc.BlockCustomerParams{
		Code:        code,
		BlockReason: pgutil.ToPgTextFromString(reason),
	})
}

func (r *CustomerRepositorySQLC) UnblockCustomer(ctx context.Context, code int64) error {
	return r.q.UnblockCustomer(ctx, code)
}

func (r *CustomerRepositorySQLC) NextCustomerCode(ctx context.Context) (int64, error) {
	code, err := r.q.NextCustomerCode(ctx)
	return int64(code), err
}

func customerToEntity(row sqlc.Customer) *entity.Customer {
	return &entity.Customer{
		ID:                    row.ID,
		Code:                  row.Code,
		CorporateCode:         row.CorporateCode,
		IsCorporate:           row.IsCorporate,
		Name:                  row.Name,
		TradeName:             pgutil.FromPgTextPtr(row.TradeName),
		DocumentType:          entity.DocumentType(row.DocumentType),
		DocumentNumber:        row.DocumentNumber,
		StateRegistration:     pgutil.FromPgTextPtr(row.StateRegistration),
		MunicipalRegistration: pgutil.FromPgTextPtr(row.MunicipalRegistration),
		SuframaCode:           pgutil.FromPgTextPtr(row.SuframaCode),
		SuframaExpiry:         pgutil.FromPgDateToPtr(row.SuframaExpiry),
		RegionID:              row.RegionID,
		MarketSegmentID:       row.MarketSegmentID,
		CustomerTypeID:        row.CustomerTypeID,
		PaymentConditionID:    row.PaymentConditionID,
		SalesTableID:          row.SalesTableID,
		CarrierID:             row.CarrierID,
		CarrierGroupID:        row.CarrierGroupID,
		InvoiceTypeID:         row.InvoiceTypeID,
		TaxTypeID:             row.TaxTypeID,
		PaymentCondVisibility: entity.PaymentCondVisibility(row.PaymentCondVisibility),
		CreditLimit:           pgutil.FromPgNumericToFloat64(row.CreditLimit),
		Website:               pgutil.FromPgTextPtr(row.Website),
		IsActive:              row.IsActive,
		Blocked:               row.Blocked,
		BlockReason:           pgutil.FromPgTextPtr(row.BlockReason),
		CreatedAt:             pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy:             pgutil.FromPgUUID(row.CreatedBy),
		UpdatedAt:             pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}

// ─── Customer Addresses ───────────────────────────────────────────────────────

func (r *CustomerRepositorySQLC) AddAddress(ctx context.Context, addr *entity.CustomerAddress) (*entity.CustomerAddress, error) {
	row, err := r.q.AddAddress(ctx, sqlc.AddAddressParams{
		CustomerID:   addr.CustomerID,
		AddressType:  sqltypes.CustomerAddressTypeEnum(addr.AddressType),
		ZipCode:      pgutil.ToPgTextFromPtr(addr.ZipCode),
		Street:       pgutil.ToPgTextFromPtr(addr.Street),
		Number:       pgutil.ToPgTextFromPtr(addr.Number),
		Complement:   pgutil.ToPgTextFromPtr(addr.Complement),
		Neighborhood: pgutil.ToPgTextFromPtr(addr.Neighborhood),
		City:         pgutil.ToPgTextFromPtr(addr.City),
		Uf:           pgutil.ToPgTextFromPtr(addr.UF),
		Country:      addr.Country,
		IsDefault:    addr.IsDefault,
	})
	if err != nil {
		return nil, fmt.Errorf("adding address: %w", err)
	}
	return addressToEntity(row), nil
}

func (r *CustomerRepositorySQLC) UpdateAddress(ctx context.Context, addr *entity.CustomerAddress) (*entity.CustomerAddress, error) {
	return addr, nil
}

func (r *CustomerRepositorySQLC) ListAddresses(ctx context.Context, customerID int64) ([]*entity.CustomerAddress, error) {
	rows, err := r.q.ListAddresses(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("listing addresses: %w", err)
	}
	out := make([]*entity.CustomerAddress, 0, len(rows))
	for _, row := range rows {
		out = append(out, addressToEntity(row))
	}
	return out, nil
}

func (r *CustomerRepositorySQLC) DeleteAddress(ctx context.Context, id int64) error {
	return r.q.DeleteAddress(ctx, id)
}

func addressToEntity(row sqlc.CustomerAddress) *entity.CustomerAddress {
	return &entity.CustomerAddress{
		ID:           row.ID,
		CustomerID:   row.CustomerID,
		AddressType:  entity.AddressType(row.AddressType),
		ZipCode:      pgutil.FromPgTextPtr(row.ZipCode),
		Street:       pgutil.FromPgTextPtr(row.Street),
		Number:       pgutil.FromPgTextPtr(row.Number),
		Complement:   pgutil.FromPgTextPtr(row.Complement),
		Neighborhood: pgutil.FromPgTextPtr(row.Neighborhood),
		City:         pgutil.FromPgTextPtr(row.City),
		UF:           pgutil.FromPgTextPtr(row.Uf),
		Country:      row.Country,
		IsDefault:    row.IsDefault,
		CreatedAt:    pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── Customer Contacts ────────────────────────────────────────────────────────

func (r *CustomerRepositorySQLC) AddContact(ctx context.Context, c *entity.CustomerContact) (*entity.CustomerContact, error) {
	row, err := r.q.AddContact(ctx, sqlc.AddContactParams{
		CustomerID:    c.CustomerID,
		ContactTypeID: c.ContactTypeID,
		Name:          c.Name,
		Email:         pgutil.ToPgTextFromPtr(c.Email),
		Phone:         pgutil.ToPgTextFromPtr(c.Phone),
		Mobile:        pgutil.ToPgTextFromPtr(c.Mobile),
		Position:      pgutil.ToPgTextFromPtr(c.Position),
		IsPrimary:     c.IsPrimary,
	})
	if err != nil {
		return nil, fmt.Errorf("adding contact: %w", err)
	}
	return contactToEntity(row), nil
}

func (r *CustomerRepositorySQLC) UpdateContact(ctx context.Context, c *entity.CustomerContact) (*entity.CustomerContact, error) {
	return c, nil
}

func (r *CustomerRepositorySQLC) ListContacts(ctx context.Context, customerID int64) ([]*entity.CustomerContact, error) {
	rows, err := r.q.ListContacts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("listing contacts: %w", err)
	}
	out := make([]*entity.CustomerContact, 0, len(rows))
	for _, row := range rows {
		out = append(out, contactToEntity(row))
	}
	return out, nil
}

func (r *CustomerRepositorySQLC) DeleteContact(ctx context.Context, id int64) error {
	return r.q.DeleteContact(ctx, id)
}

func contactToEntity(row sqlc.CustomerContact) *entity.CustomerContact {
	return &entity.CustomerContact{
		ID:            row.ID,
		CustomerID:    row.CustomerID,
		ContactTypeID: row.ContactTypeID,
		Name:          row.Name,
		Email:         pgutil.FromPgTextPtr(row.Email),
		Phone:         pgutil.FromPgTextPtr(row.Phone),
		Mobile:        pgutil.FromPgTextPtr(row.Mobile),
		Position:      pgutil.FromPgTextPtr(row.Position),
		IsPrimary:     row.IsPrimary,
		IsActive:      row.IsActive,
		CreatedAt:     pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── Sales Table Prices ───────────────────────────────────────────────────────

func (r *CustomerRepositorySQLC) CreateSalesTablePrice(ctx context.Context, p *entity.SalesTablePrice) (*entity.SalesTablePrice, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO sales_table_prices
		 (sales_table_id, item_code, price, ume, umc, price_conv, formula, situation, blocked, observation, product_line_id, item_mask)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		 RETURNING id, created_at`,
		p.SalesTableID, p.ItemCode, p.Price, p.UME, p.UMC, p.PriceConv,
		p.Formula, string(p.Situation), p.Blocked, p.Observation, p.ProductLineID, p.ItemMask,
	).Scan(&p.ID, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating sales table price: %w", err)
	}
	return p, nil
}

func (r *CustomerRepositorySQLC) UpsertSalesTablePrice(ctx context.Context, p *entity.SalesTablePrice) (*entity.SalesTablePrice, *float64, error) {
	var oldPrice pgtype.Numeric
	row := r.pool.QueryRow(ctx,
		`WITH old AS (
			SELECT price FROM sales_table_prices WHERE sales_table_id=$1 AND item_code=$2
		 ),
		 upserted AS (
			INSERT INTO sales_table_prices
			 (sales_table_id, item_code, price, ume, umc, price_conv, formula, situation, blocked, observation, product_line_id, item_mask)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			 ON CONFLICT (sales_table_id, item_code) DO UPDATE
			 SET price=EXCLUDED.price, ume=EXCLUDED.ume, umc=EXCLUDED.umc,
			     price_conv=EXCLUDED.price_conv, formula=EXCLUDED.formula,
			     situation=EXCLUDED.situation, blocked=EXCLUDED.blocked,
			     observation=EXCLUDED.observation, product_line_id=EXCLUDED.product_line_id,
			     item_mask=EXCLUDED.item_mask
			 RETURNING id, sales_table_id, item_code, price, ume, umc, price_conv,
			           formula, situation, blocked, observation, product_line_id, item_mask, created_at
		 )
		 SELECT upserted.id, upserted.sales_table_id, upserted.item_code, upserted.price,
		        upserted.ume, upserted.umc, upserted.price_conv, upserted.formula,
		        upserted.situation, upserted.blocked, upserted.observation,
		        upserted.product_line_id, upserted.item_mask, upserted.created_at,
		        (SELECT price FROM old)
		 FROM upserted`,
		p.SalesTableID, p.ItemCode, p.Price, p.UME, p.UMC, p.PriceConv,
		p.Formula, string(p.Situation), p.Blocked, p.Observation, p.ProductLineID, p.ItemMask,
	)
	var out entity.SalesTablePrice
	var sit string
	err := row.Scan(&out.ID, &out.SalesTableID, &out.ItemCode, &out.Price, &out.UME, &out.UMC,
		&out.PriceConv, &out.Formula, &sit, &out.Blocked, &out.Observation,
		&out.ProductLineID, &out.ItemMask, &out.CreatedAt, &oldPrice)
	if err != nil {
		return nil, nil, fmt.Errorf("upserting sales table price: %w", err)
	}
	out.Situation = entity.PriceSituation(sit)
	var oldPtr *float64
	if oldPrice.Valid {
		v := pgutil.FromPgNumericToFloat64(oldPrice)
		oldPtr = &v
	}
	return &out, oldPtr, nil
}

func (r *CustomerRepositorySQLC) UpdateSalesTablePrice(ctx context.Context, p *entity.SalesTablePrice) (*entity.SalesTablePrice, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE sales_table_prices
		 SET price=$1, ume=$2, umc=$3, price_conv=$4, formula=$5, situation=$6, blocked=$7, observation=$8, product_line_id=$9, item_mask=$10
		 WHERE id=$11
		 RETURNING id, sales_table_id, item_code, price, ume, umc, price_conv, formula, situation, blocked, observation, product_line_id, item_mask, created_at`,
		p.Price, p.UME, p.UMC, p.PriceConv, p.Formula, string(p.Situation),
		p.Blocked, p.Observation, p.ProductLineID, p.ItemMask, p.ID,
	)
	updated, err := scanSalesTablePrice(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("sales table price %d not found", p.ID)
		}
		return nil, fmt.Errorf("updating sales table price %d: %w", p.ID, err)
	}
	return updated, nil
}

func (r *CustomerRepositorySQLC) GetSalesTablePrice(ctx context.Context, salesTableID int64, itemCode string) (*entity.SalesTablePrice, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, sales_table_id, item_code, price, ume, umc, price_conv, formula, situation, blocked, observation, product_line_id, item_mask, created_at
		 FROM sales_table_prices WHERE sales_table_id=$1 AND item_code=$2`,
		salesTableID, itemCode)
	p, err := scanSalesTablePrice(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("sales table price not found for table %d item %s", salesTableID, itemCode)
		}
		return nil, fmt.Errorf("getting sales table price: %w", err)
	}
	return p, nil
}

func (r *CustomerRepositorySQLC) GetSalesTablePriceByID(ctx context.Context, id int64) (*entity.SalesTablePrice, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, sales_table_id, item_code, price, ume, umc, price_conv, formula, situation, blocked, observation, product_line_id, item_mask, created_at
		 FROM sales_table_prices WHERE id=$1`, id)
	p, err := scanSalesTablePrice(row)
	if err != nil {
		return nil, fmt.Errorf("fetching sales table price %d: %w", id, err)
	}
	return p, nil
}

func (r *CustomerRepositorySQLC) ListSalesTablePrices(ctx context.Context, salesTableID int64) ([]*entity.SalesTablePrice, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, sales_table_id, item_code, price, ume, umc, price_conv, formula, situation, blocked, observation, product_line_id, item_mask, created_at
		 FROM sales_table_prices WHERE sales_table_id=$1 ORDER BY item_code`,
		salesTableID)
	if err != nil {
		return nil, fmt.Errorf("listing sales table prices: %w", err)
	}
	defer rows.Close()
	var out []*entity.SalesTablePrice
	for rows.Next() {
		p, err := scanSalesTablePrice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *CustomerRepositorySQLC) DeleteSalesTablePrice(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sales_table_prices WHERE id=$1`, id)
	return err
}

func (r *CustomerRepositorySQLC) ResolveSalesCost(ctx context.Context, itemCode int64, mask string, source entity.SalesCostSource, warehouseID *int64) (float64, string, error) {
	var value float64
	switch source {
	case entity.SalesCostStandardTotal:
		err := r.pool.QueryRow(ctx,
			`SELECT total_cost FROM item_standard_costs
			 WHERE item_code=$1 AND mask=$2 ORDER BY calculated_at DESC LIMIT 1`, itemCode, mask).Scan(&value)
		return value, string(source), normalizeCostErr(err, itemCode, source)
	case entity.SalesCostStandardMaterial:
		err := r.pool.QueryRow(ctx,
			`SELECT material_cost FROM item_standard_costs
			 WHERE item_code=$1 AND mask=$2 ORDER BY calculated_at DESC LIMIT 1`, itemCode, mask).Scan(&value)
		return value, string(source), normalizeCostErr(err, itemCode, source)
	case entity.SalesCostPurchase:
		err := r.pool.QueryRow(ctx,
			`SELECT unit_cost FROM item_purchase_costs
			 WHERE item_code=$1 ORDER BY updated_at DESC LIMIT 1`, itemCode).Scan(&value)
		return value, string(source), normalizeCostErr(err, itemCode, source)
	case entity.SalesCostStockAvg:
		return r.resolveStockCost(ctx, itemCode, mask, warehouseID, true)
	case entity.SalesCostStockLast:
		return r.resolveStockCost(ctx, itemCode, mask, warehouseID, false)
	default:
		return 0, string(source), fmt.Errorf("cost source %s requires informed base_cost", source)
	}
}

func (r *CustomerRepositorySQLC) resolveStockCost(ctx context.Context, itemCode int64, mask string, warehouseID *int64, avg bool) (float64, string, error) {
	col := "last_cost"
	source := "STOCK_LAST"
	if avg {
		col = "avg_cost"
		source = "STOCK_AVG"
	}
	var value float64
	var err error
	if warehouseID != nil {
		err = r.pool.QueryRow(ctx,
			fmt.Sprintf(`SELECT %s FROM stock_balances WHERE item_code=$1 AND mask=$2 AND warehouse_id=$3`, col),
			itemCode, mask, *warehouseID).Scan(&value)
	} else {
		err = r.pool.QueryRow(ctx,
			fmt.Sprintf(`SELECT %s FROM stock_balances WHERE item_code=$1 AND mask=$2 ORDER BY quantity DESC LIMIT 1`, col),
			itemCode, mask).Scan(&value)
	}
	return value, source, normalizeCostErr(err, itemCode, entity.SalesCostSource(source))
}

func normalizeCostErr(err error, itemCode int64, source entity.SalesCostSource) error {
	if err == nil {
		return nil
	}
	if err == pgx.ErrNoRows {
		return fmt.Errorf("cost not found for item %d using source %s", itemCode, source)
	}
	return err
}

func (r *CustomerRepositorySQLC) CreateSalesTablePriceHistory(ctx context.Context, h *entity.SalesTablePriceHistory) (*entity.SalesTablePriceHistory, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO sales_table_price_history (
			sales_table_price_id, sales_table_id, sales_table_code, item_code,
			old_price, new_price, base_cost, source, policy_code, reason
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, sales_table_price_id, sales_table_id, sales_table_code, item_code,
		          old_price, new_price, base_cost, source, policy_code, reason, created_at`,
		h.SalesTablePriceID, h.SalesTableID, h.SalesTableCode, h.ItemCode,
		h.OldPrice, h.NewPrice, h.BaseCost, h.Source, h.PolicyCode, h.Reason)
	created, err := scanSalesTablePriceHistory(row)
	if err != nil {
		return nil, fmt.Errorf("creating sales table price history: %w", err)
	}
	return created, nil
}

func (r *CustomerRepositorySQLC) ListSalesTablePriceHistory(ctx context.Context, salesTableCode int64, itemCode *string) ([]*entity.SalesTablePriceHistory, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, sales_table_price_id, sales_table_id, sales_table_code, item_code,
		       old_price, new_price, base_cost, source, policy_code, reason, created_at
		FROM sales_table_price_history
		WHERE sales_table_code=$1 AND ($2::TEXT IS NULL OR item_code=$2)
		ORDER BY created_at DESC`, salesTableCode, itemCode)
	if err != nil {
		return nil, fmt.Errorf("listing sales table price history: %w", err)
	}
	defer rows.Close()
	out := make([]*entity.SalesTablePriceHistory, 0)
	for rows.Next() {
		h, err := scanSalesTablePriceHistory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func scanSalesTablePriceHistory(row scannable) (*entity.SalesTablePriceHistory, error) {
	var h entity.SalesTablePriceHistory
	err := row.Scan(&h.ID, &h.SalesTablePriceID, &h.SalesTableID, &h.SalesTableCode,
		&h.ItemCode, &h.OldPrice, &h.NewPrice, &h.BaseCost, &h.Source,
		&h.PolicyCode, &h.Reason, &h.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanSalesTablePrice(row scannable) (*entity.SalesTablePrice, error) {
	var p entity.SalesTablePrice
	var sit string
	err := row.Scan(&p.ID, &p.SalesTableID, &p.ItemCode, &p.Price, &p.UME, &p.UMC,
		&p.PriceConv, &p.Formula, &sit, &p.Blocked, &p.Observation, &p.ProductLineID, &p.ItemMask, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	p.Situation = entity.PriceSituation(sit)
	return &p, nil
}
