package stock_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type GetStockBalanceUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *GetStockBalanceUseCase) Execute(ctx context.Context, itemCode int64, mask string, warehouseID int64) (*response.StockBalanceResponse, error) {
	if !uc.Auth.CanGetStockBalance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	b, err := uc.Repo.GetBalance(ctx, itemCode, mask, warehouseID)
	if err != nil {
		return nil, err
	}
	return toStockBalanceResponse(b), nil
}

func (uc *GetStockBalanceUseCase) List(ctx context.Context) ([]*response.StockBalanceResponse, error) {
	if !uc.Auth.CanGetStockBalance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListBalances(ctx)
	if err != nil {
		return nil, err
	}
	return toStockBalanceResponses(list), nil
}

func (uc *GetStockBalanceUseCase) ByWarehouse(ctx context.Context, warehouseID int64) ([]*response.StockBalanceResponse, error) {
	if !uc.Auth.CanGetStockBalance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListBalancesByWarehouse(ctx, warehouseID)
	if err != nil {
		return nil, err
	}
	return toStockBalanceResponses(list), nil
}

func (uc *GetStockBalanceUseCase) ByItem(ctx context.Context, itemCode int64) ([]*response.StockBalanceResponse, error) {
	if !uc.Auth.CanGetStockBalance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListBalancesByItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	return toStockBalanceResponses(list), nil
}

// ATP returns the available-to-promise of an item across all warehouses. When
// mask is non-empty, only balances of that configuration are considered.
func (uc *GetStockBalanceUseCase) ATP(ctx context.Context, itemCode int64, mask string) (*response.ATPResponse, error) {
	if !uc.Auth.CanGetStockBalance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListBalancesByItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}

	out := &response.ATPResponse{ItemCode: itemCode, Mask: mask, Warehouses: []*response.StockBalanceResponse{}}
	for _, b := range list {
		if mask != "" && b.Mask != mask {
			continue
		}
		out.TotalOnHand += b.Quantity
		out.TotalReserved += b.ReservedQty
		out.TotalAvailable += b.AvailableQty
		out.Warehouses = append(out.Warehouses, toStockBalanceResponse(b))
	}
	return out, nil
}
