package third_party_service_uc

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	domain "github.com/FelipePn10/panossoerp/internal/domain/third_party_service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type UseCase struct{ repo domain.Repository }

func New(repo domain.Repository) *UseCase { return &UseCase{repo: repo} }
func priceFrom(dto request.ThirdPartyPriceDTO, by uuid.UUID) (*domain.Price, error) {
	unit, e := decimal.NewFromString(strings.TrimSpace(dto.UnitPrice))
	if e != nil {
		return nil, errors.New("unit_price must be decimal")
	}
	freight, e := decimal.NewFromString(defaultDecimal(dto.FreightValue))
	if e != nil {
		return nil, errors.New("freight_value must be decimal")
	}
	tax, e := decimal.NewFromString(defaultDecimal(dto.TaxPercent))
	if e != nil {
		return nil, errors.New("tax_percent must be decimal")
	}
	p := &domain.Price{ItemCode: dto.ItemCode, Mask: dto.Mask, SupplierCode: dto.SupplierCode, OperationID: dto.OperationID, UOM: dto.UOM, ReferenceDate: dto.ReferenceDate, Preferred: dto.Preferred, UnitPrice: unit, FreightType: dto.FreightType, FreightValue: freight, TaxPercent: tax, Formula: dto.Formula, CreatedBy: by}
	if p.FreightType == "" {
		p.FreightType = "FIXED"
	}
	if dto.ConversionFactor != nil {
		v, e := decimal.NewFromString(*dto.ConversionFactor)
		if e != nil {
			return nil, errors.New("conversion_factor must be decimal")
		}
		p.ConversionFactor = &v
	}
	for _, v := range dto.Rules {
		p.Rules = append(p.Rules, domain.PriceRule{Characteristic: v.Characteristic, Answer: v.Answer})
	}
	return p, p.Validate()
}
func defaultDecimal(v string) string {
	if strings.TrimSpace(v) == "" {
		return "0"
	}
	return v
}
func (u *UseCase) CreatePrice(ctx context.Context, d request.ThirdPartyPriceDTO, by uuid.UUID) (response.ThirdPartyPriceResponse, error) {
	p, e := priceFrom(d, by)
	if e != nil {
		return response.ThirdPartyPriceResponse{}, e
	}
	v, e := u.repo.CreatePrice(ctx, p, d.Reason)
	return priceResponse(v), e
}
func (u *UseCase) UpdatePrice(ctx context.Context, id int64, d request.ThirdPartyPriceDTO, by uuid.UUID) (response.ThirdPartyPriceResponse, error) {
	p, e := priceFrom(d, by)
	if e != nil {
		return response.ThirdPartyPriceResponse{}, e
	}
	p.ID = id
	v, e := u.repo.UpdatePrice(ctx, p, d.Reason)
	return priceResponse(v), e
}
func (u *UseCase) DeletePrice(ctx context.Context, id int64, reason string, by uuid.UUID) error {
	return u.repo.DeletePrice(ctx, id, reason, by)
}
func (u *UseCase) GetPrice(ctx context.Context, id int64) (response.ThirdPartyPriceResponse, error) {
	v, e := u.repo.GetPrice(ctx, id)
	return priceResponse(v), e
}
func (u *UseCase) ListPrices(ctx context.Context, f domain.PriceFilter) ([]response.ThirdPartyPriceResponse, error) {
	rows, e := u.repo.ListPrices(ctx, f)
	out := make([]response.ThirdPartyPriceResponse, 0, len(rows))
	for i := range rows {
		out = append(out, priceResponse(&rows[i]))
	}
	return out, e
}
func (u *UseCase) ResolvePrice(ctx context.Context, item int64, mask string, supplier, op int64, at time.Time, attrs map[string]string) (response.ThirdPartyPriceResponse, error) {
	v, e := u.repo.ResolvePrice(ctx, item, mask, supplier, op, at, attrs)
	return priceResponse(v), e
}
func (u *UseCase) StandardCostPerUnit(ctx context.Context, item int64, mask string, operationID int64, at time.Time) (decimal.Decimal, error) {
	v, e := u.CostPerUnit(ctx, item, mask, operationID, at, "STANDARD")
	return v.EffectiveUnitCost, e
}

type CostBreakdown struct{ GrossUnitCost, Freight, RecoverableTaxes, ConversionFactor, EffectiveUnitCost decimal.Decimal }

func (u *UseCase) CostPerUnit(ctx context.Context, item int64, mask string, operationID int64, at time.Time, mode string) (CostBreakdown, error) {
	p, e := u.repo.ResolvePrice(ctx, item, mask, 0, operationID, at, nil)
	if e != nil {
		return CostBreakdown{}, e
	}
	mode = strings.ToUpper(strings.TrimSpace(mode))
	if mode == "" {
		mode = "STANDARD"
	}
	if mode != "STANDARD" && mode != "REAL" {
		return CostBreakdown{}, errors.New("mode must be STANDARD or REAL")
	}
	result := CostBreakdown{GrossUnitCost: p.UnitPrice, ConversionFactor: decimal.NewFromInt(1)}
	if mode == "STANDARD" {
		if p.FreightType == "PERCENT" {
			result.Freight = p.UnitPrice.Mul(p.FreightValue).Div(decimal.NewFromInt(100))
		} else {
			result.Freight = p.FreightValue
		}
	}
	if mode == "REAL" {
		result.RecoverableTaxes = p.UnitPrice.Mul(p.TaxPercent).Div(decimal.NewFromInt(100))
	}
	cost := p.UnitPrice.Add(result.Freight).Sub(result.RecoverableTaxes)
	factor := p.ConversionFactor
	if factor == nil {
		factor, _ = u.repo.ResolveConversionFactor(ctx, item, mask, p.UOM)
	}
	if factor != nil {
		result.ConversionFactor = *factor
		cost = cost.Div(*factor)
	}
	result.EffectiveUnitCost = cost
	return result, nil
}
func (u *UseCase) History(ctx context.Context, id int64) ([]response.ThirdPartyHistoryResponse, error) {
	rows, e := u.repo.History(ctx, id)
	out := make([]response.ThirdPartyHistoryResponse, 0, len(rows))
	for _, v := range rows {
		out = append(out, response.ThirdPartyHistoryResponse{ID: v.ID, PriceID: v.PriceID, Action: v.Action, Reason: v.Reason, Snapshot: v.Snapshot, ChangedBy: v.ChangedBy.String(), ChangedAt: v.ChangedAt})
	}
	return out, e
}
func (u *UseCase) Readjust(ctx context.Context, d request.ThirdPartyReadjustDTO, by uuid.UUID) ([]response.ThirdPartyPriceResponse, error) {
	pct, e := decimal.NewFromString(d.Percent)
	if e != nil {
		return nil, errors.New("percent must be decimal")
	}
	rows, e := u.repo.Readjust(ctx, d.IDs, pct, d.ReferenceDate, d.Reason, by)
	out := make([]response.ThirdPartyPriceResponse, 0, len(rows))
	for i := range rows {
		out = append(out, priceResponse(&rows[i]))
	}
	return out, e
}
func (u *UseCase) CopyMove(ctx context.Context, d request.ThirdPartyCopyMoveDTO, by uuid.UUID) ([]response.ThirdPartyPriceResponse, error) {
	rows, e := u.repo.CopyMove(ctx, d.IDs, d.SupplierCode, d.OperationID, d.Move, d.ReferenceDate, d.Reason, by)
	out := make([]response.ThirdPartyPriceResponse, 0, len(rows))
	for i := range rows {
		out = append(out, priceResponse(&rows[i]))
	}
	return out, e
}
func (u *UseCase) CreateOrders(ctx context.Context, productionID int64, by uuid.UUID) ([]response.ThirdPartyOrderResponse, error) {
	rows, e := u.repo.CreateOrdersForProduction(ctx, productionID, by)
	return orderResponses(rows), e
}
func (u *UseCase) ListOrders(ctx context.Context, f domain.OrderFilter) ([]response.ThirdPartyOrderResponse, error) {
	rows, e := u.repo.ListOrders(ctx, f)
	return orderResponses(rows), e
}
func (u *UseCase) GetOrder(ctx context.Context, id int64) (response.ThirdPartyOrderResponse, error) {
	v, e := u.repo.GetOrder(ctx, id)
	return orderResponse(v), e
}
func (u *UseCase) UpdateOrderStatus(ctx context.Context, id int64, d request.ThirdPartyOrderStatusDTO, by uuid.UUID) (response.ThirdPartyOrderResponse, error) {
	v, e := u.repo.UpdateOrderStatus(ctx, id, d.Status, d.PurchaseRequisitionCode, d.PurchaseOrderCode, by)
	return orderResponse(v), e
}
func (u *UseCase) AddMovement(ctx context.Context, id int64, d request.ThirdPartyMovementDTO, by uuid.UUID) (response.ThirdPartyMovementResponse, error) {
	q, e := decimal.NewFromString(d.Quantity)
	if e != nil {
		return response.ThirdPartyMovementResponse{}, errors.New("quantity must be decimal")
	}
	v, e := u.repo.AddMovement(ctx, id, domain.Movement{MovementType: d.MovementType, Quantity: q, OccurredAt: d.OccurredAt, ReferenceType: d.ReferenceType, ReferenceCode: d.ReferenceCode, Notes: d.Notes, IdempotencyKey: d.IdempotencyKey, WarehouseID: d.WarehouseID, Lot: d.Lot, CreatedBy: by})
	return movementResponse(v), e
}
func (u *UseCase) UpsertGlobalConversion(ctx context.Context, d request.GlobalUnitConversionDTO, by uuid.UUID) (response.GlobalUnitConversionResponse, error) {
	f, e := decimal.NewFromString(d.Factor)
	if e != nil {
		return response.GlobalUnitConversionResponse{}, errors.New("factor must be decimal")
	}
	v, e := u.repo.UpsertGlobalConversion(ctx, domain.GlobalConversion{FromUOM: d.FromUOM, ToUOM: d.ToUOM, Factor: f, CreatedBy: by})
	if e != nil {
		return response.GlobalUnitConversionResponse{}, e
	}
	return globalConversionResponse(*v), nil
}
func (u *UseCase) ListGlobalConversions(ctx context.Context) ([]response.GlobalUnitConversionResponse, error) {
	rows, e := u.repo.ListGlobalConversions(ctx)
	out := make([]response.GlobalUnitConversionResponse, 0, len(rows))
	for _, v := range rows {
		out = append(out, globalConversionResponse(v))
	}
	return out, e
}
func (u *UseCase) DeleteGlobalConversion(ctx context.Context, id int64) error {
	return u.repo.DeleteGlobalConversion(ctx, id)
}
func (u *UseCase) OrderHistory(ctx context.Context, id int64) ([]response.ThirdPartyOrderHistoryResponse, error) {
	rows, e := u.repo.OrderHistory(ctx, id)
	out := make([]response.ThirdPartyOrderHistoryResponse, 0, len(rows))
	for _, v := range rows {
		var qty *string
		if v.Quantity != nil {
			s := v.Quantity.StringFixed(6)
			qty = &s
		}
		out = append(out, response.ThirdPartyOrderHistoryResponse{ID: v.ID, ServiceOrderID: v.ServiceOrderID, EventType: v.EventType, PreviousStatus: v.PreviousStatus, NewStatus: v.NewStatus, Quantity: qty, ReferenceType: v.ReferenceType, ReferenceCode: v.ReferenceCode, ActorID: v.ActorID.String(), OccurredAt: v.OccurredAt})
	}
	return out, e
}
func globalConversionResponse(v domain.GlobalConversion) response.GlobalUnitConversionResponse {
	return response.GlobalUnitConversionResponse{ID: v.ID, FromUOM: v.FromUOM, ToUOM: v.ToUOM, Factor: v.Factor.StringFixed(8), IsActive: v.IsActive}
}
func (u *UseCase) ListMovements(ctx context.Context, id int64) ([]response.ThirdPartyMovementResponse, error) {
	rows, e := u.repo.ListMovements(ctx, id)
	out := make([]response.ThirdPartyMovementResponse, 0, len(rows))
	for i := range rows {
		out = append(out, movementResponse(&rows[i]))
	}
	return out, e
}
func priceResponse(p *domain.Price) response.ThirdPartyPriceResponse {
	if p == nil {
		return response.ThirdPartyPriceResponse{}
	}
	v := response.ThirdPartyPriceResponse{ID: p.ID, ItemCode: p.ItemCode, Mask: p.Mask, SupplierCode: p.SupplierCode, OperationID: p.OperationID, ItemDescription: p.ItemDescription, SupplierName: p.SupplierName, OperationName: p.OperationName, UOM: p.UOM, ReferenceDate: p.ReferenceDate, Preferred: p.Preferred, UnitPrice: p.UnitPrice.StringFixed(6), FreightType: p.FreightType, FreightValue: p.FreightValue.StringFixed(6), TaxPercent: p.TaxPercent.StringFixed(6), Formula: p.Formula, IsActive: p.IsActive, Rules: []response.ThirdPartyPriceRuleResponse{}}
	if p.ConversionFactor != nil {
		s := p.ConversionFactor.StringFixed(8)
		v.ConversionFactor = &s
	}
	for _, x := range p.Rules {
		v.Rules = append(v.Rules, response.ThirdPartyPriceRuleResponse{ID: x.ID, Characteristic: x.Characteristic, Answer: x.Answer})
	}
	return v
}
func orderResponse(o *domain.ServiceOrder) response.ThirdPartyOrderResponse {
	if o == nil {
		return response.ThirdPartyOrderResponse{}
	}
	return response.ThirdPartyOrderResponse{ID: o.ID, Code: o.Code, PlannedSuggestionCode: o.PlannedSuggestionCode, PlanCode: o.PlanCode, ProductionOrderID: o.ProductionOrderID, RouteOperationID: o.RouteOperationID, OperationID: o.OperationID, ItemCode: o.ItemCode, ItemDescription: o.ItemDescription, SupplierName: o.SupplierName, OperationName: o.OperationName, Mask: o.Mask, SupplierCode: o.SupplierCode, ServiceItemCode: o.ServiceItemCode, UOM: o.UOM, Quantity: o.Quantity.StringFixed(6), FulfilledQuantity: o.FulfilledQuantity.StringFixed(6), PendingQuantity: o.Pending().StringFixed(6), StartDate: o.StartDate, DueDate: o.DueDate, Status: o.Status, PurchaseRequisitionCode: o.PurchaseRequisitionCode, PurchaseOrderCode: o.PurchaseOrderCode, RemittanceType: o.RemittanceType, Kanban: o.Kanban, Notes: o.Notes}
}
func orderResponses(rows []domain.ServiceOrder) []response.ThirdPartyOrderResponse {
	out := make([]response.ThirdPartyOrderResponse, 0, len(rows))
	for i := range rows {
		out = append(out, orderResponse(&rows[i]))
	}
	return out
}
func movementResponse(v *domain.Movement) response.ThirdPartyMovementResponse {
	if v == nil {
		return response.ThirdPartyMovementResponse{}
	}
	return response.ThirdPartyMovementResponse{ID: v.ID, ServiceOrderID: v.ServiceOrderID, MovementType: v.MovementType, Quantity: v.Quantity.StringFixed(6), OccurredAt: v.OccurredAt, ReferenceType: v.ReferenceType, ReferenceCode: v.ReferenceCode, Notes: v.Notes, IdempotencyKey: v.IdempotencyKey, WarehouseID: v.WarehouseID, Lot: v.Lot}
}
