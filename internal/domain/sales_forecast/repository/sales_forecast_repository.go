package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity"
)

type SalesForecastRepository interface {
	// Forecasts
	CreateForecast(ctx context.Context, f *entity.SalesForecast) (*entity.SalesForecast, error)
	UpdateForecast(ctx context.Context, f *entity.SalesForecast) (*entity.SalesForecast, error)
	GetForecastByItem(ctx context.Context, itemCode int64) ([]*entity.SalesForecast, error)
	ListForecasts(ctx context.Context, year int) ([]*entity.SalesForecast, error)
	DeleteForecast(ctx context.Context, id int64) error

	// Forecast Blocks
	CreateBlock(ctx context.Context, b *entity.SalesForecastBlock) (*entity.SalesForecastBlock, error)
	ListBlocks(ctx context.Context) ([]*entity.SalesForecastBlock, error)
	IsBlocked(ctx context.Context, date time.Time) (bool, error)
	DeleteBlock(ctx context.Context, id int64) error

	// Appropriation Tables
	CreateAppropriation(ctx context.Context, a *entity.AppropriationTable) (*entity.AppropriationTable, error)
	UpdateAppropriation(ctx context.Context, a *entity.AppropriationTable) (*entity.AppropriationTable, error)
	GetDefaultAppropriation(ctx context.Context) (*entity.AppropriationTable, error)
	ListAppropriations(ctx context.Context) ([]*entity.AppropriationTable, error)
	SetDefaultAppropriation(ctx context.Context, id int64) error
}
