package sales_order_uc

import (
	"context"
	"errors"
	"testing"

	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	custrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	repentity "github.com/FelipePn10/panossoerp/internal/domain/representative/entity"
)

type clienteComRepresentante struct {
	custrepo.CustomerRepository
	rep *int64
}

func (c clienteComRepresentante) GetCustomerByCode(_ context.Context, code int64) (*customerentity.Customer, error) {
	return &customerentity.Customer{Code: code, IsActive: true, RepresentativeCode: c.rep}, nil
}

type representantesFake map[int64]*repentity.Representative

func (r representantesFake) Get(_ context.Context, code int64) (*repentity.Representative, error) {
	if rep, ok := r[code]; ok {
		return rep, nil
	}
	return nil, errors.New("não encontrado")
}

func TestRepresentanteDoClienteViraPadraoDoPedido(t *testing.T) {
	ctx, cliente, rep := context.Background(), int64(5), int64(7)
	ativo := representantesFake{7: {Code: 7, IsActive: true}}
	if got := representanteDoCliente(ctx, clienteComRepresentante{rep: &rep}, ativo, &cliente); got == nil || *got != 7 {
		t.Fatalf("representante do cliente não veio: %v", got)
	}
	if got := representanteDoCliente(ctx, clienteComRepresentante{}, ativo, &cliente); got != nil {
		t.Fatal("cliente sem representante não devia trazer nenhum")
	}
	inativo := representantesFake{7: {Code: 7, IsActive: false}}
	if got := representanteDoCliente(ctx, clienteComRepresentante{rep: &rep}, inativo, &cliente); got != nil {
		t.Fatal("representante inativo não pode virar padrão do pedido")
	}
	if got := representanteDoCliente(ctx, clienteComRepresentante{rep: &rep}, ativo, nil); got != nil {
		t.Fatal("pedido sem cliente não tem representante padrão")
	}
}
