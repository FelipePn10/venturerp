package fiscal_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
)

// linhaParaNota: o que vai para a nota de uma linha do pedido.
type linhaParaNota struct {
	linha *linhaPedido
	qtd   decimal.Decimal
}

// exigenciasDoCliente aplica o que o cadastro do cliente (VCLI0500) exige no
// faturamento: sem faturamento parcial, a nota leva o saldo inteiro de todas
// as linhas; com certificado de qualidade, cada item faturado precisa de
// certificado emitido para o pedido (VINS0700).
func exigenciasDoCliente(c *customerentity.Customer, linhas []linhaPedido, faturadas []linhaParaNota, comCertificado map[int64]bool) []string {
	if c == nil {
		return nil
	}
	var out []string
	if !c.AllowsPartialBilling {
		quanto := map[int64]decimal.Decimal{}
		for _, f := range faturadas {
			quanto[f.linha.item.Code] = f.qtd
		}
		for i := range linhas {
			l := &linhas[i]
			if l.pendente.IsPositive() && !quanto[l.item.Code].Equal(l.pendente) {
				out = append(out, fmt.Sprintf("o cliente não aceita faturamento parcial: o item %s tem de sair com o saldo inteiro (%s)", codigoDaLinha(l), l.pendente.String()))
			}
		}
	}
	if c.RequiresQualityCert {
		var faltam []string
		for _, f := range faturadas {
			if f.qtd.IsPositive() && !comCertificado[f.linha.item.ItemCode] {
				faltam = append(faltam, codigoDaLinha(f.linha))
			}
		}
		if len(faltam) > 0 {
			out = append(out, "o cliente exige certificado de qualidade: emita na VINS0700, para este pedido, o certificado do(s) item(ns) "+strings.Join(semRepetir(faltam), ", "))
		}
	}
	return out
}

// exigenciasDoPedido carrega o cliente e os certificados e aplica as exigências.
func (uc *FaturarPedidoUseCase) exigenciasDoPedido(ctx context.Context, customerCode *int64, salesOrderCode int64, linhas []linhaPedido, faturadas []linhaParaNota) ([]string, error) {
	if customerCode == nil || uc.Clientes == nil {
		return nil, nil
	}
	c, err := uc.Clientes.GetCustomerByCode(ctx, *customerCode)
	if err != nil {
		return nil, err
	}
	var certificados map[int64]bool
	if c.RequiresQualityCert {
		if certificados, err = uc.Faturamento.ItensComCertificado(ctx, salesOrderCode); err != nil {
			return nil, err
		}
	}
	return exigenciasDoCliente(c, linhas, faturadas, certificados), nil
}
