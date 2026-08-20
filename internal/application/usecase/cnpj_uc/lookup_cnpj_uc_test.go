package cnpj_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/cnpj/entity"
)

type completeCNPJProvider struct{}

func (completeCNPJProvider) Lookup(context.Context, string) (*entity.Company, error) {
	return &entity.Company{
		CNPJ:               "52454668000102",
		LegalName:          "TECNOFER FABRICACAO E MONTAGEM DE ESTRUTURAS METALICAS LTDA",
		TradeName:          "Tecnofer",
		RegistrationStatus: "ATIVA",
		LegalNature:        "Sociedade Empresária Limitada",
		Size:               "ME",
		OpeningDate:        "2023-10-06",
		Email:              "comercial@tecnofer.com.br",
		Phone:              "44999999999",
		Address: entity.Address{
			ZipCode:      "86975000",
			Street:       "Rua Jacob Evaldo Stadler",
			Number:       "83",
			Neighborhood: "Parque Industrial III",
			City:         "Mandaguari",
			UF:           "PR",
		},
		MainActivity: entity.Activity{Code: "2511000", Description: "Fabricação de estruturas metálicas"},
		StateRegistrations: []entity.StateRegistration{
			{UF: "SP", Number: "IE-SP", Enabled: true},
			{UF: "PR", Number: "IE-PR", Enabled: true},
		},
	}, nil
}

func TestLookupReturnsAllDataRequiredForCustomerAndSupplierAutofill(t *testing.T) {
	result, err := NewLookupCNPJUseCase(completeCNPJProvider{}).Execute(context.Background(), "52.454.668/0001-02")
	if err != nil {
		t.Fatal(err)
	}

	if result.LegalName == "" || result.TradeName == "" || result.StateRegistration != "IE-PR" {
		t.Fatalf("identificação incompleta: %+v", result)
	}
	if result.Address.ZipCode == "" || result.Address.Street == "" || result.Address.Number == "" || result.Address.Neighborhood == "" || result.Address.City == "" || result.Address.UF == "" {
		t.Fatalf("endereço incompleto: %+v", result.Address)
	}
	if result.Email == "" || result.Phone == "" || result.MainActivity.Code == "" || result.RegistrationStatus == "" || result.LegalNature == "" || result.Size == "" || result.OpeningDate == "" {
		t.Fatalf("enriquecimento incompleto: %+v", result)
	}
}
