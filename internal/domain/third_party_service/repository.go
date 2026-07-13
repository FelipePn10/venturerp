package third_party_service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Repository interface {
	CreatePrice(context.Context, *Price, string) (*Price, error)
	UpdatePrice(context.Context, *Price, string) (*Price, error)
	DeletePrice(context.Context, int64, string, uuid.UUID) error
	GetPrice(context.Context, int64) (*Price, error)
	ListPrices(context.Context, PriceFilter) ([]Price, error)
	ResolvePrice(context.Context, int64, string, int64, int64, time.Time, map[string]string) (*Price, error)
	ResolveConversionFactor(context.Context, int64, string, string) (*decimal.Decimal, error)
	History(context.Context, int64) ([]History, error)
	Readjust(context.Context, []int64, decimal.Decimal, time.Time, string, uuid.UUID) ([]Price, error)
	CopyMove(context.Context, []int64, int64, int64, bool, time.Time, string, uuid.UUID) ([]Price, error)
	CreateOrdersForProduction(context.Context, int64, uuid.UUID) ([]ServiceOrder, error)
	LinkRequisitionToProduction(context.Context, int64, int64) error
	ListOrders(context.Context, OrderFilter) ([]ServiceOrder, error)
	GetOrder(context.Context, int64) (*ServiceOrder, error)
	UpdateOrderStatus(context.Context, int64, string, *int64, *int64, uuid.UUID) (*ServiceOrder, error)
	AddMovement(context.Context, int64, Movement) (*Movement, error)
	ListMovements(context.Context, int64) ([]Movement, error)
	UpsertGlobalConversion(context.Context, GlobalConversion) (*GlobalConversion, error)
	ListGlobalConversions(context.Context) ([]GlobalConversion, error)
	DeleteGlobalConversion(context.Context, int64) error
	OrderHistory(context.Context, int64) ([]OrderHistory, error)
}
