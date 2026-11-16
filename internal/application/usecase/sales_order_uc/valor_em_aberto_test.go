package sales_order_uc

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
)

// Pedido faturado em parte: a exposição é só o saldo (o faturado já é título).
func TestValorEmAbertoDoPedido(t *testing.T) {
	d := decimal.RequireFromString
	capa := &entity.SalesOrder{TotalWithIPIWithST: d("3300")}
	linhas := []*entity.SalesOrderItem{
		// 10 pedidas, 4 faturadas, 1 cancelada: 5/10 de 2.200 = 1.100
		{IsActive: true, RequestedQty: d("10"), AttendedQty: d("4"), CancelledQty: d("1"), TotalNetWithIPI: d("2000"), TotalST: d("200")},
		// faturada por inteiro: nada em aberto
		{IsActive: true, RequestedQty: d("5"), AttendedQty: d("5"), TotalNetWithIPI: d("1100")},
		// linha cancelada não conta
		{IsActive: false, RequestedQty: d("3"), TotalNetWithIPI: d("900")},
	}
	if v := valorEmAberto(capa, linhas); !v.Equal(d("1100")) {
		t.Fatalf("em aberto %s, esperado 1100", v)
	}
	if v := valorEmAberto(capa, nil); !v.Equal(d("3300")) {
		t.Fatalf("sem linhas vale a capa: %s", v)
	}
}
