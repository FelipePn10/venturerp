package location

import (
	"context"

	locationEntity "github.com/FelipePn10/panossoerp/internal/domain/location/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/location/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

type LocationRepositorySQLC struct {
	q *sqlc.Queries
}

var _ domainrepo.LocationRepository = (*LocationRepositorySQLC)(nil)

func New(q *sqlc.Queries) *LocationRepositorySQLC {
	return &LocationRepositorySQLC{q: q}
}

// ─── Countries ────────────────────────────────────────────────────────────────

func (r *LocationRepositorySQLC) CreateCountry(ctx context.Context, c *locationEntity.Country) (*locationEntity.Country, error) {
	row, err := r.q.CreateCountry(ctx, sqlc.CreateCountryParams{
		Sigla:     c.Sigla,
		Name:      c.Name,
		Ddi:       pgutil.ToPgTextFromPtr(c.DDI),
		BacenCode: pgutil.ToPgTextFromPtr(c.BacenCode),
		SisComex:  pgutil.ToPgTextFromPtr(c.SisComex),
	})
	if err != nil {
		return nil, err
	}
	return countryToEntity(row), nil
}

func (r *LocationRepositorySQLC) UpdateCountry(ctx context.Context, c *locationEntity.Country) (*locationEntity.Country, error) {
	row, err := r.q.UpdateCountry(ctx, sqlc.UpdateCountryParams{
		ID:        c.ID,
		Name:      c.Name,
		Ddi:       pgutil.ToPgTextFromPtr(c.DDI),
		BacenCode: pgutil.ToPgTextFromPtr(c.BacenCode),
		SisComex:  pgutil.ToPgTextFromPtr(c.SisComex),
		IsActive:  c.IsActive,
	})
	if err != nil {
		return nil, err
	}
	return countryToEntity(row), nil
}

func (r *LocationRepositorySQLC) GetCountryBySigla(ctx context.Context, sigla string) (*locationEntity.Country, error) {
	row, err := r.q.GetCountryBySigla(ctx, sigla)
	if err != nil {
		return nil, err
	}
	return countryToEntity(row), nil
}

func (r *LocationRepositorySQLC) ListCountries(ctx context.Context, onlyActive bool) ([]*locationEntity.Country, error) {
	rows, err := r.q.ListCountries(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	result := make([]*locationEntity.Country, len(rows))
	for i, row := range rows {
		result[i] = countryToEntity(row)
	}
	return result, nil
}

func countryToEntity(row sqlc.Country) *locationEntity.Country {
	return &locationEntity.Country{
		ID:        row.ID,
		Sigla:     row.Sigla,
		Name:      row.Name,
		DDI:       pgutil.FromPgTextPtr(row.Ddi),
		BacenCode: pgutil.FromPgTextPtr(row.BacenCode),
		SisComex:  pgutil.FromPgTextPtr(row.SisComex),
		IsActive:  row.IsActive,
		CreatedAt: pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── UFs ──────────────────────────────────────────────────────────────────────

func (r *LocationRepositorySQLC) CreateUF(ctx context.Context, u *locationEntity.UF) (*locationEntity.UF, error) {
	row, err := r.q.CreateUF(ctx, sqlc.CreateUFParams{
		Sigla:     u.Sigla,
		Name:      u.Name,
		CountryID: u.CountryID,
		IbgeCode:  pgutil.ToPgTextFromPtr(u.IBGECode),
	})
	if err != nil {
		return nil, err
	}
	return ufToEntity(row), nil
}

func (r *LocationRepositorySQLC) UpdateUF(ctx context.Context, u *locationEntity.UF) (*locationEntity.UF, error) {
	row, err := r.q.UpdateUF(ctx, sqlc.UpdateUFParams{
		ID:       u.ID,
		Name:     u.Name,
		IbgeCode: pgutil.ToPgTextFromPtr(u.IBGECode),
		IsActive: u.IsActive,
	})
	if err != nil {
		return nil, err
	}
	return ufToEntity(row), nil
}

func (r *LocationRepositorySQLC) GetUFBySigla(ctx context.Context, sigla string) (*locationEntity.UF, error) {
	row, err := r.q.GetUFBySigla(ctx, sigla)
	if err != nil {
		return nil, err
	}
	return ufToEntity(row), nil
}

func (r *LocationRepositorySQLC) ListUFs(ctx context.Context, onlyActive bool) ([]*locationEntity.UF, error) {
	rows, err := r.q.ListUFs(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	result := make([]*locationEntity.UF, len(rows))
	for i, row := range rows {
		result[i] = ufToEntity(row)
	}
	return result, nil
}

func (r *LocationRepositorySQLC) ListUFsByCountry(ctx context.Context, countrySigla string, onlyActive bool) ([]*locationEntity.UF, error) {
	rows, err := r.q.ListUFsByCountry(ctx, sqlc.ListUFsByCountryParams{
		Sigla:   countrySigla,
		Column2: onlyActive,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*locationEntity.UF, len(rows))
	for i, row := range rows {
		result[i] = ufToEntity(row)
	}
	return result, nil
}

func ufToEntity(row sqlc.Uf) *locationEntity.UF {
	return &locationEntity.UF{
		ID:        row.ID,
		Sigla:     row.Sigla,
		Name:      row.Name,
		CountryID: row.CountryID,
		IBGECode:  pgutil.FromPgTextPtr(row.IbgeCode),
		IsActive:  row.IsActive,
		CreatedAt: pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}
