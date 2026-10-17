package stock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// sinalSQL é o sentido do movimento no saldo — o mesmo de entity.SignedQuantity.
const sinalSQL = `(CASE WHEN m.movement_type IN ('IN','TRANSFER_IN','EP','EPP','EPE','ENTRADA') THEN 1
	WHEN m.movement_type IN ('OUT','TRANSFER_OUT','REP','SAIDA') THEN -1
	WHEN m.movement_type = 'ADJUSTMENT' THEN SIGN(m.quantity) ELSE 0 END)`

// qtdSQL e valorSQL: quantidade e valor com sinal. O ajuste de custo não mexe na
// quantidade e guarda o valor já com sinal.
const qtdSQL = `(` + sinalSQL + ` * ABS(m.quantity))`
const valorSQL = `(CASE WHEN m.movement_type = 'AJUSTE_CUSTO' THEN m.total_price ELSE ` + sinalSQL + ` * ABS(m.total_price) END)`

// almoxPrometivelSQL: almoxarifado cujo saldo conta para promessa (ATP).
const almoxPrometivelSQL = `(w.disposition AND w.location::text NOT IN ('INSPECAO','REJEICAO'))`

// ---------- Transferência entre almoxarifados ----------

// TransferirEntreAlmoxarifados lança a saída da origem (ao custo médio) e a
// entrada no destino pelo mesmo custo, na mesma transação. As duas pernas se
// referenciam pelo id da saída (reference_type TRANSFERENCIA).
func (r *StockRepositorySQLC) TransferirEntreAlmoxarifados(ctx context.Context, t entity.TransferenciaEstoque) (*entity.ResultadoTransferencia, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if t.DeAlmox == t.ParaAlmox {
		return nil, errorsuc.NewValidationError("origem e destino são o mesmo almoxarifado: para mudar de endereço use a transferência entre endereços")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transfer tx: %w", err)
	}
	defer tx.Rollback(ctx)
	var n int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM warehouse WHERE enterprise_id=$1 AND id = ANY($2)`, empresa, []int64{t.DeAlmox, t.ParaAlmox}).Scan(&n); err != nil {
		return nil, fmt.Errorf("checking transfer warehouses: %w", err)
	}
	if n != 2 {
		return nil, errorsuc.NewValidationError("almoxarifado de origem ou de destino não existe nesta empresa")
	}
	qtd, _ := t.Quantidade.Float64()
	ref := entity.ReferenceTypeTransfer
	saida := &entity.StockMovement{
		ItemCode: t.ItemCode, Mask: t.Mask, WarehouseID: t.DeAlmox, MovementType: entity.MovementTypeTransferOut,
		Quantity: qtd, ExactQuantity: t.Quantidade, Lot: t.Lote, Address: t.EnderecoDe,
		ReferenceType: &ref, Notes: t.Notas, CreatedBy: t.Ator,
	}
	if err := CreateMovementTx(ctx, tx, empresa, saida); err != nil {
		return nil, err
	}
	codigo := saida.ID
	if _, err := tx.Exec(ctx, `UPDATE stock_movements SET reference_code=$1 WHERE id=$1`, codigo); err != nil {
		return nil, fmt.Errorf("linking transfer legs: %w", err)
	}
	saida.ReferenceCode = &codigo
	entrada := &entity.StockMovement{
		ItemCode: t.ItemCode, Mask: t.Mask, WarehouseID: t.ParaAlmox, MovementType: entity.MovementTypeTransferIn,
		Quantity: qtd, ExactQuantity: t.Quantidade, UnitPrice: saida.UnitPrice, TotalPrice: saida.TotalPrice,
		Lot: t.Lote, Address: t.EnderecoPara, ReferenceType: &ref, ReferenceCode: &codigo, Notes: t.Notas, CreatedBy: t.Ator,
	}
	if err := CreateMovementTx(ctx, tx, empresa, entrada); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transfer: %w", err)
	}
	return &entity.ResultadoTransferencia{Saida: saida, Entrada: entrada}, nil
}

// ---------- Estorno ----------

type movimentoOriginal struct {
	id, item, almox     int64
	mask, tipo          string
	qtd                 decimal.Decimal
	preco, total        float64
	refTipo             *string
	refCodigo           *int64
	lote, endereco      *string
	tipoCodigo          *string
	centro              *int32
	estorna, estornadoP *int64
}

var tipoOposto = map[string]string{
	entity.MovementTypeIn: entity.MovementTypeOut, entity.MovementTypeOut: entity.MovementTypeIn,
	entity.MovementTypeTransferIn: entity.MovementTypeTransferOut, entity.MovementTypeTransferOut: entity.MovementTypeTransferIn,
	"ENTRADA": "SAIDA", "SAIDA": "ENTRADA", entity.MovementTypeAdjustment: entity.MovementTypeAdjustment,
}

// EstornarMovimento lança o movimento contrário, pelo mesmo valor, e liga os
// dois. Só movimento manual ou transferência (as duas pernas juntas): o que
// nasceu de um documento — nota, apontamento, inventário — é desfeito pelo
// próprio documento, senão o estoque e o documento passam a discordar.
func (r *StockRepositorySQLC) EstornarMovimento(ctx context.Context, id int64, ator uuid.UUID, motivo string) ([]*entity.StockMovement, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return nil, errorsuc.NewValidationError("informe o motivo do estorno")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning reversal tx: %w", err)
	}
	defer tx.Rollback(ctx)

	orig, err := lerMovimentoParaEstorno(ctx, tx, empresa, id)
	if err != nil {
		return nil, err
	}
	pernas := []*movimentoOriginal{orig}
	ref := ""
	if orig.refTipo != nil {
		ref = *orig.refTipo
	}
	switch ref {
	case "", entity.ReferenceTypeManual:
	case entity.ReferenceTypeTransfer:
		if orig.refCodigo == nil {
			return nil, errorsuc.NewValidationError("transferência sem a outra perna: estorne lançando a transferência de volta")
		}
		var outro int64
		err := tx.QueryRow(ctx, `SELECT id FROM stock_movements WHERE enterprise_id=$1 AND reference_type='TRANSFERENCIA' AND reference_code=$2 AND id<>$3`,
			empresa, *orig.refCodigo, orig.id).Scan(&outro)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("finding the other transfer leg: %w", err)
		}
		if err == nil {
			o, err := lerMovimentoParaEstorno(ctx, tx, empresa, outro)
			if err != nil {
				return nil, err
			}
			pernas = append(pernas, o)
		}
		// A entrada no destino é desfeita primeiro: é ela que pode já ter sido consumida.
		sort.Slice(pernas, func(i, j int) bool { return pernas[i].tipo == entity.MovementTypeTransferIn })
	default:
		codigo := ""
		if orig.refCodigo != nil {
			codigo = fmt.Sprintf(" %d", *orig.refCodigo)
		}
		return nil, errorsuc.NewValidationError(fmt.Sprintf(
			"o movimento foi gerado por %s%s: desfaça pelo próprio documento (cancelar a nota, estornar o apontamento, reabrir o inventário…)", ref, codigo))
	}

	nota := fmt.Sprintf("Estorno do movimento %d: %s", orig.id, motivo)
	refEstorno := entity.ReferenceTypeReversal
	var out []*entity.StockMovement
	for _, p := range pernas {
		oposto, ok := tipoOposto[p.tipo]
		if !ok {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("movimento do tipo %s não é estornável por aqui", p.tipo))
		}
		qtd := p.qtd.Abs()
		if p.tipo == entity.MovementTypeAdjustment {
			qtd = p.qtd.Neg()
		}
		f, _ := qtd.Float64()
		origID := p.id
		rev := &entity.StockMovement{
			ItemCode: p.item, Mask: p.mask, WarehouseID: p.almox, MovementType: oposto,
			Quantity: f, ExactQuantity: qtd, UnitPrice: p.preco, TotalPrice: p.total,
			ReferenceType: &refEstorno, ReferenceCode: &origID, Lot: p.lote, Address: p.endereco,
			Notes: &nota, CreatedBy: ator, MovementTypeCode: p.tipoCodigo, CostCenterCode: p.centro,
			ReversesMovementID: &origID,
		}
		if err := CreateMovementTx(ctx, tx, empresa, rev); err != nil {
			return nil, fmt.Errorf("estornando o movimento %d: %w", p.id, err)
		}
		if _, err := tx.Exec(ctx, `UPDATE stock_movements SET reversed_by_movement_id=$2 WHERE id=$1`, p.id, rev.ID); err != nil {
			return nil, fmt.Errorf("linking the reversal: %w", err)
		}
		out = append(out, rev)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing reversal: %w", err)
	}
	return out, nil
}

func lerMovimentoParaEstorno(ctx context.Context, tx pgx.Tx, empresa, id int64) (*movimentoOriginal, error) {
	var m movimentoOriginal
	err := tx.QueryRow(ctx,
		`SELECT id, item_code, warehouse_id, mask, movement_type, quantity, unit_price, total_price,
		        reference_type, reference_code, lot, address, movement_type_code, cost_center_code,
		        reverses_movement_id, reversed_by_movement_id
		   FROM stock_movements WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, id, empresa).Scan(
		&m.id, &m.item, &m.almox, &m.mask, &m.tipo, &m.qtd, &m.preco, &m.total,
		&m.refTipo, &m.refCodigo, &m.lote, &m.endereco, &m.tipoCodigo, &m.centro, &m.estorna, &m.estornadoP)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("movimento %d não encontrado", id))
	}
	if err != nil {
		return nil, fmt.Errorf("reading movement to reverse: %w", err)
	}
	if m.estornadoP != nil {
		return nil, errorsuc.NewConflictError(fmt.Sprintf("o movimento %d já foi estornado pelo movimento %d", id, *m.estornadoP))
	}
	if m.estorna != nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o movimento %d é o estorno do %d: estorno não se estorna — lance o movimento de novo", id, *m.estorna))
	}
	return &m, nil
}

// ---------- Kardex ----------

// Kardex devolve o extrato do item no período: saldo anterior, cada movimento
// com o saldo acumulado (quantidade e valor) e o saldo final.
func (r *StockRepositorySQLC) Kardex(ctx context.Context, itemCode int64, almox *int64, de, ate time.Time) (*entity.Kardex, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	k := &entity.Kardex{ItemCode: itemCode, Almoxarifado: almox, De: de, Ate: ate}
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(NULLIF(business_code,''), code::text), COALESCE(name,''), COALESCE(warehouse_unit_of_measurement::text,'')
		   FROM items WHERE code=$1 AND enterprise_id=$2`, itemCode, empresa).Scan(&k.Codigo, &k.Nome, &k.Unidade)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("item %d não encontrado nesta empresa", itemCode))
	}
	if err != nil {
		return nil, fmt.Errorf("reading kardex item: %w", err)
	}
	fim := ate.AddDate(0, 0, 1)
	if err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(`+qtdSQL+`),0), COALESCE(SUM(`+valorSQL+`),0)::float8
		   FROM stock_movements m WHERE m.enterprise_id=$1 AND m.item_code=$2 AND m.created_at < $3
		    AND ($4::bigint IS NULL OR m.warehouse_id=$4)`, empresa, itemCode, de, almox).Scan(&k.SaldoAnterior, &k.ValorAnterior); err != nil {
		return nil, fmt.Errorf("reading kardex opening balance: %w", err)
	}
	rows, err := r.pool.Query(ctx,
		`SELECT m.id, m.created_at, m.movement_type, m.movement_type_code, m.reference_type, m.reference_code,
		        m.warehouse_id, m.lot, m.address, `+qtdSQL+`, m.unit_price, `+valorSQL+`::float8,
		        m.cost_center_code, m.reversed_by_movement_id IS NOT NULL, m.reverses_movement_id IS NOT NULL, m.notes
		   FROM stock_movements m WHERE m.enterprise_id=$1 AND m.item_code=$2 AND m.created_at >= $3 AND m.created_at < $4
		    AND ($5::bigint IS NULL OR m.warehouse_id=$5)
		  ORDER BY m.created_at, m.id`, empresa, itemCode, de, fim, almox)
	if err != nil {
		return nil, fmt.Errorf("listing kardex movements: %w", err)
	}
	defer rows.Close()
	saldo, valor := k.SaldoAnterior, k.ValorAnterior
	for rows.Next() {
		var l entity.LinhaKardex
		var q decimal.Decimal
		if err := rows.Scan(&l.MovementID, &l.Data, &l.Tipo, &l.TipoCodigo, &l.Referencia, &l.ReferenciaCodigo,
			&l.Almoxarifado, &l.Lote, &l.Endereco, &q, &l.CustoUnitario, &l.Valor,
			&l.CentroCusto, &l.Estornado, &l.EstornaOutro, &l.Notas); err != nil {
			return nil, fmt.Errorf("scanning kardex line: %w", err)
		}
		if q.IsPositive() {
			l.Entrada = q
			k.Entradas = k.Entradas.Add(q)
		} else {
			l.Saida = q.Neg()
			k.Saidas = k.Saidas.Add(q.Neg())
		}
		saldo = saldo.Add(q)
		valor += l.Valor
		l.Saldo, l.ValorSaldo = saldo, math.Round(valor*100)/100
		k.Linhas = append(k.Linhas, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	k.SaldoFinal, k.ValorFinal = saldo, math.Round(valor*100)/100
	return k, nil
}

// ---------- Posição em data ----------

// PosicaoEmData devolve o saldo de cada item por almoxarifado ao fim do dia
// pedido, somando os movimentos até ele, com a unidade de compra como segunda
// unidade quando há conversão cadastrada.
func (r *StockRepositorySQLC) PosicaoEmData(ctx context.Context, data time.Time, almox *int64) ([]entity.PosicaoItem, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT m.item_code, COALESCE(NULLIF(i.business_code,''), i.code::text), COALESCE(i.name,''),
		        COALESCE(i.warehouse_unit_of_measurement::text,''), COALESCE(i.supplies_purchase_uom,''), m.warehouse_id,
		        SUM(`+qtdSQL+`), SUM(`+valorSQL+`)::float8
		   FROM stock_movements m JOIN items i ON i.code = m.item_code AND i.enterprise_id = m.enterprise_id
		  WHERE m.enterprise_id=$1 AND m.created_at < $2 AND ($3::bigint IS NULL OR m.warehouse_id=$3)
		  GROUP BY 1,2,3,4,5,6
		 HAVING SUM(`+qtdSQL+`) <> 0
		  ORDER BY 2, 6`, empresa, data.AddDate(0, 0, 1), almox)
	if err != nil {
		return nil, fmt.Errorf("listing stock position: %w", err)
	}
	defer rows.Close()
	var out []entity.PosicaoItem
	compra := map[int64]string{}
	for rows.Next() {
		var p entity.PosicaoItem
		var uc string
		if err := rows.Scan(&p.ItemCode, &p.Codigo, &p.Nome, &p.Unidade, &uc, &p.Almoxarifado, &p.Quantidade, &p.Valor); err != nil {
			return nil, fmt.Errorf("scanning stock position: %w", err)
		}
		if q, _ := p.Quantidade.Float64(); q > 0 {
			p.CustoMedio = p.Valor / q
		}
		p.Valor = math.Round(p.Valor*100) / 100
		if uc != "" && !strings.EqualFold(uc, p.Unidade) {
			compra[p.ItemCode] = strings.ToUpper(uc)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	fatores, err := r.fatoresParaUnidade(ctx, empresa, compra)
	if err != nil {
		return nil, err
	}
	for i := range out {
		if f, ok := fatores[out[i].ItemCode]; ok {
			u := compra[out[i].ItemCode]
			q := out[i].Quantidade.Mul(f).Round(4)
			out[i].UnidadeAlternativa, out[i].QuantidadeAlt = &u, &q
		}
	}
	return out, nil
}

// fatoresParaUnidade: fator unidade de estoque → unidade pedida, por item
// (conversão direta ou a inversa da cadastrada).
func (r *StockRepositorySQLC) fatoresParaUnidade(ctx context.Context, empresa int64, alvo map[int64]string) (map[int64]decimal.Decimal, error) {
	out := map[int64]decimal.Decimal{}
	if len(alvo) == 0 {
		return out, nil
	}
	codes := make([]int64, 0, len(alvo))
	for c := range alvo {
		codes = append(codes, c)
	}
	rows, err := r.pool.Query(ctx,
		`SELECT c.item_code, upper(c.from_uom), upper(c.to_uom), c.factor, upper(i.warehouse_unit_of_measurement::text)
		   FROM item_unit_conversions c JOIN items i ON i.code=c.item_code AND i.enterprise_id=$1
		  WHERE c.item_code = ANY($2) AND c.is_active AND c.mask=''`, empresa, codes)
	if err != nil {
		return nil, fmt.Errorf("reading conversions for the second unit: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item int64
		var de, para, estoque string
		var f decimal.Decimal
		if err := rows.Scan(&item, &de, &para, &f, &estoque); err != nil {
			return nil, err
		}
		u := alvo[item]
		switch {
		case de == estoque && para == u:
			out[item] = f
		case de == u && para == estoque && f.IsPositive():
			if _, ja := out[item]; !ja {
				out[item] = decimal.NewFromInt(1).DivRound(f, 10)
			}
		}
	}
	return out, rows.Err()
}

// ---------- ATP com situação do estoque ----------

// SaldosComSituacao devolve o saldo do item por almoxarifado dizendo se conta
// para promessa. Antes o ATP somava inspeção e rejeição: material aguardando
// laudo aparecia como disponível para vender.
func (r *StockRepositorySQLC) SaldosComSituacao(ctx context.Context, itemCode int64) ([]entity.SaldoComSituacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT b.id, b.item_code, b.mask, b.warehouse_id, SUM(b.quantity)::float8, SUM(b.reserved_qty)::float8,
		        MAX(b.avg_cost)::float8, SUM(b.total_cost)::float8,
		        COALESCE(BOOL_OR(`+almoxPrometivelSQL+`), TRUE),
		        COALESCE(MAX(CASE WHEN w.location::text IN ('INSPECAO') THEN 'INSPECAO'
		                          WHEN w.location::text = 'REJEICAO' OR NOT w.disposition THEN 'BLOQUEADO' END), 'LIVRE')
		   FROM stock_balances b LEFT JOIN warehouse w ON w.id=b.warehouse_id AND w.enterprise_id=b.enterprise_id
		  WHERE b.enterprise_id=$1 AND b.item_code=$2
		  GROUP BY b.id, b.item_code, b.mask, b.warehouse_id
		  ORDER BY b.warehouse_id`, empresa, itemCode)
	if err != nil {
		return nil, fmt.Errorf("listing balances with status: %w", err)
	}
	defer rows.Close()
	var out []entity.SaldoComSituacao
	for rows.Next() {
		var s entity.SaldoComSituacao
		if err := rows.Scan(&s.ID, &s.ItemCode, &s.Mask, &s.WarehouseID, &s.Quantity, &s.ReservedQty,
			&s.AvgCost, &s.TotalCost, &s.Prometivel, &s.Situacao); err != nil {
			return nil, err
		}
		s.AvailableQty = s.Quantity - s.ReservedQty
		out = append(out, s)
	}
	return out, rows.Err()
}

// ---------- Reposição (ponto de pedido / mínimo) ----------

type pontoDePedido struct {
	TR int16 `json:"TR"`
	CM int16 `json:"CM"`
	CR int   `json:"CR"`
	ES int16 `json:"ES"`
}

// SugestoesDeReposicao lista os itens com projeção (disponível − reservado +
// em pedido) no ponto de pedido ou abaixo dele, e quanto comprar.
//
// Ponto de pedido: o do cadastro (TR × CM / CR + ES); sem consumo cadastrado,
// o consumo médio apurado; sem nada disso, o estoque mínimo do item.
func (r *StockRepositorySQLC) SugestoesDeReposicao(ctx context.Context) ([]entity.SugestaoReposicao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT i.code, COALESCE(NULLIF(i.business_code,''), i.code::text), COALESCE(i.name,''),
		        COALESCE(i.warehouse_unit_of_measurement::text,''), i.planning_reorder_point,
		        COALESCE(i.warehouse_minimum_stock,0), COALESCE(i.planning_safety_stock,0),
		        COALESCE(i.planning_minimum_lot,0), COALESCE(i.planning_multiple_lot,0),
		        COALESCE((SELECT SUM(b.quantity) FROM stock_balances b JOIN warehouse w ON w.id=b.warehouse_id AND w.enterprise_id=b.enterprise_id
		                   WHERE b.enterprise_id=i.enterprise_id AND b.item_code=i.code AND `+almoxPrometivelSQL+`), 0),
		        COALESCE((SELECT SUM(b.reserved_qty) FROM stock_balances b JOIN warehouse w ON w.id=b.warehouse_id AND w.enterprise_id=b.enterprise_id
		                   WHERE b.enterprise_id=i.enterprise_id AND b.item_code=i.code AND `+almoxPrometivelSQL+`), 0),
		        COALESCE((SELECT SUM(GREATEST(p.requested_qty - p.received_qty - p.cancelled_qty, 0)
		                             * CASE WHEN p.internal_qty > 0 AND p.requested_qty > 0 THEN p.internal_qty / p.requested_qty ELSE 1 END)
		                    FROM purchase_order_items p JOIN purchase_orders o ON o.code = p.purchase_order_code
		                    JOIN enterprise e ON e.code = o.enterprise_code
		                   WHERE e.id = i.enterprise_id AND p.item_code = i.code AND p.is_active AND o.is_active
		                     AND p.status IN ('OPEN','PARTIAL') AND o.status IN ('REQUESTED','APPROVED','PARTIAL')), 0)
		        -- Requisição de compra ainda não atendida também já está pedida.
		        + COALESCE((SELECT SUM(GREATEST(ri.quantity - ri.attended_qty - ri.cancelled_qty, 0))
		                      FROM purchase_requisition_items ri JOIN purchase_requisitions rq ON rq.code = ri.requisition_code
		                      JOIN enterprise e2 ON e2.code = rq.enterprise_code
		                     WHERE e2.id = i.enterprise_id AND ri.item_code = i.code AND ri.is_active AND rq.is_active
		                       AND rq.status IN ('OPEN','PARTIAL')), 0),
		        COALESCE((SELECT a.avg_monthly_consumption FROM item_consumption_averages a
		                   WHERE a.enterprise_id=i.enterprise_id AND a.item_code=i.code ORDER BY a.calculated_at DESC LIMIT 1), 0)
		   FROM items i
		  WHERE i.enterprise_id=$1
		    AND (COALESCE(i.warehouse_minimum_stock,0) > 0 OR (i.planning_reorder_point IS NOT NULL AND i.planning_reorder_point <> 'null'::jsonb))
		  ORDER BY 2`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listing replenishment candidates: %w", err)
	}
	defer rows.Close()
	var out []entity.SugestaoReposicao
	for rows.Next() {
		var s entity.SugestaoReposicao
		var ppRaw []byte
		var minimo, seg, lmin, lmul int64
		if err := rows.Scan(&s.ItemCode, &s.Codigo, &s.Nome, &s.Unidade, &ppRaw, &minimo, &seg, &lmin, &lmul,
			&s.Disponivel, &s.Reservado, &s.EmPedido, &s.ConsumoMedio); err != nil {
			return nil, fmt.Errorf("scanning replenishment candidate: %w", err)
		}
		s.EstoqueSeguranca = decimal.NewFromInt(seg)
		s.LoteMinimo, s.LoteMultiplo = decimal.NewFromInt(lmin), decimal.NewFromInt(lmul)
		s.PontoPedido, s.Motivo = calcularPontoDePedido(ppRaw, minimo, s.ConsumoMedio)
		if !s.PontoPedido.IsPositive() {
			continue
		}
		s.Projetado = s.Disponivel.Sub(s.Reservado).Add(s.EmPedido)
		if s.Projetado.GreaterThan(s.PontoPedido) {
			continue
		}
		s.Sugerido = quantidadeDeReposicao(s.PontoPedido, s.EstoqueSeguranca, s.Projetado, s.LoteMinimo, s.LoteMultiplo)
		out = append(out, s)
	}
	return out, rows.Err()
}

// calcularPontoDePedido: TR × CM / CR + ES do cadastro; CM zerado usa o consumo
// médio apurado (mensal, levado ao ciclo CR em dias); sem fórmula, o mínimo.
func calcularPontoDePedido(raw []byte, minimo int64, consumoMensal decimal.Decimal) (decimal.Decimal, string) {
	var pp pontoDePedido
	if len(raw) > 0 && string(raw) != "null" && json.Unmarshal(raw, &pp) == nil && pp.CR > 0 && pp.TR > 0 {
		cm := decimal.NewFromInt(int64(pp.CM))
		origem := "ponto de pedido do cadastro"
		if !cm.IsPositive() && consumoMensal.IsPositive() {
			cm = consumoMensal.Mul(decimal.NewFromInt(int64(pp.CR))).Div(decimal.NewFromInt(30))
			origem = "ponto de pedido com o consumo médio apurado"
		}
		if cm.IsPositive() {
			v := decimal.NewFromInt(int64(pp.TR)).Mul(cm).Div(decimal.NewFromInt(int64(pp.CR))).Add(decimal.NewFromInt(int64(pp.ES))).Ceil()
			return v, origem
		}
	}
	if minimo > 0 {
		return decimal.NewFromInt(minimo), "estoque mínimo do cadastro"
	}
	return decimal.Zero, ""
}

// quantidadeDeReposicao: o que leva a projeção de volta ao ponto de pedido mais
// o estoque de segurança, no mínimo o lote mínimo, arredondado para cima ao
// múltiplo de compra.
func quantidadeDeReposicao(ponto, seguranca, projetado, loteMin, multiplo decimal.Decimal) decimal.Decimal {
	q := ponto.Add(seguranca).Sub(projetado)
	if !q.IsPositive() {
		q = decimal.NewFromInt(1)
	}
	if loteMin.IsPositive() && q.LessThan(loteMin) {
		q = loteMin
	}
	if multiplo.IsPositive() {
		q = q.Div(multiplo).Ceil().Mul(multiplo)
	}
	return q.Ceil()
}

// ---------- Análise (giro, cobertura, parados) ----------

// AnaliseDeEstoque: por item com saldo ou consumo na janela — valor em estoque,
// consumo (saídas que não são transferência nem estorno), consumo médio,
// cobertura em meses, giro anualizado, dias sem movimento e classe ABC.
func (r *StockRepositorySQLC) AnaliseDeEstoque(ctx context.Context, meses int, diasParado int) ([]entity.AnaliseItem, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if meses <= 0 {
		meses = 6
	}
	if diasParado <= 0 {
		diasParado = 90
	}
	desde := time.Now().AddDate(0, -meses, 0)
	rows, err := r.pool.Query(ctx,
		`WITH saldo AS (
		     SELECT item_code, SUM(quantity) q, SUM(total_cost) v FROM stock_balances WHERE enterprise_id=$1 GROUP BY item_code),
		  consumo AS (
		     SELECT m.item_code, SUM(ABS(m.quantity)) q, SUM(ABS(m.total_price)) v
		       FROM stock_movements m
		      WHERE m.enterprise_id=$1 AND m.created_at >= $2 AND m.movement_type IN ('OUT','SAIDA','REP')
		        AND m.reverses_movement_id IS NULL AND m.reversed_by_movement_id IS NULL
		      GROUP BY m.item_code),
		  ultimo AS (
		     SELECT item_code, MAX(created_at) quando FROM stock_movements WHERE enterprise_id=$1 GROUP BY item_code)
		 SELECT i.code, COALESCE(NULLIF(i.business_code,''), i.code::text), COALESCE(i.name,''),
		        COALESCE(i.warehouse_unit_of_measurement::text,''),
		        COALESCE(s.q,0), COALESCE(s.v,0)::float8, COALESCE(c.q,0), COALESCE(c.v,0)::float8,
		        u.quando, NULLIF(i.planning_abc_class::text,'')
		   FROM items i
		   LEFT JOIN saldo s ON s.item_code=i.code
		   LEFT JOIN consumo c ON c.item_code=i.code
		   LEFT JOIN ultimo u ON u.item_code=i.code
		  WHERE i.enterprise_id=$1 AND (COALESCE(s.q,0) <> 0 OR COALESCE(c.q,0) <> 0)
		  ORDER BY COALESCE(s.v,0) DESC`, empresa, desde)
	if err != nil {
		return nil, fmt.Errorf("listing stock analysis: %w", err)
	}
	defer rows.Close()
	agora := time.Now()
	var out []entity.AnaliseItem
	for rows.Next() {
		var a entity.AnaliseItem
		if err := rows.Scan(&a.ItemCode, &a.Codigo, &a.Nome, &a.Unidade, &a.Saldo, &a.Valor, &a.ConsumoQtd, &a.ConsumoValor,
			&a.UltimoMovimento, &a.ClasseABC); err != nil {
			return nil, fmt.Errorf("scanning stock analysis: %w", err)
		}
		a.Valor = math.Round(a.Valor*100) / 100
		a.ConsumoMedioMensal = a.ConsumoQtd.Div(decimal.NewFromInt(int64(meses))).Round(4)
		if cm, _ := a.ConsumoMedioMensal.Float64(); cm > 0 {
			saldo, _ := a.Saldo.Float64()
			cob := math.Round(saldo/cm*10) / 10
			a.CoberturaMeses = &cob
		}
		if a.Valor > 0 && a.ConsumoValor > 0 {
			giro := math.Round(a.ConsumoValor*12/float64(meses)/a.Valor*100) / 100
			a.Giro = &giro
		}
		if a.UltimoMovimento != nil {
			d := int(agora.Sub(*a.UltimoMovimento).Hours() / 24)
			a.DiasSemMovimento = &d
			a.Parado = d >= diasParado && a.Saldo.IsPositive()
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// LotesAVencer: saldo em lote vencido ou que vence dentro de `dias`.
func (r *StockRepositorySQLC) LotesAVencer(ctx context.Context, dias int) ([]entity.LoteAVencer, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if dias <= 0 {
		dias = 30
	}
	rows, err := r.pool.Query(ctx,
		`SELECT lb.item_code, COALESCE(NULLIF(i.business_code,''), i.code::text), COALESCE(i.name,''), lb.lot, lb.warehouse_id,
		        SUM(lb.quantity), l.expires_at
		   FROM stock_lot_balances lb
		   JOIN stock_lots l ON l.item_code=lb.item_code AND l.lot=lb.lot AND l.enterprise_id=lb.enterprise_id
		   JOIN items i ON i.code=lb.item_code AND i.enterprise_id=lb.enterprise_id
		  WHERE lb.enterprise_id=$1 AND l.expires_at IS NOT NULL AND l.expires_at <= $2
		  GROUP BY 1,2,3,4,5,7 HAVING SUM(lb.quantity) > 0
		  ORDER BY l.expires_at, 2`, empresa, time.Now().AddDate(0, 0, dias))
	if err != nil {
		return nil, fmt.Errorf("listing expiring lots: %w", err)
	}
	defer rows.Close()
	hoje := time.Now().Truncate(24 * time.Hour)
	var out []entity.LoteAVencer
	for rows.Next() {
		var l entity.LoteAVencer
		if err := rows.Scan(&l.ItemCode, &l.Codigo, &l.Nome, &l.Lote, &l.Almoxarifado, &l.Quantidade, &l.Validade); err != nil {
			return nil, err
		}
		l.Dias = int(l.Validade.Sub(hoje).Hours() / 24)
		l.Vencido = l.Dias < 0
		out = append(out, l)
	}
	return out, rows.Err()
}

// TipoDeMovimento lê o tipo de movimento da empresa pela sigla.
func (r *StockRepositorySQLC) TipoDeMovimento(ctx context.Context, sigla string) (*entity.TipoMovimentoConfig, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var t entity.TipoMovimentoConfig
	err = r.pool.QueryRow(ctx,
		`SELECT sigla, description, entry_exit::text, updates_avg_cost, requires_cost_center, allows_manual, is_active
		   FROM stock_movement_types WHERE enterprise_id=$1 AND upper(sigla)=upper($2) LIMIT 1`, empresa, strings.TrimSpace(sigla)).Scan(
		&t.Sigla, &t.Descricao, &t.Direcao, &t.AtualizaCusto, &t.ExigeCentroCusto, &t.PermiteManual, &t.Ativo)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("tipo de movimento %q não cadastrado nesta empresa (VEST0200)", sigla))
	}
	if err != nil {
		return nil, fmt.Errorf("reading movement type: %w", err)
	}
	return &t, nil
}
