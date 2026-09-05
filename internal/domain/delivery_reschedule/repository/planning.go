package repository

import (
	"context"
	"errors"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/google/uuid"
)

var ErrOrderNotFound = errors.New("pedido de venda não encontrado")
var ErrBatchConflict = errors.New("esta chave de controle de duplicidade já foi usada com outros dados")
var ErrLinePrecondition = errors.New("a linha da reprogramação de entrega não atende às condições necessárias")

type PlanningItem struct {
	SalesOrderItemCode   int64                `json:"sales_order_item_code"`
	ItemCode             valueobject.ItemCode `json:"item_code"`
	Sequence             int32                `json:"sequence"`
	RequestedQty         string               `json:"requested_qty"`
	AttendedQty          string               `json:"attended_qty"`
	CancelledQty         string               `json:"cancelled_qty"`
	OpenQty              string               `json:"open_qty"`
	CurrentDate          *time.Time           `json:"current_date,omitempty"`
	FirmDate             bool                 `json:"firm_date"`
	ReservedQty          string               `json:"reserved_qty"`
	IndependentDemandQty string               `json:"independent_demand_qty"`
	PlannedOrderCount    int64                `json:"planned_order_count"`
	FirmOrderCount       int64                `json:"firm_order_count"`
	PurchaseOrderCount   int64                `json:"purchase_order_count"`
	ShipmentCount        int64                `json:"shipment_count"`
	ShipmentStatus       *string              `json:"shipment_status,omitempty"`
	CRPOverloaded        bool                 `json:"crp_overloaded"`
	APSDate              *time.Time           `json:"aps_date,omitempty"`
	InvoicedQty          string               `json:"invoiced_qty"`
	CanReschedule        bool                 `json:"can_reschedule"`
	Severity             string               `json:"severity"`
	SuggestionSource     string               `json:"suggestion_source"`
	SuggestedDate        *time.Time           `json:"suggested_date,omitempty"`
	Justification        string               `json:"justification"`
}

type BatchLine struct {
	SalesOrderItemCode int64
	ItemCode           valueobject.ItemCode
	OldDate            time.Time
	NewDate            time.Time
	Reason             *string
}

type BatchCommand struct {
	ID             uuid.UUID
	IdempotencyKey string
	PayloadHash    string
	SalesOrderCode int64
	CreatedBy      uuid.UUID
	Lines          []BatchLine
}

type BatchResult struct {
	Codes    []int64 `json:"codes"`
	Replayed bool    `json:"replayed"`
}

type PlanningRepository interface {
	Preview(ctx context.Context, salesOrderCode int64) ([]PlanningItem, error)
	CreateBatch(ctx context.Context, command BatchCommand) (*BatchResult, error)
}
