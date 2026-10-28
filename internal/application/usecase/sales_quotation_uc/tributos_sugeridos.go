package sales_quotation_uc

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
)

// preencherTributos calcula IPI e ST da linha do orçamento pela regra de
// ICMS/IPI — o mesmo cálculo da nota — quando o vendedor não informou nenhum
// dos dois e não marcou os tributos como manuais. Orçamento sem cliente
// (prospecção) fica como digitado: sem destinatário não há como tributar.
func (uc *UseCase) preencherTributos(ctx context.Context, q *entity.SalesQuotation, itemCode int64, item *entity.SalesQuotationItem, manuais bool) error {
	if uc.Tributos == nil || manuais || q == nil || q.CustomerCode == nil || !item.IPIPct.IsZero() || !item.STPct.IsZero() {
		return nil
	}
	liquido := item.UnitPrice.Mul(decimal.NewFromInt(1).Sub(item.DiscountPct.Div(decimal.NewFromInt(100))))
	sug, err := uc.Tributos.SugerirTributos(ctx, ports.ConsultaTributosVenda{
		CustomerCode: q.CustomerCode, ItemCode: itemCode, Quantidade: item.RequestedQty, ValorUnitario: liquido,
	})
	if err != nil || sug == nil {
		return err
	}
	item.IPIPct, item.STPct = sug.IPIPct, sug.STPct
	return nil
}
