package location_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/location/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/location/repository"
)

type LocationUseCase struct {
	Repo repository.LocationRepository
}

func New(repo repository.LocationRepository) *LocationUseCase {
	return &LocationUseCase{Repo: repo}
}

// ─── Countries ────────────────────────────────────────────────────────────────

func (uc *LocationUseCase) CreateCountry(ctx context.Context, dto request.CreateCountryDTO) (*entity.Country, error) {
	c := &entity.Country{
		Sigla:     dto.Sigla,
		Name:      dto.Name,
		DDI:       dto.DDI,
		BacenCode: dto.BacenCode,
		SisComex:  dto.SisComex,
		IsActive:  true,
	}
	return uc.Repo.CreateCountry(ctx, c)
}

func (uc *LocationUseCase) UpdateCountry(ctx context.Context, dto request.UpdateCountryDTO) (*entity.Country, error) {
	c := &entity.Country{
		ID:        dto.ID,
		Sigla:     dto.Sigla,
		Name:      dto.Name,
		DDI:       dto.DDI,
		BacenCode: dto.BacenCode,
		SisComex:  dto.SisComex,
		IsActive:  dto.IsActive,
	}
	return uc.Repo.UpdateCountry(ctx, c)
}

func (uc *LocationUseCase) GetCountryBySigla(ctx context.Context, sigla string) (*entity.Country, error) {
	return uc.Repo.GetCountryBySigla(ctx, sigla)
}

func (uc *LocationUseCase) ListCountries(ctx context.Context, onlyActive bool) ([]*entity.Country, error) {
	return uc.Repo.ListCountries(ctx, onlyActive)
}

// ─── UFs ──────────────────────────────────────────────────────────────────────

func (uc *LocationUseCase) CreateUF(ctx context.Context, dto request.CreateUFDTO) (*entity.UF, error) {
	country, err := uc.Repo.GetCountryBySigla(ctx, dto.CountrySigla)
	if err != nil {
		return nil, err
	}
	u := &entity.UF{
		Sigla:     dto.Sigla,
		Name:      dto.Name,
		CountryID: country.ID,
		IBGECode:  dto.IBGECode,
		IsActive:  true,
	}
	return uc.Repo.CreateUF(ctx, u)
}

func (uc *LocationUseCase) UpdateUF(ctx context.Context, dto request.UpdateUFDTO) (*entity.UF, error) {
	u := &entity.UF{
		ID:       dto.ID,
		Sigla:    dto.Sigla,
		Name:     dto.Name,
		IBGECode: dto.IBGECode,
		IsActive: dto.IsActive,
	}
	return uc.Repo.UpdateUF(ctx, u)
}

func (uc *LocationUseCase) GetUFBySigla(ctx context.Context, sigla string) (*entity.UF, error) {
	return uc.Repo.GetUFBySigla(ctx, sigla)
}

func (uc *LocationUseCase) ListUFs(ctx context.Context, onlyActive bool) ([]*entity.UF, error) {
	return uc.Repo.ListUFs(ctx, onlyActive)
}

func (uc *LocationUseCase) ListUFsByCountry(ctx context.Context, countrySigla string, onlyActive bool) ([]*entity.UF, error) {
	return uc.Repo.ListUFsByCountry(ctx, countrySigla, onlyActive)
}
