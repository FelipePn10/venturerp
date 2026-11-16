package customer_uc

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/representativevalidation"
	cnpjservice "github.com/FelipePn10/panossoerp/internal/domain/cnpj/service"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	"github.com/FelipePn10/panossoerp/internal/shared/ptrutil"

	"github.com/shopspring/decimal"
)

// CustomerUseCase consolidates all customer-related operations.
type CustomerUseCase struct {
	repo repository.CustomerRepository
	// life e cnpj ligam o crédito, a situação na Receita e o ciclo de vida do
	// cadastro (credit_uc.go); sem eles o cadastro básico continua funcionando.
	life repository.CustomerLifecycleRepository
	cnpj cnpjservice.Provider
	reps representativevalidation.Repository
	now  func() time.Time
}

func NewCustomerUseCase(repo repository.CustomerRepository) *CustomerUseCase {
	return &CustomerUseCase{repo: repo, now: time.Now}
}

// WithLifecycle liga crédito, Receita e ciclo de vida do cadastro, e a
// conferência do representante padrão (vínculo com a empresa, ativo).
func (uc *CustomerUseCase) WithLifecycle(life repository.CustomerLifecycleRepository, cnpj cnpjservice.Provider, reps representativevalidation.Repository) *CustomerUseCase {
	uc.life, uc.cnpj, uc.reps = life, cnpj, reps
	return uc
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
		return nil, fmt.Errorf("região não encontrada: %w", err)
	}
	reg.Description = dto.Description
	reg.UF = dto.UF
	reg.City = dto.City
	reg.IsActive = ptrutil.BoolOr(dto.IsActive, reg.IsActive)
	updated, err := uc.repo.UpdateRegion(ctx, reg)
	if err != nil {
		return nil, err
	}
	return toRegionResponse(updated), nil
}

func (uc *CustomerUseCase) GetRegion(ctx context.Context, code int64) (*response.RegionResponse, error) {
	reg, err := uc.repo.GetRegionByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("região %d não encontrada: %w", code, err)
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
			return nil, fmt.Errorf("segmento pai não encontrado: %w", err)
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

func (uc *CustomerUseCase) UpdateMarketSegment(ctx context.Context, code int64, dto request.UpdateMarketSegmentDTO) (*response.MarketSegmentResponse, error) {
	current, err := uc.repo.GetMarketSegmentByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	current.Description, current.HasPISCOFINSRetention, current.RetentionIndicator = dto.Description, dto.HasPISCOFINSRetention, dto.RetentionIndicator
	current.IsActive = ptrutil.BoolOr(dto.IsActive, current.IsActive)
	if dto.ParentID != nil {
		current.ParentID = dto.ParentID
	}
	updated, err := uc.repo.UpdateMarketSegment(ctx, current)
	if err != nil {
		return nil, err
	}
	return toSegmentResponse(updated), nil
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

func (uc *CustomerUseCase) UpdateContactType(ctx context.Context, code int64, dto request.UpdateContactTypeDTO) (*response.ContactTypeResponse, error) {
	current, err := uc.repo.GetContactTypeByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(dto.Description) == "" {
		return nil, errorsuc.NewValidationError("a descrição é obrigatória")
	}
	current.Description = strings.TrimSpace(dto.Description)
	current.IsActive = ptrutil.BoolOr(dto.IsActive, current.IsActive)
	updated, err := uc.repo.UpdateContactType(ctx, current)
	if err != nil {
		return nil, err
	}
	return toContactTypeResponse(updated), nil
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

func (uc *CustomerUseCase) UpdateCustomerType(ctx context.Context, code int64, dto request.UpdateCustomerTypeDTO) (*response.CustomerTypeResponse, error) {
	current, err := uc.repo.GetCustomerTypeByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	validated, err := entity.NewCustomerType(code, dto.Description, entity.CustomerCategory(dto.Category), dto.DeliveryDays)
	if err != nil {
		return nil, err
	}
	validated.ID, validated.IsActive = current.ID, ptrutil.BoolOr(dto.IsActive, current.IsActive)
	updated, err := uc.repo.UpdateCustomerType(ctx, validated)
	if err != nil {
		return nil, err
	}
	return toCustomerTypeResponse(updated), nil
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

func (uc *CustomerUseCase) UpdateCarrier(ctx context.Context, code int64, dto request.UpdateCarrierDTO) (*response.CarrierResponse, error) {
	current, err := uc.repo.GetCarrierByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	validated, err := entity.NewCarrier(code, dto.Description, entity.CarrierBillingType(dto.BillingType))
	if err != nil {
		return nil, err
	}
	validated.ID, validated.IsActive = current.ID, ptrutil.BoolOr(dto.IsActive, current.IsActive)
	validated.UsesCreditLimit, validated.ConsiderAvailable, validated.PostponeDueDate, validated.ReceiptDays, validated.PaymentDays = dto.UsesCreditLimit, dto.ConsiderAvailable, dto.PostponeDueDate, dto.ReceiptDays, dto.PaymentDays
	updated, err := uc.repo.UpdateCarrier(ctx, validated)
	if err != nil {
		return nil, err
	}
	return toCarrierResponse(updated), nil
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

func (uc *CustomerUseCase) UpdateCarrierGroup(ctx context.Context, code int64, dto request.UpdateCarrierGroupDTO) (*response.CarrierGroupResponse, error) {
	if strings.TrimSpace(dto.Description) == "" {
		return nil, errorsuc.NewValidationError("a descrição é obrigatória")
	}
	current, err := uc.repo.GetCarrierGroupByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	current.Description = strings.TrimSpace(dto.Description)
	updated, err := uc.repo.UpdateCarrierGroup(ctx, current)
	if err != nil {
		return nil, err
	}
	return &response.CarrierGroupResponse{ID: updated.ID, Code: updated.Code, Description: updated.Description}, nil
}

func (uc *CustomerUseCase) AddCarrierToGroup(ctx context.Context, dto request.CarrierGroupMemberDTO) error {
	group, err := uc.repo.GetCarrierGroupByCode(ctx, dto.CarrierGroupCode)
	if err != nil {
		return fmt.Errorf("grupo de transportadoras não encontrado: %w", err)
	}
	carrier, err := uc.repo.GetCarrierByCode(ctx, dto.CarrierCode)
	if err != nil {
		return fmt.Errorf("transportadora não encontrada: %w", err)
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
	parcelStart, err := entity.NormalizarParcelStart(dto.ParcelStart)
	if err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}
	pc.ParcelStart = parcelStart
	pc.Expenses = dto.Expenses
	pc.AverageTerm = dto.AverageTerm
	pc.IsSpecial = dto.IsSpecial
	pc.IsRevenue = dto.IsRevenue
	pc.IsAtSight = dto.IsAtSight
	if dto.CarrierCode != nil {
		c, err := uc.repo.GetCarrierByCode(ctx, *dto.CarrierCode)
		if err != nil {
			return nil, fmt.Errorf("transportadora não encontrada: %w", err)
		}
		pc.CarrierID = &c.ID
	}
	created, err := uc.repo.CreatePaymentCondition(ctx, pc)
	if err != nil {
		return nil, err
	}
	return toPaymentCondResponse(created), nil
}

func (uc *CustomerUseCase) UpdatePaymentCondition(ctx context.Context, code int64, dto request.UpdatePaymentConditionDTO) (*response.PaymentConditionResponse, error) {
	current, err := uc.repo.GetPaymentConditionByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	validated, err := entity.NewPaymentCondition(code, dto.Description, entity.PaymentAnalysis(dto.AnalysisType))
	if err != nil {
		return nil, err
	}
	validated.ID, validated.IsActive = current.ID, ptrutil.BoolOr(dto.IsActive, current.IsActive)
	parcelStart, err := entity.NormalizarParcelStart(dto.ParcelStart)
	if err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}
	validated.ParcelStart, validated.Expenses, validated.AverageTerm = parcelStart, dto.Expenses, dto.AverageTerm
	validated.IsSpecial, validated.IsRevenue, validated.IsAtSight = dto.IsSpecial, dto.IsRevenue, dto.IsAtSight
	if dto.CarrierCode != nil {
		carrier, err := uc.repo.GetCarrierByCode(ctx, *dto.CarrierCode)
		if err != nil {
			return nil, fmt.Errorf("portador não encontrado: %w", err)
		}
		validated.CarrierID = &carrier.ID
	}
	updated, err := uc.repo.UpdatePaymentCondition(ctx, validated)
	if err != nil {
		return nil, err
	}
	return toPaymentCondResponse(updated), nil
}

func (uc *CustomerUseCase) AddInstallment(ctx context.Context, dto request.AddInstallmentDTO) (*response.InstallmentResponse, error) {
	pc, err := uc.repo.GetPaymentConditionByCode(ctx, dto.PaymentConditionCode)
	if err != nil {
		return nil, fmt.Errorf("condição de pagamento não encontrada: %w", err)
	}
	if dto.InstallmentNumber <= 0 {
		return nil, errorsuc.NewValidationError("o número da parcela começa em 1")
	}
	if dto.DueDays < 0 {
		return nil, errorsuc.NewValidationError("o prazo da parcela não pode ser negativo")
	}
	evento := entity.BaseEmissao
	if dto.BaseEvent != nil && strings.TrimSpace(*dto.BaseEvent) != "" {
		evento = entity.PaymentBaseEvent(strings.ToUpper(strings.TrimSpace(*dto.BaseEvent)))
		if !evento.IsValid() {
			return nil, errorsuc.NewValidationError("o vencimento da parcela conta a partir da emissão, da entrada, da entrega ou do faturamento")
		}
	}
	dias := dto.DueDays
	if evento == entity.BaseEntrada {
		// Entrada é pagamento no ato: prazo em dias não significa nada, e a
		// restrição do banco recusaria a linha com uma mensagem técnica.
		dias = 0
	}
	if dto.Percentage != nil && (!dto.Percentage.IsPositive() || dto.Percentage.GreaterThan(decimal.NewFromInt(100))) {
		return nil, errorsuc.NewValidationError("o percentual da parcela precisa ficar entre 0 e 100")
	}
	inst := &entity.PaymentInstallment{
		PaymentConditionID: pc.ID,
		InstallmentNumber:  dto.InstallmentNumber,
		DueDays:            dias,
		Percentage:         dto.Percentage,
		BaseEvent:          string(evento),
		Description:        dto.Description,
		DocumentType:       dto.DocumentType,
		MovementType:       dto.MovementType,
	}
	if dto.CarrierCode != nil {
		c, err := uc.repo.GetCarrierByCode(ctx, *dto.CarrierCode)
		if err != nil {
			return nil, fmt.Errorf("transportadora não encontrada: %w", err)
		}
		inst.CarrierID = &c.ID
	}
	created, err := uc.repo.AddInstallment(ctx, inst)
	if err != nil {
		return nil, err
	}
	// A condição inteira precisa fechar 100%: a parcela sozinha não tem como
	// saber disso, e uma condição que soma 80% gera título a menos em todo
	// pedido que a usar. O aviso sai aqui, com a condição já gravada, para não
	// travar o cadastro no meio — quem monta 30/20/25/25 passa por 30%, 50%,
	// 75% até fechar.
	var aviso *string
	if completa, cErr := uc.repo.GetPaymentConditionByCode(ctx, dto.PaymentConditionCode); cErr == nil {
		if vErr := entity.ValidarPercentuais(completa.Installments); vErr != nil {
			// Aviso, não erro: devolver 422 aqui fazia a tela dizer "falhou" em
			// cada parcela do caminho, embora todas tenham sido gravadas.
			texto := vErr.Error() + " — a parcela foi gravada; acerte os percentuais antes de usar esta condição"
			aviso = &texto
		}
	}
	return &response.InstallmentResponse{
		Aviso:             aviso,
		ID:                created.ID,
		InstallmentNumber: created.InstallmentNumber,
		DueDays:           created.DueDays,
		Description:       created.Description,
		DocumentType:      created.DocumentType,
		MovementType:      created.MovementType,
		CarrierID:         created.CarrierID,
		Percentage:        fptrD(created.Percentage),
		BaseEvent:         created.BaseEvent,
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
		Expenses:     pc.Expenses.InexactFloat64(),
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
	// A formação de preço é conferida ANTES do construtor para a recusa sair como
	// validação (422) em vez de erro interno (500): o construtor devolve fmt.Errorf,
	// que RespondUseCaseError trata como falha do sistema e esconde atrás de
	// "erro interno do servidor". Ver error-status-contract.
	if formacao := entity.PriceFormation(dto.PriceFormation); !formacao.IsValid() {
		return nil, errorsuc.NewValidationError(fmt.Sprintf(
			"formação de preço %q não existe — use uma destas: %s",
			dto.PriceFormation, strings.Join(entity.FormacoesDePreco(), ", ")))
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
		return nil, fmt.Errorf("tabela de venda não encontrada: %w", err)
	}
	return toSalesTableResponse(st), nil
}

func (uc *CustomerUseCase) UpdateSalesTable(ctx context.Context, dto request.UpdateSalesTableDTO) (*response.SalesTableResponse, error) {
	if dto.Code == 0 {
		return nil, errorsuc.NewValidationError("informe o código")
	}
	st, err := uc.repo.GetSalesTableByCode(ctx, dto.Code)
	if err != nil {
		return nil, fmt.Errorf("tabela de venda não encontrada: %w", err)
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
	st.IsActive = ptrutil.BoolOr(dto.IsActive, st.IsActive)
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
		ToleranceMinPct:            st.ToleranceMinPct.InexactFloat64(),
		ToleranceMaxPct:            st.ToleranceMaxPct.InexactFloat64(),
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
		return nil, errorsuc.NewValidationError("informe o código")
	}
	p, err := uc.repo.GetSalesPricePolicyByCode(ctx, dto.Code)
	if err != nil {
		return nil, fmt.Errorf("política de preço de venda não encontrada: %w", err)
	}
	if err := uc.applySalesPricePolicyDTO(ctx, p, dto); err != nil {
		return nil, err
	}
	p.IsActive = ptrutil.BoolOr(dto.IsActive, p.IsActive)
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

// A política criada nasce ativa; o campo é ponteiro para que uma atualização
// que não o mencione preserve a situação atual.
var verdadeiro = true

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
		IsActive:       &verdadeiro,
		Observation:    dto.Observation,
	}
}

func (uc *CustomerUseCase) applySalesPricePolicyDTO(ctx context.Context, p *entity.SalesPricePolicy, dto request.UpdateSalesPricePolicyDTO) error {
	if dto.Description == "" {
		return errorsuc.NewValidationError("informe a descrição")
	}
	source := entity.SalesCostSource(dto.CostSource)
	if source == "" {
		source = entity.SalesCostStandardTotal
	}
	if !entity.ValidSalesCostSource(source) {
		return errorsuc.NewValidationError("origem de custo inválida")
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
		return errorsuc.NewValidationError("a abrangência da política deve ser precificação ou formação de preço")
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
	if p.MaxMarginPct.IsPositive() && p.MinMarginPct.GreaterThan(p.MaxMarginPct) {
		return errorsuc.NewValidationError("a margem mínima não pode ser maior que a máxima")
	}
	if p.IdealMarginPct.IsPositive() {
		if p.IdealMarginPct.LessThan(p.MinMarginPct) {
			return errorsuc.NewValidationError("a margem ideal não pode ser menor que a margem mínima")
		}
		if p.MaxMarginPct.IsPositive() && p.IdealMarginPct.GreaterThan(p.MaxMarginPct) {
			return errorsuc.NewValidationError("a margem ideal não pode ser maior que a margem máxima")
		}
	}
	p.IncidencesJSON = string(dto.IncidencesJSON)
	if p.IncidencesJSON == "" {
		p.IncidencesJSON = "[]"
	}
	if !json.Valid([]byte(p.IncidencesJSON)) {
		return errorsuc.NewValidationError("a configuração de incidências está inválida")
	}
	p.ValidityStart = dto.ValidityStart
	p.ValidityEnd = dto.ValidityEnd
	p.Observation = dto.Observation
	p.SalesTableID = nil
	if dto.SalesTableCode != nil {
		st, err := uc.repo.GetSalesTableByCode(ctx, *dto.SalesTableCode)
		if err != nil {
			return fmt.Errorf("tabela de venda não encontrada: %w", err)
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
		MarkupPct:      p.MarkupPct.InexactFloat64(),
		MarginPct:      p.MarginPct.InexactFloat64(),
		MaxMarginPct:   p.MaxMarginPct.InexactFloat64(),
		IdealMarginPct: p.IdealMarginPct.InexactFloat64(),
		MarginStepPct:  p.MarginStepPct.InexactFloat64(),
		ExpensesPct:    p.ExpensesPct.InexactFloat64(),
		TaxesPct:       p.TaxesPct.InexactFloat64(),
		FreightPct:     p.FreightPct.InexactFloat64(),
		CommissionPct:  p.CommissionPct.InexactFloat64(),
		DiscountPct:    p.DiscountPct.InexactFloat64(),
		MinMarginPct:   p.MinMarginPct.InexactFloat64(),
		MaxDiscountPct: p.MaxDiscountPct.InexactFloat64(),
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
		return nil, errorsuc.NewValidationError("informe o código")
	}
	p, err := uc.repo.GetCommercialPolicyByCode(ctx, dto.Code)
	if err != nil {
		return nil, fmt.Errorf("política comercial não encontrada: %w", err)
	}
	if err := applyCommercialPolicyDTO(p, dto); err != nil {
		return nil, err
	}
	p.IsActive = ptrutil.BoolOr(dto.IsActive, p.IsActive)
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
			return nil, errorsuc.NewValidationError("tipo de política inválido")
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
		return nil, errorsuc.NewValidationError("informe a política comercial")
	}
	if dto.ItemCode == nil && dto.ProductLineID == nil && dto.ItemClassification == nil {
		return nil, errorsuc.NewValidationError("informe o item, a linha de produto ou a classificação do item")
	}
	p, err := uc.repo.GetCommercialPolicyByCode(ctx, dto.PolicyCode)
	if err != nil {
		return nil, fmt.Errorf("política comercial não encontrada: %w", err)
	}
	var itemCode *string
	if dto.ItemCode != nil {
		value := dto.ItemCode.String()
		itemCode = &value
	}
	item, err := uc.repo.AddCommercialPolicySpecificItem(ctx, &entity.CommercialPolicySpecificItem{
		PolicyID:           p.ID,
		ItemCode:           itemCode,
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
	var itemCode *string
	if dto.ItemCode != nil {
		value := dto.ItemCode.String()
		itemCode = &value
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
		ItemCode:           itemCode,
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
		IsActive:               &verdadeiro,
		Observation:            dto.Observation,
	}
}

func applyCommercialPolicyDTO(p *entity.CommercialPolicy, dto request.UpdateCommercialPolicyDTO) error {
	if dto.Description == "" {
		return errorsuc.NewValidationError("informe a descrição")
	}
	kind := entity.CommercialPolicyKind(dto.Kind)
	if !entity.ValidCommercialPolicyKind(kind) {
		return errorsuc.NewValidationError("tipo de política inválido")
	}
	calcType := entity.CommercialPolicyCalcType(dto.CalcType)
	if calcType == "" {
		calcType = entity.CommercialPolicyPercent
	}
	if !entity.ValidCommercialPolicyCalcType(calcType) {
		return errorsuc.NewValidationError("tipo de cálculo inválido")
	}
	for name, value := range map[string]decimal.Decimal{
		"percent_value":   dto.PercentValue,
		"fixed_value":     dto.FixedValue,
		"max_percent":     dto.MaxPercent,
		"max_value":       dto.MaxValue,
		"min_gross_value": dto.MinGrossValue,
		"max_gross_value": dto.MaxGrossValue,
		"min_quantity":    dto.MinQuantity,
		"max_quantity":    dto.MaxQuantity,
	} {
		if value.IsNegative() {
			return errorsuc.NewValidationError(fmt.Sprintf("%s deve ser maior ou igual a zero", name))
		}
	}
	if dto.MaxGrossValue.IsPositive() && dto.MinGrossValue.GreaterThan(dto.MaxGrossValue) {
		return errorsuc.NewValidationError("o valor bruto mínimo não pode ser maior que o máximo")
	}
	if dto.MaxQuantity.IsPositive() && dto.MinQuantity.GreaterThan(dto.MaxQuantity) {
		return errorsuc.NewValidationError("a quantidade mínima não pode ser maior que a máxima")
	}
	p.Description = dto.Description
	p.Kind = kind
	choiceType := entity.CommercialPolicyChoiceType(dto.ChoiceType)
	if choiceType == "" {
		choiceType = entity.CommercialPolicyInformation
	}
	if !entity.ValidCommercialPolicyChoiceType(choiceType) {
		return errorsuc.NewValidationError("tipo de escolha inválido")
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
		return errorsuc.NewValidationError("a configuração de tipos de dados está inválida")
	}
	p.CommissionDiscountMode = dto.CommissionDiscountMode
	if p.CommissionDiscountMode == "" {
		p.CommissionDiscountMode = "REAL"
	}
	if p.CommissionDiscountMode != "REAL" && p.CommissionDiscountMode != "NOMINAL" {
		return errorsuc.NewValidationError("o desconto de comissão deve ser real ou nominal")
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
		return errorsuc.NewValidationError("a configuração da regra está inválida")
	}
	p.ValidityStart = dto.ValidityStart
	p.ValidityEnd = dto.ValidityEnd
	p.Observation = dto.Observation
	return nil
}

func (uc *CustomerUseCase) AddCommercialPolicyLine(ctx context.Context, dto request.CommercialPolicyLineDTO) (*response.CommercialPolicyLineResponse, error) {
	if dto.PolicyCode == 0 {
		return nil, errorsuc.NewValidationError("informe a política comercial")
	}
	if dto.LineNumber == 0 {
		return nil, errorsuc.NewValidationError("informe o número da linha")
	}
	p, err := uc.repo.GetCommercialPolicyByCode(ctx, dto.PolicyCode)
	if err != nil {
		return nil, fmt.Errorf("política comercial não encontrada: %w", err)
	}
	calcType := entity.CommercialPolicyCalcType(dto.CalcType)
	if calcType == "" {
		calcType = entity.CommercialPolicyPercent
	}
	if !entity.ValidCommercialPolicyCalcType(calcType) {
		return nil, errorsuc.NewValidationError("tipo de cálculo inválido")
	}
	if dto.PercentValue.IsNegative() || dto.FixedValue.IsNegative() || dto.MinValue.IsNegative() || dto.MaxValue.IsNegative() {
		return nil, errorsuc.NewValidationError("os valores da linha devem ser maiores ou iguais a zero")
	}
	if dto.MaxValue.IsPositive() && dto.MinValue.GreaterThan(dto.MaxValue) {
		return nil, errorsuc.NewValidationError("o valor mínimo não pode ser maior que o máximo")
	}
	variables := string(dto.VariablesJSON)
	if variables == "" {
		variables = "{}"
	}
	if !json.Valid([]byte(variables)) {
		return nil, errorsuc.NewValidationError("a configuração de variáveis está inválida")
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
		PercentValue:           p.PercentValue.InexactFloat64(),
		FixedValue:             p.FixedValue.InexactFloat64(),
		MaxPercent:             p.MaxPercent.InexactFloat64(),
		MaxValue:               p.MaxValue.InexactFloat64(),
		MinGrossValue:          p.MinGrossValue.InexactFloat64(),
		MaxGrossValue:          p.MaxGrossValue.InexactFloat64(),
		MinQuantity:            p.MinQuantity.InexactFloat64(),
		MaxQuantity:            p.MaxQuantity.InexactFloat64(),
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
		PercentValue:   line.PercentValue.InexactFloat64(),
		FixedValue:     line.FixedValue.InexactFloat64(),
		MinValue:       line.MinValue.InexactFloat64(),
		MaxValue:       line.MaxValue.InexactFloat64(),
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
		return nil, errorsuc.NewValidationError("informe o código")
	}
	if dto.Description == "" {
		return nil, errorsuc.NewValidationError("informe a descrição")
	}
	it := &entity.InvoiceType{
		Code:        dto.Code,
		Description: dto.Description,
		Type:        entity.InvoiceTypeKind(dto.Type),
		IsActive:    ptrutil.BoolOrTrue(dto.IsActive),
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
		ICMSPct:                  it.ICMSPct.InexactFloat64(),
		ICMSReductionPct:         it.ICMSReductionPct.InexactFloat64(),
		IPIPct:                   it.IPIPct.InexactFloat64(),
		PISPct:                   it.PISPct.InexactFloat64(),
		COFINSPct:                it.COFINSPct.InexactFloat64(),
		ISSQNPct:                 it.ISSQNPct.InexactFloat64(),
		IRPct:                    it.IRPct.InexactFloat64(),
		CSLLPct:                  it.CSLLPct.InexactFloat64(),
		INSSPct:                  it.INSSPct.InexactFloat64(),
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
		IRPctPresumption:         it.IRPctPresumption.InexactFloat64(),
		CSLLPctPresumption:       it.CSLLPctPresumption.InexactFloat64(),
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
			return nil, errorsuc.NewValidationError("o código da tabela de vendas é obrigatório")
		}
		var err error
		st, err = uc.repo.GetSalesTableByCode(ctx, dto.SalesTableCode)
		if err != nil {
			return nil, fmt.Errorf("tabela de vendas não encontrada: %w", err)
		}
		dto.SalesTableID = st.ID
	} else {
		var err error
		st, err = uc.repo.GetSalesTableByID(ctx, dto.SalesTableID)
		if err != nil {
			return nil, fmt.Errorf("tabela de vendas não encontrada: %w", err)
		}
	}
	if dto.ItemCode == "" {
		return nil, errorsuc.NewValidationError("o código do item é obrigatório")
	}
	if !dto.Price.IsPositive() {
		return nil, errorsuc.NewValidationError("o preço deve ser maior que zero")
	}
	if err := validateManualSalesTablePrice(st, dto.Price); err != nil {
		return nil, err
	}
	if dto.Situation != "" && !validPriceSituations[dto.Situation] {
		return nil, errorsuc.NewValidationError("a situação deve ser ATIVO, INATIVO ou PROMOCIONAL")
	}
	if dto.Situation == "" {
		dto.Situation = "ATIVO"
	}
	p := &entity.SalesTablePrice{
		SalesTableID:  dto.SalesTableID,
		ItemCode:      dto.ItemCode.String(),
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

func validateManualSalesTablePrice(st *entity.SalesTable, price decimal.Decimal) error {
	if st == nil {
		return nil
	}
	switch st.PriceFormation {
	case entity.PriceCustoMedio, entity.PriceCustoStandardTotal, entity.PriceCustoStandardMaterial:
		return errorsuc.NewValidationError(fmt.Sprintf("o preço não pode ser informado manualmente quando a formação da tabela é %s", st.PriceFormation))
	}
	if price.LessThan(decimal.New(1, -2)) && !st.AllowItemsBelowCent {
		return errorsuc.NewValidationError("a tabela não permite preços de item abaixo de 0,01")
	}
	return nil
}

func (uc *CustomerUseCase) UpdateSalesTablePrice(ctx context.Context, dto request.UpdateSalesTablePriceDTO) (*response.SalesTablePriceResponse, error) {
	if dto.ID == 0 {
		return nil, errorsuc.NewValidationError("informe o identificador")
	}
	if dto.Price.IsNegative() {
		return nil, errorsuc.NewValidationError("o preço deve ser maior ou igual a zero")
	}
	current, err := uc.repo.GetSalesTablePriceByID(ctx, dto.ID)
	if err != nil {
		return nil, err
	}
	st, err := uc.repo.GetSalesTableByID(ctx, current.SalesTableID)
	if err != nil {
		return nil, errorsuc.NewValidationError("tabela de vendas não encontrada")
	}
	if err := validateManualSalesTablePrice(st, dto.Price); err != nil {
		return nil, err
	}
	if dto.Situation != "" && !validPriceSituations[dto.Situation] {
		return nil, errorsuc.NewValidationError("a situação deve ser ativo, inativo ou promocional")
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
		return nil, fmt.Errorf("tabela de venda não encontrada: %w", err)
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
		return nil, fmt.Errorf("tabela de venda não encontrada: %w", err)
	}
	return uc.ListSalesTablePrices(ctx, st.ID)
}

func (uc *CustomerUseCase) PriceSalesItem(ctx context.Context, dto request.PriceSalesItemDTO) (*response.SalesItemPricingResponse, error) {
	if dto.SalesTableCode == 0 {
		return nil, errorsuc.NewValidationError("o código da tabela de vendas é obrigatório")
	}
	if dto.ItemCode == "" {
		return nil, errorsuc.NewValidationError("o código do item é obrigatório")
	}
	if !dto.Quantity.IsPositive() {
		dto.Quantity = decimal.NewFromFloat(1)
	}
	st, err := uc.repo.GetSalesTableByCode(ctx, dto.SalesTableCode)
	if err != nil {
		return nil, errorsuc.NewValidationError("tabela de vendas não encontrada")
	}
	if err := validateSalesTableForPricing(st, time.Now()); err != nil {
		return nil, err
	}
	p, err := uc.repo.GetSalesTablePrice(ctx, st.ID, dto.ItemCode.String())
	if err != nil {
		return nil, err
	}
	if p.Blocked {
		return nil, errorsuc.NewValidationError("o preço do item está bloqueado na tabela")
	}
	if p.Situation == entity.PriceSituationInativo {
		return nil, errorsuc.NewValidationError("o preço do item está inativo na tabela")
	}
	return &response.SalesItemPricingResponse{
		SalesTableCode: st.Code,
		SalesTableID:   st.ID,
		ItemCode:       p.ItemCode,
		UnitPrice:      p.Price.InexactFloat64(),
		Quantity:       dto.Quantity.InexactFloat64(),
		TotalGross:     p.Price.Mul(dto.Quantity).InexactFloat64(),
		Source:         "SALES_TABLE",
		Situation:      string(p.Situation),
		Blocked:        p.Blocked,
		UME:            p.UME,
		UMC:            p.UMC,
		Formula:        p.Formula,
	}, nil
}

func (uc *CustomerUseCase) ResolveSalesTablesForItem(ctx context.Context, dto request.ResolveSalesTablesForItemDTO) (*response.SalesTableResolutionResponse, error) {
	if strings.TrimSpace(dto.ItemCode) == "" {
		return nil, errorsuc.NewValidationError("informe o código do item")
	}
	if !dto.Quantity.IsPositive() {
		return nil, errorsuc.NewValidationError("a quantidade deve ser maior que zero")
	}
	if dto.Currency == "" {
		dto.Currency = "BRL"
	}
	if !strings.EqualFold(dto.Currency, "BRL") {
		return nil, errorsuc.NewValidationError("não existem tabelas configuradas para a moeda informada")
	}
	if dto.ReferenceAt.IsZero() {
		dto.ReferenceAt = time.Now()
	}
	var customer *entity.Customer
	if dto.CustomerCode != nil {
		var err error
		customer, err = uc.repo.GetCustomerByCode(ctx, *dto.CustomerCode)
		if err != nil {
			return nil, errorsuc.NewValidationError("cliente não encontrado")
		}
	}
	tables, err := uc.repo.ListSalesTables(ctx, true)
	if err != nil {
		return nil, err
	}
	policies, err := uc.repo.ListCommercialPolicies(ctx, true, nil)
	if err != nil {
		return nil, err
	}
	candidates := make([]response.SalesTableCandidateResponse, 0)
	for _, table := range tables {
		if validateSalesTableForPricing(table, dto.ReferenceAt) != nil {
			continue
		}
		price, priceErr := uc.repo.GetSalesTablePrice(ctx, table.ID, dto.ItemCode)
		if priceErr != nil || price.Blocked || price.Situation == entity.PriceSituationInativo || !price.Price.IsPositive() {
			continue
		}
		unit := ""
		if price.UME != nil {
			unit = *price.UME
		}
		if dto.Unit != "" && !strings.EqualFold(dto.Unit, unit) && (price.UMC == nil || !strings.EqualFold(dto.Unit, *price.UMC)) {
			continue
		}
		priority := 40
		source := "SALES_TABLE"
		if table.TableType == entity.TableTypePromocional {
			priority = 30
		}
		applied := price.Price
		qtd := dto.Quantity
		context := entity.CommercialPolicyContext{GrossValue: price.Price.Mul(qtd), Quantity: qtd, SalesTableID: &table.ID, ItemCode: &dto.ItemCode}
		if customer != nil {
			context.CustomerCode = &customer.Code
			context.CustomerTypeID = customer.CustomerTypeID
			context.MarketSegmentID = customer.MarketSegmentID
			context.RegionID = customer.RegionID
			context.CarrierID = customer.CarrierID
		}
		if evaluation, evaluationErr := entity.EvaluateCommercialPolicies(policies, context); evaluationErr == nil && len(evaluation.Effects) > 0 {
			if qtd.IsPositive() {
				// Calculado em decimal; o preço da tabela ainda é float na resposta.
				applied = evaluation.NetValue.Div(qtd).Round(6)
			}
			if priority > 20 {
				priority = 20
			}
			source = "COMMERCIAL_POLICY"
		}
		candidates = append(candidates, response.SalesTableCandidateResponse{SalesTableCode: table.Code, Description: table.Description, BasePrice: price.Price.InexactFloat64(), AppliedPrice: applied.InexactFloat64(), Unit: unit, Currency: "BRL", Source: source, Priority: priority})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Priority == candidates[j].Priority {
			return candidates[i].SalesTableCode < candidates[j].SalesTableCode
		}
		return candidates[i].Priority < candidates[j].Priority
	})
	result := &response.SalesTableResolutionResponse{Candidates: candidates}
	if len(candidates) == 1 || (len(candidates) > 1 && candidates[0].Priority < candidates[1].Priority) {
		result.AutoSelected = true
		result.Selected = &candidates[0]
	}
	return result, nil
}

func (uc *CustomerUseCase) FormSalesPrice(ctx context.Context, dto request.FormSalesPriceDTO) (*response.SalesPriceFormationResponse, error) {
	decimalPlaces := int16(2)
	if dto.PolicyCode != nil {
		p, err := uc.repo.GetSalesPricePolicyByCode(ctx, *dto.PolicyCode)
		if err != nil {
			return nil, errorsuc.NewValidationError("política de formação de preço não encontrada")
		}
		if err := validateSalesPricePolicyForPricing(p, time.Now()); err != nil {
			return nil, err
		}
		dto.MarkupPct = p.MarkupPct
		dto.MarginPct = p.MarginPct
		if p.IdealMarginPct.IsPositive() {
			dto.MarginPct = p.IdealMarginPct
		}
		dto.ExpensesPct = p.ExpensesPct
		dto.TaxesPct = p.TaxesPct
		dto.FreightPct = p.FreightPct
		dto.CommissionPct = p.CommissionPct
		dto.DiscountPct = p.DiscountPct
		if dto.BaseCost.IsZero() && p.CostSource != entity.SalesCostInformed && dto.ItemCode != "" {
			itemCode, err := strconv.ParseInt(dto.ItemCode.String(), 10, 64)
			if err != nil {
				return nil, errorsuc.NewValidationError("não foi possível resolver internamente o código do item para calcular o custo")
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
			return nil, errorsuc.NewValidationError("tabela de vendas não encontrada")
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
		ItemCode:                dto.ItemCode.String(),
		BaseCost:                formed.BaseCost.InexactFloat64(),
		SuggestedPrice:          formed.SuggestedPrice.InexactFloat64(),
		MarkupPct:               formed.MarkupPct.InexactFloat64(),
		MarginPct:               formed.MarginPct.InexactFloat64(),
		ExpensesPct:             formed.ExpensesPct.InexactFloat64(),
		TaxesPct:                formed.TaxesPct.InexactFloat64(),
		FreightPct:              formed.FreightPct.InexactFloat64(),
		CommissionPct:           formed.CommissionPct.InexactFloat64(),
		DiscountPct:             formed.DiscountPct.InexactFloat64(),
		ContributionMarginPct:   formed.ContributionMarginPct.InexactFloat64(),
		ContributionMarginValue: formed.ContributionMarginValue.InexactFloat64(),
		DecimalPlaces:           decimalPlaces,
	}, nil
}

func (uc *CustomerUseCase) GenerateSalesTablePrices(ctx context.Context, dto request.GenerateSalesTablePricesDTO) (*response.GenerateSalesTablePricesResponse, error) {
	if dto.SalesTableCode == 0 {
		return nil, errorsuc.NewValidationError("o código da tabela de vendas é obrigatório")
	}
	if dto.PolicyCode == 0 {
		return nil, errorsuc.NewValidationError("o código da política de preço é obrigatório")
	}
	if len(dto.ItemCodes) == 0 {
		return nil, errorsuc.NewValidationError("informe ao menos um código de item")
	}
	st, err := uc.repo.GetSalesTableByCode(ctx, dto.SalesTableCode)
	if err != nil {
		return nil, errorsuc.NewValidationError("tabela de vendas não encontrada")
	}
	if err := validateSalesTableForPricing(st, time.Now()); err != nil {
		return nil, err
	}
	policy, err := uc.repo.GetSalesPricePolicyByCode(ctx, dto.PolicyCode)
	if err != nil {
		return nil, errorsuc.NewValidationError("política de formação de preço não encontrada")
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
		itemCode, err := strconv.ParseInt(rawItemCode.String(), 10, 64)
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("item %s ignorado: não foi possível resolver internamente o código para calcular o custo", rawItemCode))
			continue
		}
		baseCost, costSource, err := uc.repo.ResolveSalesCost(ctx, itemCode, "", policy.CostSource, dto.WarehouseID)
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("item %s ignorado: %v", rawItemCode, err))
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
			out.Warnings = append(out.Warnings, fmt.Sprintf("item %s ignorado: %v", rawItemCode, err))
			continue
		}
		priceRow := &entity.SalesTablePrice{
			SalesTableID: st.ID,
			ItemCode:     rawItemCode.String(),
			Price:        formed.SuggestedPrice,
			PriceConv:    formed.SuggestedPrice,
			Situation:    entity.PriceSituationAtivo,
		}
		formula := fmt.Sprintf("POLICY:%d/%s", policy.Code, costSource)
		priceRow.Formula = &formula
		saved, oldPrice, err := uc.repo.UpsertSalesTablePrice(ctx, priceRow)
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("item %s ignorado: %v", rawItemCode, err))
			continue
		}
		base := baseCost
		policyCode := policy.Code
		history, err := uc.repo.CreateSalesTablePriceHistory(ctx, &entity.SalesTablePriceHistory{
			SalesTablePriceID: &saved.ID,
			SalesTableID:      st.ID,
			SalesTableCode:    st.Code,
			ItemCode:          rawItemCode.String(),
			OldPrice:          oldPrice,
			NewPrice:          saved.Price,
			BaseCost:          &base,
			Source:            "POLICY",
			PolicyCode:        &policyCode,
			Reason:            dto.Reason,
		})
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("item %s reprecificado sem histórico: %v", rawItemCode, err))
		}
		var historyID int64
		if history != nil {
			historyID = history.ID
		}
		out.Generated = append(out.Generated, response.GeneratedSalesTablePriceResponse{
			ItemCode:   rawItemCode.String(),
			BaseCost:   baseCost.InexactFloat64(),
			CostSource: costSource,
			OldPrice:   fptrD(oldPrice),
			Price:      saved.Price.InexactFloat64(),
			HistoryID:  historyID,
			PriceRow:   toSalesTablePriceResponse(saved),
		})
	}
	return out, nil
}

func effectivePolicyMargin(policy *entity.SalesPricePolicy) decimal.Decimal {
	if policy.IdealMarginPct.IsPositive() {
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
		OldPrice:          fptrD(h.OldPrice),
		NewPrice:          h.NewPrice.InexactFloat64(),
		BaseCost:          fptrD(h.BaseCost),
		Source:            h.Source,
		PolicyCode:        h.PolicyCode,
		Reason:            h.Reason,
		CreatedAt:         h.CreatedAt,
	}
}

func validateSalesTableForPricing(st *entity.SalesTable, now time.Time) error {
	if !st.IsActive {
		return errorsuc.NewValidationError("a tabela de venda está inativa")
	}
	if st.ValidityStart != nil && now.Before(*st.ValidityStart) {
		return errorsuc.NewValidationError("a vigência da tabela de venda ainda não começou")
	}
	if st.ValidityEnd != nil && now.After(st.ValidityEnd.Add(24*time.Hour)) {
		return errorsuc.NewValidationError("a vigência da tabela de venda expirou")
	}
	return nil
}

func validateSalesPricePolicyForPricing(p *entity.SalesPricePolicy, now time.Time) error {
	if !p.IsActive {
		return errorsuc.NewValidationError("a política de preço de venda está inativa")
	}
	if p.ValidityStart != nil && now.Before(*p.ValidityStart) {
		return errorsuc.NewValidationError("a vigência da política de preço de venda ainda não começou")
	}
	if p.ValidityEnd != nil && now.After(p.ValidityEnd.Add(24*time.Hour)) {
		return errorsuc.NewValidationError("a vigência da política de preço expirou")
	}
	margin := effectivePolicyMargin(p)
	if margin.LessThan(p.MinMarginPct) {
		return errorsuc.NewValidationError("a margem da política de preço é menor que a margem mínima")
	}
	if p.MaxMarginPct.IsPositive() && margin.GreaterThan(p.MaxMarginPct) {
		return errorsuc.NewValidationError("a margem da política de preço é maior que a margem máxima")
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
		Price:         p.Price.InexactFloat64(),
		UME:           p.UME,
		UMC:           p.UMC,
		PriceConv:     p.PriceConv.InexactFloat64(),
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
	fillTaxType(tt, dto)
	created, err := uc.repo.CreateTaxType(ctx, tt)
	if err != nil {
		return nil, err
	}
	return toTaxTypeResponse(created), nil
}

func fillTaxType(tt *entity.TaxType, dto request.CreateTaxTypeDTO) {
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
}

// UpdateTaxType altera o tipo de imposto (antes só dava para criar).
func (uc *CustomerUseCase) UpdateTaxType(ctx context.Context, code int64, dto request.CreateTaxTypeDTO) (*response.TaxTypeResponse, error) {
	current, err := uc.repo.GetTaxTypeByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	tt, err := entity.NewTaxType(code, dto.Description)
	if err != nil {
		return nil, err
	}
	fillTaxType(tt, dto)
	tt.ID, tt.IsActive = current.ID, current.IsActive
	updated, err := uc.repo.UpdateTaxType(ctx, tt)
	if err != nil {
		return nil, err
	}
	return toTaxTypeResponse(updated), nil
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
	c.SuframaExpiry = dto.SuframaExpiry
	if err := uc.applyCustomerDTO(ctx, c, dto); err != nil {
		return nil, err
	}
	if err := uc.rejectDuplicateDocument(ctx, c); err != nil {
		return nil, err
	}

	// O limite nasce da análise de crédito (credit_uc.go). Sem o módulo de
	// crédito ligado (testes), grava o que veio.
	var analise *creditAnalysis
	if uc.life != nil {
		analise, err = uc.analyzeCredit(ctx, nil, c.DocumentType, c.DocumentNumber, nil)
		if err != nil {
			return nil, err
		}
		approved := analise.suggestion.SuggestedLimit
		// A tela antiga digitava o limite sem motivo: vale o sugerido.
		if dto.CreditLimit != nil && dto.FormVersion >= 2 {
			approved = *dto.CreditLimit
		}
		if _, err := uc.checkApproval(ctx, analise, approved, dto.CreditLimitReason); err != nil {
			return nil, err
		}
		c.CreditLimit = approved
	} else if dto.CreditLimit != nil {
		c.CreditLimit = *dto.CreditLimit
	}

	created, err := uc.repo.CreateCustomer(ctx, c)
	if err != nil {
		return nil, err
	}
	if analise != nil {
		// Cliente com limite e sem trilha não pode ficar: se a trilha não
		// gravar, desfaz o cadastro recém-criado (ainda sem nenhum uso).
		err := uc.recordCredit(ctx, created, analise, c.CreditLimit, dto.CreditLimitReason, "CADASTRO")
		if err == nil && analise.company != nil {
			err = uc.life.SetCustomerRegistration(ctx, created.Code, snapshotOf(analise.company))
		}
		if err != nil {
			if derr := uc.life.DeleteCustomer(ctx, created.Code, entity.CheckCanDelete); derr != nil {
				return nil, fmt.Errorf("gravando a análise de crédito: %w (e o cadastro %d não pôde ser desfeito: %v)", err, created.Code, derr)
			}
			return nil, err
		}
		if created, err = uc.repo.GetCustomerByCode(ctx, created.Code); err != nil {
			return nil, err
		}
	}
	return uc.customerResponse(ctx, created)
}

// UpdateCustomer substitui os dados editáveis do cliente pelo que a tela
// mandou (a VCLI0500 manda o formulário inteiro: campo vazio limpa). Código,
// documento e limite de crédito não mudam por aqui.
func (uc *CustomerUseCase) UpdateCustomer(ctx context.Context, code int64, dto request.CreateCustomerDTO) (*response.CustomerResponse, error) {
	if strings.TrimSpace(dto.Name) == "" {
		return nil, errorsuc.NewValidationError("informe o nome")
	}
	existing, err := uc.repo.GetCustomerByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if dto.FormVersion < 2 {
		preserveNewFields(&dto, existing)
	}
	existing.Name = dto.Name
	existing.SuframaExpiry = dto.SuframaExpiry
	if err := uc.applyCustomerDTO(ctx, existing, dto); err != nil {
		return nil, err
	}
	updated, err := uc.repo.UpdateCustomer(ctx, existing)
	if err != nil {
		return nil, err
	}
	return uc.customerResponse(ctx, updated)
}

var (
	ieIndicators = map[string]bool{"CONTRIBUINTE": true, "ISENTO": true, "NAO_CONTRIBUINTE": true}
	taxRegimes   = map[string]bool{"SIMPLES": true, "MEI": true, "NORMAL": true}
	freightTypes = map[string]bool{"CIF": true, "FOB": true, "TERCEIROS": true, "SEM_FRETE": true}
	emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	cnaePattern  = regexp.MustCompile(`^\d{7}$`)
)

// applyCustomerDTO valida e copia os campos editáveis (inclusão e alteração).
func (uc *CustomerUseCase) applyCustomerDTO(ctx context.Context, c *entity.Customer, dto request.CreateCustomerDTO) error {
	c.CorporateCode = dto.CorporateCode
	c.IsCorporate = dto.IsCorporate
	c.TradeName = blankToNil(dto.TradeName)
	c.StateRegistration = blankToNil(dto.StateRegistration)
	c.MunicipalRegistration = blankToNil(dto.MunicipalRegistration)
	c.SuframaCode = blankToNil(dto.SuframaCode)
	c.Website = blankToNil(dto.Website)
	c.FinalidadeCompra = nil
	if dto.FinalidadeCompra != nil {
		f, err := finalidadeDoCliente(*dto.FinalidadeCompra)
		if err != nil {
			return err
		}
		c.FinalidadeCompra = f
	}
	if c.CorporateCode != nil && *c.CorporateCode == c.Code {
		return errorsuc.NewValidationError("o cliente não pode ser matriz de si mesmo")
	}
	if c.SuframaCode != nil && c.SuframaExpiry == nil {
		return errorsuc.NewValidationError("informe a validade da inscrição SUFRAMA")
	}

	var err error
	if c.IEIndicator, err = enumOrNil(dto.IEIndicator, ieIndicators, "indicador de IE"); err != nil {
		return err
	}
	if c.IEIndicator != nil && *c.IEIndicator == "ISENTO" && c.StateRegistration == nil {
		// A NF-e só declara contribuinte isento (indIEDest 2) com IE "ISENTO".
		isento := "ISENTO"
		c.StateRegistration = &isento
	}
	if c.IEIndicator != nil {
		ie := strings.ToUpper(strings.TrimSpace(derefString(c.StateRegistration)))
		switch {
		case *c.IEIndicator == "CONTRIBUINTE" && (ie == "" || ie == "ISENTO"):
			return errorsuc.NewValidationError("cliente contribuinte precisa da inscrição estadual")
		case *c.IEIndicator == "NAO_CONTRIBUINTE" && ie != "" && ie != "ISENTO":
			return errorsuc.NewValidationError("cliente não contribuinte não tem inscrição estadual; limpe a IE ou mude o indicador")
		}
	}
	if c.TaxRegime, err = enumOrNil(dto.TaxRegime, taxRegimes, "regime tributário"); err != nil {
		return err
	}
	if c.FreightType, err = enumOrNil(dto.FreightType, freightTypes, "tipo de frete"); err != nil {
		return err
	}
	c.MainCNAE = blankToNil(dto.MainCNAE)
	if c.MainCNAE != nil {
		digits := strings.NewReplacer(".", "", "-", "", "/", "", " ", "").Replace(*c.MainCNAE)
		if !cnaePattern.MatchString(digits) {
			return errorsuc.NewValidationError("o CNAE tem 7 dígitos")
		}
		c.MainCNAE = &digits
	}
	for _, e := range []struct {
		v     *string
		label string
	}{{dto.NFeEmail, "e-mail para NF-e"}, {dto.BillingEmail, "e-mail de cobrança"}} {
		if v := blankToNil(e.v); v != nil {
			for _, part := range strings.Split(*v, ";") {
				if !emailPattern.MatchString(strings.TrimSpace(part)) {
					return errorsuc.NewValidationError(fmt.Sprintf("%s inválido: %s", e.label, part))
				}
			}
		}
	}
	c.NFeEmail = blankToNil(dto.NFeEmail)
	c.BillingEmail = blankToNil(dto.BillingEmail)
	if dto.RepresentativeCode != nil && *dto.RepresentativeCode != 0 && uc.reps != nil &&
		(c.RepresentativeCode == nil || *c.RepresentativeCode != *dto.RepresentativeCode) {
		if err := representativevalidation.Validate(ctx, uc.reps, dto.RepresentativeCode); err != nil {
			return err
		}
	}
	c.RepresentativeCode = dto.RepresentativeCode
	if c.RepresentativeCode != nil && *c.RepresentativeCode == 0 {
		c.RepresentativeCode = nil
	}
	c.ShippingCarrierID = dto.ShippingCarrierID
	c.RequiresCustomerPO = dto.RequiresCustomerPO
	c.RequiresQualityCert = dto.RequiresQualityCert
	c.AllowsPartialBilling = dto.AllowsPartialBilling == nil || *dto.AllowsPartialBilling
	c.ReceivingNotes = blankToNil(dto.ReceivingNotes)
	c.CommercialNotes = blankToNil(dto.CommercialNotes)
	c.FinancialNotes = blankToNil(dto.FinancialNotes)

	if c.CorporateCode != nil {
		if _, err := uc.repo.GetCustomerByCode(ctx, *c.CorporateCode); err != nil {
			return errorsuc.NewValidationError(fmt.Sprintf("matriz %d não encontrada", *c.CorporateCode))
		}
	}
	return uc.resolveCustomerFKs(ctx, c, dto)
}

// rejectDuplicateDocument: o mesmo CNPJ/CPF não vira dois clientes na empresa
// (filiais têm CNPJ próprio; o grupo é ligado pela raiz).
func (uc *CustomerUseCase) rejectDuplicateDocument(ctx context.Context, c *entity.Customer) error {
	if uc.life != nil && onlyDigits(c.DocumentNumber) != "" {
		// Compara só os dígitos: "11.222.333/0001-81" e "11222333000181" são o mesmo.
		code, name, found, err := uc.life.FindCustomerByDocumentDigits(ctx, onlyDigits(c.DocumentNumber))
		if err != nil {
			return err
		}
		if found && code != c.Code {
			return errorsuc.NewConflictError(fmt.Sprintf("o documento %s já está no cliente %d — %s", c.DocumentNumber, code, name))
		}
		return nil
	}
	other, err := uc.repo.GetCustomerByDocument(ctx, c.DocumentNumber)
	if err == nil && other != nil && other.Code != c.Code {
		return errorsuc.NewConflictError(fmt.Sprintf("o documento %s já está no cliente %d — %s", c.DocumentNumber, other.Code, other.Name))
	}
	return nil
}

// preserveNewFields: a tela antiga não manda os campos da VCLI0500 revista;
// sem isto, editar um cliente nela apagaria IE, e-mails, logística e exigências.
func preserveNewFields(dto *request.CreateCustomerDTO, c *entity.Customer) {
	dto.IEIndicator, dto.TaxRegime, dto.MainCNAE = c.IEIndicator, c.TaxRegime, c.MainCNAE
	dto.NFeEmail, dto.BillingEmail = c.NFeEmail, c.BillingEmail
	dto.RepresentativeCode, dto.ShippingCarrierID, dto.FreightType = c.RepresentativeCode, c.ShippingCarrierID, c.FreightType
	dto.RequiresCustomerPO, dto.RequiresQualityCert = c.RequiresCustomerPO, c.RequiresQualityCert
	partial := c.AllowsPartialBilling
	dto.AllowsPartialBilling = &partial
	dto.ReceivingNotes, dto.CommercialNotes, dto.FinancialNotes = c.ReceivingNotes, c.CommercialNotes, c.FinancialNotes
}

func blankToNil(v *string) *string {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	t := strings.TrimSpace(*v)
	return &t
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func enumOrNil(v *string, allowed map[string]bool, label string) (*string, error) {
	t := blankToNil(v)
	if t == nil {
		return nil, nil
	}
	u := strings.ToUpper(*t)
	if !allowed[u] {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("%s inválido: %s", label, *t))
	}
	return &u, nil
}

// resolveCustomerFKs troca os códigos dos cadastros de apoio pelos ids.
// Código ausente limpa o vínculo.
func (uc *CustomerUseCase) resolveCustomerFKs(ctx context.Context, c *entity.Customer, dto request.CreateCustomerDTO) error {
	c.RegionID, c.MarketSegmentID, c.CustomerTypeID, c.CarrierID, c.TaxTypeID = nil, nil, nil, nil, nil
	if dto.RegionCode != nil {
		r, err := uc.repo.GetRegionByCode(ctx, *dto.RegionCode)
		if err != nil {
			return errorsuc.NewValidationError(fmt.Sprintf("região %d não encontrada", *dto.RegionCode))
		}
		c.RegionID = &r.ID
	}
	if dto.MarketSegmentCode != nil {
		s, err := uc.repo.GetMarketSegmentByCode(ctx, *dto.MarketSegmentCode)
		if err != nil {
			return errorsuc.NewValidationError(fmt.Sprintf("segmento %d não encontrado", *dto.MarketSegmentCode))
		}
		c.MarketSegmentID = &s.ID
	}
	if dto.CustomerTypeCode != nil {
		t, err := uc.repo.GetCustomerTypeByCode(ctx, *dto.CustomerTypeCode)
		if err != nil {
			return errorsuc.NewValidationError(fmt.Sprintf("tipo de cliente %d não encontrado", *dto.CustomerTypeCode))
		}
		c.CustomerTypeID = &t.ID
	}
	if dto.CarrierCode != nil {
		car, err := uc.repo.GetCarrierByCode(ctx, *dto.CarrierCode)
		if err != nil {
			return errorsuc.NewValidationError(fmt.Sprintf("portador %d não encontrado", *dto.CarrierCode))
		}
		c.CarrierID = &car.ID
	}
	if dto.TaxTypeCode != nil {
		tt, err := uc.repo.GetTaxTypeByCode(ctx, *dto.TaxTypeCode)
		if err != nil {
			return errorsuc.NewValidationError(fmt.Sprintf("tipo de imposto %d não encontrado", *dto.TaxTypeCode))
		}
		c.TaxTypeID = &tt.ID
	}
	return nil
}

func (uc *CustomerUseCase) GetCustomer(ctx context.Context, code int64) (*response.CustomerResponse, error) {
	c, err := uc.repo.GetCustomerByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	resp, err := uc.customerResponse(ctx, c)
	if err != nil {
		return nil, err
	}
	// Load addresses and contacts
	addrs, errEnderecos := uc.repo.ListAddresses(ctx, c.ID)
	if errEnderecos != nil {
		return nil, fmt.Errorf("lendo os endereços do cliente: %w", errEnderecos)
	}
	for _, a := range addrs {
		resp.Addresses = append(resp.Addresses, toAddressResponse(a))
	}
	contacts, errContatos := uc.repo.ListContacts(ctx, c.ID)
	if errContatos != nil {
		return nil, fmt.Errorf("lendo os contatos do cliente: %w", errContatos)
	}
	var tipos []*entity.CustomerContactType
	if len(contacts) > 0 {
		if tipos, err = uc.repo.ListContactTypes(ctx, false); err != nil {
			return nil, err
		}
	}
	for _, ct := range contacts {
		r := toContactResponse(ct)
		for _, t := range tipos {
			if ct.ContactTypeID != nil && t.ID == *ct.ContactTypeID {
				r.ContactTypeCode = &t.Code
			}
		}
		resp.Contacts = append(resp.Contacts, r)
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
		r := toCustomerResponse(c)
		r.CreditLimitExpired = c.CreditLimitExpiresAt != nil && c.CreditLimitExpiresAt.Before(dateOnly(uc.now()))
		out = append(out, r)
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
		r := toCustomerResponse(c)
		r.CreditLimitExpired = c.CreditLimitExpiresAt != nil && c.CreditLimitExpiresAt.Before(dateOnly(uc.now()))
		out = append(out, r)
	}
	return out, nil
}

func (uc *CustomerUseCase) BlockCustomer(ctx context.Context, dto request.BlockCustomerDTO) error {
	if dto.Reason == "" {
		return errorsuc.NewValidationError("informe o motivo do bloqueio")
	}
	return uc.repo.BlockCustomer(ctx, dto.CustomerCode, dto.Reason)
}

func (uc *CustomerUseCase) UnblockCustomer(ctx context.Context, code int64) error {
	return uc.repo.UnblockCustomer(ctx, code)
}

func (uc *CustomerUseCase) AddAddress(ctx context.Context, dto request.AddAddressDTO) (*response.CustomerAddressResponse, error) {
	c, err := uc.repo.GetCustomerByCode(ctx, dto.CustomerCode)
	if err != nil {
		return nil, fmt.Errorf("cliente não encontrado: %w", err)
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
		return nil, fmt.Errorf("cliente não encontrado: %w", err)
	}
	if dto.Name == "" {
		return nil, errorsuc.NewValidationError("informe o nome do contato")
	}
	var contactTypeID *int64
	if dto.ContactTypeCode != nil {
		ct, err := uc.repo.GetContactTypeByCode(ctx, *dto.ContactTypeCode)
		if err != nil {
			return nil, fmt.Errorf("tipo de contato não encontrado: %w", err)
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
	r := &response.CustomerResponse{
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
		CarrierID:             c.CarrierID,
		TaxTypeID:             c.TaxTypeID,
		FinalidadeCompra:      c.FinalidadeCompra,
		CreditLimit:           c.CreditLimit.InexactFloat64(),
		Website:               c.Website,
		IsActive:              c.IsActive,
		Blocked:               c.Blocked,
		BlockReason:           c.BlockReason,
		CreatedAt:             c.CreatedAt,
		UpdatedAt:             c.UpdatedAt,
		IEIndicator:           c.IEIndicator,
		TaxRegime:             c.TaxRegime,
		MainCNAE:              c.MainCNAE,
		NFeEmail:              c.NFeEmail,
		BillingEmail:          c.BillingEmail,
		RepresentativeCode:    c.RepresentativeCode,
		ShippingCarrierID:     c.ShippingCarrierID,
		FreightType:           c.FreightType,
		RequiresCustomerPO:    c.RequiresCustomerPO,
		RequiresQualityCert:   c.RequiresQualityCert,
		AllowsPartialBilling:  c.AllowsPartialBilling,
		ReceivingNotes:        c.ReceivingNotes,
		CommercialNotes:       c.CommercialNotes,
		FinancialNotes:        c.FinancialNotes,
		RegistrationStatus:    c.RegistrationStatus,
		RegistrationCheckedAt: c.RegistrationCheckedAt,
		OpeningDate:           c.OpeningDate,
		ShareCapital:          floatPtr(c.ShareCapital),
		CompanySize:           c.CompanySize,
		LegalNature:           c.LegalNature,
		CreditScore:           c.CreditScore,
		CreditRiskClass:       c.CreditRiskClass,
		CreditLimitSuggested:  floatPtr(c.CreditLimitSuggested),
		CreditLimitExpiresAt:  c.CreditLimitExpiresAt,
		CreditReviewedAt:      c.CreditReviewedAt,
	}
	return r
}

func floatPtr(d *decimal.Decimal) *float64 {
	if d == nil {
		return nil
	}
	f := d.InexactFloat64()
	return &f
}

// customerResponse completa a resposta com os códigos dos cadastros de apoio
// (a tela trabalha com código; o banco guarda o id) e a validade do limite.
func (uc *CustomerUseCase) customerResponse(ctx context.Context, c *entity.Customer) (*response.CustomerResponse, error) {
	r := toCustomerResponse(c)
	if c.CreditLimitExpiresAt != nil {
		r.CreditLimitExpired = c.CreditLimitExpiresAt.Before(dateOnly(uc.now()))
	}
	if c.RegionID != nil {
		list, err := uc.repo.ListRegions(ctx, false)
		if err != nil {
			return nil, err
		}
		for _, x := range list {
			if x.ID == *c.RegionID {
				r.RegionCode = &x.Code
			}
		}
	}
	if c.MarketSegmentID != nil {
		list, err := uc.repo.ListMarketSegments(ctx, false)
		if err != nil {
			return nil, err
		}
		for _, x := range list {
			if x.ID == *c.MarketSegmentID {
				r.MarketSegmentCode = &x.Code
			}
		}
	}
	if c.CustomerTypeID != nil {
		list, err := uc.repo.ListCustomerTypes(ctx, false)
		if err != nil {
			return nil, err
		}
		for _, x := range list {
			if x.ID == *c.CustomerTypeID {
				r.CustomerTypeCode = &x.Code
			}
		}
	}
	if c.CarrierID != nil {
		car, err := uc.repo.GetCarrierByID(ctx, *c.CarrierID)
		if err != nil {
			return nil, err
		}
		r.CarrierCode = &car.Code
	}
	if c.TaxTypeID != nil {
		list, err := uc.repo.ListTaxTypes(ctx, false)
		if err != nil {
			return nil, err
		}
		for _, x := range list {
			if x.ID == *c.TaxTypeID {
				r.TaxTypeCode = &x.Code
			}
		}
	}
	return r, nil
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
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

// finalidadeDoCliente valida a finalidade da compra; vazia limpa (nil).
func finalidadeDoCliente(v string) (*string, error) {
	f := strings.ToUpper(strings.TrimSpace(v))
	if f == "" {
		return nil, nil
	}
	switch f {
	case "REVENDA", "INDUSTRIALIZACAO", "USO_CONSUMO":
		return &f, nil
	}
	return nil, errorsuc.NewValidationError("finalidade da compra inválida: use REVENDA, INDUSTRIALIZACAO ou USO_CONSUMO")
}

// fptrD: o percentual ou valor opcional em decimal, como número na resposta.
func fptrD(d *decimal.Decimal) *float64 {
	if d == nil {
		return nil
	}
	f := d.InexactFloat64()
	return &f
}
