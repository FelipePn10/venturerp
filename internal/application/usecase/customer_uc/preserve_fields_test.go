package customer_uc

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
)

// Desktop antigo (sem form_version) não apaga os campos novos do cliente.
func TestTelaAntigaPreservaCamposNovos(t *testing.T) {
	email, ie, frete, rep := "nfe@x.com", "CONTRIBUINTE", "FOB", int64(7)
	c := &entity.Customer{NFeEmail: &email, IEIndicator: &ie, FreightType: &frete, RepresentativeCode: &rep,
		RequiresCustomerPO: true, AllowsPartialBilling: false}
	dto := request.CreateCustomerDTO{Name: "X"}
	preserveNewFields(&dto, c)
	if dto.NFeEmail == nil || *dto.NFeEmail != email || dto.IEIndicator == nil || *dto.IEIndicator != ie ||
		dto.RepresentativeCode == nil || *dto.RepresentativeCode != rep || !dto.RequiresCustomerPO ||
		dto.AllowsPartialBilling == nil || *dto.AllowsPartialBilling {
		t.Fatalf("campos novos perdidos: %+v", dto)
	}
}
