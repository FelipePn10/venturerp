package financial

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// PrevisoesDeCaixa: títulos em aberto e pedidos abertos (venda e compra) com a
// data prevista de caixa até ate. O pedido entra pelo que falta faturar/receber,
// na data de entrega mais o prazo médio da condição de pagamento. Título vencido
// entra hoje ($2).
func (r *FinancialRepositoryPG) PrevisoesDeCaixa(ctx context.Context, ate time.Time) ([]entity.PrevisaoDeCaixa, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	codigo, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	hoje := time.Now().Format("2006-01-02")
	rows, err := r.pool.Query(ctx, `
		SELECT 'RECEBER_TITULO', GREATEST(data_vencimento::date, $2::date), valor_bruto - COALESCE(valor_recebido,0) - COALESCE(desconto,0), COALESCE(numero_documento,'')
		  FROM contas_receber WHERE enterprise_id = $1 AND is_active AND UPPER(status) NOT IN ('RECEBIDO','CANCELADO')
		   AND valor_bruto - COALESCE(valor_recebido,0) - COALESCE(desconto,0) > 0
		UNION ALL
		SELECT 'PAGAR_TITULO', GREATEST(data_vencimento::date, $2::date), valor_bruto - COALESCE(valor_pago,0) - COALESCE(desconto,0) - COALESCE(valor_adiantamento_abatido,0), ''
		  FROM contas_pagar WHERE enterprise_id = $1 AND UPPER(status) NOT IN ('PAGO','CANCELADO')
		   AND valor_bruto - COALESCE(valor_pago,0) - COALESCE(desconto,0) - COALESCE(valor_adiantamento_abatido,0) > 0
		UNION ALL
		SELECT 'RECEBER_PEDIDO', GREATEST(COALESCE(l.delivery_date::date, $2::date), $2::date) + COALESCE(pc.average_term, 0),
		       COALESCE(NULLIF(l.total_net_with_ipi,0), l.total_net, 0) * (l.requested_qty - COALESCE(l.attended_qty,0) - COALESCE(l.cancelled_qty,0)) / NULLIF(l.requested_qty,0),
		       'PV ' || so.order_number
		  FROM sales_order_items l JOIN sales_orders so ON so.code = l.sales_order_code AND so.enterprise_code = $3
		  LEFT JOIN payment_conditions pc ON pc.code = so.payment_term_code AND pc.enterprise_id = $1
		 WHERE so.status IN ('P','A') AND COALESCE(l.is_active, TRUE) AND COALESCE(l.status,'') <> 'CANCELLED'
		   AND l.requested_qty - COALESCE(l.attended_qty,0) - COALESCE(l.cancelled_qty,0) > 0
		UNION ALL
		SELECT 'PAGAR_PEDIDO', GREATEST(COALESCE(l.promised_date::date, l.delivery_date::date, $2::date), $2::date) + COALESCE(pc.average_term, 0),
		       COALESCE(l.total_price, 0) * (l.requested_qty - COALESCE(l.received_qty,0) - COALESCE(l.cancelled_qty,0)) / NULLIF(l.requested_qty,0),
		       'PC ' || p.order_number
		  FROM purchase_order_items l JOIN purchase_orders p ON p.code = l.purchase_order_code AND p.enterprise_code = $3
		  LEFT JOIN payment_conditions pc ON pc.code = p.payment_term_code AND pc.enterprise_id = $1
		 WHERE p.status IN ('REQUESTED','APPROVED','PARTIAL') AND COALESCE(l.status,'') NOT IN ('CANCELLED','RECEIVED')
		   AND l.requested_qty - COALESCE(l.received_qty,0) - COALESCE(l.cancelled_qty,0) > 0`, empresa, hoje, codigo)
	if err != nil {
		return nil, fmt.Errorf("lendo as previsões de caixa: %w", err)
	}
	defer rows.Close()
	var out []entity.PrevisaoDeCaixa
	for rows.Next() {
		var p entity.PrevisaoDeCaixa
		if err := rows.Scan(&p.Origem, &p.Data, &p.Valor, &p.Documento); err != nil {
			return nil, fmt.Errorf("lendo as previsões de caixa: %w", err)
		}
		if !p.Data.After(ate) {
			out = append(out, p)
		}
	}
	return out, rows.Err()
}
