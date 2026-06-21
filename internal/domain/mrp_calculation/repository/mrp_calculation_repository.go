package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	orderpriority "github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity"
	salesdivision "github.com/FelipePn10/panossoerp/internal/domain/sales_division/entity"
)

type MRPCalculationRepository interface {
	CreateProfile(ctx context.Context, p *entity.MRPItemProfile) (*entity.MRPItemProfile, error)
	GetProfiles(ctx context.Context, itemCode, planCode int64) ([]*entity.MRPItemProfile, error)
	DeleteProfilesByPlan(ctx context.Context, planCode int64) error

	StartCalculation(ctx context.Context, planCode int64) (*entity.MRPCalculationLog, error)
	FinishCalculation(ctx context.Context, logCode int64, status string, errors map[string]interface{}, totalItems, totalOrders int) (*entity.MRPCalculationLog, error)
	GetCalculationLog(ctx context.Context, logCode int64) (*entity.MRPCalculationLog, error)
	ListCalculationLogs(ctx context.Context, planCode int64) ([]*entity.MRPCalculationLog, error)

	CreateStockSnapshot(ctx context.Context, s *entity.StockSnapshot) (*entity.StockSnapshot, error)
	GetStockSnapshot(ctx context.Context, itemCode int64) (*entity.StockSnapshot, error)

	CreateSalesOrderDemand(ctx context.Context, d *entity.SalesOrderDemand) (*entity.SalesOrderDemand, error)
	GetSalesOrderDemand(ctx context.Context, code int64) (*entity.SalesOrderDemand, error)
	ListSalesOrderDemandsByItem(ctx context.Context, itemCode int64) ([]*entity.SalesOrderDemand, error)
	UpdateSalesOrderDemandStatus(ctx context.Context, code int64, status string, deliveredQty float64) (*entity.SalesOrderDemand, error)

	CreateConfiguredItemRule(ctx context.Context, r *entity.ConfiguredItemRule) (*entity.ConfiguredItemRule, error)
	GetConfiguredItemRules(ctx context.Context, itemCode int64) ([]*entity.ConfiguredItemRule, error)
	DeleteConfiguredItemRule(ctx context.Context, code int64) error

	CreatePlannedOrderSuggestion(ctx context.Context, s *entity.PlannedOrderSuggestion) (*entity.PlannedOrderSuggestion, error)
	GetSuggestionByCode(ctx context.Context, code int64) (*entity.PlannedOrderSuggestion, error)
	ListSuggestionsByPlan(ctx context.Context, planCode int64) ([]*entity.PlannedOrderSuggestion, error)
	DeleteSuggestionsByPlan(ctx context.Context, planCode int64) error

	// Bulk pre-load methods for MRP run optimization (Problem 3).
	// Returns maps keyed by item_code, built in a single query each.
	ListAllStockSnapshots(ctx context.Context) (map[int64]*entity.StockSnapshot, error)
	ListAllConfiguredRules(ctx context.Context) (map[int64][]*entity.ConfiguredItemRule, error)

	// Exception messages — generated when existing firm orders diverge from demand.
	CreateExceptionMessage(ctx context.Context, msg *entity.MRPExceptionMessage) error
	ListExceptionsByPlan(ctx context.Context, planCode int64) ([]*entity.MRPExceptionMessage, error)
	DeleteExceptionsByPlan(ctx context.Context, planCode int64) error

	// ---------- New methods for enhanced MRP features ----------

	// LoadTypedPlanningParams loads the planning_params table and returns a typed struct.
	LoadTypedPlanningParams(ctx context.Context) (*entity.TypedPlanningParams, error)

	// ListAllOrderPriorities returns all active order priority rules.
	ListAllOrderPriorities(ctx context.Context) ([]*orderpriority.OrderPriority, error)

	// ListItemMachineTimes returns machine-time associations for the given item codes,
	// grouped by item_code. Each item may be assigned to multiple machines.
	ListItemMachineTimes(ctx context.Context, itemCodes []int64) (map[int64][]*entity.MachineTimeInfo, error)

	// ListKanbanCards returns all active kanban cards.
	ListKanbanCards(ctx context.Context) ([]*entity.KanbanCardInfo, error)

	// ListMPSItems returns all MPS schedule entries for the given plan.
	ListMPSItems(ctx context.Context, planCode int64) ([]*entity.MPSItemInfo, error)

	// ListItemPlanningExtras returns extra planning parameters (e.g. maximum_stock),
	// keyed by item_code.
	ListItemPlanningExtras(ctx context.Context) (map[int64]*entity.ItemPlanningExtra, error)

	// CreateMachineSchedule persists a machine allocation record.
	CreateMachineSchedule(ctx context.Context, schedule *entity.MachineScheduleInfo) error

	// ListItemSalesDivisions returns the sales divisions associated with each item,
	// used for technical-assistance order classification.
	ListItemSalesDivisions(ctx context.Context, itemCodes []int64) (map[int64]*salesdivision.SalesDivision, error)

	// UpdatePlannedOrderPriority sets the priority field on a planned order suggestion.
	UpdatePlannedOrderPriority(ctx context.Context, suggestionCode int64, priority string) error

	// UpdatePlannedOrderMachine sets machine_id and production_time on a planned order.
	UpdatePlannedOrderMachine(ctx context.Context, suggestionCode int64, machineID int64, productionTime float64) error
}
