package production_order_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"github.com/google/uuid"
)

type ProductionMaterialControlUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
}

type phase4MaterialRepository interface {
	AllocateLotsWithPolicy(context.Context, int64, string, []entity.LotAllocation, bool, uuid.UUID) ([]entity.LotAllocation, error)
	AllocateLotsBatchWithPolicy(context.Context, []int64, string, []entity.LotAllocation, bool, uuid.UUID) ([]entity.LotAllocation, error)
	DeleteScrapDestination(context.Context, int64, uuid.UUID) error
	UpdateScrapDestination(context.Context, int64, *entity.ScrapDestination) (*entity.ScrapDestination, error)
	ConfigureManufacturingStock(context.Context, entity.ManufacturingStockParameters) error
	ConfigureManufacturingItemStock(context.Context, entity.ManufacturingItemStockControl) error
	ConfigureWarehouseAddress(context.Context, int64, string, bool) error
	ConfigureTemporaryLot(context.Context, entity.TemporaryProductionLot) (*entity.TemporaryProductionLot, error)
	GetMaintenance(context.Context, *int64) ([]entity.ProductionOrderMaintenanceView, error)
}

func (uc *ProductionMaterialControlUseCase) phase4() (phase4MaterialRepository, error) {
	repo, ok := uc.Repo.(phase4MaterialRepository)
	if !ok {
		return nil, errorsuc.NewValidationError("production repository does not support phase 4 controls")
	}
	return repo, nil
}

func (uc *ProductionMaterialControlUseCase) List(ctx context.Context, orderID int64, kind string) ([]*entity.ProductionOrderMaterial, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListMaterials(ctx, orderID, entity.MaterialKind(strings.ToUpper(strings.TrimSpace(kind))))
}

func (uc *ProductionMaterialControlUseCase) Add(ctx context.Context, dto request.AddProductionMaterialDTO) (*entity.ProductionOrderMaterial, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	kind := entity.MaterialKind(strings.ToUpper(strings.TrimSpace(dto.Kind)))
	if dto.ProductionOrderID == 0 || dto.ItemCode == 0 || dto.WarehouseID == 0 ||
		(kind != entity.MaterialDemand && kind != entity.MaterialReturn) || !dto.Quantity.IsPositive() {
		return nil, errorsuc.NewValidationError("production_order_id, kind, item_code, warehouse_id and positive quantity are required")
	}
	return uc.Repo.AddMaterial(ctx, &entity.ProductionOrderMaterial{ProductionOrderID: dto.ProductionOrderID,
		Kind: kind, ItemCode: dto.ItemCode, Mask: dto.Mask, SubstitutedItemCode: dto.SubstitutedItemCode,
		Quantity: dto.Quantity, WarehouseID: dto.WarehouseID, AutomaticIssue: dto.AutomaticIssue,
		Notes: dto.Notes, CreatedBy: dto.CreatedBy})
}

func (uc *ProductionMaterialControlUseCase) Replace(ctx context.Context, dto request.ReplaceProductionMaterialDTO) ([]*entity.ProductionOrderMaterial, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.MaterialID == 0 || len(dto.Replacements) == 0 {
		return nil, errorsuc.NewValidationError("material_id and replacements are required")
	}
	replacements := make([]entity.MaterialSubstitution, 0, len(dto.Replacements))
	for _, replacement := range dto.Replacements {
		replacements = append(replacements, entity.MaterialSubstitution{ItemCode: replacement.ItemCode,
			Mask: replacement.Mask, Quantity: replacement.Quantity, WarehouseID: replacement.WarehouseID})
	}
	return uc.Repo.ReplaceMaterial(ctx, dto.MaterialID, replacements, dto.CreatedBy)
}

func (uc *ProductionMaterialControlUseCase) Delete(ctx context.Context, materialID int64) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.DeleteMaterial(ctx, materialID)
}

func (uc *ProductionMaterialControlUseCase) AllocateLots(ctx context.Context, dto request.AllocateProductionLotsDTO) ([]entity.LotAllocation, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.MaterialID == 0 {
		return nil, errorsuc.NewValidationError("material_id is required")
	}
	allocations := make([]entity.LotAllocation, 0, len(dto.Allocations))
	for _, allocation := range dto.Allocations {
		allocations = append(allocations, entity.LotAllocation{WarehouseID: allocation.WarehouseID,
			Lot: allocation.Lot, Address: allocation.Address, Quantity: allocation.Quantity})
	}
	if repo, ok := uc.Repo.(phase4MaterialRepository); ok {
		return repo.AllocateLotsWithPolicy(ctx, dto.MaterialID, dto.MovementKind, allocations, dto.ConfirmPartial, dto.CreatedBy)
	}
	return uc.Repo.AllocateLots(ctx, dto.MaterialID, dto.MovementKind, allocations, dto.CreatedBy)
}

func (uc *ProductionMaterialControlUseCase) AllocateLotsBatch(ctx context.Context, dto request.BatchAllocateProductionLotsDTO) ([]entity.LotAllocation, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if len(dto.MaterialIDs) == 0 {
		return nil, errorsuc.NewValidationError("material_ids are required")
	}
	lots := make([]entity.LotAllocation, 0, len(dto.Lots))
	for _, lot := range dto.Lots {
		lots = append(lots, entity.LotAllocation{WarehouseID: lot.WarehouseID, Lot: lot.Lot, Address: lot.Address, Quantity: lot.Quantity})
	}
	if repo, ok := uc.Repo.(phase4MaterialRepository); ok {
		return repo.AllocateLotsBatchWithPolicy(ctx, dto.MaterialIDs, dto.MovementKind, lots, dto.ConfirmPartial, dto.CreatedBy)
	}
	return uc.Repo.AllocateLotsBatch(ctx, dto.MaterialIDs, dto.MovementKind, lots, dto.CreatedBy)
}

func (uc *ProductionMaterialControlUseCase) AddScrap(ctx context.Context, dto request.AddScrapDestinationDTO) (*entity.ScrapDestination, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	date := datetime.ParseDateOrDefault(dto.DestinationDate, time.Now())
	if dto.ProductionOrderID == 0 || dto.ScrapItemCode == 0 || dto.WarehouseID == 0 || (!dto.Quantity.IsPositive() && !dto.ReturnQuantity.Add(dto.ScrapQuantity).IsPositive()) {
		return nil, fmt.Errorf("production_order_id, scrap_item_code, warehouse_id and positive quantity are required")
	}
	kind := strings.ToUpper(strings.TrimSpace(dto.DestinationKind))
	if kind == "" {
		kind = "ORDER_ITEM"
	}
	quantity := dto.Quantity
	if dto.ReturnQuantity.IsPositive() || dto.ScrapQuantity.IsPositive() {
		quantity = dto.ReturnQuantity.Add(dto.ScrapQuantity)
	}
	return uc.Repo.AddScrapDestination(ctx, &entity.ScrapDestination{ProductionOrderID: dto.ProductionOrderID,
		ProductionOrderMaterialID: dto.ProductionOrderMaterialID, ScrapItemCode: dto.ScrapItemCode,
		WarehouseID: dto.WarehouseID, Lot: dto.Lot, Address: dto.Address, Quantity: quantity,
		DestinationDate: date, CreatedBy: dto.CreatedBy, DestinationKind: kind, ReturnQuantity: dto.ReturnQuantity,
		ScrapQuantity: dto.ScrapQuantity, SourceUOM: strings.ToUpper(strings.TrimSpace(dto.SourceUOM)), ScrapUOM: strings.ToUpper(strings.TrimSpace(dto.ScrapUOM))})
}

func (uc *ProductionMaterialControlUseCase) UpdateScrap(ctx context.Context, id int64, dto request.AddScrapDestinationDTO) (*entity.ScrapDestination, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	dto.CreatedBy = userID
	date := datetime.ParseDateOrDefault(dto.DestinationDate, time.Now())
	kind := strings.ToUpper(strings.TrimSpace(dto.DestinationKind))
	if kind == "" {
		kind = "ORDER_ITEM"
	}
	quantity := dto.Quantity
	if dto.ReturnQuantity.IsPositive() || dto.ScrapQuantity.IsPositive() {
		quantity = dto.ReturnQuantity.Add(dto.ScrapQuantity)
	}
	if id == 0 || dto.ProductionOrderID == 0 || dto.ScrapItemCode == 0 || dto.WarehouseID == 0 || !quantity.IsPositive() {
		return nil, errorsuc.NewValidationError("valid destination, order, item, warehouse and quantity are required")
	}
	repo, err := uc.phase4()
	if err != nil {
		return nil, err
	}
	return repo.UpdateScrapDestination(ctx, id, &entity.ScrapDestination{ProductionOrderID: dto.ProductionOrderID, ProductionOrderMaterialID: dto.ProductionOrderMaterialID, ScrapItemCode: dto.ScrapItemCode, WarehouseID: dto.WarehouseID, Lot: dto.Lot, Address: dto.Address, Quantity: quantity, DestinationDate: date, CreatedBy: userID, DestinationKind: kind, ReturnQuantity: dto.ReturnQuantity, ScrapQuantity: dto.ScrapQuantity, SourceUOM: strings.ToUpper(strings.TrimSpace(dto.SourceUOM)), ScrapUOM: strings.ToUpper(strings.TrimSpace(dto.ScrapUOM))})
}

func (uc *ProductionMaterialControlUseCase) DeleteScrap(ctx context.Context, id int64) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	repo, err := uc.phase4()
	if err != nil {
		return err
	}
	return repo.DeleteScrapDestination(ctx, id, userID)
}

func (uc *ProductionMaterialControlUseCase) ConfigureWMS(ctx context.Context, dto request.ConfigureWMSWarehouseDTO) (*entity.WMSWarehouseSettings, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.WarehouseID == 0 {
		return nil, errorsuc.NewValidationError("warehouse_id is required")
	}
	if dto.IsWMS && dto.IntermediateOutWarehouseID == nil {
		return nil, errorsuc.NewValidationError("intermediate_out_warehouse_id is required for WMS")
	}
	return uc.Repo.UpsertWMSSettings(ctx, entity.WMSWarehouseSettings{WarehouseID: dto.WarehouseID, IsWMS: dto.IsWMS, IntermediateOutWarehouseID: dto.IntermediateOutWarehouseID})
}

func (uc *ProductionMaterialControlUseCase) ConfigureStock(ctx context.Context, dto request.ConfigureManufacturingStockDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	mode := strings.ToUpper(strings.TrimSpace(dto.LotReturnMode))
	if mode != "A" && mode != "I" && mode != "E" {
		return errorsuc.NewValidationError("lot_return_mode must be A, I or E")
	}
	var from, to *time.Time
	if dto.MovementFrom != nil {
		from = datetime.ParseDatePtr(dto.MovementFrom)
	}
	if dto.MovementTo != nil {
		to = datetime.ParseDatePtr(dto.MovementTo)
	}
	if from != nil && to != nil && from.After(*to) {
		return errorsuc.NewValidationError("movement_from must not be after movement_to")
	}
	repo, err := uc.phase4()
	if err != nil {
		return err
	}
	return repo.ConfigureManufacturingStock(ctx, entity.ManufacturingStockParameters{LotReturnMode: mode, AutoIssueLots: dto.AutoIssueLots, MovementFrom: from, MovementTo: to})
}
func (uc *ProductionMaterialControlUseCase) ConfigureItemStock(ctx context.Context, dto request.ConfigureManufacturingItemStockDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	dto.InventoryGroupType = strings.ToUpper(strings.TrimSpace(dto.InventoryGroupType))
	dto.AutomaticIssueType = strings.ToUpper(strings.TrimSpace(dto.AutomaticIssueType))
	if dto.AutomaticIssueType == "" {
		dto.AutomaticIssueType = "ISSUE"
	}
	if dto.ItemCode == 0 || dto.StockUOM == "" || (dto.InventoryGroupType != "STANDARD" && dto.InventoryGroupType != "SECONDARY_MATERIAL") {
		return errorsuc.NewValidationError("valid item_code, stock_uom and inventory_group_type are required")
	}
	if (dto.AutomaticIssueType != "ISSUE" && dto.AutomaticIssueType != "TRANSFER") || (dto.AutomaticIssueType == "TRANSFER" && dto.LineWarehouseID == nil) {
		return errorsuc.NewValidationError("automatic_issue_type must be ISSUE or TRANSFER with line_warehouse_id")
	}
	repo, err := uc.phase4()
	if err != nil {
		return err
	}
	return repo.ConfigureManufacturingItemStock(ctx, entity.ManufacturingItemStockControl{ItemCode: dto.ItemCode, StockUOM: strings.ToUpper(dto.StockUOM), ControlsLot: dto.ControlsLot, ControlsAddress: dto.ControlsAddress, InventoryGroupType: dto.InventoryGroupType, AutomaticIssueType: dto.AutomaticIssueType, LineWarehouseID: dto.LineWarehouseID})
}
func (uc *ProductionMaterialControlUseCase) ConfigureAddress(ctx context.Context, dto request.ConfigureWarehouseAddressDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if dto.WarehouseID == 0 || strings.TrimSpace(dto.Address) == "" {
		return errorsuc.NewValidationError("warehouse_id and address are required")
	}
	repo, err := uc.phase4()
	if err != nil {
		return err
	}
	return repo.ConfigureWarehouseAddress(ctx, dto.WarehouseID, strings.TrimSpace(dto.Address), dto.IsActive)
}
func (uc *ProductionMaterialControlUseCase) ConfigureTemporaryLot(ctx context.Context, dto request.ConfigureTemporaryProductionLotDTO) (*entity.TemporaryProductionLot, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	manufactured := datetime.ParseDateOrDefault(dto.ManufacturedOn, time.Time{})
	expires := datetime.ParseDateOrDefault(dto.ExpiresOn, time.Time{})
	if dto.ProductionOrderID == 0 || strings.TrimSpace(dto.Lot) == "" || manufactured.IsZero() || expires.Before(manufactured) || expires.Before(time.Now().Truncate(24*time.Hour)) {
		return nil, errorsuc.NewValidationError("valid order, lot, manufacture and expiration dates are required")
	}
	repo, err := uc.phase4()
	if err != nil {
		return nil, err
	}
	return repo.ConfigureTemporaryLot(ctx, entity.TemporaryProductionLot{ProductionOrderID: dto.ProductionOrderID, Lot: strings.TrimSpace(dto.Lot), ManufacturedOn: manufactured, ExpiresOn: expires})
}
func (uc *ProductionMaterialControlUseCase) Maintenance(ctx context.Context, id *int64) ([]entity.ProductionOrderMaintenanceView, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	repo, err := uc.phase4()
	if err != nil {
		return nil, err
	}
	return repo.GetMaintenance(ctx, id)
}
