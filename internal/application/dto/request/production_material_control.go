package request

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AddProductionMaterialDTO struct {
	ProductionOrderID   int64           `json:"production_order_id"`
	Kind                string          `json:"kind"`
	ItemCode            int64           `json:"item_code"`
	Mask                string          `json:"mask,omitempty"`
	SubstitutedItemCode *int64          `json:"substituted_item_code,omitempty"`
	Quantity            decimal.Decimal `json:"quantity"`
	WarehouseID         int64           `json:"warehouse_id"`
	AutomaticIssue      bool            `json:"automatic_issue"`
	Notes               *string         `json:"notes,omitempty"`
	CreatedBy           uuid.UUID       `json:"created_by"`
}

type MaterialReplacementDTO struct {
	ItemCode    int64           `json:"item_code"`
	Mask        string          `json:"mask,omitempty"`
	Quantity    decimal.Decimal `json:"quantity"`
	WarehouseID int64           `json:"warehouse_id"`
}

type ReplaceProductionMaterialDTO struct {
	MaterialID   int64                    `json:"material_id"`
	Replacements []MaterialReplacementDTO `json:"replacements"`
	CreatedBy    uuid.UUID                `json:"created_by"`
}

type LotAllocationDTO struct {
	WarehouseID int64           `json:"warehouse_id"`
	Lot         string          `json:"lot"`
	Address     *string         `json:"address,omitempty"`
	Quantity    decimal.Decimal `json:"quantity"`
}

type AllocateProductionLotsDTO struct {
	MaterialID     int64              `json:"material_id"`
	MovementKind   string             `json:"movement_kind"`
	Allocations    []LotAllocationDTO `json:"allocations"`
	CreatedBy      uuid.UUID          `json:"created_by"`
	ConfirmPartial bool               `json:"confirm_partial"`
}

type BatchAllocateProductionLotsDTO struct {
	MaterialIDs    []int64            `json:"material_ids"`
	MovementKind   string             `json:"movement_kind"`
	Lots           []LotAllocationDTO `json:"lots"`
	CreatedBy      uuid.UUID          `json:"created_by"`
	ConfirmPartial bool               `json:"confirm_partial"`
}

type AddScrapDestinationDTO struct {
	ProductionOrderID         int64           `json:"production_order_id"`
	ProductionOrderMaterialID *int64          `json:"production_order_material_id,omitempty"`
	ScrapItemCode             int64           `json:"scrap_item_code"`
	WarehouseID               int64           `json:"warehouse_id"`
	Lot                       *string         `json:"lot,omitempty"`
	Address                   *string         `json:"address,omitempty"`
	Quantity                  decimal.Decimal `json:"quantity"`
	DestinationDate           string          `json:"destination_date"`
	CreatedBy                 uuid.UUID       `json:"created_by"`
	DestinationKind           string          `json:"destination_kind"`
	ReturnQuantity            decimal.Decimal `json:"return_quantity"`
	ScrapQuantity             decimal.Decimal `json:"scrap_quantity"`
	SourceUOM                 string          `json:"source_uom"`
	ScrapUOM                  string          `json:"scrap_uom"`
}

type ConfigureWMSWarehouseDTO struct {
	WarehouseID                int64  `json:"warehouse_id"`
	IsWMS                      bool   `json:"is_wms"`
	IntermediateOutWarehouseID *int64 `json:"intermediate_out_warehouse_id,omitempty"`
}

type ConfigureManufacturingStockDTO struct {
	LotReturnMode string  `json:"lot_return_mode"`
	AutoIssueLots bool    `json:"auto_issue_lots"`
	MovementFrom  *string `json:"movement_from,omitempty"`
	MovementTo    *string `json:"movement_to,omitempty"`
}

type ConfigureManufacturingItemStockDTO struct {
	ItemCode           int64  `json:"item_code"`
	StockUOM           string `json:"stock_uom"`
	ControlsLot        bool   `json:"controls_lot"`
	ControlsAddress    bool   `json:"controls_address"`
	InventoryGroupType string `json:"inventory_group_type"`
	AutomaticIssueType string `json:"automatic_issue_type"`
	LineWarehouseID    *int64 `json:"line_warehouse_id,omitempty"`
}

type ConfigureWarehouseAddressDTO struct {
	WarehouseID int64  `json:"warehouse_id"`
	Address     string `json:"address"`
	IsActive    bool   `json:"is_active"`
}

type ConfigureTemporaryProductionLotDTO struct {
	ProductionOrderID int64  `json:"production_order_id"`
	Lot               string `json:"lot"`
	ManufacturedOn    string `json:"manufactured_on"`
	ExpiresOn         string `json:"expires_on"`
}
