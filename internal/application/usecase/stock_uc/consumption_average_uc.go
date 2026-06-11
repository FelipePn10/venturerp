package stock_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

// RecalcConsumptionAverageUseCase recomputes the average monthly consumption of
// a single item or of every item with recent outbound movements.
type RecalcConsumptionAverageUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

// RecalcResult is returned for a bulk recalculation.
type RecalcResult struct {
	UpdatedItems int `json:"updated_items"`
	WindowMonths int `json:"window_months"`
}

func (uc *RecalcConsumptionAverageUseCase) Execute(ctx context.Context, dto request.RecalcConsumptionAverageDTO) (interface{}, error) {
	if !uc.Auth.CanCreateStockMovement(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	window := dto.WindowMonths
	if window <= 0 {
		window = 6
	}
	if dto.ItemCode != nil {
		return uc.Repo.RecalcConsumptionAverage(ctx, *dto.ItemCode, window)
	}
	n, err := uc.Repo.RecalcAllConsumptionAverages(ctx, window)
	if err != nil {
		return nil, err
	}
	return RecalcResult{UpdatedItems: n, WindowMonths: window}, nil
}

// GetConsumptionAverageUseCase reads the stored average monthly consumption.
type GetConsumptionAverageUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *GetConsumptionAverageUseCase) Execute(ctx context.Context, itemCode int64) (*entity.ItemConsumptionAverage, error) {
	if !uc.Auth.CanGetStockBalance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetConsumptionAverage(ctx, itemCode)
}
