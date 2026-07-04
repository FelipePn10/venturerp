package customer_uc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
)

// CustomerUseCase consolidates all customer-related operations.
type CustomerUseCase struct {
	repo repository.CustomerRepository
}

func NewCustomerUseCase(repo repository.CustomerRepository) *CustomerUseCase {
	return &CustomerUseCase{repo: repo}
}

// ─── Regions ─────────────────────────────────────────────────────────────────

func (uc *CustomerUseCase) CreateRegion(ctx context.Context, dto request.CreateRegionDTO) (*response.RegionResponse, error) {
	code, err := uc.repo.NextRegionCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating region code: %w", err)
	}
	reg, err := entity.NewRegion(code, dto.Description, dto.UF, dto.City, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.CreateRegion(ctx, reg)
	if err != nil {
		return nil, err
	}
	return toRegionResponse(created), nil
}

func (uc *CustomerUseCase) UpdateRegion(ctx context.Context, dto request.UpdateRegionDTO) (*response.RegionResponse, error) {
	reg, err := uc.repo.GetRegionByCode(ctx, dto.ID)
	if err != nil {
		return nil, fmt.Errorf("region not found: %w", err)
	}
	reg.Description = dto.Description
	reg.UF = dto.UF
	reg.City = dto.City
	reg.IsActive = dto.IsActive
	updated, err := uc.repo.UpdateRegion(ctx, reg)
	if err != nil {
		return nil, err
	}
	return toRegionResponse(updated), nil
}

func (uc *CustomerUseCase) GetRegion(ctx context.Context, code int64) (*response.RegionResponse, error) {
	reg, err := uc.repo.GetRegionByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("region %d not found: %w", code, err)
	}
	return toRegionResponse(reg), nil
}

func (uc *CustomerUseCase) ListRegions(ctx context.Context, onlyActive bool) ([]*response.RegionResponse, error) {
	regs, err := uc.repo.ListRegions(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.RegionResponse, 0, len(regs))
	for _, r := range regs {
		out = append(out, toRegionResponse(r))
	}
	return out, nil
}

func toRegionResponse(r *entity.Region) *response.RegionResponse {
	return &response.RegionResponse{
		ID:          r.ID,
		Code:        r.Code,
		Description: r.Description,
		UF:          r.UF,
		City:        r.City,
		IsActive:    r.IsActive,
	}
}

// ─── Market Segments ──────────────────────────────────────────────────────────

func (uc *CustomerUseCase) CreateMarketSegment(ctx context.Context, dto request.CreateMarketSegmentDTO) (*response.MarketSegmentResponse, error) {
	code, err := uc.repo.NextMarketSegmentCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating segment code: %w", err)
	}
	var parentID *int64
	if dto.ParentCode != nil {
		seg, err := uc.repo.GetMarketSegmentByCode(ctx, *dto.ParentCode)
		if err != nil {
			return nil, fmt.Errorf("parent segment not found: %w", err)
		}
		parentID = &seg.ID
	}
	seg, err := entity.NewMarketSegment(code, dto.Description, parentID, dto.HasPISCOFINSRetention, dto.RetentionIndicator)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.CreateMarketSegment(ctx, seg)
	if err != nil {
		return nil, err
	}
	return toSegmentResponse(created), nil
}

func (uc *CustomerUseCase) ListMarketSegments(ctx context.Context, onlyActive bool) ([]*response.MarketSegmentResponse, error) {
	segs, err := uc.repo.ListMarketSegments(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.MarketSegmentResponse, 0, len(segs))
	for _, s := range segs {
		out = append(out, toSegmentResponse(s))
	}
	return out, nil
}

func toSegmentResponse(s *entity.MarketSegment) *response.MarketSegmentResponse {
	return &response.MarketSegmentResponse{
		ID:                    s.ID,
		Code:                  s.Code,
		Description:           s.Description,
		ParentID:              s.ParentID,
		HasPISCOFINSRetention: s.HasPISCOFINSRetention,
		RetentionIndicator:    s.RetentionIndicator,
		IsActive:              s.IsActive,
	}
}

// ─── Customer Contact Types ───────────────────────────────────────────────────

func (uc *CustomerUseCase) CreateContactType(ctx context.Context, dto request.CreateContactTypeDTO) (*response.ContactTypeResponse, error) {
	code, err := uc.repo.NextContactTypeCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating contact type code: %w", err)
	}
	ct, err := entity.NewContactType(code, dto.Description)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.CreateContactType(ctx, ct)
	if err != nil {
		return nil, err
	}
	return toContactTypeResponse(created), nil
}

func (uc *CustomerUseCase) ListContactTypes(ctx context.Context, onlyActive bool) ([]*response.ContactTypeResponse, error) {
	cts, err := uc.repo.ListContactTypes(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ContactTypeResponse, 0, len(cts))
	for _, ct := range cts {
		out = append(out, toContactTypeResponse(ct))
	}
	return out, nil
}

func toContactTypeResponse(ct *entity.CustomerContactType) *response.ContactTypeResponse {
	return &response.ContactTypeResponse{
		ID:          ct.ID,
		Code:        ct.Code,
		Description: ct.Description,
		IsActive:    ct.IsActive,
	}
}

// ─── Customer Types ───────────────────────────────────────────────────────────

func (uc *CustomerUseCase) CreateCustomerType(ctx context.Context, dto request.CreateCustomerTypeDTO) (*response.CustomerTypeResponse, error) {
	ct, err := entity.NewCustomerType(dto.Code, dto.Description, entity.CustomerCategory(dto.Category), dto.DeliveryDays)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.CreateCustomerType(ctx, ct)
	if err != nil {
		return nil, err
	}
	return toCustomerTypeResponse(created), nil
}

func (uc *CustomerUseCase) ListCustomerTypes(ctx context.Context, onlyActive bool) ([]*response.CustomerTypeResponse, error) {
	cts, err := uc.repo.ListCustomerTypes(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CustomerTypeResponse, 0, len(cts))
	for _, ct := range cts {
		out = append(out, toCustomerTypeResponse(ct))
	}
	return out, nil
}

func toCustomerTypeResponse(ct *entity.CustomerType) *response.CustomerTypeResponse {
	return &response.CustomerTypeResponse{
		ID:           ct.ID,
		Code:         ct.Code,
		Description:  ct.Description,
		Category:     string(ct.Category),
		DeliveryDays: ct.DeliveryDays,
		IsActive:     ct.IsActive,
	}
}

// ─── Carriers ─────────────────────────────────────────────────────────────────

func (uc *CustomerUseCase) CreateCarrier(ctx context.Context, dto request.CreateCarrierDTO) (*response.CarrierResponse, error) {
	code, err := uc.repo.NextCarrierCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating carrier code: %w", err)
	}
	c, err := entity.NewCarrier(code, dto.Description, entity.CarrierBillingType(dto.BillingType))
	if err != nil {
		return nil, err
	}
	c.UsesCreditLimit = dto.UsesCreditLimit
	c.ConsiderAvailable = dto.ConsiderAvailable
	c.PostponeDueDate = dto.PostponeDueDate
	c.ReceiptDays = dto.ReceiptDays
	c.PaymentDays = dto.PaymentDays
	created, err := uc.repo.CreateCarrier(ctx, c)
	if err != nil {
		return nil, err
	}
	return toCarrierResponse(created), nil
}

func (uc *CustomerUseCase) ListCarriers(ctx context.Context, onlyActive bool) ([]*response.CarrierResponse, error) {
	carriers, err := uc.repo.ListCarriers(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CarrierResponse, 0, len(carriers))
	for _, c := range carriers {
		out = append(out, toCarrierResponse(c))
	}
	return out, nil
}

func toCarrierResponse(c *entity.Carrier) *response.CarrierResponse {
	return &response.CarrierResponse{
		ID:                c.ID,
		Code:              c.Code,
		Description:       c.Description,
		BillingType:       string(c.BillingType),
		UsesCreditLimit:   c.UsesCreditLimit,
		ConsiderAvailable: c.ConsiderAvailable,
		PostponeDueDate:   c.PostponeDueDate,
		ReceiptDays:       c.ReceiptDays,
		PaymentDays:       c.PaymentDays,
		IsActive:          c.IsActive,
	}
}

// ─── Carrier Groups ───────────────────────────────────────────────────────────

func (uc *CustomerUseCase) CreateCarrierGroup(ctx context.Context, dto request.CreateCarrierGroupDTO) (*response.CarrierGroupResponse, error) {
	code, err := uc.repo.NextCarrierGroupCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating carrier group code: %w", err)
	}
	created, err := uc.repo.CreateCarrierGroup(ctx, &entity.CarrierGroup{
		Code:        code,
		Description: dto.Description,
	})
	if err != nil {
		return nil, err
	}
	return &response.CarrierGroupResponse{ID: created.ID, Code: created.Code, Description: created.Description}, nil
}

func (uc *CustomerUseCase) ListCarrierGroups(ctx context.Context) ([]*response.CarrierGroupResponse, error) {
	groups, err := uc.repo.ListCarrierGroups(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CarrierGroupResponse, 0, len(groups))
	for _, g := range groups {
		out = append(out, &response.CarrierGroupResponse{ID: g.ID, Code: g.Code, Description: g.Description})
	}
	return out, nil
}

func (uc *CustomerUseCase) AddCarrierToGroup(ctx context.Context, dto request.CarrierGroupMemberDTO) error {
	group, err := uc.repo.GetCarrierGroupByCode(ctx, dto.CarrierGroupCode)
	if err != nil {
		return fmt.Errorf("carrier group not found: %w", err)
	}
	carrier, err := uc.repo.GetCarrierByCode(ctx, dto.CarrierCode)
	if err != nil {
		return fmt.Errorf("carrier not found: %w", err)
	}
	return uc.repo.AddCarrierToGroup(ctx, group.ID, carrier.ID)
}

// ─── Payment Conditions ───────────────────────────────────────────────────────

func (uc *CustomerUseCase) CreatePaymentCondition(ctx context.Context, dto request.CreatePaymentConditionDTO) (*response.PaymentConditionResponse, error) {
	code, err := uc.repo.NextPaymentConditionCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating payment condition code: %w", err)
	}
	pc, err := entity.NewPaymentCondition(code, dto.Description, entity.PaymentAnalysis(dto.AnalysisType))
	if err != nil {
		return nil, err
	}
	pc.ParcelStart = entity.PaymentParcelStart(dto.ParcelStart)
	pc.Expenses = dto.Expenses
	pc.AverageTerm = dto.AverageTerm
	pc.IsSpecial = dto.IsSpecial
	pc.IsRevenue = dto.IsRevenue
	pc.IsAtSight = dto.IsAtSight
	if dto.CarrierCode != nil {
		c, err := uc.repo.GetCarrierByCode(ctx, *dto.CarrierCode)
		if err != nil {
			return nil, fmt.Errorf("carrier not found: %w", err)
		}
		pc.CarrierID = &c.ID
	}
	created, err := uc.repo.CreatePaymentCondition(ctx, pc)
	if err != nil {
		return nil, err
	}
	return toPaymentCondResponse(created), nil
}

func (uc *CustomerUseCase) AddInstallment(ctx context.Context, dto request.AddInstallmentDTO) (*response.InstallmentResponse, error) {
	pc, err := uc.repo.GetPaymentConditionByCode(ctx, dto.PaymentConditionCode)
	if err != nil {
		return nil, fmt.Errorf("payment condition not found: %w", err)
	}
	inst := &entity.PaymentInstallment{
		PaymentConditionID: pc.ID,
		InstallmentNumber:  dto.InstallmentNumber,
		DueDays:            dto.DueDays,
		Description:        dto.Description,
		DocumentType:       dto.DocumentType,
		MovementType:       dto.MovementType,
	}
	if dto.CarrierCode != nil {
		c, err := uc.repo.GetCarrierByCode(ctx, *dto.CarrierCode)
		if err != nil {
			return nil, fmt.Errorf("carrier not found: %w", err)
		}
		inst.CarrierID = &c.ID
	}
	created, err := uc.repo.AddInstallment(ctx, inst)
	if err != nil {
		return nil, err
	}
	return &response.InstallmentResponse{
		ID:                created.ID,
		InstallmentNumber: created.InstallmentNumber,
		DueDays:           created.DueDays,
		Description:       created.Description,
		DocumentType:      created.DocumentType,
		MovementType:      created.MovementType,
		CarrierID:         created.CarrierID,
	}, nil
}

func (uc *CustomerUseCase) ListPaymentConditions(ctx context.Context, onlyActive bool) ([]*response.PaymentConditionResponse, error) {
	pcs, err := uc.repo.ListPaymentConditions(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.PaymentConditionResponse, 0, len(pcs))
	for _, pc := range pcs {
		out = append(out, toPaymentCondResponse(pc))
	}
	return out, nil
}

func toPaymentCondResponse(pc *entity.PaymentCondition) *response.PaymentConditionResponse {
	return &response.PaymentConditionResponse{
		ID:           pc.ID,
		Code:         pc.Code,
		Description:  pc.Description,
		CarrierID:    pc.CarrierID,
		AnalysisType: string(pc.AnalysisType),
		ParcelStart:  string(pc.ParcelStart),
		Expenses:     pc.Expenses,
		AverageTerm:  pc.AverageTerm,
		IsSpecial:    pc.IsSpecial,
		IsRevenue:    pc.IsRevenue,
		IsAtSight:    pc.IsAtSight,
		IsActive:     pc.IsActive,
	}
}

// ─── Sales Tables ─────────────────────────────────────────────────────────────

func (uc *CustomerUseCase) CreateSalesTable(ctx context.Context, dto request.CreateSalesTableDTO) (*response.SalesTableResponse, error) {
	code, err := uc.repo.NextSalesTableCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating sales table code: %w", err)
	}
	st, err := entity.NewSalesTable(code, dto.Description, entity.PriceFormation(dto.PriceFormation))
	if err != nil {
		return nil, err
	}
	st.ValidityStart = dto.ValidityStart
	st.ValidityEnd = dto.ValidityEnd
	st.ToleranceMinPct = dto.ToleranceMinPct
	st.ToleranceMaxPct = dto.ToleranceMaxPct
	if dto.DecimalPlaces > 0 {
		st.DecimalPlaces = dto.DecimalPlaces
	}
	if dto.Composition != "" {
		st.Composition = entity.TableComposition(dto.Composition)
	}
	if dto.TableType != "" {
		st.TableType = entity.TableType(dto.TableType)
	}
	if dto.BaseDate != "" {
		st.BaseDate = entity.BaseDate(dto.BaseDate)
	}
	st.AllowItemsBelowCent = dto.AllowItemsBelowCent
	st.ICMSInterestadualPorDentro = dto.ICMSInterestadualPorDentro
	st.Observation = dto.Observation
	created, err := uc.repo.CreateSalesTable(ctx, st)
	if err != nil {
		return nil, err
	}
	return toSalesTableResponse(created), nil
}

func (uc *CustomerUseCase) ListSalesTables(ctx context.Context, onlyActive bool) ([]*response.SalesTableResponse, error) {
	tables, err := uc.repo.ListSalesTables(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.SalesTableResponse, 0, len(tables))
	for _, t := range tables {
		out = append(out, toSalesTableResponse(t))
	}
	return out, nil
}

func (uc *CustomerUseCase) GetSalesTable(ctx context.Context, code int64) (*response.SalesTableResponse, error) {
	st, err := uc.repo.GetSalesTableByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("sales table not found: %w", err)
	}
	return toSalesTableResponse(st), nil
}

func (uc *CustomerUseCase) UpdateSalesTable(ctx context.Context, dto request.UpdateSalesTableDTO) (*response.SalesTableResponse, error) {
	if dto.Code == 0 {
		return nil, fmt.Errorf("code is required")
	}
	st, err := uc.repo.GetSalesTableByCode(ctx, dto.Code)
	if err != nil {
		return nil, fmt.Errorf("sales table not found: %w", err)
	}
	st.Description = dto.Description
	st.ValidityStart = dto.ValidityStart
	st.ValidityEnd = dto.ValidityEnd
	st.ToleranceMinPct = dto.ToleranceMinPct
	st.ToleranceMaxPct = dto.ToleranceMaxPct
	st.PriceFormation = entity.PriceFormation(dto.PriceFormation)
	if dto.DecimalPlaces > 0 {
		st.DecimalPlaces = dto.DecimalPlaces
	}
	st.IsActive = dto.IsActive
	if dto.Composition != "" {
		st.Composition = entity.TableComposition(dto.Composition)
	}
	if dto.TableType != "" {
		st.TableType = entity.TableType(dto.TableType)
	}
	if dto.BaseDate != "" {
		st.BaseDate = entity.BaseDate(dto.BaseDate)
	}
	st.AllowItemsBelowCent = dto.AllowItemsBelowCent
	st.ICMSInterestadualPorDentro = dto.ICMSInterestadualPorDentro
	st.Observation = dto.Observation
	updated, err := uc.repo.UpdateSalesTable(ctx, st)
	if err != nil {
		return nil, err
	}
	return toSalesTableResponse(updated), nil
}

func toSalesTableResponse(st *entity.SalesTable) *response.SalesTableResponse {
	return &response.SalesTableResponse{
		ID:                         st.ID,
		Code:                       st.Code,
		Description:                st.Description,
		ValidityStart:              st.ValidityStart,
		ValidityEnd:                st.ValidityEnd,
		ToleranceMinPct:            st.ToleranceMinPct,
		ToleranceMaxPct:            st.ToleranceMaxPct,
		PriceFormation:             string(st.PriceFormation),
		DecimalPlaces:              st.DecimalPlaces,
		IsActive:                   st.IsActive,
		Composition:                string(st.Composition),
		TableType:                  string(st.TableType),
		BaseDate:                   string(st.BaseDate),
		AllowItemsBelowCent:        st.AllowItemsBelowCent,
		ICMSInterestadualPorDentro: st.ICMSInterestadualPorDentro,
		Observation:                st.Observation,
	}
}

func (uc *CustomerUseCase) CreateSalesPricePolicy(ctx context.Context, dto request.CreateSalesPricePolicyDTO) (*response.SalesPricePolicyResponse, error) {
	code, err := uc.repo.NextSalesPricePolicyCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating sales price policy code: %w", err)
	}
	p, err := entity.NewSalesPricePolicy(code, dto.Description, entity.SalesCostSource(dto.CostSource))
	if err != nil {
		return nil, err
	}
	if err := uc.applySalesPricePolicyDTO(ctx, p, dtoToUpdatePolicy(dto)); err != nil {
		return nil, err
	}
	created, err := uc.repo.CreateSalesPricePolicy(ctx, p)
	if err != nil {
		return nil, err
	}
	return toSalesPricePolicyResponse(created), nil
}

func (uc *CustomerUseCase) UpdateSalesPricePolicy(ctx context.Context, dto request.UpdateSalesPricePolicyDTO) (*response.SalesPricePolicyResponse, error) {
	if dto.Code == 0 {
		return nil, fmt.Errorf("code is required")
	}
	p, err := uc.repo.GetSalesPricePolicyByCode(ctx, dto.Code)
	if err != nil {
		return nil, fmt.Errorf("sales price policy not found: %w", err)
	}
	if err := uc.applySalesPricePolicyDTO(ctx, p, dto); err != nil {
		return nil, err
	}
	p.IsActive = dto.IsActive
	updated, err := uc.repo.UpdateSalesPricePolicy(ctx, p)
	if err != nil {
		return nil, err
	}
	return toSalesPricePolicyResponse(updated), nil
}

func (uc *CustomerUseCase) GetSalesPricePolicy(ctx context.Context, code int64) (*response.SalesPricePolicyResponse, error) {
	p, err := uc.repo.GetSalesPricePolicyByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toSalesPricePolicyResponse(p), nil
}

func (uc *CustomerUseCase) ListSalesPricePolicies(ctx context.Context, onlyActive bool) ([]*response.SalesPricePolicyResponse, error) {
	policies, err := uc.repo.ListSalesPricePolicies(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.SalesPricePolicyResponse, 0, len(policies))
	for _, p := range policies {
		out = append(out, toSalesPricePolicyResponse(p))
	}
	return out, nil
}

func dtoToUpdatePolicy(dto request.CreateSalesPricePolicyDTO) request.UpdateSalesPricePolicyDTO {
	return request.UpdateSalesPricePolicyDTO{
		Description:    dto.Description,
		CostSource:     dto.CostSource,
		Priority:       dto.Priority,
		Sequence:       dto.Sequence,
		PolicyScope:    dto.PolicyScope,
		PolicyTypes:    dto.PolicyTypes,
		MarkupPct:      dto.MarkupPct,
		MarginPct:      dto.MarginPct,
		MaxMarginPct:   dto.MaxMarginPct,
		IdealMarginPct: dto.IdealMarginPct,
		MarginStepPct:  dto.MarginStepPct,
		ExpensesPct:    dto.ExpensesPct,
		TaxesPct:       dto.TaxesPct,
		FreightPct:     dto.FreightPct,
		CommissionPct:  dto.CommissionPct,
		DiscountPct:    dto.DiscountPct,
		MinMarginPct:   dto.MinMarginPct,
		MaxDiscountPct: dto.MaxDiscountPct,
		IncidencesJSON: dto.IncidencesJSON,
		SalesTableCode: dto.SalesTableCode,
		ValidityStart:  dto.ValidityStart,
		ValidityEnd:    dto.ValidityEnd,
		IsActive:       true,
		Observation:    dto.Observation,
	}
}

func (uc *CustomerUseCase) applySalesPricePolicyDTO(ctx context.Context, p *entity.SalesPricePolicy, dto request.UpdateSalesPricePolicyDTO) error {
	if dto.Description == "" {
		return fmt.Errorf("description is required")
	}
	source := entity.SalesCostSource(dto.CostSource)
	if source == "" {
		source = entity.SalesCostStandardTotal
	}
	if !entity.ValidSalesCostSource(source) {
		return fmt.Errorf("invalid cost_source")
	}
	p.Description = dto.Description
	p.CostSource = source
	p.Priority = dto.Priority
	if p.Priority == 0 {
		p.Priority = 10
	}
	p.Sequence = dto.Sequence
	if p.Sequence == 0 {
		p.Sequence = 10
	}
	p.PolicyScope = dto.PolicyScope
	if p.PolicyScope == "" {
		p.PolicyScope = "PREC"
	}
	if p.PolicyScope != "PREC" && p.PolicyScope != "FPPV" {
		return fmt.Errorf("policy_scope must be PREC or FPPV")
	}
	p.PolicyTypes = dto.PolicyTypes
	p.MarkupPct = dto.MarkupPct
	p.MarginPct = dto.MarginPct
	p.MaxMarginPct = dto.MaxMarginPct
	p.IdealMarginPct = dto.IdealMarginPct
	p.MarginStepPct = dto.MarginStepPct
	p.ExpensesPct = dto.ExpensesPct
	p.TaxesPct = dto.TaxesPct
	p.FreightPct = dto.FreightPct
	p.CommissionPct = dto.CommissionPct
	p.DiscountPct = dto.DiscountPct
	p.MinMarginPct = dto.MinMarginPct
	p.MaxDiscountPct = dto.MaxDiscountPct
	if p.MaxMarginPct > 0 && p.MinMarginPct > p.MaxMarginPct {
		return fmt.Errorf("min_margin_pct cannot be greater than max_margin_pct")
	}
	if p.IdealMarginPct > 0 {
		if p.IdealMarginPct < p.MinMarginPct {
			return fmt.Errorf("ideal_margin_pct cannot be lower than min_margin_pct")
		}
		if p.MaxMarginPct > 0 && p.IdealMarginPct > p.MaxMarginPct {
			return fmt.Errorf("ideal_margin_pct cannot be greater than max_margin_pct")
		}
	}
	p.IncidencesJSON = string(dto.IncidencesJSON)
	if p.IncidencesJSON == "" {
		p.IncidencesJSON = "[]"
	}
	if !json.Valid([]byte(p.IncidencesJSON)) {
		return fmt.Errorf("incidences_json must be a valid JSON document")
	}
	p.ValidityStart = dto.ValidityStart
	p.ValidityEnd = dto.ValidityEnd
	p.Observation = dto.Observation
	p.SalesTableID = nil
	if dto.SalesTableCode != nil {
		st, err := uc.repo.GetSalesTableByCode(ctx, *dto.SalesTableCode)
		if err != nil {
			return fmt.Errorf("sales table not found: %w", err)
		}
		p.SalesTableID = &st.ID
	}
	return nil
}

func toSalesPricePolicyResponse(p *entity.SalesPricePolicy) *response.SalesPricePolicyResponse {
	return &response.SalesPricePolicyResponse{
		ID:             p.ID,
		Code:           p.Code,
		Description:    p.Description,
		CostSource:     string(p.CostSource),
		Priority:       p.Priority,
		Sequence:       p.Sequence,
		PolicyScope:    p.PolicyScope,
		PolicyTypes:    p.PolicyTypes,
		MarkupPct:      p.MarkupPct,
		MarginPct:      p.MarginPct,
		MaxMarginPct:   p.MaxMarginPct,
		IdealMarginPct: p.IdealMarginPct,
		MarginStepPct:  p.MarginStepPct,
		ExpensesPct:    p.ExpensesPct,
		TaxesPct:       p.TaxesPct,
		FreightPct:     p.FreightPct,
		CommissionPct:  p.CommissionPct,
		DiscountPct:    p.DiscountPct,
		MinMarginPct:   p.MinMarginPct,
		MaxDiscountPct: p.MaxDiscountPct,
		IncidencesJSON: json.RawMessage(p.IncidencesJSON),
		SalesTableID:   p.SalesTableID,
		ValidityStart:  p.ValidityStart,
		ValidityEnd:    p.ValidityEnd,
		IsActive:       p.IsActive,
		Observation:    p.Observation,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

// ─── Commercial Policies ─────────────────────────────────────────────────────

func (uc *CustomerUseCase) CreateCommercialPolicy(ctx context.Context, dto request.CreateCommercialPolicyDTO) (*response.CommercialPolicyResponse, error) {
	code, err := uc.repo.NextCommercialPolicyCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating commercial policy code: %w", err)
	}
	p, err := entity.NewCommercialPolicy(code, dto.Description, entity.CommercialPolicyKind(dto.Kind))
	if err != nil {
		return nil, err
	}
	if err := applyCommercialPolicyDTO(p, commercialPolicyCreateToUpdate(dto)); err != nil {
		return nil, err
	}
	created, err := uc.repo.CreateCommercialPolicy(ctx, p)
	if err != nil {
		return nil, err
	}
	return toCommercialPolicyResponse(created), nil
}

func (uc *CustomerUseCase) UpdateCommercialPolicy(ctx context.Context, dto request.UpdateCommercialPolicyDTO) (*response.CommercialPolicyResponse, error) {
	if dto.Code == 0 {
		return nil, fmt.Errorf("code is required")
	}
	p, err := uc.repo.GetCommercialPolicyByCode(ctx, dto.Code)
	if err != nil {
		return nil, fmt.Errorf("commercial policy not found: %w", err)
	}
	if err := applyCommercialPolicyDTO(p, dto); err != nil {
		return nil, err
	}
	p.IsActive = dto.IsActive
	updated, err := uc.repo.UpdateCommercialPolicy(ctx, p)
	if err != nil {
		return nil, err
	}
	return toCommercialPolicyResponse(updated), nil
}

func (uc *CustomerUseCase) GetCommercialPolicy(ctx context.Context, code int64) (*response.CommercialPolicyResponse, error) {
	p, err := uc.repo.GetCommercialPolicyByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toCommercialPolicyResponse(p), nil
}

func (uc *CustomerUseCase) ListCommercialPolicies(ctx context.Context, onlyActive bool, rawKind string) ([]*response.CommercialPolicyResponse, error) {
	var kind *entity.CommercialPolicyKind
	if rawKind != "" {
		k := entity.CommercialPolicyKind(rawKind)
		if !entity.ValidCommercialPolicyKind(k) {
			return nil, fmt.Errorf("invalid kind")
		}
		kind = &k
	}
	policies, err := uc.repo.ListCommercialPolicies(ctx, onlyActive, kind)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CommercialPolicyResponse, 0, len(policies))
	for _, p := range policies {
		out = append(out, toCommercialPolicyResponse(p))
	}
	return out, nil
}

func (uc *CustomerUseCase) AddCommercialPolicySpecificItem(ctx context.Context, dto request.CommercialPolicySpecificItemDTO) (*response.CommercialPolicySpecificItemResponse, error) {
	if dto.PolicyCode == 0 {
		return nil, fmt.Errorf("policy_code is required")
	}
	if dto.ItemCode == nil && dto.ProductLineID == nil && dto.ItemClassification == nil {
		return nil, fmt.Errorf("item_code, product_line_id or item_classification is required")
	}
	p, err := uc.repo.GetCommercialPolicyByCode(ctx, dto.PolicyCode)
	if err != nil {
		return nil, fmt.Errorf("commercial policy not found: %w", err)
	}
	item, err := uc.repo.AddCommercialPolicySpecificItem(ctx, &entity.CommercialPolicySpecificItem{
		PolicyID:           p.ID,
		ItemCode:           dto.ItemCode,
		ItemMask:           dto.ItemMask,
		ProductLineID:      dto.ProductLineID,
		ItemClassification: dto.ItemClassification,
		ValidityStart:      dto.ValidityStart,
		ValidityEnd:        dto.ValidityEnd,
		BlockDiscount:      dto.BlockDiscount,
		BlockSurcharge:     dto.BlockSurcharge,
		IgnoreItemPolicies: dto.IgnoreItemPolicies,
		BlockManualChange:  dto.BlockManualChange,
	})
	if err != nil {
		return nil, err
	}
	return toCommercialPolicySpecificItemResponse(item), nil
}

func (uc *CustomerUseCase) ListCommercialPolicySpecificItems(ctx context.Context, policyCode int64) ([]*response.CommercialPolicySpecificItemResponse, error) {
	items, err := uc.repo.ListCommercialPolicySpecificItems(ctx, policyCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CommercialPolicySpecificItemResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toCommercialPolicySpecificItemResponse(item))
	}
	return out, nil
}

func (uc *CustomerUseCase) EvaluateCommercialPolicies(ctx context.Context, dto request.EvaluateCommercialPoliciesDTO) (*response.CommercialPolicyEvaluationResponse, error) {
	policies, err := uc.repo.ListCommercialPolicies(ctx, true, nil)
	if err != nil {
		return nil, err
	}
	result, err := entity.EvaluateCommercialPolicies(policies, entity.CommercialPolicyContext{
		GrossValue:         dto.GrossValue,
		Quantity:           dto.Quantity,
		CustomerCode:       dto.CustomerCode,
		CustomerTypeID:     dto.CustomerTypeID,
		MarketSegmentID:    dto.MarketSegmentID,
		RegionID:           dto.RegionID,
		SalesTableID:       dto.SalesTableID,
		PaymentConditionID: dto.PaymentConditionID,
		CarrierID:          dto.CarrierID,
		ItemCode:           dto.ItemCode,
		ItemMask:           dto.ItemMask,
		ProductLineID:      dto.ProductLineID,
		ItemClassification: dto.ItemClassification,
	})
	if err != nil {
		return nil, err
	}
	return toCommercialPolicyEvaluationResponse(result), nil
}

func commercialPolicyCreateToUpdate(dto request.CreateCommercialPolicyDTO) request.UpdateCommercialPolicyDTO {
	return request.UpdateCommercialPolicyDTO{
		Description:            dto.Description,
		Kind:                   dto.Kind,
		ChoiceType:             dto.ChoiceType,
		CalcType:               dto.CalcType,
		PercentValue:           dto.PercentValue,
		FixedValue:             dto.FixedValue,
		MaxPercent:             dto.MaxPercent,
		MaxValue:               dto.MaxValue,
		MinGrossValue:          dto.MinGrossValue,
		MaxGrossValue:          dto.MaxGrossValue,
		MinQuantity:            dto.MinQuantity,
		MaxQuantity:            dto.MaxQuantity,
		Priority:               dto.Priority,
		Sequence:               dto.Sequence,
		Stackable:              dto.Stackable,
		RequiresApproval:       dto.RequiresApproval,
		AppliesOnNetValue:      dto.AppliesOnNetValue,
		AllowManualChange:      dto.AllowManualChange,
		AllowHigherValues:      dto.AllowHigherValues,
		UsedInCommission:       dto.UsedInCommission,
		AppliesToItems:         dto.AppliesToItems,
		SubtractCommissionBase: dto.SubtractCommissionBase,
		DataTypesJSON:          dto.DataTypesJSON,
		CommissionDiscountMode: dto.CommissionDiscountMode,
		CustomerCode:           dto.CustomerCode,
		CustomerTypeID:         dto.CustomerTypeID,
		MarketSegmentID:        dto.MarketSegmentID,
		RegionID:               dto.RegionID,
		SalesTableID:           dto.SalesTableID,
		PaymentConditionID:     dto.PaymentConditionID,
		CarrierID:              dto.CarrierID,
		ItemCode:               dto.ItemCode,
		ItemMask:               dto.ItemMask,
		ProductLineID:          dto.ProductLineID,
		ItemClassification:     dto.ItemClassification,
		RuleJSON:               dto.RuleJSON,
		ValidityStart:          dto.ValidityStart,
		ValidityEnd:            dto.ValidityEnd,
		IsActive:               true,
		Observation:            dto.Observation,
	}
}

func applyCommercialPolicyDTO(p *entity.CommercialPolicy, dto request.UpdateCommercialPolicyDTO) error {
	if dto.Description == "" {
		return fmt.Errorf("description is required")
	}
	kind := entity.CommercialPolicyKind(dto.Kind)
	if !entity.ValidCommercialPolicyKind(kind) {
		return fmt.Errorf("invalid kind")
	}
	calcType := entity.CommercialPolicyCalcType(dto.CalcType)
	if calcType == "" {
		calcType = entity.CommercialPolicyPercent
	}
	if !entity.ValidCommercialPolicyCalcType(calcType) {
		return fmt.Errorf("invalid calc_type")
	}
	for name, value := range map[string]float64{
		"percent_value":   dto.PercentValue,
		"fixed_value":     dto.FixedValue,
		"max_percent":     dto.MaxPercent,
		"max_value":       dto.MaxValue,
		"min_gross_value": dto.MinGrossValue,
		"max_gross_value": dto.MaxGrossValue,
		"min_quantity":    dto.MinQuantity,
		"max_quantity":    dto.MaxQuantity,
	} {
		if value < 0 {
			return fmt.Errorf("%s must be >= 0", name)
		}
	}
	if dto.MaxGrossValue > 0 && dto.MinGrossValue > dto.MaxGrossValue {
		return fmt.Errorf("min_gross_value cannot be greater than max_gross_value")
	}
	if dto.MaxQuantity > 0 && dto.MinQuantity > dto.MaxQuantity {
		return fmt.Errorf("min_quantity cannot be greater than max_quantity")
	}
	p.Description = dto.Description
	p.Kind = kind
	choiceType := entity.CommercialPolicyChoiceType(dto.ChoiceType)
	if choiceType == "" {
		choiceType = entity.CommercialPolicyInformation
	}
	if !entity.ValidCommercialPolicyChoiceType(choiceType) {
		return fmt.Errorf("invalid choice_type")
	}
	p.ChoiceType = choiceType
	p.CalcType = calcType
	p.PercentValue = dto.PercentValue
	p.FixedValue = dto.FixedValue
	p.MaxPercent = dto.MaxPercent
	p.MaxValue = dto.MaxValue
	p.MinGrossValue = dto.MinGrossValue
	p.MaxGrossValue = dto.MaxGrossValue
	p.MinQuantity = dto.MinQuantity
	p.MaxQuantity = dto.MaxQuantity
	p.Priority = dto.Priority
	if p.Priority == 0 {
		p.Priority = 10
	}
	p.Sequence = dto.Sequence
	if p.Sequence == 0 {
		p.Sequence = 10
	}
	p.Stackable = dto.Stackable
	p.RequiresApproval = dto.RequiresApproval
	p.AppliesOnNetValue = dto.AppliesOnNetValue
	p.AllowManualChange = dto.AllowManualChange
	p.AllowHigherValues = dto.AllowHigherValues
	p.UsedInCommission = dto.UsedInCommission
	p.AppliesToItems = dto.AppliesToItems
	p.SubtractCommissionBase = dto.SubtractCommissionBase
	p.DataTypesJSON = string(dto.DataTypesJSON)
	if p.DataTypesJSON == "" {
		p.DataTypesJSON = "[]"
	}
	if !json.Valid([]byte(p.DataTypesJSON)) {
		return fmt.Errorf("data_types_json must be a valid JSON document")
	}
	p.CommissionDiscountMode = dto.CommissionDiscountMode
	if p.CommissionDiscountMode == "" {
		p.CommissionDiscountMode = "REAL"
	}
	if p.CommissionDiscountMode != "REAL" && p.CommissionDiscountMode != "NOMINAL" {
		return fmt.Errorf("commission_discount_mode must be REAL or NOMINAL")
	}
	p.CustomerCode = dto.CustomerCode
	p.CustomerTypeID = dto.CustomerTypeID
	p.MarketSegmentID = dto.MarketSegmentID
	p.RegionID = dto.RegionID
	p.SalesTableID = dto.SalesTableID
	p.PaymentConditionID = dto.PaymentConditionID
	p.CarrierID = dto.CarrierID
	p.ItemCode = dto.ItemCode
	p.ItemMask = dto.ItemMask
	p.ProductLineID = dto.ProductLineID
	p.ItemClassification = dto.ItemClassification
	p.RuleJSON = string(dto.RuleJSON)
	if p.RuleJSON == "" {
		p.RuleJSON = "{}"
	}
	if !json.Valid([]byte(p.RuleJSON)) {
		return fmt.Errorf("rule_json must be a valid JSON document")
	}
	p.ValidityStart = dto.ValidityStart
	p.ValidityEnd = dto.ValidityEnd
	p.Observation = dto.Observation
	return nil
}

func (uc *CustomerUseCase) AddCommercialPolicyLine(ctx context.Context, dto request.CommercialPolicyLineDTO) (*response.CommercialPolicyLineResponse, error) {
	if dto.PolicyCode == 0 {
		return nil, fmt.Errorf("policy_code is required")
	}
	if dto.LineNumber == 0 {
		return nil, fmt.Errorf("line_number is required")
	}
	p, err := uc.repo.GetCommercialPolicyByCode(ctx, dto.PolicyCode)
	if err != nil {
		return nil, fmt.Errorf("commercial policy not found: %w", err)
	}
	calcType := entity.CommercialPolicyCalcType(dto.CalcType)
	if calcType == "" {
		calcType = entity.CommercialPolicyPercent
	}
	if !entity.ValidCommercialPolicyCalcType(calcType) {
		return nil, fmt.Errorf("invalid calc_type")
	}
	if dto.PercentValue < 0 || dto.FixedValue < 0 || dto.MinValue < 0 || dto.MaxValue < 0 {
		return nil, fmt.Errorf("line values must be >= 0")
	}
	if dto.MaxValue > 0 && dto.MinValue > dto.MaxValue {
		return nil, fmt.Errorf("min_value cannot be greater than max_value")
	}
	variables := string(dto.VariablesJSON)
	if variables == "" {
		variables = "{}"
	}
	if !json.Valid([]byte(variables)) {
		return nil, fmt.Errorf("variables_json must be a valid JSON document")
	}
	seq := dto.SequenceNumber
	if seq == 0 {
		seq = 1
	}
	line, err := uc.repo.AddCommercialPolicyLine(ctx, &entity.CommercialPolicyLine{
		PolicyID:       p.ID,
		LineNumber:     dto.LineNumber,
		SequenceNumber: seq,
		Description:    dto.Description,
		CalcType:       calcType,
		PercentValue:   dto.PercentValue,
		FixedValue:     dto.FixedValue,
		MinValue:       dto.MinValue,
		MaxValue:       dto.MaxValue,
		VariablesJSON:  variables,
		ValidityStart:  dto.ValidityStart,
		ValidityEnd:    dto.ValidityEnd,
	})
	if err != nil {
		return nil, err
	}
	return toCommercialPolicyLineResponse(line), nil
}

func (uc *CustomerUseCase) ListCommercialPolicyLines(ctx context.Context, policyCode int64) ([]*response.CommercialPolicyLineResponse, error) {
	lines, err := uc.repo.ListCommercialPolicyLines(ctx, policyCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CommercialPolicyLineResponse, 0, len(lines))
	for _, line := range lines {
		out = append(out, toCommercialPolicyLineResponse(line))
	}
	return out, nil
}

func toCommercialPolicyResponse(p *entity.CommercialPolicy) *response.CommercialPolicyResponse {
	out := &response.CommercialPolicyResponse{
		ID:                     p.ID,
		Code:                   p.Code,
		Description:            p.Description,
		Kind:                   string(p.Kind),
		ChoiceType:             string(p.ChoiceType),
		CalcType:               string(p.CalcType),
		PercentValue:           p.PercentValue,
		FixedValue:             p.FixedValue,
		MaxPercent:             p.MaxPercent,
		MaxValue:               p.MaxValue,
		MinGrossValue:          p.MinGrossValue,
		MaxGrossValue:          p.MaxGrossValue,
		MinQuantity:            p.MinQuantity,
		MaxQuantity:            p.MaxQuantity,
		Priority:               p.Priority,
		Sequence:               p.Sequence,
		Stackable:              p.Stackable,
		RequiresApproval:       p.RequiresApproval,
		AppliesOnNetValue:      p.AppliesOnNetValue,
		AllowManualChange:      p.AllowManualChange,
		AllowHigherValues:      p.AllowHigherValues,
		UsedInCommission:       p.UsedInCommission,
		AppliesToItems:         p.AppliesToItems,
		SubtractCommissionBase: p.SubtractCommissionBase,
		DataTypesJSON:          json.RawMessage(p.DataTypesJSON),
		CommissionDiscountMode: p.CommissionDiscountMode,
		CustomerCode:           p.CustomerCode,
		CustomerTypeID:         p.CustomerTypeID,
		MarketSegmentID:        p.MarketSegmentID,
		RegionID:               p.RegionID,
		SalesTableID:           p.SalesTableID,
		PaymentConditionID:     p.PaymentConditionID,
		CarrierID:              p.CarrierID,
		ItemCode:               p.ItemCode,
		ItemMask:               p.ItemMask,
		ProductLineID:          p.ProductLineID,
		ItemClassification:     p.ItemClassification,
		RuleJSON:               json.RawMessage(p.RuleJSON),
		ValidityStart:          p.ValidityStart,
		ValidityEnd:            p.ValidityEnd,
		IsActive:               p.IsActive,
		Observation:            p.Observation,
		CreatedAt:              p.CreatedAt,
		UpdatedAt:              p.UpdatedAt,
	}
	for _, line := range p.Lines {
		out.Lines = append(out.Lines, *toCommercialPolicyLineResponse(line))
	}
	return out
}

func toCommercialPolicyLineResponse(line *entity.CommercialPolicyLine) *response.CommercialPolicyLineResponse {
	return &response.CommercialPolicyLineResponse{
		ID:             line.ID,
		PolicyID:       line.PolicyID,
		LineNumber:     line.LineNumber,
		SequenceNumber: line.SequenceNumber,
		Description:    line.Description,
		CalcType:       string(line.CalcType),
		PercentValue:   line.PercentValue,
		FixedValue:     line.FixedValue,
		MinValue:       line.MinValue,
		MaxValue:       line.MaxValue,
		VariablesJSON:  json.RawMessage(line.VariablesJSON),
		ValidityStart:  line.ValidityStart,
		ValidityEnd:    line.ValidityEnd,
		IsActive:       line.IsActive,
		CreatedAt:      line.CreatedAt,
		UpdatedAt:      line.UpdatedAt,
	}
}

func toCommercialPolicySpecificItemResponse(item *entity.CommercialPolicySpecificItem) *response.CommercialPolicySpecificItemResponse {
	return &response.CommercialPolicySpecificItemResponse{
		ID:                 item.ID,
		PolicyID:           item.PolicyID,
		ItemCode:           item.ItemCode,
		ItemMask:           item.ItemMask,
		ProductLineID:      item.ProductLineID,
		ItemClassification: item.ItemClassification,
		ValidityStart:      item.ValidityStart,
		ValidityEnd:        item.ValidityEnd,
		BlockDiscount:      item.BlockDiscount,
		BlockSurcharge:     item.BlockSurcharge,
		IgnoreItemPolicies: item.IgnoreItemPolicies,
		BlockManualChange:  item.BlockManualChange,
		CreatedAt:          item.CreatedAt,
	}
}

func toCommercialPolicyEvaluationResponse(eval *entity.CommercialPolicyEvaluation) *response.CommercialPolicyEvaluationResponse {
	out := &response.CommercialPolicyEvaluationResponse{
		GrossValue:       eval.GrossValue,
		DiscountValue:    eval.DiscountValue,
		SurchargeValue:   eval.SurchargeValue,
		FreightValue:     eval.FreightValue,
		CommissionValue:  eval.CommissionValue,
		NetValue:         eval.NetValue,
		RequiresApproval: eval.RequiresApproval,
		Effects:          make([]response.CommercialPolicyEffectResponse, 0, len(eval.Effects)),
	}
	for _, effect := range eval.Effects {
		out.Effects = append(out.Effects, response.CommercialPolicyEffectResponse{
			PolicyCode:       effect.PolicyCode,
			PolicyID:         effect.PolicyID,
			Description:      effect.Description,
			Kind:             string(effect.Kind),
			CalcType:         string(effect.CalcType),
			PercentValue:     effect.PercentValue,
			FixedValue:       effect.FixedValue,
			AppliedValue:     effect.AppliedValue,
			RequiresApproval: effect.RequiresApproval,
			Stackable:        effect.Stackable,
		})
	}
	return out
}

// ─── Invoice Types ────────────────────────────────────────────────────────────

func (uc *CustomerUseCase) CreateInvoiceType(ctx context.Context, dto request.CreateInvoiceTypeDTO) (*response.InvoiceTypeResponse, error) {
	code, err := uc.repo.NextInvoiceTypeCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating invoice type code: %w", err)
	}
	it, err := entity.NewInvoiceType(code, dto.Description, entity.InvoiceTypeKind(dto.Type))
	if err != nil {
		return nil, err
	}
	applyInvoiceTypeDTOToEntity(dto, it)
	created, err := uc.repo.CreateInvoiceType(ctx, it)
	if err != nil {
		return nil, err
	}
	return toInvoiceTypeResponse(created), nil
}

func (uc *CustomerUseCase) ListInvoiceTypes(ctx context.Context, onlyActive bool) ([]*response.InvoiceTypeResponse, error) {
	types, err := uc.repo.ListInvoiceTypes(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.InvoiceTypeResponse, 0, len(types))
	for _, t := range types {
		out = append(out, toInvoiceTypeResponse(t))
	}
	return out, nil
}

func (uc *CustomerUseCase) UpdateInvoiceType(ctx context.Context, dto request.UpdateInvoiceTypeDTO) (*response.InvoiceTypeResponse, error) {
	if dto.Code == 0 {
		return nil, fmt.Errorf("code is required")
	}
	if dto.Description == "" {
		return nil, fmt.Errorf("description is required")
	}
	it := &entity.InvoiceType{
		Code:        dto.Code,
		Description: dto.Description,
		Type:        entity.InvoiceTypeKind(dto.Type),
		IsActive:    dto.IsActive,
	}
	// reuse the same shared helper — field sets are identical between Create and Update DTOs
	createLike := request.CreateInvoiceTypeDTO{
		Description:              dto.Description,
		Type:                     dto.Type,
		StockMovement:            dto.StockMovement,
		ICMSType:                 dto.ICMSType,
		ICMSPct:                  dto.ICMSPct,
		ICMSReductionPct:         dto.ICMSReductionPct,
		IPIPct:                   dto.IPIPct,
		PISPct:                   dto.PISPct,
		COFINSPct:                dto.COFINSPct,
		ISSQNPct:                 dto.ISSQNPct,
		IRPct:                    dto.IRPct,
		CSLLPct:                  dto.CSLLPct,
		INSSPct:                  dto.INSSPct,
		GeneratesRevenue:         dto.GeneratesRevenue,
		UpdatesInventory:         dto.UpdatesInventory,
		GeneratesFinancialTitle:  dto.GeneratesFinancialTitle,
		ConsidersGoals:           dto.ConsidersGoals,
		CalcSubstitutionTax:      dto.CalcSubstitutionTax,
		CalcICMSDeferral:         dto.CalcICMSDeferral,
		CalcPISCOFINS:            dto.CalcPISCOFINS,
		CalcDIFAL:                dto.CalcDIFAL,
		RequiresSalesOrder:       dto.RequiresSalesOrder,
		ListsFiscalBooks:         dto.ListsFiscalBooks,
		ModelNF:                  dto.ModelNF,
		CSTICMS:                  dto.CSTICMS,
		CSOSNTICMS:               dto.CSOSNTICMS,
		CSTIPI:                   dto.CSTIPI,
		CSTPIS:                   dto.CSTPIS,
		CSTCOFINS:                dto.CSTCOFINS,
		BaixaPedido:              dto.BaixaPedido,
		GeraTituloDev:            dto.GeraTituloDev,
		ExigeSuframa:             dto.ExigeSuframa,
		IRPctPresumption:         dto.IRPctPresumption,
		CSLLPctPresumption:       dto.CSLLPctPresumption,
		DescriptionNF:            dto.DescriptionNF,
		ImpostosNFe:              dto.ImpostosNFe,
		CFOPId:                   dto.CFOPId,
		DispositivoLegalIPIId:    dto.DispositivoLegalIPIId,
		DispositivoLegalICMSId:   dto.DispositivoLegalICMSId,
		DispositivoLegalICMSSTId: dto.DispositivoLegalICMSSTId,
		DispositivoLegalPISId:    dto.DispositivoLegalPISId,
		DispositivoLegalCOFINSId: dto.DispositivoLegalCOFINSId,
		HierarchyIPI:             dto.HierarchyIPI,
		HierarchyICMS:            dto.HierarchyICMS,
		HierarchyICMSST:          dto.HierarchyICMSST,
		HierarchyPIS:             dto.HierarchyPIS,
		HierarchyCOFINS:          dto.HierarchyCOFINS,
		IPITransferSalesTableId:  dto.IPITransferSalesTableId,
		ListaValorContabil:       dto.ListaValorContabil,
		ListaRegistroSaida:       dto.ListaRegistroSaida,
		ListaICMSIPI:             dto.ListaICMSIPI,
		SintegraSpedFiscal:       dto.SintegraSpedFiscal,
		CalcFomentar:             dto.CalcFomentar,
		ExcecaoFomentar:          dto.ExcecaoFomentar,
		CompRessRetST:            dto.CompRessRetST,
		CalcReducao:              dto.CalcReducao,
		ComplementoItens:         dto.ComplementoItens,
		BuscaTipoNF:              dto.BuscaTipoNF,
		ICMSSTUltEntrada:         dto.ICMSSTUltEntrada,
		SomenteConsultaLotes:     dto.SomenteConsultaLotes,
		CalcImpIBPT:              dto.CalcImpIBPT,
		CredPresumidoICMS:        dto.CredPresumidoICMS,
		CIAP:                     dto.CIAP,
		VlrAgregadoBaseSubst:     dto.VlrAgregadoBaseSubst,
		ContratoFacon:            dto.ContratoFacon,
		DescICMSLicitacoes:       dto.DescICMSLicitacoes,
		Sisdeclara:               dto.Sisdeclara,
		CodClasTrib:              dto.CodClasTrib,
		CodClasTribTribReg:       dto.CodClasTribTribReg,
		CodMotivoRestCompICMSST:  dto.CodMotivoRestCompICMSST,
		CodBeneficioFiscal:       dto.CodBeneficioFiscal,
	}
	applyInvoiceTypeDTOToEntity(createLike, it)
	updated, err := uc.repo.UpdateInvoiceType(ctx, it)
	if err != nil {
		return nil, err
	}
	return toInvoiceTypeResponse(updated), nil
}

func toInvoiceTypeResponse(it *entity.InvoiceType) *response.InvoiceTypeResponse {
	var impostosNFe *string
	if it.ImpostosNFe != nil {
		s := string(*it.ImpostosNFe)
		impostosNFe = &s
	}
	return &response.InvoiceTypeResponse{
		ID:                       it.ID,
		Code:                     it.Code,
		Description:              it.Description,
		Type:                     string(it.Type),
		StockMovement:            string(it.StockMovement),
		ICMSType:                 string(it.ICMSType),
		ICMSPct:                  it.ICMSPct,
		ICMSReductionPct:         it.ICMSReductionPct,
		IPIPct:                   it.IPIPct,
		PISPct:                   it.PISPct,
		COFINSPct:                it.COFINSPct,
		ISSQNPct:                 it.ISSQNPct,
		IRPct:                    it.IRPct,
		CSLLPct:                  it.CSLLPct,
		INSSPct:                  it.INSSPct,
		GeneratesRevenue:         it.GeneratesRevenue,
		UpdatesInventory:         it.UpdatesInventory,
		GeneratesFinancialTitle:  it.GeneratesFinancialTitle,
		ConsidersGoals:           it.ConsidersGoals,
		CalcSubstitutionTax:      it.CalcSubstitutionTax,
		CalcICMSDeferral:         it.CalcICMSDeferral,
		CalcPISCOFINS:            it.CalcPISCOFINS,
		CalcDIFAL:                it.CalcDIFAL,
		RequiresSalesOrder:       it.RequiresSalesOrder,
		ListsFiscalBooks:         it.ListsFiscalBooks,
		IsActive:                 it.IsActive,
		ModelNF:                  it.ModelNF,
		CSTICMS:                  it.CSTICMS,
		CSOSNTICMS:               it.CSOSNTICMS,
		CSTIPI:                   it.CSTIPI,
		CSTPIS:                   it.CSTPIS,
		CSTCOFINS:                it.CSTCOFINS,
		BaixaPedido:              it.BaixaPedido,
		GeraTituloDev:            it.GeraTituloDev,
		ExigeSuframa:             it.ExigeSuframa,
		IRPctPresumption:         it.IRPctPresumption,
		CSLLPctPresumption:       it.CSLLPctPresumption,
		DescriptionNF:            it.DescriptionNF,
		ImpostosNFe:              impostosNFe,
		CFOPId:                   it.CFOPId,
		DispositivoLegalIPIId:    it.DispositivoLegalIPIId,
		DispositivoLegalICMSId:   it.DispositivoLegalICMSId,
		DispositivoLegalICMSSTId: it.DispositivoLegalICMSSTId,
		DispositivoLegalPISId:    it.DispositivoLegalPISId,
		DispositivoLegalCOFINSId: it.DispositivoLegalCOFINSId,
		HierarchyIPI:             it.HierarchyIPI,
		HierarchyICMS:            it.HierarchyICMS,
		HierarchyICMSST:          it.HierarchyICMSST,
		HierarchyPIS:             it.HierarchyPIS,
		HierarchyCOFINS:          it.HierarchyCOFINS,
		IPITransferSalesTableId:  it.IPITransferSalesTableId,
		ListaValorContabil:       it.ListaValorContabil,
		ListaRegistroSaida:       it.ListaRegistroSaida,
		ListaICMSIPI:             it.ListaICMSIPI,
		SintegraSpedFiscal:       it.SintegraSpedFiscal,
		CalcFomentar:             it.CalcFomentar,
		ExcecaoFomentar:          it.ExcecaoFomentar,
		CompRessRetST:            it.CompRessRetST,
		CalcReducao:              it.CalcReducao,
		ComplementoItens:         it.ComplementoItens,
		BuscaTipoNF:              it.BuscaTipoNF,
		ICMSSTUltEntrada:         it.ICMSSTUltEntrada,
		SomenteConsultaLotes:     it.SomenteConsultaLotes,
		CalcImpIBPT:              it.CalcImpIBPT,
		CredPresumidoICMS:        it.CredPresumidoICMS,
		CIAP:                     it.CIAP,
		VlrAgregadoBaseSubst:     it.VlrAgregadoBaseSubst,
		ContratoFacon:            it.ContratoFacon,
		DescICMSLicitacoes:       it.DescICMSLicitacoes,
		Sisdeclara:               it.Sisdeclara,
		CodClasTrib:              it.CodClasTrib,
		CodClasTribTribReg:       it.CodClasTribTribReg,
		CodMotivoRestCompICMSST:  it.CodMotivoRestCompICMSST,
		CodBeneficioFiscal:       it.CodBeneficioFiscal,
		CreatedAt:                it.CreatedAt,
	}
}

func applyInvoiceTypeDTOToEntity(dto request.CreateInvoiceTypeDTO, it *entity.InvoiceType) {
	it.StockMovement = entity.InvoiceStock(dto.StockMovement)
	it.ICMSType = entity.InvoiceICMSType(dto.ICMSType)
	it.ICMSPct = dto.ICMSPct
	it.ICMSReductionPct = dto.ICMSReductionPct
	it.IPIPct = dto.IPIPct
	it.PISPct = dto.PISPct
	it.COFINSPct = dto.COFINSPct
	it.ISSQNPct = dto.ISSQNPct
	it.IRPct = dto.IRPct
	it.CSLLPct = dto.CSLLPct
	it.INSSPct = dto.INSSPct
	it.GeneratesRevenue = dto.GeneratesRevenue
	it.UpdatesInventory = dto.UpdatesInventory
	it.GeneratesFinancialTitle = dto.GeneratesFinancialTitle
	it.ConsidersGoals = dto.ConsidersGoals
	it.CalcSubstitutionTax = dto.CalcSubstitutionTax
	it.CalcICMSDeferral = dto.CalcICMSDeferral
	it.CalcPISCOFINS = dto.CalcPISCOFINS
	it.CalcDIFAL = dto.CalcDIFAL
	it.RequiresSalesOrder = dto.RequiresSalesOrder
	it.ListsFiscalBooks = dto.ListsFiscalBooks
	it.ModelNF = dto.ModelNF
	it.CSTICMS = dto.CSTICMS
	it.CSOSNTICMS = dto.CSOSNTICMS
	it.CSTIPI = dto.CSTIPI
	it.CSTPIS = dto.CSTPIS
	it.CSTCOFINS = dto.CSTCOFINS
	it.BaixaPedido = dto.BaixaPedido
	it.GeraTituloDev = dto.GeraTituloDev
	it.ExigeSuframa = dto.ExigeSuframa
	it.IRPctPresumption = dto.IRPctPresumption
	it.CSLLPctPresumption = dto.CSLLPctPresumption
	it.DescriptionNF = dto.DescriptionNF
	if dto.ImpostosNFe != nil {
		v := entity.ImpostosNFe(*dto.ImpostosNFe)
		it.ImpostosNFe = &v
	}
	it.CFOPId = dto.CFOPId
	it.DispositivoLegalIPIId = dto.DispositivoLegalIPIId
	it.DispositivoLegalICMSId = dto.DispositivoLegalICMSId
	it.DispositivoLegalICMSSTId = dto.DispositivoLegalICMSSTId
	it.DispositivoLegalPISId = dto.DispositivoLegalPISId
	it.DispositivoLegalCOFINSId = dto.DispositivoLegalCOFINSId
	it.HierarchyIPI = dto.HierarchyIPI
	it.HierarchyICMS = dto.HierarchyICMS
	it.HierarchyICMSST = dto.HierarchyICMSST
	it.HierarchyPIS = dto.HierarchyPIS
	it.HierarchyCOFINS = dto.HierarchyCOFINS
	it.IPITransferSalesTableId = dto.IPITransferSalesTableId
	it.ListaValorContabil = dto.ListaValorContabil
	it.ListaRegistroSaida = dto.ListaRegistroSaida
	it.ListaICMSIPI = dto.ListaICMSIPI
	it.SintegraSpedFiscal = dto.SintegraSpedFiscal
	it.CalcFomentar = dto.CalcFomentar
	it.ExcecaoFomentar = dto.ExcecaoFomentar
	it.CompRessRetST = dto.CompRessRetST
	it.CalcReducao = dto.CalcReducao
	it.ComplementoItens = dto.ComplementoItens
	it.BuscaTipoNF = dto.BuscaTipoNF
	it.ICMSSTUltEntrada = dto.ICMSSTUltEntrada
	it.SomenteConsultaLotes = dto.SomenteConsultaLotes
	it.CalcImpIBPT = dto.CalcImpIBPT
	it.CredPresumidoICMS = dto.CredPresumidoICMS
	it.CIAP = dto.CIAP
	it.VlrAgregadoBaseSubst = dto.VlrAgregadoBaseSubst
	it.ContratoFacon = dto.ContratoFacon
	it.DescICMSLicitacoes = dto.DescICMSLicitacoes
	it.Sisdeclara = dto.Sisdeclara
	it.CodClasTrib = dto.CodClasTrib
	it.CodClasTribTribReg = dto.CodClasTribTribReg
	it.CodMotivoRestCompICMSST = dto.CodMotivoRestCompICMSST
	it.CodBeneficioFiscal = dto.CodBeneficioFiscal
}

// ─── Sales Table Prices ───────────────────────────────────────────────────────

func (uc *CustomerUseCase) CreateSalesTablePrice(ctx context.Context, dto request.CreateSalesTablePriceDTO) (*response.SalesTablePriceResponse, error) {
	var st *entity.SalesTable
	if dto.SalesTableID == 0 {
		if dto.SalesTableCode == 0 {
			return nil, fmt.Errorf("sales_table_code is required")
		}
		var err error
		st, err = uc.repo.GetSalesTableByCode(ctx, dto.SalesTableCode)
		if err != nil {
			return nil, fmt.Errorf("sales table not found: %w", err)
		}
		dto.SalesTableID = st.ID
	} else {
		var err error
		st, err = uc.repo.GetSalesTableByID(ctx, dto.SalesTableID)
		if err != nil {
			return nil, fmt.Errorf("sales table not found: %w", err)
		}
	}
	if dto.ItemCode == "" {
		return nil, fmt.Errorf("item_code is required")
	}
	if dto.Price < 0 {
		return nil, fmt.Errorf("price must be >= 0")
	}
	if err := validateManualSalesTablePrice(st, dto.Price); err != nil {
		return nil, err
	}
	if dto.Situation != "" && !validPriceSituations[dto.Situation] {
		return nil, fmt.Errorf("situation must be ATIVO, INATIVO or PROMOCIONAL")
	}
	if dto.Situation == "" {
		dto.Situation = "ATIVO"
	}
	p := &entity.SalesTablePrice{
		SalesTableID:  dto.SalesTableID,
		ItemCode:      dto.ItemCode,
		Price:         dto.Price,
		UME:           dto.UME,
		UMC:           dto.UMC,
		PriceConv:     dto.PriceConv,
		Formula:       dto.Formula,
		Situation:     entity.PriceSituation(dto.Situation),
		Blocked:       dto.Blocked,
		Observation:   dto.Observation,
		ProductLineID: dto.ProductLineID,
		ItemMask:      dto.ItemMask,
	}
	created, err := uc.repo.CreateSalesTablePrice(ctx, p)
	if err != nil {
		return nil, err
	}
	return toSalesTablePriceResponse(created), nil
}

var validPriceSituations = map[string]bool{"ATIVO": true, "INATIVO": true, "PROMOCIONAL": true}

func validateManualSalesTablePrice(st *entity.SalesTable, price float64) error {
	if st == nil {
		return nil
	}
	switch st.PriceFormation {
	case entity.PriceCustoMedio, entity.PriceCustoStandardTotal, entity.PriceCustoStandardMaterial:
		return fmt.Errorf("sales table price cannot be manually maintained when price_formation is %s", st.PriceFormation)
	}
	if price < 0.01 && !st.AllowItemsBelowCent {
		return fmt.Errorf("sales table does not allow item prices below 0.01")
	}
	return nil
}

func (uc *CustomerUseCase) UpdateSalesTablePrice(ctx context.Context, dto request.UpdateSalesTablePriceDTO) (*response.SalesTablePriceResponse, error) {
	if dto.ID == 0 {
		return nil, fmt.Errorf("id is required")
	}
	if dto.Price < 0 {
		return nil, fmt.Errorf("price must be >= 0")
	}
	current, err := uc.repo.GetSalesTablePriceByID(ctx, dto.ID)
	if err != nil {
		return nil, err
	}
	st, err := uc.repo.GetSalesTableByID(ctx, current.SalesTableID)
	if err != nil {
		return nil, fmt.Errorf("sales table not found: %w", err)
	}
	if err := validateManualSalesTablePrice(st, dto.Price); err != nil {
		return nil, err
	}
	if dto.Situation != "" && !validPriceSituations[dto.Situation] {
		return nil, fmt.Errorf("situation must be ATIVO, INATIVO or PROMOCIONAL")
	}
	if dto.Situation == "" {
		dto.Situation = "ATIVO"
	}
	p := &entity.SalesTablePrice{
		ID:            dto.ID,
		Price:         dto.Price,
		UME:           dto.UME,
		UMC:           dto.UMC,
		PriceConv:     dto.PriceConv,
		Formula:       dto.Formula,
		Situation:     entity.PriceSituation(dto.Situation),
		Blocked:       dto.Blocked,
		Observation:   dto.Observation,
		ProductLineID: dto.ProductLineID,
		ItemMask:      dto.ItemMask,
	}
	updated, err := uc.repo.UpdateSalesTablePrice(ctx, p)
	if err != nil {
		return nil, err
	}
	return toSalesTablePriceResponse(updated), nil
}

func (uc *CustomerUseCase) GetSalesTablePrice(ctx context.Context, salesTableID int64, itemCode string) (*response.SalesTablePriceResponse, error) {
	p, err := uc.repo.GetSalesTablePrice(ctx, salesTableID, itemCode)
	if err != nil {
		return nil, err
	}
	return toSalesTablePriceResponse(p), nil
}

func (uc *CustomerUseCase) GetSalesTablePriceByCode(ctx context.Context, salesTableCode int64, itemCode string) (*response.SalesTablePriceResponse, error) {
	st, err := uc.repo.GetSalesTableByCode(ctx, salesTableCode)
	if err != nil {
		return nil, fmt.Errorf("sales table not found: %w", err)
	}
	return uc.GetSalesTablePrice(ctx, st.ID, itemCode)
}

func (uc *CustomerUseCase) ListSalesTablePrices(ctx context.Context, salesTableID int64) ([]*response.SalesTablePriceResponse, error) {
	prices, err := uc.repo.ListSalesTablePrices(ctx, salesTableID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.SalesTablePriceResponse, 0, len(prices))
	for _, p := range prices {
		out = append(out, toSalesTablePriceResponse(p))
	}
	return out, nil
}

func (uc *CustomerUseCase) ListSalesTablePricesByCode(ctx context.Context, salesTableCode int64) ([]*response.SalesTablePriceResponse, error) {
	st, err := uc.repo.GetSalesTableByCode(ctx, salesTableCode)
	if err != nil {
		return nil, fmt.Errorf("sales table not found: %w", err)
	}
	return uc.ListSalesTablePrices(ctx, st.ID)
}

func (uc *CustomerUseCase) PriceSalesItem(ctx context.Context, dto request.PriceSalesItemDTO) (*response.SalesItemPricingResponse, error) {
	if dto.SalesTableCode == 0 {
		return nil, fmt.Errorf("sales_table_code is required")
	}
	if dto.ItemCode == "" {
		return nil, fmt.Errorf("item_code is required")
	}
	if dto.Quantity <= 0 {
		dto.Quantity = 1
	}
	st, err := uc.repo.GetSalesTableByCode(ctx, dto.SalesTableCode)
	if err != nil {
		return nil, fmt.Errorf("sales table not found: %w", err)
	}
	if err := validateSalesTableForPricing(st, time.Now()); err != nil {
		return nil, err
	}
	p, err := uc.repo.GetSalesTablePrice(ctx, st.ID, dto.ItemCode)
	if err != nil {
		return nil, err
	}
	if p.Blocked {
		return nil, fmt.Errorf("sales table price is blocked")
	}
	if p.Situation == entity.PriceSituationInativo {
		return nil, fmt.Errorf("sales table price is inactive")
	}
	return &response.SalesItemPricingResponse{
		SalesTableCode: st.Code,
		SalesTableID:   st.ID,
		ItemCode:       p.ItemCode,
		UnitPrice:      p.Price,
		Quantity:       dto.Quantity,
		TotalGross:     p.Price * dto.Quantity,
		Source:         "SALES_TABLE",
		Situation:      string(p.Situation),
		Blocked:        p.Blocked,
		UME:            p.UME,
		UMC:            p.UMC,
		Formula:        p.Formula,
	}, nil
}

func (uc *CustomerUseCase) FormSalesPrice(ctx context.Context, dto request.FormSalesPriceDTO) (*response.SalesPriceFormationResponse, error) {
	decimalPlaces := int16(2)
	if dto.PolicyCode != nil {
		p, err := uc.repo.GetSalesPricePolicyByCode(ctx, *dto.PolicyCode)
		if err != nil {
			return nil, fmt.Errorf("sales price policy not found: %w", err)
		}
		if err := validateSalesPricePolicyForPricing(p, time.Now()); err != nil {
			return nil, err
		}
		dto.MarkupPct = p.MarkupPct
		dto.MarginPct = p.MarginPct
		if p.IdealMarginPct > 0 {
			dto.MarginPct = p.IdealMarginPct
		}
		dto.ExpensesPct = p.ExpensesPct
		dto.TaxesPct = p.TaxesPct
		dto.FreightPct = p.FreightPct
		dto.CommissionPct = p.CommissionPct
		dto.DiscountPct = p.DiscountPct
		if dto.BaseCost == 0 && p.CostSource != entity.SalesCostInformed && dto.ItemCode != "" {
			itemCode, err := strconv.ParseInt(dto.ItemCode, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("item_code must be numeric when resolving cost from policy")
			}
			baseCost, _, err := uc.repo.ResolveSalesCost(ctx, itemCode, "", p.CostSource, nil)
			if err != nil {
				return nil, err
			}
			dto.BaseCost = baseCost
		}
	}
	if dto.SalesTableCode != 0 {
		st, err := uc.repo.GetSalesTableByCode(ctx, dto.SalesTableCode)
		if err != nil {
			return nil, fmt.Errorf("sales table not found: %w", err)
		}
		decimalPlaces = st.DecimalPlaces
		if decimalPlaces < 0 {
			decimalPlaces = 2
		}
	}
	formed, err := entity.FormSalesPrice(entity.SalesPriceFormationInput{
		BaseCost:      dto.BaseCost,
		MarkupPct:     dto.MarkupPct,
		MarginPct:     dto.MarginPct,
		ExpensesPct:   dto.ExpensesPct,
		TaxesPct:      dto.TaxesPct,
		FreightPct:    dto.FreightPct,
		CommissionPct: dto.CommissionPct,
		DiscountPct:   dto.DiscountPct,
		DecimalPlaces: decimalPlaces,
	})
	if err != nil {
		return nil, err
	}
	return &response.SalesPriceFormationResponse{
		SalesTableCode:          dto.SalesTableCode,
		ItemCode:                dto.ItemCode,
		BaseCost:                formed.BaseCost,
		SuggestedPrice:          formed.SuggestedPrice,
		MarkupPct:               formed.MarkupPct,
		MarginPct:               formed.MarginPct,
		ExpensesPct:             formed.ExpensesPct,
		TaxesPct:                formed.TaxesPct,
		FreightPct:              formed.FreightPct,
		CommissionPct:           formed.CommissionPct,
		DiscountPct:             formed.DiscountPct,
		ContributionMarginPct:   formed.ContributionMarginPct,
		ContributionMarginValue: formed.ContributionMarginValue,
		DecimalPlaces:           decimalPlaces,
	}, nil
}

func (uc *CustomerUseCase) GenerateSalesTablePrices(ctx context.Context, dto request.GenerateSalesTablePricesDTO) (*response.GenerateSalesTablePricesResponse, error) {
	if dto.SalesTableCode == 0 {
		return nil, fmt.Errorf("sales_table_code is required")
	}
	if dto.PolicyCode == 0 {
		return nil, fmt.Errorf("policy_code is required")
	}
	if len(dto.ItemCodes) == 0 {
		return nil, fmt.Errorf("item_codes is required")
	}
	st, err := uc.repo.GetSalesTableByCode(ctx, dto.SalesTableCode)
	if err != nil {
		return nil, fmt.Errorf("sales table not found: %w", err)
	}
	if err := validateSalesTableForPricing(st, time.Now()); err != nil {
		return nil, err
	}
	policy, err := uc.repo.GetSalesPricePolicyByCode(ctx, dto.PolicyCode)
	if err != nil {
		return nil, fmt.Errorf("sales price policy not found: %w", err)
	}
	if err := validateSalesPricePolicyForPricing(policy, time.Now()); err != nil {
		return nil, err
	}

	out := &response.GenerateSalesTablePricesResponse{
		SalesTableCode: st.Code,
		PolicyCode:     policy.Code,
		Generated:      make([]response.GeneratedSalesTablePriceResponse, 0, len(dto.ItemCodes)),
	}
	for _, rawItemCode := range dto.ItemCodes {
		itemCode, err := strconv.ParseInt(rawItemCode, 10, 64)
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("item %s ignored: item_code must be numeric to resolve cost", rawItemCode))
			continue
		}
		baseCost, costSource, err := uc.repo.ResolveSalesCost(ctx, itemCode, "", policy.CostSource, dto.WarehouseID)
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("item %s ignored: %v", rawItemCode, err))
			continue
		}
		formed, err := entity.FormSalesPrice(entity.SalesPriceFormationInput{
			BaseCost:      baseCost,
			MarkupPct:     policy.MarkupPct,
			MarginPct:     effectivePolicyMargin(policy),
			ExpensesPct:   policy.ExpensesPct,
			TaxesPct:      policy.TaxesPct,
			FreightPct:    policy.FreightPct,
			CommissionPct: policy.CommissionPct,
			DiscountPct:   policy.DiscountPct,
			DecimalPlaces: st.DecimalPlaces,
		})
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("item %s ignored: %v", rawItemCode, err))
			continue
		}
		priceRow := &entity.SalesTablePrice{
			SalesTableID: st.ID,
			ItemCode:     rawItemCode,
			Price:        formed.SuggestedPrice,
			PriceConv:    formed.SuggestedPrice,
			Situation:    entity.PriceSituationAtivo,
		}
		formula := fmt.Sprintf("POLICY:%d/%s", policy.Code, costSource)
		priceRow.Formula = &formula
		saved, oldPrice, err := uc.repo.UpsertSalesTablePrice(ctx, priceRow)
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("item %s ignored: %v", rawItemCode, err))
			continue
		}
		base := baseCost
		policyCode := policy.Code
		history, err := uc.repo.CreateSalesTablePriceHistory(ctx, &entity.SalesTablePriceHistory{
			SalesTablePriceID: &saved.ID,
			SalesTableID:      st.ID,
			SalesTableCode:    st.Code,
			ItemCode:          rawItemCode,
			OldPrice:          oldPrice,
			NewPrice:          saved.Price,
			BaseCost:          &base,
			Source:            "POLICY",
			PolicyCode:        &policyCode,
			Reason:            dto.Reason,
		})
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("item %s repriced without history: %v", rawItemCode, err))
		}
		var historyID int64
		if history != nil {
			historyID = history.ID
		}
		out.Generated = append(out.Generated, response.GeneratedSalesTablePriceResponse{
			ItemCode:   rawItemCode,
			BaseCost:   baseCost,
			CostSource: costSource,
			OldPrice:   oldPrice,
			Price:      saved.Price,
			HistoryID:  historyID,
			PriceRow:   toSalesTablePriceResponse(saved),
		})
	}
	return out, nil
}

func effectivePolicyMargin(policy *entity.SalesPricePolicy) float64 {
	if policy.IdealMarginPct > 0 {
		return policy.IdealMarginPct
	}
	return policy.MarginPct
}

func (uc *CustomerUseCase) ListSalesTablePriceHistory(ctx context.Context, salesTableCode int64, itemCode *string) ([]*response.SalesTablePriceHistoryResponse, error) {
	items, err := uc.repo.ListSalesTablePriceHistory(ctx, salesTableCode, itemCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.SalesTablePriceHistoryResponse, 0, len(items))
	for _, h := range items {
		out = append(out, toSalesTablePriceHistoryResponse(h))
	}
	return out, nil
}

func toSalesTablePriceHistoryResponse(h *entity.SalesTablePriceHistory) *response.SalesTablePriceHistoryResponse {
	return &response.SalesTablePriceHistoryResponse{
		ID:                h.ID,
		SalesTablePriceID: h.SalesTablePriceID,
		SalesTableID:      h.SalesTableID,
		SalesTableCode:    h.SalesTableCode,
		ItemCode:          h.ItemCode,
		OldPrice:          h.OldPrice,
		NewPrice:          h.NewPrice,
		BaseCost:          h.BaseCost,
		Source:            h.Source,
		PolicyCode:        h.PolicyCode,
		Reason:            h.Reason,
		CreatedAt:         h.CreatedAt,
	}
}

func validateSalesTableForPricing(st *entity.SalesTable, now time.Time) error {
	if !st.IsActive {
		return fmt.Errorf("sales table is inactive")
	}
	if st.ValidityStart != nil && now.Before(*st.ValidityStart) {
		return fmt.Errorf("sales table validity has not started")
	}
	if st.ValidityEnd != nil && now.After(st.ValidityEnd.Add(24*time.Hour)) {
		return fmt.Errorf("sales table validity has expired")
	}
	return nil
}

func validateSalesPricePolicyForPricing(p *entity.SalesPricePolicy, now time.Time) error {
	if !p.IsActive {
		return fmt.Errorf("sales price policy is inactive")
	}
	if p.ValidityStart != nil && now.Before(*p.ValidityStart) {
		return fmt.Errorf("sales price policy validity has not started")
	}
	if p.ValidityEnd != nil && now.After(p.ValidityEnd.Add(24*time.Hour)) {
		return fmt.Errorf("sales price policy validity has expired")
	}
	margin := effectivePolicyMargin(p)
	if margin < p.MinMarginPct {
		return fmt.Errorf("sales price policy margin is lower than min_margin_pct")
	}
	if p.MaxMarginPct > 0 && margin > p.MaxMarginPct {
		return fmt.Errorf("sales price policy margin is greater than max_margin_pct")
	}
	return nil
}

func (uc *CustomerUseCase) DeleteSalesTablePrice(ctx context.Context, id int64) error {
	return uc.repo.DeleteSalesTablePrice(ctx, id)
}

func toSalesTablePriceResponse(p *entity.SalesTablePrice) *response.SalesTablePriceResponse {
	return &response.SalesTablePriceResponse{
		ID:            p.ID,
		SalesTableID:  p.SalesTableID,
		ItemCode:      p.ItemCode,
		Price:         p.Price,
		UME:           p.UME,
		UMC:           p.UMC,
		PriceConv:     p.PriceConv,
		Formula:       p.Formula,
		Situation:     string(p.Situation),
		Blocked:       p.Blocked,
		Observation:   p.Observation,
		ProductLineID: p.ProductLineID,
		ItemMask:      p.ItemMask,
		CreatedAt:     p.CreatedAt,
	}
}

// ─── Tax Types ────────────────────────────────────────────────────────────────

func (uc *CustomerUseCase) CreateTaxType(ctx context.Context, dto request.CreateTaxTypeDTO) (*response.TaxTypeResponse, error) {
	code, err := uc.repo.NextTaxTypeCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating tax type code: %w", err)
	}
	tt, err := entity.NewTaxType(code, dto.Description)
	if err != nil {
		return nil, err
	}
	tt.IPIBaseTotalItems = dto.IPIBaseTotalItems
	tt.IPIBaseSubtractDiscount = dto.IPIBaseSubtractDiscount
	tt.IPIBaseAddFreight = dto.IPIBaseAddFreight
	tt.IPIBaseAddExpenses = dto.IPIBaseAddExpenses
	tt.ICMSBaseTotalItems = dto.ICMSBaseTotalItems
	tt.ICMSBaseSubtractDiscount = dto.ICMSBaseSubtractDiscount
	tt.ICMSBaseAddFreight = dto.ICMSBaseAddFreight
	tt.ICMSBaseAddIPI = dto.ICMSBaseAddIPI
	tt.ICMSBaseAddExpenses = dto.ICMSBaseAddExpenses
	tt.PISCOFINSBaseTotalItems = dto.PISCOFINSBaseTotalItems
	tt.PISCOFINSBaseSubtractDiscount = dto.PISCOFINSBaseSubtractDiscount
	tt.PISCOFINSBaseAddFreight = dto.PISCOFINSBaseAddFreight
	tt.PISCOFINSBaseAddInsurance = dto.PISCOFINSBaseAddInsurance
	tt.PISCOFINSBaseAddExpenses = dto.PISCOFINSBaseAddExpenses
	tt.CSLLBaseTotalItems = dto.CSLLBaseTotalItems
	tt.CSLLBaseSubtractDiscount = dto.CSLLBaseSubtractDiscount
	tt.CSLLBaseAddFreight = dto.CSLLBaseAddFreight
	tt.IRBaseTotalItems = dto.IRBaseTotalItems
	tt.IRBaseSubtractDiscount = dto.IRBaseSubtractDiscount
	tt.IRBaseAddFreight = dto.IRBaseAddFreight
	tt.IsConsumer = dto.IsConsumer
	created, err := uc.repo.CreateTaxType(ctx, tt)
	if err != nil {
		return nil, err
	}
	return toTaxTypeResponse(created), nil
}

func (uc *CustomerUseCase) ListTaxTypes(ctx context.Context, onlyActive bool) ([]*response.TaxTypeResponse, error) {
	types, err := uc.repo.ListTaxTypes(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.TaxTypeResponse, 0, len(types))
	for _, t := range types {
		out = append(out, toTaxTypeResponse(t))
	}
	return out, nil
}

func toTaxTypeResponse(tt *entity.TaxType) *response.TaxTypeResponse {
	return &response.TaxTypeResponse{
		ID:                            tt.ID,
		Code:                          tt.Code,
		Description:                   tt.Description,
		IPIBaseTotalItems:             tt.IPIBaseTotalItems,
		IPIBaseSubtractDiscount:       tt.IPIBaseSubtractDiscount,
		IPIBaseAddFreight:             tt.IPIBaseAddFreight,
		IPIBaseAddExpenses:            tt.IPIBaseAddExpenses,
		ICMSBaseTotalItems:            tt.ICMSBaseTotalItems,
		ICMSBaseSubtractDiscount:      tt.ICMSBaseSubtractDiscount,
		ICMSBaseAddFreight:            tt.ICMSBaseAddFreight,
		ICMSBaseAddIPI:                tt.ICMSBaseAddIPI,
		ICMSBaseAddExpenses:           tt.ICMSBaseAddExpenses,
		PISCOFINSBaseTotalItems:       tt.PISCOFINSBaseTotalItems,
		PISCOFINSBaseSubtractDiscount: tt.PISCOFINSBaseSubtractDiscount,
		PISCOFINSBaseAddFreight:       tt.PISCOFINSBaseAddFreight,
		PISCOFINSBaseAddInsurance:     tt.PISCOFINSBaseAddInsurance,
		PISCOFINSBaseAddExpenses:      tt.PISCOFINSBaseAddExpenses,
		CSLLBaseTotalItems:            tt.CSLLBaseTotalItems,
		CSLLBaseSubtractDiscount:      tt.CSLLBaseSubtractDiscount,
		CSLLBaseAddFreight:            tt.CSLLBaseAddFreight,
		IRBaseTotalItems:              tt.IRBaseTotalItems,
		IRBaseSubtractDiscount:        tt.IRBaseSubtractDiscount,
		IRBaseAddFreight:              tt.IRBaseAddFreight,
		IsConsumer:                    tt.IsConsumer,
		IsActive:                      tt.IsActive,
	}
}

// ─── Customers ────────────────────────────────────────────────────────────────

func (uc *CustomerUseCase) CreateCustomer(ctx context.Context, dto request.CreateCustomerDTO) (*response.CustomerResponse, error) {
	c, err := entity.NewCustomer(dto.Code, dto.Name, entity.DocumentType(dto.DocumentType), dto.DocumentNumber, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	c.CorporateCode = dto.CorporateCode
	c.IsCorporate = dto.IsCorporate
	c.TradeName = dto.TradeName
	c.StateRegistration = dto.StateRegistration
	c.MunicipalRegistration = dto.MunicipalRegistration
	c.SuframaCode = dto.SuframaCode
	c.SuframaExpiry = dto.SuframaExpiry
	c.CreditLimit = dto.CreditLimit
	c.Website = dto.Website
	c.PaymentCondVisibility = entity.PaymentCondVisibility(dto.PaymentCondVisibility)

	// Resolve FK IDs from codes
	if err := uc.resolveCustomerFKs(ctx, c, dto); err != nil {
		return nil, err
	}

	created, err := uc.repo.CreateCustomer(ctx, c)
	if err != nil {
		return nil, err
	}
	return toCustomerResponse(created), nil
}

// UpdateCustomer applies mutable fields to an existing customer identified by
// its business code. It shares CreateCustomerDTO so the front-end uses the
// same code-based contract for create and update; the customer code and
// document are immutable and taken from the persisted record.
func (uc *CustomerUseCase) UpdateCustomer(ctx context.Context, code int64, dto request.CreateCustomerDTO) (*response.CustomerResponse, error) {
	if dto.Name == "" {
		return nil, errorsuc.NewValidationError("name is required")
	}
	existing, err := uc.repo.GetCustomerByCode(ctx, code)
	if err != nil {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("customer %d not found", code))
	}

	existing.Name = dto.Name
	existing.TradeName = dto.TradeName
	existing.StateRegistration = dto.StateRegistration
	existing.MunicipalRegistration = dto.MunicipalRegistration
	existing.SuframaCode = dto.SuframaCode
	existing.SuframaExpiry = dto.SuframaExpiry
	existing.CreditLimit = dto.CreditLimit
	existing.Website = dto.Website
	if dto.PaymentCondVisibility != "" {
		existing.PaymentCondVisibility = entity.PaymentCondVisibility(dto.PaymentCondVisibility)
	}

	// Resolve FK codes -> IDs (only the ones provided are changed).
	if err := uc.resolveCustomerFKs(ctx, existing, dto); err != nil {
		return nil, err
	}

	updated, err := uc.repo.UpdateCustomer(ctx, existing)
	if err != nil {
		return nil, err
	}
	return toCustomerResponse(updated), nil
}

func (uc *CustomerUseCase) resolveCustomerFKs(ctx context.Context, c *entity.Customer, dto request.CreateCustomerDTO) error {
	if dto.RegionCode != nil {
		r, err := uc.repo.GetRegionByCode(ctx, *dto.RegionCode)
		if err != nil {
			return fmt.Errorf("region not found: %w", err)
		}
		c.RegionID = &r.ID
	}
	if dto.MarketSegmentCode != nil {
		s, err := uc.repo.GetMarketSegmentByCode(ctx, *dto.MarketSegmentCode)
		if err != nil {
			return fmt.Errorf("market segment not found: %w", err)
		}
		c.MarketSegmentID = &s.ID
	}
	if dto.CustomerTypeCode != nil {
		t, err := uc.repo.GetCustomerTypeByCode(ctx, *dto.CustomerTypeCode)
		if err != nil {
			return fmt.Errorf("customer type not found: %w", err)
		}
		c.CustomerTypeID = &t.ID
	}
	if dto.PaymentConditionCode != nil {
		pc, err := uc.repo.GetPaymentConditionByCode(ctx, *dto.PaymentConditionCode)
		if err != nil {
			return fmt.Errorf("payment condition not found: %w", err)
		}
		c.PaymentConditionID = &pc.ID
	}
	if dto.SalesTableCode != nil {
		st, err := uc.repo.GetSalesTableByCode(ctx, *dto.SalesTableCode)
		if err != nil {
			return fmt.Errorf("sales table not found: %w", err)
		}
		c.SalesTableID = &st.ID
	}
	if dto.CarrierCode != nil {
		car, err := uc.repo.GetCarrierByCode(ctx, *dto.CarrierCode)
		if err != nil {
			return fmt.Errorf("carrier not found: %w", err)
		}
		c.CarrierID = &car.ID
	}
	if dto.CarrierGroupCode != nil {
		cg, err := uc.repo.GetCarrierGroupByCode(ctx, *dto.CarrierGroupCode)
		if err != nil {
			return fmt.Errorf("carrier group not found: %w", err)
		}
		c.CarrierGroupID = &cg.ID
	}
	if dto.InvoiceTypeCode != nil {
		it, err := uc.repo.GetInvoiceTypeByCode(ctx, *dto.InvoiceTypeCode)
		if err != nil {
			return fmt.Errorf("invoice type not found: %w", err)
		}
		c.InvoiceTypeID = &it.ID
	}
	if dto.TaxTypeCode != nil {
		tt, err := uc.repo.GetTaxTypeByCode(ctx, *dto.TaxTypeCode)
		if err != nil {
			return fmt.Errorf("tax type not found: %w", err)
		}
		c.TaxTypeID = &tt.ID
	}
	return nil
}

func (uc *CustomerUseCase) GetCustomer(ctx context.Context, code int64) (*response.CustomerResponse, error) {
	c, err := uc.repo.GetCustomerByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("customer %d not found: %w", code, err)
	}
	resp := toCustomerResponse(c)
	// Load addresses and contacts
	addrs, _ := uc.repo.ListAddresses(ctx, c.ID)
	for _, a := range addrs {
		resp.Addresses = append(resp.Addresses, toAddressResponse(a))
	}
	contacts, _ := uc.repo.ListContacts(ctx, c.ID)
	for _, ct := range contacts {
		resp.Contacts = append(resp.Contacts, toContactResponse(ct))
	}
	return resp, nil
}

func (uc *CustomerUseCase) ListCustomers(ctx context.Context, onlyActive bool) ([]*response.CustomerResponse, error) {
	customers, err := uc.repo.ListCustomers(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CustomerResponse, 0, len(customers))
	for _, c := range customers {
		out = append(out, toCustomerResponse(c))
	}
	return out, nil
}

func (uc *CustomerUseCase) ListEstablishments(ctx context.Context, corporateCode int64) ([]*response.CustomerResponse, error) {
	customers, err := uc.repo.ListEstablishments(ctx, corporateCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CustomerResponse, 0, len(customers))
	for _, c := range customers {
		out = append(out, toCustomerResponse(c))
	}
	return out, nil
}

func (uc *CustomerUseCase) BlockCustomer(ctx context.Context, dto request.BlockCustomerDTO) error {
	if dto.Reason == "" {
		return fmt.Errorf("block reason is required")
	}
	return uc.repo.BlockCustomer(ctx, dto.CustomerCode, dto.Reason)
}

func (uc *CustomerUseCase) UnblockCustomer(ctx context.Context, code int64) error {
	return uc.repo.UnblockCustomer(ctx, code)
}

func (uc *CustomerUseCase) AddAddress(ctx context.Context, dto request.AddAddressDTO) (*response.CustomerAddressResponse, error) {
	c, err := uc.repo.GetCustomerByCode(ctx, dto.CustomerCode)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}
	country := dto.Country
	if country == "" {
		country = "Brasil"
	}
	addr := &entity.CustomerAddress{
		CustomerID:   c.ID,
		AddressType:  entity.AddressType(dto.AddressType),
		ZipCode:      dto.ZipCode,
		Street:       dto.Street,
		Number:       dto.Number,
		Complement:   dto.Complement,
		Neighborhood: dto.Neighborhood,
		City:         dto.City,
		UF:           dto.UF,
		Country:      country,
		IsDefault:    dto.IsDefault,
	}
	created, err := uc.repo.AddAddress(ctx, addr)
	if err != nil {
		return nil, err
	}
	r := toAddressResponse(created)
	return &r, nil
}

func (uc *CustomerUseCase) AddContact(ctx context.Context, dto request.AddContactDTO) (*response.CustomerContactResponse, error) {
	c, err := uc.repo.GetCustomerByCode(ctx, dto.CustomerCode)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}
	if dto.Name == "" {
		return nil, fmt.Errorf("contact name is required")
	}
	var contactTypeID *int64
	if dto.ContactTypeCode != nil {
		ct, err := uc.repo.GetContactTypeByCode(ctx, *dto.ContactTypeCode)
		if err != nil {
			return nil, fmt.Errorf("contact type not found: %w", err)
		}
		contactTypeID = &ct.ID
	}
	contact := &entity.CustomerContact{
		CustomerID:    c.ID,
		ContactTypeID: contactTypeID,
		Name:          dto.Name,
		Email:         dto.Email,
		Phone:         dto.Phone,
		Mobile:        dto.Mobile,
		Position:      dto.Position,
		IsPrimary:     dto.IsPrimary,
		IsActive:      true,
	}
	created, err := uc.repo.AddContact(ctx, contact)
	if err != nil {
		return nil, err
	}
	r := toContactResponse(created)
	return &r, nil
}

func toCustomerResponse(c *entity.Customer) *response.CustomerResponse {
	return &response.CustomerResponse{
		ID:                    c.ID,
		Code:                  c.Code,
		CorporateCode:         c.CorporateCode,
		IsCorporate:           c.IsCorporate,
		Name:                  c.Name,
		TradeName:             c.TradeName,
		DocumentType:          string(c.DocumentType),
		DocumentNumber:        c.DocumentNumber,
		StateRegistration:     c.StateRegistration,
		MunicipalRegistration: c.MunicipalRegistration,
		SuframaCode:           c.SuframaCode,
		SuframaExpiry:         c.SuframaExpiry,
		RegionID:              c.RegionID,
		MarketSegmentID:       c.MarketSegmentID,
		CustomerTypeID:        c.CustomerTypeID,
		PaymentConditionID:    c.PaymentConditionID,
		SalesTableID:          c.SalesTableID,
		CarrierID:             c.CarrierID,
		CarrierGroupID:        c.CarrierGroupID,
		InvoiceTypeID:         c.InvoiceTypeID,
		TaxTypeID:             c.TaxTypeID,
		PaymentCondVisibility: string(c.PaymentCondVisibility),
		CreditLimit:           c.CreditLimit,
		Website:               c.Website,
		IsActive:              c.IsActive,
		Blocked:               c.Blocked,
		BlockReason:           c.BlockReason,
		CreatedAt:             c.CreatedAt,
		UpdatedAt:             c.UpdatedAt,
	}
}

func toAddressResponse(a *entity.CustomerAddress) response.CustomerAddressResponse {
	return response.CustomerAddressResponse{
		ID:           a.ID,
		AddressType:  string(a.AddressType),
		ZipCode:      a.ZipCode,
		Street:       a.Street,
		Number:       a.Number,
		Complement:   a.Complement,
		Neighborhood: a.Neighborhood,
		City:         a.City,
		UF:           a.UF,
		Country:      a.Country,
		IsDefault:    a.IsDefault,
	}
}

func toContactResponse(c *entity.CustomerContact) response.CustomerContactResponse {
	return response.CustomerContactResponse{
		ID:            c.ID,
		ContactTypeID: c.ContactTypeID,
		Name:          c.Name,
		Email:         c.Email,
		Phone:         c.Phone,
		Mobile:        c.Mobile,
		Position:      c.Position,
		IsPrimary:     c.IsPrimary,
	}
}
