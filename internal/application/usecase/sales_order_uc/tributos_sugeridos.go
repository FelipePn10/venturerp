package sales_order_uc

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

// percentuaisDaLinha são os impostos da linha do pedido, em %.
type percentuaisDaLinha struct {
	IPI, ICMS, PIS, COFINS, ST *float64
}

func (p percentuaisDaLinha) zerados() bool {
	return *p.IPI == 0 && *p.ICMS == 0 && *p.PIS == 0 && *p.COFINS == 0 && *p.ST == 0
}

// preencherTributos calcula IPI, ICMS, PIS, COFINS e ST da linha pela regra de
// ICMS/IPI — o mesmo cálculo da nota — quando o vendedor não informou nenhum
// percentual e não marcou os tributos como manuais. Percentual digitado nunca
// é sobrescrito: é negociação, e a prévia da nota avisa a diferença.
func preencherTributos(ctx context.Context, s ports.SugestorTributosVenda, manuais bool, pedido, item int64,
	qtd, preco decimal.Decimal, descontoPct float64, p percentuaisDaLinha) error {
	if s == nil || manuais || !p.zerados() {
		return nil
	}
	liquido := preco.Mul(decimal.NewFromInt(1).Sub(decimal.NewFromFloat(descontoPct).Div(decimal.NewFromInt(100))))
	sug, err := s.SugerirTributos(ctx, ports.ConsultaTributosVenda{SalesOrderCode: &pedido, ItemCode: item, Quantidade: qtd, ValorUnitario: liquido})
	if err != nil || sug == nil {
		return err
	}
	*p.IPI, *p.ICMS, *p.PIS = sug.IPIPct.InexactFloat64(), sug.ICMSPct.InexactFloat64(), sug.PISPct.InexactFloat64()
	*p.COFINS, *p.ST = sug.COFINSPct.InexactFloat64(), sug.STPct.InexactFloat64()
	return nil
}
