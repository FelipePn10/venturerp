package procurement_uc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/procurement/entity"
)

// ---- Receiving notice + divergences (FAVR) ----

func (uc *UseCase) CreateReceivingNotice(ctx context.Context, dto request.CreateReceivingNoticeDTO) (*response.ReceivingNoticeResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.EnterpriseCode == 0 {
		dto.EnterpriseCode = 1
	}
	scheduledAt, err := parseTimePtr(dto.ScheduledAt)
	if err != nil {
		return nil, fmt.Errorf("invalid scheduled_at: %w", err)
	}
	actor, _ := uc.Auth.UserID(ctx)
	notice := &entity.ReceivingNotice{
		EnterpriseCode:    dto.EnterpriseCode,
		SupplierCode:      dto.SupplierCode,
		PurchaseOrderCode: dto.PurchaseOrderCode,
		CarrierCode:       dto.CarrierCode,
		Status:            "SCHEDULED",
		Dock:              dto.Dock,
		ScheduledAt:       scheduledAt,
		InvoiceNumber:     dto.InvoiceNumber,
		Notes:             dto.Notes,
		CreatedBy:         &actor,
	}
	for _, it := range dto.Items {
		if it.ItemCode <= 0 {
			return nil, fmt.Errorf("receiving notice item requires item_code")
		}
		notice.Items = append(notice.Items, &entity.ReceivingNoticeItem{
			PurchaseOrderItemCode: it.PurchaseOrderItemCode,
			ItemCode:              it.ItemCode,
			Mask:                  it.Mask,
			ExpectedQty:           it.ExpectedQty,
			Unit:                  it.Unit,
			Notes:                 it.Notes,
		})
	}
	created, err := uc.Repo.CreateReceivingNotice(ctx, notice)
	if err != nil {
		return nil, err
	}
	return toReceivingNoticeResponse(created), nil
}

func (uc *UseCase) GetReceivingNotice(ctx context.Context, id int64) (*response.ReceivingNoticeResponse, error) {
	notice, err := uc.Repo.GetReceivingNotice(ctx, id)
	if err != nil {
		return nil, err
	}
	return toReceivingNoticeResponse(notice), nil
}

func (uc *UseCase) ListReceivingNotices(ctx context.Context, status string) ([]*response.ReceivingNoticeResponse, error) {
	notices, err := uc.Repo.ListReceivingNotices(ctx, status)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ReceivingNoticeResponse, 0, len(notices))
	for _, n := range notices {
		out = append(out, toReceivingNoticeResponse(n))
	}
	return out, nil
}

func (uc *UseCase) UpdateReceivingNoticeStatus(ctx context.Context, id int64, dto request.UpdateReceivingNoticeStatusDTO) (*response.ReceivingNoticeResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if !validNoticeStatus(dto.Status) {
		return nil, fmt.Errorf("invalid status %q", dto.Status)
	}
	notice, err := uc.Repo.UpdateReceivingNoticeStatus(ctx, id, dto.Status, dto.Blocked)
	if err != nil {
		return nil, err
	}
	return toReceivingNoticeResponse(notice), nil
}

func (uc *UseCase) CreateReceivingDivergence(ctx context.Context, dto request.CreateReceivingDivergenceDTO) (*response.ReceivingDivergenceResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if !validDivergenceType(dto.DivergenceType) {
		return nil, fmt.Errorf("invalid divergence_type %q", dto.DivergenceType)
	}
	actor, _ := uc.Auth.UserID(ctx)
	div := &entity.ReceivingDivergence{
		NoticeID:              dto.NoticeID,
		PurchaseOrderCode:     dto.PurchaseOrderCode,
		PurchaseOrderItemCode: dto.PurchaseOrderItemCode,
		SupplierCode:          dto.SupplierCode,
		ItemCode:              dto.ItemCode,
		Mask:                  dto.Mask,
		DivergenceType:        dto.DivergenceType,
		ExpectedQty:           dto.ExpectedQty,
		ActualQty:             dto.ActualQty,
		ExpectedPrice:         dto.ExpectedPrice,
		ActualPrice:           dto.ActualPrice,
		Resolution:            "PENDING",
		AffectsSupplierScore:  dto.AffectsSupplierScore,
		Notes:                 dto.Notes,
		CreatedBy:             &actor,
	}
	created, err := uc.Repo.CreateReceivingDivergence(ctx, div)
	if err != nil {
		return nil, err
	}
	return toDivergenceResponse(created), nil
}

func (uc *UseCase) ListReceivingDivergences(ctx context.Context, supplierCode *int64, resolution string) ([]*response.ReceivingDivergenceResponse, error) {
	rows, err := uc.Repo.ListReceivingDivergences(ctx, supplierCode, resolution)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ReceivingDivergenceResponse, 0, len(rows))
	for _, d := range rows {
		out = append(out, toDivergenceResponse(d))
	}
	return out, nil
}

func (uc *UseCase) ResolveReceivingDivergence(ctx context.Context, id int64, dto request.ResolveReceivingDivergenceDTO) (*response.ReceivingDivergenceResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if !validDivergenceResolution(dto.Resolution) {
		return nil, fmt.Errorf("invalid resolution %q", dto.Resolution)
	}
	updated, err := uc.Repo.ResolveReceivingDivergence(ctx, id, dto.Resolution)
	if err != nil {
		return nil, err
	}
	return toDivergenceResponse(updated), nil
}

// ---- Supplier EDI (FEDS) ----

func (uc *UseCase) CreateEDIMessage(ctx context.Context, dto request.CreateEDIMessageDTO) (*response.SupplierEDIMessageResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.EnterpriseCode == 0 {
		dto.EnterpriseCode = 1
	}
	if dto.Direction != "INBOUND" && dto.Direction != "OUTBOUND" {
		return nil, fmt.Errorf("direction must be INBOUND or OUTBOUND")
	}
	if !validEDIMessageType(dto.MessageType) {
		return nil, fmt.Errorf("invalid message_type %q", dto.MessageType)
	}
	payload := dto.Payload
	if len(payload) == 0 || !json.Valid(payload) {
		payload = json.RawMessage(`{}`)
	}
	actor, _ := uc.Auth.UserID(ctx)
	msg := &entity.SupplierEDIMessage{
		EnterpriseCode:    dto.EnterpriseCode,
		SupplierCode:      dto.SupplierCode,
		Direction:         dto.Direction,
		MessageType:       dto.MessageType,
		PurchaseOrderCode: dto.PurchaseOrderCode,
		ExternalReference: dto.ExternalReference,
		Payload:           payload,
		Notes:             dto.Notes,
		CreatedBy:         &actor,
	}
	divergences := 0
	for _, l := range dto.Lines {
		confirmedDate, err := parseDatePtr(l.ConfirmedDate)
		if err != nil {
			return nil, fmt.Errorf("invalid confirmed_date: %w", err)
		}
		poDate, err := parseDatePtr(l.PoDate)
		if err != nil {
			return nil, fmt.Errorf("invalid po_date: %w", err)
		}
		line := &entity.SupplierEDILine{
			PurchaseOrderItemCode: l.PurchaseOrderItemCode,
			ItemCode:              l.ItemCode,
			Mask:                  l.Mask,
			ConfirmedQty:          l.ConfirmedQty,
			ConfirmedPrice:        l.ConfirmedPrice,
			ConfirmedDate:         confirmedDate,
			Notes:                 l.Notes,
		}
		// Detect divergence against the PO reference values, when they were provided.
		if l.PoQty > 0 || l.PoPrice > 0 {
			code := entity.DetectEDILineDivergence(l.PoQty, l.PoPrice, poDate, l.ConfirmedQty, l.ConfirmedPrice, confirmedDate, dto.QtyTolerance, dto.PriceTolerance)
			if code != "" {
				line.Divergence = &code
				divergences++
			}
		}
		msg.Lines = append(msg.Lines, line)
	}
	msg.DivergenceCount = divergences
	switch {
	case dto.Direction == "OUTBOUND":
		msg.Status = "SENT"
		now := time.Now()
		msg.ProcessedAt = &now
	case divergences > 0:
		msg.Status = "WITH_DIVERGENCE"
	default:
		msg.Status = "PROCESSED"
		now := time.Now()
		msg.ProcessedAt = &now
	}
	created, err := uc.Repo.CreateEDIMessage(ctx, msg)
	if err != nil {
		return nil, err
	}
	return toEDIMessageResponse(created), nil
}

func (uc *UseCase) GetEDIMessage(ctx context.Context, id int64) (*response.SupplierEDIMessageResponse, error) {
	msg, err := uc.Repo.GetEDIMessage(ctx, id)
	if err != nil {
		return nil, err
	}
	return toEDIMessageResponse(msg), nil
}

func (uc *UseCase) ListEDIMessages(ctx context.Context, supplierCode *int64) ([]*response.SupplierEDIMessageResponse, error) {
	msgs, err := uc.Repo.ListEDIMessages(ctx, supplierCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.SupplierEDIMessageResponse, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, toEDIMessageResponse(m))
	}
	return out, nil
}

// ---- Import landed cost (FREC0203 / FIMP) ----

func (uc *UseCase) CreateImportProcess(ctx context.Context, dto request.CreateImportProcessDTO) (*response.ImportProcessResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.EnterpriseCode == 0 {
		dto.EnterpriseCode = 1
	}
	if dto.Currency == "" {
		dto.Currency = "USD"
	}
	if dto.ExchangeRate <= 0 {
		return nil, fmt.Errorf("exchange_rate must be positive")
	}
	if dto.ApportionBasis == "" {
		dto.ApportionBasis = "VALUE"
	}
	if !validApportionBasis(dto.ApportionBasis) {
		return nil, fmt.Errorf("apportion_basis must be VALUE, WEIGHT or QUANTITY")
	}
	if len(dto.Items) == 0 {
		return nil, fmt.Errorf("at least one import item is required")
	}
	actor, _ := uc.Auth.UserID(ctx)
	process := &entity.ImportProcess{
		EnterpriseCode:    dto.EnterpriseCode,
		SupplierCode:      dto.SupplierCode,
		PurchaseOrderCode: dto.PurchaseOrderCode,
		Reference:         dto.Reference,
		Incoterm:          dto.Incoterm,
		Currency:          dto.Currency,
		ExchangeRate:      dto.ExchangeRate,
		ApportionBasis:    dto.ApportionBasis,
		Status:            "OPEN",
		Notes:             dto.Notes,
		CreatedBy:         &actor,
	}
	for _, it := range dto.Items {
		if it.ItemCode <= 0 || it.Quantity <= 0 {
			return nil, fmt.Errorf("import item requires item_code and positive quantity")
		}
		process.Items = append(process.Items, &entity.ImportProcessItem{
			ItemCode:     it.ItemCode,
			Mask:         it.Mask,
			Quantity:     it.Quantity,
			Weight:       it.Weight,
			FobUnitPrice: it.FobUnitPrice,
			Notes:        it.Notes,
		})
	}
	for _, e := range dto.Expenses {
		if e.Amount < 0 {
			return nil, fmt.Errorf("expense amount must be >= 0")
		}
		inCost := true
		if e.InItemCost != nil {
			inCost = *e.InItemCost
		}
		process.Expenses = append(process.Expenses, &entity.ImportExpense{
			ExpenseType: e.ExpenseType,
			Amount:      e.Amount,
			InItemCost:  inCost,
			Notes:       e.Notes,
		})
	}
	// Compute landed costs before persisting so the stored values are authoritative.
	entity.ComputeLandedCosts(process)
	created, err := uc.Repo.CreateImportProcess(ctx, process)
	if err != nil {
		return nil, err
	}
	return toImportProcessResponse(created), nil
}

func (uc *UseCase) GetImportProcess(ctx context.Context, id int64) (*response.ImportProcessResponse, error) {
	p, err := uc.Repo.GetImportProcess(ctx, id)
	if err != nil {
		return nil, err
	}
	return toImportProcessResponse(p), nil
}

func (uc *UseCase) ListImportProcesses(ctx context.Context, status string) ([]*response.ImportProcessResponse, error) {
	rows, err := uc.Repo.ListImportProcesses(ctx, status)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ImportProcessResponse, 0, len(rows))
	for _, p := range rows {
		out = append(out, toImportProcessResponse(p))
	}
	return out, nil
}

// RecomputeImportProcess reloads a process, recomputes the landed costs from its
// current items/expenses and persists them.
func (uc *UseCase) RecomputeImportProcess(ctx context.Context, id int64) (*response.ImportProcessResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	p, err := uc.Repo.GetImportProcess(ctx, id)
	if err != nil {
		return nil, err
	}
	entity.ComputeLandedCosts(p)
	if err := uc.Repo.UpdateImportItemCosts(ctx, p.Items); err != nil {
		return nil, err
	}
	return toImportProcessResponse(p), nil
}

func (uc *UseCase) UpdateImportProcessStatus(ctx context.Context, id int64, dto request.UpdateImportProcessStatusDTO) (*response.ImportProcessResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if !validImportStatus(dto.Status) {
		return nil, fmt.Errorf("invalid status %q", dto.Status)
	}
	p, err := uc.Repo.UpdateImportProcessStatus(ctx, id, dto.Status)
	if err != nil {
		return nil, err
	}
	return toImportProcessResponse(p), nil
}

// ---- Procurement parameters (FUTL0125) ----

func (uc *UseCase) UpsertParameter(ctx context.Context, dto request.UpsertProcurementParameterDTO) (*response.ProcurementParameterResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.EnterpriseCode == 0 {
		dto.EnterpriseCode = 1
	}
	if !validParameterDomain(dto.Domain) {
		return nil, fmt.Errorf("invalid domain %q", dto.Domain)
	}
	if dto.Key == "" {
		return nil, fmt.Errorf("param_key is required")
	}
	if dto.ValueType == "" {
		dto.ValueType = "STRING"
	}
	if !validParameterValueType(dto.ValueType) {
		return nil, fmt.Errorf("value_type must be STRING, NUMBER, BOOL or JSON")
	}
	actor, _ := uc.Auth.UserID(ctx)
	param := &entity.ProcurementParameter{
		EnterpriseCode: dto.EnterpriseCode,
		Domain:         dto.Domain,
		Key:            dto.Key,
		Value:          dto.Value,
		ValueType:      dto.ValueType,
		Description:    dto.Description,
		UpdatedBy:      &actor,
	}
	saved, err := uc.Repo.UpsertParameter(ctx, param)
	if err != nil {
		return nil, err
	}
	return toParameterResponse(saved), nil
}

func (uc *UseCase) ListParameters(ctx context.Context, enterpriseCode int64, domain string) ([]*response.ProcurementParameterResponse, error) {
	if enterpriseCode == 0 {
		enterpriseCode = 1
	}
	params, err := uc.Repo.ListParameters(ctx, enterpriseCode, domain)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ProcurementParameterResponse, 0, len(params))
	for _, p := range params {
		out = append(out, toParameterResponse(p))
	}
	return out, nil
}

// ---- Supplier homologation (FAVF0203) ----

func (uc *UseCase) CreateSupplierHomologation(ctx context.Context, dto request.CreateSupplierHomologationDTO) (*response.SupplierHomologationResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.SupplierCode <= 0 {
		return nil, fmt.Errorf("supplier_code is required")
	}
	validUntil, err := parseDatePtr(dto.ValidUntil)
	if err != nil {
		return nil, fmt.Errorf("invalid valid_until: %w", err)
	}
	actor, _ := uc.Auth.UserID(ctx)
	h := &entity.SupplierHomologation{
		SupplierCode: dto.SupplierCode,
		Category:     dto.Category,
		ValidUntil:   validUntil,
		Notes:        dto.Notes,
		DecidedBy:    &actor,
	}
	// If a status is not forced, derive it from the IQF over the given period.
	if dto.Status != "" {
		if !validHomologationStatus(dto.Status) {
			return nil, fmt.Errorf("invalid status %q", dto.Status)
		}
		h.Status = dto.Status
	} else {
		start, err := time.Parse("2006-01-02", dto.PeriodStart)
		if err != nil {
			return nil, fmt.Errorf("invalid period_start: %w", err)
		}
		end, err := time.Parse("2006-01-02", dto.PeriodEnd)
		if err != nil {
			return nil, fmt.Errorf("invalid period_end: %w", err)
		}
		agg, err := uc.Repo.AggregateSupplierPerformance(ctx, dto.SupplierCode, start, end)
		if err != nil {
			return nil, err
		}
		quality := ratioScore(agg.InspectedQty-agg.RejectedQty, agg.InspectedQty)
		delivery := ratioScore(float64(agg.TotalReceipts-agg.LateReceipts), float64(agg.TotalReceipts))
		iqf := overallIQF(quality, delivery, 100, 100)
		homologatedMin := dto.HomologatedMin
		if homologatedMin <= 0 {
			homologatedMin = 80
		}
		conditionalMin := dto.ConditionalMin
		if conditionalMin <= 0 {
			conditionalMin = 60
		}
		h.IQFScore = &iqf
		h.Status = entity.HomologationStatusForIQF(iqf, homologatedMin, conditionalMin)
	}
	created, err := uc.Repo.CreateHomologation(ctx, h)
	if err != nil {
		return nil, err
	}
	return toHomologationResponse(created), nil
}

// GenerateItemSuppliers (FFOR0204) creates item-supplier links for every item
// bought from the supplier that is not linked yet, returning how many were created.
func (uc *UseCase) GenerateItemSuppliers(ctx context.Context, supplierCode int64) (int, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return 0, errorsuc.ErrUnauthorized
	}
	if supplierCode <= 0 {
		return 0, fmt.Errorf("supplier_code is required")
	}
	actor, _ := uc.Auth.UserID(ctx)
	return uc.Repo.GenerateItemSuppliersFromHistory(ctx, supplierCode, actor)
}

func (uc *UseCase) ListSupplierHomologations(ctx context.Context, supplierCode int64) ([]*response.SupplierHomologationResponse, error) {
	rows, err := uc.Repo.ListHomologations(ctx, supplierCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.SupplierHomologationResponse, 0, len(rows))
	for _, h := range rows {
		out = append(out, toHomologationResponse(h))
	}
	return out, nil
}

// ---- validation helpers ----

func validNoticeStatus(s string) bool {
	switch s {
	case "SCHEDULED", "ARRIVED", "IN_CONFERENCE", "RELEASED", "BLOCKED", "CANCELLED":
		return true
	}
	return false
}

func validDivergenceType(s string) bool {
	switch s {
	case "SHORTAGE", "EXCESS", "DAMAGE", "WRONG_ITEM", "PRICE", "DOCUMENT", "LATE", "OTHER":
		return true
	}
	return false
}

func validDivergenceResolution(s string) bool {
	switch s {
	case "PENDING", "ACCEPTED", "PARTIAL_RETURN", "FULL_RETURN", "WAIVED", "SUPPLIER_DEBIT":
		return true
	}
	return false
}

func validEDIMessageType(s string) bool {
	switch s {
	case "ORDER_CONFIRMATION", "SHIP_NOTICE", "INVOICE", "ORDER", "OTHER":
		return true
	}
	return false
}

func validApportionBasis(s string) bool {
	switch s {
	case "VALUE", "WEIGHT", "QUANTITY":
		return true
	}
	return false
}

func validImportStatus(s string) bool {
	switch s {
	case "OPEN", "NATIONALIZED", "CANCELLED":
		return true
	}
	return false
}

func validParameterDomain(s string) bool {
	switch s {
	case "PURCHASE_TABLE", "PURCHASE_ORDER", "QUOTATION", "REQUISITION", "RECEIVING_NOTICE",
		"INSPECTION", "SUPPLIER_EVALUATION", "CONTRACT", "SUPPLIER", "NF_ENTRY":
		return true
	}
	return false
}

func validParameterValueType(s string) bool {
	switch s {
	case "STRING", "NUMBER", "BOOL", "JSON":
		return true
	}
	return false
}

func validHomologationStatus(s string) bool {
	switch s {
	case "HOMOLOGATED", "CONDITIONAL", "PENDING", "SUSPENDED", "REJECTED":
		return true
	}
	return false
}

func parseTimePtr(value *string) (*time.Time, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339, *value); err == nil {
		return &t, nil
	}
	t, err := time.Parse("2006-01-02", *value)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ---- mappers ----

func toReceivingNoticeResponse(n *entity.ReceivingNotice) *response.ReceivingNoticeResponse {
	out := &response.ReceivingNoticeResponse{
		ID:                n.ID,
		EnterpriseCode:    n.EnterpriseCode,
		NoticeNumber:      n.NoticeNumber,
		SupplierCode:      n.SupplierCode,
		PurchaseOrderCode: n.PurchaseOrderCode,
		CarrierCode:       n.CarrierCode,
		Status:            n.Status,
		Dock:              n.Dock,
		ScheduledAt:       n.ScheduledAt,
		ArrivedAt:         n.ArrivedAt,
		InvoiceNumber:     n.InvoiceNumber,
		Blocked:           n.Blocked,
		Notes:             n.Notes,
		CreatedBy:         n.CreatedBy,
		CreatedAt:         n.CreatedAt,
	}
	for _, it := range n.Items {
		out.Items = append(out.Items, response.ReceivingNoticeItemResponse{
			ID:                    it.ID,
			PurchaseOrderItemCode: it.PurchaseOrderItemCode,
			ItemCode:              it.ItemCode,
			Mask:                  it.Mask,
			ExpectedQty:           it.ExpectedQty,
			ReceivedQty:           it.ReceivedQty,
			Unit:                  it.Unit,
			Notes:                 it.Notes,
		})
	}
	return out
}

func toDivergenceResponse(d *entity.ReceivingDivergence) *response.ReceivingDivergenceResponse {
	return &response.ReceivingDivergenceResponse{
		ID:                    d.ID,
		NoticeID:              d.NoticeID,
		PurchaseOrderCode:     d.PurchaseOrderCode,
		PurchaseOrderItemCode: d.PurchaseOrderItemCode,
		SupplierCode:          d.SupplierCode,
		ItemCode:              d.ItemCode,
		Mask:                  d.Mask,
		DivergenceType:        d.DivergenceType,
		ExpectedQty:           d.ExpectedQty,
		ActualQty:             d.ActualQty,
		ExpectedPrice:         d.ExpectedPrice,
		ActualPrice:           d.ActualPrice,
		Resolution:            d.Resolution,
		AffectsSupplierScore:  d.AffectsSupplierScore,
		Notes:                 d.Notes,
		CreatedAt:             d.CreatedAt,
		ResolvedAt:            d.ResolvedAt,
	}
}

func toEDIMessageResponse(m *entity.SupplierEDIMessage) *response.SupplierEDIMessageResponse {
	out := &response.SupplierEDIMessageResponse{
		ID:                m.ID,
		EnterpriseCode:    m.EnterpriseCode,
		SupplierCode:      m.SupplierCode,
		Direction:         m.Direction,
		MessageType:       m.MessageType,
		PurchaseOrderCode: m.PurchaseOrderCode,
		ExternalReference: m.ExternalReference,
		Status:            m.Status,
		DivergenceCount:   m.DivergenceCount,
		Payload:           m.Payload,
		Notes:             m.Notes,
		CreatedAt:         m.CreatedAt,
		ProcessedAt:       m.ProcessedAt,
	}
	for _, l := range m.Lines {
		out.Lines = append(out.Lines, response.SupplierEDILineResponse{
			ID:                    l.ID,
			PurchaseOrderItemCode: l.PurchaseOrderItemCode,
			ItemCode:              l.ItemCode,
			Mask:                  l.Mask,
			ConfirmedQty:          l.ConfirmedQty,
			ConfirmedPrice:        l.ConfirmedPrice,
			ConfirmedDate:         l.ConfirmedDate,
			Divergence:            l.Divergence,
			Notes:                 l.Notes,
		})
	}
	return out
}

func toImportProcessResponse(p *entity.ImportProcess) *response.ImportProcessResponse {
	out := &response.ImportProcessResponse{
		ID:                p.ID,
		EnterpriseCode:    p.EnterpriseCode,
		ProcessNumber:     p.ProcessNumber,
		SupplierCode:      p.SupplierCode,
		PurchaseOrderCode: p.PurchaseOrderCode,
		Reference:         p.Reference,
		Incoterm:          p.Incoterm,
		Currency:          p.Currency,
		ExchangeRate:      p.ExchangeRate,
		ApportionBasis:    p.ApportionBasis,
		Status:            p.Status,
		Notes:             p.Notes,
		CreatedAt:         p.CreatedAt,
		NationalizedAt:    p.NationalizedAt,
	}
	for _, it := range p.Items {
		out.Items = append(out.Items, response.ImportProcessItemResponse{
			ID:                  it.ID,
			ItemCode:            it.ItemCode,
			Mask:                it.Mask,
			Quantity:            it.Quantity,
			Weight:              it.Weight,
			FobUnitPrice:        it.FobUnitPrice,
			ApportionedExpenses: it.ApportionedExpenses,
			LandedUnitCost:      it.LandedUnitCost,
			Notes:               it.Notes,
		})
		out.TotalLandedValue += it.LandedUnitCost * it.Quantity
	}
	for _, e := range p.Expenses {
		out.Expenses = append(out.Expenses, response.ImportExpenseResponse{
			ID:          e.ID,
			ExpenseType: e.ExpenseType,
			Amount:      e.Amount,
			InItemCost:  e.InItemCost,
			Notes:       e.Notes,
		})
		out.TotalExpenses += e.Amount
	}
	return out
}

func toParameterResponse(p *entity.ProcurementParameter) *response.ProcurementParameterResponse {
	return &response.ProcurementParameterResponse{
		ID:          p.ID,
		Domain:      p.Domain,
		Key:         p.Key,
		Value:       p.Value,
		ValueType:   p.ValueType,
		Description: p.Description,
		UpdatedBy:   p.UpdatedBy,
		UpdatedAt:   p.UpdatedAt,
	}
}

func toHomologationResponse(h *entity.SupplierHomologation) *response.SupplierHomologationResponse {
	return &response.SupplierHomologationResponse{
		ID:           h.ID,
		SupplierCode: h.SupplierCode,
		Status:       h.Status,
		IQFScore:     h.IQFScore,
		Category:     h.Category,
		ValidUntil:   h.ValidUntil,
		Notes:        h.Notes,
		DecidedBy:    h.DecidedBy,
		DecidedAt:    h.DecidedAt,
	}
}
