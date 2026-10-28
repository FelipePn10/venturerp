package financial

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	costcenterrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/cost_center"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// referenciaUso é uma coluna que aponta para o cadastro pelo id. Várias não têm
// FK (títulos, rateio de frete, regras de rateio): sem conferir aqui, excluir
// deixaria esses registros apontando para nada.
type referenciaUso struct {
	tabela, coluna, onde string
}

var usosPlanoContas = []referenciaUso{
	{"contas_pagar", "plano_contas_id", "títulos a pagar"},
	{"contas_receber", "plano_contas_id", "títulos a receber"},
	{"contas_pagar_rateios", "plano_contas_id", "rateios de títulos a pagar"},
	{"fiscal_entry_items", "plano_contas_id", "itens de notas de entrada"},
	{"fiscal_entry_installment_allocations", "plano_contas_id", "rateios de parcelas de notas de entrada"},
	{"fiscal_freight_allocations", "plano_contas_id", "rateios de frete"},
	{"cost_overhead_rules", "plano_contas_id", "regras de rateio de custos indiretos"},
}

var usosCentroCusto = []referenciaUso{
	{"contas_pagar", "centro_custo_id", "títulos a pagar"},
	{"contas_receber", "centro_custo_id", "títulos a receber"},
	{"contas_pagar_rateios", "centro_custo_id", "rateios de títulos a pagar"},
	{"fiscal_entry_items", "centro_custo_id", "itens de notas de entrada"},
	{"fiscal_entry_installment_allocations", "centro_custo_id", "rateios de parcelas de notas de entrada"},
	{"fiscal_freight_allocations", "centro_custo_id", "rateios de frete"},
	{"cost_overhead_rules", "centro_custo_id", "regras de rateio de custos indiretos"},
	{"accounting_account_refs", "cost_center_id", "referências do plano contábil (ECD)"},
}

func (r *FinancialRepositoryPG) contarUsos(ctx context.Context, refs []referenciaUso, id int64) ([]entity.UsoCadastro, error) {
	var out []entity.UsoCadastro
	for _, ref := range refs {
		var n int64
		// Tabela e coluna vêm da lista fixa acima, nunca da requisição.
		if err := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM %s WHERE %s = $1`, ref.tabela, ref.coluna), id).Scan(&n); err != nil {
			return nil, fmt.Errorf("conferindo o uso em %s: %w", ref.onde, err)
		}
		if n > 0 {
			out = append(out, entity.UsoCadastro{Onde: ref.onde, Quantidade: n})
		}
	}
	return out, nil
}

// ---------- Plano de Contas ----------

const planoCols = `id, codigo, descricao, tipo, natureza, parent_code, nivel, is_active, created_at, accounting_account_id, exige_centro_custo`

func scanPlano(row pgx.Row) (*entity.PlanoContas, error) {
	var p entity.PlanoContas
	if err := row.Scan(&p.ID, &p.Codigo, &p.Descricao, &p.Tipo, &p.Natureza, &p.ParentCode, &p.Nivel, &p.IsActive, &p.CreatedAt, &p.AccountingAccountID, &p.ExigeCentroCusto); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError("plano de contas não encontrado")
		}
		return nil, err
	}
	return &p, nil
}

func (r *FinancialRepositoryPG) ListPlanoContasComInativos(ctx context.Context) ([]*entity.PlanoContas, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+planoCols+` FROM plano_contas WHERE enterprise_id = $1 ORDER BY ordem_natural(codigo)`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando o plano de contas: %w", err)
	}
	defer rows.Close()
	var out []*entity.PlanoContas
	for rows.Next() {
		p, err := scanPlano(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *FinancialRepositoryPG) GetPlanoContas(ctx context.Context, id int64) (*entity.PlanoContas, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return scanPlano(r.pool.QueryRow(ctx, `SELECT `+planoCols+` FROM plano_contas WHERE id = $1 AND enterprise_id = $2`, id, empresa))
}

func (r *FinancialRepositoryPG) GetPlanoContasPorCodigo(ctx context.Context, codigo string) (*entity.PlanoContas, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return scanPlano(r.pool.QueryRow(ctx, `SELECT `+planoCols+` FROM plano_contas WHERE codigo = $1 AND enterprise_id = $2`, codigo, empresa))
}

func (r *FinancialRepositoryPG) UpdatePlanoContas(ctx context.Context, p *entity.PlanoContas) (*entity.PlanoContas, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return scanPlano(r.pool.QueryRow(ctx,
		`UPDATE plano_contas SET codigo = $3, descricao = $4, tipo = $5, natureza = $6, parent_code = $7, nivel = $8, is_active = $9,
		        exige_centro_custo = $10
		  WHERE id = $1 AND enterprise_id = $2 RETURNING `+planoCols,
		p.ID, empresa, p.Codigo, p.Descricao, p.Tipo, p.Natureza, p.ParentCode, p.Nivel, p.IsActive, p.ExigeCentroCusto))
}

func (r *FinancialRepositoryPG) DeletePlanoContas(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM plano_contas WHERE id = $1 AND enterprise_id = $2`, id, empresa)
	if err != nil {
		return fmt.Errorf("excluindo o plano de contas: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("plano de contas não encontrado")
	}
	return nil
}

func (r *FinancialRepositoryPG) UsosPlanoContas(ctx context.Context, id int64) ([]entity.UsoCadastro, error) {
	out, err := r.contarUsos(ctx, usosPlanoContas, id)
	if err != nil {
		return nil, err
	}
	// Contas filhas apontam para o CÓDIGO do pai (da mesma empresa).
	var filhas int64
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM plano_contas f JOIN plano_contas p ON p.codigo = f.parent_code AND p.enterprise_id = f.enterprise_id
		  WHERE p.id = $1`, id).Scan(&filhas); err != nil {
		return nil, fmt.Errorf("conferindo as contas filhas: %w", err)
	}
	if filhas > 0 {
		out = append(out, entity.UsoCadastro{Onde: "contas filhas", Quantidade: filhas})
	}
	return out, nil
}

func (r *FinancialRepositoryPG) FilhasAtivasPlanoContas(ctx context.Context, codigo string) (int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	var n int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM plano_contas WHERE enterprise_id = $1 AND parent_code = $2 AND is_active`, empresa, codigo).Scan(&n); err != nil {
		return 0, fmt.Errorf("conferindo as contas filhas ativas: %w", err)
	}
	return n, nil
}

// ---------- Centros de Custo ----------

const centroCols = `id, codigo, descricao, tipo, is_active, created_at`

// centrosCustoSQL: o cadastro financeiro de centro de custo (VFIN0130) é uma
// vista do cadastro único (cost_centers + a gestão, que guarda o código da
// empresa). O tipo segue os nomes antigos desta rota.
const centrosCustoSQL = `(SELECT c.id, g.codigo, c.description AS descricao,
		CASE c.type WHEN 'PRODUCTIVE' THEN 'PRODUTIVO' WHEN 'COMMERCIAL' THEN 'COMERCIAL'
		            WHEN 'AUXILIARY' THEN 'AUXILIAR' ELSE 'ADMINISTRATIVO' END AS tipo,
		c.is_active, c.created_at, c.enterprise_id, c.start_date, c.end_date
	   FROM cost_centers c JOIN cost_center_gestao g ON g.cost_center_id = c.id) centros_custo`

// tipoIndustrial: o tipo da rota financeira no cadastro único.
func tipoIndustrial(t string) string {
	switch strings.ToUpper(strings.TrimSpace(t)) {
	case "PRODUTIVO":
		return "PRODUCTIVE"
	case "COMERCIAL":
		return "COMMERCIAL"
	case "AUXILIAR":
		return "AUXILIARY"
	}
	return "ADMINISTRATIVE"
}

func scanCentro(row pgx.Row) (*entity.CentroCusto, error) {
	var c entity.CentroCusto
	if err := row.Scan(&c.ID, &c.Codigo, &c.Descricao, &c.Tipo, &c.IsActive, &c.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError("centro de custo não encontrado")
		}
		return nil, err
	}
	return &c, nil
}

func (r *FinancialRepositoryPG) ListCentrosCustoComInativos(ctx context.Context) ([]*entity.CentroCusto, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+centroCols+` FROM `+centrosCustoSQL+` WHERE enterprise_id = $1 ORDER BY codigo`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando os centros de custo: %w", err)
	}
	defer rows.Close()
	var out []*entity.CentroCusto
	for rows.Next() {
		c, err := scanCentro(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *FinancialRepositoryPG) GetCentroCusto(ctx context.Context, id int64) (*entity.CentroCusto, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return scanCentro(r.pool.QueryRow(ctx, `SELECT `+centroCols+` FROM `+centrosCustoSQL+` WHERE id = $1 AND enterprise_id = $2`, id, empresa))
}

func (r *FinancialRepositoryPG) UpdateCentroCusto(ctx context.Context, c *entity.CentroCusto) (*entity.CentroCusto, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	// A mesma regra do cadastro único: desativar com centro filho ativo, não.
	if !c.IsActive {
		var filhos int64
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM cost_centers f JOIN cost_centers p ON p.code = f.parent_code
			WHERE p.id = $1 AND p.enterprise_id = $2 AND f.is_active`, c.ID, empresa).Scan(&filhos); err != nil {
			return nil, err
		}
		if filhos > 0 {
			return nil, errorsuc.NewConflictError(fmt.Sprintf("o centro de custo tem %d centro(s) filho(s) ativo(s): desative os filhos antes", filhos))
		}
	}
	tag, err := tx.Exec(ctx, `UPDATE cost_centers SET description = $3, type = $4::type_cc_enum, is_active = $5, updated_at = NOW()
		  WHERE id = $1 AND enterprise_id = $2`, c.ID, empresa, c.Descricao, tipoIndustrial(c.Tipo), c.IsActive)
	if err != nil {
		return nil, fmt.Errorf("alterando o centro de custo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, errorsuc.NewNotFoundError("centro de custo não encontrado")
	}
	if _, err := tx.Exec(ctx, `UPDATE cost_center_gestao SET codigo = $2, updated_at = NOW() WHERE cost_center_id = $1`, c.ID, strings.TrimSpace(c.Codigo)); err != nil {
		return nil, traduzirCodigoRepetido(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetCentroCusto(ctx, c.ID)
}

func (r *FinancialRepositoryPG) DeleteCentroCusto(ctx context.Context, id int64) error {
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

// UsosCentroCusto: a mesma lista do cadastro único (lado industrial e
// financeiro) — excluir por aqui não pode deixar máquina ou estrutura órfã.
func (r *FinancialRepositoryPG) UsosCentroCusto(ctx context.Context, id int64) ([]entity.UsoCadastro, error) {
	usos, err := costcenterrepo.UsosDoCentro(ctx, r.pool, id)
	if err != nil {
		return nil, err
	}
	out := make([]entity.UsoCadastro, 0, len(usos))
	for _, u := range usos {
		out = append(out, entity.UsoCadastro{Onde: u.Onde, Quantidade: u.Quantidade})
	}
	return out, nil
}

// traduzirCodigoRepetido: o código do centro é único na empresa.
func traduzirCodigoRepetido(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return errorsuc.NewConflictError("já existe centro de custo com este código nesta empresa")
	}
	return err
}
