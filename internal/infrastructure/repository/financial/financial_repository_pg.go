package financial

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	fiscalEntity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// ---------- Contas Bancarias ----------

func (r *FinancialRepositoryPG) CreateContaBancaria(ctx context.Context, c *entity.ContaBancaria) (*entity.ContaBancaria, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO contas_bancarias (banco, agencia, conta, digito, descricao, titular, saldo_inicial, chave_pix, tipo_chave_pix, is_active, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 RETURNING id, created_at, updated_at`,
		c.Banco, c.Agencia, c.Conta, c.Digito, c.Descricao, c.Titular,
		c.SaldoInicial.InexactFloat64(), c.ChavePix, c.TipoChavePix, c.IsActive, c.CreatedBy,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating conta bancaria: %w", err)
	}
	return c, nil
}

func (r *FinancialRepositoryPG) ListContasBancarias(ctx context.Context) ([]*entity.ContaBancaria, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, banco, agencia, conta, digito, descricao, titular, saldo_inicial, chave_pix, tipo_chave_pix, is_active, created_at, updated_at, created_by
		 FROM contas_bancarias WHERE is_active = true ORDER BY descricao`)
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
			&c.CreatedAt, &c.UpdatedAt, &c.CreatedBy); err != nil {
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
	err := r.pool.QueryRow(ctx,
		`SELECT id, banco, agencia, conta, digito, descricao, titular, saldo_inicial, chave_pix, tipo_chave_pix, is_active, created_at, updated_at, created_by
		 FROM contas_bancarias WHERE id = $1`, id,
	).Scan(&c.ID, &c.Banco, &c.Agencia, &c.Conta, &c.Digito, &c.Descricao,
		&c.Titular, &saldo, &c.ChavePix, &c.TipoChavePix, &c.IsActive,
		&c.CreatedAt, &c.UpdatedAt, &c.CreatedBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("conta bancaria %d not found", id)
		}
		return nil, fmt.Errorf("getting conta bancaria: %w", err)
	}
	c.SaldoInicial = decimal.NewFromFloat(saldo)
	return &c, nil
}

func (r *FinancialRepositoryPG) UpdateSaldo(ctx context.Context, id int64, novoSaldo float64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE contas_bancarias SET saldo_inicial = $1, updated_at = NOW() WHERE id = $2`,
		novoSaldo, id)
	if err != nil {
		return fmt.Errorf("updating saldo conta %d: %w", id, err)
	}
	return nil
}

// ---------- Condicoes Pagamento ----------

func (r *FinancialRepositoryPG) CreateCondicaoPagamento(ctx context.Context, c *entity.CondicaoPagamento) (*entity.CondicaoPagamento, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO condicoes_pagamento (nome, parcelas, ativo)
		 VALUES ($1,$2,$3)
		 RETURNING id, created_at, updated_at`,
		c.Nome, c.Parcelas, c.Ativo,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating condicao pagamento: %w", err)
	}
	return c, nil
}

func (r *FinancialRepositoryPG) ListCondicoesPagamento(ctx context.Context) ([]*entity.CondicaoPagamento, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, nome, parcelas, ativo, created_at, updated_at
		 FROM condicoes_pagamento WHERE ativo = true ORDER BY nome`)
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
	err := r.pool.QueryRow(ctx,
		`INSERT INTO plano_contas (codigo, descricao, tipo, natureza, parent_code, nivel, is_active)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 RETURNING id, created_at`,
		p.Codigo, p.Descricao, p.Tipo, p.Natureza, p.ParentCode, p.Nivel, p.IsActive,
	).Scan(&p.ID, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating plano contas: %w", err)
	}
	return p, nil
}

func (r *FinancialRepositoryPG) ListPlanoContas(ctx context.Context) ([]*entity.PlanoContas, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, codigo, descricao, tipo, natureza, parent_code, nivel, is_active, created_at
		 FROM plano_contas WHERE is_active = true ORDER BY codigo`)
	if err != nil {
		return nil, fmt.Errorf("listing plano contas: %w", err)
	}
	defer rows.Close()

	var out []*entity.PlanoContas
	for rows.Next() {
		var p entity.PlanoContas
		if err := rows.Scan(&p.ID, &p.Codigo, &p.Descricao, &p.Tipo, &p.Natureza,
			&p.ParentCode, &p.Nivel, &p.IsActive, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning plano contas: %w", err)
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// ---------- Centros de Custo ----------

func (r *FinancialRepositoryPG) CreateCentroCusto(ctx context.Context, c *entity.CentroCusto) (*entity.CentroCusto, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO centros_custo (codigo, descricao, tipo, is_active)
		 VALUES ($1,$2,$3,$4)
		 RETURNING id, created_at`,
		c.Codigo, c.Descricao, c.Tipo, c.IsActive,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating centro custo: %w", err)
	}
	return c, nil
}

func (r *FinancialRepositoryPG) ListCentrosCusto(ctx context.Context) ([]*entity.CentroCusto, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, codigo, descricao, tipo, is_active, created_at
		 FROM centros_custo WHERE is_active = true ORDER BY codigo`)
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

func (r *FinancialRepositoryPG) CreateContaPagar(ctx context.Context, c *entity.ContaPagar) (*entity.ContaPagar, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO contas_pagar
			(numero_documento, tipo_documento, fornecedor_id, fiscal_entry_id, purchase_order_id,
			 data_lancamento, data_emissao, data_vencimento,
			 valor_bruto, desconto, juros, multa, valor_pago,
			 parcela_numero, parcela_total, parcela_pai_id,
			 forma_pagamento, plano_contas_id, centro_custo_id,
			 status_aprovacao, status,
			 observacao, is_active, criado_por)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)
		 RETURNING id, created_at, updated_at`,
		c.NumeroDocumento, c.TipoDocumento, c.FornecedorID, c.FiscalEntryID, c.PurchaseOrderID,
		c.DataLancamento, c.DataEmissao, c.DataVencimento,
		c.ValorBruto.InexactFloat64(), c.Desconto.InexactFloat64(), c.Juros.InexactFloat64(), c.Multa.InexactFloat64(), c.ValorPago.InexactFloat64(),
		c.ParcelaNumero, c.ParcelaTotal, c.ParcelaPaiID,
		c.FormaPagamento, c.PlanoContasID, c.CentroCustoID,
		string(c.StatusAprovacao), string(c.Status),
		c.Observacao, c.IsActive, c.CriadoPor,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating conta pagar: %w", err)
	}
	return c, nil
}

func (r *FinancialRepositoryPG) GetContaPagar(ctx context.Context, id int64) (*entity.ContaPagar, error) {
	return r.scanContaPagarRow(r.pool.QueryRow(ctx,
		`SELECT id, numero_documento, tipo_documento, fornecedor_id, fiscal_entry_id, purchase_order_id,
		        data_lancamento, data_emissao, data_vencimento, data_pagamento,
		        valor_bruto, desconto, juros, multa, valor_pago,
		        parcela_numero, parcela_total, parcela_pai_id,
		        conta_bancaria_id, forma_pagamento,
		        plano_contas_id, centro_custo_id,
		        status_aprovacao, aprovado_por, data_aprovacao, motivo_rejeicao,
		        status, adiantamento_id, valor_adiantamento_abatido,
		        comprovante_path, observacao,
		        is_active, criado_por, baixado_por, created_at, updated_at
		 FROM contas_pagar WHERE id = $1`, id))
}

func (r *FinancialRepositoryPG) ListContasPagar(ctx context.Context, filters repository.CPFilter) ([]*entity.ContaPagar, error) {
	query := `SELECT id, numero_documento, tipo_documento, fornecedor_id, fiscal_entry_id, purchase_order_id,
		        data_lancamento, data_emissao, data_vencimento, data_pagamento,
		        valor_bruto, desconto, juros, multa, valor_pago,
		        parcela_numero, parcela_total, parcela_pai_id,
		        conta_bancaria_id, forma_pagamento,
		        plano_contas_id, centro_custo_id,
		        status_aprovacao, aprovado_por, data_aprovacao, motivo_rejeicao,
		        status, adiantamento_id, valor_adiantamento_abatido,
		        comprovante_path, observacao,
		        is_active, criado_por, baixado_por, created_at, updated_at
		 FROM contas_pagar WHERE is_active = true`

	var args []interface{}
	argIdx := 1

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.FornecedorID != nil {
		query += fmt.Sprintf(" AND fornecedor_id = $%d", argIdx)
		args = append(args, *filters.FornecedorID)
		argIdx++
	}
	if filters.StartDate != nil {
		query += fmt.Sprintf(" AND data_vencimento >= $%d", argIdx)
		args = append(args, *filters.StartDate)
		argIdx++
	}
	if filters.EndDate != nil {
		query += fmt.Sprintf(" AND data_vencimento <= $%d", argIdx)
		args = append(args, *filters.EndDate)
		argIdx++
	}
	query += " ORDER BY data_vencimento ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing contas pagar: %w", err)
	}
	defer rows.Close()

	var out []*entity.ContaPagar
	for rows.Next() {
		c, err := r.scanContaPagar(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *FinancialRepositoryPG) UpdateContaPagar(ctx context.Context, c *entity.ContaPagar) (*entity.ContaPagar, error) {
	_, err := r.pool.Exec(ctx,
		`UPDATE contas_pagar SET
			numero_documento=$1, tipo_documento=$2, fornecedor_id=$3,
			data_emissao=$4, data_vencimento=$5,
			valor_bruto=$6, desconto=$7, juros=$8, multa=$9, valor_pago=$10,
			parcela_numero=$11, parcela_total=$12,
			forma_pagamento=$13, plano_contas_id=$14, centro_custo_id=$15,
			observacao=$16, updated_at=NOW()
		 WHERE id=$17`,
		c.NumeroDocumento, c.TipoDocumento, c.FornecedorID,
		c.DataEmissao, c.DataVencimento,
		c.ValorBruto.InexactFloat64(), c.Desconto.InexactFloat64(), c.Juros.InexactFloat64(), c.Multa.InexactFloat64(), c.ValorPago.InexactFloat64(),
		c.ParcelaNumero, c.ParcelaTotal,
		c.FormaPagamento, c.PlanoContasID, c.CentroCustoID,
		c.Observacao, c.ID)
	if err != nil {
		return nil, fmt.Errorf("updating conta pagar %d: %w", c.ID, err)
	}
	return r.GetContaPagar(ctx, c.ID)
}

func (r *FinancialRepositoryPG) ApproveContaPagar(ctx context.Context, id int64, approvedBy uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE contas_pagar SET
			status_aprovacao = 'APROVADO',
			status = 'APROVADO',
			aprovado_por = $1,
			data_aprovacao = NOW(),
			updated_at = NOW()
		 WHERE id = $2`, approvedBy, id)
	if err != nil {
		return fmt.Errorf("approving conta pagar %d: %w", id, err)
	}
	return nil
}

func (r *FinancialRepositoryPG) BaixarContaPagar(ctx context.Context, id int64, params repository.BaixaParams) error {
	_, err := r.pool.Exec(ctx,
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
		 WHERE id = $9`,
		params.DataPagamento, params.ValorPago, params.Juros, params.Multa,
		params.Desconto, params.ContaBancariaID, params.BaixadoPor,
		params.Observacao, id)
	if err != nil {
		return fmt.Errorf("baixando conta pagar %d: %w", id, err)
	}
	return nil
}

func (r *FinancialRepositoryPG) CancelContaPagar(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE contas_pagar SET status = 'CANCELADO', is_active = false, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("cancelling conta pagar %d: %w", id, err)
	}
	return nil
}

func (r *FinancialRepositoryPG) GetAgingContasPagar(ctx context.Context) ([]*repository.AgingResult, error) {
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
		 WHERE is_active = true AND status IN ('PENDENTE', 'APROVADO', 'VENCIDO')
		 GROUP BY period
		 ORDER BY period`)
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
	err := r.pool.QueryRow(ctx,
		`INSERT INTO contas_receber
			(numero_documento, cliente_id, fiscal_exit_id, sales_order_id,
			 data_lancamento, data_emissao, data_vencimento,
			 valor_bruto, desconto, juros, multa, valor_recebido,
			 parcela_numero, parcela_total,
			 forma_pagamento, plano_contas_id, centro_custo_id,
			 status, is_active, criado_por)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
		 RETURNING id, created_at, updated_at`,
		c.NumeroDocumento, c.ClienteID, c.FiscalExitID, c.SalesOrderID,
		c.DataLancamento, c.DataEmissao, c.DataVencimento,
		c.ValorBruto.InexactFloat64(), c.Desconto.InexactFloat64(), c.Juros.InexactFloat64(), c.Multa.InexactFloat64(), c.ValorRecebido.InexactFloat64(),
		c.ParcelaNumero, c.ParcelaTotal,
		c.FormaPagamento, c.PlanoContasID, c.CentroCustoID,
		string(c.Status), c.IsActive, c.CriadoPor,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating conta receber: %w", err)
	}
	return c, nil
}

func (r *FinancialRepositoryPG) GetContaReceber(ctx context.Context, id int64) (*entity.ContaReceber, error) {
	return r.scanContaReceberRow(r.pool.QueryRow(ctx,
		`SELECT id, numero_documento, cliente_id, fiscal_exit_id, sales_order_id,
		        data_lancamento, data_emissao, data_vencimento, data_recebimento,
		        valor_bruto, desconto, juros, multa, valor_recebido,
		        parcela_numero, parcela_total, parcela_pai_id,
		        conta_bancaria_id, forma_pagamento,
		        nosso_numero, linha_digitavel, codigo_barras, chave_pix_gerada,
		        plano_contas_id, centro_custo_id,
		        status, em_protesto,
		        is_active, criado_por, baixado_por, created_at, updated_at
		 FROM contas_receber WHERE id = $1`, id))
}

func (r *FinancialRepositoryPG) ListContasReceber(ctx context.Context, filters repository.CRFilter) ([]*entity.ContaReceber, error) {
	query := `SELECT id, numero_documento, cliente_id, fiscal_exit_id, sales_order_id,
		        data_lancamento, data_emissao, data_vencimento, data_recebimento,
		        valor_bruto, desconto, juros, multa, valor_recebido,
		        parcela_numero, parcela_total, parcela_pai_id,
		        conta_bancaria_id, forma_pagamento,
		        nosso_numero, linha_digitavel, codigo_barras, chave_pix_gerada,
		        plano_contas_id, centro_custo_id,
		        status, em_protesto,
		        is_active, criado_por, baixado_por, created_at, updated_at
		 FROM contas_receber WHERE is_active = true`

	var args []interface{}
	argIdx := 1

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.ClienteID != nil {
		query += fmt.Sprintf(" AND cliente_id = $%d", argIdx)
		args = append(args, *filters.ClienteID)
		argIdx++
	}
	if filters.StartDate != nil {
		query += fmt.Sprintf(" AND data_vencimento >= $%d", argIdx)
		args = append(args, *filters.StartDate)
		argIdx++
	}
	if filters.EndDate != nil {
		query += fmt.Sprintf(" AND data_vencimento <= $%d", argIdx)
		args = append(args, *filters.EndDate)
		argIdx++
	}
	query += " ORDER BY data_vencimento ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing contas receber: %w", err)
	}
	defer rows.Close()

	var out []*entity.ContaReceber
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
	_, err := r.pool.Exec(ctx,
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
		 WHERE id = $8`,
		params.DataPagamento, params.ValorPago, params.Juros, params.Multa,
		params.Desconto, params.ContaBancariaID, params.BaixadoPor, id)
	if err != nil {
		return fmt.Errorf("baixando conta receber %d: %w", id, err)
	}
	return nil
}

func (r *FinancialRepositoryPG) CancelContaReceber(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE contas_receber SET status = 'CANCELADO', is_active = false, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("cancelling conta receber %d: %w", id, err)
	}
	return nil
}

func (r *FinancialRepositoryPG) GetAgingContasReceber(ctx context.Context) ([]*repository.AgingResult, error) {
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
		 WHERE is_active = true AND status IN ('PENDENTE', 'APROVADO', 'VENCIDO')
		 GROUP BY period
		 ORDER BY period`)
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
	err := r.pool.QueryRow(ctx,
		`INSERT INTO fluxo_caixa
			(data, tipo, valor, conta_bancaria_id, conta_bancaria_destino_id,
			 contas_pagar_id, contas_receber_id, descricao, conciliado)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING id, created_at`,
		f.Data, string(f.Tipo), f.Valor.InexactFloat64(), f.ContaBancariaID, f.ContaBancariaDestinoID,
		f.ContasPagarID, f.ContasReceberID, f.Descricao, f.Conciliado,
	).Scan(&f.ID, &f.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating fluxo caixa: %w", err)
	}
	return f, nil
}

func (r *FinancialRepositoryPG) GetFluxoCaixa(ctx context.Context, startDate, endDate time.Time) ([]*entity.FluxoCaixa, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, data, tipo, valor, conta_bancaria_id, conta_bancaria_destino_id,
		        contas_pagar_id, contas_receber_id, descricao, conciliado, extrato_hash, created_at
		 FROM fluxo_caixa
		 WHERE data >= $1 AND data <= $2
		 ORDER BY data ASC`, startDate, endDate)
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
	rows, err := r.pool.Query(ctx,
		`WITH projected AS (
			SELECT data_vencimento AS data, valor_bruto - COALESCE(valor_pago, 0) AS valor, 'SAIDA' AS tipo
			FROM contas_pagar
			WHERE is_active = true AND status IN ('PENDENTE', 'APROVADO', 'VENCIDO')
			UNION ALL
			SELECT data_vencimento AS data, valor_bruto - COALESCE(valor_recebido, 0) AS valor, 'ENTRADA' AS tipo
			FROM contas_receber
			WHERE is_active = true AND status IN ('PENDENTE', 'APROVADO', 'VENCIDO')
		)
		SELECT data,
			COALESCE(SUM(CASE WHEN tipo = 'ENTRADA' THEN valor ELSE 0 END), 0) AS valor_entradas,
			COALESCE(SUM(CASE WHEN tipo = 'SAIDA' THEN valor ELSE 0 END), 0) AS valor_saidas
		 FROM projected
		 WHERE data >= $1
		 GROUP BY data
		 ORDER BY data ASC`, startDate)
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
	var saldo float64
	err := r.pool.QueryRow(ctx,
		`SELECT saldo_inicial FROM contas_bancarias WHERE id = $1`, contaID,
	).Scan(&saldo)
	if err != nil {
		return 0, fmt.Errorf("getting saldo conta %d: %w", contaID, err)
	}

	// Add cash flow movements
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
	var saldo float64
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(saldo_inicial), 0) FROM contas_bancarias WHERE is_active = true`,
	).Scan(&saldo)
	if err != nil {
		return 0, fmt.Errorf("getting saldo consolidado: %w", err)
	}

	var movSum float64
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(CASE WHEN tipo = 'ENTRADA' THEN valor ELSE -valor END), 0)
		 FROM fluxo_caixa WHERE conciliado = true`,
	).Scan(&movSum)
	if err == nil {
		saldo += movSum
	}

	return saldo, nil
}

func (r *FinancialRepositoryPG) MarcarConciliado(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE fluxo_caixa SET conciliado = true WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("marcando conciliado fluxo %d: %w", id, err)
	}
	return nil
}

// ---------- Tax Assessment ----------

func (r *FinancialRepositoryPG) CreateTaxAssessment(ctx context.Context, t *entity.TaxAssessment) (*entity.TaxAssessment, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO tax_assessments
			(imposto, competencia, debitos, creditos, saldo_devedor, saldo_credor, status, cp_id, data_vencimento)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING id, created_at, updated_at`,
		t.Imposto, t.Competencia,
		t.Debitos.InexactFloat64(), t.Creditos.InexactFloat64(),
		t.SaldoDevedor.InexactFloat64(), t.SaldoCredor.InexactFloat64(),
		string(t.Status), t.CpID, t.DataVencimento,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating tax assessment: %w", err)
	}
	return t, nil
}

func (r *FinancialRepositoryPG) GetTaxAssessment(ctx context.Context, imposto, competencia string) (*entity.TaxAssessment, error) {
	var t entity.TaxAssessment
	var debitos, creditos, saldoDevedor, saldoCredor float64
	err := r.pool.QueryRow(ctx,
		`SELECT id, imposto, competencia, debitos, creditos, saldo_devedor, saldo_credor,
		        status, cp_id, data_vencimento, created_at, updated_at
		 FROM tax_assessments WHERE imposto = $1 AND competencia = $2`,
		imposto, competencia,
	).Scan(&t.ID, &t.Imposto, &t.Competencia,
		&debitos, &creditos, &saldoDevedor, &saldoCredor,
		&t.Status, &t.CpID, &t.DataVencimento, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tax assessment %s/%s not found", imposto, competencia)
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
	rows, err := r.pool.Query(ctx,
		`SELECT id, imposto, competencia, debitos, creditos, saldo_devedor, saldo_credor,
		        status, cp_id, data_vencimento, created_at, updated_at
		 FROM tax_assessments WHERE competencia = $1 ORDER BY imposto`, competencia)
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

func (r *FinancialRepositoryPG) GetFiscalDebits(ctx context.Context, competencia string) (map[string]float64, error) {
	query := `SELECT 'ICMS' as imposto, COALESCE(SUM(valor_icms), 0) FROM fiscal_exits WHERE status = 'AUTHORIZED' AND to_char(data_emissao, 'MM/YYYY') = $1
	 UNION ALL SELECT 'IPI', COALESCE(SUM(valor_ipi), 0) FROM fiscal_exits WHERE status = 'AUTHORIZED' AND to_char(data_emissao, 'MM/YYYY') = $1
	 UNION ALL SELECT 'PIS', COALESCE(SUM(valor_pis), 0) FROM fiscal_exits WHERE status = 'AUTHORIZED' AND to_char(data_emissao, 'MM/YYYY') = $1
	 UNION ALL SELECT 'COFINS', COALESCE(SUM(valor_cofins), 0) FROM fiscal_exits WHERE status = 'AUTHORIZED' AND to_char(data_emissao, 'MM/YYYY') = $1`
	rows, err := r.pool.Query(ctx, query, competencia)
	if err != nil {
		return nil, fmt.Errorf("getting fiscal debits: %w", err)
	}
	defer rows.Close()
	result := make(map[string]float64)
	for rows.Next() {
		var imposto string
		var total float64
		if err := rows.Scan(&imposto, &total); err != nil {
			return nil, fmt.Errorf("scanning fiscal debit: %w", err)
		}
		result[imposto] = total
	}
	return result, rows.Err()
}

func (r *FinancialRepositoryPG) GetFiscalCredits(ctx context.Context, competencia string) (map[string]float64, error) {
	query := `SELECT 'ICMS' as imposto, COALESCE(SUM(valor_icms), 0) FROM fiscal_entries WHERE status = 'APPROVED' AND to_char(data_emissao, 'MM/YYYY') = $1
	 UNION ALL SELECT 'IPI', COALESCE(SUM(valor_ipi), 0) FROM fiscal_entries WHERE status = 'APPROVED' AND to_char(data_emissao, 'MM/YYYY') = $1
	 UNION ALL SELECT 'PIS', COALESCE(SUM(valor_pis), 0) FROM fiscal_entries WHERE status = 'APPROVED' AND to_char(data_emissao, 'MM/YYYY') = $1
	 UNION ALL SELECT 'COFINS', COALESCE(SUM(valor_cofins), 0) FROM fiscal_entries WHERE status = 'APPROVED' AND to_char(data_emissao, 'MM/YYYY') = $1`
	rows, err := r.pool.Query(ctx, query, competencia)
	if err != nil {
		return nil, fmt.Errorf("getting fiscal credits: %w", err)
	}
	defer rows.Close()
	result := make(map[string]float64)
	for rows.Next() {
		var imposto string
		var total float64
		if err := rows.Scan(&imposto, &total); err != nil {
			return nil, fmt.Errorf("scanning fiscal credit: %w", err)
		}
		result[imposto] = total
	}
	return result, rows.Err()
}

func (r *FinancialRepositoryPG) GetFiscalConfig(ctx context.Context) (*fiscalEntity.FiscalConfig, error) {
	var cfg fiscalEntity.FiscalConfig
	err := r.pool.QueryRow(ctx,
		`SELECT id, cnpj_empresa, razao_social, ie_empresa, regime_tributario, uf_empresa,
		        icms_interno_aliquota, icms_diferimento_percentual,
		        focus_nfe_token, focus_nfe_ambiente, juros_mes, multa_atraso,
		        vencimento_icms_dia, vencimento_ipi_dia, vencimento_pis_cofins_dia,
		        created_at, updated_at, updated_by
		 FROM public.fiscal_configs ORDER BY id LIMIT 1`,
	).Scan(&cfg.ID, &cfg.CnpjEmpresa, &cfg.RazaoSocial, &cfg.IEEmpresa, &cfg.RegimeTributario, &cfg.UFEmpresa,
		&cfg.IcmsInternoAliquota, &cfg.IcmsDiferimentoPercentual,
		&cfg.FocusNfeToken, &cfg.FocusNfeAmbiente, &cfg.JurosMes, &cfg.MultaAtraso,
		&cfg.VencimentoIcmsDia, &cfg.VencimentoIPIDia, &cfg.VencimentoPisCofinsDia,
		&cfg.CreatedAt, &cfg.UpdatedAt, &cfg.UpdatedBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("fiscal config not found")
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
			return nil, fmt.Errorf("conta pagar not found")
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
		&c.ID, &c.NumeroDocumento, &c.ClienteID, &c.FiscalExitID, &c.SalesOrderID,
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
			return nil, fmt.Errorf("conta receber not found")
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
		&c.ID, &c.NumeroDocumento, &c.ClienteID, &c.FiscalExitID, &c.SalesOrderID,
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
