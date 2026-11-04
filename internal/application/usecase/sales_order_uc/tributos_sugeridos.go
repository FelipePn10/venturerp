package sales_order_uc

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

// percentuaisDaLinha são os impostos da linha do pedido, em %.
type percentuaisDaLinha struct {
	IPI, ICMS, PIS, COFINS, ST *decimal.Decimal
}

func (p percentuaisDaLinha) zerados() bool {
	return p.IPI.IsZero() && p.ICMS.IsZero() && p.PIS.IsZero() && p.COFINS.IsZero() && p.ST.IsZero()
}

// preencherTributos calcula IPI, ICMS, PIS, COFINS e ST da linha pela regra de
// ICMS/IPI — o mesmo cálculo da nota — quando o vendedor não informou nenhum
// percentual e não marcou os tributos como manuais. Percentual digitado nunca
// é sobrescrito: é negociação, e a prévia da nota avisa a diferença.
func preencherTributos(ctx context.Context, s ports.SugestorTributosVenda, manuais bool, pedido, item int64,
	qtd, preco decimal.Decimal, descontoPct decimal.Decimal, p percentuaisDaLinha) error {
	if s == nil || manuais || !p.zerados() {
		return nil
	}
	liquido := preco.Mul(decimal.NewFromInt(1).Sub(descontoPct.Div(decimal.NewFromInt(100))))
	sug, err := s.SugerirTributos(ctx, ports.ConsultaTributosVenda{SalesOrderCode: &pedido, ItemCode: item, Quantidade: qtd, ValorUnitario: liquido})
	if err != nil || sug == nil {
		return err
	}
	*p.IPI, *p.ICMS, *p.PIS = sug.IPIPct, sug.ICMSPct, sug.PISPct
	*p.COFINS, *p.ST = sug.COFINSPct, sug.STPct
	return nil
}
