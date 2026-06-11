package stock_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

// RegisterLotUseCase records (or updates) the traceability metadata of a lot:
// supplier lot, heat number (corrida) and quality certificate.
type RegisterLotUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *RegisterLotUseCase) Execute(ctx context.Context, dto request.RegisterLotDTO) (*entity.StockLot, error) {
	if !uc.Auth.CanCreateStockMovement(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	lot := &entity.StockLot{
		ItemCode:     dto.ItemCode,
		Lot:          dto.Lot,
		HeatNumber:   dto.HeatNumber,
		Certificate:  dto.Certificate,
		SupplierCode: dto.SupplierCode,
		Notes:        dto.Notes,
		CreatedBy:    userID,
	}
	if dto.ReceivedAt != nil && *dto.ReceivedAt != "" {
		if t, err := time.Parse("2006-01-02", *dto.ReceivedAt); err == nil {
			lot.ReceivedAt = &t
		}
	}

	return uc.Repo.UpsertLot(ctx, lot)
}

// ListLotBalancesUseCase lists the on-hand quantity per lot of an item.
type ListLotBalancesUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *ListLotBalancesUseCase) Execute(ctx context.Context, itemCode int64) ([]*entity.StockLotBalance, error) {
	if !uc.Auth.CanGetStockBalance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListLotBalancesByItem(ctx, itemCode)
}

// GetLotGenealogyUseCase returns the full traceability of an item lot.
type GetLotGenealogyUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *GetLotGenealogyUseCase) Execute(ctx context.Context, itemCode int64, lot string) (*entity.LotGenealogy, error) {
	if !uc.Auth.CanGetStockBalance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetLotGenealogy(ctx, itemCode, lot)
}
