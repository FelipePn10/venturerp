package financial

import (
	"context"
	"fmt"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/accounting/contabilizacao"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/journal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// CreateAdiantamentoAtomico inserts an advance, posts the matching cash-flow
// movement and updates the bank balance in a single transaction. A PAGAR
// advance is cash out (SAIDA); a RECEBER advance is cash in (ENTRADA).
func (r *FinancialRepositoryPG) CreateAdiantamentoAtomico(ctx context.Context, a *entity.Adiantamento, fc entity.FluxoCaixa, lote *contabilizacao.Lote) (*entity.Adiantamento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	// A conta bancária tem de ser da empresa: sem esta conferência, o id de
	// conta vindo do corpo movimentava o saldo do banco de outra empresa.
	if err := r.contaDaEmpresa(ctx, a.ContaBancariaID); err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx,
		`INSERT INTO public.adiantamentos
		    (tipo, parceiro_id, conta_bancaria_id, numero_documento, data_adiantamento,
		     valor_original, valor_utilizado, status, descricao, created_by, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,0,'ABERTO',$7,$8,$9)
		 RETURNING id, status, valor_utilizado, is_active, created_at, updated_at`,
		string(a.Tipo), a.ParceiroID, a.ContaBancariaID, a.NumeroDocumento, a.DataAdiantamento,
		a.ValorOriginal, a.Descricao, a.CreatedBy, empresa,
	).Scan(&a.ID, &a.Status, &a.ValorUtilizado, &a.IsActive, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating adiantamento: %w", err)
	}

	// Cash-flow movement.
	_, err = tx.Exec(ctx,
		`INSERT INTO fluxo_caixa (data, tipo, valor, conta_bancaria_id, descricao, conciliado, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,false,$6)`,
		fc.Data, string(fc.Tipo), fc.Valor, fc.ContaBancariaID, fc.Descricao, empresa)
	if err != nil {
		return nil, fmt.Errorf("creating fluxo caixa for adiantamento: %w", err)
	}

	// Update bank balance: ENTRADA adds, SAIDA subtracts.
	sign := "-"
	if fc.Tipo == entity.FluxoCaixaTipoEntrada {
		sign = "+"
	}
	_, err = tx.Exec(ctx,
		fmt.Sprintf(`UPDATE contas_bancarias SET saldo_inicial = saldo_inicial %s $1, updated_at = NOW() WHERE id = $2 AND enterprise_id = $3`, sign),
		fc.Valor, a.ContaBancariaID, empresa)
	if err != nil {
		return nil, fmt.Errorf("updating saldo for adiantamento: %w", err)
	}

	if lote != nil {
		lote.SourceID = a.ID
		if err := journal.Gravar(ctx, tx, empresa, *lote); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return a, nil
}

// AplicarAdiantamentoAtomico applies part (or all) of an advance balance onto a
// conta a pagar / a receber. No cash moves here — the cash already moved when
// the advance was created; this only settles the title against the advance.
func (r *FinancialRepositoryPG) AplicarAdiantamentoAtomico(ctx context.Context, advID int64, contaTipo string, contaID int64, valor decimal.Decimal, userID uuid.UUID, dataAplicacao time.Time, lote *contabilizacao.Lote) (*entity.AdiantamentoAplicacao, error) {
	// O título abatido tem de ser da empresa da sessão: a aplicação recebe o id
	// do título direto do corpo, e sem esta cláusula o adiantamento de uma
	// empresa quitaria a conta a pagar da outra.
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock and read the advance.
	var advTipo, advStatus string
	var valorOriginal, valorUtilizado decimal.Decimal
	var advActive bool
	var advParceiro *int64
	err = tx.QueryRow(ctx,
		`SELECT tipo, status, valor_original, valor_utilizado, is_active, parceiro_id
		   FROM public.adiantamentos WHERE id = $1 AND `+adiantamentoDaEmpresa+` FOR UPDATE`, advID, empresa,
	).Scan(&advTipo, &advStatus, &valorOriginal, &valorUtilizado, &advActive, &advParceiro)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("adiantamento %d não encontrado", advID))
		}
		return nil, fmt.Errorf("reading adiantamento: %w", err)
	}
	if !advActive || advStatus == string(entity.AdiantamentoStatusCancelado) {
		return nil, errorsuc.NewConflictError(fmt.Sprintf("o adiantamento %d foi cancelado e não pode ser aplicado", advID))
	}
	if advTipo != contaTipo {
		return nil, errorsuc.NewValidationError("adiantamento pago a fornecedor só abate conta a pagar, e o recebido de cliente só abate conta a receber")
	}

	saldoAdv := valorOriginal.Sub(valorUtilizado)
	if valor.GreaterThan(saldoAdv) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o valor de R$ %s passa do saldo do adiantamento (R$ %s)", valor.StringFixed(2), saldoAdv.StringFixed(2)))
	}
	// O saldo é do parceiro que recebeu ou pagou o adiantamento: abater o título
	// de outro quitaria uma dívida com o dinheiro de um terceiro.
	mesmoParceiro := func(do *int64, rotulo string) error {
		if advParceiro == nil {
			return nil
		}
		if do == nil || *do != *advParceiro {
			return errorsuc.NewValidationError("este título é de outro " + rotulo + ": o adiantamento só abate títulos do " + rotulo + " que o pagou ou recebeu")
		}
		return nil
	}

	// Settle the title and validate its remaining balance.
	switch contaTipo {
	case string(entity.AdiantamentoTipoPagar):
		var bruto, desconto, pago, abatido decimal.Decimal
		var fornecedor *int64
		var status string
		var ativo bool
		err = tx.QueryRow(ctx,
			`SELECT valor_bruto, desconto, valor_pago, COALESCE(valor_adiantamento_abatido,0), fornecedor_id, status, is_active
			   FROM contas_pagar WHERE id = $1 AND enterprise_id = $2 FOR UPDATE`, contaID, empresa,
		).Scan(&bruto, &desconto, &pago, &abatido, &fornecedor, &status, &ativo)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, errorsuc.NewNotFoundError(fmt.Sprintf("conta a pagar %d não encontrada", contaID))
			}
			return nil, fmt.Errorf("reading conta pagar: %w", err)
		}
		if !ativo || status == "PAGO" || status == "CANCELADO" {
			return nil, errorsuc.NewConflictError("esta conta a pagar já está paga ou cancelada")
		}
		if err := mesmoParceiro(fornecedor, "fornecedor"); err != nil {
			return nil, err
		}
		remaining := bruto.Sub(desconto).
			Sub(pago).Sub(abatido)
		if valor.GreaterThan(remaining) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o valor de R$ %s passa do saldo da conta a pagar (R$ %s)", valor.StringFixed(2), remaining.StringFixed(2)))
		}
		novoAbatido := abatido.Add(valor)
		quita := pago.Add(novoAbatido).
			GreaterThanOrEqual(bruto.Sub(desconto))
		if quita {
			_, err = tx.Exec(ctx,
				`UPDATE contas_pagar SET valor_adiantamento_abatido=$1, adiantamento_id=$2,
				     status='PAGO', data_pagamento=$3, updated_at=NOW() WHERE id=$4 AND enterprise_id=$5`,
				novoAbatido, advID, dataAplicacao, contaID, empresa)
		} else {
			_, err = tx.Exec(ctx,
				`UPDATE contas_pagar SET valor_adiantamento_abatido=$1, adiantamento_id=$2, updated_at=NOW() WHERE id=$3 AND enterprise_id=$4`,
				novoAbatido, advID, contaID, empresa)
		}
		if err != nil {
			return nil, fmt.Errorf("abatendo conta pagar: %w", err)
		}

	case string(entity.AdiantamentoTipoReceber):
		var bruto, desconto, recebido decimal.Decimal
		var cliente *int64
		var status string
		var ativo bool
		err = tx.QueryRow(ctx,
			`SELECT valor_bruto, desconto, valor_recebido, cliente_id, status, is_active
			   FROM contas_receber WHERE id = $1 AND enterprise_id = $2 FOR UPDATE`, contaID, empresa,
		).Scan(&bruto, &desconto, &recebido, &cliente, &status, &ativo)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, errorsuc.NewNotFoundError(fmt.Sprintf("conta a receber %d não encontrada", contaID))
			}
			return nil, fmt.Errorf("reading conta receber: %w", err)
		}
		if !ativo || status == "RECEBIDO" || status == "CANCELADO" {
			return nil, errorsuc.NewConflictError("esta conta a receber já está recebida ou cancelada")
		}
		if err := mesmoParceiro(cliente, "cliente"); err != nil {
			return nil, err
		}
		remaining := bruto.Sub(desconto).Sub(recebido)
		if valor.GreaterThan(remaining) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o valor de R$ %s passa do saldo da conta a receber (R$ %s)", valor.StringFixed(2), remaining.StringFixed(2)))
		}
		novoRecebido := recebido.Add(valor)
		quita := novoRecebido.GreaterThanOrEqual(bruto.Sub(desconto))
		if quita {
			_, err = tx.Exec(ctx,
				`UPDATE contas_receber SET valor_recebido=$1, status='RECEBIDO', data_recebimento=$2, updated_at=NOW() WHERE id=$3 AND enterprise_id=$4`,
				novoRecebido, dataAplicacao, contaID, empresa)
		} else {
			_, err = tx.Exec(ctx,
				`UPDATE contas_receber SET valor_recebido=$1, updated_at=NOW() WHERE id=$2 AND enterprise_id=$3`,
				novoRecebido, contaID, empresa)
		}
		if err != nil {
			return nil, fmt.Errorf("abatendo conta receber: %w", err)
		}

	default:
		return nil, errorsuc.NewValidationError(fmt.Sprintf("tipo de conta inválido: %s", contaTipo))
	}

	// Update the advance usage / status.
	novoUtilizado := valorUtilizado.Add(valor)
	novoStatus := entity.AdiantamentoStatusParcial
	if novoUtilizado.GreaterThanOrEqual(valorOriginal) {
		novoStatus = entity.AdiantamentoStatusQuitado
	}
	_, err = tx.Exec(ctx,
		`UPDATE public.adiantamentos SET valor_utilizado=$1, status=$2, updated_at=NOW() WHERE id=$3 AND enterprise_id=$4`,
		novoUtilizado, string(novoStatus), advID, empresa)
	if err != nil {
		return nil, fmt.Errorf("atualizando adiantamento: %w", err)
	}

	// Record the application.
	ap := &entity.AdiantamentoAplicacao{
		AdiantamentoID: advID,
		ContaTipo:      contaTipo,
		ContaID:        contaID,
		ValorAplicado:  valor,
		CreatedBy:      userID,
	}
	err = tx.QueryRow(ctx,
		`INSERT INTO public.adiantamento_aplicacoes
		    (adiantamento_id, conta_tipo, conta_id, valor_aplicado, data_aplicacao, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 RETURNING id, data_aplicacao, created_at`,
		advID, contaTipo, contaID, valor, dataAplicacao, userID,
	).Scan(&ap.ID, &ap.DataAplicacao, &ap.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("registrando aplicação: %w", err)
	}

	if lote != nil {
		lote.SourceID = ap.ID
		if err := journal.Gravar(ctx, tx, empresa, *lote); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return ap, nil
}

// adiantamentoDaEmpresa: $2 nas consultas que o usam é a empresa da sessão.
const adiantamentoDaEmpresa = `adiantamentos.enterprise_id = $2`

func (r *FinancialRepositoryPG) GetAdiantamento(ctx context.Context, id int64) (*entity.Adiantamento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return scanAdiantamentoRow(r.pool.QueryRow(ctx, adiantamentoSelect+` WHERE id = $1 AND `+adiantamentoDaEmpresa, id, empresa))
}

func (r *FinancialRepositoryPG) ListAdiantamentos(ctx context.Context, tipo *string, parceiroID *int64) ([]*entity.Adiantamento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		adiantamentoSelect+`
		 WHERE is_active = TRUE
		   AND ($1::varchar IS NULL OR tipo = $1)
		   AND ($3::bigint IS NULL OR parceiro_id = $3)
		   AND `+adiantamentoDaEmpresa+`
		 ORDER BY data_adiantamento DESC, id DESC`, tipo, empresa, parceiroID)
	if err != nil {
		return nil, fmt.Errorf("listing adiantamentos: %w", err)
	}
	defer rows.Close()

	var result []*entity.Adiantamento
	for rows.Next() {
		a, err := scanAdiantamento(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (r *FinancialRepositoryPG) ListAplicacoesByAdiantamento(ctx context.Context, advID int64) ([]*entity.AdiantamentoAplicacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, adiantamento_id, conta_tipo, conta_id, valor_aplicado, data_aplicacao, created_by, created_at
		   FROM public.adiantamento_aplicacoes
		  WHERE adiantamento_id = $1
		    AND EXISTS (SELECT 1 FROM public.adiantamentos WHERE adiantamentos.id = $1 AND `+adiantamentoDaEmpresa+`)
		  ORDER BY id`, advID, empresa)
	if err != nil {
		return nil, fmt.Errorf("listing aplicações: %w", err)
	}
	defer rows.Close()

	var result []*entity.AdiantamentoAplicacao
	for rows.Next() {
		var ap entity.AdiantamentoAplicacao
		var valor decimal.Decimal
		if err := rows.Scan(&ap.ID, &ap.AdiantamentoID, &ap.ContaTipo, &ap.ContaID, &valor, &ap.DataAplicacao, &ap.CreatedBy, &ap.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning aplicação: %w", err)
		}
		ap.ValorAplicado = valor
		result = append(result, &ap)
	}
	return result, rows.Err()
}

const adiantamentoSelect = `SELECT id, tipo, parceiro_id, COALESCE(conta_bancaria_id, 0), numero_documento, data_adiantamento,
	    valor_original, valor_utilizado, status, descricao, is_active, created_by, created_at, updated_at
	 FROM public.adiantamentos`

func scanAdiantamentoRow(row pgx.Row) (*entity.Adiantamento, error) {
	var a entity.Adiantamento
	var valorOriginal, valorUtilizado decimal.Decimal
	err := row.Scan(&a.ID, &a.Tipo, &a.ParceiroID, &a.ContaBancariaID, &a.NumeroDocumento, &a.DataAdiantamento,
		&valorOriginal, &valorUtilizado, &a.Status, &a.Descricao, &a.IsActive, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errorsuc.NewNotFoundError("adiantamento não encontrado")
		}
		return nil, fmt.Errorf("scanning adiantamento: %w", err)
	}
	a.ValorOriginal = valorOriginal
	a.ValorUtilizado = valorUtilizado
	return &a, nil
}

func scanAdiantamento(rows pgx.Rows) (*entity.Adiantamento, error) {
	var a entity.Adiantamento
	var valorOriginal, valorUtilizado decimal.Decimal
	err := rows.Scan(&a.ID, &a.Tipo, &a.ParceiroID, &a.ContaBancariaID, &a.NumeroDocumento, &a.DataAdiantamento,
		&valorOriginal, &valorUtilizado, &a.Status, &a.Descricao, &a.IsActive, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scanning adiantamento: %w", err)
	}
	a.ValorOriginal = valorOriginal
	a.ValorUtilizado = valorUtilizado
	return &a, nil
}

// CancelarAdiantamentoAtomico desfaz um adiantamento lançado errado: o dinheiro
// volta à conta bancária (movimento inverso no fluxo de caixa) e o saldo deixa de
// poder ser aplicado. Só vale para adiantamento sem aplicação: depois de abater
// um título, desfazer a aplicação vem antes.
func (r *FinancialRepositoryPG) CancelarAdiantamentoAtomico(ctx context.Context, id int64, motivo string, userID uuid.UUID) (*entity.Adiantamento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var tipo, status string
	var valor, utilizado decimal.Decimal
	var conta int64
	err = tx.QueryRow(ctx,
		`SELECT tipo, status, valor_original, valor_utilizado, conta_bancaria_id
		   FROM public.adiantamentos WHERE id = $1 AND `+adiantamentoDaEmpresa+` AND is_active FOR UPDATE`, id, empresa,
	).Scan(&tipo, &status, &valor, &utilizado, &conta)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("adiantamento %d não encontrado", id))
		}
		return nil, fmt.Errorf("reading adiantamento: %w", err)
	}
	if status == string(entity.AdiantamentoStatusCancelado) {
		return nil, errorsuc.NewConflictError("este adiantamento já está cancelado")
	}
	if utilizado.IsPositive() {
		return nil, errorsuc.NewConflictError(fmt.Sprintf("este adiantamento já abateu R$ %s em títulos e não pode ser cancelado", utilizado.StringFixed(2)))
	}

	// O inverso do registro: o pago a fornecedor volta para o banco, o recebido de
	// cliente sai.
	fcTipo, sinal := entity.FluxoCaixaTipoEntrada, "+"
	if tipo == string(entity.AdiantamentoTipoReceber) {
		fcTipo, sinal = entity.FluxoCaixaTipoSaida, "-"
	}
	desc := fmt.Sprintf("Cancelamento do adiantamento %d: %s", id, motivo)
	if _, err = tx.Exec(ctx,
		`INSERT INTO fluxo_caixa (data, tipo, valor, conta_bancaria_id, descricao, conciliado, enterprise_id)
		 VALUES (CURRENT_DATE,$1,$2,$3,$4,false,$5)`, string(fcTipo), valor, conta, desc, empresa); err != nil {
		return nil, fmt.Errorf("estornando o fluxo de caixa do adiantamento: %w", err)
	}
	if _, err = tx.Exec(ctx,
		fmt.Sprintf(`UPDATE contas_bancarias SET saldo_inicial = saldo_inicial %s $1, updated_at = NOW() WHERE id = $2 AND enterprise_id = $3`, sinal),
		valor, conta, empresa); err != nil {
		return nil, fmt.Errorf("estornando o saldo bancário do adiantamento: %w", err)
	}
	if _, err = tx.Exec(ctx,
		`UPDATE public.adiantamentos
		    SET status = 'CANCELADO', updated_at = NOW(),
		        descricao = CONCAT_WS(' — ', NULLIF(descricao, ''), $3::text)
		  WHERE id = $1 AND enterprise_id = $2`, id, empresa, "Cancelado: "+motivo); err != nil {
		return nil, fmt.Errorf("cancelando o adiantamento: %w", err)
	}
	// A entrada contabilizada (se houve) é estornada na mesma transação, com a
	// data do banco (a mesma CURRENT_DATE do fluxo de caixa acima).
	var hoje time.Time
	if err := tx.QueryRow(ctx, `SELECT CURRENT_DATE`).Scan(&hoje); err != nil {
		return nil, fmt.Errorf("lendo a data do estorno: %w", err)
	}
	if _, err := journal.Estornar(ctx, tx, empresa, contabilizacao.OrigemAdiantamento, id,
		contabilizacao.OrigemAdiantamentoEstorno, "EAD", hoje); err != nil {
		return nil, fmt.Errorf("estornando a contabilização do adiantamento: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	_ = userID
	return r.GetAdiantamento(ctx, id)
}
