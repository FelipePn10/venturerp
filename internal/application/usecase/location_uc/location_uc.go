package location_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
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

func (uc *LocationUseCase) CreateCountry(ctx context.Context, dto request.CreateCountryDTO) (*response.CountryResponse, error) {
	c := &entity.Country{
		Sigla:     dto.Sigla,
		Name:      dto.Name,
		DDI:       dto.DDI,
		BacenCode: dto.BacenCode,
		SisComex:  dto.SisComex,
		IsActive:  true,
	}
	created, err := uc.Repo.CreateCountry(ctx, c)
	if err != nil {
		return nil, err
	}
	return toCountryResponse(created), nil
}

func (uc *LocationUseCase) UpdateCountry(ctx context.Context, dto request.UpdateCountryDTO) (*response.CountryResponse, error) {
	c := &entity.Country{
		ID:        dto.ID,
		Sigla:     dto.Sigla,
		Name:      dto.Name,
		DDI:       dto.DDI,
		BacenCode: dto.BacenCode,
		SisComex:  dto.SisComex,
		IsActive:  dto.IsActive,
	}
	updated, err := uc.Repo.UpdateCountry(ctx, c)
	if err != nil {
		return nil, err
	}
	return toCountryResponse(updated), nil
}

func (uc *LocationUseCase) GetCountryBySigla(ctx context.Context, sigla string) (*response.CountryResponse, error) {
	c, err := uc.Repo.GetCountryBySigla(ctx, sigla)
	if err != nil {
		return nil, err
	}
	return toCountryResponse(c), nil
}

func (uc *LocationUseCase) ListCountries(ctx context.Context, onlyActive bool) ([]*response.CountryResponse, error) {
	list, err := uc.Repo.ListCountries(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	return toCountryResponses(list), nil
}

// ─── UFs ──────────────────────────────────────────────────────────────────────

func (uc *LocationUseCase) CreateUF(ctx context.Context, dto request.CreateUFDTO) (*response.UFResponse, error) {
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
	created, err := uc.Repo.CreateUF(ctx, u)
	if err != nil {
		return nil, err
	}
	return toUFResponse(created), nil
}

func (uc *LocationUseCase) UpdateUF(ctx context.Context, dto request.UpdateUFDTO) (*response.UFResponse, error) {
	u := &entity.UF{
		ID:       dto.ID,
		Sigla:    dto.Sigla,
		Name:     dto.Name,
		IBGECode: dto.IBGECode,
		IsActive: dto.IsActive,
	}
	updated, err := uc.Repo.UpdateUF(ctx, u)
	if err != nil {
		return nil, err
	}
	return toUFResponse(updated), nil
}

func (uc *LocationUseCase) GetUFBySigla(ctx context.Context, sigla string) (*response.UFResponse, error) {
	u, err := uc.Repo.GetUFBySigla(ctx, sigla)
	if err != nil {
		return nil, err
	}
	return toUFResponse(u), nil
}

func (uc *LocationUseCase) ListUFs(ctx context.Context, onlyActive bool) ([]*response.UFResponse, error) {
	list, err := uc.Repo.ListUFs(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	return toUFResponses(list), nil
}

func (uc *LocationUseCase) ListUFsByCountry(ctx context.Context, countrySigla string, onlyActive bool) ([]*response.UFResponse, error) {
	list, err := uc.Repo.ListUFsByCountry(ctx, countrySigla, onlyActive)
	if err != nil {
		return nil, err
	}
	return toUFResponses(list), nil
}
