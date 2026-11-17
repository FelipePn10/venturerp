package customer_uc

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

// A transportadora padrão do cliente passa pela mesma conferência de carga,
// romaneio e pedido: inexistente na empresa ou inativa é recusada.
func TestTransportadoraPadraoDoClienteEhConferida(t *testing.T) {
	uc := NewCustomerUseCase(&codigoRepo{proximo: 1, usados: map[int64]bool{}})
	conferidas := []int64{}
	uc.ConferirTransportadora = func(_ context.Context, id int64) error {
		conferidas = append(conferidas, id)
		return errorsuc.NewValidationError("a transportadora está inativa")
	}
	inativa := int64(9)
	_, err := uc.CreateCustomer(context.Background(), request.CreateCustomerDTO{Name: "CLIENTE", DocumentType: "CNPJ",
		DocumentNumber: "11444777000161", CreatedBy: uuid.New(), ShippingCarrierID: &inativa})
	if err == nil || len(conferidas) != 1 || conferidas[0] != 9 {
		t.Fatalf("cliente gravado com transportadora inativa (err=%v, conferidas=%v)", err, conferidas)
	}
}
