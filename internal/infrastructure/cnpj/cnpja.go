package cnpj

import (
	"context"
	"net/http"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/cnpj/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cnpj/service"
)

// cnpjaProvider adapts the CNPJá Open API (open.cnpja.com/office/{cnpj}). Its
// distinguishing feature is the `registrations` array, which carries the
// Inscrições Estaduais the metalworking ERP needs at cadastro time.
type cnpjaProvider struct {
	base string
	http *http.Client
}

type cnpjaResponse struct {
	TaxID   string `json:"taxId"`
	Alias   string `json:"alias"`
	Founded string `json:"founded"`
	Company struct {
		Name   string `json:"name"`
		Nature struct {
			Text string `json:"text"`
		} `json:"nature"`
		Size struct {
			Acronym string `json:"acronym"`
		} `json:"size"`
		Simples struct {
			Optant bool `json:"optant"`
		} `json:"simples"`
		Simei struct {
			Optant bool `json:"optant"`
		} `json:"simei"`
	} `json:"company"`
	Status struct {
		Text string `json:"text"`
	} `json:"status"`
	Address struct {
		Zip      string `json:"zip"`
		Street   string `json:"street"`
		Number   string `json:"number"`
		Details  string `json:"details"`
		District string `json:"district"`
		City     string `json:"city"`
		State    string `json:"state"`
	} `json:"address"`
	Phones []struct {
		Area   string `json:"area"`
		Number string `json:"number"`
	} `json:"phones"`
	Emails []struct {
		Address string `json:"address"`
	} `json:"emails"`
	MainActivity struct {
		ID   int64  `json:"id"`
		Text string `json:"text"`
	} `json:"mainActivity"`
	SideActivities []struct {
		ID   int64  `json:"id"`
		Text string `json:"text"`
	} `json:"sideActivities"`
	Registrations []struct {
		State   string `json:"state"`
		Number  string `json:"number"`
		Enabled bool   `json:"enabled"`
	} `json:"registrations"`
}

func (p *cnpjaProvider) Lookup(ctx context.Context, cnpj string) (*entity.Company, error) {
	digits := onlyDigits(cnpj)
	var r cnpjaResponse
	if err := doGET(ctx, p.http, p.base+"/office/"+digits, &r); err != nil {
		return nil, err
	}
	if r.TaxID == "" && r.Company.Name == "" {
		return nil, service.ErrNotFound
	}

	c := &entity.Company{
		CNPJ:               digits,
		LegalName:          strings.TrimSpace(r.Company.Name),
		TradeName:          strings.TrimSpace(r.Alias),
		RegistrationStatus: strings.ToUpper(strings.TrimSpace(r.Status.Text)),
		LegalNature:        strings.TrimSpace(r.Company.Nature.Text),
		Size:               strings.TrimSpace(r.Company.Size.Acronym),
		OpeningDate:        r.Founded,
		SimplesOptant:      r.Company.Simples.Optant,
		MEI:                r.Company.Simei.Optant,
		Source:             "cnpja",
		Address: entity.Address{
			ZipCode:      onlyDigits(r.Address.Zip),
			Street:       strings.TrimSpace(r.Address.Street),
			Number:       strings.TrimSpace(r.Address.Number),
			Complement:   strings.TrimSpace(r.Address.Details),
			Neighborhood: strings.TrimSpace(r.Address.District),
			City:         strings.TrimSpace(r.Address.City),
			UF:           strings.ToUpper(strings.TrimSpace(r.Address.State)),
		},
	}
	if len(r.Emails) > 0 {
		c.Email = strings.TrimSpace(r.Emails[0].Address)
	}
	if len(r.Phones) > 0 {
		c.Phone = strings.TrimSpace(r.Phones[0].Area + r.Phones[0].Number)
	}
	if r.MainActivity.ID != 0 {
		c.MainActivity = entity.Activity{
			Code:        formatCNAE(r.MainActivity.ID),
			Description: strings.TrimSpace(r.MainActivity.Text),
		}
	}
	for _, s := range r.SideActivities {
		c.SecondaryActivities = append(c.SecondaryActivities, entity.Activity{
			Code:        formatCNAE(s.ID),
			Description: strings.TrimSpace(s.Text),
		})
	}
	for _, reg := range r.Registrations {
		c.StateRegistrations = append(c.StateRegistrations, entity.StateRegistration{
			UF:      strings.ToUpper(strings.TrimSpace(reg.State)),
			Number:  strings.TrimSpace(reg.Number),
			Enabled: reg.Enabled,
		})
	}
	return c, nil
}
