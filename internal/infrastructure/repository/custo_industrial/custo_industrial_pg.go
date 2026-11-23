// Package custo_industrial lê a variação das OFs e o custo do pedido de venda.
package custo_industrial

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/custo_industrial_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

var _ uc.Repo = (*Repo)(nil)

// Variacoes: as OFs da empresa apuradas no período.
func (r *Repo) Variacoes(ctx context.Context, de, ate time.Time) ([]uc.VariacaoDaOF, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT o.id, o.order_number, o.item_code, COALESCE(NULLIF(i.name,''), i.pdm_description_technique, ''), c.produced_qty, c.total_cost_std, c.total_cost_real,
		       c.material_variance, c.labor_variance, c.machine_variance, c.overhead_variance, c.subcontract_variance, c.total_variance,
		       c.settled_at
		  FROM production_order_costs c
		  JOIN production_orders o ON o.id = c.production_order_id AND o.enterprise_id = $1
		  LEFT JOIN items i ON i.code = o.item_code AND i.enterprise_id = o.enterprise_id
		 WHERE c.settled_at >= $2::date AND c.settled_at < $3::date + 1
		 ORDER BY c.total_variance DESC, o.id`, empresa, de.Format("2006-01-02"), ate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("lendo as variações das OFs: %w", err)
	}
	defer rows.Close()
	var out []uc.VariacaoDaOF
	for rows.Next() {
		var v uc.VariacaoDaOF
		if err := rows.Scan(&v.ProductionOrderID, &v.OrderNumber, &v.ItemCode, &v.ItemName, &v.ProducedQty, &v.Padrao, &v.Real,
			&v.Material, &v.MaoDeObra, &v.Maquina, &v.Indiretos, &v.Terceiros, &v.Total, &v.ApuradaEm); err != nil {
			return nil, fmt.Errorf("lendo as variações das OFs: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// LinhasDoPedido: cada linha do pedido (comercial: enterprise_code) com o
// custo-padrão do item (da empresa do ITEM) e o CMV do que as NF-e autorizadas
// do pedido baixaram. Sem a linha da nota ligada à do pedido, o CMV do item é
// repartido pelas linhas dele na proporção da quantidade.
func (r *Repo) LinhasDoPedido(ctx context.Context, pedido int64) ([]uc.LinhaDoPedidoCusto, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	codigo, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		WITH linhas AS (
		    SELECT l.code, l.item_code, l.requested_qty - COALESCE(l.cancelled_qty,0) qtd, COALESCE(l.total_net,0) receita
		      FROM sales_order_items l JOIN sales_orders so ON so.code = l.sales_order_code AND so.enterprise_code = $2
		     WHERE l.sales_order_code = $3 AND COALESCE(l.is_active, TRUE) AND COALESCE(l.status,'') <> 'CANCELLED'),
		cmv AS (
		    SELECT m.item_code, SUM(ABS(m.quantity)) qtd, SUM(ABS(m.total_price)) valor
		      FROM stock_movements m JOIN fiscal_exits x ON x.id = m.reference_code AND x.enterprise_id = m.enterprise_id
		     WHERE m.enterprise_id = $1 AND m.reference_type = 'NF_SAIDA' AND x.sales_order_code = $3 AND x.status = 'AUTHORIZED'
		       AND m.movement_type = 'OUT' AND m.reversed_by_movement_id IS NULL
		     GROUP BY 1),
		peso AS (SELECT item_code, SUM(qtd) qtd FROM linhas GROUP BY 1)
		SELECT l.code, l.item_code, COALESCE(NULLIF(i.name,''), i.pdm_description_technique, ''), l.qtd, l.receita,
		       COALESCE(sc.total_cost, 0), sc.total_cost IS NOT NULL,
		       COALESCE(c.qtd * l.qtd / NULLIF(p.qtd,0), 0), COALESCE(c.valor * l.qtd / NULLIF(p.qtd,0), 0)
		  FROM linhas l
		  JOIN peso p ON p.item_code = l.item_code
		  LEFT JOIN items i ON i.code = l.item_code AND i.enterprise_id = $1
		  LEFT JOIN LATERAL (SELECT s.total_cost FROM item_standard_costs s WHERE s.item_code = l.item_code AND i.code IS NOT NULL
		                      ORDER BY (s.mask = '') DESC, s.calculated_at DESC LIMIT 1) sc ON TRUE
		  LEFT JOIN cmv c ON c.item_code = l.item_code
		 ORDER BY l.code`, empresa, codigo, pedido)
	if err != nil {
		return nil, fmt.Errorf("lendo o custo do pedido: %w", err)
	}
	defer rows.Close()
	var out []uc.LinhaDoPedidoCusto
	for rows.Next() {
		var l uc.LinhaDoPedidoCusto
		if err := rows.Scan(&l.Linha, &l.ItemCode, &l.ItemName, &l.Quantidade, &l.Receita, &l.CustoPadraoUnit, &l.TemPadrao, &l.FaturadoQtd, &l.CustoReal); err != nil {
			return nil, fmt.Errorf("lendo o custo do pedido: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// ItensParaApurar: itens fabricados e comprados da empresa (engineering_type 0 e 1).
func (r *Repo) ItensParaApurar(ctx context.Context) ([]int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT code FROM items WHERE enterprise_id = $1 AND engineering_type IN (0, 1) ORDER BY code`, empresa)
	if err != nil {
		return nil, fmt.Errorf("lendo os itens a apurar: %w", err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var c int64
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
