package stock_movement_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/stock_movement/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock_movement/repository"
)

type StockMovementTypeUseCase struct {
	Repo repository.StockMovementTypeRepository
}

func New(repo repository.StockMovementTypeRepository) *StockMovementTypeUseCase {
	return &StockMovementTypeUseCase{Repo: repo}
}

func (uc *StockMovementTypeUseCase) Create(ctx context.Context, s *entity.StockMovementType) (*response.StockMovementTypeResponse, error) {
	if s.Sigla == "" {
		return nil, errors.New("sigla is required")
	}
	if s.Description == "" {
		return nil, errors.New("description is required")
	}
	if s.UsageType == "" {
		s.UsageType = entity.UsageGeral
	}
	if s.EntryExit == "" {
		s.EntryExit = entity.DirAmbos
	}
	s.ShowsInSummary = true
	s.IsActive = true
	created, err := uc.Repo.Create(ctx, s)
	if err != nil {
		return nil, err
	}
	return toStockMovementTypeResponse(created), nil
}

func (uc *StockMovementTypeUseCase) Update(ctx context.Context, s *entity.StockMovementType) (*response.StockMovementTypeResponse, error) {
	if s.ID == 0 {
		return nil, errors.New("id is required")
	}
	updated, err := uc.Repo.Update(ctx, s)
	if err != nil {
		return nil, err
	}
	return toStockMovementTypeResponse(updated), nil
}

func (uc *StockMovementTypeUseCase) GetByID(ctx context.Context, id int64) (*response.StockMovementTypeResponse, error) {
	s, err := uc.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toStockMovementTypeResponse(s), nil
}

func (uc *StockMovementTypeUseCase) GetBySigla(ctx context.Context, sigla string) (*response.StockMovementTypeResponse, error) {
	s, err := uc.Repo.GetBySigla(ctx, sigla)
	if err != nil {
		return nil, err
	}
	return toStockMovementTypeResponse(s), nil
}

func (uc *StockMovementTypeUseCase) List(ctx context.Context, onlyActive bool) ([]*response.StockMovementTypeResponse, error) {
	list, err := uc.Repo.List(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	return toStockMovementTypeResponses(list), nil
}
