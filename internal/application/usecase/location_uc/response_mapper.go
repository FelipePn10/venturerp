package location_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/location/entity"
)

func toCountryResponse(c *entity.Country) *response.CountryResponse {
	if c == nil {
		return nil
	}
	return &response.CountryResponse{
		ID:        c.ID,
		Sigla:     c.Sigla,
		Name:      c.Name,
		DDI:       c.DDI,
		BacenCode: c.BacenCode,
		SisComex:  c.SisComex,
		IsActive:  c.IsActive,
		CreatedAt: c.CreatedAt,
	}
}

func toCountryResponses(list []*entity.Country) []*response.CountryResponse {
	out := make([]*response.CountryResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toCountryResponse(c))
	}
	return out
}

func toUFResponse(u *entity.UF) *response.UFResponse {
	if u == nil {
		return nil
	}
	return &response.UFResponse{
		ID:        u.ID,
		Sigla:     u.Sigla,
		Name:      u.Name,
		CountryID: u.CountryID,
		IBGECode:  u.IBGECode,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
	}
}

func toUFResponses(list []*entity.UF) []*response.UFResponse {
	out := make([]*response.UFResponse, 0, len(list))
	for _, u := range list {
		out = append(out, toUFResponse(u))
	}
	return out
}
