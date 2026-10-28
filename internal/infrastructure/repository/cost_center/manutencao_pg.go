package cost_center

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// ManutencaoPG altera, exclui e confere o uso do centro de custo industrial.
//
// O centro é referenciado de dois jeitos: pelo id e pelo código (que é único no
// banco inteiro, migração 343). Metade das referências por código não tem FK —
// estrutura de produto, itens de pedido e requisição de compra, venda recorrente,
// contabilização do representante —, então a exclusão confere tudo aqui em vez
// de confiar no banco.
type ManutencaoPG struct {
	pool *pgxpool.Pool
}

func NewManutencaoPG(pool *pgxpool.Pool) *ManutencaoPG {
	return &ManutencaoPG{pool: pool}
}

type referencia struct {
	tabela, coluna, onde string
	porCodigo            bool
}

var usosCentroIndustrial = []referencia{
	{"production_orders", "cost_center_id", "ordens de produção", false},
	{"employee_functions", "cost_center_id", "funções de funcionários", false},
	{"machine_types", "labor_cost_center_id", "tipos de máquina (mão de obra)", false},
	{"machine_types", "machine_cost_center_id", "tipos de máquina (máquina)", false},
	{"overhead_allocation_targets", "cost_center_id", "destinos de rateio", false},
	{"overhead_allocations", "cost_center_id", "rateios apurados", false},
	{"machines", "cost_center_code", "máquinas", true},
	{"allocation_base_items", "cost_center_code", "bases de rateio", true},
	{"overhead_allocation_targets", "cost_center_code", "destinos de rateio (por código)", true},
	{"overhead_allocations", "cost_center_code", "rateios apurados (por código)", true},
	{"independent_demands", "cost_center_code", "demandas independentes", true},
	{"planned_orders", "cost_center_code", "ordens planejadas", true},
	{"item_structures", "cost_center_code", "estruturas de produto", true},
	{"purchase_requisition_items", "cost_center_code", "itens de requisição de compra", true},
	{"purchase_order_items", "cost_center_code", "itens de pedido de compra", true},
	{"recurring_sales", "cost_center_code", "vendas recorrentes", true},
	{"representative_accounting", "credit_cost_center_code", "contabilização de representantes (crédito)", true},
	{"representative_accounting", "debit_cost_center_code", "contabilização de representantes (débito)", true},
	// Desde o cadastro único (migração 447), o lado financeiro também aponta
	// para cost_centers pelo id.
	{"contas_pagar", "centro_custo_id", "títulos a pagar", false},
	{"contas_receber", "centro_custo_id", "títulos a receber", false},
	{"contas_pagar_rateios", "centro_custo_id", "rateios de títulos a pagar", false},
	{"fiscal_entry_items", "centro_custo_id", "itens de notas de entrada", false},
	{"fiscal_entry_installment_allocations", "centro_custo_id", "rateios de parcelas de notas de entrada", false},
	{"fiscal_freight_allocations", "centro_custo_id", "rateios de frete", false},
	{"cost_overhead_rules", "centro_custo_id", "regras de rateio de custos indiretos", false},
	{"accounting_account_refs", "cost_center_id", "referências do plano contábil (ECD)", false},
	{"requisicao_aprovacoes_centro", "cost_center_id", "aprovações de requisição de compra", false},
}

// UsosDoCentro: onde o centro (pelo id) é usado, nos dois lados — a rota
// financeira antiga (VFIN0130) confere a mesma lista antes de excluir.
func UsosDoCentro(ctx context.Context, pool consulta, id int64) ([]repository.UsoCentroCusto, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var code int32
	if err := pool.QueryRow(ctx, `SELECT code FROM cost_centers WHERE id = $1 AND enterprise_id = $2`, id, empresa).Scan(&code); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError("centro de custo não encontrado")
		}
		return nil, err
	}
	return usos(ctx, pool, &entity.CostCenter{ID: id, Code: code})
}

// consulta: o pool ou uma transação.
type consulta interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

const colunasCentro = `id, code, description, parent_code, type, is_ratio, start_date, end_date, is_active, created_at, updated_at, created_by, enterprise_id`

func (r *ManutencaoPG) ListarComInativos(ctx context.Context) ([]*entity.CostCenter, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunasCentro+` FROM cost_centers WHERE enterprise_id = $1 ORDER BY code`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando os centros de custo: %w", err)
	}
	defer rows.Close()
	var out []*entity.CostCenter
	for rows.Next() {
		var c sqlc.CostCenter
		if err := rows.Scan(&c.ID, &c.Code, &c.Description, &c.ParentCode, &c.Type, &c.IsRatio, &c.StartDate,
			&c.EndDate, &c.IsActive, &c.CreatedAt, &c.UpdatedAt, &c.CreatedBy, &c.EnterpriseID); err != nil {
			return nil, err
		}
		out = append(out, rowToEntity(c))
	}
	return out, rows.Err()
}

func (r *ManutencaoPG) Usos(ctx context.Context, cc *entity.CostCenter) ([]repository.UsoCentroCusto, error) {
	return usos(ctx, r.pool, cc)
}

func usos(ctx context.Context, q consulta, cc *entity.CostCenter) ([]repository.UsoCentroCusto, error) {
	var out []repository.UsoCentroCusto
	for _, ref := range usosCentroIndustrial {
		var chave any = cc.ID
		if ref.porCodigo {
			chave = cc.Code
		}
		var n int64
		// Tabela e coluna vêm da lista fixa acima, nunca da requisição.
		if err := q.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM %s WHERE %s = $1`, ref.tabela, ref.coluna), chave).Scan(&n); err != nil {
			return nil, fmt.Errorf("conferindo o uso em %s: %w", ref.onde, err)
		}
		if n > 0 {
			out = append(out, repository.UsoCentroCusto{Onde: ref.onde, Quantidade: n})
		}
	}
	var filhos int64
	if err := q.QueryRow(ctx, `SELECT count(*) FROM cost_centers WHERE parent_code = $1`, cc.Code).Scan(&filhos); err != nil {
		return nil, fmt.Errorf("conferindo os centros filhos: %w", err)
	}
	if filhos > 0 {
		out = append(out, repository.UsoCentroCusto{Onde: "centros de custo filhos", Quantidade: filhos})
	}
	return out, nil
}

func (r *ManutencaoPG) FilhosAtivos(ctx context.Context, code int32) (int64, error) {
	var n int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM cost_centers WHERE parent_code = $1 AND is_active`, code).Scan(&n); err != nil {
		return 0, fmt.Errorf("conferindo os centros filhos ativos: %w", err)
	}
	return n, nil
}

func (r *ManutencaoPG) Alterar(ctx context.Context, a repository.AlteracaoCentroCusto) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE cost_centers SET description = $3, parent_code = $4, type = $5, is_ratio = $6,
		        start_date = $7, end_date = $8, is_active = $9, updated_at = NOW()
		  WHERE id = $1 AND enterprise_id = $2`,
		a.ID, empresa, a.Description, a.ParentCode, a.Type, a.IsRatio, a.StartDate, a.EndDate, a.IsActive)
	if err != nil {
		return fmt.Errorf("alterando o centro de custo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("centro de custo não encontrado")
	}
	return nil
}

func (r *ManutencaoPG) Excluir(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM cost_centers WHERE id = $1 AND enterprise_id = $2`, id, empresa)
	if err != nil {
		return fmt.Errorf("excluindo o centro de custo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("centro de custo não encontrado")
	}
	return nil
}

// ProximoCodigo: o próximo code numérico livre (único no banco). Dois cadastros
// ao mesmo tempo: o segundo recebe conflito do índice único e tenta de novo.
func (r *ManutencaoPG) ProximoCodigo(ctx context.Context) (int32, error) {
	var c int32
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(code), 0) + 1 FROM cost_centers`).Scan(&c); err != nil {
		return 0, fmt.Errorf("calculando o próximo código de centro de custo: %w", err)
	}
	return c, nil
}
