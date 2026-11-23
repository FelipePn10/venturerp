package financial_uc

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
)

func TestSemanasDoFluxoComPedidos(t *testing.T) {
	d := decimal.NewFromInt
	seg := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)
	s := SemanasDoFluxo(seg, 14, []entity.PrevisaoDeCaixa{
		{Origem: "RECEBER_TITULO", Data: seg, Valor: d(100)},
		{Origem: "PAGAR_PEDIDO", Data: seg.AddDate(0, 0, 3), Valor: d(40)},
		{Origem: "RECEBER_PEDIDO", Data: seg.AddDate(0, 0, 8), Valor: d(70)},
	})
	if len(s) != 3 || !s[0].Saldo.Equal(d(60)) || !s[1].ReceberPedidos.Equal(d(70)) || !s[1].Acumulado.Equal(d(130)) {
		t.Fatalf("%+v", s)
	}
}
