package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/location/entity"
)

type LocationRepository interface {
	CreateCountry(ctx context.Context, c *entity.Country) (*entity.Country, error)
	UpdateCountry(ctx context.Context, c *entity.Country) (*entity.Country, error)
	GetCountryBySigla(ctx context.Context, sigla string) (*entity.Country, error)
	ListCountries(ctx context.Context, onlyActive bool) ([]*entity.Country, error)

	CreateUF(ctx context.Context, u *entity.UF) (*entity.UF, error)
	UpdateUF(ctx context.Context, u *entity.UF) (*entity.UF, error)
	GetUFBySigla(ctx context.Context, sigla string) (*entity.UF, error)
	ListUFs(ctx context.Context, onlyActive bool) ([]*entity.UF, error)
	ListUFsByCountry(ctx context.Context, countrySigla string, onlyActive bool) ([]*entity.UF, error)
}
