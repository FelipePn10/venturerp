package stock_uc

import (
	"context"
	"strings"
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
		Mask:         dto.Mask,
		Lot:          dto.Lot,
		HeatNumber:   dto.HeatNumber,
		Certificate:  dto.Certificate,
		SupplierCode: dto.SupplierCode,
		Notes:        dto.Notes,
		CreatedBy:    userID,
	}
	// Data inválida era descartada em silêncio: quem digitasse errado gravava o
	// lote sem recebimento e sem aviso — e, com validade, isso tiraria o lote da
	// ordenação do FEFO sem ninguém perceber.
	if lot.ReceivedAt, err = dataDoLote(dto.ReceivedAt, "data de recebimento"); err != nil {
		return nil, err
	}
	if lot.ExpiresAt, err = dataDoLote(dto.ExpiresAt, "data de validade"); err != nil {
		return nil, err
	}
	if lot.ReceivedAt != nil && lot.ExpiresAt != nil && lot.ExpiresAt.Before(*lot.ReceivedAt) {
		return nil, errorsuc.NewValidationError("a validade do lote é anterior ao recebimento")
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

// dataDoLote converte uma data AAAA-MM-DD opcional, recusando o que não for
// data em vez de descartar.
func dataDoLote(valor *string, campo string) (*time.Time, error) {
	if valor == nil || strings.TrimSpace(*valor) == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(*valor))
	if err != nil {
		return nil, errorsuc.NewValidationError(campo + " inválida: use o formato AAAA-MM-DD")
	}
	return &t, nil
}
