// Package devolucaovenda grava a devolução de venda e a efetiva numa transação
// (estoque, títulos a receber, crédito do cliente, comissão e séries), sempre
// da empresa da sessão.
package devolucaovenda

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/devolucaovenda"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/devolucao"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	serierepository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/serie"
	stockrepository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

func (r *PG) LinhasDaNota(ctx context.Context, fiscalExitID int64) (int64, *int64, string, []dom.LinhaDaNota, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, nil, "", nil, err
	}
	var numero int64
	var cliente *int64
	var chave *string
	var status string
	var tipo int
	err = r.pool.QueryRow(ctx, `SELECT numero_nf, customer_code, chave_acesso, status, tipo_documento FROM fiscal_exits WHERE id=$1 AND enterprise_id=$2`,
		fiscalExitID, empresa).Scan(&numero, &cliente, &chave, &status, &tipo)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil, "", nil, errorsuc.NewNotFoundError(fmt.Sprintf("NF de saída %d", fiscalExitID))
	}
	if err != nil {
		return 0, nil, "", nil, err
	}
	if status != "AUTHORIZED" || tipo != 1 {
		return 0, nil, "", nil, errorsuc.NewValidationError(fmt.Sprintf("a NF %d está %s: só se devolve NF de saída autorizada", numero, status))
	}
	rows, err := r.pool.Query(ctx, `SELECT it.id, it.sequence, it.item_code, COALESCE(it.description,''), COALESCE(it.cfop,''), it.quantity, it.unit_price, it.total_price,
			COALESCE(it.base_icms,0), COALESCE(it.valor_icms,0), COALESCE(it.base_ipi,0), COALESCE(it.valor_ipi,0), COALESCE(it.valor_pis,0), COALESCE(it.valor_cofins,0),
			COALESCE(it.base_icms_st,0), COALESCE(it.valor_icms_st,0),
			COALESCE((SELECT SUM(di.quantidade) FROM devolucoes_venda_itens di JOIN devolucoes_venda d ON d.id=di.devolucao_id
			           WHERE di.fiscal_exit_item_id=it.id AND d.enterprise_id=$2 AND d.situacao <> 'CANCELADA'), 0),
			COALESCE((SELECT p.controla_serie FROM item_pos_venda p WHERE p.enterprise_id=$2 AND p.item_code=it.item_code), FALSE)
		  FROM fiscal_exit_items it WHERE it.fiscal_exit_id=$1 ORDER BY it.sequence, it.id`, fiscalExitID, empresa)
	if err != nil {
		return 0, nil, "", nil, err
	}
	linhas := []dom.LinhaDaNota{}
	for rows.Next() {
		var l dom.LinhaDaNota
		var o devolucao.Origem
		if err := rows.Scan(&l.FiscalExitItemID, &l.Sequencia, &l.ItemCode, &l.Descricao, &l.CFOP, &o.Quantidade, &o.ValorUnitario, &o.Total,
			&o.BaseICMS, &o.ICMS, &o.BaseIPI, &o.IPI, &o.PIS, &o.COFINS, &o.BaseST, &o.ST, &o.Devolvida, &l.ControlaSerie); err != nil {
			rows.Close()
			return 0, nil, "", nil, err
		}
		o.ItemID, o.Descricao = l.FiscalExitItemID, l.Descricao
		l.Quantidade, l.Devolvida, l.ValorUnitario = o.Quantidade, o.Devolvida, o.ValorUnitario
		l.DefinirOrigem(o)
		l.Series = []dom.SerieDaNota{}
		linhas = append(linhas, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, nil, "", nil, err
	}
	// Séries que saíram em cada linha e ainda estão com o cliente.
	srows, err := r.pool.Query(ctx, `SELECT fs.fiscal_exit_item_id, s.id, s.numero_serie FROM fiscal_exit_item_series fs
		  JOIN numeros_serie s ON s.id=fs.serie_id AND s.enterprise_id=fs.enterprise_id
		 WHERE fs.enterprise_id=$1 AND fs.fiscal_exit_id=$2 AND fs.ativo AND s.situacao='NO_CLIENTE' ORDER BY s.numero_serie`, empresa, fiscalExitID)
	if err != nil {
		return 0, nil, "", nil, err
	}
	defer srows.Close()
	idx := map[int64]int{}
	for i, l := range linhas {
		idx[l.FiscalExitItemID] = i
	}
	for srows.Next() {
		var item int64
		var s dom.SerieDaNota
		if err := srows.Scan(&item, &s.ID, &s.Numero); err != nil {
			return 0, nil, "", nil, err
		}
		if i, ok := idx[item]; ok {
			linhas[i].Series = append(linhas[i].Series, s)
		}
	}
	ch := ""
	if chave != nil {
		ch = *chave
	}
	return numero, cliente, ch, linhas, srows.Err()
}

func (r *PG) Criar(ctx context.Context, d *dom.Devolucao, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var ok bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM warehouse WHERE id=$1 AND enterprise_id=$2)`, d.WarehouseID, empresa).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return errorsuc.NewNotFoundError(fmt.Sprintf("almoxarifado %d", d.WarehouseID))
	}
	// Trava a nota: duas devoluções simultâneas não passam do vendido.
	if _, err := tx.Exec(ctx, `SELECT id FROM fiscal_exits WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, d.FiscalExitID, empresa); err != nil {
		return err
	}
	for _, it := range d.Itens {
		var vendido, devolvido decimal.Decimal
		if err := tx.QueryRow(ctx, `SELECT it.quantity, COALESCE((SELECT SUM(di.quantidade) FROM devolucoes_venda_itens di JOIN devolucoes_venda dv ON dv.id=di.devolucao_id
				 WHERE di.fiscal_exit_item_id=it.id AND dv.enterprise_id=$3 AND dv.situacao <> 'CANCELADA'),0)
			  FROM fiscal_exit_items it WHERE it.id=$1 AND it.fiscal_exit_id=$2`, it.FiscalExitItemID, d.FiscalExitID, empresa).Scan(&vendido, &devolvido); err != nil {
			return err
		}
		if devolvido.Add(it.Quantidade).GreaterThan(vendido) {
			return errorsuc.NewValidationError(fmt.Sprintf("a linha %d já tem %s devolvido de %s vendido: não cabe mais %s", it.FiscalExitItemID, devolvido, vendido, it.Quantidade))
		}
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('devolucoes_venda'), $1::int)`, empresa); err != nil {
		return err
	}
	err = tx.QueryRow(ctx, `INSERT INTO devolucoes_venda (enterprise_id, numero, fiscal_exit_id, customer_code, modo, motivo, observacao, warehouse_id,
			nf_cliente_chave, nf_cliente_numero, valor_total, created_by, updated_by)
		VALUES ($1, (SELECT COALESCE(MAX(numero),0)+1 FROM devolucoes_venda WHERE enterprise_id=$1), $2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)
		RETURNING id, numero, created_at`, empresa, d.FiscalExitID, d.CustomerCode, d.Modo, d.Motivo, d.Observacao, d.WarehouseID,
		d.NFClienteChave, d.NFClienteNumero, d.ValorTotal, por).Scan(&d.ID, &d.Numero, &d.CreatedAt)
	if err != nil {
		return fmt.Errorf("gravando a devolução: %w", err)
	}
	for i := range d.Itens {
		it := &d.Itens[i]
		if err := tx.QueryRow(ctx, `INSERT INTO devolucoes_venda_itens (enterprise_id, devolucao_id, fiscal_exit_item_id, item_code, quantidade, valor_produtos,
				valor_icms, valor_icms_st, valor_ipi, valor_pis, valor_cofins, valor_total, series, base_icms, base_ipi, base_icms_st)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) RETURNING id`, empresa, d.ID, it.FiscalExitItemID, it.ItemCode, it.Quantidade, it.ValorProdutos,
			it.ICMS, it.ST, it.IPI, it.PIS, it.COFINS, it.Total, it.Series, it.BaseICMS, it.BaseIPI, it.BaseST).Scan(&it.ID); err != nil {
			return fmt.Errorf("gravando o item da devolução: %w", err)
		}
	}
	return tx.Commit(ctx)
}

const colunas = `d.id, d.numero, d.fiscal_exit_id, fe.numero_nf, COALESCE(fe.chave_acesso,''), d.customer_code, COALESCE(c.name,''), d.modo, d.situacao, d.motivo, d.observacao,
	d.warehouse_id, d.nf_cliente_chave, d.nf_cliente_numero, d.nf_propria_id, np.status, d.valor_total, d.abatido, d.credito_cliente, d.adiantamento_id,
	d.efetivada_em, d.cancelada_em, d.motivo_cancelamento, d.created_at`

const de = ` FROM devolucoes_venda d
	JOIN fiscal_exits fe ON fe.id=d.fiscal_exit_id
	LEFT JOIN fiscal_exits np ON np.id=d.nf_propria_id
	LEFT JOIN customers c ON c.code=d.customer_code AND c.enterprise_id=d.enterprise_id`

func scan(row pgx.Row) (*dom.Devolucao, error) {
	var d dom.Devolucao
	err := row.Scan(&d.ID, &d.Numero, &d.FiscalExitID, &d.NumeroNF, &d.ChaveNF, &d.CustomerCode, &d.ClienteNome, &d.Modo, &d.Situacao, &d.Motivo, &d.Observacao,
		&d.WarehouseID, &d.NFClienteChave, &d.NFClienteNumero, &d.NFPropriaID, &d.NFPropriaStatus, &d.ValorTotal, &d.Abatido, &d.CreditoCliente, &d.AdiantamentoID,
		&d.EfetivadaEm, &d.CanceladaEm, &d.MotivoCancel, &d.CreatedAt)
	return &d, err
}

func itens(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, empresa, id int64) ([]dom.Item, error) {
	rows, err := q.Query(ctx, `SELECT di.id, di.fiscal_exit_item_id, di.item_code, COALESCE(fi.description,''), COALESCE(fi.cfop,''), di.quantidade, di.valor_produtos,
			di.valor_icms, di.valor_icms_st, di.valor_ipi, di.valor_pis, di.valor_cofins, di.valor_total, di.series, di.custo_unitario,
			di.base_icms, di.base_ipi, di.base_icms_st
		  FROM devolucoes_venda_itens di JOIN fiscal_exit_items fi ON fi.id=di.fiscal_exit_item_id
		 WHERE di.enterprise_id=$1 AND di.devolucao_id=$2 ORDER BY di.id`, empresa, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.Item{}
	for rows.Next() {
		var it dom.Item
		var cfopVenda string
		if err := rows.Scan(&it.ID, &it.FiscalExitItemID, &it.ItemCode, &it.Descricao, &cfopVenda, &it.Quantidade, &it.ValorProdutos, &it.ICMS, &it.ST,
			&it.IPI, &it.PIS, &it.COFINS, &it.Total, &it.Series, &it.CustoUnitario, &it.BaseICMS, &it.BaseIPI, &it.BaseST); err != nil {
			return nil, err
		}
		it.CFOP = dom.CFOP(cfopVenda)
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *PG) Obter(ctx context.Context, id int64) (*dom.Devolucao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	d, err := scan(r.pool.QueryRow(ctx, `SELECT `+colunas+de+` WHERE d.enterprise_id=$1 AND d.id=$2`, empresa, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("devolução de venda %d", id))
	}
	if err != nil {
		return nil, err
	}
	if d.Itens, err = itens(ctx, r.pool, empresa, id); err != nil {
		return nil, err
	}
	return d, nil
}

func (r *PG) Listar(ctx context.Context, fiscalExitID *int64, situacao string) ([]dom.Devolucao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunas+de+` WHERE d.enterprise_id=$1 AND ($2::bigint IS NULL OR d.fiscal_exit_id=$2) AND ($3='' OR d.situacao=$3)
		 ORDER BY d.numero DESC LIMIT 500`, empresa, fiscalExitID, situacao)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.Devolucao{}
	for rows.Next() {
		d, err := scan(rows)
		if err != nil {
			return nil, err
		}
		d.Itens = []dom.Item{}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *PG) VincularNFPropria(ctx context.Context, id, exitID int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE devolucoes_venda SET nf_propria_id=$3, updated_at=NOW() WHERE enterprise_id=$1 AND id=$2 AND situacao='RASCUNHO' AND nf_propria_id IS NULL`, empresa, id, exitID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewValidationError("a devolução já tem NF-e de entrada ou não está em rascunho")
	}
	return nil
}

func (r *PG) MarcarEmitida(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `UPDATE devolucoes_venda SET situacao='EMITIDA', updated_at=NOW() WHERE enterprise_id=$1 AND id=$2 AND situacao='RASCUNHO'`, empresa, id)
	return err
}

func (r *PG) Efetivar(ctx context.Context, e dom.Efetivacao) (*dom.Resultado, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	d := e.Devolucao
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var situacao string
	if err := tx.QueryRow(ctx, `SELECT situacao FROM devolucoes_venda WHERE enterprise_id=$1 AND id=$2 FOR UPDATE`, empresa, d.ID).Scan(&situacao); err != nil {
		return nil, err
	}
	res := &dom.Resultado{}
	if situacao == dom.Efetivada {
		return res, nil // idempotente
	}
	por := e.Por
	ref := dom.ReferenciaEstoque
	nota := fmt.Sprintf("devolução de venda %d — NF %d", d.Numero, d.NumeroNF)

	// 1. Estoque: entra pelo custo com que saiu na NF original.
	for i := range d.Itens {
		it := &d.Itens[i]
		if it.ItemCode == nil {
			continue
		}
		var custo decimal.Decimal
		if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(total_price) / NULLIF(SUM(quantity),0), 0) FROM stock_movements
			 WHERE enterprise_id=$1 AND reference_type=$2 AND reference_code=$3 AND item_code=$4 AND movement_type='OUT'`,
			empresa, stockentity.ReferenceTypeNFExit, d.FiscalExitID, *it.ItemCode).Scan(&custo); err != nil {
			return nil, fmt.Errorf("lendo o custo da saída original: %w", err)
		}
		custo = custo.Round(6)
		m := &stockentity.StockMovement{ItemCode: *it.ItemCode, WarehouseID: d.WarehouseID, MovementType: stockentity.MovementTypeIn,
			Quantity: it.Quantidade, UnitPrice: custo, TotalPrice: custo.Mul(it.Quantidade).Round(4),
			ReferenceType: &ref, ReferenceCode: &d.ID, Notes: &nota, CreatedBy: por}
		if err := stockrepository.CreateMovementTx(ctx, tx, empresa, m); err != nil {
			return nil, fmt.Errorf("devolvendo ao estoque o item da linha %d: %w", it.FiscalExitItemID, err)
		}
		if _, err := tx.Exec(ctx, `UPDATE devolucoes_venda_itens SET custo_unitario=$3, stock_movement_id=$4 WHERE enterprise_id=$1 AND id=$2`,
			empresa, it.ID, custo, m.ID); err != nil {
			return nil, err
		}
		res.Entradas++
	}

	// 2. Séries de volta ao estoque.
	for _, it := range d.Itens {
		if len(it.Series) == 0 {
			continue
		}
		if err := serierepository.DevolucaoTx(ctx, tx, empresa, it.Series, d.WarehouseID, d.ID, &por); err != nil {
			return nil, err
		}
		res.Series += len(it.Series)
	}

	// 3. Abatimento dos títulos a receber em aberto da NF original.
	rows, err := tx.Query(ctx, `SELECT id, valor_bruto - COALESCE(desconto,0) - COALESCE(valor_recebido,0)
		  FROM contas_receber WHERE enterprise_id=$1 AND fiscal_exit_id=$2 AND is_active AND status IN ('PENDENTE','APROVADO','VENCIDO')
		 ORDER BY data_vencimento DESC, id DESC FOR UPDATE`, empresa, d.FiscalExitID)
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
	restante := d.ValorTotal.Round(2)
	for _, a := range abertos {
		if !restante.IsPositive() {
			break
		}
		v := decimal.Min(restante, a.saldo)
		quita := v.Equal(a.saldo)
		if _, err := tx.Exec(ctx, `UPDATE contas_receber SET desconto=COALESCE(desconto,0)+$3,
				status=CASE WHEN $4 THEN 'RECEBIDO' ELSE status END,
				data_recebimento=CASE WHEN $4 THEN $5::date ELSE data_recebimento END, updated_at=NOW()
			  WHERE id=$1 AND enterprise_id=$2`, a.id, empresa, v, quita, e.Data); err != nil {
			return nil, fmt.Errorf("abatendo o título %d: %w", a.id, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO fiscal_return_settlements (enterprise_id, devolucao_venda_id, fiscal_exit_id, conta_receber_id, valor, quitou_titulo)
			VALUES ($1,$2,$3,$4,$5,$6)`, empresa, d.ID, d.NFPropriaID, a.id, v, quita); err != nil {
			return nil, err
		}
		restante = restante.Sub(v)
		res.Abatido = res.Abatido.Add(v)
	}

	// 4. O que sobrou (títulos já recebidos) vira crédito do cliente.
	if restante.IsPositive() {
		if d.CustomerCode == nil {
			return nil, errorsuc.NewValidationError("a NF original não tem cliente para receber o crédito da devolução")
		}
		var adiantamento int64
		if err := tx.QueryRow(ctx, `INSERT INTO adiantamentos (tipo, parceiro_id, numero_documento, data_adiantamento, valor_original, valor_utilizado, status, descricao, is_active, created_by, enterprise_id)
			VALUES ('RECEBER',$1,$2,$3::date,$4,0,'ABERTO',$5,TRUE,$6,$7) RETURNING id`, *d.CustomerCode, fmt.Sprintf("DEV-VENDA-%d", d.Numero), e.Data, restante,
			fmt.Sprintf("crédito da devolução de venda %d (NF %d)", d.Numero, d.NumeroNF), por, empresa).Scan(&adiantamento); err != nil {
			return nil, fmt.Errorf("gerando o crédito do cliente: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO fiscal_return_settlements (enterprise_id, devolucao_venda_id, fiscal_exit_id, adiantamento_id, valor)
			VALUES ($1,$2,$3,$4,$5)`, empresa, d.ID, d.NFPropriaID, adiantamento, restante); err != nil {
			return nil, err
		}
		res.CreditoCliente, res.AdiantamentoID = restante, &adiantamento
	}

	// 5. Comissão de faturamento: estorno proporcional ao valor devolvido.
	crow, err := tx.Query(ctx, `SELECT code, base_amount, amount FROM commercial_commission_ledger
		 WHERE enterprise_id=$1 AND fiscal_exit_id=$2 AND event_type='COMPETENCIA_FATURAMENTO' AND status <> 'ESTORNADO'`, empresa, d.FiscalExitID)
	if err != nil {
		return nil, err
	}
	type lanc struct {
		code        int64
		base, valor decimal.Decimal
	}
	var lancs []lanc
	for crow.Next() {
		var l lanc
		if err := crow.Scan(&l.code, &l.base, &l.valor); err != nil {
			crow.Close()
			return nil, err
		}
		lancs = append(lancs, l)
	}
	crow.Close()
	for _, l := range lancs {
		if !l.base.IsPositive() {
			continue
		}
		fator := decimal.Min(decimal.NewFromInt(1), d.ValorTotal.Div(l.base))
		estorno := l.valor.Mul(fator).Round(4).Neg()
		if _, err := tx.Exec(ctx, `INSERT INTO commercial_commission_ledger (enterprise_id, representative_code, sales_order_code, fiscal_exit_id, event_type,
				competence_date, base_amount, commission_pct, amount, status, reversal_of, idempotency_key, actor_id)
			SELECT enterprise_id, representative_code, sales_order_code, fiscal_exit_id, 'ESTORNO_FATURAMENTO', $3::date, -($4::numeric), commission_pct, $5,
				'CONCILIADO', code, 'devolucao_venda:'||$6::bigint::text||':'||code::text, $7
			  FROM commercial_commission_ledger WHERE enterprise_id=$1 AND code=$2
			ON CONFLICT (enterprise_id, idempotency_key) DO NOTHING`, empresa, l.code, e.Data, d.ValorTotal.Round(2), estorno, d.ID, por); err != nil {
			return nil, fmt.Errorf("estornando a comissão: %w", err)
		}
		res.Comissao = res.Comissao.Add(estorno)
	}

	if _, err := tx.Exec(ctx, `UPDATE devolucoes_venda SET situacao='EFETIVADA', abatido=$3, credito_cliente=$4, adiantamento_id=$5, efetivada_em=NOW(),
			updated_by=$6, updated_at=NOW() WHERE enterprise_id=$1 AND id=$2`, empresa, d.ID, res.Abatido, res.CreditoCliente, res.AdiantamentoID, por); err != nil {
		return nil, err
	}
	return res, tx.Commit(ctx)
}

// Cancelar: rascunho (ou emitida com a NF-e própria cancelada) volta sem
// efeito; efetivada só cancela desfazendo pelo estorno — não implementado:
// a devolução efetivada se corrige com uma nova operação.
func (r *PG) Cancelar(ctx context.Context, id int64, motivo string, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var situacao string
	var propria *int64
	err = tx.QueryRow(ctx, `SELECT situacao, nf_propria_id FROM devolucoes_venda WHERE enterprise_id=$1 AND id=$2 FOR UPDATE`, empresa, id).Scan(&situacao, &propria)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("devolução de venda %d", id))
	}
	if err != nil {
		return err
	}
	switch situacao {
	case dom.Cancelada:
		return nil
	case dom.Efetivada:
		return errorsuc.NewValidationError("a devolução já foi efetivada (estoque e financeiro): não se cancela")
	}
	if propria != nil {
		var st string
		if err := tx.QueryRow(ctx, `SELECT status FROM fiscal_exits WHERE id=$1 AND enterprise_id=$2`, *propria, empresa).Scan(&st); err != nil {
			return err
		}
		if st == "AUTHORIZED" {
			return errorsuc.NewValidationError("a NF-e de entrada da devolução está autorizada: cancele-a na SEFAZ antes")
		}
		if st != "CANCELLED" {
			if _, err := tx.Exec(ctx, `UPDATE fiscal_exits SET status='CANCELLED', is_active=FALSE, motivo_cancelamento=$3, updated_at=NOW()
				WHERE id=$1 AND enterprise_id=$2 AND status IN ('DRAFT','REJECTED','PENDENTE')`, *propria, empresa, "devolução cancelada"); err != nil {
				return err
			}
		}
	}
	_, err = tx.Exec(ctx, `UPDATE devolucoes_venda SET situacao='CANCELADA', cancelada_em=NOW(), motivo_cancelamento=$3, updated_by=$4, updated_at=NOW()
		WHERE enterprise_id=$1 AND id=$2`, empresa, id, motivo, por)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
