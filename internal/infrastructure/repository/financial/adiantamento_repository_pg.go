package financial

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// CreateAdiantamentoAtomico inserts an advance, posts the matching cash-flow
// movement and updates the bank balance in a single transaction. A PAGAR
// advance is cash out (SAIDA); a RECEBER advance is cash in (ENTRADA).
func (r *FinancialRepositoryPG) CreateAdiantamentoAtomico(ctx context.Context, a *entity.Adiantamento, fc entity.FluxoCaixa) (*entity.Adiantamento, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx,
		`INSERT INTO public.adiantamentos
		    (tipo, parceiro_id, conta_bancaria_id, numero_documento, data_adiantamento,
		     valor_original, valor_utilizado, status, descricao, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,0,'ABERTO',$7,$8)
		 RETURNING id, status, valor_utilizado, is_active, created_at, updated_at`,
		string(a.Tipo), a.ParceiroID, a.ContaBancariaID, a.NumeroDocumento, a.DataAdiantamento,
		a.ValorOriginal.InexactFloat64(), a.Descricao, a.CreatedBy,
	).Scan(&a.ID, &a.Status, &a.ValorUtilizado, &a.IsActive, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating adiantamento: %w", err)
	}

	// Cash-flow movement.
	_, err = tx.Exec(ctx,
		`INSERT INTO fluxo_caixa (data, tipo, valor, conta_bancaria_id, descricao, conciliado)
		 VALUES ($1,$2,$3,$4,$5,false)`,
		fc.Data, string(fc.Tipo), fc.Valor.InexactFloat64(), fc.ContaBancariaID, fc.Descricao)
	if err != nil {
		return nil, fmt.Errorf("creating fluxo caixa for adiantamento: %w", err)
	}

	// Update bank balance: ENTRADA adds, SAIDA subtracts.
	sign := "-"
	if fc.Tipo == entity.FluxoCaixaTipoEntrada {
		sign = "+"
	}
	_, err = tx.Exec(ctx,
		fmt.Sprintf(`UPDATE contas_bancarias SET saldo_inicial = saldo_inicial %s $1, updated_at = NOW() WHERE id = $2`, sign),
		fc.Valor.InexactFloat64(), a.ContaBancariaID)
	if err != nil {
		return nil, fmt.Errorf("updating saldo for adiantamento: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return a, nil
}

// AplicarAdiantamentoAtomico applies part (or all) of an advance balance onto a
// conta a pagar / a receber. No cash moves here — the cash already moved when
// the advance was created; this only settles the title against the advance.
func (r *FinancialRepositoryPG) AplicarAdiantamentoAtomico(ctx context.Context, advID int64, contaTipo string, contaID int64, valor decimal.Decimal, userID uuid.UUID, dataAplicacao time.Time) (*entity.AdiantamentoAplicacao, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock and read the advance.
	var advTipo, advStatus string
	var valorOriginal, valorUtilizado float64
	var advActive bool
	err = tx.QueryRow(ctx,
		`SELECT tipo, status, valor_original, valor_utilizado, is_active
		   FROM public.adiantamentos WHERE id = $1 FOR UPDATE`, advID,
	).Scan(&advTipo, &advStatus, &valorOriginal, &valorUtilizado, &advActive)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("adiantamento %d não encontrado", advID)
		}
		return nil, fmt.Errorf("reading adiantamento: %w", err)
	}
	if !advActive || advStatus == string(entity.AdiantamentoStatusCancelado) {
		return nil, fmt.Errorf("adiantamento %d não está disponível para uso", advID)
	}
	if advTipo != contaTipo {
		return nil, fmt.Errorf("tipo do adiantamento (%s) não corresponde ao tipo da conta (%s)", advTipo, contaTipo)
	}

	saldoAdv := decimal.NewFromFloat(valorOriginal).Sub(decimal.NewFromFloat(valorUtilizado))
	if valor.GreaterThan(saldoAdv) {
		return nil, fmt.Errorf("valor %s excede o saldo do adiantamento (%s)", valor.StringFixed(2), saldoAdv.StringFixed(2))
	}

	// Settle the title and validate its remaining balance.
	switch contaTipo {
	case string(entity.AdiantamentoTipoPagar):
		var bruto, desconto, pago, abatido float64
		err = tx.QueryRow(ctx,
			`SELECT valor_bruto, desconto, valor_pago, COALESCE(valor_adiantamento_abatido,0)
			   FROM contas_pagar WHERE id = $1 FOR UPDATE`, contaID,
		).Scan(&bruto, &desconto, &pago, &abatido)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, fmt.Errorf("conta a pagar %d não encontrada", contaID)
			}
			return nil, fmt.Errorf("reading conta pagar: %w", err)
		}
		remaining := decimal.NewFromFloat(bruto).Sub(decimal.NewFromFloat(desconto)).
			Sub(decimal.NewFromFloat(pago)).Sub(decimal.NewFromFloat(abatido))
		if valor.GreaterThan(remaining) {
			return nil, fmt.Errorf("valor %s excede o saldo da conta a pagar (%s)", valor.StringFixed(2), remaining.StringFixed(2))
		}
		novoAbatido := decimal.NewFromFloat(abatido).Add(valor)
		quita := decimal.NewFromFloat(pago).Add(novoAbatido).
			GreaterThanOrEqual(decimal.NewFromFloat(bruto).Sub(decimal.NewFromFloat(desconto)))
		if quita {
			_, err = tx.Exec(ctx,
				`UPDATE contas_pagar SET valor_adiantamento_abatido=$1, adiantamento_id=$2,
				     status='PAGO', data_pagamento=$3, updated_at=NOW() WHERE id=$4`,
				novoAbatido.InexactFloat64(), advID, dataAplicacao, contaID)
		} else {
			_, err = tx.Exec(ctx,
				`UPDATE contas_pagar SET valor_adiantamento_abatido=$1, adiantamento_id=$2, updated_at=NOW() WHERE id=$3`,
				novoAbatido.InexactFloat64(), advID, contaID)
		}
		if err != nil {
			return nil, fmt.Errorf("abatendo conta pagar: %w", err)
		}

	case string(entity.AdiantamentoTipoReceber):
		var bruto, desconto, recebido float64
		err = tx.QueryRow(ctx,
			`SELECT valor_bruto, desconto, valor_recebido
			   FROM contas_receber WHERE id = $1 FOR UPDATE`, contaID,
		).Scan(&bruto, &desconto, &recebido)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, fmt.Errorf("conta a receber %d não encontrada", contaID)
			}
			return nil, fmt.Errorf("reading conta receber: %w", err)
		}
		remaining := decimal.NewFromFloat(bruto).Sub(decimal.NewFromFloat(desconto)).Sub(decimal.NewFromFloat(recebido))
		if valor.GreaterThan(remaining) {
			return nil, fmt.Errorf("valor %s excede o saldo da conta a receber (%s)", valor.StringFixed(2), remaining.StringFixed(2))
		}
		novoRecebido := decimal.NewFromFloat(recebido).Add(valor)
		quita := novoRecebido.GreaterThanOrEqual(decimal.NewFromFloat(bruto).Sub(decimal.NewFromFloat(desconto)))
		if quita {
			_, err = tx.Exec(ctx,
				`UPDATE contas_receber SET valor_recebido=$1, status='RECEBIDO', data_recebimento=$2, updated_at=NOW() WHERE id=$3`,
				novoRecebido.InexactFloat64(), dataAplicacao, contaID)
		} else {
			_, err = tx.Exec(ctx,
				`UPDATE contas_receber SET valor_recebido=$1, updated_at=NOW() WHERE id=$2`,
				novoRecebido.InexactFloat64(), contaID)
		}
		if err != nil {
			return nil, fmt.Errorf("abatendo conta receber: %w", err)
		}

	default:
		return nil, fmt.Errorf("tipo de conta inválido: %s", contaTipo)
	}

	// Update the advance usage / status.
	novoUtilizado := decimal.NewFromFloat(valorUtilizado).Add(valor)
	novoStatus := entity.AdiantamentoStatusParcial
	if novoUtilizado.GreaterThanOrEqual(decimal.NewFromFloat(valorOriginal)) {
		novoStatus = entity.AdiantamentoStatusQuitado
	}
	_, err = tx.Exec(ctx,
		`UPDATE public.adiantamentos SET valor_utilizado=$1, status=$2, updated_at=NOW() WHERE id=$3`,
		novoUtilizado.InexactFloat64(), string(novoStatus), advID)
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
		advID, contaTipo, contaID, valor.InexactFloat64(), dataAplicacao, userID,
	).Scan(&ap.ID, &ap.DataAplicacao, &ap.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("registrando aplicação: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return ap, nil
}

func (r *FinancialRepositoryPG) GetAdiantamento(ctx context.Context, id int64) (*entity.Adiantamento, error) {
	return scanAdiantamentoRow(r.pool.QueryRow(ctx, adiantamentoSelect+` WHERE id = $1`, id))
}

func (r *FinancialRepositoryPG) ListAdiantamentos(ctx context.Context, tipo *string, parceiroID *int64) ([]*entity.Adiantamento, error) {
	rows, err := r.pool.Query(ctx,
		adiantamentoSelect+`
		 WHERE is_active = TRUE
		   AND ($1::varchar IS NULL OR tipo = $1)
		   AND ($2::bigint IS NULL OR parceiro_id = $2)
		 ORDER BY data_adiantamento DESC, id DESC`, tipo, parceiroID)
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
	rows, err := r.pool.Query(ctx,
		`SELECT id, adiantamento_id, conta_tipo, conta_id, valor_aplicado, data_aplicacao, created_by, created_at
		   FROM public.adiantamento_aplicacoes WHERE adiantamento_id = $1 ORDER BY id`, advID)
	if err != nil {
		return nil, fmt.Errorf("listing aplicações: %w", err)
	}
	defer rows.Close()

	var result []*entity.AdiantamentoAplicacao
	for rows.Next() {
		var ap entity.AdiantamentoAplicacao
		var valor float64
		if err := rows.Scan(&ap.ID, &ap.AdiantamentoID, &ap.ContaTipo, &ap.ContaID, &valor, &ap.DataAplicacao, &ap.CreatedBy, &ap.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning aplicação: %w", err)
		}
		ap.ValorAplicado = decimal.NewFromFloat(valor)
		result = append(result, &ap)
	}
	return result, rows.Err()
}

const adiantamentoSelect = `SELECT id, tipo, parceiro_id, conta_bancaria_id, numero_documento, data_adiantamento,
	    valor_original, valor_utilizado, status, descricao, is_active, created_by, created_at, updated_at
	 FROM public.adiantamentos`

func scanAdiantamentoRow(row pgx.Row) (*entity.Adiantamento, error) {
	var a entity.Adiantamento
	var valorOriginal, valorUtilizado float64
	err := row.Scan(&a.ID, &a.Tipo, &a.ParceiroID, &a.ContaBancariaID, &a.NumeroDocumento, &a.DataAdiantamento,
		&valorOriginal, &valorUtilizado, &a.Status, &a.Descricao, &a.IsActive, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("adiantamento não encontrado")
		}
		return nil, fmt.Errorf("scanning adiantamento: %w", err)
	}
	a.ValorOriginal = decimal.NewFromFloat(valorOriginal)
	a.ValorUtilizado = decimal.NewFromFloat(valorUtilizado)
	return &a, nil
}

func scanAdiantamento(rows pgx.Rows) (*entity.Adiantamento, error) {
	var a entity.Adiantamento
	var valorOriginal, valorUtilizado float64
	err := rows.Scan(&a.ID, &a.Tipo, &a.ParceiroID, &a.ContaBancariaID, &a.NumeroDocumento, &a.DataAdiantamento,
		&valorOriginal, &valorUtilizado, &a.Status, &a.Descricao, &a.IsActive, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scanning adiantamento: %w", err)
	}
	a.ValorOriginal = decimal.NewFromFloat(valorOriginal)
	a.ValorUtilizado = decimal.NewFromFloat(valorUtilizado)
	return &a, nil
}
