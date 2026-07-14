// Package cnpj_uc orchestrates CNPJ registry lookups: it validates the input
// and delegates to a provider port, keeping HTTP/provider concerns out of the
// interface layer.
package cnpj_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/cnpj/entity"
	cnpjsvc "github.com/FelipePn10/panossoerp/internal/domain/cnpj/service"
	"github.com/FelipePn10/panossoerp/internal/pkg/validation"
)

// LookupCNPJUseCase resolves company data for a CNPJ.
type LookupCNPJUseCase struct {
	provider cnpjsvc.Provider
}

func NewLookupCNPJUseCase(provider cnpjsvc.Provider) *LookupCNPJUseCase {
	return &LookupCNPJUseCase{provider: provider}
}

// ErrInvalidCNPJ is returned when the supplied document fails check-digit
// validation, before any external call is made.
var ErrInvalidCNPJ = fmt.Errorf("CNPJ inválido")

// Execute validates the CNPJ and returns the registry data as a response DTO.
// The domain errors (service.ErrNotFound / service.ErrUnavailable) are passed
// through so the handler can map them to the right HTTP status.
func (uc *LookupCNPJUseCase) Execute(ctx context.Context, cnpj string) (*response.CNPJLookupResponse, error) {
	if !validation.ValidateCNPJ(cnpj) {
		return nil, ErrInvalidCNPJ
	}
	company, err := uc.provider.Lookup(ctx, cnpj)
	if err != nil {
		return nil, err
	}
	return toResponse(company), nil
}

func toResponse(c *entity.Company) *response.CNPJLookupResponse {
	out := &response.CNPJLookupResponse{
		CNPJ:               c.CNPJ,
		LegalName:          c.LegalName,
		TradeName:          c.TradeName,
		RegistrationStatus: c.RegistrationStatus,
		LegalNature:        c.LegalNature,
		Size:               c.Size,
		OpeningDate:        c.OpeningDate,
		Email:              c.Email,
		Phone:              c.Phone,
		SimplesOptant:      c.SimplesOptant,
		MEI:                c.MEI,
		StateRegistration:  c.PrimaryStateRegistration(),
		Source:             c.Source,
		Address: response.CNPJAddressResponse{
			ZipCode:      c.Address.ZipCode,
			Street:       c.Address.Street,
			Number:       c.Address.Number,
			Complement:   c.Address.Complement,
			Neighborhood: c.Address.Neighborhood,
			City:         c.Address.City,
			UF:           c.Address.UF,
		},
		MainActivity: response.CNPJActivityResponse{
			Code:        c.MainActivity.Code,
			Description: c.MainActivity.Description,
		},
	}
	for _, r := range c.StateRegistrations {
		out.StateRegistrations = append(out.StateRegistrations, response.StateRegistrationResponse{
			UF:      r.UF,
			Number:  r.Number,
			Enabled: r.Enabled,
		})
	}
	for _, a := range c.SecondaryActivities {
		out.SecondaryActivity = append(out.SecondaryActivity, response.CNPJActivityResponse{
			Code:        a.Code,
			Description: a.Description,
		})
	}
	return out
}
