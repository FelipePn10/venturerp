package stock

import (
	"context"
	"errors"
	"fmt"
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
const almoxPrometivelSQL = `(w.disposition AND w.location::text NOT IN ('INSPECAO','REJEICAO','TRANSITO'))`

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
	out, err := transferirTx(ctx, tx, empresa, t, t.ReferenciaTipo == nil || *t.ReferenciaTipo == "" || t.ReferenciaCodigo == nil)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transfer: %w", err)
	}
	return out, nil
}

// transferirTx lança as duas pernas na transação de quem chamou. exigirRastreio
// cobra lote e endereço na saída: quem transfere por ordem de inspeção já os fixou.
func transferirTx(ctx context.Context, tx pgx.Tx, empresa int64, t entity.TransferenciaEstoque, exigirRastreio bool) (*entity.ResultadoTransferencia, error) {
	var n int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM warehouse WHERE enterprise_id=$1 AND id = ANY($2)`, empresa, []int64{t.DeAlmox, t.ParaAlmox}).Scan(&n); err != nil {
		return nil, fmt.Errorf("checking transfer warehouses: %w", err)
	}
	if n != 2 {
		return nil, errorsuc.NewValidationError("almoxarifado de origem ou de destino não existe nesta empresa")
	}
	ref := entity.ReferenceTypeTransfer
	documento := t.ReferenciaTipo != nil && *t.ReferenciaTipo != "" && t.ReferenciaCodigo != nil
	if documento {
		ref = *t.ReferenciaTipo
	}
	saida := &entity.StockMovement{
		ItemCode: t.ItemCode, Mask: t.Mask, WarehouseID: t.DeAlmox, MovementType: entity.MovementTypeTransferOut,
		Quantity: t.Quantidade, Lot: t.Lote, Address: t.EnderecoDe,
		ReferenceType: &ref, ReferenceCode: t.ReferenciaCodigo, Notes: t.Notas, CreatedBy: t.Ator,
		ExigirRastreio: exigirRastreio,
	}
	if err := CreateMovementTx(ctx, tx, empresa, saida); err != nil {
		return nil, err
	}
	codigo := saida.ID
	if documento {
		codigo = *t.ReferenciaCodigo
	} else {
		if _, err := tx.Exec(ctx, `UPDATE stock_movements SET reference_code=$1 WHERE id=$1`, codigo); err != nil {
			return nil, fmt.Errorf("linking transfer legs: %w", err)
		}
	}
	saida.ReferenceCode = &codigo
	entrada := &entity.StockMovement{
		ItemCode: t.ItemCode, Mask: t.Mask, WarehouseID: t.ParaAlmox, MovementType: entity.MovementTypeTransferIn,
		Quantity: t.Quantidade, UnitPrice: saida.UnitPrice, TotalPrice: saida.TotalPrice,
		Lot: t.Lote, Address: t.EnderecoPara, ReferenceType: &ref, ReferenceCode: &codigo, Notes: t.Notas, CreatedBy: t.Ator,
	}
	if err := CreateMovementTx(ctx, tx, empresa, entrada); err != nil {
		return nil, err
	}
	return &entity.ResultadoTransferencia{Saida: saida, Entrada: entrada}, nil
}

// ---------- Estorno ----------

type movimentoOriginal struct {
	id, item, almox     int64
	mask, tipo          string
	qtd                 decimal.Decimal
	preco, total        decimal.Decimal
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
	// A saída que atende a demanda independente é lançamento manual com
	// referência: estornada, a demanda volta a ter o saldo.
	case "", entity.ReferenceTypeManual, entity.ReferenceTypeIndependentDemand:
	case entity.ReferenceTypeRequisition:
		if orig.tipo != entity.MovementTypeOut || orig.refCodigo == nil {
			return nil, errorsuc.NewValidationError("a entrega de ordem de reposição se desfaz lançando a transferência de volta")
		}
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
			"o movimento foi gerado por %s%s: desfaça pelo próprio documento (cancelar a nota, estornar o apontamento, reabrir o inventário…)", nomeDoDocumento(ref), codigo))
	}

	nota := fmt.Sprintf("Estorno do movimento %d: %s", orig.id, motivo)
	refEstorno := entity.ReferenceTypeReversal
	var out []*entity.StockMovement
	for _, p := range pernas {
		oposto, ok := tipoOposto[p.tipo]
		if !ok {
			return nil, errorsuc.NewValidationError("este tipo de movimento não pode ser estornado por aqui")
		}
		qtd := p.qtd.Abs()
		if p.tipo == entity.MovementTypeAdjustment {
			qtd = p.qtd.Neg()
		}
		origID := p.id
		rev := &entity.StockMovement{
			ItemCode: p.item, Mask: p.mask, WarehouseID: p.almox, MovementType: oposto,
			Quantity: qtd, UnitPrice: p.preco, TotalPrice: p.total,
			ReferenceType: &refEstorno, ReferenceCode: &origID, Lot: p.lote, Address: p.endereco,
			Notes: &nota, CreatedBy: ator, MovementTypeCode: p.tipoCodigo, CostCenterCode: p.centro,
			ReversesMovementID: &origID, SaidaPeloPreco: true,
		}
		if err := CreateMovementTx(ctx, tx, empresa, rev); err != nil {
			return nil, fmt.Errorf("estornando o movimento %d: %w", p.id, err)
		}
		if _, err := tx.Exec(ctx, `UPDATE stock_movements SET reversed_by_movement_id=$2 WHERE id=$1`, p.id, rev.ID); err != nil {
			return nil, fmt.Errorf("linking the reversal: %w", err)
		}
		if ref == entity.ReferenceTypeRequisition {
			if err := devolverEntregaDaRequisicaoTx(ctx, tx, empresa, *p.refCodigo, p.item, p.mask, p.qtd.Abs()); err != nil {
				return nil, err
			}
		}
		// Estornada a saída para a demanda, a peça volta reservada para ela.
		if ref == entity.ReferenceTypeIndependentDemand && p.refCodigo != nil {
			if _, err := ReservarDeVoltaDoEstornoTx(ctx, tx, empresa, *p.refCodigo, rev, ator); err != nil {
				return nil, err
			}
		}
		out = append(out, rev)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing reversal: %w", err)
	}
	return out, nil
}

// nomeDoDocumento diz em português de onde veio o movimento.
func nomeDoDocumento(ref string) string {
	nomes := map[string]string{
		entity.ReferenceTypePurchaseOrder: "pedido de compra", entity.ReferenceTypeProductionOrder: "ordem de produção",
		entity.ReferenceTypeSalesOrder: "pedido de venda", entity.ReferenceTypeNFEntry: "nota de entrada", entity.ReferenceTypeNFExit: "nota de saída",
		entity.ReferenceTypeInventory: "inventário", entity.ReferenceTypeCycleCount: "contagem cíclica", entity.ReferenceTypeReversal: "estorno",
		entity.ReferenceTypeRequisition: "requisição ao almoxarifado", entity.ReferenceTypeTransit: "envio em trânsito",
		entity.ReferenceTypeLotImplant: "implantação de lote", "ONDA_SEPARACAO": "onda de separação", "QUALITY_LOT": "lote de inspeção",
	}
	if n, ok := nomes[ref]; ok {
		return n
	}
	return "outro documento"
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
// fusoDaEmpresa: o dia do estoque é o do relógio da empresa — o mesmo fuso que
// agenda notificações e o APS. Com o limite calculado em UTC, "até hoje" acabava
// às 21:00 de Brasília e o movimento lançado à noite sumia do kardex e da posição.
func (r *StockRepositorySQLC) fusoDaEmpresa(ctx context.Context, empresa int64) (*time.Location, error) {
	var tz string
	if err := r.pool.QueryRow(ctx,
		`SELECT COALESCE((SELECT NULLIF(timezone,'') FROM enterprise_notification_settings WHERE enterprise_id=$1),'America/Sao_Paulo')`,
		empresa).Scan(&tz); err != nil {
		return nil, fmt.Errorf("lendo o fuso da empresa: %w", err)
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("fuso da empresa %q inválido: %w", tz, err)
	}
	return loc, nil
}

// diaNoFuso: o dia do calendário de t (como foi digitado) à meia-noite do fuso.
func diaNoFuso(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// Kardex do item no período [de, ate], dias do calendário da empresa. Data zero
// vale o padrão: até hoje e desde o primeiro dia do mês corrente.
func (r *StockRepositorySQLC) Kardex(ctx context.Context, itemCode int64, almox *int64, de, ate time.Time) (*entity.Kardex, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	loc, err := r.fusoDaEmpresa(ctx, empresa)
	if err != nil {
		return nil, err
	}
	hoje := time.Now().In(loc)
	if ate.IsZero() {
		ate = hoje
	}
	if de.IsZero() {
		de = time.Date(hoje.Year(), hoje.Month(), 1, 0, 0, 0, 0, loc)
	}
	de, ate = diaNoFuso(de, loc), diaNoFuso(ate, loc)
	if ate.Before(de) {
		return nil, errorsuc.NewValidationError("a data final é anterior à inicial")
	}
	k := &entity.Kardex{ItemCode: itemCode, Almoxarifado: almox, De: de, Ate: ate}
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(NULLIF(business_code,''), code::text), COALESCE(name,''), COALESCE(warehouse_unit_of_measurement::text,'')
		   FROM items WHERE code=$1 AND enterprise_id=$2`, itemCode, empresa).Scan(&k.Codigo, &k.Nome, &k.Unidade)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError("item não encontrado nesta empresa")
	}
	if err != nil {
		return nil, fmt.Errorf("reading kardex item: %w", err)
	}
	fim := ate.AddDate(0, 0, 1)
	if err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(`+qtdSQL+`),0), COALESCE(SUM(`+valorSQL+`),0)
		   FROM stock_movements m WHERE m.enterprise_id=$1 AND m.item_code=$2 AND m.movement_date < $3::date
		    AND ($4::bigint IS NULL OR m.warehouse_id=$4)`, empresa, itemCode, de.Format("2006-01-02"), almox).Scan(&k.SaldoAnterior, &k.ValorAnterior); err != nil {
		return nil, fmt.Errorf("reading kardex opening balance: %w", err)
	}
	rows, err := r.pool.Query(ctx,
		`SELECT m.id, m.created_at, to_char(m.movement_date,'YYYY-MM-DD'), m.movement_type, m.movement_type_code, m.reference_type, m.reference_code,
		        m.warehouse_id, m.lot, m.address, `+qtdSQL+`, m.unit_price, `+valorSQL+`,
		        m.cost_center_code, m.reversed_by_movement_id IS NOT NULL, m.reverses_movement_id IS NOT NULL, m.notes
		   FROM stock_movements m WHERE m.enterprise_id=$1 AND m.item_code=$2 AND m.movement_date >= $3::date AND m.movement_date < $4::date
		    AND ($5::bigint IS NULL OR m.warehouse_id=$5)
		  ORDER BY m.movement_date, m.id`, empresa, itemCode, de.Format("2006-01-02"), fim.Format("2006-01-02"), almox)
	if err != nil {
		return nil, fmt.Errorf("listing kardex movements: %w", err)
	}
	defer rows.Close()
	saldo, valor := k.SaldoAnterior, k.ValorAnterior
	for rows.Next() {
		var l entity.LinhaKardex
		var q decimal.Decimal
		if err := rows.Scan(&l.MovementID, &l.Data, &l.DataDocumento, &l.Tipo, &l.TipoCodigo, &l.Referencia, &l.ReferenciaCodigo,
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
		valor = valor.Add(l.Valor)
		l.Saldo, l.ValorSaldo = saldo, valor.Round(2)
		k.Linhas = append(k.Linhas, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	k.SaldoFinal, k.ValorFinal = saldo, valor.Round(2)
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
	loc, err := r.fusoDaEmpresa(ctx, empresa)
	if err != nil {
		return nil, err
	}
	if data.IsZero() {
		data = time.Now().In(loc)
	}
	data = diaNoFuso(data, loc)
	rows, err := r.pool.Query(ctx,
		`SELECT m.item_code, COALESCE(NULLIF(i.business_code,''), i.code::text), COALESCE(i.name,''),
		        COALESCE(i.warehouse_unit_of_measurement::text,''), COALESCE(i.supplies_purchase_uom,''), m.warehouse_id,
		        SUM(`+qtdSQL+`), SUM(`+valorSQL+`)
		   FROM stock_movements m JOIN items i ON i.code = m.item_code AND i.enterprise_id = m.enterprise_id
		  WHERE m.enterprise_id=$1 AND m.movement_date <= $2::date AND ($3::bigint IS NULL OR m.warehouse_id=$3)
		  GROUP BY 1,2,3,4,5,6
		 HAVING SUM(`+qtdSQL+`) <> 0
		  ORDER BY 2, 6`, empresa, data.Format("2006-01-02"), almox)
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
		if p.Quantidade.IsPositive() {
			p.CustoMedio = p.Valor.DivRound(p.Quantidade, entity.CasasCusto)
		}
		p.Valor = p.Valor.Round(2)
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
		`SELECT b.id, b.item_code, b.mask, b.warehouse_id, SUM(b.quantity), SUM(b.reserved_qty),
		        MAX(b.avg_cost), SUM(b.total_cost),
		        COALESCE((SELECT SUM(x.quantity) FROM stock_lot_balances x WHERE x.enterprise_id=b.enterprise_id AND x.item_code=b.item_code
		                    AND x.mask=b.mask AND x.warehouse_id=b.warehouse_id AND x.address=b.address AND x.quantity > 0
		                    AND `+strings.ReplaceAll(loteForaDoDisponivelSQL, "= b.", "= x.")+`), 0),
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
		var retido decimal.Decimal
		if err := rows.Scan(&s.ID, &s.ItemCode, &s.Mask, &s.WarehouseID, &s.Quantity, &s.ReservedQty,
			&s.AvgCost, &s.TotalCost, &retido, &s.Prometivel, &s.Situacao); err != nil {
			return nil, err
		}
		// Lote vencido ou em situação que não conta fica fora do disponível.
		s.AvailableQty = s.Quantity.Sub(s.ReservedQty).Sub(retido)
		out = append(out, s)
	}
	return out, rows.Err()
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
		      WHERE m.enterprise_id=$1 AND m.movement_date >= $2::date AND `+consumoSQL+`
		      GROUP BY m.item_code),
		  ultimo AS (
		     -- Contar, transferir e estornar não tira o item da prateleira: se
		     -- contassem, o ajuste de inventário escondia o estoque parado.
		     SELECT item_code, MAX(movement_date) quando FROM stock_movements
		      WHERE enterprise_id=$1 AND reverses_movement_id IS NULL AND reversed_by_movement_id IS NULL
		        AND movement_type NOT IN ('TRANSFER_OUT','TRANSFER_IN','TRANSF_ENDERECO','AJUSTE_CUSTO','ADJUSTMENT')
		        AND COALESCE(reference_type,'') NOT IN ('INVENTORY','CONTAGEM_CICLICA','TRANSFERENCIA','TRANSITO','ESTORNO')
		      GROUP BY item_code),
		  demanda AS (
		     SELECT item_code, SUM(q) q FROM (
		       SELECT si.item_code, GREATEST(si.requested_qty - si.attended_qty - si.cancelled_qty, 0) q
		         FROM sales_order_items si JOIN sales_orders so ON so.code = si.sales_order_code
		         JOIN enterprise e ON e.code = so.enterprise_code
		        WHERE e.id=$1 AND si.is_active AND so.is_active AND si.status IN ('OPEN','PARTIAL')
		       UNION ALL
		       SELECT m.item_code, GREATEST(m.quantity - m.attended_quantity, 0)
		         FROM production_order_materials m JOIN production_orders o ON o.id = m.production_order_id AND o.enterprise_id = $1
		        WHERE m.enterprise_id=$1 AND m.material_kind='DEMAND' AND o.is_active AND o.status IN ('OPEN','RELEASED','IN_PROGRESS')
		       UNION ALL
		       SELECT d.item_code, d.quantity FROM independent_demands d
		        WHERE d.enterprise_id=$1 AND d.is_active AND d.encerrada_em IS NULL) x
		      GROUP BY item_code)
		 SELECT i.code, COALESCE(NULLIF(i.business_code,''), i.code::text), COALESCE(i.name,''),
		        COALESCE(i.warehouse_unit_of_measurement::text,''),
		        COALESCE(s.q,0), COALESCE(s.v,0), COALESCE(c.q,0), COALESCE(c.v,0),
		        u.quando, NULLIF(i.planning_abc_class::text,''), COALESCE(d.q,0)
		   FROM items i
		   LEFT JOIN saldo s ON s.item_code=i.code
		   LEFT JOIN consumo c ON c.item_code=i.code
		   LEFT JOIN ultimo u ON u.item_code=i.code
		   LEFT JOIN demanda d ON d.item_code=i.code
		  WHERE i.enterprise_id=$1 AND (COALESCE(s.q,0) <> 0 OR COALESCE(c.q,0) <> 0 OR COALESCE(d.q,0) <> 0)
		  ORDER BY COALESCE(s.v,0) DESC`, empresa, desde)
	if err != nil {
		return nil, fmt.Errorf("listing stock analysis: %w", err)
	}
	defer rows.Close()
	param, err := lerParametrosDeEstoque(ctx, r.pool, empresa)
	if err != nil {
		return nil, err
	}
	agora := time.Now()
	var out []entity.AnaliseItem
	for rows.Next() {
		var a entity.AnaliseItem
		if err := rows.Scan(&a.ItemCode, &a.Codigo, &a.Nome, &a.Unidade, &a.Saldo, &a.Valor, &a.ConsumoQtd, &a.ConsumoValor,
			&a.UltimoMovimento, &a.ClasseABC, &a.DemandaAberta); err != nil {
			return nil, fmt.Errorf("scanning stock analysis: %w", err)
		}
		a.Valor = a.Valor.Round(2)
		a.ConsumoMedioMensal = a.ConsumoQtd.Div(decimal.NewFromInt(int64(meses))).Round(4)
		if a.ConsumoMedioMensal.IsPositive() {
			cob := a.Saldo.Div(a.ConsumoMedioMensal).Round(1).InexactFloat64()
			a.CoberturaMeses = &cob
		}
		if a.Valor.IsPositive() && a.ConsumoValor.IsPositive() {
			giro := a.ConsumoValor.Mul(decimal.NewFromInt(12)).Div(decimal.NewFromInt(int64(meses))).Div(a.Valor).Round(2).InexactFloat64()
			a.Giro = &giro
		}
		if a.UltimoMovimento != nil {
			d := int(agora.Sub(*a.UltimoMovimento).Hours() / 24)
			a.DiasSemMovimento = &d
			a.Parado = d >= diasParado && a.Saldo.IsPositive()
		} else {
			// Saldo que só entrou por ajuste ou transferência nunca girou.
			a.Parado = a.Saldo.IsPositive()
		}
		a.Falta = decimal.Max(a.DemandaAberta.Sub(a.Saldo), decimal.Zero)
		if param.ObsoleteDays != nil && param.ObsoleteProvisionPct != nil && a.Saldo.IsPositive() &&
			(a.DiasSemMovimento == nil || *a.DiasSemMovimento >= *param.ObsoleteDays) {
			a.Provisao = a.Valor.Mul(*param.ObsoleteProvisionPct).Div(decimal.NewFromInt(100)).Round(2)
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
	// O dia é o do relógio da empresa: em UTC, às 22h de Brasília o lote que vence
	// hoje já aparecia vencido.
	loc, err := r.fusoDaEmpresa(ctx, empresa)
	if err != nil {
		return nil, err
	}
	agora := time.Now().In(loc)
	hoje := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, time.UTC)
	rows, err := r.pool.Query(ctx,
		`SELECT lb.item_code, COALESCE(NULLIF(i.business_code,''), i.code::text), COALESCE(i.name,''), lb.lot, lb.warehouse_id,
		        SUM(lb.quantity), l.expires_at
		   FROM stock_lot_balances lb
		   JOIN stock_lots l ON l.item_code=lb.item_code AND l.lot=lb.lot AND l.mask=lb.mask AND l.enterprise_id=lb.enterprise_id
		   JOIN items i ON i.code=lb.item_code AND i.enterprise_id=lb.enterprise_id
		  WHERE lb.enterprise_id=$1 AND l.expires_at IS NOT NULL AND l.expires_at <= $2
		  GROUP BY 1,2,3,4,5,7 HAVING SUM(lb.quantity) > 0
		  ORDER BY l.expires_at, 2`, empresa, hoje.AddDate(0, 0, dias))
	if err != nil {
		return nil, fmt.Errorf("listing expiring lots: %w", err)
	}
	defer rows.Close()
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
