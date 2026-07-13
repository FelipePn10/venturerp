package mrp_uc

import (
	"context"
	"errors"
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
type orderTransitioner interface {
	ExecuteTransition(context.Context, request.TransitionPlannedOrderDTO) ([]*response.PlannedOrderResponse, error)
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
	case "INTER_FACTORY":
		return types.DemandSalesOrder
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
	return uc.execute(ctx, suggestionCode, true)
}

func (uc *FirmarSugestaoMRPUseCase) execute(ctx context.Context, suggestionCode int64, firm bool) (*FirmarSugestaoMRPResponse, error) {
	if !uc.Auth.CanCreatePlannedOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	sugg, err := uc.MRPRepo.GetSuggestionByCode(ctx, suggestionCode)
	if err != nil {
		return nil, fmt.Errorf("suggestion %d not found: %w", suggestionCode, err)
	}
	if existing, existingErr := uc.PlannedRepo.GetByMRPSuggestionCode(ctx, suggestionCode); existingErr == nil {
		return suggestionResponse(sugg.Code, existing), nil
	}

	var nextNum int64
	if sugg.OrderNumber != nil {
		nextNum = *sugg.OrderNumber
	} else {
		nextNum, err = uc.PlannedRepo.GetNextOrderNumber(ctx)
		if err != nil {
			return nil, fmt.Errorf("getting next order number: %w", err)
		}
	}

	userID, _ := uc.Auth.UserID(ctx)

	order := &plannedentity.PlannedOrder{
		OrderNumber:          nextNum,
		ItemCode:             sugg.ItemCode,
		Mask:                 stringPtrIfNotEmpty(sugg.Mask),
		Quantity:             sugg.Quantity,
		OrderType:            mapMRPOrderType(sugg.OrderType),
		Status:               types.StatusPlanned,
		DemandType:           mapMRPDemandType(sugg.DemandType),
		NeedDate:             sugg.NeedDate,
		StartDate:            sugg.StartDate,
		LLC:                  sugg.LLC,
		WarehouseCode:        sugg.WarehouseCode,
		InterFactory:         sugg.InterFactory,
		SourceEnterpriseCode: sugg.SourceEnterpriseCode,
		AutoRelease:          sugg.AutoRelease,
		MRPSuggestionCode:    &sugg.Code,
		Notes:                sugg.Notes,
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

	var firmedResponse *response.PlannedOrderResponse
	// Firm the freshly-created order so its OF / service requisition is generated.
	if uc.Firmer != nil && firm {
		firmed, ferr := uc.Firmer.Execute(ctx, request.FirmOrderDTO{OrderCode: created.Code})
		if ferr != nil {
			return nil, fmt.Errorf("firming planned order %d from suggestion %d: %w", created.Code, suggestionCode, ferr)
		}
		firmedResponse = firmed
	} else if uc.Firmer != nil {
		transitioner, ok := uc.Firmer.(orderTransitioner)
		if !ok {
			return nil, errors.New("configured order releaser does not support RELEASED transition")
		}
		released, releaseErr := transitioner.ExecuteTransition(ctx, request.TransitionPlannedOrderDTO{OrderCodes: []int64{created.Code}, Target: "RELEASED"})
		if releaseErr != nil {
			return nil, fmt.Errorf("releasing planned order %d from suggestion %d: %w", created.Code, suggestionCode, releaseErr)
		}
		firmedResponse = released[0]
	}

	result := suggestionResponse(sugg.Code, created)
	if firmedResponse != nil {
		result.Status = firmedResponse.Status
		result.IsFirm = firmedResponse.IsFirm
	}
	return result, nil
}

func stringPtrIfNotEmpty(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

// ExecuteAutoRelease converts only inter-factory suggestions explicitly marked
// for automatic release. The suggestion origin unique key makes reruns idempotent.
func (uc *FirmarSugestaoMRPUseCase) ExecuteAutoRelease(ctx context.Context, planCode int64) error {
	suggestions, err := uc.MRPRepo.ListSuggestionsByPlan(ctx, planCode)
	if err != nil {
		return err
	}
	for _, suggestion := range suggestions {
		if !suggestion.InterFactory || !suggestion.AutoRelease {
			continue
		}
		if _, err := uc.execute(ctx, suggestion.Code, false); err != nil {
			return fmt.Errorf("auto-releasing suggestion %d: %w", suggestion.Code, err)
		}
	}
	return nil
}

func suggestionResponse(suggestionCode int64, created *plannedentity.PlannedOrder) *FirmarSugestaoMRPResponse {
	return &FirmarSugestaoMRPResponse{
		SuggestionCode: suggestionCode,
		PlannedCode:    created.Code,
		OrderNumber:    created.OrderNumber,
		ItemCode:       created.ItemCode,
		Quantity:       created.Quantity,
		OrderType:      string(created.OrderType),
		NeedDate:       created.NeedDate,
		Status:         string(created.Status),
		IsFirm:         created.IsFirm,
		PlanCode:       created.PlanCode,
	}
}

func mergeNote(existing *string, extra string) *string {
	if existing == nil || *existing == "" {
		return &extra
	}
	s := *existing + "; " + extra
	return &s
}
