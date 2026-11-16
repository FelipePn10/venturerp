package sales_order

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// PedidoCompraRepo grava só o número do pedido de compra do cliente na capa do
// pedido de venda — o PUT do pedido regrava a capa inteira.
type PedidoCompraRepo struct{ pool *pgxpool.Pool }

func NewPedidoCompraRepo(pool *pgxpool.Pool) *PedidoCompraRepo { return &PedidoCompraRepo{pool: pool} }

// SetCustomerPO devolve quantas linhas mudaram: zero quando o pedido não é da
// empresa, não existe ou já foi faturado/cancelado.
func (r *PedidoCompraRepo) SetCustomerPO(ctx context.Context, code int64, po *string) (int64, error) {
	empresa, err := tenant.Code(ctx)
	if err != nil {
		return 0, err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE sales_orders SET customer_po_number = NULLIF(btrim($1::varchar), ''), updated_at = NOW()
		 WHERE code = $2 AND enterprise_code = $3 AND is_active AND status NOT IN ('F', 'CANCELLED')`, po, code, empresa)
	if err != nil {
		return 0, fmt.Errorf("gravando o pedido de compra do cliente: %w", err)
	}
	return tag.RowsAffected(), nil
}

// LockCustomerCredit trava a análise de crédito do cliente até o fim da
// transação em que o repositório roda (a da confirmação do pedido).
func (r *SalesOrderRepositorySQLC) LockCustomerCredit(ctx context.Context, customerCode int64) error {
	empresa, err := tenant.Code(ctx)
	if err != nil {
		return err
	}
	if err := r.q.LockCustomerCredit(ctx, sqlc.LockCustomerCreditParams{TenantEnterpriseCode: empresa, CustomerCode: customerCode}); err != nil {
		return fmt.Errorf("travando a análise de crédito do cliente %d: %w", customerCode, err)
	}
	return nil
}
