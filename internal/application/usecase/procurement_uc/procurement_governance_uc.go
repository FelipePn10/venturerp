package procurement_uc

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/procurement/entity"
	"github.com/jackc/pgx/v5"
)

// ---- IQF auto-computation (avaliação de fornecedor a partir de dados reais) ----

// ComputeSupplierScorecard derives quality and delivery scores from real receiving
// inspection and purchase delivery data, instead of manual entry. Commercial and
// service scores remain manual inputs (default 100) since they have no objective
// source yet. When Persist is true the computed scorecard is also stored.
func (uc *UseCase) ComputeSupplierScorecard(ctx context.Context, dto request.ComputeSupplierScorecardDTO) (*response.SupplierScorecardResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.SupplierCode <= 0 {
		return nil, fmt.Errorf("supplier_code is required")
	}
	start, err := time.Parse("2006-01-02", dto.PeriodStart)
	if err != nil {
		return nil, fmt.Errorf("invalid period_start: %w", err)
	}
	end, err := time.Parse("2006-01-02", dto.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("invalid period_end: %w", err)
	}
	if end.Before(start) {
		return nil, fmt.Errorf("period_end must be on or after period_start")
	}
	agg, err := uc.Repo.AggregateSupplierPerformance(ctx, dto.SupplierCode, start, end)
	if err != nil {
		return nil, err
	}

	quality := ratioScore(agg.InspectedQty-agg.RejectedQty, agg.InspectedQty)
	delivery := ratioScore(float64(agg.TotalReceipts-agg.LateReceipts), float64(agg.TotalReceipts))
	commercial := defaultScore(dto.CommercialScore)
	service := defaultScore(dto.ServiceScore)

	score := &entity.SupplierScorecard{
		SupplierCode:     dto.SupplierCode,
		PeriodStart:      start,
		PeriodEnd:        end,
		QualityScore:     quality,
		DeliveryScore:    delivery,
		CommercialScore:  commercial,
		ServiceScore:     service,
		OverallScore:     overallIQF(quality, delivery, commercial, service),
		TotalReceipts:    agg.TotalReceipts,
		RejectedReceipts: agg.RejectedReceipts,
		LateReceipts:     agg.LateReceipts,
		Notes:            dto.Notes,
	}

	out := toScorecardResponse(score)
	out.Computed = true
	if dto.Persist {
		actor, _ := uc.Auth.UserID(ctx)
		score.CreatedBy = &actor
		created, err := uc.Repo.CreateSupplierScorecard(ctx, score)
		if err != nil {
			return nil, err
		}
		out = toScorecardResponse(created)
		out.Computed = true
		out.Persisted = true
	}
	return out, nil
}

// ratioScore returns num/den as a 0..100 score, defaulting to 100 when there is no
// denominator (no data to penalize the supplier).
func ratioScore(num, den float64) float64 {
	if den <= 0 {
		return 100
	}
	score := num / den * 100
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func overallIQF(quality, delivery, commercial, service float64) float64 {
	return quality*0.40 + delivery*0.30 + commercial*0.20 + service*0.10
}

// ---- Approval limits (alçada de valores) ----

func (uc *UseCase) CreateApprovalLimit(ctx context.Context, dto request.CreateApprovalLimitDTO) (*response.ApprovalLimitResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.EnterpriseCode == 0 {
		dto.EnterpriseCode = 1
	}
	switch dto.Scope {
	case "GLOBAL", "SUPPLIER", "COST_CENTER", "CATEGORY":
	default:
		return nil, fmt.Errorf("scope must be GLOBAL, SUPPLIER, COST_CENTER or CATEGORY")
	}
	if dto.Scope != "GLOBAL" && (dto.ScopeRef == nil || *dto.ScopeRef == "") {
		return nil, fmt.Errorf("scope_ref is required when scope is not GLOBAL")
	}
	if dto.AutoApproveMax < 0 {
		return nil, fmt.Errorf("auto_approve_max must be >= 0")
	}
	if dto.BlockAbove != nil && *dto.BlockAbove < dto.AutoApproveMax {
		return nil, fmt.Errorf("block_above must be >= auto_approve_max")
	}
	if dto.Currency == "" {
		dto.Currency = "BRL"
	}
	validFrom, err := parseOptionalDate(dto.ValidFrom, time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid valid_from: %w", err)
	}
	validTo, err := parseDatePtr(dto.ValidTo)
	if err != nil {
		return nil, fmt.Errorf("invalid valid_to: %w", err)
	}
	actor, _ := uc.Auth.UserID(ctx)
	limit := &entity.ApprovalLimit{
		EnterpriseCode: dto.EnterpriseCode,
		Scope:          dto.Scope,
		ScopeRef:       dto.ScopeRef,
		Currency:       dto.Currency,
		AutoApproveMax: dto.AutoApproveMax,
		BlockAbove:     dto.BlockAbove,
		ValidFrom:      validFrom,
		ValidTo:        validTo,
		Notes:          dto.Notes,
		CreatedBy:      &actor,
	}
	created, err := uc.Repo.CreateApprovalLimit(ctx, limit)
	if err != nil {
		return nil, err
	}
	return toApprovalLimitResponse(created), nil
}

func (uc *UseCase) ListApprovalLimits(ctx context.Context, enterpriseCode int64) ([]*response.ApprovalLimitResponse, error) {
	if enterpriseCode == 0 {
		enterpriseCode = 1
	}
	limits, err := uc.Repo.ListApprovalLimits(ctx, enterpriseCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ApprovalLimitResponse, 0, len(limits))
	for _, l := range limits {
		out = append(out, toApprovalLimitResponse(l))
	}
	return out, nil
}

// EvaluatePurchaseApproval resolves the applicable approval rule for a purchase
// amount. When no rule is configured the purchase is auto-approvable (no alçada
// control). Implements the purchase_order_uc.ApprovalPolicy port.
func (uc *UseCase) EvaluatePurchaseApproval(ctx context.Context, enterpriseCode int64, supplierCode *int64, amount float64) (*entity.ApprovalDecision, error) {
	if enterpriseCode == 0 {
		enterpriseCode = 1
	}
	var supplierRef *string
	if supplierCode != nil {
		s := strconv.FormatInt(*supplierCode, 10)
		supplierRef = &s
	}
	limit, err := uc.Repo.FindApprovalLimit(ctx, enterpriseCode, supplierRef, nil, nil)
	if err == pgx.ErrNoRows {
		// No rule configured: no alçada control, auto-approve.
		return &entity.ApprovalDecision{AutoApprove: true}, nil
	}
	if err != nil {
		return nil, err
	}
	decision := &entity.ApprovalDecision{
		Ceiling:     limit.AutoApproveMax,
		HardCeiling: limit.BlockAbove,
		LimitID:     &limit.ID,
		AutoApprove: amount <= limit.AutoApproveMax+0.0001,
	}
	if limit.BlockAbove != nil && amount > *limit.BlockAbove+0.0001 {
		decision.Blocked = true
	}
	return decision, nil
}

// ---- Supplier contracts ----

func (uc *UseCase) CreateSupplierContract(ctx context.Context, dto request.CreateSupplierContractDTO) (*response.SupplierContractResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.SupplierCode <= 0 {
		return nil, fmt.Errorf("supplier_code is required")
	}
	if dto.ContractNumber == "" {
		return nil, fmt.Errorf("contract_number is required")
	}
	if dto.EnterpriseCode == 0 {
		dto.EnterpriseCode = 1
	}
	if dto.Currency == "" {
		dto.Currency = "BRL"
	}
	status := dto.Status
	if status == "" {
		status = "DRAFT"
	}
	if !validContractStatus(status) {
		return nil, fmt.Errorf("invalid status %q", status)
	}
	validFrom, err := parseOptionalDate(dto.ValidFrom, time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid valid_from: %w", err)
	}
	validTo, err := parseDatePtr(dto.ValidTo)
	if err != nil {
		return nil, fmt.Errorf("invalid valid_to: %w", err)
	}
	actor, _ := uc.Auth.UserID(ctx)
	contract := &entity.SupplierContract{
		EnterpriseCode: dto.EnterpriseCode,
		SupplierCode:   dto.SupplierCode,
		ContractNumber: dto.ContractNumber,
		Description:    dto.Description,
		Status:         status,
		Currency:       dto.Currency,
		ValidFrom:      validFrom,
		ValidTo:        validTo,
		PriceIndex:     dto.PriceIndex,
		Notes:          dto.Notes,
		CreatedBy:      &actor,
	}
	for _, it := range dto.Items {
		if it.ItemCode <= 0 {
			return nil, fmt.Errorf("contract item requires item_code")
		}
		if it.ContractedQty < 0 || it.UnitPrice < 0 {
			return nil, fmt.Errorf("contract item quantities/prices must be >= 0")
		}
		contract.Items = append(contract.Items, &entity.SupplierContractItem{
			ItemCode:      it.ItemCode,
			Mask:          it.Mask,
			Unit:          it.Unit,
			ContractedQty: it.ContractedQty,
			UnitPrice:     it.UnitPrice,
			MinOrderQty:   it.MinOrderQty,
			Notes:         it.Notes,
		})
	}
	created, err := uc.Repo.CreateSupplierContract(ctx, contract)
	if err != nil {
		return nil, err
	}
	return toSupplierContractResponse(created), nil
}

func (uc *UseCase) GetSupplierContract(ctx context.Context, id int64) (*response.SupplierContractResponse, error) {
	contract, err := uc.Repo.GetSupplierContract(ctx, id)
	if err != nil {
		return nil, err
	}
	return toSupplierContractResponse(contract), nil
}

func (uc *UseCase) ListSupplierContracts(ctx context.Context, supplierCode int64, status string) ([]*response.SupplierContractResponse, error) {
	contracts, err := uc.Repo.ListSupplierContracts(ctx, supplierCode, status)
	if err != nil {
		return nil, err
	}
	out := make([]*response.SupplierContractResponse, 0, len(contracts))
	for _, c := range contracts {
		out = append(out, toSupplierContractResponse(c))
	}
	return out, nil
}

func (uc *UseCase) UpdateSupplierContractStatus(ctx context.Context, id int64, dto request.UpdateSupplierContractStatusDTO) (*response.SupplierContractResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if !validContractStatus(dto.Status) {
		return nil, fmt.Errorf("invalid status %q", dto.Status)
	}
	contract, err := uc.Repo.UpdateSupplierContractStatus(ctx, id, dto.Status)
	if err != nil {
		return nil, err
	}
	return toSupplierContractResponse(contract), nil
}

func (uc *UseCase) ConsumeSupplierContract(ctx context.Context, id int64, dto request.ConsumeSupplierContractDTO) (*response.SupplierContractItemResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be positive")
	}
	contract, err := uc.Repo.GetSupplierContract(ctx, id)
	if err != nil {
		return nil, err
	}
	if contract.Status != "ACTIVE" {
		return nil, fmt.Errorf("contract %d is not ACTIVE (status %s)", id, contract.Status)
	}
	item, err := uc.Repo.FindContractItem(ctx, id, dto.ItemCode, dto.Mask)
	if err != nil {
		return nil, fmt.Errorf("item %d/%s is not in contract %d", dto.ItemCode, dto.Mask, id)
	}
	updated, err := uc.Repo.ConsumeContractItem(ctx, item.ID, dto.Quantity)
	if err != nil {
		return nil, err
	}
	return toSupplierContractItemResponse(updated), nil
}

// ---- Consolidated purchase movement history ----

func (uc *UseCase) ListPurchaseMovementHistory(ctx context.Context, supplierCode, itemCode *int64, limit int) ([]*response.PurchaseMovementHistoryResponse, error) {
	rows, err := uc.Repo.ListPurchaseMovementHistory(ctx, supplierCode, itemCode, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*response.PurchaseMovementHistoryResponse, 0, len(rows))
	for _, h := range rows {
		open := h.RequestedQty - h.ReceivedQty - h.CancelledQty
		if open < 0 {
			open = 0
		}
		out = append(out, &response.PurchaseMovementHistoryResponse{
			SupplierCode:      h.SupplierCode,
			PurchaseOrderCode: h.PurchaseOrderCode,
			OrderNumber:       h.OrderNumber,
			ItemCode:          h.ItemCode,
			Mask:              h.Mask,
			RequestedQty:      h.RequestedQty,
			ReceivedQty:       h.ReceivedQty,
			CancelledQty:      h.CancelledQty,
			OpenQty:           open,
			UnitPrice:         h.UnitPrice,
			Status:            h.Status,
			EmissionDate:      h.EmissionDate,
			DeliveryDate:      h.DeliveryDate,
		})
	}
	return out, nil
}

// ---- FINS0212: inspection gate driven from purchase receipt ----

// ResolveInspectionRoute returns the inspection warehouse of an active route matching
// the item, and whether a route matched. Implements the
// purchase_order_uc.ReceivingInspectionGate port.
func (uc *UseCase) ResolveInspectionRoute(ctx context.Context, itemCode int64, mask string) (int64, bool) {
	route, err := uc.Repo.FindReceivingInspectionRoute(ctx, 1, itemCode, mask, nil)
	if err != nil {
		return 0, false
	}
	return route.InspectionWarehouseID, true
}

// OpenInspectionOrderFromReceipt opens a PENDING_INSPECTION order for material that a
// purchase receipt routed into the inspection warehouse.
func (uc *UseCase) OpenInspectionOrderFromReceipt(ctx context.Context, itemCode int64, mask string, quantity float64, inspectionWarehouseID int64, supplierCode, purchaseOrderCode, purchaseOrderItemCode *int64, lot *string) (int64, int64, error) {
	route, err := uc.Repo.FindReceivingInspectionRoute(ctx, 1, itemCode, mask, nil)
	var routeID *int64
	if err == nil {
		routeID = &route.ID
	}
	order := &entity.ReceivingInspectionOrder{
		RouteID:               routeID,
		Source:                "PURCHASE_RECEIPT",
		SupplierCode:          supplierCode,
		PurchaseOrderCode:     purchaseOrderCode,
		PurchaseOrderItemCode: purchaseOrderItemCode,
		ItemCode:              itemCode,
		Mask:                  mask,
		Lot:                   lot,
		WarehouseID:           inspectionWarehouseID,
		Quantity:              quantity,
	}
	created, err := uc.Repo.CreateReceivingInspectionOrder(ctx, order)
	if err != nil {
		return 0, 0, err
	}
	return created.ID, created.OrderNumber, nil
}

func validContractStatus(s string) bool {
	switch s {
	case "DRAFT", "ACTIVE", "SUSPENDED", "CLOSED", "CANCELLED":
		return true
	default:
		return false
	}
}

func toApprovalLimitResponse(l *entity.ApprovalLimit) *response.ApprovalLimitResponse {
	return &response.ApprovalLimitResponse{
		ID:             l.ID,
		EnterpriseCode: l.EnterpriseCode,
		Scope:          l.Scope,
		ScopeRef:       l.ScopeRef,
		Currency:       l.Currency,
		AutoApproveMax: l.AutoApproveMax,
		BlockAbove:     l.BlockAbove,
		IsActive:       l.IsActive,
		ValidFrom:      l.ValidFrom,
		ValidTo:        l.ValidTo,
		Notes:          l.Notes,
		CreatedBy:      l.CreatedBy,
		CreatedAt:      l.CreatedAt,
	}
}

func toSupplierContractResponse(c *entity.SupplierContract) *response.SupplierContractResponse {
	out := &response.SupplierContractResponse{
		ID:             c.ID,
		EnterpriseCode: c.EnterpriseCode,
		SupplierCode:   c.SupplierCode,
		ContractNumber: c.ContractNumber,
		Description:    c.Description,
		Status:         c.Status,
		Currency:       c.Currency,
		ValidFrom:      c.ValidFrom,
		ValidTo:        c.ValidTo,
		PriceIndex:     c.PriceIndex,
		Notes:          c.Notes,
		CreatedBy:      c.CreatedBy,
		CreatedAt:      c.CreatedAt,
	}
	for _, it := range c.Items {
		out.Items = append(out.Items, *toSupplierContractItemResponse(it))
	}
	return out
}

func toSupplierContractItemResponse(it *entity.SupplierContractItem) *response.SupplierContractItemResponse {
	return &response.SupplierContractItemResponse{
		ID:            it.ID,
		ContractID:    it.ContractID,
		ItemCode:      it.ItemCode,
		Mask:          it.Mask,
		Unit:          it.Unit,
		ContractedQty: it.ContractedQty,
		ConsumedQty:   it.ConsumedQty,
		RemainingQty:  it.RemainingQty(),
		UnitPrice:     it.UnitPrice,
		MinOrderQty:   it.MinOrderQty,
		Notes:         it.Notes,
	}
}
