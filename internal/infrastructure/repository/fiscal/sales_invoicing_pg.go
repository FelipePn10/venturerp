package fiscal

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

var _ repository.SalesOrderInvoicingRepository = (*FiscalRepositoryPG)(nil)

// NewSalesOrderInvoicingRepositoryPG expõe o repositório fiscal pelo contrato
// de faturamento do pedido.
func NewSalesOrderInvoicingRepositoryPG(r repository.FiscalRepository) repository.SalesOrderInvoicingRepository {
	return r.(*FiscalRepositoryPG)
}

func (r *FiscalRepositoryPG) QuantidadesEmNota(ctx context.Context, salesOrderCode int64) (map[int64]decimal.Decimal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT i.sales_order_item_code, SUM(i.quantity)
		   FROM fiscal_exit_items i
		   JOIN fiscal_exits e ON e.id = i.fiscal_exit_id
		  WHERE e.enterprise_id = $1 AND e.sales_order_code = $2 AND e.is_active
		    AND e.status IN ('DRAFT','AGUARDANDO_AUTORIZACAO')
		    AND i.sales_order_item_code IS NOT NULL
		  GROUP BY i.sales_order_item_code`, empresa, salesOrderCode)
	if err != nil {
		return nil, fmt.Errorf("somando quantidades do pedido em nota: %w", err)
	}
	defer rows.Close()
	out := map[int64]decimal.Decimal{}
	for rows.Next() {
		var code int64
		var qtd decimal.Decimal
		if err := rows.Scan(&code, &qtd); err != nil {
			return nil, err
		}
		out[code] = qtd
	}
	return out, rows.Err()
}

func (r *FiscalRepositoryPG) RegistrarFaturamento(ctx context.Context, fiscalExitID int64, estorno bool) (bool, bool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return false, false, err
	}
	// Pedido de venda usa a convenção enterprise_code (comercial).
	empresaCode, err := tenant.Code(ctx)
	if err != nil {
		return false, false, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var pedido *int64
	if err := tx.QueryRow(ctx, `SELECT sales_order_code FROM fiscal_exits WHERE id=$1 AND enterprise_id=$2`, fiscalExitID, empresa).Scan(&pedido); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, nil
		}
		return false, false, err
	}
	if pedido == nil {
		return false, false, nil
	}
	sinal := 1
	if estorno {
		sinal = -1
	}
	tag, err := tx.Exec(ctx,
		`WITH q AS (
		    SELECT i.sales_order_item_code AS code, SUM(i.quantity) AS qtd
		      FROM fiscal_exit_items i
		     WHERE i.fiscal_exit_id = $1 AND i.sales_order_item_code IS NOT NULL
		     GROUP BY i.sales_order_item_code)
		 UPDATE sales_order_items soi
		    SET attended_qty = GREATEST(soi.attended_qty + $4 * q.qtd, 0),
		        status = CASE
		            WHEN soi.status = 'CANCELLED' THEN soi.status
		            WHEN GREATEST(soi.attended_qty + $4 * q.qtd, 0) + soi.cancelled_qty >= soi.requested_qty THEN 'DELIVERED'
		            WHEN GREATEST(soi.attended_qty + $4 * q.qtd, 0) > 0 THEN 'PARTIAL'
		            ELSE 'OPEN' END,
		        updated_at = NOW()
		   FROM q, sales_orders so
		  WHERE soi.code = q.code AND soi.sales_order_code = $2
		    AND so.code = soi.sales_order_code AND so.enterprise_code = $3`,
		fiscalExitID, *pedido, empresaCode, sinal)
	if err != nil {
		return false, false, fmt.Errorf("registrando faturamento no pedido %d: %w", *pedido, err)
	}
	if tag.RowsAffected() == 0 {
		return false, false, tx.Commit(ctx)
	}
	var faltando bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM sales_order_items
		                 WHERE sales_order_code = $1 AND is_active AND status <> 'CANCELLED'
		                   AND attended_qty + cancelled_qty < requested_qty)`, *pedido).Scan(&faltando); err != nil {
		return false, false, err
	}
	return true, !faltando, tx.Commit(ctx)
}

func (r *FiscalRepositoryPG) NomeRepresentante(ctx context.Context, code int64) (string, error) {
	var n string
	err := r.pool.QueryRow(ctx, `SELECT name FROM representatives WHERE code=$1`, code).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return n, err
}

func (r *FiscalRepositoryPG) NotasDoPedido(ctx context.Context, salesOrderCode int64) ([]repository.NotaResumo, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, numero_nf, status, valor_total FROM fiscal_exits
		  WHERE enterprise_id=$1 AND sales_order_code=$2 AND is_active ORDER BY id`, empresa, salesOrderCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []repository.NotaResumo
	for rows.Next() {
		var n repository.NotaResumo
		if err := rows.Scan(&n.ID, &n.NumeroNF, &n.Status, &n.ValorTotal); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// TravarPedido usa um advisory lock de sessão — a nota é criada em transação
// própria, então o bloqueio precisa durar além dela — numa conexão reservada
// até a liberação. A tentativa NÃO espera: quem chega com o pedido já em
// faturamento recebe conflito na hora. Esperar segurando a conexão esgota o
// pool quando vários faturam o mesmo pedido, e o dono do bloqueio deixa de
// conseguir conexão para terminar (o teste concorrente travava assim). Se o
// desbloqueio falhar, a conexão é fechada: devolver ao pool uma sessão que
// ainda segura o lock travaria o pedido indefinidamente.
func (r *FiscalRepositoryPG) TravarPedido(ctx context.Context, salesOrderCode int64) (func(), error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("reservando conexão para o faturamento: %w", err)
	}
	chave := fmt.Sprintf("faturamento-pedido:%d:%d", empresa, salesOrderCode)
	var obtido bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended($1, 0))`, chave).Scan(&obtido); err != nil {
		conn.Release()
		return nil, fmt.Errorf("bloqueando o pedido %d para faturar: %w", salesOrderCode, err)
	}
	if !obtido {
		conn.Release()
		return nil, errorsuc.NewConflictError(fmt.Sprintf("o pedido %d está sendo faturado por outro usuário neste momento; aguarde e confira a prévia antes de tentar de novo", salesOrderCode))
	}
	liberar := func() {
		fim := context.WithoutCancel(ctx)
		if _, err := conn.Exec(fim, `SELECT pg_advisory_unlock(hashtextextended($1, 0))`, chave); err != nil {
			_ = conn.Conn().Close(fim)
		}
		conn.Release()
	}
	return liberar, nil
}
