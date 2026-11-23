// Package indicadores lê do banco os indicadores prontos (melhoria 12).
package indicadores

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/indicadores_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

var _ uc.Repo = (*Repo)(nil)

func dia(t time.Time) string { return t.Format("2006-01-02") }

// vendasSQL: linhas de venda das NF-e autorizadas do período ($1 empresa, $2 de, $3 até),
// sem remessa/retorno de industrialização, com o CMV que a nota baixou do item.
const vendasSQL = `
WITH notas AS (
    SELECT id, customer_code FROM fiscal_exits
     WHERE enterprise_id = $1 AND status = 'AUTHORIZED' AND data_emissao BETWEEN $2::date AND $3::date),
receita AS (
    SELECT n.id nota, n.customer_code, i.item_code, SUM(i.total_price) valor
      FROM fiscal_exit_items i JOIN notas n ON n.id = i.fiscal_exit_id
     WHERE i.item_code IS NOT NULL AND substr(replace(i.cfop,'.',''), 2) NOT IN ('901','902','903')
     GROUP BY 1, 2, 3),
cmv AS (
    SELECT m.reference_code nota, m.item_code, SUM(ABS(m.total_price)) valor
      FROM stock_movements m JOIN notas n ON n.id = m.reference_code
     WHERE m.enterprise_id = $1 AND m.reference_type = 'NF_SAIDA' AND m.movement_type = 'OUT' AND m.reversed_by_movement_id IS NULL
     GROUP BY 1, 2)
SELECT r.customer_code, r.item_code, r.valor receita, COALESCE(c.valor, 0) cmv
  FROM receita r LEFT JOIN cmv c ON c.nota = r.nota AND c.item_code = r.item_code`

// Vendas: receita e CMV das notas de venda autorizadas do período.
func (r *Repo) Vendas(ctx context.Context, de, ate time.Time) (*uc.Margem, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	m := &uc.Margem{Nome: "Vendas do período"}
	if err := r.pool.QueryRow(ctx, `WITH v AS (`+vendasSQL+`)
		SELECT COALESCE(SUM(v.receita), 0), COALESCE(SUM(v.cmv), 0) FROM v`, empresa, dia(de), dia(ate)).Scan(&m.Receita, &m.CMV); err != nil {
		return nil, fmt.Errorf("lendo as vendas do período: %w", err)
	}
	return m, nil
}

// OEE: horas apontadas (fim − início; o turno que vira a noite soma 24 h) sobre
// as horas disponíveis (horas/dia da máquina × dias úteis do período); desempenho
// pela hora padrão da etapa apontada.
func (r *Repo) OEE(ctx context.Context, de, ate time.Time) ([]uc.OEE, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		WITH uteis AS (SELECT COUNT(*) n FROM generate_series($2::date, $3::date, INTERVAL '1 day') d WHERE EXTRACT(ISODOW FROM d) < 6)
		SELECT m.id, m.code, COALESCE(m.name,''),
		       COALESCE(SUM(CASE WHEN a.start_time IS NULL OR a.end_time IS NULL THEN 0
		                         ELSE EXTRACT(EPOCH FROM (a.end_time - a.start_time + CASE WHEN a.end_time < a.start_time THEN INTERVAL '24 hours' ELSE INTERVAL '0' END)) / 3600 END), 0)::numeric(18,4),
		       COALESCE(m.available_hours_per_day, 8) * (SELECT n FROM uteis),
		       COALESCE(SUM(a.produced_qty), 0), COALESCE(SUM(a.scrapped_qty), 0),
		       -- horas padrão do que saiu: (boas + refugo) × horas previstas da etapa ÷ quantidade da OF
		       SUM((a.produced_qty + a.scrapped_qty) * op.planned_hours / NULLIF(o.planned_qty, 0))
		         FILTER (WHERE op.planned_hours > 0 AND a.start_time IS NOT NULL AND a.end_time IS NOT NULL)::numeric(18,4)
		  FROM machines m
		  JOIN production_appointments a ON a.machine_id = m.id AND a.appointment_date BETWEEN $2::date AND $3::date
		  JOIN production_orders o ON o.id = a.production_order_id AND o.enterprise_id = $1
		  LEFT JOIN production_order_operations op ON op.id = a.operation_id AND op.enterprise_id = $1
		 WHERE m.enterprise_id = $1
		 GROUP BY m.id, m.code, m.name, m.available_hours_per_day
		 ORDER BY m.code`, empresa, dia(de), dia(ate))
	if err != nil {
		return nil, fmt.Errorf("lendo o OEE: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (uc.OEE, error) {
		var o uc.OEE
		var padrao *decimal.Decimal
		err := row.Scan(&o.MachineID, &o.Codigo, &o.Nome, &o.HorasApontadas, &o.HorasDisponiveis, &o.Boas, &o.Refugo, &padrao)
		// Desempenho = horas padrão ÷ horas apontadas (só com a etapa e o tempo previsto).
		if padrao != nil && padrao.IsPositive() && o.HorasApontadas.IsPositive() {
			d := decimal.Min(uc.Pct(*padrao, o.HorasApontadas), decimal.NewFromInt(100))
			o.Desempenho = &d
		}
		return o, err
	})
}

// Giro: CMV do período (saídas de venda, consumo e requisição) sobre o valor do
// estoque de hoje nos almoxarifados próprios.
func (r *Repo) Giro(ctx context.Context, de, ate time.Time) (*uc.Giro, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var g uc.Giro
	if err := r.pool.QueryRow(ctx, `
		SELECT COALESCE((SELECT SUM(ABS(m.total_price)) FROM stock_movements m
		                  WHERE m.enterprise_id = $1 AND m.movement_date BETWEEN $2::date AND $3::date
		                    AND m.movement_type IN ('OUT','REP') AND m.reversed_by_movement_id IS NULL
		                    AND COALESCE(m.reference_type,'') NOT IN ('INVENTORY','CONTAGEM_CICLICA','TRANSFERENCIA','TRANSITO','ESTORNO','OS_TERCEIRO')), 0),
		       COALESCE((SELECT SUM(b.total_cost) FROM stock_balances b JOIN warehouse w ON w.id = b.warehouse_id AND w.enterprise_id = b.enterprise_id
		                  WHERE b.enterprise_id = $1 AND b.quantity > 0 AND w.supplier_code IS NULL), 0)`,
		empresa, dia(de), dia(ate)).Scan(&g.CMV, &g.ValorEstoque); err != nil {
		return nil, fmt.Errorf("lendo o giro do estoque: %w", err)
	}
	dias := decimal.NewFromInt(int64(ate.Sub(de).Hours()/24) + 1)
	if g.ValorEstoque.IsPositive() {
		g.GiroAnualizado = g.CMV.Mul(decimal.NewFromInt(365)).Div(dias).Div(g.ValorEstoque).Round(2)
	}
	if g.CMV.IsPositive() {
		g.CoberturaDias = g.ValorEstoque.Div(g.CMV.Div(dias)).Round(1)
	}
	return &g, nil
}

// PontualidadeFornecedores: linhas de pedido de compra recebidas no período
// (nota de entrada aprovada ligada à linha) contra a data prometida.
func (r *Repo) PontualidadeFornecedores(ctx context.Context, de, ate time.Time) ([]uc.Pontualidade, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		WITH linhas AS (
		    SELECT e.supplier_code, e.data_entrada::date chegou, COALESCE(p.promised_date, p.delivery_date)::date prometido
		      FROM fiscal_entry_items i
		      JOIN fiscal_entries e ON e.id = i.fiscal_entry_id AND e.enterprise_id = $1
		      JOIN purchase_order_items p ON p.code = i.purchase_order_item_code
		     WHERE e.status IN ('APPROVED','WRITTEN_OFF') AND e.data_entrada BETWEEN $2::date AND $3::date AND e.supplier_code IS NOT NULL
		       AND COALESCE(p.promised_date, p.delivery_date) IS NOT NULL)
		SELECT l.supplier_code, COALESCE(MAX(s.name),''), COUNT(*), COUNT(*) FILTER (WHERE l.chegou <= l.prometido),
		       COALESCE(AVG(GREATEST(l.chegou - l.prometido, 0)), 0)::numeric(10,1)
		  FROM linhas l LEFT JOIN suppliers s ON s.code = l.supplier_code AND s.enterprise_id = $1
		 GROUP BY 1 ORDER BY 4::numeric / NULLIF(COUNT(*),0), 1`, empresa, dia(de), dia(ate))
	if err != nil {
		return nil, fmt.Errorf("lendo a pontualidade dos fornecedores: %w", err)
	}
	lista, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (uc.Pontualidade, error) {
		var p uc.Pontualidade
		err := row.Scan(&p.Code, &p.Nome, &p.Linhas, &p.NoPrazo, &p.AtrasoMedio)
		return p, err
	})
	for i := range lista {
		lista[i].Percentual = uc.Pct(decimal.NewFromInt(int64(lista[i].NoPrazo)), decimal.NewFromInt(int64(lista[i].Linhas)))
	}
	return lista, err
}

// AcertoDaPromessa: linhas de pedido de venda faturadas no período (NF-e
// autorizada ligada à linha) contra a data de entrega prometida.
func (r *Repo) AcertoDaPromessa(ctx context.Context, de, ate time.Time) (*uc.Pontualidade, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var p uc.Pontualidade
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE COALESCE(x.data_saida, x.data_emissao)::date <= l.delivery_date::date),
		       COALESCE(AVG(GREATEST(COALESCE(x.data_saida, x.data_emissao)::date - l.delivery_date::date, 0)), 0)::numeric(10,1)
		  FROM fiscal_exit_items i
		  JOIN fiscal_exits x ON x.id = i.fiscal_exit_id AND x.enterprise_id = $1 AND x.status = 'AUTHORIZED'
		  JOIN sales_order_items l ON l.code = i.sales_order_item_code
		 WHERE x.data_emissao BETWEEN $2::date AND $3::date AND l.delivery_date IS NOT NULL`, empresa, dia(de), dia(ate)).
		Scan(&p.Linhas, &p.NoPrazo, &p.AtrasoMedio); err != nil {
		return nil, fmt.Errorf("lendo o acerto da promessa: %w", err)
	}
	p.Nome = "Pedidos de venda"
	p.Percentual = uc.Pct(decimal.NewFromInt(int64(p.NoPrazo)), decimal.NewFromInt(int64(p.Linhas)))
	return &p, nil
}
