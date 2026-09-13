package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type MaterialKind string

const (
	MaterialDemand MaterialKind = "DEMAND"
	MaterialReturn MaterialKind = "RETURN"
)

// Devolvida direto pelo handler de materiais da ordem, então as tags são o
// contrato da API. Sem elas os campos saíam em PascalCase e o middleware de
// código de item não reconhecia `ItemCode`: a lista de materiais da ordem
// mostrava a chave interna no lugar do código que o usuário conhece.
type ProductionOrderMaterial struct {
	ID                  int64           `json:"id"`
	ProductionOrderID   int64           `json:"production_order_id"`
	Kind                MaterialKind    `json:"kind"`
	ItemCode            int64           `json:"item_code"`
	Mask                string          `json:"mask"`
	SubstitutedItemCode *int64          `json:"substituted_item_code"`
	Quantity            decimal.Decimal `json:"quantity"`
	AttendedQuantity    decimal.Decimal `json:"attended_quantity"`
	WarehouseID         int64           `json:"warehouse_id"`
	AutomaticIssue      bool            `json:"automatic_issue"`
	Notes               *string         `json:"notes"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	CreatedBy           uuid.UUID       `json:"created_by"`
}

type LotAllocation struct {
	ID                        int64
	ProductionOrderMaterialID int64
	MovementKind              string
	WarehouseID               int64
	Lot                       string
	Address                   *string
	Quantity                  decimal.Decimal
	CreatedAt                 time.Time
	CreatedBy                 uuid.UUID
}

type MaterialSubstitution struct {
	ItemCode    int64
	Mask        string
	Quantity    decimal.Decimal
	WarehouseID int64
}

type ScrapDestination struct {
	ID                        int64
	ProductionOrderID         int64
	ProductionOrderMaterialID *int64
	ScrapItemCode             int64
	WarehouseID               int64
	Lot                       *string
	Address                   *string
	Quantity                  decimal.Decimal
	DestinationDate           time.Time
	CreatedAt                 time.Time
	CreatedBy                 uuid.UUID
	DestinationKind           string
	ReturnQuantity            decimal.Decimal
	ScrapQuantity             decimal.Decimal
	SourceUOM                 string
	ScrapUOM                  string
}

type ManufacturingStockParameters struct {
	LotReturnMode            string
	AutoIssueLots            bool
	MovementFrom, MovementTo *time.Time
}
type ManufacturingItemStockControl struct {
	ItemCode                     int64
	StockUOM                     string
	ControlsLot, ControlsAddress bool
	InventoryGroupType           string
	AutomaticIssueType           string
	LineWarehouseID              *int64
}
type TemporaryProductionLot struct {
	ProductionOrderID         int64
	Lot                       string
	ManufacturedOn, ExpiresOn time.Time
}

type ProductionOrderMaintenanceView struct {
	ProductionOrder *ProductionOrder        `json:"production_order"`
	OriginType      string                  `json:"origin_type"`
	OrderType       string                  `json:"order_type"`
	Rework          bool                    `json:"rework"`
	TemporaryLot    *TemporaryProductionLot `json:"temporary_lot,omitempty"`
}

type WMSWarehouseSettings struct {
	WarehouseID                int64  `json:"warehouse_id"`
	IsWMS                      bool   `json:"is_wms"`
	IntermediateOutWarehouseID *int64 `json:"intermediate_out_warehouse_id,omitempty"`
}
