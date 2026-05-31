package customer_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
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
	st.DecimalPlaces = dto.DecimalPlaces
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
		Description:             dto.Description,
		Type:                    dto.Type,
		StockMovement:           dto.StockMovement,
		ICMSType:                dto.ICMSType,
		ICMSPct:                 dto.ICMSPct,
		ICMSReductionPct:        dto.ICMSReductionPct,
		IPIPct:                  dto.IPIPct,
		PISPct:                  dto.PISPct,
		COFINSPct:               dto.COFINSPct,
		ISSQNPct:                dto.ISSQNPct,
		IRPct:                   dto.IRPct,
		CSLLPct:                 dto.CSLLPct,
		INSSPct:                 dto.INSSPct,
		GeneratesRevenue:        dto.GeneratesRevenue,
		UpdatesInventory:        dto.UpdatesInventory,
		GeneratesFinancialTitle: dto.GeneratesFinancialTitle,
		ConsidersGoals:          dto.ConsidersGoals,
		CalcSubstitutionTax:     dto.CalcSubstitutionTax,
		CalcICMSDeferral:        dto.CalcICMSDeferral,
		CalcPISCOFINS:           dto.CalcPISCOFINS,
		CalcDIFAL:               dto.CalcDIFAL,
		RequiresSalesOrder:      dto.RequiresSalesOrder,
		ListsFiscalBooks:        dto.ListsFiscalBooks,
		ModelNF:                 dto.ModelNF,
		CSTICMS:                 dto.CSTICMS,
		CSOSNTICMS:              dto.CSOSNTICMS,
		CSTIPI:                  dto.CSTIPI,
		CSTPIS:                  dto.CSTPIS,
		CSTCOFINS:               dto.CSTCOFINS,
		BaixaPedido:             dto.BaixaPedido,
		GeraTituloDev:           dto.GeraTituloDev,
		ExigeSuframa:            dto.ExigeSuframa,
		IRPctPresumption:        dto.IRPctPresumption,
		CSLLPctPresumption:      dto.CSLLPctPresumption,
		DescriptionNF:           dto.DescriptionNF,
		ImpostosNFe:             dto.ImpostosNFe,
		CFOPId:                  dto.CFOPId,
		DispositivoLegalIPIId:   dto.DispositivoLegalIPIId,
		DispositivoLegalICMSId:  dto.DispositivoLegalICMSId,
		DispositivoLegalICMSSTId: dto.DispositivoLegalICMSSTId,
		DispositivoLegalPISId:   dto.DispositivoLegalPISId,
		DispositivoLegalCOFINSId: dto.DispositivoLegalCOFINSId,
		HierarchyIPI:            dto.HierarchyIPI,
		HierarchyICMS:           dto.HierarchyICMS,
		HierarchyICMSST:         dto.HierarchyICMSST,
		HierarchyPIS:            dto.HierarchyPIS,
		HierarchyCOFINS:         dto.HierarchyCOFINS,
		IPITransferSalesTableId: dto.IPITransferSalesTableId,
		ListaValorContabil:      dto.ListaValorContabil,
		ListaRegistroSaida:      dto.ListaRegistroSaida,
		ListaICMSIPI:            dto.ListaICMSIPI,
		SintegraSpedFiscal:      dto.SintegraSpedFiscal,
		CalcFomentar:            dto.CalcFomentar,
		ExcecaoFomentar:         dto.ExcecaoFomentar,
		CompRessRetST:           dto.CompRessRetST,
		CalcReducao:             dto.CalcReducao,
		ComplementoItens:        dto.ComplementoItens,
		BuscaTipoNF:             dto.BuscaTipoNF,
		ICMSSTUltEntrada:        dto.ICMSSTUltEntrada,
		SomenteConsultaLotes:    dto.SomenteConsultaLotes,
		CalcImpIBPT:             dto.CalcImpIBPT,
		CredPresumidoICMS:       dto.CredPresumidoICMS,
		CIAP:                    dto.CIAP,
		VlrAgregadoBaseSubst:    dto.VlrAgregadoBaseSubst,
		ContratoFacon:           dto.ContratoFacon,
		DescICMSLicitacoes:      dto.DescICMSLicitacoes,
		Sisdeclara:              dto.Sisdeclara,
		CodClasTrib:             dto.CodClasTrib,
		CodClasTribTribReg:      dto.CodClasTribTribReg,
		CodMotivoRestCompICMSST: dto.CodMotivoRestCompICMSST,
		CodBeneficioFiscal:      dto.CodBeneficioFiscal,
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
		ID:                          it.ID,
		Code:                        it.Code,
		Description:                 it.Description,
		Type:                        string(it.Type),
		StockMovement:               string(it.StockMovement),
		ICMSType:                    string(it.ICMSType),
		ICMSPct:                     it.ICMSPct,
		ICMSReductionPct:            it.ICMSReductionPct,
		IPIPct:                      it.IPIPct,
		PISPct:                      it.PISPct,
		COFINSPct:                   it.COFINSPct,
		ISSQNPct:                    it.ISSQNPct,
		IRPct:                       it.IRPct,
		CSLLPct:                     it.CSLLPct,
		INSSPct:                     it.INSSPct,
		GeneratesRevenue:            it.GeneratesRevenue,
		UpdatesInventory:            it.UpdatesInventory,
		GeneratesFinancialTitle:     it.GeneratesFinancialTitle,
		ConsidersGoals:              it.ConsidersGoals,
		CalcSubstitutionTax:         it.CalcSubstitutionTax,
		CalcICMSDeferral:            it.CalcICMSDeferral,
		CalcPISCOFINS:               it.CalcPISCOFINS,
		CalcDIFAL:                   it.CalcDIFAL,
		RequiresSalesOrder:          it.RequiresSalesOrder,
		ListsFiscalBooks:            it.ListsFiscalBooks,
		IsActive:                    it.IsActive,
		ModelNF:                     it.ModelNF,
		CSTICMS:                     it.CSTICMS,
		CSOSNTICMS:                  it.CSOSNTICMS,
		CSTIPI:                      it.CSTIPI,
		CSTPIS:                      it.CSTPIS,
		CSTCOFINS:                   it.CSTCOFINS,
		BaixaPedido:                 it.BaixaPedido,
		GeraTituloDev:               it.GeraTituloDev,
		ExigeSuframa:                it.ExigeSuframa,
		IRPctPresumption:            it.IRPctPresumption,
		CSLLPctPresumption:          it.CSLLPctPresumption,
		DescriptionNF:               it.DescriptionNF,
		ImpostosNFe:                 impostosNFe,
		CFOPId:                      it.CFOPId,
		DispositivoLegalIPIId:       it.DispositivoLegalIPIId,
		DispositivoLegalICMSId:      it.DispositivoLegalICMSId,
		DispositivoLegalICMSSTId:    it.DispositivoLegalICMSSTId,
		DispositivoLegalPISId:       it.DispositivoLegalPISId,
		DispositivoLegalCOFINSId:    it.DispositivoLegalCOFINSId,
		HierarchyIPI:                it.HierarchyIPI,
		HierarchyICMS:               it.HierarchyICMS,
		HierarchyICMSST:             it.HierarchyICMSST,
		HierarchyPIS:                it.HierarchyPIS,
		HierarchyCOFINS:             it.HierarchyCOFINS,
		IPITransferSalesTableId:     it.IPITransferSalesTableId,
		ListaValorContabil:          it.ListaValorContabil,
		ListaRegistroSaida:          it.ListaRegistroSaida,
		ListaICMSIPI:                it.ListaICMSIPI,
		SintegraSpedFiscal:          it.SintegraSpedFiscal,
		CalcFomentar:                it.CalcFomentar,
		ExcecaoFomentar:             it.ExcecaoFomentar,
		CompRessRetST:               it.CompRessRetST,
		CalcReducao:                 it.CalcReducao,
		ComplementoItens:            it.ComplementoItens,
		BuscaTipoNF:                 it.BuscaTipoNF,
		ICMSSTUltEntrada:            it.ICMSSTUltEntrada,
		SomenteConsultaLotes:        it.SomenteConsultaLotes,
		CalcImpIBPT:                 it.CalcImpIBPT,
		CredPresumidoICMS:           it.CredPresumidoICMS,
		CIAP:                        it.CIAP,
		VlrAgregadoBaseSubst:        it.VlrAgregadoBaseSubst,
		ContratoFacon:               it.ContratoFacon,
		DescICMSLicitacoes:          it.DescICMSLicitacoes,
		Sisdeclara:                  it.Sisdeclara,
		CodClasTrib:                 it.CodClasTrib,
		CodClasTribTribReg:          it.CodClasTribTribReg,
		CodMotivoRestCompICMSST:     it.CodMotivoRestCompICMSST,
		CodBeneficioFiscal:          it.CodBeneficioFiscal,
		CreatedAt:                   it.CreatedAt,
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
	if dto.SalesTableID == 0 {
		return nil, fmt.Errorf("sales_table_id is required")
	}
	if dto.ItemCode == "" {
		return nil, fmt.Errorf("item_code is required")
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

func (uc *CustomerUseCase) UpdateSalesTablePrice(ctx context.Context, dto request.UpdateSalesTablePriceDTO) (*response.SalesTablePriceResponse, error) {
	if dto.ID == 0 {
		return nil, fmt.Errorf("id is required")
	}
	if dto.Price < 0 {
		return nil, fmt.Errorf("price must be >= 0")
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
