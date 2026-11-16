package sales_order_uc

import (
	"context"
	"fmt"
	"strings"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/representativevalidation"
	custrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
)

// exigirClienteAtivo: cliente inativado na VCLI0500 não recebe pedido novo.
// Bloqueado pode (o crédito o segura na confirmação); inativo não.
func exigirClienteAtivo(ctx context.Context, customers custrepo.CustomerRepository, code *int64) error {
	if customers == nil || code == nil || *code == 0 {
		return nil
	}
	c, err := customers.GetCustomerByCode(ctx, *code)
	if err != nil {
		return err
	}
	if !c.IsActive {
		return errorsuc.NewValidationError(fmt.Sprintf("o cliente %d está inativo; reative-o na VCLI0500 para vender para ele", *code))
	}
	return nil
}

// representanteDoCliente: o representante do cadastro do cliente, se ainda
// puder vender (o vínculo com a empresa vale); senão nenhum.
func representanteDoCliente(ctx context.Context, customers custrepo.CustomerRepository, reps representativevalidation.Repository, code *int64) *int64 {
	if customers == nil || code == nil || *code == 0 {
		return nil
	}
	c, err := customers.GetCustomerByCode(ctx, *code)
	if err != nil || c.RepresentativeCode == nil || *c.RepresentativeCode == 0 {
		return nil
	}
	if representativevalidation.Validate(ctx, reps, c.RepresentativeCode) != nil {
		return nil
	}
	return c.RepresentativeCode
}

// validarPedidoDeCompra: o xPed do item da NF-e aceita até 15 caracteres.
func validarPedidoDeCompra(po *string) error {
	if po != nil && len([]rune(strings.TrimSpace(*po))) > 15 {
		return errorsuc.NewValidationError("o número do pedido de compra do cliente tem no máximo 15 caracteres (limite da NF-e)")
	}
	return nil
}

// exigirPedidoDeCompra: cliente que exige o número do pedido de compra
// (VCLI0500) não tem pedido confirmado sem ele — a nota sairia sem o xPed e
// voltaria recusada.
func exigirPedidoDeCompra(ctx context.Context, customers custrepo.CustomerRepository, customerCode *int64, po *string) error {
	if customers == nil || customerCode == nil || *customerCode == 0 {
		return nil
	}
	c, err := customers.GetCustomerByCode(ctx, *customerCode)
	if err != nil {
		return err
	}
	if c.RequiresCustomerPO && (po == nil || strings.TrimSpace(*po) == "") {
		return errorsuc.NewValidationError(fmt.Sprintf("o cliente %d exige o número do pedido de compra dele; informe-o no pedido antes de confirmar", *customerCode))
	}
	return nil
}
