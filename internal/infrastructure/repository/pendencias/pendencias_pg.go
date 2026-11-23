// Package pendencias lê do banco as contagens da central de pendências do dia.
package pendencias

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/FelipePn10/panossoerp/internal/domain/pendencias"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// contagensSQL: (chave, quantidade, valor, mais antiga) da empresa $1 (id) e
// $2 (código — compras usam enterprise_code), com a data de hoje em $3 (a data
// do processo, não CURRENT_DATE: ver AGENTS.md, dois relógios).
const contagensSQL = `
WITH plano AS (SELECT MAX(plan_code) p FROM mrp_planned_suggestions WHERE enterprise_id = $1)
SELECT 'ofs_atrasadas', COUNT(*), 0::numeric, MIN(end_date)::date
  FROM production_orders WHERE enterprise_id = $1 AND status IN ('OPEN','IN_PROGRESS') AND end_date < $3::date
UNION ALL
SELECT 'sugestoes_a_firmar', COUNT(*), 0, MIN(s.start_date)::date
  FROM mrp_planned_suggestions s, plano
 WHERE s.enterprise_id = $1 AND s.plan_code = plano.p AND s.start_date <= $3::date + 7
   AND NOT EXISTS (SELECT 1 FROM planned_orders o WHERE o.enterprise_id = $1 AND o.mrp_suggestion_code = s.code AND o.is_active)
UNION ALL
SELECT 'excecoes_mrp', COUNT(*), 0, NULL::date
  FROM mrp_exception_messages m, plano WHERE m.enterprise_id = $1 AND m.plan_code = plano.p
UNION ALL
SELECT 'compras_sem_preco', COUNT(*), 0, MIN(po.emission_date)::date
  FROM purchase_order_items i JOIN purchase_orders po ON po.code = i.purchase_order_code
 WHERE po.enterprise_code = $2 AND po.status IN ('DRAFT','REQUESTED','APPROVED','PARTIAL')
   AND COALESCE(i.status,'') NOT IN ('CANCELLED','RECEIVED') AND COALESCE(i.unit_price, 0) <= 0
UNION ALL
SELECT 'pedidos_portal', COUNT(*), COALESCE(SUM(total_net),0), MIN(emission_date)::date
  FROM sales_orders WHERE enterprise_code = $2 AND origin = 'PORTAL' AND status = 'R'
UNION ALL
SELECT 'notas_a_lancar', COUNT(*), COALESCE(SUM(valor_total),0), MIN(data_entrada)::date
  FROM fiscal_entries WHERE enterprise_id = $1 AND status IN ('PENDING','CONFERRED') AND is_active
UNION ALL
SELECT 'efeitos_nfe_pendentes', COUNT(*), 0, NULL::date
  FROM fiscal_exit_pendencias WHERE enterprise_id = $1 AND resolvido_em IS NULL
UNION ALL
SELECT 'pagar_vencendo', COUNT(*), COALESCE(SUM(valor_bruto - COALESCE(valor_pago,0)),0), MIN(data_vencimento)::date
  FROM contas_pagar WHERE enterprise_id = $1 AND status NOT IN ('PAGO','CANCELADO') AND data_vencimento <= $3::date + 3
UNION ALL
SELECT 'receber_vencidos', COUNT(*), COALESCE(SUM(valor_bruto - COALESCE(valor_recebido,0)),0), MIN(data_vencimento)::date
  FROM contas_receber WHERE enterprise_id = $1 AND status NOT IN ('RECEBIDO','CANCELADO') AND data_vencimento < $3::date
UNION ALL
SELECT 'inventarios_abertos', COUNT(*), 0, MIN(start_date)::date
  FROM physical_inventories WHERE enterprise_id = $1 AND status IN ('OPEN','EM_CONFERENCIA','AGUARDANDO_APROVACAO')`

// Contagens lê as pendências da empresa da sessão.
func (r *Repo) Contagens(ctx context.Context) (map[string]domain.Contagem, error) {
	id, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	code, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, contagensSQL, id, code, time.Now().Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("lendo a central de pendências: %w", err)
	}
	defer rows.Close()
	out := map[string]domain.Contagem{}
	for rows.Next() {
		var chave string
		var c domain.Contagem
		if err := rows.Scan(&chave, &c.Quantidade, &c.Valor, &c.MaisAntiga); err != nil {
			return nil, fmt.Errorf("lendo a central de pendências: %w", err)
		}
		out[chave] = c
	}
	return out, rows.Err()
}
