package mrp_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	mrpcalcrepo "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	plannedentity "github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
	plannedrepo "github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository"
)

// orderFirmer firms a freshly-created planned order, generating its Production Order
// (OF) and/or service requisition. FirmPlannedOrderUseCase satisfies it.
type orderFirmer interface {
	Execute(ctx context.Context, dto request.FirmOrderDTO) (*response.PlannedOrderResponse, error)
}

// mapMRPOrderType converts the MRP service's internal order type strings
// (FABRICACAO, COMPRA, SERVICO, TECHNICAL_ASSISTANCE) into the planned_orders
// enum values accepted by the database.
func mapMRPOrderType(mrpType string) types.OrderType {
	switch strings.ToUpper(mrpType) {
	case "COMPRA":
		return types.OrderPurchase
	case "SERVICO":
		return types.OrderOutsourcing
	case "TECHNICAL_ASSISTANCE":
		return types.OrderTechnicalAssistance
	default: // FABRICACAO and anything else
		return types.OrderProduction
	}
}

// mapMRPDemandType converts MRP internal demand type strings (INDEPENDENTE, DEPENDENTE)
// into demand_type_enum values accepted by the planned_orders table.
func mapMRPDemandType(mrpDemand string) types.DemandType {
	switch strings.ToUpper(mrpDemand) {
	case "INDEPENDENTE":
		return types.DemandIndependent
	case "DEPENDENTE":
		return types.DemandReplenishment // BOM-driven demand maps to replenishment
	case "FORECAST":
		return types.DemandForecast
	case "SALES_ORDER":
		return types.DemandSalesOrder
	case "SAFETY_STOCK":
		return types.DemandSafetyStock
	default:
		return types.DemandIndependent
	}
}

// FirmarSugestaoMRPUseCase converts a single mrp_planned_suggestions row into
// a firm planned_order. This is the "accept MRP suggestion" step that bridges
// the two tables: the MRP engine writes to mrp_planned_suggestions; the
// procurement/production flow reads from planned_orders.
type FirmarSugestaoMRPUseCase struct {
	MRPRepo     mrpcalcrepo.MRPCalculationRepository
	PlannedRepo plannedrepo.PlannedOrderRepository
	Auth        ports.AuthService
	// Firmer is optional. When set, accepting a suggestion not only creates the
	// planned order but also firms it — generating the Production Order (OF) for
	// PRODUCTION orders and the service requisition for external operations —
	// mirroring SAP's "convert planned order → production order" in one step.
	Firmer orderFirmer
}

type FirmarSugestaoMRPResponse struct {
	SuggestionCode int64     `json:"suggestion_code"`
	PlannedCode    int64     `json:"planned_code"`
	OrderNumber    int64     `json:"order_number"`
	ItemCode       int64     `json:"item_code"`
	Quantity       float64   `json:"quantity"`
	OrderType      string    `json:"order_type"`
	NeedDate       time.Time `json:"need_date"`
	Status         string    `json:"status"`
	IsFirm         bool      `json:"is_firm"`
	PlanCode       *int64    `json:"plan_code"`
}

func (uc *FirmarSugestaoMRPUseCase) Execute(ctx context.Context, suggestionCode int64) (*FirmarSugestaoMRPResponse, error) {
	if !uc.Auth.CanCreatePlannedOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	sugg, err := uc.MRPRepo.GetSuggestionByCode(ctx, suggestionCode)
	if err != nil {
		return nil, fmt.Errorf("suggestion %d not found: %w", suggestionCode, err)
	}

	nextNum, err := uc.PlannedRepo.GetNextOrderNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting next order number: %w", err)
	}

	userID, _ := uc.Auth.UserID(ctx)

	order := &plannedentity.PlannedOrder{
		OrderNumber: nextNum,
		ItemCode:    sugg.ItemCode,
		Quantity:    sugg.Quantity,
		OrderType:   mapMRPOrderType(sugg.OrderType),
		Status:      types.StatusPlanned,
		DemandType:  mapMRPDemandType(sugg.DemandType),
		NeedDate:    sugg.NeedDate,
		StartDate:   sugg.StartDate,
		LLC:         sugg.LLC,
		Notes:       sugg.Notes,
		// When a Firmer is wired, create the order NOT firm so the firm step's
		// "first firming" guard fires and generates the OF/requisition. Without a
		// Firmer, keep the legacy behaviour (mark firm directly).
		IsFirm:    uc.Firmer == nil,
		IsActive:  true,
		CreatedBy: userID,
	}

	if sugg.PlanCode != 0 {
		pc := sugg.PlanCode
		order.PlanCode = &pc
	}

	if sugg.ParentItemCode != nil {
		order.Notes = mergeNote(order.Notes, fmt.Sprintf("parent_item=%d", *sugg.ParentItemCode))
	}

	created, err := uc.PlannedRepo.Create(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("creating planned order from suggestion %d: %w", suggestionCode, err)
	}

	isFirm := created.IsFirm
	// Firm the freshly-created order so its OF / service requisition is generated.
	if uc.Firmer != nil {
		if _, ferr := uc.Firmer.Execute(ctx, request.FirmOrderDTO{OrderCode: created.Code}); ferr != nil {
			return nil, fmt.Errorf("firming planned order %d from suggestion %d: %w", created.Code, suggestionCode, ferr)
		}
		isFirm = true
	}

	return &FirmarSugestaoMRPResponse{
		SuggestionCode: sugg.Code,
		PlannedCode:    created.Code,
		OrderNumber:    created.OrderNumber,
		ItemCode:       created.ItemCode,
		Quantity:       created.Quantity,
		OrderType:      string(created.OrderType),
		NeedDate:       created.NeedDate,
		Status:         string(created.Status),
		IsFirm:         isFirm,
		PlanCode:       created.PlanCode,
	}, nil
}

func mergeNote(existing *string, extra string) *string {
	if existing == nil || *existing == "" {
		return &extra
	}
	s := *existing + "; " + extra
	return &s
}
