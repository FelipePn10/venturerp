package financial

import (
	"context"
	"errors"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/accounting/contabilizacao"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/journal"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	fiscalEntity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// ---------- Contas Bancarias ----------

// Todo cadastro deste arquivo nasce e é lido dentro da empresa autenticada. Até
// a migração 361 nenhuma destas tabelas tinha coluna de empresa: as consultas
// eram `FROM contas_bancarias WHERE is_active = true`, e a alteração de saldo
// localizava a conta só pelo id — uma empresa movimentaria o caixa da outra.
func (r *FinancialRepositoryPG) CreateContaBancaria(ctx context.Context, c *entity.ContaBancaria) (*entity.ContaBancaria, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx,
		`INSERT INTO contas_bancarias (banco, agencia, conta, digito, descricao, titular, saldo_inicial, chave_pix, tipo_chave_pix, is_active, created_by, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		 RETURNING id, created_at, updated_at`,
		c.Banco, c.Agencia, c.Conta, c.Digito, c.Descricao, c.Titular,
		c.SaldoInicial.InexactFloat64(), c.ChavePix, c.TipoChavePix, c.IsActive, c.CreatedBy, empresa,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating conta bancaria: %w", err)
	}
	return c, nil
}

func (r *FinancialRepositoryPG) ListContasBancarias(ctx context.Context) ([]*entity.ContaBancaria, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, banco, agencia, conta, digito, descricao, titular, saldo_inicial, chave_pix, tipo_chave_pix, is_active, created_at, updated_at, created_by,
		        accounting_account_id
		 FROM contas_bancarias WHERE is_active = true AND enterprise_id = $1 ORDER BY descricao`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listing contas bancarias: %w", err)
	}
	defer rows.Close()

	var out []*entity.ContaBancaria
	for rows.Next() {
		var c entity.ContaBancaria
		var saldo float64
		if err := rows.Scan(&c.ID, &c.Banco, &c.Agencia, &c.Conta, &c.Digito, &c.Descricao,
			&c.Titular, &saldo, &c.ChavePix, &c.TipoChavePix, &c.IsActive,
			&c.CreatedAt, &c.UpdatedAt, &c.CreatedBy, &c.AccountingAccountID); err != nil {
			return nil, fmt.Errorf("scanning conta bancaria: %w", err)
		}
		c.SaldoInicial = decimal.NewFromFloat(saldo)
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (r *FinancialRepositoryPG) GetContaBancaria(ctx context.Context, id int64) (*entity.ContaBancaria, error) {
	var c entity.ContaBancaria
	var saldo float64
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx,
		`SELECT id, banco, agencia, conta, digito, descricao, titular, saldo_inicial, chave_pix, tipo_chave_pix, is_active, created_at, updated_at, created_by
		 FROM contas_bancarias WHERE id = $1 AND enterprise_id = $2`, id, empresa,
	).Scan(&c.ID, &c.Banco, &c.Agencia, &c.Conta, &c.Digito, &c.Descricao,
		&c.Titular, &saldo, &c.ChavePix, &c.TipoChavePix, &c.IsActive,
		&c.CreatedAt, &c.UpdatedAt, &c.CreatedBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("conta bancária %d não encontrada", id))
		}
		return nil, fmt.Errorf("getting conta bancaria: %w", err)
	}
	c.SaldoInicial = decimal.NewFromFloat(saldo)
	return &c, nil
}

func (r *FinancialRepositoryPG) UpdateSaldo(ctx context.Context, id int64, novoSaldo float64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE contas_bancarias SET saldo_inicial = $1, updated_at = NOW() WHERE id = $2 AND enterprise_id = $3`,
		novoSaldo, id, empresa)
	if err != nil {
		return fmt.Errorf("updating saldo conta %d: %w", id, err)
	}
	// Zero linhas aqui não é "nada mudou": é a conta de OUTRA empresa. Devolver
	// sucesso silencioso faria a tela mostrar um saldo que não foi gravado.
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("conta bancária %d não encontrada", id))
	}
	return nil
}

// ---------- Condicoes Pagamento ----------

func (r *FinancialRepositoryPG) CreateCondicaoPagamento(ctx context.Context, c *entity.CondicaoPagamento) (*entity.CondicaoPagamento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx,
		`INSERT INTO condicoes_pagamento (nome, parcelas, ativo, enterprise_id)
		 VALUES ($1,$2,$3,$4)
		 RETURNING id, created_at, updated_at`,
		c.Nome, c.Parcelas, c.Ativo, empresa,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating condicao pagamento: %w", err)
	}
	return c, nil
}

func (r *FinancialRepositoryPG) ListCondicoesPagamento(ctx context.Context) ([]*entity.CondicaoPagamento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, nome, parcelas, ativo, created_at, updated_at
		 FROM condicoes_pagamento WHERE ativo = true AND enterprise_id = $1 ORDER BY nome`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listing condicoes pagamento: %w", err)
	}
	defer rows.Close()

	var out []*entity.CondicaoPagamento
	for rows.Next() {
		var c entity.CondicaoPagamento
		if err := rows.Scan(&c.ID, &c.Nome, &c.Parcelas, &c.Ativo, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning condicao pagamento: %w", err)
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// ---------- Plano de Contas ----------

func (r *FinancialRepositoryPG) CreatePlanoContas(ctx context.Context, p *entity.PlanoContas) (*entity.PlanoContas, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx,
		`INSERT INTO plano_contas (codigo, descricao, tipo, natureza, parent_code, nivel, is_active, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING id, created_at`,
		p.Codigo, p.Descricao, p.Tipo, p.Natureza, p.ParentCode, p.Nivel, p.IsActive, empresa,
	).Scan(&p.ID, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating plano contas: %w", err)
	}
	return p, nil
}

func (r *FinancialRepositoryPG) ListPlanoContas(ctx context.Context) ([]*entity.PlanoContas, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, codigo, descricao, tipo, natureza, parent_code, nivel, is_active, created_at, accounting_account_id
		 FROM plano_contas WHERE is_active = true AND enterprise_id = $1 ORDER BY codigo`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listing plano contas: %w", err)
	}
	defer rows.Close()

	var out []*entity.PlanoContas
	for rows.Next() {
		var p entity.PlanoContas
		if err := rows.Scan(&p.ID, &p.Codigo, &p.Descricao, &p.Tipo, &p.Natureza,
			&p.ParentCode, &p.Nivel, &p.IsActive, &p.CreatedAt, &p.AccountingAccountID); err != nil {
			return nil, fmt.Errorf("scanning plano contas: %w", err)
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// ---------- Centros de Custo ----------

func (r *FinancialRepositoryPG) CreateCentroCusto(ctx context.Context, c *entity.CentroCusto) (*entity.CentroCusto, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx,
		`INSERT INTO centros_custo (codigo, descricao, tipo, is_active, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING id, created_at`,
		c.Codigo, c.Descricao, c.Tipo, c.IsActive, empresa,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating centro custo: %w", err)
	}
	return c, nil
}

func (r *FinancialRepositoryPG) ListCentrosCusto(ctx context.Context) ([]*entity.CentroCusto, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, codigo, descricao, tipo, is_active, created_at
		 FROM centros_custo WHERE is_active = true AND enterprise_id = $1 ORDER BY codigo`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listing centros custo: %w", err)
	}
	defer rows.Close()

	var out []*entity.CentroCusto
	for rows.Next() {
		var c entity.CentroCusto
		if err := rows.Scan(&c.ID, &c.Codigo, &c.Descricao, &c.Tipo, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning centro custo: %w", err)
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// ---------- Contas a Pagar ----------

// O título nasce na empresa da sessão. Até a migração 362 `contas_pagar` não
// tinha coluna de empresa: com duas empresas na base, a segunda veria — e
// poderia baixar — os títulos da primeira, e o DRE somaria as duas.
func (r *FinancialRepositoryPG) CreateContaPagar(ctx context.Context, c *entity.ContaPagar) (*entity.ContaPagar, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	// Título e rateio nascem juntos: título sem a distribuição prometida não
	// aparece no plano de contas certo.
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	err = tx.QueryRow(ctx,
		`INSERT INTO contas_pagar
			(numero_documento, tipo_documento, fornecedor_id, fiscal_entry_id, purchase_order_id,
			 data_lancamento, data_emissao, data_vencimento,
			 valor_bruto, desconto, juros, multa, valor_pago,
			 parcela_numero, parcela_total, parcela_pai_id,
			 forma_pagamento, plano_contas_id, centro_custo_id,
			 status_aprovacao, status,
			 observacao, is_active, criado_por, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
		 RETURNING id, created_at, updated_at`,
		c.NumeroDocumento, c.TipoDocumento, c.FornecedorID, c.FiscalEntryID, c.PurchaseOrderID,
		c.DataLancamento, c.DataEmissao, c.DataVencimento,
		c.ValorBruto.InexactFloat64(), c.Desconto.InexactFloat64(), c.Juros.InexactFloat64(), c.Multa.InexactFloat64(), c.ValorPago.InexactFloat64(),
		c.ParcelaNumero, c.ParcelaTotal, c.ParcelaPaiID,
		c.FormaPagamento, c.PlanoContasID, c.CentroCustoID,
		string(c.StatusAprovacao), string(c.Status),
		c.Observacao, c.IsActive, c.CriadoPor, empresa,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating conta pagar: %w", err)
	}
	for i := range c.Rateios {
		rt := &c.Rateios[i]
		if err = tx.QueryRow(ctx,
			`INSERT INTO contas_pagar_rateios (enterprise_id, conta_pagar_id, plano_contas_id, centro_custo_id, valor)
			 VALUES ($1,$2,$3,$4,$5) RETURNING id`, empresa, c.ID, rt.PlanoContasID, rt.CentroCustoID, rt.Valor).Scan(&rt.ID); err != nil {
			return nil, fmt.Errorf("gravando rateio do título: %w", err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// ListRateiosContasPagar devolve o rateio dos títulos, com os nomes do plano
// de contas e do centro de custo.
func (r *FinancialRepositoryPG) ListRateiosContasPagar(ctx context.Context, ids []int64) (map[int64][]entity.RateioContaPagar, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	out := map[int64][]entity.RateioContaPagar{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT r.conta_pagar_id, r.id, r.plano_contas_id, r.centro_custo_id, r.valor, pc.codigo, pc.descricao, cc.descricao
		   FROM contas_pagar_rateios r
		   LEFT JOIN plano_contas pc ON pc.id = r.plano_contas_id
		   LEFT JOIN centros_custo cc ON cc.id = r.centro_custo_id
		  WHERE r.enterprise_id = $1 AND r.conta_pagar_id = ANY($2)
		  ORDER BY r.conta_pagar_id, pc.codigo, r.id`, empresa, ids)
	if err != nil {
		return nil, fmt.Errorf("lendo rateio do contas a pagar: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cpID int64
		var rt entity.RateioContaPagar
		if err := rows.Scan(&cpID, &rt.ID, &rt.PlanoContasID, &rt.CentroCustoID, &rt.Valor, &rt.PlanoContasCodigo, &rt.PlanoContasNome, &rt.CentroCustoNome); err != nil {
			return nil, err
		}
		out[cpID] = append(out[cpID], rt)
	}
	return out, rows.Err()
}

func (r *FinancialRepositoryPG) anexarRateios(ctx context.Context, titulos []*entity.ContaPagar) error {
	ids := make([]int64, 0, len(titulos))
	for _, t := range titulos {
		ids = append(ids, t.ID)
	}
	rateios, err := r.ListRateiosContasPagar(ctx, ids)
	if err != nil {
		return err
	}
	for _, t := range titulos {
		t.Rateios = rateios[t.ID]
	}
	return nil
}

// ContasPagarPorPlano agrupa o contas a pagar por plano de contas. Título com
// rateio entra pela parte de cada plano; o pago de cada parte é proporcional
// ao pagamento do título.
func (r *FinancialRepositoryPG) ContasPagarPorPlano(ctx context.Context, f repository.CPPorPlanoFilter) ([]repository.CPPorPlano, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	conds := []string{"cp.enterprise_id = $1", "cp.is_active", "UPPER(cp.status) <> 'CANCELADO'"}
	args := []any{empresa}
	col := "cp." + f.DateField.Coluna()
	if f.StartDate != nil {
		args = append(args, *f.StartDate)
		conds = append(conds, fmt.Sprintf("%s >= $%d", col, len(args)))
	}
	if f.EndDate != nil {
		args = append(args, *f.EndDate)
		conds = append(conds, fmt.Sprintf("%s <= $%d", col, len(args)))
	}
	if f.Status != nil && *f.Status != "" {
		args = append(args, *f.Status)
		conds = append(conds, fmt.Sprintf("UPPER(cp.status) = UPPER($%d)", len(args)))
	}
	if f.FornecedorID != nil {
		args = append(args, *f.FornecedorID)
		conds = append(conds, fmt.Sprintf("cp.fornecedor_id = $%d", len(args)))
	}
	q := `WITH titulos AS (
	        SELECT cp.id, cp.valor_bruto, COALESCE(cp.valor_pago,0) AS valor_pago, UPPER(cp.status) AS status,
	               cp.data_vencimento, cp.plano_contas_id, cp.centro_custo_id
	          FROM contas_pagar cp WHERE ` + strings.Join(conds, " AND ") + `
	      ), partes AS (
	        SELECT t.id, r.plano_contas_id, r.centro_custo_id, r.valor AS parte, t.valor_bruto, t.valor_pago, t.status, t.data_vencimento
	          FROM titulos t JOIN contas_pagar_rateios r ON r.conta_pagar_id = t.id
	        UNION ALL
	        SELECT t.id, t.plano_contas_id, t.centro_custo_id, t.valor_bruto, t.valor_bruto, t.valor_pago, t.status, t.data_vencimento
	          FROM titulos t WHERE NOT EXISTS (SELECT 1 FROM contas_pagar_rateios r WHERE r.conta_pagar_id = t.id)
	      ), valores AS (
	        SELECT p.*, CASE WHEN p.status = 'PAGO' THEN p.parte
	                         WHEN p.valor_bruto > 0 THEN ROUND(p.parte * LEAST(p.valor_pago, p.valor_bruto) / p.valor_bruto, 2)
	                         ELSE 0 END AS pago
	          FROM partes p
	      )
	      SELECT v.plano_contas_id, COALESCE(pc.codigo, ''), COALESCE(pc.descricao, 'Sem plano de contas'),
	             v.centro_custo_id, COALESCE(cc.descricao, ''),
	             COUNT(DISTINCT v.id), SUM(v.parte), SUM(v.pago), SUM(v.parte - v.pago),
	             SUM(CASE WHEN v.data_vencimento < CURRENT_DATE AND v.status <> 'PAGO' THEN v.parte - v.pago ELSE 0 END)
	        FROM valores v
	        LEFT JOIN plano_contas pc ON pc.id = v.plano_contas_id
	        LEFT JOIN centros_custo cc ON cc.id = v.centro_custo_id
	       GROUP BY v.plano_contas_id, pc.codigo, pc.descricao, v.centro_custo_id, cc.descricao
	       ORDER BY pc.codigo NULLS LAST, cc.descricao NULLS FIRST`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("agrupando contas a pagar por plano de contas: %w", err)
	}
	defer rows.Close()
	out := []repository.CPPorPlano{}
	for rows.Next() {
		var x repository.CPPorPlano
		if err := rows.Scan(&x.PlanoContasID, &x.PlanoContasCodigo, &x.PlanoContasNome, &x.CentroCustoID, &x.CentroCustoNome,
			&x.QtdTitulos, &x.ValorTotal, &x.ValorPago, &x.ValorAberto, &x.ValorVencido); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *FinancialRepositoryPG) GetContaPagar(ctx context.Context, id int64) (*entity.ContaPagar, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	c, err := r.scanContaPagarRow(r.pool.QueryRow(ctx,
		`SELECT id, numero_documento, tipo_documento, fornecedor_id, fiscal_entry_id, purchase_order_id,
		        data_lancamento, data_emissao, data_vencimento, data_pagamento,
		        COALESCE(valor_bruto,0), COALESCE(desconto,0), COALESCE(juros,0), COALESCE(multa,0), COALESCE(valor_pago,0),
		        parcela_numero, parcela_total, parcela_pai_id,
		        conta_bancaria_id, forma_pagamento,
		        plano_contas_id, centro_custo_id,
		        status_aprovacao, aprovado_por, data_aprovacao, motivo_rejeicao,
		        status, adiantamento_id, COALESCE(valor_adiantamento_abatido,0),
		        comprovante_path, observacao,
		        is_active, criado_por, baixado_por, created_at, updated_at
		 FROM contas_pagar WHERE id = $1 AND enterprise_id = $2`, id, empresa))
	if err != nil {
		return nil, err
	}
	if err := r.anexarRateios(ctx, []*entity.ContaPagar{c}); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *FinancialRepositoryPG) ListContasPagar(ctx context.Context, filters repository.CPFilter) ([]*entity.ContaPagar, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT id, numero_documento, tipo_documento, fornecedor_id, fiscal_entry_id, purchase_order_id,
		        data_lancamento, data_emissao, data_vencimento, data_pagamento,
		        COALESCE(valor_bruto,0), COALESCE(desconto,0), COALESCE(juros,0), COALESCE(multa,0), COALESCE(valor_pago,0),
		        parcela_numero, parcela_total, parcela_pai_id,
		        conta_bancaria_id, forma_pagamento,
		        plano_contas_id, centro_custo_id,
		        status_aprovacao, aprovado_por, data_aprovacao, motivo_rejeicao,
		        status, adiantamento_id, COALESCE(valor_adiantamento_abatido,0),
		        comprovante_path, observacao,
		        is_active, criado_por, baixado_por, created_at, updated_at
		 FROM contas_pagar WHERE is_active = true AND enterprise_id = $1`

	args := []interface{}{empresa}
	argIdx := 2

	// A coluna da data vem de DateField.Coluna(), um domínio fechado de dois
	// valores — nunca de texto do cliente, que aqui seria injeção de SQL.
	colunaData := filters.DateField.Coluna()

	// Situação compara sem diferenciar caixa: a coluna é VARCHAR gravada em
	// MAIÚSCULAS (default 'PENDENTE') e a tela historicamente manda minúsculas.
	// Comparação literal aqui é um filtro que nunca casa — e não acusa nada.
	if filters.Status != nil {
		query += fmt.Sprintf(" AND UPPER(status) = UPPER($%d)", argIdx)
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.StatusAprovacao != nil {
		query += fmt.Sprintf(" AND UPPER(status_aprovacao) = UPPER($%d)", argIdx)
		args = append(args, *filters.StatusAprovacao)
		argIdx++
	}
	if filters.FornecedorID != nil {
		query += fmt.Sprintf(" AND fornecedor_id = $%d", argIdx)
		args = append(args, *filters.FornecedorID)
		argIdx++
	}
	// Plano de contas e centro de custo valem pelo título OU pelo rateio: a
	// nota com matéria-prima e EPI aparece no filtro de cada um dos dois.
	if filters.PlanoContasID != nil {
		query += fmt.Sprintf(" AND (plano_contas_id = $%d OR EXISTS (SELECT 1 FROM contas_pagar_rateios rt WHERE rt.conta_pagar_id = contas_pagar.id AND rt.plano_contas_id = $%d))", argIdx, argIdx)
		args = append(args, *filters.PlanoContasID)
		argIdx++
	}
	if filters.CentroCustoID != nil {
		query += fmt.Sprintf(" AND (centro_custo_id = $%d OR EXISTS (SELECT 1 FROM contas_pagar_rateios rt WHERE rt.conta_pagar_id = contas_pagar.id AND rt.centro_custo_id = $%d))", argIdx, argIdx)
		args = append(args, *filters.CentroCustoID)
		argIdx++
	}
	if filters.TipoDocumento != nil {
		query += fmt.Sprintf(" AND tipo_documento = $%d", argIdx)
		args = append(args, *filters.TipoDocumento)
		argIdx++
	}
	if filters.Documento != nil {
		query += fmt.Sprintf(" AND numero_documento ILIKE $%d", argIdx)
		args = append(args, "%"+*filters.Documento+"%")
		argIdx++
	}
	if filters.StartDate != nil {
		query += fmt.Sprintf(" AND %s >= $%d", colunaData, argIdx)
		args = append(args, *filters.StartDate)
		argIdx++
	}
	if filters.EndDate != nil {
		query += fmt.Sprintf(" AND %s <= $%d", colunaData, argIdx)
		args = append(args, *filters.EndDate)
		argIdx++
	}
	if filters.ValorMinimo != nil {
		query += fmt.Sprintf(" AND COALESCE(valor_bruto,0) >= $%d", argIdx)
		args = append(args, *filters.ValorMinimo)
		argIdx++
	}
	if filters.ValorMaximo != nil {
		query += fmt.Sprintf(" AND COALESCE(valor_bruto,0) <= $%d", argIdx)
		args = append(args, *filters.ValorMaximo)
		argIdx++
	}
	if filters.SomenteVencidos {
		// "Vencido" é vencimento passado E título ainda em aberto. Sem a segunda
		// metade, a lista traz tudo que já foi pago no passado — o oposto do que
		// quem cobra procura.
		query += " AND data_vencimento < CURRENT_DATE AND UPPER(status) NOT IN ('PAGO','CANCELADO')"
	}
	query += " ORDER BY data_vencimento ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing contas pagar: %w", err)
	}
	defer rows.Close()

	out := make([]*entity.ContaPagar, 0)
	for rows.Next() {
		c, err := r.scanContaPagar(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if err := r.anexarRateios(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *FinancialRepositoryPG) UpdateContaPagar(ctx context.Context, c *entity.ContaPagar) (*entity.ContaPagar, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE contas_pagar SET
			numero_documento=$1, tipo_documento=$2, fornecedor_id=$3,
			data_emissao=$4, data_vencimento=$5,
			valor_bruto=$6, desconto=$7, juros=$8, multa=$9, valor_pago=$10,
			parcela_numero=$11, parcela_total=$12,
			forma_pagamento=$13, plano_contas_id=$14, centro_custo_id=$15,
			observacao=$16, updated_at=NOW()
		 WHERE id=$17 AND enterprise_id=$18`,
		c.NumeroDocumento, c.TipoDocumento, c.FornecedorID,
		c.DataEmissao, c.DataVencimento,
		c.ValorBruto.InexactFloat64(), c.Desconto.InexactFloat64(), c.Juros.InexactFloat64(), c.Multa.InexactFloat64(), c.ValorPago.InexactFloat64(),
		c.ParcelaNumero, c.ParcelaTotal,
		c.FormaPagamento, c.PlanoContasID, c.CentroCustoID,
		c.Observacao, c.ID, empresa)
	if err != nil {
		return nil, fmt.Errorf("updating conta pagar %d: %w", c.ID, err)
	}
	return r.GetContaPagar(ctx, c.ID)
}

func (r *FinancialRepositoryPG) ApproveContaPagar(ctx context.Context, id int64, approvedBy uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE contas_pagar SET
			status_aprovacao = 'APROVADO',
			status = 'APROVADO',
			aprovado_por = $1,
			data_aprovacao = NOW(),
			updated_at = NOW()
		 WHERE id = $2 AND enterprise_id = $3`, approvedBy, id, empresa)
	if err != nil {
		return fmt.Errorf("approving conta pagar %d: %w", id, err)
	}
	return nil
}

func (r *FinancialRepositoryPG) BaixarContaPagar(ctx context.Context, id int64, params repository.BaixaParams) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE contas_pagar SET
			status = 'PAGO',
			data_pagamento = $1,
			valor_pago = $2,
			juros = $3,
			multa = $4,
			desconto = $5,
			conta_bancaria_id = $6,
			baixado_por = $7,
			observacao = COALESCE(observacao, '') || ' | ' || $8,
			updated_at = NOW()
		 WHERE id = $9 AND enterprise_id = $10`,
		params.DataPagamento, params.ValorPago, params.Juros, params.Multa,
		params.Desconto, params.ContaBancariaID, params.BaixadoPor,
		params.Observacao, id, empresa)
	if err != nil {
		return fmt.Errorf("baixando conta pagar %d: %w", id, err)
	}
	return nil
}

func (r *FinancialRepositoryPG) CancelContaPagar(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE contas_pagar SET status = 'CANCELADO', is_active = false, updated_at = NOW() WHERE id = $1 AND enterprise_id = $2`, id, empresa)
	if err != nil {
		return fmt.Errorf("cancelling conta pagar %d: %w", id, err)
	}
	return nil
}

func (r *FinancialRepositoryPG) GetAgingContasPagar(ctx context.Context) ([]*repository.AgingResult, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT
			CASE
				WHEN data_vencimento < NOW() THEN 'Vencido'
				WHEN data_vencimento <= NOW() + INTERVAL '7 days' THEN '7 dias'
				WHEN data_vencimento <= NOW() + INTERVAL '15 days' THEN '15 dias'
				WHEN data_vencimento <= NOW() + INTERVAL '30 days' THEN '30 dias'
				WHEN data_vencimento <= NOW() + INTERVAL '60 days' THEN '60 dias'
				ELSE '60+ dias'
			END AS period,
			COALESCE(SUM(valor_bruto - COALESCE(valor_pago, 0)), 0) AS total
		 FROM contas_pagar
		 WHERE is_active = true AND status IN ('PENDENTE', 'APROVADO', 'VENCIDO') AND enterprise_id = $1
		 GROUP BY period
		 ORDER BY period`, empresa)
	if err != nil {
		return nil, fmt.Errorf("getting aging contas pagar: %w", err)
	}
	defer rows.Close()

	var out []*repository.AgingResult
	for rows.Next() {
		var a repository.AgingResult
		if err := rows.Scan(&a.Period, &a.Total); err != nil {
			return nil, fmt.Errorf("scanning aging result: %w", err)
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

// ---------- Contas a Receber ----------

func (r *FinancialRepositoryPG) CreateContaReceber(ctx context.Context, c *entity.ContaReceber) (*entity.ContaReceber, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx,
		`INSERT INTO contas_receber
			(numero_documento, cliente_id, fiscal_exit_id, sales_order_id,
			 data_lancamento, data_emissao, data_vencimento,
			 valor_bruto, desconto, juros, multa, valor_recebido,
			 parcela_numero, parcela_total,
			 forma_pagamento, plano_contas_id, centro_custo_id,
			 status, is_active, criado_por, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)
		 RETURNING id, created_at, updated_at`,
		c.NumeroDocumento, c.ClienteID, c.FiscalExitID, c.SalesOrderID,
		c.DataLancamento, c.DataEmissao, c.DataVencimento,
		c.ValorBruto.InexactFloat64(), c.Desconto.InexactFloat64(), c.Juros.InexactFloat64(), c.Multa.InexactFloat64(), c.ValorRecebido.InexactFloat64(),
		c.ParcelaNumero, c.ParcelaTotal,
		c.FormaPagamento, c.PlanoContasID, c.CentroCustoID,
		string(c.Status), c.IsActive, c.CriadoPor, empresa,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating conta receber: %w", err)
	}
	return c, nil
}

func (r *FinancialRepositoryPG) GetContaReceber(ctx context.Context, id int64) (*entity.ContaReceber, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return r.scanContaReceberRow(r.pool.QueryRow(ctx,
		`SELECT id, numero_documento, cliente_id, fiscal_exit_id, sales_order_id, fornecedor_id,
		        data_lancamento, data_emissao, data_vencimento, data_recebimento,
		        COALESCE(valor_bruto,0), COALESCE(desconto,0), COALESCE(juros,0), COALESCE(multa,0), COALESCE(valor_recebido,0),
		        parcela_numero, parcela_total, parcela_pai_id,
		        conta_bancaria_id, forma_pagamento,
		        nosso_numero, linha_digitavel, codigo_barras, chave_pix_gerada,
		        plano_contas_id, centro_custo_id,
		        status, em_protesto,
		        is_active, criado_por, baixado_por, created_at, updated_at
		 FROM contas_receber WHERE id = $1 AND enterprise_id = $2`, id, empresa))
}

func (r *FinancialRepositoryPG) ListContasReceber(ctx context.Context, filters repository.CRFilter) ([]*entity.ContaReceber, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT id, numero_documento, cliente_id, fiscal_exit_id, sales_order_id, fornecedor_id,
		        data_lancamento, data_emissao, data_vencimento, data_recebimento,
		        COALESCE(valor_bruto,0), COALESCE(desconto,0), COALESCE(juros,0), COALESCE(multa,0), COALESCE(valor_recebido,0),
		        parcela_numero, parcela_total, parcela_pai_id,
		        conta_bancaria_id, forma_pagamento,
		        nosso_numero, linha_digitavel, codigo_barras, chave_pix_gerada,
		        plano_contas_id, centro_custo_id,
		        status, em_protesto,
		        is_active, criado_por, baixado_por, created_at, updated_at
		 FROM contas_receber WHERE is_active = true AND enterprise_id = $1`

	args := []interface{}{empresa}
	argIdx := 2

	colunaData := filters.DateField.Coluna()

	if filters.Status != nil {
		query += fmt.Sprintf(" AND UPPER(status) = UPPER($%d)", argIdx)
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.ClienteID != nil {
		query += fmt.Sprintf(" AND cliente_id = $%d", argIdx)
		args = append(args, *filters.ClienteID)
		argIdx++
	}
	if filters.SalesOrderID != nil {
		query += fmt.Sprintf(" AND sales_order_id = $%d", argIdx)
		args = append(args, *filters.SalesOrderID)
		argIdx++
	}
	if filters.FiscalExitID != nil {
		query += fmt.Sprintf(" AND fiscal_exit_id = $%d", argIdx)
		args = append(args, *filters.FiscalExitID)
		argIdx++
	}
	if filters.Documento != nil {
		query += fmt.Sprintf(" AND numero_documento ILIKE $%d", argIdx)
		args = append(args, "%"+*filters.Documento+"%")
		argIdx++
	}
	if filters.StartDate != nil {
		query += fmt.Sprintf(" AND %s >= $%d", colunaData, argIdx)
		args = append(args, *filters.StartDate)
		argIdx++
	}
	if filters.EndDate != nil {
		query += fmt.Sprintf(" AND %s <= $%d", colunaData, argIdx)
		args = append(args, *filters.EndDate)
		argIdx++
	}
	if filters.ValorMinimo != nil {
		query += fmt.Sprintf(" AND COALESCE(valor_bruto,0) >= $%d", argIdx)
		args = append(args, *filters.ValorMinimo)
		argIdx++
	}
	if filters.ValorMaximo != nil {
		query += fmt.Sprintf(" AND COALESCE(valor_bruto,0) <= $%d", argIdx)
		args = append(args, *filters.ValorMaximo)
		argIdx++
	}
	if filters.SomenteVencidos {
		query += " AND data_vencimento < CURRENT_DATE AND UPPER(status) NOT IN ('PAGO','RECEBIDO','CANCELADO')"
	}
	query += " ORDER BY data_vencimento ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing contas receber: %w", err)
	}
	defer rows.Close()

	out := make([]*entity.ContaReceber, 0)
	for rows.Next() {
		c, err := r.scanContaReceber(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *FinancialRepositoryPG) BaixarContaReceber(ctx context.Context, id int64, params repository.BaixaParams) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE contas_receber SET
			status = 'RECEBIDO',
			data_recebimento = $1,
			valor_recebido = $2,
			juros = $3,
			multa = $4,
			desconto = $5,
			conta_bancaria_id = $6,
			baixado_por = $7,
			updated_at = NOW()
		 WHERE id = $8 AND enterprise_id = $9`,
		params.DataPagamento, params.ValorPago, params.Juros, params.Multa,
		params.Desconto, params.ContaBancariaID, params.BaixadoPor, id, empresa)
	if err != nil {
		return fmt.Errorf("baixando conta receber %d: %w", id, err)
	}
	return nil
}

func (r *FinancialRepositoryPG) CancelContaReceber(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE contas_receber SET status = 'CANCELADO', is_active = false, updated_at = NOW() WHERE id = $1 AND enterprise_id = $2`, id, empresa)
	if err != nil {
		return fmt.Errorf("cancelling conta receber %d: %w", id, err)
	}
	return nil
}

func (r *FinancialRepositoryPG) GetAgingContasReceber(ctx context.Context) ([]*repository.AgingResult, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT
			CASE
				WHEN data_vencimento < NOW() THEN 'Vencido'
				WHEN data_vencimento <= NOW() + INTERVAL '7 days' THEN '7 dias'
				WHEN data_vencimento <= NOW() + INTERVAL '15 days' THEN '15 dias'
				WHEN data_vencimento <= NOW() + INTERVAL '30 days' THEN '30 dias'
				WHEN data_vencimento <= NOW() + INTERVAL '60 days' THEN '60 dias'
				ELSE '60+ dias'
			END AS period,
			COALESCE(SUM(valor_bruto - COALESCE(valor_recebido, 0)), 0) AS total
		 FROM contas_receber
		 WHERE is_active = true AND status IN ('PENDENTE', 'APROVADO', 'VENCIDO') AND enterprise_id = $1
		 GROUP BY period
		 ORDER BY period`, empresa)
	if err != nil {
		return nil, fmt.Errorf("getting aging contas receber: %w", err)
	}
	defer rows.Close()

	var out []*repository.AgingResult
	for rows.Next() {
		var a repository.AgingResult
		if err := rows.Scan(&a.Period, &a.Total); err != nil {
			return nil, fmt.Errorf("scanning aging result: %w", err)
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

// ---------- Cash Flow ----------

func (r *FinancialRepositoryPG) CreateFluxoCaixa(ctx context.Context, f *entity.FluxoCaixa) (*entity.FluxoCaixa, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx,
		`INSERT INTO fluxo_caixa
			(data, tipo, valor, conta_bancaria_id, conta_bancaria_destino_id,
			 contas_pagar_id, contas_receber_id, descricao, conciliado, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING id, created_at`,
		f.Data, string(f.Tipo), f.Valor.InexactFloat64(), f.ContaBancariaID, f.ContaBancariaDestinoID,
		f.ContasPagarID, f.ContasReceberID, f.Descricao, f.Conciliado, empresa,
	).Scan(&f.ID, &f.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating fluxo caixa: %w", err)
	}
	return f, nil
}

func (r *FinancialRepositoryPG) GetFluxoCaixa(ctx context.Context, startDate, endDate time.Time) ([]*entity.FluxoCaixa, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, data, tipo, valor, conta_bancaria_id, conta_bancaria_destino_id,
		        contas_pagar_id, contas_receber_id, descricao, conciliado, extrato_hash, created_at
		 FROM fluxo_caixa
		 WHERE data >= $1 AND data <= $2 AND enterprise_id = $3
		 ORDER BY data ASC`, startDate, endDate, empresa)
	if err != nil {
		return nil, fmt.Errorf("getting fluxo caixa: %w", err)
	}
	defer rows.Close()

	var out []*entity.FluxoCaixa
	for rows.Next() {
		var f entity.FluxoCaixa
		var valor float64
		if err := rows.Scan(&f.ID, &f.Data, &f.Tipo, &valor, &f.ContaBancariaID,
			&f.ContaBancariaDestinoID, &f.ContasPagarID, &f.ContasReceberID,
			&f.Descricao, &f.Conciliado, &f.ExtratoHash, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning fluxo caixa: %w", err)
		}
		f.Valor = decimal.NewFromFloat(valor)
		out = append(out, &f)
	}
	return out, rows.Err()
}

func (r *FinancialRepositoryPG) GetFluxoProjetado(ctx context.Context, startDate time.Time) ([]*repository.ProjectedFlow, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`WITH projected AS (
			SELECT data_vencimento AS data, valor_bruto - COALESCE(valor_pago, 0) AS valor, 'SAIDA' AS tipo
			FROM contas_pagar
			WHERE is_active = true AND status IN ('PENDENTE', 'APROVADO', 'VENCIDO') AND enterprise_id = $2
			UNION ALL
			SELECT data_vencimento AS data, valor_bruto - COALESCE(valor_recebido, 0) AS valor, 'ENTRADA' AS tipo
			FROM contas_receber
			WHERE is_active = true AND status IN ('PENDENTE', 'APROVADO', 'VENCIDO') AND enterprise_id = $2
		)
		SELECT data,
			COALESCE(SUM(CASE WHEN tipo = 'ENTRADA' THEN valor ELSE 0 END), 0) AS valor_entradas,
			COALESCE(SUM(CASE WHEN tipo = 'SAIDA' THEN valor ELSE 0 END), 0) AS valor_saidas
		 FROM projected
		 WHERE data >= $1
		 GROUP BY data
		 ORDER BY data ASC`, startDate, empresa)
	if err != nil {
		return nil, fmt.Errorf("getting fluxo projetado: %w", err)
	}
	defer rows.Close()

	var out []*repository.ProjectedFlow
	var saldo float64
	for rows.Next() {
		var p repository.ProjectedFlow
		if err := rows.Scan(&p.Data, &p.ValorEntradas, &p.ValorSaidas); err != nil {
			return nil, fmt.Errorf("scanning projected flow: %w", err)
		}
		saldo += p.ValorEntradas - p.ValorSaidas
		p.Saldo = saldo
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (r *FinancialRepositoryPG) GetSaldoConta(ctx context.Context, contaID int64) (float64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	var saldo float64
	err = r.pool.QueryRow(ctx,
		`SELECT saldo_inicial FROM contas_bancarias WHERE id = $1 AND enterprise_id = $2`, contaID, empresa,
	).Scan(&saldo)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, errorsuc.NewNotFoundError(fmt.Sprintf("conta bancária %d não encontrada", contaID))
		}
		return 0, fmt.Errorf("getting saldo conta %d: %w", contaID, err)
	}

	// O movimento segue a conta: `fluxo_caixa` não tem coluna de empresa, e a
	// posse dele é a da conta bancária que ele movimenta.
	var movSum float64
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(CASE WHEN tipo = 'ENTRADA' THEN valor ELSE -valor END), 0)
		 FROM fluxo_caixa WHERE conta_bancaria_id = $1 AND conciliado = true`, contaID,
	).Scan(&movSum)
	if err == nil {
		saldo += movSum
	}

	return saldo, nil
}

func (r *FinancialRepositoryPG) GetSaldoConsolidado(ctx context.Context) (float64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	var saldo float64
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(saldo_inicial), 0) FROM contas_bancarias WHERE is_active = true AND enterprise_id = $1`, empresa,
	).Scan(&saldo)
	if err != nil {
		return 0, fmt.Errorf("getting saldo consolidado: %w", err)
	}

	// Somar `fluxo_caixa` inteiro somaria o caixa das outras empresas ao saldo
	// desta. O movimento entra pela conta bancária a que pertence.
	var movSum float64
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(CASE WHEN f.tipo = 'ENTRADA' THEN f.valor ELSE -f.valor END), 0)
		 FROM fluxo_caixa f
		 JOIN contas_bancarias cb ON cb.id = f.conta_bancaria_id AND cb.enterprise_id = $1
		 WHERE f.conciliado = true`, empresa,
	).Scan(&movSum)
	if err == nil {
		saldo += movSum
	}

	return saldo, nil
}

func (r *FinancialRepositoryPG) MarcarConciliado(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE fluxo_caixa SET conciliado = true WHERE id = $1 AND enterprise_id = $2`, id, empresa)
	if err != nil {
		return fmt.Errorf("marcando conciliado fluxo %d: %w", id, err)
	}
	return nil
}

// ---------- Tax Assessment ----------

func (r *FinancialRepositoryPG) CreateTaxAssessment(ctx context.Context, t *entity.TaxAssessment) (*entity.TaxAssessment, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx,
		`INSERT INTO tax_assessments
			(imposto, competencia, debitos, creditos, saldo_devedor, saldo_credor, status, cp_id, data_vencimento, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING id, created_at, updated_at`,
		t.Imposto, t.Competencia,
		t.Debitos.InexactFloat64(), t.Creditos.InexactFloat64(),
		t.SaldoDevedor.InexactFloat64(), t.SaldoCredor.InexactFloat64(),
		string(t.Status), t.CpID, t.DataVencimento, empresa,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating tax assessment: %w", err)
	}
	return t, nil
}

func (r *FinancialRepositoryPG) GetTaxAssessment(ctx context.Context, imposto, competencia string) (*entity.TaxAssessment, error) {
	var t entity.TaxAssessment
	var debitos, creditos, saldoDevedor, saldoCredor float64
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx,
		`SELECT id, imposto, competencia, debitos, creditos, saldo_devedor, saldo_credor,
		        status, cp_id, data_vencimento, created_at, updated_at
		 FROM tax_assessments WHERE imposto = $1 AND competencia = $2 AND enterprise_id = $3`,
		imposto, competencia, empresa,
	).Scan(&t.ID, &t.Imposto, &t.Competencia,
		&debitos, &creditos, &saldoDevedor, &saldoCredor,
		&t.Status, &t.CpID, &t.DataVencimento, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("apuração de %s da competência %s não encontrada", imposto, competencia))
		}
		return nil, fmt.Errorf("getting tax assessment: %w", err)
	}
	t.Debitos = decimal.NewFromFloat(debitos)
	t.Creditos = decimal.NewFromFloat(creditos)
	t.SaldoDevedor = decimal.NewFromFloat(saldoDevedor)
	t.SaldoCredor = decimal.NewFromFloat(saldoCredor)
	return &t, nil
}

func (r *FinancialRepositoryPG) ListTaxAssessments(ctx context.Context, competencia string) ([]*entity.TaxAssessment, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, imposto, competencia, debitos, creditos, saldo_devedor, saldo_credor,
		        status, cp_id, data_vencimento, created_at, updated_at
		 FROM tax_assessments WHERE competencia = $1 AND enterprise_id = $2 ORDER BY imposto`, competencia, empresa)
	if err != nil {
		return nil, fmt.Errorf("listing tax assessments: %w", err)
	}
	defer rows.Close()

	var out []*entity.TaxAssessment
	for rows.Next() {
		var t entity.TaxAssessment
		var debitos, creditos, saldoDevedor, saldoCredor float64
		if err := rows.Scan(&t.ID, &t.Imposto, &t.Competencia,
			&debitos, &creditos, &saldoDevedor, &saldoCredor,
			&t.Status, &t.CpID, &t.DataVencimento, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning tax assessment: %w", err)
		}
		t.Debitos = decimal.NewFromFloat(debitos)
		t.Creditos = decimal.NewFromFloat(creditos)
		t.SaldoDevedor = decimal.NewFromFloat(saldoDevedor)
		t.SaldoCredor = decimal.NewFromFloat(saldoCredor)
		out = append(out, &t)
	}
	return out, rows.Err()
}

// ---------- Fiscal Data for Tax Assessment ----------

// GetFiscalDebits soma os impostos destacados nas NF-e emitidas e autorizadas
// no mês (a devolução de compra inclusive: ela debita o imposto que a entrada
// creditou), pela data de emissão.
func (r *FinancialRepositoryPG) GetFiscalDebits(ctx context.Context, competencia string) (map[string]decimal.Decimal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var icms, ipi, pis, cofins decimal.Decimal
	if err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(valor_icms),0), COALESCE(SUM(valor_ipi),0), COALESCE(SUM(valor_pis),0), COALESCE(SUM(valor_cofins),0)
		   FROM fiscal_exits
		  WHERE enterprise_id = $2 AND status = 'AUTHORIZED' AND to_char(data_emissao, 'MM/YYYY') = $1`, competencia, empresa).
		Scan(&icms, &ipi, &pis, &cofins); err != nil {
		return nil, fmt.Errorf("getting fiscal debits: %w", err)
	}
	return map[string]decimal.Decimal{"ICMS": icms, "IPI": ipi, "PIS": pis, "COFINS": cofins}, nil
}

// GetFiscalCredits soma os créditos que a empresa efetivamente toma no mês — a
// mesma regra da EFD e da contabilização: só os itens com crédito
// (gera_credito_*) das notas de entrada aprovadas, pela data de entrada, mais
// o ICMS dos CT-e de frete lançados com crédito. Imposto destacado sem direito
// a crédito (uso e consumo, por exemplo) não abate o débito.
func (r *FinancialRepositoryPG) GetFiscalCredits(ctx context.Context, competencia string) (map[string]decimal.Decimal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var icms, ipi, pis, cofins, icmsFrete decimal.Decimal
	if err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(CASE WHEN fi.gera_credito_icms THEN fi.valor_icms ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN fi.gera_credito_ipi THEN fi.valor_ipi ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN fi.gera_credito_pis THEN fi.valor_pis ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN fi.gera_credito_cofins THEN fi.valor_cofins ELSE 0 END),0)
		   FROM fiscal_entry_items fi
		   JOIN fiscal_entries fe ON fe.id = fi.fiscal_entry_id
		  WHERE fe.enterprise_id = $2 AND fe.is_active AND fe.status IN ('APPROVED','WRITTEN_OFF')
		    AND to_char(fe.data_entrada, 'MM/YYYY') = $1`, competencia, empresa).
		Scan(&icms, &ipi, &pis, &cofins); err != nil {
		return nil, fmt.Errorf("getting fiscal credits: %w", err)
	}
	if err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(valor_icms),0) FROM fiscal_freight_documents
		  WHERE enterprise_id = $2 AND status = 'LANCADO' AND credita_icms AND to_char(lancado_em::date, 'MM/YYYY') = $1`, competencia, empresa).
		Scan(&icmsFrete); err != nil {
		return nil, fmt.Errorf("getting freight ICMS credits: %w", err)
	}
	return map[string]decimal.Decimal{"ICMS": icms.Add(icmsFrete), "IPI": ipi, "PIS": pis, "COFINS": cofins}, nil
}

func (r *FinancialRepositoryPG) GetFiscalConfig(ctx context.Context) (*fiscalEntity.FiscalConfig, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var cfg fiscalEntity.FiscalConfig
	err = r.pool.QueryRow(ctx,
		`SELECT id, enterprise_id, cnpj_empresa, razao_social, COALESCE(trade_name,''), COALESCE(email,''), ie_empresa, regime_tributario, uf_empresa,
		        icms_interno_aliquota, icms_diferimento_percentual,
		        focus_nfe_token, focus_nfe_ambiente, juros_mes, multa_atraso,
		        vencimento_icms_dia, vencimento_ipi_dia, vencimento_pis_cofins_dia,
		        created_at, updated_at, updated_by
		 FROM public.fiscal_configs WHERE enterprise_id=$1`, enterpriseID,
	).Scan(&cfg.ID, &cfg.EnterpriseID, &cfg.CnpjEmpresa, &cfg.RazaoSocial, &cfg.TradeName, &cfg.Email, &cfg.IEEmpresa, &cfg.RegimeTributario, &cfg.UFEmpresa,
		&cfg.IcmsInternoAliquota, &cfg.IcmsDiferimentoPercentual,
		&cfg.FocusNfeToken, &cfg.FocusNfeAmbiente, &cfg.JurosMes, &cfg.MultaAtraso,
		&cfg.VencimentoIcmsDia, &cfg.VencimentoIPIDia, &cfg.VencimentoPisCofinsDia,
		&cfg.CreatedAt, &cfg.UpdatedAt, &cfg.UpdatedBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errorsuc.NewNotFoundError("parametrização fiscal não encontrada")
		}
		return nil, fmt.Errorf("getting fiscal config: %w", err)
	}
	return &cfg, nil
}

// ---------- scan helpers ----------

func (r *FinancialRepositoryPG) scanContaPagarRow(row pgx.Row) (*entity.ContaPagar, error) {
	var c entity.ContaPagar
	var valorBruto, desconto, juros, multa, valorPago, valorAdiantAbatido float64
	err := row.Scan(
		&c.ID, &c.NumeroDocumento, &c.TipoDocumento, &c.FornecedorID, &c.FiscalEntryID, &c.PurchaseOrderID,
		&c.DataLancamento, &c.DataEmissao, &c.DataVencimento, &c.DataPagamento,
		&valorBruto, &desconto, &juros, &multa, &valorPago,
		&c.ParcelaNumero, &c.ParcelaTotal, &c.ParcelaPaiID,
		&c.ContaBancariaID, &c.FormaPagamento,
		&c.PlanoContasID, &c.CentroCustoID,
		&c.StatusAprovacao, &c.AprovadoPor, &c.DataAprovacao, &c.MotivoRejeicao,
		&c.Status, &c.AdiantamentoID, &valorAdiantAbatido,
		&c.ComprovantePath, &c.Observacao,
		&c.IsActive, &c.CriadoPor, &c.BaixadoPor, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errorsuc.NewNotFoundError("conta a pagar não encontrada")
		}
		return nil, fmt.Errorf("scanning conta pagar: %w", err)
	}
	c.ValorBruto = decimal.NewFromFloat(valorBruto)
	c.Desconto = decimal.NewFromFloat(desconto)
	c.Juros = decimal.NewFromFloat(juros)
	c.Multa = decimal.NewFromFloat(multa)
	c.ValorPago = decimal.NewFromFloat(valorPago)
	c.ValorAdiantamentoAbatido = decimal.NewFromFloat(valorAdiantAbatido)
	return &c, nil
}

func (r *FinancialRepositoryPG) scanContaPagar(rows pgx.Rows) (*entity.ContaPagar, error) {
	var c entity.ContaPagar
	var valorBruto, desconto, juros, multa, valorPago, valorAdiantAbatido float64
	err := rows.Scan(
		&c.ID, &c.NumeroDocumento, &c.TipoDocumento, &c.FornecedorID, &c.FiscalEntryID, &c.PurchaseOrderID,
		&c.DataLancamento, &c.DataEmissao, &c.DataVencimento, &c.DataPagamento,
		&valorBruto, &desconto, &juros, &multa, &valorPago,
		&c.ParcelaNumero, &c.ParcelaTotal, &c.ParcelaPaiID,
		&c.ContaBancariaID, &c.FormaPagamento,
		&c.PlanoContasID, &c.CentroCustoID,
		&c.StatusAprovacao, &c.AprovadoPor, &c.DataAprovacao, &c.MotivoRejeicao,
		&c.Status, &c.AdiantamentoID, &valorAdiantAbatido,
		&c.ComprovantePath, &c.Observacao,
		&c.IsActive, &c.CriadoPor, &c.BaixadoPor, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning conta pagar: %w", err)
	}
	c.ValorBruto = decimal.NewFromFloat(valorBruto)
	c.Desconto = decimal.NewFromFloat(desconto)
	c.Juros = decimal.NewFromFloat(juros)
	c.Multa = decimal.NewFromFloat(multa)
	c.ValorPago = decimal.NewFromFloat(valorPago)
	c.ValorAdiantamentoAbatido = decimal.NewFromFloat(valorAdiantAbatido)
	return &c, nil
}

func (r *FinancialRepositoryPG) scanContaReceberRow(row pgx.Row) (*entity.ContaReceber, error) {
	var c entity.ContaReceber
	var valorBruto, desconto, juros, multa, valorRecebido float64
	err := row.Scan(
		&c.ID, &c.NumeroDocumento, &c.ClienteID, &c.FiscalExitID, &c.SalesOrderID, &c.FornecedorID,
		&c.DataLancamento, &c.DataEmissao, &c.DataVencimento, &c.DataRecebimento,
		&valorBruto, &desconto, &juros, &multa, &valorRecebido,
		&c.ParcelaNumero, &c.ParcelaTotal, &c.ParcelaPaiID,
		&c.ContaBancariaID, &c.FormaPagamento,
		&c.NossoNumero, &c.LinhaDigitavel, &c.CodigoBarras, &c.ChavePixGerada,
		&c.PlanoContasID, &c.CentroCustoID,
		&c.Status, &c.EmProtesto,
		&c.IsActive, &c.CriadoPor, &c.BaixadoPor, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errorsuc.NewNotFoundError("conta a receber não encontrada")
		}
		return nil, fmt.Errorf("scanning conta receber: %w", err)
	}
	c.ValorBruto = decimal.NewFromFloat(valorBruto)
	c.Desconto = decimal.NewFromFloat(desconto)
	c.Juros = decimal.NewFromFloat(juros)
	c.Multa = decimal.NewFromFloat(multa)
	c.ValorRecebido = decimal.NewFromFloat(valorRecebido)
	return &c, nil
}

func (r *FinancialRepositoryPG) scanContaReceber(rows pgx.Rows) (*entity.ContaReceber, error) {
	var c entity.ContaReceber
	var valorBruto, desconto, juros, multa, valorRecebido float64
	err := rows.Scan(
		&c.ID, &c.NumeroDocumento, &c.ClienteID, &c.FiscalExitID, &c.SalesOrderID, &c.FornecedorID,
		&c.DataLancamento, &c.DataEmissao, &c.DataVencimento, &c.DataRecebimento,
		&valorBruto, &desconto, &juros, &multa, &valorRecebido,
		&c.ParcelaNumero, &c.ParcelaTotal, &c.ParcelaPaiID,
		&c.ContaBancariaID, &c.FormaPagamento,
		&c.NossoNumero, &c.LinhaDigitavel, &c.CodigoBarras, &c.ChavePixGerada,
		&c.PlanoContasID, &c.CentroCustoID,
		&c.Status, &c.EmProtesto,
		&c.IsActive, &c.CriadoPor, &c.BaixadoPor, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning conta receber: %w", err)
	}
	c.ValorBruto = decimal.NewFromFloat(valorBruto)
	c.Desconto = decimal.NewFromFloat(desconto)
	c.Juros = decimal.NewFromFloat(juros)
	c.Multa = decimal.NewFromFloat(multa)
	c.ValorRecebido = decimal.NewFromFloat(valorRecebido)
	return &c, nil
}

// ---------- UpsertTaxAssessmentCredito ----------

func (r *FinancialRepositoryPG) UpsertTaxAssessmentCredito(ctx context.Context, t *entity.TaxAssessment) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	// O alvo do ON CONFLICT acompanha a chave única, que passou a incluir a
	// empresa (migração 362). Sem isso, a segunda empresa a apurar a mesma
	// competência somaria o crédito no registro da primeira.
	_, err = r.pool.Exec(ctx,
		`INSERT INTO tax_assessments (imposto, competencia, debitos, creditos, saldo_devedor, saldo_credor, status, enterprise_id)
		 VALUES ($1, $2, 0, $3, 0, $3, 'APURAR', $4)
		 ON CONFLICT (enterprise_id, imposto, competencia) DO UPDATE SET
		     creditos = tax_assessments.creditos + EXCLUDED.creditos,
		     updated_at = NOW()`,
		t.Imposto, t.Competencia, t.Creditos.InexactFloat64(), empresa)
	if err != nil {
		return fmt.Errorf("upserting tax assessment credito: %w", err)
	}
	return nil
}

// ---------- CancelContasReceberByFiscalExit ----------

func (r *FinancialRepositoryPG) CancelContasReceberByFiscalExit(ctx context.Context, fiscalExitID int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE contas_receber SET status = 'CANCELADO', is_active = false, updated_at = NOW()
		 WHERE fiscal_exit_id = $1 AND status IN ('PENDENTE', 'APROVADO', 'VENCIDO') AND enterprise_id = $2`, fiscalExitID, empresa)
	if err != nil {
		return fmt.Errorf("cancelling contas receber for exit %d: %w", fiscalExitID, err)
	}
	return nil
}

// ---------- Reports ----------

func (r *FinancialRepositoryPG) GetLivroEntradas(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT fe.id, fe.data_entrada, fe.numero_nf, fe.serie, fe.razao_social_emitente,
		        fei.cfop, fe.valor_produtos, fe.valor_ipi, fe.valor_icms, fe.valor_pis, fe.valor_cofins, fe.valor_total
		 FROM fiscal_entries fe
		 LEFT JOIN fiscal_entry_items fei ON fei.fiscal_entry_id = fe.id
		 WHERE fe.data_entrada BETWEEN $1 AND $2 AND fe.status = 'APPROVED' AND fe.enterprise_id = $3
		 ORDER BY fe.data_entrada, fe.numero_nf`, startDate, endDate, empresa)
	if err != nil {
		return nil, fmt.Errorf("livro entradas: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"id", "data_entrada", "numero_nf", "serie", "emitente", "cfop", "valor_produtos", "valor_ipi", "valor_icms", "valor_pis", "valor_cofins", "valor_total"})
}

func (r *FinancialRepositoryPG) GetLivroSaidas(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT fe.id, fe.data_emissao, fe.numero_nf, fe.serie, fe.razao_social_destinatario,
		        fei.cfop, fe.valor_produtos, fe.valor_ipi, fe.valor_icms, fe.valor_pis, fe.valor_cofins, fe.valor_total
		 FROM fiscal_exits fe
		 LEFT JOIN fiscal_exit_items fei ON fei.fiscal_exit_id = fe.id
		 WHERE fe.data_emissao BETWEEN $1 AND $2 AND fe.status = 'AUTHORIZED' AND fe.enterprise_id = $3
		 ORDER BY fe.data_emissao, fe.numero_nf`, startDate, endDate, empresa)
	if err != nil {
		return nil, fmt.Errorf("livro saidas: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"id", "data_emissao", "numero_nf", "serie", "destinatario", "cfop", "valor_produtos", "valor_ipi", "valor_icms", "valor_pis", "valor_cofins", "valor_total"})
}

func (r *FinancialRepositoryPG) GetImpostosSaidas(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT fe.numero_nf, fe.data_emissao, fe.razao_social_destinatario, fei.cfop,
		        fei.base_icms, fei.valor_icms, fei.base_ipi, fei.valor_ipi, fei.valor_pis, fei.valor_cofins
		 FROM fiscal_exits fe
		 JOIN fiscal_exit_items fei ON fei.fiscal_exit_id = fe.id
		 WHERE fe.data_emissao BETWEEN $1 AND $2 AND fe.status = 'AUTHORIZED' AND fe.enterprise_id = $3
		 ORDER BY fe.data_emissao`, startDate, endDate, empresa)
	if err != nil {
		return nil, fmt.Errorf("impostos saidas: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"numero_nf", "data", "cliente", "cfop", "base_icms", "valor_icms", "base_ipi", "valor_ipi", "valor_pis", "valor_cofins"})
}

func (r *FinancialRepositoryPG) GetImpostosEntradas(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT fe.numero_nf, fe.data_entrada, fe.razao_social_emitente, fei.cfop,
		        fei.base_icms, fei.valor_icms, fei.base_ipi, fei.valor_ipi, fei.valor_pis, fei.valor_cofins
		 FROM fiscal_entries fe
		 JOIN fiscal_entry_items fei ON fei.fiscal_entry_id = fe.id
		 WHERE fe.data_entrada BETWEEN $1 AND $2 AND fe.status = 'APPROVED' AND fe.enterprise_id = $3
		 ORDER BY fe.data_entrada`, startDate, endDate, empresa)
	if err != nil {
		return nil, fmt.Errorf("impostos entradas: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"numero_nf", "data", "fornecedor", "cfop", "base_icms", "valor_icms", "base_ipi", "valor_ipi", "valor_pis", "valor_cofins"})
}

func (r *FinancialRepositoryPG) GetDRE(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var receitaBruta, impostosVendas float64
	if err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(valor_produtos),0), COALESCE(SUM(valor_icms + valor_ipi + valor_pis + valor_cofins),0)
		 FROM fiscal_exits WHERE data_emissao BETWEEN $1 AND $2 AND status = 'AUTHORIZED' AND enterprise_id = $3`,
		startDate, endDate, empresa).Scan(&receitaBruta, &impostosVendas); err != nil {
		return nil, fmt.Errorf("DRE receita: %w", err)
	}

	var despesas float64
	if err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(valor_bruto - COALESCE(valor_pago,0)),0)
		 FROM contas_pagar WHERE data_vencimento BETWEEN $1 AND $2 AND status IN ('PAGO','CANCELADO') AND enterprise_id = $3`,
		startDate, endDate, empresa).Scan(&despesas); err != nil {
		return nil, fmt.Errorf("DRE despesas: %w", err)
	}

	receitaLiquida := receitaBruta - impostosVendas
	resultado := receitaLiquida - despesas

	return map[string]interface{}{
		"receita_bruta":     receitaBruta,
		"impostos_vendas":   impostosVendas,
		"receita_liquida":   receitaLiquida,
		"despesas":          despesas,
		"resultado_liquido": resultado,
		"periodo_inicio":    startDate.Format("2006-01-02"),
		"periodo_fim":       endDate.Format("2006-01-02"),
	}, nil
}

func (r *FinancialRepositoryPG) GetAgingReceberDetalhado(ctx context.Context) ([]map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, numero_documento, cliente_id, data_vencimento, valor_bruto,
		 CASE
		     WHEN data_vencimento >= CURRENT_DATE THEN 'A vencer'
		     WHEN data_vencimento >= CURRENT_DATE - 30 THEN '1-30 dias'
		     WHEN data_vencimento >= CURRENT_DATE - 60 THEN '31-60 dias'
		     WHEN data_vencimento >= CURRENT_DATE - 90 THEN '61-90 dias'
		     WHEN data_vencimento >= CURRENT_DATE - 180 THEN '91-180 dias'
		     ELSE '+180 dias'
		 END AS faixa
		 FROM contas_receber
		 WHERE status IN ('PENDENTE','VENCIDO') AND is_active = true AND enterprise_id = $1
		 ORDER BY data_vencimento`, empresa)
	if err != nil {
		return nil, fmt.Errorf("aging receber: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"id", "numero_documento", "cliente_id", "data_vencimento", "valor_bruto", "faixa"})
}

func (r *FinancialRepositoryPG) GetAgingPagarDetalhado(ctx context.Context) ([]map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, numero_documento, fornecedor_id, data_vencimento, valor_bruto,
		 CASE
		     WHEN data_vencimento >= CURRENT_DATE THEN 'A vencer'
		     WHEN data_vencimento >= CURRENT_DATE - 30 THEN '1-30 dias'
		     WHEN data_vencimento >= CURRENT_DATE - 60 THEN '31-60 dias'
		     WHEN data_vencimento >= CURRENT_DATE - 90 THEN '61-90 dias'
		     WHEN data_vencimento >= CURRENT_DATE - 180 THEN '91-180 dias'
		     ELSE '+180 dias'
		 END AS faixa
		 FROM contas_pagar
		 WHERE status IN ('PENDENTE','APROVADO','VENCIDO') AND is_active = true AND enterprise_id = $1
		 ORDER BY data_vencimento`, empresa)
	if err != nil {
		return nil, fmt.Errorf("aging pagar: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"id", "numero_documento", "fornecedor_id", "data_vencimento", "valor_bruto", "faixa"})
}

func (r *FinancialRepositoryPG) GetExtratoPorFornecedor(ctx context.Context, fornecedorID int64) ([]map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT numero_documento, data_emissao, data_vencimento, data_pagamento,
		        valor_bruto, desconto, juros, multa, COALESCE(valor_pago,0) as valor_pago, status
		 FROM contas_pagar WHERE fornecedor_id = $1 AND is_active = true AND enterprise_id = $2 ORDER BY data_emissao`, fornecedorID, empresa)
	if err != nil {
		return nil, fmt.Errorf("extrato fornecedor: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"numero_documento", "data_emissao", "data_vencimento", "data_pagamento", "valor_bruto", "desconto", "juros", "multa", "valor_pago", "status"})
}

func (r *FinancialRepositoryPG) GetExtratoPorCliente(ctx context.Context, clienteID int64) ([]map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT numero_documento, data_emissao, data_vencimento, data_recebimento,
		        valor_bruto, desconto, juros, multa, COALESCE(valor_recebido,0) as valor_recebido, status
		 FROM contas_receber WHERE cliente_id = $1 AND is_active = true AND enterprise_id = $2 ORDER BY data_emissao`, clienteID, empresa)
	if err != nil {
		return nil, fmt.Errorf("extrato cliente: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"numero_documento", "data_emissao", "data_vencimento", "data_recebimento", "valor_bruto", "desconto", "juros", "multa", "valor_recebido", "status"})
}

// scanToMaps converts pgx rows to []map[string]interface{} using provided column names.
func scanToMaps(rows pgx.Rows, cols []string) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		row := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			if i < len(vals) {
				row[col] = vals[i]
			}
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// ---------- BaixarContaPagarAtomico ----------

func (r *FinancialRepositoryPG) BaixarContaPagarAtomico(ctx context.Context, id int64, params repository.BaixaParams, fc entity.FluxoCaixa, valorOriginal decimal.Decimal, contaBancariaID int64, lote *contabilizacao.Lote) error {
	return r.baixarAtomico(ctx, ladoPagar, id, params, fc, valorOriginal, contaBancariaID, lote)
}

// ---------- BaixarContaReceberAtomico ----------

func (r *FinancialRepositoryPG) BaixarContaReceberAtomico(ctx context.Context, id int64, params repository.BaixaParams, fc entity.FluxoCaixa, valorOriginal decimal.Decimal, contaBancariaID int64, lote *contabilizacao.Lote) error {
	return r.baixarAtomico(ctx, ladoReceber, id, params, fc, valorOriginal, contaBancariaID, lote)
}

type ladoBaixa struct {
	tabela, colunaPago, colunaData, statusQuitado, colunaFluxo, sinalSaldo string
	statusAbertos                                                          []string
	copiaSaldo                                                             string // colunas copiadas para o título do saldo
	temRateio                                                              bool
}

var (
	ladoPagar = ladoBaixa{
		tabela: "contas_pagar", colunaPago: "valor_pago", colunaData: "data_pagamento", statusQuitado: "PAGO",
		colunaFluxo: "contas_pagar_id", sinalSaldo: "-", statusAbertos: []string{"PENDENTE", "APROVADO", "VENCIDO"},
		copiaSaldo: `tipo_documento, fornecedor_id, fornecedor_cnpj, fiscal_entry_id, purchase_order_id, data_emissao, data_vencimento,
		             parcela_numero, parcela_total, forma_pagamento, plano_contas_id, centro_custo_id, observacao, retencao_tipo,
		             status_aprovacao, criado_por, enterprise_id`,
		temRateio: true,
	}
	ladoReceber = ladoBaixa{
		tabela: "contas_receber", colunaPago: "valor_recebido", colunaData: "data_recebimento", statusQuitado: "RECEBIDO",
		colunaFluxo: "contas_receber_id", sinalSaldo: "+", statusAbertos: []string{"PENDENTE", "APROVADO", "VENCIDO"},
		copiaSaldo: `cliente_id, fiscal_exit_id, sales_order_id, data_emissao, data_vencimento, parcela_numero, parcela_total,
		             forma_pagamento, plano_contas_id, centro_custo_id, condicao_pagamento_id, criado_por, enterprise_id`,
	}
)

// baixarAtomico quita o título numa transação: trava o título (dois cliques
// não pagam duas vezes), grava o pagamento, cria o título do saldo na baixa
// parcial (com empresa, fornecedor/cliente, plano, centro de custo e o rateio
// proporcional — antes o INSERT não levava a empresa e a baixa parcial falhava),
// lança o fluxo de caixa, movimenta o saldo bancário e, se configurado, grava
// a contabilização na mesma transação.
func (r *FinancialRepositoryPG) baixarAtomico(ctx context.Context, lado ladoBaixa, id int64, params repository.BaixaParams, fc entity.FluxoCaixa, valorOriginal decimal.Decimal, contaBancariaID int64, lote *contabilizacao.Lote) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	var bruto decimal.Decimal
	err = tx.QueryRow(ctx, `SELECT status, valor_bruto FROM `+lado.tabela+` WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, id, empresa).Scan(&status, &bruto)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("título %d não encontrado", id))
	}
	if err != nil {
		return err
	}
	aberto := false
	for _, s := range lado.statusAbertos {
		if status == s {
			aberto = true
		}
	}
	if !aberto {
		return errorsuc.NewConflictError(fmt.Sprintf("o título %d já está %s", id, status))
	}

	principal := decimal.NewFromFloat(params.ValorPago).Round(2)
	desconto := decimal.NewFromFloat(params.Desconto).Round(2)
	quitado := principal.Add(desconto)
	saldo := valorOriginal.Sub(quitado).Round(2)
	parcial := saldo.IsPositive()

	if parcial {
		// O título baixado fica com o que foi quitado; o saldo vira outro título.
		if _, err = tx.Exec(ctx,
			`INSERT INTO `+lado.tabela+` (numero_documento, valor_bruto, desconto, juros, multa, `+lado.colunaPago+`,
			     data_lancamento, status, is_active, parcela_pai_id, `+lado.copiaSaldo+`)
			 SELECT left(numero_documento || '/P', 60), $1, 0, 0, 0, 0, $3::date, 'PENDENTE', TRUE, id, `+lado.copiaSaldo+`
			   FROM `+lado.tabela+` WHERE id = $2`, saldo, id, params.DataPagamento); err != nil {
			return fmt.Errorf("criando o título do saldo: %w", err)
		}
		if lado.temRateio {
			if err = reparteRateio(ctx, tx, empresa, id, bruto, saldo); err != nil {
				return err
			}
		}
	}
	novoBruto := bruto
	if parcial {
		novoBruto = bruto.Sub(saldo)
	}
	if _, err = tx.Exec(ctx,
		`UPDATE `+lado.tabela+` SET status=$1, `+lado.colunaData+`=$2, `+lado.colunaPago+`=$3, juros=$4, multa=$5,
		     desconto=COALESCE(desconto,0)+$6, valor_bruto=$7, conta_bancaria_id=$8, baixado_por=$9, updated_at=NOW()
		  WHERE id=$10 AND enterprise_id=$11`,
		lado.statusQuitado, params.DataPagamento, principal, params.Juros, params.Multa, desconto, novoBruto,
		params.ContaBancariaID, params.BaixadoPor, id, empresa); err != nil {
		return fmt.Errorf("baixando o título: %w", err)
	}

	// ⚠️ `enterprise_id` é NOT NULL desde o isolamento do financeiro (migração
	// 362): sem ele o INSERT falha e a baixa não acontece.
	if _, err = tx.Exec(ctx,
		`INSERT INTO fluxo_caixa (data, tipo, valor, conta_bancaria_id, `+lado.colunaFluxo+`, descricao, conciliado, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		fc.Data, string(fc.Tipo), fc.Valor.InexactFloat64(), fc.ContaBancariaID, id, fc.Descricao, false, empresa); err != nil {
		return fmt.Errorf("creating fluxo caixa: %w", err)
	}

	// A conta movimentada tem de ser da empresa autenticada.
	tagSaldo, err := tx.Exec(ctx,
		`UPDATE contas_bancarias SET saldo_inicial = saldo_inicial `+lado.sinalSaldo+` $1, updated_at = NOW() WHERE id = $2 AND enterprise_id = $3`,
		fc.Valor.InexactFloat64(), contaBancariaID, empresa)
	if err != nil {
		return fmt.Errorf("updating saldo: %w", err)
	}
	if tagSaldo.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("conta bancária %d não encontrada", contaBancariaID))
	}

	if lote != nil {
		if err := journal.Gravar(ctx, tx, empresa, *lote); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// reparteRateio divide o rateio do título entre o que foi quitado e o saldo,
// na proporção (o saldo leva os centavos), para os relatórios por plano de
// contas continuarem fechando.
func reparteRateio(ctx context.Context, tx pgx.Tx, empresa, id int64, bruto, saldo decimal.Decimal) error {
	rows, err := tx.Query(ctx, `SELECT id, plano_contas_id, centro_custo_id, valor FROM contas_pagar_rateios WHERE conta_pagar_id=$1 AND enterprise_id=$2 ORDER BY id`, id, empresa)
	if err != nil {
		return err
	}
	type linha struct {
		id, plano int64
		cc        *int64
		valor     decimal.Decimal
	}
	var ls []linha
	for rows.Next() {
		var l linha
		if err := rows.Scan(&l.id, &l.plano, &l.cc, &l.valor); err != nil {
			rows.Close()
			return err
		}
		ls = append(ls, l)
	}
	rows.Close()
	if len(ls) == 0 || !bruto.IsPositive() {
		return nil
	}
	var novo int64
	if err := tx.QueryRow(ctx, `SELECT MAX(id) FROM contas_pagar WHERE parcela_pai_id=$1 AND enterprise_id=$2`, id, empresa).Scan(&novo); err != nil {
		return err
	}
	acumulado := decimal.Zero
	for i, l := range ls {
		parte := l.valor.Mul(saldo).Div(bruto).Round(2)
		if i == len(ls)-1 {
			parte = saldo.Sub(acumulado)
		}
		acumulado = acumulado.Add(parte)
		if parte.IsPositive() {
			if _, err := tx.Exec(ctx, `INSERT INTO contas_pagar_rateios (enterprise_id, conta_pagar_id, plano_contas_id, centro_custo_id, valor) VALUES ($1,$2,$3,$4,$5)`,
				empresa, novo, l.plano, l.cc, parte); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE contas_pagar_rateios SET valor=valor-$1 WHERE id=$2`, parte, l.id); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `DELETE FROM contas_pagar_rateios WHERE conta_pagar_id=$1 AND enterprise_id=$2 AND valor <= 0`, id, empresa)
	return err
}

// ---------- Reports R13-R19 ----------

func (r *FinancialRepositoryPG) GetProdutosVendidos(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT fei.item_code, COALESCE(CAST(i.code AS text),'') AS codigo, COALESCE(i.pdm_description_technique,'') AS descricao,
		        fei.ncm, SUM(fei.quantity) AS qtd_vendida,
		        ROUND(AVG(fei.unit_price)::numeric,2) AS preco_medio,
		        COALESCE(MAX(c.custo),0) AS custo_medio,
		        SUM(fei.total_price) AS valor_total,
		        ROUND((AVG(fei.unit_price) - COALESCE(MAX(c.custo),0))::numeric,2) AS margem_bruta
		 FROM fiscal_exit_items fei
		 JOIN fiscal_exits fe ON fe.id = fei.fiscal_exit_id
		 LEFT JOIN items i ON i.code = fei.item_code AND i.enterprise_id = fe.enterprise_id
		 LEFT JOIN LATERAL (SELECT (SELECT SUM(sb.total_cost) / NULLIF(SUM(sb.quantity),0) FROM stock_balances sb WHERE sb.item_code = fei.item_code AND sb.enterprise_id = $3) AS custo) c ON TRUE
		 WHERE fe.data_emissao BETWEEN $1 AND $2 AND fe.status = 'AUTHORIZED' AND fe.enterprise_id = $3
		 GROUP BY fei.item_code, i.code, i.pdm_description_technique, fei.ncm
		 ORDER BY valor_total DESC`, startDate, endDate, empresa)
	if err != nil {
		return nil, fmt.Errorf("produtos vendidos: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"item_code", "codigo", "descricao", "ncm", "qtd_vendida", "preco_medio", "custo_medio", "valor_total", "margem_bruta"})
}

func (r *FinancialRepositoryPG) GetProdutosProduzidos(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT po.item_code, COALESCE(CAST(i.code AS text),'') AS codigo, COALESCE(i.pdm_description_technique,'') AS descricao,
		        SUM(po.produced_qty) AS qtd_produzida,
		        COALESCE(MAX(c.custo),0) AS custo_unitario,
		        COALESCE(MAX(c.custo),0) * SUM(po.produced_qty) AS custo_total
		 FROM production_orders po
		 LEFT JOIN items i ON i.code = po.item_code AND i.enterprise_id = po.enterprise_id
		 LEFT JOIN LATERAL (SELECT (SELECT SUM(sb.total_cost) / NULLIF(SUM(sb.quantity),0) FROM stock_balances sb WHERE sb.item_code = po.item_code AND sb.enterprise_id = $3) AS custo) c ON TRUE
		 WHERE po.created_at BETWEEN $1 AND $2 AND po.status IN ('COMPLETED','CLOSED') AND po.enterprise_id = $3
		 GROUP BY po.item_code, i.code, i.pdm_description_technique
		 ORDER BY custo_total DESC`, startDate, endDate, empresa)
	if err != nil {
		return nil, fmt.Errorf("produtos produzidos: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"item_code", "codigo", "descricao", "qtd_produzida", "custo_unitario", "custo_total"})
}

func (r *FinancialRepositoryPG) GetHistoricoCustos(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT sb.item_code, COALESCE(CAST(i.code AS text),'') AS codigo, COALESCE(i.pdm_description_technique,'') AS descricao,
		        sb.avg_cost AS custo_medio_atual, sb.last_cost AS custo_ultima_compra,
		        COALESCE((SELECT MIN(unit_price) FROM stock_movements sm WHERE sm.item_code = sb.item_code AND sm.enterprise_id = $3 AND sm.created_at BETWEEN $1 AND $2 AND sm.movement_type='IN'),0) AS custo_minimo,
		        COALESCE((SELECT MAX(unit_price) FROM stock_movements sm WHERE sm.item_code = sb.item_code AND sm.enterprise_id = $3 AND sm.created_at BETWEEN $1 AND $2 AND sm.movement_type='IN'),0) AS custo_maximo
		 FROM stock_balances sb
		 LEFT JOIN items i ON i.code = sb.item_code AND i.enterprise_id = sb.enterprise_id
		 WHERE sb.enterprise_id = $3
		 ORDER BY sb.item_code`, startDate, endDate, empresa)
	if err != nil {
		return nil, fmt.Errorf("historico custos: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"item_code", "codigo", "descricao", "custo_medio_atual", "custo_ultima_compra", "custo_minimo", "custo_maximo"})
}

func (r *FinancialRepositoryPG) GetFichaTecnicaCusto(ctx context.Context, itemCode int64) ([]map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT s.child_code AS insumo_code, COALESCE(CAST(i.code AS text),'') AS insumo_codigo,
		        COALESCE(i.pdm_description_technique, i.name, '') AS insumo_descricao,
		        s.quantity AS qtd_por_unidade,
		        COALESCE(c.custo,0) AS custo_unitario,
		        s.quantity * COALESCE(c.custo,0) AS custo_total
		 FROM item_structures s
		 JOIN items pai ON pai.code = s.parent_code AND pai.enterprise_id = $2
		 LEFT JOIN items i ON i.code = s.child_code AND i.enterprise_id = $2
		 LEFT JOIN LATERAL (SELECT SUM(sb.total_cost) / NULLIF(SUM(sb.quantity),0) AS custo
		                      FROM stock_balances sb WHERE sb.item_code = s.child_code AND sb.enterprise_id = $2) c ON TRUE
		 WHERE s.parent_code = $1 AND s.is_active
		   AND (s.start_date IS NULL OR s.start_date <= CURRENT_DATE)
		   AND (s.end_date IS NULL OR s.end_date >= CURRENT_DATE)
		 ORDER BY custo_total DESC`, itemCode, empresa)
	if err != nil {
		return nil, fmt.Errorf("ficha tecnica: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"insumo_code", "insumo_codigo", "insumo_descricao", "qtd_por_unidade", "custo_unitario", "custo_total"})
}

func (r *FinancialRepositoryPG) GetCurvaABCClientes(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`WITH vendas AS (
		     SELECT razao_social_destinatario AS cliente, SUM(valor_total) AS total
		     FROM fiscal_exits
		     WHERE status = 'AUTHORIZED' AND data_emissao BETWEEN $1 AND $2 AND enterprise_id = $3
		     GROUP BY razao_social_destinatario
		 ), soma AS (SELECT SUM(total) AS grand_total FROM vendas)
		 SELECT cliente, total,
		        ROUND((total / grand_total * 100)::numeric, 2) AS pct,
		        ROUND(SUM(total) OVER (ORDER BY total DESC) / grand_total * 100::numeric, 2) AS pct_acum,
		        CASE WHEN SUM(total) OVER (ORDER BY total DESC) / grand_total <= 0.8 THEN 'A'
		             WHEN SUM(total) OVER (ORDER BY total DESC) / grand_total <= 0.95 THEN 'B'
		             ELSE 'C' END AS classe
		 FROM vendas, soma ORDER BY total DESC`, startDate, endDate, empresa)
	if err != nil {
		return nil, fmt.Errorf("curva abc clientes: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"cliente", "total", "pct", "pct_acum", "classe"})
}

func (r *FinancialRepositoryPG) GetCurvaABCProdutos(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`WITH vendas AS (
		     SELECT fei.item_code, COALESCE(CAST(i.code AS text),'') AS codigo, COALESCE(i.pdm_description_technique,'') AS descricao,
		            SUM(fei.total_price) AS total
		     FROM fiscal_exit_items fei
		     JOIN fiscal_exits fe ON fe.id = fei.fiscal_exit_id
		     LEFT JOIN items i ON i.code = fei.item_code AND i.enterprise_id = fe.enterprise_id
		     WHERE fe.status = 'AUTHORIZED' AND fe.data_emissao BETWEEN $1 AND $2 AND fe.enterprise_id = $3
		     GROUP BY fei.item_code, i.code, i.pdm_description_technique
		 ), soma AS (SELECT SUM(total) AS grand_total FROM vendas)
		 SELECT item_code, codigo, descricao, total,
		        ROUND((total / grand_total * 100)::numeric, 2) AS pct,
		        ROUND(SUM(total) OVER (ORDER BY total DESC) / grand_total * 100::numeric, 2) AS pct_acum,
		        CASE WHEN SUM(total) OVER (ORDER BY total DESC) / grand_total <= 0.8 THEN 'A'
		             WHEN SUM(total) OVER (ORDER BY total DESC) / grand_total <= 0.95 THEN 'B'
		             ELSE 'C' END AS classe
		 FROM vendas, soma ORDER BY total DESC`, startDate, endDate, empresa)
	if err != nil {
		return nil, fmt.Errorf("curva abc produtos: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"item_code", "codigo", "descricao", "total", "pct", "pct_acum", "classe"})
}

func (r *FinancialRepositoryPG) GetComprasPeriodo(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT fe.razao_social_emitente AS fornecedor, fei.item_code,
		        COALESCE(CAST(i.code AS text),'') AS codigo_produto, COALESCE(i.pdm_description_technique,'') AS descricao, fei.ncm,
		        SUM(fei.quantity) AS qtd, ROUND(AVG(fei.unit_price)::numeric,4) AS preco_unitario,
		        SUM(fei.total_price) AS valor_total, fe.numero_nf AS nfe
		 FROM fiscal_entry_items fei
		 JOIN fiscal_entries fe ON fe.id = fei.fiscal_entry_id
		 LEFT JOIN items i ON i.code = fei.item_code AND i.enterprise_id = fe.enterprise_id
		 WHERE fe.data_entrada BETWEEN $1 AND $2 AND fe.status = 'APPROVED' AND fe.enterprise_id = $3
		 GROUP BY fe.razao_social_emitente, fei.item_code, i.code, i.pdm_description_technique, fei.ncm, fe.numero_nf
		 ORDER BY fe.razao_social_emitente, valor_total DESC`, startDate, endDate, empresa)
	if err != nil {
		return nil, fmt.Errorf("compras periodo: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"fornecedor", "item_code", "codigo_produto", "descricao", "ncm", "qtd", "preco_unitario", "valor_total", "nfe"})
}

// ---------- DRE with CMV ----------

func (r *FinancialRepositoryPG) GetDREComCMV(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var receitaBruta, impostosVendas float64
	if err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(valor_produtos),0), COALESCE(SUM(valor_icms + valor_ipi + valor_pis + valor_cofins),0)
		 FROM fiscal_exits WHERE data_emissao BETWEEN $1 AND $2 AND status = 'AUTHORIZED' AND enterprise_id = $3`,
		startDate, endDate, empresa).Scan(&receitaBruta, &impostosVendas); err != nil {
		return nil, fmt.Errorf("DRE receita: %w", err)
	}

	// CMV: qty sold × avg_cost at time of sale (uses current avg_cost as best approximation)
	var cmv float64
	if err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(fei.quantity * COALESCE(c.custo, fei.unit_price * 0.6)), 0)
		 FROM fiscal_exit_items fei
		 JOIN fiscal_exits fe ON fe.id = fei.fiscal_exit_id
		 LEFT JOIN LATERAL (SELECT (SELECT SUM(sb.total_cost) / NULLIF(SUM(sb.quantity),0) FROM stock_balances sb WHERE sb.item_code = fei.item_code AND sb.enterprise_id = $3) AS custo) c ON TRUE
		 WHERE fe.data_emissao BETWEEN $1 AND $2 AND fe.status = 'AUTHORIZED' AND fe.enterprise_id = $3`,
		startDate, endDate, empresa).Scan(&cmv); err != nil {
		return nil, fmt.Errorf("DRE CMV: %w", err)
	}

	var despesasOperacionais float64
	if err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(valor_bruto - COALESCE(desconto,0)),0)
		 FROM contas_pagar
		 WHERE data_vencimento BETWEEN $1 AND $2 AND status IN ('PAGO') AND tipo_documento NOT IN ('IMPOSTO') AND enterprise_id = $3`,
		startDate, endDate, empresa).Scan(&despesasOperacionais); err != nil {
		return nil, fmt.Errorf("DRE despesas operacionais: %w", err)
	}

	var despesasFinanceiras float64
	if err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(juros + multa),0) FROM contas_pagar WHERE data_pagamento BETWEEN $1 AND $2 AND status='PAGO' AND enterprise_id = $3`,
		startDate, endDate, empresa).Scan(&despesasFinanceiras); err != nil {
		return nil, fmt.Errorf("DRE despesas financeiras: %w", err)
	}

	var receitasFinanceiras float64
	if err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(juros + multa),0) FROM contas_receber WHERE data_recebimento BETWEEN $1 AND $2 AND status='RECEBIDO' AND enterprise_id = $3`,
		startDate, endDate, empresa).Scan(&receitasFinanceiras); err != nil {
		return nil, fmt.Errorf("DRE receitas financeiras: %w", err)
	}

	receitaLiquida := receitaBruta - impostosVendas
	lucroBruto := receitaLiquida - cmv
	resultado := lucroBruto - despesasOperacionais - despesasFinanceiras + receitasFinanceiras

	return map[string]interface{}{
		"receita_bruta":         receitaBruta,
		"impostos_vendas":       impostosVendas,
		"receita_liquida":       receitaLiquida,
		"cmv":                   cmv,
		"lucro_bruto":           lucroBruto,
		"despesas_operacionais": despesasOperacionais,
		"despesas_financeiras":  despesasFinanceiras,
		"receitas_financeiras":  receitasFinanceiras,
		"resultado_liquido":     resultado,
		"periodo_inicio":        startDate.Format("2006-01-02"),
		"periodo_fim":           endDate.Format("2006-01-02"),
	}, nil
}

// ---------- OFX / Conciliacao Bancaria ----------

// contaDaEmpresa recusa um id de conta bancária que não seja da empresa
// autenticada. O extrato não tem coluna de empresa: a posse dele é a da conta,
// e é aqui que ela é conferida antes de gravar ou ler.
func (r *FinancialRepositoryPG) contaDaEmpresa(ctx context.Context, contaID int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var existe bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM contas_bancarias WHERE id = $1 AND enterprise_id = $2)`,
		contaID, empresa).Scan(&existe); err != nil {
		return fmt.Errorf("checking conta bancaria %d: %w", contaID, err)
	}
	if !existe {
		return errorsuc.NewNotFoundError(fmt.Sprintf("conta bancária %d não encontrada", contaID))
	}
	return nil
}

func (r *FinancialRepositoryPG) SaveExtratoItem(ctx context.Context, contaID int64, data time.Time, valor float64, tipo, descricao, fitid, hash string) (bool, error) {
	if err := r.contaDaEmpresa(ctx, contaID); err != nil {
		return false, err
	}
	tag, err := r.pool.Exec(ctx,
		`INSERT INTO extrato_bancario (conta_bancaria_id, data_transacao, valor, tipo, descricao, fitid, extrato_hash)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (extrato_hash) DO NOTHING`,
		contaID, data, valor, tipo, descricao, fitid, hash)
	if err != nil {
		return false, fmt.Errorf("gravando lançamento do extrato: %w", err)
	}
	// ON CONFLICT DO NOTHING não é erro: zero linha afetada significa que o
	// lançamento já estava importado.
	return tag.RowsAffected() > 0, nil
}

func (r *FinancialRepositoryPG) GetExtratoPendente(ctx context.Context, contaID int64) ([]map[string]interface{}, error) {
	if err := r.contaDaEmpresa(ctx, contaID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, data_transacao, valor, tipo, descricao, fitid, extrato_hash, conciliado
		 FROM extrato_bancario WHERE conta_bancaria_id = $1 ORDER BY data_transacao`, contaID)
	if err != nil {
		return nil, fmt.Errorf("getting extrato pendente: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"id", "data_transacao", "valor", "tipo", "descricao", "fitid", "extrato_hash", "conciliado"})
}

func (r *FinancialRepositoryPG) ConciliarExtrato(ctx context.Context, extratoID, fluxoID int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx,
		// O extrato herda a posse da conta bancária; o fluxo tem empresa própria.
		`UPDATE extrato_bancario e SET conciliado=true, fluxo_caixa_id=$1
		 FROM contas_bancarias cb
		 WHERE e.id=$2 AND cb.id = e.conta_bancaria_id AND cb.enterprise_id=$3`,
		fluxoID, extratoID, empresa)
	if err != nil {
		return fmt.Errorf("updating extrato: %w", err)
	}
	_, err = tx.Exec(ctx,
		`UPDATE fluxo_caixa SET conciliado=true WHERE id=$1 AND enterprise_id=$2`, fluxoID, empresa)
	if err != nil {
		return fmt.Errorf("marking fluxo conciliado: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *FinancialRepositoryPG) AutoMatchExtrato(ctx context.Context, contaID int64) (int, error) {
	if err := r.contaDaEmpresa(ctx, contaID); err != nil {
		return 0, err
	}
	empresaAuto, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	// Try to match extrato entries to fluxo_caixa by exact value and date ±3 days
	rows, err := r.pool.Query(ctx,
		`SELECT id, data_transacao, valor, tipo FROM extrato_bancario
		 WHERE conta_bancaria_id=$1 AND conciliado=false`, contaID)
	if err != nil {
		return 0, fmt.Errorf("querying extrato for auto-match: %w", err)
	}
	defer rows.Close()

	type entry struct {
		id    int64
		data  time.Time
		valor float64
		tipo  string
	}
	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.id, &e.data, &e.valor, &e.tipo); err != nil {
			return 0, err
		}
		entries = append(entries, e)
	}
	_ = rows.Err()

	matched := 0
	for _, e := range entries {
		fcTipo := "ENTRADA"
		if e.tipo == "DEBIT" {
			fcTipo = "SAIDA"
		}
		var fluxoID int64
		err := r.pool.QueryRow(ctx,
			`SELECT id FROM fluxo_caixa
			 WHERE conta_bancaria_id=$1 AND tipo=$2
			   AND ABS(valor - $3) < 0.01
			   AND data BETWEEN $4 AND $5
			   AND conciliado=false
			   AND enterprise_id=$6
			 LIMIT 1`,
			contaID, fcTipo, e.valor, e.data.AddDate(0, 0, -3), e.data.AddDate(0, 0, 3), empresaAuto,
		).Scan(&fluxoID)
		if err != nil {
			continue
		}
		if err := r.ConciliarExtrato(ctx, e.id, fluxoID); err == nil {
			matched++
		}
	}
	return matched, nil
}
