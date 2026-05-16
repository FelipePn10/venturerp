package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

type StockRepository interface {
	// Stock movements
	CreateMovement(ctx context.Context, m *entity.StockMovement) (*entity.StockMovement, error)
	ListMovements(ctx context.Context) ([]*entity.StockMovement, error)
	ListMovementsByItem(ctx context.Context, itemCode int64) ([]*entity.StockMovement, error)
	ListMovementsByWarehouse(ctx context.Context, warehouseID int64) ([]*entity.StockMovement, error)
	ListMovementsByDateRange(ctx context.Context, from, to time.Time) ([]*entity.StockMovement, error)

	// Stock balance
	GetBalance(ctx context.Context, itemCode int64, mask string, warehouseID int64) (*entity.StockBalance, error)
	ListBalances(ctx context.Context) ([]*entity.StockBalance, error)
	ListBalancesByWarehouse(ctx context.Context, warehouseID int64) ([]*entity.StockBalance, error)
	ListBalancesByItem(ctx context.Context, itemCode int64) ([]*entity.StockBalance, error)
	UpsertBalance(ctx context.Context, b *entity.StockBalance) error

	// Stock reservations
	CreateReservation(ctx context.Context, r *entity.StockReservation) (*entity.StockReservation, error)
	GetReservation(ctx context.Context, id int64) (*entity.StockReservation, error)
	ListReservations(ctx context.Context) ([]*entity.StockReservation, error)
	ListReservationsByItem(ctx context.Context, itemCode int64) ([]*entity.StockReservation, error)
	ListActiveReservations(ctx context.Context) ([]*entity.StockReservation, error)
	CancelReservation(ctx context.Context, id int64) error
	ConsumeReservation(ctx context.Context, id int64) error

	// Physical inventory
	CreateInventory(ctx context.Context, inv *entity.PhysicalInventory) (*entity.PhysicalInventory, error)
	GetInventory(ctx context.Context, id int64) (*entity.PhysicalInventory, error)
	GetInventoryByCode(ctx context.Context, code int64) (*entity.PhysicalInventory, error)
	ListInventories(ctx context.Context) ([]*entity.PhysicalInventory, error)
	ListInventoriesByStatus(ctx context.Context, status string) ([]*entity.PhysicalInventory, error)
	UpdateInventoryStatus(ctx context.Context, id int64, status string) error
	CloseInventory(ctx context.Context, id int64) error

	// Physical inventory items
	UpsertInventoryItem(ctx context.Context, item *entity.PhysicalInventoryItem) error
	ListInventoryItems(ctx context.Context, inventoryID int64) ([]*entity.PhysicalInventoryItem, error)
	CountInventoryItem(ctx context.Context, item *entity.PhysicalInventoryItem) error
	AdjustInventoryItem(ctx context.Context, item *entity.PhysicalInventoryItem) error
}
