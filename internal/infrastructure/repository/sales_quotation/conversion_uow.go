package sales_quotation

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	orderrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	salesorderrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConversionUnitOfWork struct{ pool *pgxpool.Pool }

func NewConversionUnitOfWork(pool *pgxpool.Pool) *ConversionUnitOfWork {
	return &ConversionUnitOfWork{pool: pool}
}
func (u *ConversionUnitOfWork) Execute(ctx context.Context, quotationCode int64, fn func(orderrepo.SalesOrderRepository) (*orderentity.SalesOrder, error)) (*orderentity.SalesOrder, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	orders := salesorderrepo.NewSalesOrderRepositorySQLC(sqlc.New(tx))
	created, err := fn(orders)
	if err != nil {
		return nil, err
	}
	tag, err := tx.Exec(ctx, `UPDATE public.sales_quotations SET status='ATTENDED',attended_reason='Convertido em pedido de venda',attended_at=NOW(),converted_sales_order_code=$3,converted_at=NOW(),updated_at=NOW() WHERE code=$1 AND enterprise_code=$2 AND converted_sales_order_code IS NULL AND status NOT IN ('CANCELLED','ATTENDED','EXPIRED')`, quotationCode, tenantID, created.Code)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, fmt.Errorf("sales quotation %d not found or cannot be converted", quotationCode)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO public.sales_quotation_events(sales_quotation_code,event_type,reason) VALUES($1,'CONVERT','Convertido em pedido de venda')`, quotationCode); err != nil {
		return nil, err
	}
	var payload []byte
	err = tx.QueryRow(ctx, `SELECT jsonb_build_object(
		'empresa_id',$1,
		'pedido',jsonb_build_object('codigo',o.code,'numero',o.order_number,'status',o.status,'emissao',o.emission_date,'entrega',o.delivery_date,'pagamento_codigo',o.payment_term_code,'moeda',o.currency_code,'transportadora_codigo',o.carrier_code,'frete',o.freight_value,'desconto',o.discount_value,'acrescimo',o.surcharge_value,'total_bruto',o.total_gross,'total_liquido',o.total_net),
		'orcamento',jsonb_build_object('codigo',q.code,'numero',q.quotation_number,'revisao',1,'emissao',q.emission_date,'entrega',q.delivery_date),
		'cliente_codigo',o.customer_code,'representante_codigo',o.representative_code,'responsavel_usuario_id',o.created_by,
		'itens',COALESCE((SELECT jsonb_agg(jsonb_build_object('sequencia',i.sequence,'codigo',i.item_code::text,'descricao',COALESCE(master.description,''),'mascara',i.mask,'um',i.sales_uom,'quantidade',i.requested_qty,'preco_unitario',i.unit_price,'desconto_percentual',i.discount_pct,'ipi_percentual',i.ipi_pct,'st_percentual',i.st_pct,'total_bruto',i.total_gross,'total_liquido',i.total_net) ORDER BY i.sequence) FROM sales_order_items i LEFT JOIN items master ON master.code=i.item_code AND master.enterprise_id=$1 WHERE i.sales_order_code=o.code),'[]'::jsonb),
		'observacoes',o.notes,'link','/sales-orders/'||o.code::text)
	FROM sales_orders o JOIN sales_quotations q ON q.code=$2 AND q.enterprise_code=$1 WHERE o.code=$3 AND o.enterprise_code=$1`, tenantID, quotationCode, created.Code).Scan(&payload)
	if err != nil {
		return nil, err
	}
	var actor any
	if user, ok := ctx.Value(contextkey.UserKey).(*appsecurity.AuthUser); ok && user != nil && user.ID != "" {
		actor = user.ID
	}
	if _, err = tx.Exec(ctx, `INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key,originator_user_id) VALUES($1,'COMERCIAL_ORCAMENTO_CONVERTIDO_PEDIDO',1,'PEDIDO_VENDA',$2,$3,$4,$5,$6) ON CONFLICT(enterprise_id,event_key,deduplication_key) DO NOTHING`, tenantID, created.Code, created.OrderNumber, payload, fmt.Sprintf("orcamento:%d:pedido:%d", quotationCode, created.Code), actor); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

var _ ports.SalesQuotationConversionUnitOfWork = (*ConversionUnitOfWork)(nil)
