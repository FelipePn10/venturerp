package fiscal

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/journal"
	stockrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

var _ repository.DevolucaoRepository = (*FiscalRepositoryPG)(nil)

// NewDevolucaoRepositoryPG expõe o repositório fiscal pelo contrato da devolução.
func NewDevolucaoRepositoryPG(r repository.FiscalRepository) repository.DevolucaoRepository {
	return r.(*FiscalRepositoryPG)
}

func (r *FiscalRepositoryPG) QuantidadesDevolvidas(ctx context.Context, entryID int64) (map[int64]decimal.Decimal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT i.fiscal_entry_item_id, SUM(i.quantity)
		   FROM fiscal_exit_items i JOIN fiscal_exits x ON x.id = i.fiscal_exit_id
		  WHERE x.enterprise_id=$1 AND x.fiscal_entry_id=$2 AND x.is_active
		    AND x.status NOT IN ('CANCELLED','REJECTED') AND i.fiscal_entry_item_id IS NOT NULL
		  GROUP BY i.fiscal_entry_item_id`, empresa, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]decimal.Decimal{}
	for rows.Next() {
		var id int64
		var q decimal.Decimal
		if err := rows.Scan(&id, &q); err != nil {
			return nil, err
		}
		out[id] = q
	}
	return out, rows.Err()
}

// EfetivarDevolucao: a mercadoria sai do estoque (pelo custo médio); o valor
// devolvido abate os títulos em aberto da nota, do último vencimento para o
// primeiro (o título zerado fica PAGO); o que sobrar vira um título a receber
// do fornecedor; e a contabilização é gravada. Tudo numa transação, e uma vez
// só por nota.
func (r *FiscalRepositoryPG) EfetivarDevolucao(ctx context.Context, e repository.EfetivacaoDevolucao) (*repository.ResultadoDevolucao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT id FROM fiscal_exits WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, e.ExitID, empresa); err != nil {
		return nil, err
	}
	var ja bool
	if err = tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM stock_movements WHERE enterprise_id=$1 AND reference_type=$2 AND reference_code=$3)
		     OR EXISTS (SELECT 1 FROM fiscal_return_settlements WHERE enterprise_id=$1 AND fiscal_exit_id=$3)`,
		empresa, stockentity.ReferenceTypeNFExit, e.ExitID).Scan(&ja); err != nil {
		return nil, err
	}
	if ja {
		return &repository.ResultadoDevolucao{}, nil
	}
	res := &repository.ResultadoDevolucao{}

	// Estoque: sai pelo custo médio do momento.
	ref := stockentity.ReferenceTypeNFExit
	nota := fmt.Sprintf("devolução de compra — NF-e %d", e.NumeroNF)
	custos := map[int64]decimal.Decimal{}
	for _, s := range e.Saidas {
		if !s.Quantidade.IsPositive() {
			continue
		}
		q, _ := s.Quantidade.Float64()
		m := &stockentity.StockMovement{ItemCode: s.ItemCode, WarehouseID: s.WarehouseID, MovementType: stockentity.MovementTypeOut,
			Quantity: q, ReferenceType: &ref, ReferenceCode: &e.ExitID, Notes: &nota, CreatedBy: e.UserID}
		if err := stockrepo.CreateMovementTx(ctx, tx, empresa, m); err != nil {
			return nil, fmt.Errorf("baixando o estoque da devolução: %w", err)
		}
		custos[s.FiscalEntryItemID] = custos[s.FiscalEntryItemID].Add(decimal.NewFromFloat(m.TotalPrice))
		res.Movimentos++
	}

	// Abatimento dos títulos em aberto da nota.
	rows, err := tx.Query(ctx,
		`SELECT id, valor_bruto - COALESCE(desconto,0) - COALESCE(valor_pago,0) - COALESCE(valor_adiantamento_abatido,0)
		   FROM contas_pagar
		  WHERE enterprise_id=$1 AND fiscal_entry_id=$2 AND is_active AND tipo_documento <> 'RETENCAO'
		    AND status IN ('PENDENTE','APROVADO','VENCIDO')
		  ORDER BY data_vencimento DESC, id DESC FOR UPDATE`, empresa, e.EntryID)
	if err != nil {
		return nil, err
	}
	type aberto struct {
		id    int64
		saldo decimal.Decimal
	}
	var abertos []aberto
	for rows.Next() {
		var a aberto
		if err := rows.Scan(&a.id, &a.saldo); err != nil {
			rows.Close()
			return nil, err
		}
		if a.saldo.IsPositive() {
			abertos = append(abertos, a)
		}
	}
	rows.Close()
	restante := e.Valor.Round(2)
	obs := fmt.Sprintf("abatimento da devolução NF-e %d", e.NumeroNF)
	for _, a := range abertos {
		if !restante.IsPositive() {
			break
		}
		v := decimal.Min(restante, a.saldo)
		quita := v.Equal(a.saldo)
		if _, err := tx.Exec(ctx,
			`UPDATE contas_pagar SET desconto = COALESCE(desconto,0) + $3,
			     status = CASE WHEN $4 THEN 'PAGO' ELSE status END,
			     data_pagamento = CASE WHEN $4 THEN $5::date ELSE data_pagamento END,
			     observacao = left(COALESCE(observacao || ' | ', '') || $6, 1000), updated_at = NOW()
			  WHERE id=$1 AND enterprise_id=$2`, a.id, empresa, v, quita, e.Data, obs); err != nil {
			return nil, fmt.Errorf("abatendo o título %d: %w", a.id, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO fiscal_return_settlements (enterprise_id, fiscal_exit_id, conta_pagar_id, valor, quitou_titulo) VALUES ($1,$2,$3,$4,$5)`,
			empresa, e.ExitID, a.id, v, quita); err != nil {
			return nil, err
		}
		restante = restante.Sub(v)
		res.Abatido = res.Abatido.Add(v)
	}
	// O que não havia como abater (títulos já pagos) o fornecedor devolve.
	if restante.IsPositive() {
		doc := fmt.Sprintf("DEV-NF-%d", e.NumeroNF)
		var crID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO contas_receber (numero_documento, fornecedor_id, fiscal_exit_id, data_lancamento, data_emissao, data_vencimento,
			     valor_bruto, desconto, juros, multa, valor_recebido, parcela_numero, parcela_total, status, is_active, criado_por, enterprise_id)
			 VALUES ($1,$2,$3,$4::date,$4::date,$5::date,$6,0,0,0,0,1,1,'PENDENTE',TRUE,$7,$8) RETURNING id`,
			doc, e.FornecedorCode, e.ExitID, e.Data, e.Data.AddDate(0, 0, 30), restante, e.UserID, empresa).Scan(&crID); err != nil {
			return nil, fmt.Errorf("gerando o crédito a receber do fornecedor: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO fiscal_return_settlements (enterprise_id, fiscal_exit_id, conta_receber_id, valor) VALUES ($1,$2,$3,$4)`,
			empresa, e.ExitID, crID, restante); err != nil {
			return nil, err
		}
		res.CreditoReceber, res.ContaReceberID = restante, &crID
	}

	if e.Montar != nil {
		lote, err := e.Montar(custos)
		if err != nil {
			return nil, err
		}
		if lote != nil {
			if err := journal.Gravar(ctx, tx, empresa, *lote); err != nil {
				return nil, err
			}
		}
	}
	return res, tx.Commit(ctx)
}

// EstornarDevolucao reabre o que a devolução abateu, cancela o crédito a
// receber (recusa se o fornecedor já pagou) e estorna a contabilização.
func (r *FiscalRepositoryPG) EstornarDevolucao(ctx context.Context, exitID int64, data time.Time) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx,
		`SELECT id, conta_pagar_id, conta_receber_id, valor, quitou_titulo FROM fiscal_return_settlements
		  WHERE enterprise_id=$1 AND fiscal_exit_id=$2 AND estornado_em IS NULL ORDER BY id FOR UPDATE`, empresa, exitID)
	if err != nil {
		return err
	}
	type acerto struct {
		id     int64
		cp, cr *int64
		valor  decimal.Decimal
		quitou bool
	}
	var as []acerto
	for rows.Next() {
		var a acerto
		if err := rows.Scan(&a.id, &a.cp, &a.cr, &a.valor, &a.quitou); err != nil {
			rows.Close()
			return err
		}
		as = append(as, a)
	}
	rows.Close()
	for _, a := range as {
		if a.cp != nil {
			var pago decimal.Decimal
			var status string
			if err := tx.QueryRow(ctx, `SELECT COALESCE(valor_pago,0), status FROM contas_pagar WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, *a.cp, empresa).Scan(&pago, &status); err != nil {
				return err
			}
			if a.quitou && pago.IsPositive() {
				return errorsuc.NewValidationError(fmt.Sprintf("o título %d foi pago depois do abatimento da devolução: estorne o pagamento antes", *a.cp))
			}
			if _, err := tx.Exec(ctx,
				`UPDATE contas_pagar SET desconto = GREATEST(COALESCE(desconto,0) - $3, 0),
				     status = CASE WHEN $4 THEN 'PENDENTE' ELSE status END,
				     data_pagamento = CASE WHEN $4 THEN NULL ELSE data_pagamento END, updated_at = NOW()
				  WHERE id=$1 AND enterprise_id=$2`, *a.cp, empresa, a.valor, a.quitou); err != nil {
				return err
			}
		}
		if a.cr != nil {
			var recebido decimal.Decimal
			if err := tx.QueryRow(ctx, `SELECT COALESCE(valor_recebido,0) FROM contas_receber WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, *a.cr, empresa).Scan(&recebido); err != nil {
				return err
			}
			if recebido.IsPositive() {
				return errorsuc.NewValidationError("o fornecedor já pagou o crédito da devolução: estorne o recebimento antes de cancelar")
			}
			if _, err := tx.Exec(ctx, `UPDATE contas_receber SET status='CANCELADO', is_active=FALSE, updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, *a.cr, empresa); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE fiscal_return_settlements SET estornado_em=NOW() WHERE id=$1`, a.id); err != nil {
			return err
		}
	}
	if _, err := journal.Estornar(ctx, tx, empresa, repository.OrigemDevolucao, exitID, repository.OrigemDevolucaoEstorno, "DEVE", data); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *FiscalRepositoryPG) ChecarEstornoDevolucao(ctx context.Context, exitID int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var cpPago, crRecebido bool
	err = r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM fiscal_return_settlements s JOIN contas_pagar c ON c.id = s.conta_pagar_id
		                 WHERE s.enterprise_id=$1 AND s.fiscal_exit_id=$2 AND s.estornado_em IS NULL AND s.quitou_titulo AND COALESCE(c.valor_pago,0) > 0),
		        EXISTS (SELECT 1 FROM fiscal_return_settlements s JOIN contas_receber c ON c.id = s.conta_receber_id
		                 WHERE s.enterprise_id=$1 AND s.fiscal_exit_id=$2 AND s.estornado_em IS NULL AND COALESCE(c.valor_recebido,0) > 0)`,
		empresa, exitID).Scan(&cpPago, &crRecebido)
	if err != nil {
		return err
	}
	if cpPago {
		return errorsuc.NewValidationError("um título abatido pela devolução foi pago depois: estorne o pagamento antes de cancelar a devolução")
	}
	if crRecebido {
		return errorsuc.NewValidationError("o fornecedor já pagou o crédito da devolução: estorne o recebimento antes de cancelar")
	}
	return nil
}
