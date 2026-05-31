package stock_movement_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/domain/stock_movement/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock_movement/repository"
)

type StockMovementTypeUseCase struct {
	Repo repository.StockMovementTypeRepository
}

func New(repo repository.StockMovementTypeRepository) *StockMovementTypeUseCase {
	return &StockMovementTypeUseCase{Repo: repo}
}

func (uc *StockMovementTypeUseCase) Create(ctx context.Context, s *entity.StockMovementType) (*entity.StockMovementType, error) {
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
	return uc.Repo.Create(ctx, s)
}

func (uc *StockMovementTypeUseCase) Update(ctx context.Context, s *entity.StockMovementType) (*entity.StockMovementType, error) {
	if s.ID == 0 {
		return nil, errors.New("id is required")
	}
	return uc.Repo.Update(ctx, s)
}

func (uc *StockMovementTypeUseCase) GetByID(ctx context.Context, id int64) (*entity.StockMovementType, error) {
	return uc.Repo.GetByID(ctx, id)
}

func (uc *StockMovementTypeUseCase) GetBySigla(ctx context.Context, sigla string) (*entity.StockMovementType, error) {
	return uc.Repo.GetBySigla(ctx, sigla)
}

func (uc *StockMovementTypeUseCase) List(ctx context.Context, onlyActive bool) ([]*entity.StockMovementType, error) {
	return uc.Repo.List(ctx, onlyActive)
}
