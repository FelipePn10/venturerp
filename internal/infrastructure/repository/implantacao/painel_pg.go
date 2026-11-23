// Package implantacao lê do banco as contagens do painel de implantação.
package implantacao

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FelipePn10/panossoerp/internal/domain/implantacao"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// contagensSQL: uma linha por verificação (chave, pendentes, total) da empresa $1.
// Item comprado = engineering_type 1; fabricado = 0 (ver mrp_report).
const contagensSQL = `
WITH itens AS (SELECT code, engineering_type, NULLIF(TRIM(supplies_purchase_uom),'') compra, NULLIF(TRIM(warehouse_unit_of_measurement),'') estoque,
                      planning_multiple_lot FROM items WHERE enterprise_id = $1),
     fornecedores AS (SELECT s.id, s.code FROM suppliers s WHERE s.enterprise_id = $1 AND s.is_active AND NOT COALESCE(s.blocked, FALSE))
SELECT 'config_fiscal', CASE WHEN EXISTS (SELECT 1 FROM fiscal_configs f WHERE f.enterprise_id = $1
            AND COALESCE(TRIM(f.cnpj_empresa),'') <> '' AND COALESCE(TRIM(f.uf_empresa),'') <> ''
            AND COALESCE(NULLIF(TRIM(f.codigo_municipio::text),'0'),'') <> '') THEN 0 ELSE 1 END, 1
UNION ALL SELECT 'tipos_nota', CASE WHEN EXISTS (SELECT 1 FROM invoice_types WHERE enterprise_id = $1 AND is_active) THEN 0 ELSE 1 END, 1
UNION ALL SELECT 'tipos_entrada', CASE WHEN EXISTS (SELECT 1 FROM entry_operation_types WHERE enterprise_id = $1 AND is_active) THEN 0 ELSE 1 END, 1
UNION ALL SELECT 'plano_contas', CASE WHEN EXISTS (SELECT 1 FROM accounting_plans WHERE enterprise_id = $1) THEN 0 ELSE 1 END, 1
UNION ALL SELECT 'parametros_contabeis', CASE WHEN EXISTS (SELECT 1 FROM accounting_posting_params WHERE enterprise_id = $1
            AND plan_id IS NOT NULL AND estoque_account_id IS NOT NULL AND cmv_account_id IS NOT NULL) THEN 0 ELSE 1 END, 1
UNION ALL SELECT 'condicoes_pagamento', CASE WHEN EXISTS (SELECT 1 FROM payment_conditions WHERE enterprise_id = $1 AND is_active) THEN 0 ELSE 1 END, 1
UNION ALL SELECT 'fornecedores_endereco',
       (SELECT COUNT(*) FROM fornecedores f WHERE NOT EXISTS (SELECT 1 FROM supplier_addresses a WHERE a.supplier_id = f.id
            AND COALESCE(TRIM(a.street),'') <> '' AND COALESCE(TRIM(a.city),'') <> '' AND COALESCE(TRIM(a.uf),'') <> '')),
       (SELECT COUNT(*) FROM fornecedores)
UNION ALL SELECT 'fonte_suprimento',
       (SELECT COUNT(*) FROM itens i WHERE i.engineering_type = 1 AND NOT EXISTS (SELECT 1 FROM item_preferred_suppliers p
            WHERE p.enterprise_id = $1 AND p.item_code = i.code AND p.is_active AND NOT p.blocked)),
       (SELECT COUNT(*) FROM itens WHERE engineering_type = 1)
UNION ALL SELECT 'conversoes',
       (SELECT COUNT(*) FROM itens i WHERE i.compra IS NOT NULL AND i.estoque IS NOT NULL AND UPPER(i.compra) <> UPPER(i.estoque)
            AND NOT EXISTS (SELECT 1 FROM item_unit_conversions c WHERE c.item_code = i.code AND c.is_active
                             AND ((UPPER(c.from_uom) = UPPER(i.compra) AND UPPER(c.to_uom) = UPPER(i.estoque))
                               OR (UPPER(c.from_uom) = UPPER(i.estoque) AND UPPER(c.to_uom) = UPPER(i.compra))))
            AND NOT EXISTS (SELECT 1 FROM global_unit_conversions g WHERE g.enterprise_id = $1 AND g.is_active
                             AND ((UPPER(g.from_uom) = UPPER(i.compra) AND UPPER(g.to_uom) = UPPER(i.estoque))
                               OR (UPPER(g.from_uom) = UPPER(i.estoque) AND UPPER(g.to_uom) = UPPER(i.compra))))),
       (SELECT COUNT(*) FROM itens i WHERE i.compra IS NOT NULL AND i.estoque IS NOT NULL AND UPPER(i.compra) <> UPPER(i.estoque))
UNION ALL SELECT 'lote_multiplo',
       (SELECT COUNT(*) FROM itens WHERE engineering_type = 1 AND COALESCE(planning_multiple_lot, 0) <= 0),
       (SELECT COUNT(*) FROM itens WHERE engineering_type = 1)
UNION ALL SELECT 'custo_padrao',
       (SELECT COUNT(*) FROM itens i WHERE i.engineering_type IN (0, 1) AND NOT EXISTS (SELECT 1 FROM item_standard_costs c
            WHERE c.item_code = i.code AND c.total_cost > 0)),
       (SELECT COUNT(*) FROM itens WHERE engineering_type IN (0, 1))
UNION ALL SELECT 'taxa_mao_obra',
       (SELECT COUNT(*) FROM machine_types m WHERE m.enterprise_id = $1 AND m.is_active AND NOT EXISTS (SELECT 1 FROM work_center_costs w
            WHERE w.work_center_id = m.id AND w.enterprise_id = $1 AND COALESCE(w.labor_cost_per_hour, 0) > 0)),
       (SELECT COUNT(*) FROM machine_types m WHERE m.enterprise_id = $1 AND m.is_active)
UNION ALL SELECT 'inventarios_abertos',
       (SELECT COUNT(*) FROM physical_inventories WHERE enterprise_id = $1 AND status IN ('OPEN','EM_CONFERENCIA','AGUARDANDO_APROVACAO')),
       (SELECT COUNT(*) FROM physical_inventories WHERE enterprise_id = $1 AND status IN ('OPEN','EM_CONFERENCIA','AGUARDANDO_APROVACAO'))`

// Contagens devolve a contagem de cada verificação do painel da empresa.
func (r *Repo) Contagens(ctx context.Context) (map[string]implantacao.Contagem, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, contagensSQL, empresa)
	if err != nil {
		return nil, fmt.Errorf("lendo o painel de implantação: %w", err)
	}
	defer rows.Close()
	out := map[string]implantacao.Contagem{}
	for rows.Next() {
		var chave string
		var c implantacao.Contagem
		if err := rows.Scan(&chave, &c.Pendentes, &c.Total); err != nil {
			return nil, fmt.Errorf("lendo o painel de implantação: %w", err)
		}
		out[chave] = c
	}
	return out, rows.Err()
}
