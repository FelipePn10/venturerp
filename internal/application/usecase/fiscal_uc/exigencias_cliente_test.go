package fiscal_uc

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	salesentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
)

func duasLinhasDoPedido() []linhaPedido {
	return []linhaPedido{
		{item: &salesentity.SalesOrderItem{Code: 1, ItemCode: 100}, pendente: decimal.NewFromInt(10)},
		{item: &salesentity.SalesOrderItem{Code: 2, ItemCode: 200}, pendente: decimal.NewFromInt(5)},
	}
}

func TestExigenciasDoClienteNoFaturamento(t *testing.T) {
	linhas := duasLinhasDoPedido()
	tudo := []linhaParaNota{{&linhas[0], decimal.NewFromInt(10)}, {&linhas[1], decimal.NewFromInt(5)}}
	parcial := []linhaParaNota{{&linhas[0], decimal.NewFromInt(4)}}

	aceita := &customerentity.Customer{AllowsPartialBilling: true}
	if e := exigenciasDoCliente(aceita, linhas, parcial, nil); len(e) != 0 {
		t.Fatalf("cliente que aceita parcial: %v", e)
	}
	naoAceita := &customerentity.Customer{AllowsPartialBilling: false}
	if e := exigenciasDoCliente(naoAceita, linhas, parcial, nil); len(e) != 2 {
		t.Fatalf("parcial devia acusar as duas linhas incompletas: %v", e)
	}
	if e := exigenciasDoCliente(naoAceita, linhas, tudo, nil); len(e) != 0 {
		t.Fatalf("saldo inteiro passa: %v", e)
	}

	certificado := &customerentity.Customer{AllowsPartialBilling: true, RequiresQualityCert: true}
	e := exigenciasDoCliente(certificado, linhas, tudo, map[int64]bool{100: true})
	if len(e) != 1 || !strings.Contains(e[0], "VINS0700") {
		t.Fatalf("falta o certificado do item 200: %v", e)
	}
	if e := exigenciasDoCliente(certificado, linhas, tudo, map[int64]bool{100: true, 200: true}); len(e) != 0 {
		t.Fatalf("com os dois certificados passa: %v", e)
	}
	if e := exigenciasDoCliente(nil, linhas, parcial, nil); e != nil {
		t.Fatal("sem cliente não há exigência")
	}
}
