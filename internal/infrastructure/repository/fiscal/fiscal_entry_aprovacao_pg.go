package fiscal

// Aprovação e cancelamento da nota de entrada.
//
// Este arquivo é um coordenador de infraestrutura: a aprovação mexe em quatro
// agregados (nota, contas a pagar, estoque, pedido de compra) e na
// contabilidade, e precisa ser tudo ou nada. Por isso grava com SQL próprio e
// usa stockrepo.CreateMovementTx — o movimento de estoque é o mesmo do módulo
// de estoque (saldo e custo médio), só que dentro desta transação.

import (
	"context"
	"errors"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/journal"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	poentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

var epsQtd = decimal.NewFromFloat(0.0001)

type linhaPedidoTx struct {
	code, pedido                                      int64
	requested, received, invoiced, cancelled, interno decimal.Decimal
	status, situacaoPedido                            string
}

func lockLinhaPedido(ctx context.Context, tx pgx.Tx, empresaCode, lineCode int64) (*linhaPedidoTx, error) {
	var l linhaPedidoTx
	err := tx.QueryRow(ctx,
		`SELECT poi.code, poi.purchase_order_code, poi.requested_qty, poi.received_qty, poi.invoiced_qty, poi.cancelled_qty,
		        COALESCE(poi.internal_qty,0), poi.status, po.status
		   FROM purchase_order_items poi
		   JOIN purchase_orders po ON po.code = poi.purchase_order_code AND po.enterprise_code = $2
		  WHERE poi.code = $1
		  FOR UPDATE OF poi`, lineCode, empresaCode).Scan(
		&l.code, &l.pedido, &l.requested, &l.received, &l.invoiced, &l.cancelled, &l.interno, &l.status, &l.situacaoPedido)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a linha %d do pedido de compra não existe nesta empresa", lineCode))
	}
	if err != nil {
		return nil, fmt.Errorf("travando linha do pedido de compra: %w", err)
	}
	return &l, nil
}

// gravarLinhaPedido grava recebido/faturado da linha e recalcula a situação
// da linha e do pedido (mesma regra do recebimento físico).
func gravarLinhaPedido(ctx context.Context, tx pgx.Tx, empresaCode int64, l *linhaPedidoTx) error {
	if l.received.IsNegative() {
		l.received = decimal.Zero
	}
	if l.invoiced.IsNegative() {
		l.invoiced = decimal.Zero
	}
	status := "OPEN"
	switch {
	case l.received.Add(l.cancelled).GreaterThanOrEqual(l.requested.Sub(epsQtd)) && l.received.IsPositive():
		status = "RECEIVED"
	case l.received.IsPositive():
		status = "PARTIAL"
	}
	if _, err := tx.Exec(ctx,
		`UPDATE purchase_order_items SET received_qty=$2, invoiced_qty=$3, status=$4, updated_at=NOW() WHERE code=$1`,
		l.code, l.received, l.invoiced, status); err != nil {
		return fmt.Errorf("atualizando linha do pedido de compra: %w", err)
	}
	var todas, alguma bool
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(bool_and(status IN ('RECEIVED','CANCELLED')), false), COALESCE(bool_or(received_qty > 0), false)
		   FROM purchase_order_items WHERE purchase_order_code=$1 AND is_active`, l.pedido).Scan(&todas, &alguma); err != nil {
		return err
	}
	// Sem nada recebido (estorno da nota) o pedido volta a aprovado — "OPEN"
	// não é situação de capa de pedido de compra.
	cab := "APPROVED"
	switch {
	case todas && alguma:
		cab = "RECEIVED"
	case alguma:
		cab = "PARTIAL"
	}
	_, err := tx.Exec(ctx,
		`UPDATE purchase_orders SET status = CASE WHEN status IN ('CANCELLED') THEN status ELSE $2 END, updated_at=NOW()
		  WHERE code=$1 AND enterprise_code=$3`, l.pedido, cab, empresaCode)
	return err
}

func (r *FiscalRepositoryPG) AprovarEntrada(ctx context.Context, a repository.AprovacaoEntrada) (*repository.ResultadoAprovacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	empresaCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var atual string
	var numero int64
	var dataEntrada time.Time
	if err = tx.QueryRow(ctx, `SELECT status, numero_nf, data_entrada FROM fiscal_entries WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`,
		a.EntryID, empresa).Scan(&atual, &numero, &dataEntrada); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("nota fiscal de entrada %d não encontrada", a.EntryID))
		}
		return nil, err
	}
	if atual != string(entity.EntryStatusPending) && atual != string(entity.EntryStatusConferred) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a nota está %s: só nota pendente ou conferida pode ser aprovada", atual))
	}
	res := &repository.ResultadoAprovacao{}

	// 1. Títulos do contas a pagar (fornecedor e retenções) com rateio.
	for _, t := range a.Titulos {
		tipo := t.TipoDocumento
		if tipo == "" {
			tipo = "NFE"
		}
		var cpID int64
		err = tx.QueryRow(ctx,
			`INSERT INTO contas_pagar
				(numero_documento, tipo_documento, fornecedor_id, fornecedor_cnpj, fiscal_entry_id,
				 data_lancamento, data_emissao, data_vencimento,
				 valor_bruto, desconto, juros, multa, valor_pago,
				 parcela_numero, parcela_total, forma_pagamento, plano_contas_id, centro_custo_id,
				 status_aprovacao, status, observacao, is_active, criado_por, enterprise_id, retencao_tipo)
			 VALUES ($1,$2,$3,$4,$5, CURRENT_DATE,$6,$7, $8,0,0,0,0, $9,$10,$11,$12,$13,
			         'PENDENTE','PENDENTE',$14,TRUE,$15,$16,NULLIF($17,''))
			 RETURNING id`,
			truncarStr(t.NumeroDocumento, 60), tipo, t.FornecedorCode, nuloSeVazio(t.FornecedorCNPJ), a.EntryID,
			t.DataEmissao, t.DataVencimento, t.Valor,
			t.ParcelaNumero, t.ParcelaTotal, truncar(t.FormaPagamento, 20), t.PlanoContasID, t.CentroCustoID,
			t.Observacao, a.UserID, empresa, t.RetencaoTipo,
		).Scan(&cpID)
		if err != nil {
			return nil, fmt.Errorf("gerando conta a pagar %s: %w", t.NumeroDocumento, err)
		}
		for _, al := range t.Rateio {
			if !al.Valor.IsPositive() {
				continue
			}
			if _, err = tx.Exec(ctx,
				`INSERT INTO contas_pagar_rateios (enterprise_id, conta_pagar_id, plano_contas_id, centro_custo_id, valor)
				 VALUES ($1,$2,$3,$4,$5)`, empresa, cpID, al.PlanoContasID, al.CentroCustoID, al.Valor); err != nil {
				return nil, fmt.Errorf("gravando rateio de %s: %w", t.NumeroDocumento, err)
			}
		}
		if t.InstallmentID > 0 {
			if _, err = tx.Exec(ctx, `UPDATE fiscal_entry_installments SET conta_pagar_id=$2, updated_at=NOW() WHERE id=$1 AND enterprise_id=$3`,
				t.InstallmentID, cpID, empresa); err != nil {
				return nil, err
			}
		}
		res.TitulosGerados++
	}

	// 2. Custo de aquisição por item.
	for itemID, custo := range a.Custos {
		if _, err = tx.Exec(ctx, `UPDATE fiscal_entry_items SET custo_aquisicao=$3 WHERE id=$1 AND fiscal_entry_id=$2`, itemID, a.EntryID, custo); err != nil {
			return nil, err
		}
	}

	// 3. Pedido de compra (3-way) e estoque.
	ref := stockentity.ReferenceTypeNFEntry
	algumEstoque := false
	for _, m := range a.Movimentos {
		coberto := decimal.Zero
		var pedido *int64
		if m.PurchaseOrderItemCode != nil {
			l, err := lockLinhaPedido(ctx, tx, empresaCode, *m.PurchaseOrderItemCode)
			if err != nil {
				return nil, err
			}
			if l.status == "CANCELLED" {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("a linha %d do pedido de compra está cancelada", l.code))
			}
			if err := poentity.SituacaoAceitaRecebimento(l.pedido, poentity.PurchaseOrderStatus(l.situacaoPedido)); err != nil {
				return nil, errorsuc.NewValidationError(err.Error())
			}
			pedido = &l.pedido
			fator := m.FatorPedido
			if !fator.IsPositive() {
				fator = decimal.NewFromInt(1)
			}
			qtdPedido := m.Quantidade.Div(fator)
			// Recebido fisicamente e ainda não faturado: já está no estoque.
			pendenteFisico := l.received.Sub(l.invoiced)
			if pendenteFisico.IsNegative() {
				pendenteFisico = decimal.Zero
			}
			cobertoPedido := decimal.Min(qtdPedido, pendenteFisico)
			coberto = cobertoPedido.Mul(fator).Round(6)
			l.invoiced = l.invoiced.Add(qtdPedido)
			l.received = l.received.Add(qtdPedido.Sub(cobertoPedido))
			if err := gravarLinhaPedido(ctx, tx, empresaCode, l); err != nil {
				return nil, err
			}
			res.QtdJaRecebida = res.QtdJaRecebida.Add(coberto)
		}
		paraEstoque := m.Quantidade.Sub(coberto)
		var movID *int64
		if m.MovimentaEstoque {
			algumEstoque = true
		}
		if m.MovimentaEstoque && paraEstoque.GreaterThan(epsQtd) {
			unit := decimal.Zero
			if m.Quantidade.IsPositive() {
				unit = m.CustoTotal.Div(m.Quantidade)
			}
			total := unit.Mul(paraEstoque).Round(2)
			nota := fmt.Sprintf("Entrada da NF %d (nota de entrada %d)", numero, a.EntryID)
			mov := &stockentity.StockMovement{
				ItemCode:      m.ItemCode,
				WarehouseID:   m.WarehouseID,
				MovementType:  stockentity.MovementTypeIn,
				Quantity:      paraEstoque.InexactFloat64(),
				ExactQuantity: paraEstoque,
				UnitPrice:     unit.Round(6).InexactFloat64(),
				TotalPrice:    total.InexactFloat64(),
				ReferenceType: &ref,
				ReferenceCode: &a.EntryID,
				Notes:         &nota,
				CreatedBy:     a.UserID,
			}
			if err := stockrepo.CreateMovementTx(ctx, tx, empresa, mov); err != nil {
				return nil, fmt.Errorf("dando entrada no estoque do item %d: %w", m.ItemCode, err)
			}
			movID = &mov.ID
			res.MovimentosGerados++
			res.Movimentados = append(res.Movimentados, repository.MovimentoGerado{
				ItemID: m.ItemID, ItemCode: m.ItemCode, WarehouseID: m.WarehouseID, Quantidade: paraEstoque,
				PurchaseOrderCode: pedido, PurchaseOrderItemCode: m.PurchaseOrderItemCode,
			})
		}
		if _, err = tx.Exec(ctx, `UPDATE fiscal_entry_items SET stock_movement_id=$3, qtd_recebida_antes=$4 WHERE id=$1 AND fiscal_entry_id=$2`,
			m.ItemID, a.EntryID, movID, coberto); err != nil {
			return nil, err
		}
	}

	// 4. Contabilização.
	for i, l := range a.Lancamentos {
		if !l.Valor.IsPositive() {
			continue
		}
		if _, err = tx.Exec(ctx,
			`INSERT INTO accounting_journal_entries
				(plan_id, empresa_id, entry_date, entry_number, batch_number, debit_account_id, credit_account_id,
				 debit_cc_id, credit_cc_id, value, history_code, description, entry_type, source_type, source_id)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'',$11,'NFE_ENTRADA','NFE_ENTRADA',$12)`,
			l.PlanID, empresa, dataEntrada, fmt.Sprintf("NFE%d-%d", a.EntryID, i+1), fmt.Sprintf("NFE%d", numero),
			l.DebitoID, l.CreditoID, l.DebitoCC, l.CreditoCC, l.Valor, truncarStr(l.Historico, 500), a.EntryID); err != nil {
			return nil, fmt.Errorf("gravando lançamento contábil: %w", err)
		}
		res.LancamentosGerados++
	}

	stockStatus := entity.StockStatusNaoAplica
	if algumEstoque {
		stockStatus = entity.StockStatusConcluido
	}
	if _, err = tx.Exec(ctx,
		`UPDATE fiscal_entries SET status='APPROVED', approved_at=NOW(), approved_by=$3, stock_status=$4, updated_at=NOW()
		  WHERE id=$1 AND enterprise_id=$2`, a.EntryID, empresa, a.UserID, stockStatus); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return res, nil
}

func (r *FiscalRepositoryPG) CancelarEntrada(ctx context.Context, c repository.CancelamentoEntrada) (*repository.ResultadoCancelamento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	empresaCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var atual string
	var numero int64
	res := &repository.ResultadoCancelamento{}
	if err = tx.QueryRow(ctx, `SELECT status, numero_nf, data_entrada FROM fiscal_entries WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`,
		c.EntryID, empresa).Scan(&atual, &numero, &res.DataEntrada); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("nota fiscal de entrada %d não encontrada", c.EntryID))
		}
		return nil, err
	}
	// Frete e devolução apoiados na nota precisam sair antes dela: o frete
	// complementou o custo dos itens desta nota, a devolução tirou parte deles
	// do estoque e abateu os títulos.
	if atual != string(entity.EntryStatusCancelled) {
		var fretes, devolucoes []string
		if err = tx.QueryRow(ctx,
			`SELECT COALESCE(array_agg(DISTINCT f.numero::text || '/' || f.serie) FILTER (WHERE f.id IS NOT NULL), '{}'),
			        COALESCE((SELECT array_agg(x.numero_nf::text || '/' || x.serie ORDER BY x.id) FROM fiscal_exits x
			                   WHERE x.enterprise_id=$2 AND x.fiscal_entry_id=$1 AND x.finalidade=4 AND x.is_active
			                     AND x.status NOT IN ('CANCELLED','REJECTED')), '{}')
			   FROM fiscal_freight_document_entries fe
			   LEFT JOIN fiscal_freight_documents f ON f.id = fe.freight_id AND f.status <> 'CANCELADO'
			  WHERE fe.fiscal_entry_id=$1 AND fe.enterprise_id=$2`, c.EntryID, empresa).Scan(&fretes, &devolucoes); err != nil {
			return nil, err
		}
		var motivos []string
		if len(fretes) > 0 {
			motivos = append(motivos, "o CT-e de frete "+strings.Join(fretes, ", ")+" (cancele o frete em VFIS0210 › Fretes)")
		}
		if len(devolucoes) > 0 {
			motivos = append(motivos, "a NF-e de devolução "+strings.Join(devolucoes, ", ")+" (cancele-a em VFIS0200)")
		}
		if len(motivos) > 0 {
			return nil, errorsuc.NewValidationError("a nota não pode ser cancelada enquanto tiver " + strings.Join(motivos, " e "))
		}
	}
	switch atual {
	case string(entity.EntryStatusCancelled):
		return nil, errorsuc.NewValidationError("a nota já está cancelada")
	case string(entity.EntryStatusPending), string(entity.EntryStatusConferred):
		// Nada foi efetivado: só sai de cena (e a chave fica livre para reimportar).
	case string(entity.EntryStatusApproved):
		res.EraAprovada = true
		if err := estornarAprovacao(ctx, tx, empresa, empresaCode, c, numero, res); err != nil {
			return nil, err
		}
	default:
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a nota está %s e não pode ser cancelada", atual))
	}

	if _, err = tx.Exec(ctx,
		`UPDATE fiscal_entries SET status='CANCELLED', cancelled_at=NOW(), cancelled_by=$3, cancel_reason=$4, updated_at=NOW(),
		        stock_status = CASE WHEN stock_status='CONCLUIDO' THEN 'ESTORNADO' ELSE 'NAO_APLICA' END
		  WHERE id=$1 AND enterprise_id=$2`, c.EntryID, empresa, c.UserID, c.Motivo); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE fiscal_received_documents SET fiscal_entry_id=NULL WHERE enterprise_id=$1 AND fiscal_entry_id=$2`, empresa, c.EntryID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return res, nil
}

func estornarAprovacao(ctx context.Context, tx pgx.Tx, empresa, empresaCode int64, c repository.CancelamentoEntrada, numero int64, res *repository.ResultadoCancelamento) error {
	// Títulos: nenhum pode ter sido pago ou abatido.
	rows, err := tx.Query(ctx,
		`SELECT id, numero_documento, UPPER(status), COALESCE(valor_pago,0), COALESCE(valor_adiantamento_abatido,0)
		   FROM contas_pagar WHERE fiscal_entry_id=$1 AND enterprise_id=$2 AND is_active FOR UPDATE`, c.EntryID, empresa)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		var doc, st string
		var pago, abatido decimal.Decimal
		if err := rows.Scan(&id, &doc, &st, &pago, &abatido); err != nil {
			rows.Close()
			return err
		}
		if st == "CANCELADO" {
			continue
		}
		if pago.IsPositive() || abatido.IsPositive() || st == "PAGO" {
			rows.Close()
			return errorsuc.NewValidationError(fmt.Sprintf(
				"o título %s desta nota já teve pagamento ou adiantamento: estorne o pagamento antes de cancelar a nota", doc))
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) > 0 {
		tag, err := tx.Exec(ctx, `UPDATE contas_pagar SET status='CANCELADO', updated_at=NOW() WHERE id = ANY($1) AND enterprise_id=$2`, ids, empresa)
		if err != nil {
			return err
		}
		res.TitulosCancelados = int(tag.RowsAffected())
	}

	// Estoque e pedido de compra.
	type itemEst struct {
		id, itemCode       int64
		movID, linha       *int64
		qtd, recebidaAntes decimal.Decimal
		movimenta          bool
	}
	irows, err := tx.Query(ctx,
		`SELECT id, COALESCE(item_code,0), stock_movement_id, purchase_order_item_code,
		        COALESCE(quantidade_estoque, quantity), qtd_recebida_antes, movimenta_estoque
		   FROM fiscal_entry_items WHERE fiscal_entry_id=$1 ORDER BY id`, c.EntryID)
	if err != nil {
		return err
	}
	var itens []itemEst
	for irows.Next() {
		var it itemEst
		if err := irows.Scan(&it.id, &it.itemCode, &it.movID, &it.linha, &it.qtd, &it.recebidaAntes, &it.movimenta); err != nil {
			irows.Close()
			return err
		}
		itens = append(itens, it)
	}
	irows.Close()

	ref := stockentity.ReferenceTypeNFEntry
	for _, it := range itens {
		if it.movID != nil {
			var wh int64
			var qtd decimal.Decimal
			if err := tx.QueryRow(ctx, `SELECT warehouse_id, quantity FROM stock_movements WHERE id=$1 AND enterprise_id=$2`, *it.movID, empresa).Scan(&wh, &qtd); err != nil {
				return fmt.Errorf("lendo movimento de estoque da nota: %w", err)
			}
			var saldo decimal.Decimal
			if err := tx.QueryRow(ctx,
				`SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3 AND mask=''`,
				empresa, it.itemCode, wh).Scan(&saldo); err != nil {
				return err
			}
			if saldo.LessThan(qtd) {
				return errorsuc.NewValidationError(fmt.Sprintf(
					"o item %d tem só %s no almoxarifado da entrada e a nota deu entrada de %s: o material já foi consumido ou transferido — faça a devolução ao fornecedor em vez de cancelar a nota",
					it.itemCode, saldo.String(), qtd.String()))
			}
			nota := fmt.Sprintf("Estorno da NF %d (cancelamento da nota de entrada %d): %s", numero, c.EntryID, c.Motivo)
			mov := &stockentity.StockMovement{
				ItemCode: it.itemCode, WarehouseID: wh, MovementType: stockentity.MovementTypeOut,
				Quantity: qtd.InexactFloat64(), ExactQuantity: qtd, ReferenceType: &ref, ReferenceCode: &c.EntryID,
				Notes: &nota, CreatedBy: c.UserID,
			}
			if err := stockrepo.CreateMovementTx(ctx, tx, empresa, mov); err != nil {
				return fmt.Errorf("estornando estoque do item %d: %w", it.itemCode, err)
			}
			res.MovimentosEstorno++
		}
		if it.linha != nil {
			l, err := lockLinhaPedido(ctx, tx, empresaCode, *it.linha)
			if err != nil {
				return err
			}
			fator := decimal.NewFromInt(1)
			if l.interno.IsPositive() && l.requested.IsPositive() {
				fator = l.interno.Div(l.requested)
			}
			qtdPedido := it.qtd.Div(fator)
			l.invoiced = l.invoiced.Sub(qtdPedido)
			l.received = l.received.Sub(it.qtd.Sub(it.recebidaAntes).Div(fator))
			if err := gravarLinhaPedido(ctx, tx, empresaCode, l); err != nil {
				return err
			}
		}
	}

	// Contabilidade: lançamento inverso de cada partida da nota.
	tag, err := tx.Exec(ctx,
		`INSERT INTO accounting_journal_entries
			(plan_id, empresa_id, entry_date, entry_number, batch_number, debit_account_id, credit_account_id,
			 debit_cc_id, credit_cc_id, value, history_code, description, entry_type, source_type, source_id)
		 SELECT plan_id, empresa_id, CURRENT_DATE, 'E' || entry_number, batch_number, credit_account_id, debit_account_id,
		        credit_cc_id, debit_cc_id, value, history_code, 'Estorno: ' || description, 'NFE_ENTRADA_ESTORNO', 'NFE_ENTRADA_ESTORNO', source_id
		   FROM accounting_journal_entries
		  WHERE empresa_id=$1 AND source_type='NFE_ENTRADA' AND source_id=$2`, empresa, c.EntryID)
	if err != nil {
		return fmt.Errorf("estornando contabilização: %w", err)
	}
	res.LancamentosEstorno = int(tag.RowsAffected())
	return nil
}

// ---- apoio ----

func (r *FiscalRepositoryPG) ExistingWarehouses(ctx context.Context, ids []int64) (map[int64]string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	out := map[int64]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT id, code || ' — ' || description FROM warehouse WHERE enterprise_id=$1 AND id = ANY($2)`, empresa, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var n string
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

// EntryOperations lê os tipos de operação de entrada. O cadastro é comum às
// empresas (é a natureza fiscal da operação, não um dado da empresa).
func (r *FiscalRepositoryPG) EntryOperations(ctx context.Context, codes []int64) (map[int64]repository.EntryOperation, error) {
	out := map[int64]repository.EntryOperation{}
	if len(codes) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT code, description, nature_operation, movimenta_estoque, gera_financeiro, credita_icms, credita_ipi, credita_pis_cofins, is_active
		   FROM entry_operation_types WHERE code = ANY($1)`, codes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var o repository.EntryOperation
		if err := rows.Scan(&o.Code, &o.Description, &o.Natureza, &o.MovimentaEstoque, &o.GeraFinanceiro, &o.CreditaICMS, &o.CreditaIPI, &o.CreditaPISCOFINS, &o.IsActive); err != nil {
			return nil, err
		}
		out[o.Code] = o
	}
	return out, rows.Err()
}

const linhaPedidoCols = `poi.code, po.code, po.order_number, poi.sequence, poi.item_code, poi.status,
		        poi.requested_qty, poi.received_qty, poi.invoiced_qty, poi.cancelled_qty,
		        COALESCE(poi.internal_qty,0), poi.unit_price, COALESCE(poi.internal_price,0), COALESCE(poi.tolerance_pct,0),
		        poi.warehouse_id, poi.accounting_account, poi.operation_type_code, po.status`

func (r *FiscalRepositoryPG) OpenPurchaseOrderLines(ctx context.Context, supplierCode, itemCode int64) ([]repository.PurchaseOrderLine, error) {
	empresaCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+linhaPedidoCols+`
		   FROM purchase_order_items poi
		   JOIN purchase_orders po ON po.code = poi.purchase_order_code
		  WHERE po.enterprise_code = $1 AND po.supplier_code = $2 AND poi.item_code = $3 AND poi.is_active
		    AND poi.status <> 'CANCELLED' AND po.status IN ('APPROVED','PARTIAL')
		    AND poi.requested_qty - poi.invoiced_qty - poi.cancelled_qty > 0
		  ORDER BY po.emission_date, po.code, poi.sequence LIMIT 100`, empresaCode, supplierCode, itemCode)
	if err != nil {
		return nil, fmt.Errorf("lendo linhas em aberto do pedido de compra: %w", err)
	}
	defer rows.Close()
	return scanLinhasPedido(rows)
}

func scanLinhasPedido(rows pgx.Rows) ([]repository.PurchaseOrderLine, error) {
	var out []repository.PurchaseOrderLine
	for rows.Next() {
		var l repository.PurchaseOrderLine
		if err := rows.Scan(&l.Code, &l.PurchaseOrderCode, &l.OrderNumber, &l.Sequence, &l.ItemCode, &l.Status,
			&l.RequestedQty, &l.ReceivedQty, &l.InvoicedQty, &l.CancelledQty,
			&l.InternalQty, &l.UnitPrice, &l.InternalPrice, &l.TolerancePct,
			&l.WarehouseID, &l.AccountingAccount, &l.OperationCode, &l.OrderStatus); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *FiscalRepositoryPG) PurchaseOrderLines(ctx context.Context, supplierCode int64, refs []string, lineCodes []int64) ([]repository.PurchaseOrderLine, error) {
	empresaCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	numeros := make([]int64, 0, len(refs))
	for _, ref := range refs {
		var n int64
		if _, err := fmt.Sscan(strings.TrimLeft(soDigitosRepo(ref), "0"), &n); err == nil && n > 0 {
			numeros = append(numeros, n)
		}
	}
	if len(numeros) == 0 && len(lineCodes) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT poi.code, po.code, po.order_number, poi.sequence, poi.item_code, poi.status,
		        poi.requested_qty, poi.received_qty, poi.invoiced_qty, poi.cancelled_qty,
		        COALESCE(poi.internal_qty,0), poi.unit_price, COALESCE(poi.internal_price,0), COALESCE(poi.tolerance_pct,0),
		        poi.warehouse_id, poi.accounting_account, poi.operation_type_code, po.status
		   FROM purchase_order_items poi
		   JOIN purchase_orders po ON po.code = poi.purchase_order_code
		  WHERE po.enterprise_code = $1 AND po.supplier_code = $2 AND poi.is_active
		    AND (po.order_number = ANY($3) OR po.code = ANY($3) OR poi.code = ANY($4))
		  ORDER BY po.code, poi.sequence`, empresaCode, supplierCode, numeros, lineCodes)
	if err != nil {
		return nil, fmt.Errorf("lendo linhas do pedido de compra: %w", err)
	}
	defer rows.Close()
	var out []repository.PurchaseOrderLine
	for rows.Next() {
		var l repository.PurchaseOrderLine
		if err := rows.Scan(&l.Code, &l.PurchaseOrderCode, &l.OrderNumber, &l.Sequence, &l.ItemCode, &l.Status,
			&l.RequestedQty, &l.ReceivedQty, &l.InvoicedQty, &l.CancelledQty,
			&l.InternalQty, &l.UnitPrice, &l.InternalPrice, &l.TolerancePct,
			&l.WarehouseID, &l.AccountingAccount, &l.OperationCode, &l.OrderStatus); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *FiscalRepositoryPG) NCMIPIRates(ctx context.Context, ncms []string) (map[string]decimal.Decimal, error) {
	out := map[string]decimal.Decimal{}
	if len(ncms) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT regexp_replace(ncm,'\D','','g'), aliq_ipi FROM ncm_tax_table WHERE is_active AND regexp_replace(ncm,'\D','','g') = ANY($1)`, ncms)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var n string
		var a decimal.Decimal
		if err := rows.Scan(&n, &a); err != nil {
			return nil, err
		}
		out[n] = a
	}
	return out, rows.Err()
}

// colunasParametrosContabeis: uma lista só para o SELECT, o INSERT e o UPSERT.
const colunasParametrosContabeis = `plan_id, fornecedores_account_id, icms_recuperar_account_id, ipi_recuperar_account_id, pis_recuperar_account_id, cofins_recuperar_account_id, ibs_recuperar_account_id, cbs_recuperar_account_id, irrf_recolher_account_id, pcc_recolher_account_id, inss_recolher_account_id, iss_recolher_account_id, despesa_padrao_account_id, contabilizar_entrada, contabilizar_pagamentos, contabilizar_recebimentos, contabilizar_saidas, banco_padrao_account_id, juros_pagos_account_id, descontos_obtidos_account_id, clientes_account_id, juros_recebidos_account_id, descontos_concedidos_account_id, receita_vendas_account_id, icms_vendas_account_id, icms_recolher_account_id, icms_st_recolher_account_id, ipi_recolher_account_id, pis_vendas_account_id, pis_recolher_account_id, cofins_vendas_account_id, cofins_recolher_account_id, cmv_account_id, estoque_account_id`

func (r *FiscalRepositoryPG) AccountingParams(ctx context.Context) (*repository.AccountingPostingParams, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var p repository.AccountingPostingParams
	err = r.pool.QueryRow(ctx, `SELECT `+colunasParametrosContabeis+` FROM accounting_posting_params WHERE enterprise_id=$1`, empresa).Scan(
		&p.PlanID, &p.FornecedoresAccountID, &p.ICMSRecuperarAccountID, &p.IPIRecuperarAccountID, &p.PISRecuperarAccountID, &p.COFINSRecuperarAccountID, &p.IBSRecuperarAccountID, &p.CBSRecuperarAccountID, &p.IRRFRecolherAccountID, &p.PCCRecolherAccountID, &p.INSSRecolherAccountID, &p.ISSRecolherAccountID, &p.DespesaPadraoAccountID, &p.ContabilizarEntrada, &p.ContabilizarPagamentos, &p.ContabilizarRecebimentos, &p.ContabilizarSaidas, &p.BancoPadraoAccountID, &p.JurosPagosAccountID, &p.DescontosObtidosAccountID, &p.ClientesAccountID, &p.JurosRecebidosAccountID, &p.DescontosConcedidosAccountID, &p.ReceitaVendasAccountID, &p.ICMSVendasAccountID, &p.ICMSRecolherAccountID, &p.ICMSSTRecolherAccountID, &p.IPIRecolherAccountID, &p.PISVendasAccountID, &p.PISRecolherAccountID, &p.COFINSVendasAccountID, &p.COFINSRecolherAccountID, &p.CMVAccountID, &p.EstoqueAccountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *FiscalRepositoryPG) SaveAccountingParams(ctx context.Context, p *repository.AccountingPostingParams) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO accounting_posting_params (enterprise_id, `+colunasParametrosContabeis+`, updated_by, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,NOW())
		 ON CONFLICT (enterprise_id) DO UPDATE SET fornecedores_account_id=EXCLUDED.fornecedores_account_id, icms_recuperar_account_id=EXCLUDED.icms_recuperar_account_id, ipi_recuperar_account_id=EXCLUDED.ipi_recuperar_account_id, pis_recuperar_account_id=EXCLUDED.pis_recuperar_account_id, cofins_recuperar_account_id=EXCLUDED.cofins_recuperar_account_id, ibs_recuperar_account_id=EXCLUDED.ibs_recuperar_account_id, cbs_recuperar_account_id=EXCLUDED.cbs_recuperar_account_id, irrf_recolher_account_id=EXCLUDED.irrf_recolher_account_id, pcc_recolher_account_id=EXCLUDED.pcc_recolher_account_id, inss_recolher_account_id=EXCLUDED.inss_recolher_account_id, iss_recolher_account_id=EXCLUDED.iss_recolher_account_id, despesa_padrao_account_id=EXCLUDED.despesa_padrao_account_id, contabilizar_entrada=EXCLUDED.contabilizar_entrada, contabilizar_pagamentos=EXCLUDED.contabilizar_pagamentos, contabilizar_recebimentos=EXCLUDED.contabilizar_recebimentos, contabilizar_saidas=EXCLUDED.contabilizar_saidas, banco_padrao_account_id=EXCLUDED.banco_padrao_account_id, juros_pagos_account_id=EXCLUDED.juros_pagos_account_id, descontos_obtidos_account_id=EXCLUDED.descontos_obtidos_account_id, clientes_account_id=EXCLUDED.clientes_account_id, juros_recebidos_account_id=EXCLUDED.juros_recebidos_account_id, descontos_concedidos_account_id=EXCLUDED.descontos_concedidos_account_id, receita_vendas_account_id=EXCLUDED.receita_vendas_account_id, icms_vendas_account_id=EXCLUDED.icms_vendas_account_id, icms_recolher_account_id=EXCLUDED.icms_recolher_account_id, icms_st_recolher_account_id=EXCLUDED.icms_st_recolher_account_id, ipi_recolher_account_id=EXCLUDED.ipi_recolher_account_id, pis_vendas_account_id=EXCLUDED.pis_vendas_account_id, pis_recolher_account_id=EXCLUDED.pis_recolher_account_id, cofins_vendas_account_id=EXCLUDED.cofins_vendas_account_id, cofins_recolher_account_id=EXCLUDED.cofins_recolher_account_id, cmv_account_id=EXCLUDED.cmv_account_id, estoque_account_id=EXCLUDED.estoque_account_id,
			updated_by=EXCLUDED.updated_by, updated_at=NOW()`,
		empresa, p.PlanID, p.FornecedoresAccountID, p.ICMSRecuperarAccountID, p.IPIRecuperarAccountID, p.PISRecuperarAccountID, p.COFINSRecuperarAccountID, p.IBSRecuperarAccountID, p.CBSRecuperarAccountID, p.IRRFRecolherAccountID, p.PCCRecolherAccountID, p.INSSRecolherAccountID, p.ISSRecolherAccountID, p.DespesaPadraoAccountID, p.ContabilizarEntrada, p.ContabilizarPagamentos, p.ContabilizarRecebimentos, p.ContabilizarSaidas, p.BancoPadraoAccountID, p.JurosPagosAccountID, p.DescontosObtidosAccountID, p.ClientesAccountID, p.JurosRecebidosAccountID, p.DescontosConcedidosAccountID, p.ReceitaVendasAccountID, p.ICMSVendasAccountID, p.ICMSRecolherAccountID, p.ICMSSTRecolherAccountID, p.IPIRecolherAccountID, p.PISVendasAccountID, p.PISRecolherAccountID, p.COFINSVendasAccountID, p.COFINSRecolherAccountID, p.CMVAccountID, p.EstoqueAccountID, p.UpdatedBy)
	if err != nil {
		return fmt.Errorf("gravando parâmetros contábeis: %w", err)
	}
	return nil
}

func (r *FiscalRepositoryPG) PlanoContasAccounts(ctx context.Context, planoIDs []int64) (map[int64]int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	out := map[int64]int64{}
	if len(planoIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, accounting_account_id FROM plano_contas WHERE enterprise_id=$1 AND id = ANY($2) AND accounting_account_id IS NOT NULL`, empresa, planoIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, acc int64
		if err := rows.Scan(&id, &acc); err != nil {
			return nil, err
		}
		out[id] = acc
	}
	return out, rows.Err()
}

// ValidAccountingAccounts confere que as contas são analíticas do plano
// contábil informado (lançamento em conta sintética desequilibra o balancete).
func (r *FiscalRepositoryPG) ValidAccountingAccounts(ctx context.Context, planID int64, ids []int64) (map[int64]bool, error) {
	out := map[int64]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT id FROM accounting_accounts WHERE plan_id=$1 AND is_analytic AND id = ANY($2)`, planID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

func (r *FiscalRepositoryPG) VincularPlanoContaContabil(ctx context.Context, planoID int64, contaID *int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE plano_contas SET accounting_account_id=$3 WHERE id=$1 AND enterprise_id=$2`, planoID, empresa, contaID)
	if err != nil {
		return fmt.Errorf("vinculando plano de contas à conta contábil: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("plano de contas %d não encontrado nesta empresa", planoID))
	}
	return nil
}

func (r *FiscalRepositoryPG) AccountingPlanExists(ctx context.Context, planID int64) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM accounting_plans WHERE id=$1)`, planID).Scan(&ok)
	return ok, err
}

func (r *FiscalRepositoryPG) ContaContabilDoBanco(ctx context.Context, contaBancariaID int64) (*int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var conta *int64
	err = r.pool.QueryRow(ctx, `SELECT accounting_account_id FROM contas_bancarias WHERE id=$1 AND enterprise_id=$2`, contaBancariaID, empresa).Scan(&conta)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("conta bancária %d não encontrada", contaBancariaID))
	}
	return conta, err
}

func (r *FiscalRepositoryPG) VincularContaBancariaContabil(ctx context.Context, contaBancariaID int64, contaID *int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE contas_bancarias SET accounting_account_id=$3, updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, contaBancariaID, empresa, contaID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("conta bancária %d não encontrada", contaBancariaID))
	}
	return nil
}

func (r *FiscalRepositoryPG) Contabilizado(ctx context.Context, origem string, id int64) (bool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	return journal.Existe(ctx, r.pool, empresa, origem, id)
}

func (r *FiscalRepositoryPG) RetencaoTipo(ctx context.Context, contaPagarID int64) (string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return "", err
	}
	var tipo *string
	err = r.pool.QueryRow(ctx, `SELECT retencao_tipo FROM contas_pagar WHERE id=$1 AND enterprise_id=$2`, contaPagarID, empresa).Scan(&tipo)
	if err != nil || tipo == nil {
		return "", err
	}
	return *tipo, nil
}
