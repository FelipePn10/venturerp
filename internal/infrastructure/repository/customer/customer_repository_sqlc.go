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
	return c, nil
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
