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

// ---------- UpsertTaxAssessmentCredito ----------

func (r *FinancialRepositoryPG) UpsertTaxAssessmentCredito(ctx context.Context, t *entity.TaxAssessment) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tax_assessments (imposto, competencia, debitos, creditos, saldo_devedor, saldo_credor, status)
		 VALUES ($1, $2, 0, $3, 0, $3, 'APURAR')
		 ON CONFLICT (imposto, competencia) DO UPDATE SET
		     creditos = tax_assessments.creditos + EXCLUDED.creditos,
		     updated_at = NOW()`,
		t.Imposto, t.Competencia, t.Creditos.InexactFloat64())
	if err != nil {
		return fmt.Errorf("upserting tax assessment credito: %w", err)
	}
	return nil
}

// ---------- CancelContasReceberByFiscalExit ----------

func (r *FinancialRepositoryPG) CancelContasReceberByFiscalExit(ctx context.Context, fiscalExitID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE contas_receber SET status = 'CANCELADO', is_active = false, updated_at = NOW()
		 WHERE fiscal_exit_id = $1 AND status IN ('PENDENTE', 'APROVADO', 'VENCIDO')`, fiscalExitID)
	if err != nil {
		return fmt.Errorf("cancelling contas receber for exit %d: %w", fiscalExitID, err)
	}
	return nil
}

// ---------- Reports ----------

func (r *FinancialRepositoryPG) GetLivroEntradas(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT fe.id, fe.data_entrada, fe.numero_nf, fe.serie, fe.razao_social_emitente,
		        fei.cfop, fe.valor_produtos, fe.valor_ipi, fe.valor_icms, fe.valor_pis, fe.valor_cofins, fe.valor_total
		 FROM fiscal_entries fe
		 LEFT JOIN fiscal_entry_items fei ON fei.fiscal_entry_id = fe.id
		 WHERE fe.data_entrada BETWEEN $1 AND $2 AND fe.status = 'APPROVED'
		 ORDER BY fe.data_entrada, fe.numero_nf`, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("livro entradas: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"id", "data_entrada", "numero_nf", "serie", "emitente", "cfop", "valor_produtos", "valor_ipi", "valor_icms", "valor_pis", "valor_cofins", "valor_total"})
}

func (r *FinancialRepositoryPG) GetLivroSaidas(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT fe.id, fe.data_emissao, fe.numero_nf, fe.serie, fe.razao_social_destinatario,
		        fei.cfop, fe.valor_produtos, fe.valor_ipi, fe.valor_icms, fe.valor_pis, fe.valor_cofins, fe.valor_total
		 FROM fiscal_exits fe
		 LEFT JOIN fiscal_exit_items fei ON fei.fiscal_exit_id = fe.id
		 WHERE fe.data_emissao BETWEEN $1 AND $2 AND fe.status = 'AUTHORIZED'
		 ORDER BY fe.data_emissao, fe.numero_nf`, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("livro saidas: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"id", "data_emissao", "numero_nf", "serie", "destinatario", "cfop", "valor_produtos", "valor_ipi", "valor_icms", "valor_pis", "valor_cofins", "valor_total"})
}

func (r *FinancialRepositoryPG) GetImpostosSaidas(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT fe.numero_nf, fe.data_emissao, fe.razao_social_destinatario, fei.cfop,
		        fei.base_icms, fei.valor_icms, fei.base_ipi, fei.valor_ipi, fei.valor_pis, fei.valor_cofins
		 FROM fiscal_exits fe
		 JOIN fiscal_exit_items fei ON fei.fiscal_exit_id = fe.id
		 WHERE fe.data_emissao BETWEEN $1 AND $2 AND fe.status = 'AUTHORIZED'
		 ORDER BY fe.data_emissao`, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("impostos saidas: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"numero_nf", "data", "cliente", "cfop", "base_icms", "valor_icms", "base_ipi", "valor_ipi", "valor_pis", "valor_cofins"})
}

func (r *FinancialRepositoryPG) GetImpostosEntradas(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT fe.numero_nf, fe.data_entrada, fe.razao_social_emitente, fei.cfop,
		        fei.base_icms, fei.valor_icms, fei.base_ipi, fei.valor_ipi, fei.valor_pis, fei.valor_cofins
		 FROM fiscal_entries fe
		 JOIN fiscal_entry_items fei ON fei.fiscal_entry_id = fe.id
		 WHERE fe.data_entrada BETWEEN $1 AND $2 AND fe.status = 'APPROVED'
		 ORDER BY fe.data_entrada`, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("impostos entradas: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"numero_nf", "data", "fornecedor", "cfop", "base_icms", "valor_icms", "base_ipi", "valor_ipi", "valor_pis", "valor_cofins"})
}

func (r *FinancialRepositoryPG) GetDRE(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	var receitaBruta, impostosVendas float64
	_ = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(valor_produtos),0), COALESCE(SUM(valor_icms + valor_ipi + valor_pis + valor_cofins),0)
		 FROM fiscal_exits WHERE data_emissao BETWEEN $1 AND $2 AND status = 'AUTHORIZED'`,
		startDate, endDate).Scan(&receitaBruta, &impostosVendas)

	var despesas float64
	_ = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(valor_bruto - COALESCE(valor_pago,0)),0)
		 FROM contas_pagar WHERE data_vencimento BETWEEN $1 AND $2 AND status IN ('PAGO','CANCELADO')`,
		startDate, endDate).Scan(&despesas)

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
		 WHERE status IN ('PENDENTE','VENCIDO') AND is_active = true
		 ORDER BY data_vencimento`)
	if err != nil {
		return nil, fmt.Errorf("aging receber: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"id", "numero_documento", "cliente_id", "data_vencimento", "valor_bruto", "faixa"})
}

func (r *FinancialRepositoryPG) GetAgingPagarDetalhado(ctx context.Context) ([]map[string]interface{}, error) {
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
		 WHERE status IN ('PENDENTE','APROVADO','VENCIDO') AND is_active = true
		 ORDER BY data_vencimento`)
	if err != nil {
		return nil, fmt.Errorf("aging pagar: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"id", "numero_documento", "fornecedor_id", "data_vencimento", "valor_bruto", "faixa"})
}

func (r *FinancialRepositoryPG) GetExtratoPorFornecedor(ctx context.Context, fornecedorID int64) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT numero_documento, data_emissao, data_vencimento, data_pagamento,
		        valor_bruto, desconto, juros, multa, COALESCE(valor_pago,0) as valor_pago, status
		 FROM contas_pagar WHERE fornecedor_id = $1 AND is_active = true ORDER BY data_emissao`, fornecedorID)
	if err != nil {
		return nil, fmt.Errorf("extrato fornecedor: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"numero_documento", "data_emissao", "data_vencimento", "data_pagamento", "valor_bruto", "desconto", "juros", "multa", "valor_pago", "status"})
}

func (r *FinancialRepositoryPG) GetExtratoPorCliente(ctx context.Context, clienteID int64) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT numero_documento, data_emissao, data_vencimento, data_recebimento,
		        valor_bruto, desconto, juros, multa, COALESCE(valor_recebido,0) as valor_recebido, status
		 FROM contas_receber WHERE cliente_id = $1 AND is_active = true ORDER BY data_emissao`, clienteID)
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

func (r *FinancialRepositoryPG) BaixarContaPagarAtomico(ctx context.Context, id int64, params repository.BaixaParams, fc entity.FluxoCaixa, valorOriginal decimal.Decimal, contaBancariaID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	valorPago := decimal.NewFromFloat(params.ValorPago)
	isParcial := valorPago.LessThan(valorOriginal)

	// Update main CP
	if isParcial {
		_, err = tx.Exec(ctx,
			`UPDATE contas_pagar SET status='PAGO', data_pagamento=$1, valor_pago=$2, juros=$3, multa=$4,
			     conta_bancaria_id=$5, baixado_por=$6, updated_at=NOW() WHERE id=$7`,
			params.DataPagamento, params.ValorPago, params.Juros, params.Multa,
			params.ContaBancariaID, params.BaixadoPor, id)
		if err != nil {
			return fmt.Errorf("baixando CP parcial: %w", err)
		}
		// Create remaining CP
		remaining := valorOriginal.Sub(valorPago)
		_, err = tx.Exec(ctx,
			`INSERT INTO contas_pagar (numero_documento, tipo_documento, fornecedor_id, fiscal_entry_id,
			     data_lancamento, data_emissao, data_vencimento,
			     valor_bruto, desconto, juros, multa, valor_pago,
			     parcela_numero, parcela_total, status_aprovacao, status, is_active, criado_por)
			 SELECT numero_documento || '/P', tipo_documento, fornecedor_id, fiscal_entry_id,
			     NOW(), data_emissao, data_vencimento,
			     $1, 0, 0, 0, 0,
			     parcela_numero+1, parcela_total, 'APROVADO', 'PENDENTE', true, criado_por
			 FROM contas_pagar WHERE id = $2`,
			remaining.InexactFloat64(), id)
		if err != nil {
			return fmt.Errorf("creating remaining CP: %w", err)
		}
	} else {
		_, err = tx.Exec(ctx,
			`UPDATE contas_pagar SET status='PAGO', data_pagamento=$1, valor_pago=$2, juros=$3, multa=$4,
			     conta_bancaria_id=$5, baixado_por=$6, updated_at=NOW() WHERE id=$7`,
			params.DataPagamento, params.ValorPago, params.Juros, params.Multa,
			params.ContaBancariaID, params.BaixadoPor, id)
		if err != nil {
			return fmt.Errorf("baixando CP: %w", err)
		}
	}

	// Insert fluxo de caixa
	_, err = tx.Exec(ctx,
		`INSERT INTO fluxo_caixa (data, tipo, valor, conta_bancaria_id, contas_pagar_id, descricao, conciliado)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		fc.Data, string(fc.Tipo), fc.Valor.InexactFloat64(), fc.ContaBancariaID, fc.ContasPagarID, fc.Descricao, false)
	if err != nil {
		return fmt.Errorf("creating fluxo caixa: %w", err)
	}

	// Update saldo
	_, err = tx.Exec(ctx,
		`UPDATE contas_bancarias SET saldo_inicial = saldo_inicial - $1, updated_at = NOW() WHERE id = $2`,
		fc.Valor.InexactFloat64(), contaBancariaID)
	if err != nil {
		return fmt.Errorf("updating saldo: %w", err)
	}

	return tx.Commit(ctx)
}

// ---------- BaixarContaReceberAtomico ----------

func (r *FinancialRepositoryPG) BaixarContaReceberAtomico(ctx context.Context, id int64, params repository.BaixaParams, fc entity.FluxoCaixa, valorOriginal decimal.Decimal, contaBancariaID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	valorRecebido := decimal.NewFromFloat(params.ValorPago)
	isParcial := valorRecebido.LessThan(valorOriginal)

	if isParcial {
		_, err = tx.Exec(ctx,
			`UPDATE contas_receber SET status='RECEBIDO', data_recebimento=$1, valor_recebido=$2, juros=$3, multa=$4,
			     conta_bancaria_id=$5, baixado_por=$6, updated_at=NOW() WHERE id=$7`,
			params.DataPagamento, params.ValorPago, params.Juros, params.Multa,
			params.ContaBancariaID, params.BaixadoPor, id)
		if err != nil {
			return fmt.Errorf("baixando CR parcial: %w", err)
		}
		remaining := valorOriginal.Sub(valorRecebido)
		_, err = tx.Exec(ctx,
			`INSERT INTO contas_receber (numero_documento, cliente_id, fiscal_exit_id,
			     data_lancamento, data_emissao, data_vencimento,
			     valor_bruto, desconto, juros, multa, valor_recebido,
			     parcela_numero, parcela_total, status, is_active, criado_por)
			 SELECT numero_documento, cliente_id, fiscal_exit_id,
			     NOW(), data_emissao, data_vencimento,
			     $1, 0, 0, 0, 0,
			     parcela_numero+1, parcela_total, 'PENDENTE', true, criado_por
			 FROM contas_receber WHERE id = $2`,
			remaining.InexactFloat64(), id)
		if err != nil {
			return fmt.Errorf("creating remaining CR: %w", err)
		}
	} else {
		_, err = tx.Exec(ctx,
			`UPDATE contas_receber SET status='RECEBIDO', data_recebimento=$1, valor_recebido=$2, juros=$3, multa=$4,
			     conta_bancaria_id=$5, baixado_por=$6, updated_at=NOW() WHERE id=$7`,
			params.DataPagamento, params.ValorPago, params.Juros, params.Multa,
			params.ContaBancariaID, params.BaixadoPor, id)
		if err != nil {
			return fmt.Errorf("baixando CR: %w", err)
		}
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO fluxo_caixa (data, tipo, valor, conta_bancaria_id, contas_receber_id, descricao, conciliado)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		fc.Data, string(fc.Tipo), fc.Valor.InexactFloat64(), fc.ContaBancariaID, fc.ContasReceberID, fc.Descricao, false)
	if err != nil {
		return fmt.Errorf("creating fluxo caixa CR: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE contas_bancarias SET saldo_inicial = saldo_inicial + $1, updated_at = NOW() WHERE id = $2`,
		fc.Valor.InexactFloat64(), contaBancariaID)
	if err != nil {
		return fmt.Errorf("updating saldo CR: %w", err)
	}

	return tx.Commit(ctx)
}

// ---------- Reports R13-R19 ----------

func (r *FinancialRepositoryPG) GetProdutosVendidos(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT fei.item_code, COALESCE(i.mask,'') AS codigo, COALESCE(i.description,'') AS descricao,
		        fei.ncm, SUM(fei.quantity) AS qtd_vendida,
		        ROUND(AVG(fei.unit_price)::numeric,2) AS preco_medio,
		        COALESCE(AVG(sb.avg_cost),0) AS custo_medio,
		        SUM(fei.total_price) AS valor_total,
		        ROUND((AVG(fei.unit_price) - COALESCE(AVG(sb.avg_cost),0))::numeric,2) AS margem_bruta
		 FROM fiscal_exit_items fei
		 JOIN fiscal_exits fe ON fe.id = fei.fiscal_exit_id
		 LEFT JOIN items i ON i.code = fei.item_code
		 LEFT JOIN stock_balances sb ON sb.item_code = fei.item_code
		 WHERE fe.data_emissao BETWEEN $1 AND $2 AND fe.status = 'AUTHORIZED'
		 GROUP BY fei.item_code, i.mask, i.description, fei.ncm
		 ORDER BY valor_total DESC`, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("produtos vendidos: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"item_code", "codigo", "descricao", "ncm", "qtd_vendida", "preco_medio", "custo_medio", "valor_total", "margem_bruta"})
}

func (r *FinancialRepositoryPG) GetProdutosProduzidos(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT po.item_code, COALESCE(i.mask,'') AS codigo, COALESCE(i.description,'') AS descricao,
		        SUM(po.quantity) AS qtd_produzida,
		        COALESCE(AVG(sb.avg_cost),0) AS custo_unitario,
		        COALESCE(AVG(sb.avg_cost),0) * SUM(po.quantity) AS custo_total
		 FROM production_orders po
		 LEFT JOIN items i ON i.code = po.item_code
		 LEFT JOIN stock_balances sb ON sb.item_code = po.item_code
		 WHERE po.created_at BETWEEN $1 AND $2 AND po.status IN ('COMPLETED','CLOSED')
		 GROUP BY po.item_code, i.mask, i.description
		 ORDER BY custo_total DESC`, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("produtos produzidos: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"item_code", "codigo", "descricao", "qtd_produzida", "custo_unitario", "custo_total"})
}

func (r *FinancialRepositoryPG) GetHistoricoCustos(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT sb.item_code, COALESCE(i.mask,'') AS codigo, COALESCE(i.description,'') AS descricao,
		        sb.avg_cost AS custo_medio_atual, sb.last_cost AS custo_ultima_compra,
		        COALESCE((SELECT MIN(unit_price) FROM stock_movements sm WHERE sm.item_code = sb.item_code AND sm.created_at BETWEEN $1 AND $2 AND sm.movement_type='IN'),0) AS custo_minimo,
		        COALESCE((SELECT MAX(unit_price) FROM stock_movements sm WHERE sm.item_code = sb.item_code AND sm.created_at BETWEEN $1 AND $2 AND sm.movement_type='IN'),0) AS custo_maximo
		 FROM stock_balances sb
		 LEFT JOIN items i ON i.code = sb.item_code
		 ORDER BY sb.item_code`, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("historico custos: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"item_code", "codigo", "descricao", "custo_medio_atual", "custo_ultima_compra", "custo_minimo", "custo_maximo"})
}

func (r *FinancialRepositoryPG) GetFichaTecnicaCusto(ctx context.Context, itemCode int64) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT b.child_item_code AS insumo_code, COALESCE(i.mask,'') AS insumo_codigo,
		        COALESCE(i.description,'') AS insumo_descricao,
		        b.quantity AS qtd_por_unidade,
		        COALESCE(sb.avg_cost,0) AS custo_unitario,
		        b.quantity * COALESCE(sb.avg_cost,0) AS custo_total
		 FROM bom_items b
		 JOIN boms bom ON bom.id = b.bom_id
		 LEFT JOIN items i ON i.code = b.child_item_code
		 LEFT JOIN stock_balances sb ON sb.item_code = b.child_item_code
		 WHERE bom.item_code = $1
		 ORDER BY custo_total DESC`, itemCode)
	if err != nil {
		return nil, fmt.Errorf("ficha tecnica: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"insumo_code", "insumo_codigo", "insumo_descricao", "qtd_por_unidade", "custo_unitario", "custo_total"})
}

func (r *FinancialRepositoryPG) GetCurvaABCClientes(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx,
		`WITH vendas AS (
		     SELECT razao_social_destinatario AS cliente, SUM(valor_total) AS total
		     FROM fiscal_exits
		     WHERE status = 'AUTHORIZED' AND data_emissao BETWEEN $1 AND $2
		     GROUP BY razao_social_destinatario
		 ), soma AS (SELECT SUM(total) AS grand_total FROM vendas)
		 SELECT cliente, total,
		        ROUND((total / grand_total * 100)::numeric, 2) AS pct,
		        ROUND(SUM(total) OVER (ORDER BY total DESC) / grand_total * 100::numeric, 2) AS pct_acum,
		        CASE WHEN SUM(total) OVER (ORDER BY total DESC) / grand_total <= 0.8 THEN 'A'
		             WHEN SUM(total) OVER (ORDER BY total DESC) / grand_total <= 0.95 THEN 'B'
		             ELSE 'C' END AS classe
		 FROM vendas, soma ORDER BY total DESC`, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("curva abc clientes: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"cliente", "total", "pct", "pct_acum", "classe"})
}

func (r *FinancialRepositoryPG) GetCurvaABCProdutos(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx,
		`WITH vendas AS (
		     SELECT fei.item_code, COALESCE(i.mask,'') AS codigo, COALESCE(i.description,'') AS descricao,
		            SUM(fei.total_price) AS total
		     FROM fiscal_exit_items fei
		     JOIN fiscal_exits fe ON fe.id = fei.fiscal_exit_id
		     LEFT JOIN items i ON i.code = fei.item_code
		     WHERE fe.status = 'AUTHORIZED' AND fe.data_emissao BETWEEN $1 AND $2
		     GROUP BY fei.item_code, i.mask, i.description
		 ), soma AS (SELECT SUM(total) AS grand_total FROM vendas)
		 SELECT item_code, codigo, descricao, total,
		        ROUND((total / grand_total * 100)::numeric, 2) AS pct,
		        ROUND(SUM(total) OVER (ORDER BY total DESC) / grand_total * 100::numeric, 2) AS pct_acum,
		        CASE WHEN SUM(total) OVER (ORDER BY total DESC) / grand_total <= 0.8 THEN 'A'
		             WHEN SUM(total) OVER (ORDER BY total DESC) / grand_total <= 0.95 THEN 'B'
		             ELSE 'C' END AS classe
		 FROM vendas, soma ORDER BY total DESC`, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("curva abc produtos: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"item_code", "codigo", "descricao", "total", "pct", "pct_acum", "classe"})
}

func (r *FinancialRepositoryPG) GetComprasPeriodo(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT fe.razao_social_emitente AS fornecedor, fei.item_code,
		        COALESCE(i.mask,'') AS codigo_produto, COALESCE(i.description,'') AS descricao, fei.ncm,
		        SUM(fei.quantity) AS qtd, ROUND(AVG(fei.unit_price)::numeric,4) AS preco_unitario,
		        SUM(fei.total_price) AS valor_total, fe.numero_nf AS nfe
		 FROM fiscal_entry_items fei
		 JOIN fiscal_entries fe ON fe.id = fei.fiscal_entry_id
		 LEFT JOIN items i ON i.code = fei.item_code
		 WHERE fe.data_entrada BETWEEN $1 AND $2 AND fe.status = 'APPROVED'
		 GROUP BY fe.razao_social_emitente, fei.item_code, i.mask, i.description, fei.ncm, fe.numero_nf
		 ORDER BY fe.razao_social_emitente, valor_total DESC`, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("compras periodo: %w", err)
	}
	defer rows.Close()
	return scanToMaps(rows, []string{"fornecedor", "item_code", "codigo_produto", "descricao", "ncm", "qtd", "preco_unitario", "valor_total", "nfe"})
}

// ---------- DRE with CMV ----------

func (r *FinancialRepositoryPG) GetDREComCMV(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	var receitaBruta, impostosVendas float64
	_ = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(valor_produtos),0), COALESCE(SUM(valor_icms + valor_ipi + valor_pis + valor_cofins),0)
		 FROM fiscal_exits WHERE data_emissao BETWEEN $1 AND $2 AND status = 'AUTHORIZED'`,
		startDate, endDate).Scan(&receitaBruta, &impostosVendas)

	// CMV: qty sold × avg_cost at time of sale (uses current avg_cost as best approximation)
	var cmv float64
	_ = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(fei.quantity * COALESCE(sb.avg_cost, fei.unit_price * 0.6)), 0)
		 FROM fiscal_exit_items fei
		 JOIN fiscal_exits fe ON fe.id = fei.fiscal_exit_id
		 LEFT JOIN stock_balances sb ON sb.item_code = fei.item_code
		 WHERE fe.data_emissao BETWEEN $1 AND $2 AND fe.status = 'AUTHORIZED'`,
		startDate, endDate).Scan(&cmv)

	var despesasOperacionais float64
	_ = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(valor_bruto - COALESCE(desconto,0)),0)
		 FROM contas_pagar
		 WHERE data_vencimento BETWEEN $1 AND $2 AND status IN ('PAGO') AND tipo_documento NOT IN ('IMPOSTO')`,
		startDate, endDate).Scan(&despesasOperacionais)

	var despesasFinanceiras float64
	_ = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(juros + multa),0) FROM contas_pagar WHERE data_pagamento BETWEEN $1 AND $2 AND status='PAGO'`,
		startDate, endDate).Scan(&despesasFinanceiras)

	var receitasFinanceiras float64
	_ = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(juros + multa),0) FROM contas_receber WHERE data_recebimento BETWEEN $1 AND $2 AND status='RECEBIDO'`,
		startDate, endDate).Scan(&receitasFinanceiras)

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

func (r *FinancialRepositoryPG) SaveExtratoItem(ctx context.Context, contaID int64, data time.Time, valor float64, tipo, descricao, fitid, hash string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO extrato_bancario (conta_bancaria_id, data_transacao, valor, tipo, descricao, fitid, extrato_hash)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (extrato_hash) DO NOTHING`,
		contaID, data, valor, tipo, descricao, fitid, hash)
	if err != nil {
		return fmt.Errorf("saving extrato item: %w", err)
	}
	return nil
}

func (r *FinancialRepositoryPG) GetExtratoPendente(ctx context.Context, contaID int64) ([]map[string]interface{}, error) {
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
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx,
		`UPDATE extrato_bancario SET conciliado=true, fluxo_caixa_id=$1 WHERE id=$2`,
		fluxoID, extratoID)
	if err != nil {
		return fmt.Errorf("updating extrato: %w", err)
	}
	_, err = tx.Exec(ctx,
		`UPDATE fluxo_caixa SET conciliado=true WHERE id=$1`, fluxoID)
	if err != nil {
		return fmt.Errorf("marking fluxo conciliado: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *FinancialRepositoryPG) AutoMatchExtrato(ctx context.Context, contaID int64) (int, error) {
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
			 LIMIT 1`,
			contaID, fcTipo, e.valor, e.data.AddDate(0, 0, -3), e.data.AddDate(0, 0, 3),
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
