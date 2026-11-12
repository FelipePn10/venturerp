//go:build integration

package stock_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Regras apuradas na auditoria de estoque de 2026-11: cada teste descreve o
// comportamento certo de uma falha reproduzida antes da correção.

func (e ambienteEstoque) somar(t *testing.T, sql string, args ...any) decimal.Decimal {
	t.Helper()
	var v decimal.Decimal
	if err := testutil.Pool(t).QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func (e ambienteEstoque) saldoEndereco(t *testing.T, item, almox int64, endereco string) decimal.Decimal {
	t.Helper()
	return e.somar(t, `SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3 AND address=$4`, e.empresa, item, almox, endereco)
}

func (e ambienteEstoque) saldoLote(t *testing.T, item, almox int64, lote string) decimal.Decimal {
	t.Helper()
	return e.somar(t, `SELECT COALESCE(SUM(quantity),0) FROM stock_lot_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3 AND lot=$4`, e.empresa, item, almox, lote)
}

func (e ambienteEstoque) moverLote(t *testing.T, item, almox int64, tipo string, qtd, preco float64, endereco, lote string) error {
	t.Helper()
	m := &stockentity.StockMovement{ItemCode: item, WarehouseID: almox, MovementType: tipo, Quantity: decimal.NewFromFloat(qtd), UnitPrice: decimal.NewFromFloat(preco), CreatedBy: e.ator}
	if endereco != "" {
		m.Address = &endereco
	}
	if lote != "" {
		m.Lot = &lote
	}
	_, err := e.repo.CreateMovement(e.ctx, m)
	return err
}

func (e ambienteEstoque) reservar(t *testing.T, item, almox int64, qtd float64) (*stockentity.StockReservation, error) {
	t.Helper()
	return e.repo.CreateReservation(e.ctx, &stockentity.StockReservation{
		ItemCode: item, WarehouseID: almox, Quantity: decimal.NewFromFloat(qtd), ReferenceType: "MANUAL", ReferenceCode: 1,
		ReservationDate: time.Now(), Status: "ACTIVE", CreatedBy: e.ator,
	})
}

func exigir(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func igual(t *testing.T, oque string, obtido decimal.Decimal, esperado string) {
	t.Helper()
	if !obtido.Equal(decimal.RequireFromString(esperado)) {
		t.Errorf("%s: esperado %s, veio %s", oque, esperado, obtido)
	}
}

func TestAuditoria_ReservaComSaldoEmVariosEnderecos(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 5, "A1"))
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 5, "A2"))

	b, err := e.repo.GetBalance(e.ctx, e.item, "", e.almox)
	exigir(t, err)
	igual(t, "saldo do almoxarifado", b.Quantity, "20")

	r, err := e.reservar(t, e.item, e.almox, 15)
	if err != nil {
		t.Fatalf("reserva de 15 com 20 no almoxarifado foi recusada: %v", err)
	}
	reservado := func() decimal.Decimal {
		return e.somar(t, `SELECT COALESCE(SUM(reserved_qty),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item)
	}
	igual(t, "reservado", reservado(), "15")
	exigir(t, e.repo.CancelReservation(e.ctx, r.ID))
	igual(t, "reservado depois de cancelar", reservado(), "0")
}

func TestAuditoria_ReservaEncerradaNaoMudaDeSituacao(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 5, ""))
	r, err := e.reservar(t, e.item, e.almox, 4)
	exigir(t, err)
	exigir(t, e.repo.ConsumeReservation(e.ctx, r.ID))
	if err := e.repo.CancelReservation(e.ctx, r.ID); err == nil {
		t.Error("reserva consumida foi cancelada")
	}
	got, err := e.repo.GetReservation(e.ctx, r.ID)
	exigir(t, err)
	if got.Status != "CONSUMED" {
		t.Errorf("situação mudou para %s", got.Status)
	}
}

func TestAuditoria_AlmoxarifadoSemReservaRecusa(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 5, ""))
	testutil.Exec(t, testutil.Pool(t), `UPDATE warehouse SET reservations_allowed=false WHERE id=$1`, e.almox)
	if _, err := e.reservar(t, e.item, e.almox, 1); !ehValidacao(err) {
		t.Errorf("esperava recusa de validação, veio %v", err)
	}
}

func TestAuditoria_SaidaSemEnderecoBaixaOsEnderecos(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 100, "A1"))
	exigir(t, e.mover(t, e.item, e.almox, "OUT", 3, 0, ""))
	igual(t, "linha sem endereço", e.saldoEndereco(t, e.item, e.almox, ""), "0")
	igual(t, "endereço A1", e.saldoEndereco(t, e.item, e.almox, "A1"), "7")
	exigir(t, e.mover(t, e.item, e.almox, "IN", 7, 100, "A1"))
	_, custo := e.saldo(t, e.item, e.almox)
	if custo != 100 {
		t.Errorf("custo médio deveria seguir 100, veio %v", custo)
	}
}

func TestAuditoria_LoteNaoFicaNegativo(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "", "L1"))
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 5, ""))
	if err := e.moverLote(t, e.item, e.almox, "OUT", 15, 0, "", "L1"); !ehValidacao(err) {
		t.Errorf("saída de 15 num lote de 10: esperava recusa, veio %v", err)
	}
	igual(t, "lote", e.saldoLote(t, e.item, e.almox, "L1"), "10")
}

func TestAuditoria_SaidaSemLoteBaixaOsLotes(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "", "L1"))
	exigir(t, e.mover(t, e.item, e.almox, "OUT", 4, 0, ""))
	igual(t, "lote depois da saída sem lote", e.saldoLote(t, e.item, e.almox, "L1"), "6")
}

func TestAuditoria_EntradaCobrindoSaldoNegativoNaoInflaOCusto(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.mover(t, e.item, e.livre, "OUT", 5, 0, ""))
	exigir(t, e.mover(t, e.item, e.livre, "IN", 10, 100, ""))
	q, custo := e.saldo(t, e.item, e.livre)
	if q != 5 || custo != 100 {
		t.Errorf("esperava 5 a 100, veio %v a %v", q, custo)
	}
}

func TestAuditoria_SaidaSempreSaiPelaMedia(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 100, ""))
	exigir(t, e.mover(t, e.item, e.almox, "OUT", 2, 999, ""))
	igual(t, "valor do movimento de saída",
		e.somar(t, `SELECT total_price FROM stock_movements WHERE enterprise_id=$1 AND item_code=$2 AND movement_type='OUT'`, e.empresa, e.item), "200")
}

func TestAuditoria_AjustePositivoSemPrecoNaoDiluiOCusto(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 100, ""))
	exigir(t, e.mover(t, e.item, e.almox, "ADJUSTMENT", 10, 0, ""))
	if _, custo := e.saldo(t, e.item, e.almox); custo != 100 {
		t.Errorf("custo médio deveria seguir 100, veio %v", custo)
	}
}

func TestAuditoria_MovimentoExigeItemEAlmoxarifadoDaEmpresa(t *testing.T) {
	e := novoAmbienteEstoque(t)
	if err := e.mover(t, 999999999, e.almox, "IN", 1, 1, ""); !ehValidacao(err) {
		t.Errorf("item inexistente: esperava recusa, veio %v", err)
	}
	if err := e.mover(t, e.item, 999999999, "IN", 1, 1, ""); !ehValidacao(err) {
		t.Errorf("almoxarifado inexistente: esperava recusa, veio %v", err)
	}
}

func TestAuditoria_EnderecoBloqueadoOuDesconhecido(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM manufacturing_warehouse_addresses WHERE enterprise_id=$1`, e.empresa)
	})
	testutil.Exec(t, pool, `INSERT INTO manufacturing_warehouse_addresses(enterprise_id,warehouse_id,address,is_blocked) VALUES($1,$2,'OK',false),($1,$2,'TRAVADO',true)`, e.empresa, e.almox)
	exigir(t, e.mover(t, e.item, e.almox, "IN", 1, 1, "OK"))
	if err := e.mover(t, e.item, e.almox, "IN", 1, 1, "TRAVADO"); !ehValidacao(err) {
		t.Errorf("endereço bloqueado: esperava recusa, veio %v", err)
	}
	if err := e.mover(t, e.item, e.almox, "IN", 1, 1, "NAOEXISTE"); !ehValidacao(err) {
		t.Errorf("endereço fora do cadastro: esperava recusa, veio %v", err)
	}
}

func TestAuditoria_TransferenciaDeEnderecoRespeitaReservaLoteEPeriodo(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "A1", "L1"))
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 5, "A1"))
	mov := func(lote *string) *stockentity.StockMovement {
		return &stockentity.StockMovement{ItemCode: e.item, WarehouseID: e.almox, Quantity: decimal.NewFromInt(15), Lot: lote, CreatedBy: e.ator}
	}
	l1 := "L1"
	if _, err := e.repo.TransferirEntreEnderecos(e.ctx, mov(&l1), "A1", "A2"); !ehValidacao(err) {
		t.Errorf("lote de 10 transferindo 15: esperava recusa, veio %v", err)
	}
	if _, err := e.reservar(t, e.item, e.almox, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := e.repo.TransferirEntreEnderecos(e.ctx, mov(nil), "A1", "A2"); !ehValidacao(err) {
		t.Errorf("transferir 15 com 10 reservados de 20: esperava recusa, veio %v", err)
	}
	testutil.Exec(t, pool, `INSERT INTO manufacturing_stock_closed_periods(enterprise_id,period_from,period_to) VALUES($1,CURRENT_DATE-1,CURRENT_DATE+1)`, e.empresa)
	m := mov(nil)
	m.Quantity = decimal.NewFromInt(1)
	if _, err := e.repo.TransferirEntreEnderecos(e.ctx, m, "A1", "A2"); !ehValidacao(err) {
		t.Errorf("período fechado: esperava recusa, veio %v", err)
	}
}

func TestAuditoria_SeparacaoNaoSugereInspecao(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	var insp int64
	exigir(t, pool.QueryRow(context.Background(),
		`INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		 VALUES('INSP-'||$1,'insp',$2,'INSPECAO','NORMAL',false,false,$3) RETURNING id`, decimal.NewFromInt(e.item).String(), e.ator, e.empresa).Scan(&insp))
	exigir(t, e.moverLote(t, e.item, insp, "IN", 10, 5, "", "L1"))
	r, err := e.repo.SugerirSeparacaoFEFO(e.ctx, e.item, "", 0, decimal.NewFromInt(5), "FEFO")
	exigir(t, err)
	if len(r.Linhas) != 0 {
		t.Errorf("sugeriu tirar do almoxarifado de inspeção: %+v", r.Linhas)
	}
}

func TestAuditoria_OndaRespeitaReservaDoItem(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "A1", "L1"))
	_, err := e.reservar(t, e.item, e.almox, 10)
	exigir(t, err)
	onda, err := e.repo.CriarOndaDeSeparacao(e.ctx, e.item, e.almox, "FEFO",
		[]stockentity.NecessidadeOnda{{ItemCode: e.item, Quantity: dec("10")}}, e.ator)
	exigir(t, err)
	if len(onda.Linhas) != 0 {
		t.Errorf("onda alocou material já reservado: %+v", onda.Linhas)
	}
}

func TestAuditoria_SaldoPorEnderecoMostraItemSemLote(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 5, "A1"))
	l, err := e.repo.ListarSaldoPorEndereco(e.ctx, e.almox, e.item)
	exigir(t, err)
	if len(l) != 1 || l[0].Address != "A1" {
		t.Errorf("saldo por endereço sem o item sem lote: %+v", l)
	}
}

func TestAuditoria_GuardaNaoSugereEnderecoFixoDeOutroItem(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM manufacturing_warehouse_addresses WHERE enterprise_id=$1`, e.empresa)
	})
	testutil.Exec(t, pool, `INSERT INTO manufacturing_warehouse_addresses(enterprise_id,warehouse_id,address,fixed_item_code) VALUES($1,$2,'FIXO',$3),($1,$2,'LIVRE',NULL)`, e.empresa, e.almox, e.outro)
	s, err := e.repo.SugerirEnderecoDeGuarda(e.ctx, e.item, "", e.almox, 1, "")
	exigir(t, err)
	for _, x := range s {
		if x.Address == "FIXO" {
			t.Errorf("sugeriu o endereço fixo de outro item")
		}
	}
}

func TestAuditoria_ConsumoIgnoraCancelamentoETipoSemConsumo(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	exigir(t, e.mover(t, e.item, e.almox, "IN", 100, 1, ""))
	ref := stockentity.ReferenceTypeNFEntry
	code := int64(1)
	_, err := e.repo.CreateMovement(e.ctx, &stockentity.StockMovement{ItemCode: e.item, WarehouseID: e.almox, MovementType: "OUT",
		Quantity: decimal.NewFromInt(10), UnitPrice: decimal.NewFromInt(1), SaidaPeloPreco: true, ReferenceType: &ref, ReferenceCode: &code, CreatedBy: e.ator})
	exigir(t, err)
	c, err := e.repo.RecalcConsumptionAverage(e.ctx, e.item, 6)
	exigir(t, err)
	igual(t, "consumo com o cancelamento de nota", c.TotalConsumed, "0")

	sigla := "NC" + uuidCurto(e.item)
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM stock_movement_types WHERE enterprise_id=$1`, e.empresa) })
	testutil.Exec(t, pool, `INSERT INTO stock_movement_types(sigla,description,usage_type,considers_consumption,entry_exit,enterprise_id) VALUES($1,'x','GERAL',false,'SAIDA',$2)`, sigla, e.empresa)
	_, err = e.repo.CreateMovement(e.ctx, &stockentity.StockMovement{ItemCode: e.item, WarehouseID: e.almox, MovementType: "OUT",
		Quantity: decimal.NewFromInt(10), MovementTypeCode: &sigla, CreatedBy: e.ator})
	exigir(t, err)
	a, err := e.repo.AnaliseDeEstoque(e.ctx, 6, 90)
	exigir(t, err)
	for _, x := range a {
		if x.ItemCode == e.item {
			igual(t, "consumo da análise", x.ConsumoQtd, "0")
		}
	}
}

func uuidCurto(n int64) string { return decimal.NewFromInt(n % 100000).String() }

func TestAuditoria_InventarioBaixaEnderecosELotes(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "A1", "L1"))
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "A2", "L2"))
	inv, err := e.repo.CreateInventory(e.ctx, &stockentity.PhysicalInventory{Code: e.item, Description: "x", WarehouseID: e.almox, StartDate: time.Now(), Status: "OPEN", CreatedBy: e.ator})
	exigir(t, err)
	e.contar(t, inv.ID, "A1", "L1", 5)
	e.contar(t, inv.ID, "A2", "L2", 10)
	_, err = e.repo.FecharInventario(e.ctx, inv.ID, e.ator, false, "")
	exigir(t, err)
	igual(t, "total", e.somar(t, `SELECT SUM(quantity) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item), "15")
	igual(t, "linha sem endereço", e.saldoEndereco(t, e.item, e.almox, ""), "0")
	igual(t, "lotes", e.somar(t, `SELECT SUM(quantity) FROM stock_lot_balances WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item), "15")
}

func TestAuditoria_CurvaABCReclassificaQuemParouDeConsumir(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 5, ""))
	exigir(t, e.mover(t, e.item, e.almox, "OUT", 5, 0, ""))
	testutil.Exec(t, pool, `UPDATE items SET planning_abc_class='A', abc_consumption_value=9999 WHERE code=$1`, e.outro)
	_, err := e.repo.ApurarCurvaABC(e.ctx, 12, 80, 95)
	exigir(t, err)
	var classe string
	exigir(t, pool.QueryRow(context.Background(), `SELECT planning_abc_class::text FROM items WHERE code=$1`, e.outro).Scan(&classe))
	if classe != "C" {
		t.Errorf("item sem consumo ficou como %s", classe)
	}
}

func TestAuditoria_ConsumoParcialDaReserva(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 5, ""))
	r, err := e.reservar(t, e.item, e.almox, 8)
	exigir(t, err)
	c, err := e.repo.ConsumirReservaParcial(e.ctx, r.ID, decimal.NewFromInt(3))
	exigir(t, err)
	igual(t, "consumido", c, "3")
	got, err := e.repo.GetReservation(e.ctx, r.ID)
	exigir(t, err)
	igual(t, "resto da reserva", got.Quantity, "5")
	if got.Status != "ACTIVE" {
		t.Errorf("reserva parcial encerrou: %s", got.Status)
	}
	b, err := e.repo.GetBalance(e.ctx, e.item, "", e.almox)
	exigir(t, err)
	igual(t, "reservado no saldo", b.ReservedQty, "5")
}

func TestAuditoria_OndaSeparaItemSemLote(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 5, "A1"))
	onda, err := e.repo.CriarOndaDeSeparacao(e.ctx, e.item, e.almox, "FEFO",
		[]stockentity.NecessidadeOnda{{ItemCode: e.item, Quantity: dec("4")}}, e.ator)
	exigir(t, err)
	if len(onda.Linhas) != 1 || len(onda.EmFalta) != 0 {
		t.Fatalf("onda sem a linha do item sem lote: %+v falta %+v", onda.Linhas, onda.EmFalta)
	}
	_, err = e.repo.ConfirmarOndaDeSeparacao(e.ctx, onda.Code, nil, e.ator)
	exigir(t, err)
	igual(t, "saldo depois da onda", e.saldoEndereco(t, e.item, e.almox, "A1"), "6")
}

// ---- Fase A: inventário por endereço e lote, condução e inconsistências ----

func (e ambienteEstoque) abrirInventario(t *testing.T, ajuste func(*stockentity.PhysicalInventory)) *stockentity.PhysicalInventory {
	t.Helper()
	inv := &stockentity.PhysicalInventory{Code: e.item, Description: "x", WarehouseID: e.almox, StartDate: time.Now(), Status: "OPEN", CreatedBy: e.ator}
	if ajuste != nil {
		ajuste(inv)
	}
	out, err := e.repo.CreateInventory(e.ctx, inv)
	exigir(t, err)
	return out
}

func (e ambienteEstoque) contar(t *testing.T, inv int64, endereco, lote string, qtd int64) {
	t.Helper()
	q := decimal.NewFromInt(qtd)
	exigir(t, e.repo.CountInventoryItem(e.ctx, &stockentity.PhysicalInventoryItem{InventoryID: inv, ItemCode: e.item, WarehouseID: e.almox, Address: endereco, Lot: lote, CountedQty: &q, CountedBy: &e.ator}))
}

func TestFaseA_InventarioAjustaOEnderecoEOLoteContados(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "A1", "L1"))
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "A2", "L2"))
	exigir(t, e.mover(t, e.item, e.almox, "IN", 4, 5, "A2"))
	inv := e.abrirInventario(t, nil)
	if inv.TotalItems != 3 {
		t.Fatalf("esperava 3 linhas (A1/L1, A2/L2, A2 sem lote), veio %d", inv.TotalItems)
	}
	e.contar(t, inv.ID, "A1", "L1", 10)
	e.contar(t, inv.ID, "A2", "L2", 7)
	e.contar(t, inv.ID, "A2", "", 6)
	res, err := e.repo.FecharInventario(e.ctx, inv.ID, e.ator, false, "contagem anual")
	exigir(t, err)
	if res.Ajustes != 2 {
		t.Errorf("ajustes: %d", res.Ajustes)
	}
	igual(t, "A1 intacto", e.saldoEndereco(t, e.item, e.almox, "A1"), "10")
	igual(t, "A2", e.saldoEndereco(t, e.item, e.almox, "A2"), "13")
	igual(t, "L1", e.saldoLote(t, e.item, e.almox, "L1"), "10")
	igual(t, "L2", e.saldoLote(t, e.item, e.almox, "L2"), "7")
	igual(t, "sem endereço", e.saldoEndereco(t, e.item, e.almox, ""), "0")
}

func TestFaseA_DiferencaValeContraAFoto(t *testing.T) {
	// Exemplo da ajuda do Focco: foto 1, contou 10, faturou 5 depois, acerto +9.
	e := novoAmbienteEstoque(t)
	testutil.Exec(t, testutil.Pool(t), `UPDATE warehouse SET allow_negative_stock=true WHERE id=$1`, e.almox)
	exigir(t, e.mover(t, e.item, e.almox, "IN", 1, 5, ""))
	inv := e.abrirInventario(t, nil)
	e.contar(t, inv.ID, "", "", 10)
	exigir(t, e.mover(t, e.item, e.almox, "OUT", 5, 0, ""))
	_, err := e.repo.FecharInventario(e.ctx, inv.ID, e.ator, false, "")
	exigir(t, err)
	q, _ := e.saldo(t, e.item, e.almox)
	if q != 5 {
		t.Errorf("saldo final %v, esperado 5", q)
	}
}

func TestFaseA_BloqueioDeMovimentacao(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 5, ""))
	exigir(t, e.mover(t, e.outro, e.almox, "IN", 10, 5, ""))
	inv := e.abrirInventario(t, func(i *stockentity.PhysicalInventory) {
		i.BlocksMovements, i.ItemCodes = true, []int64{e.item}
	})
	if err := e.mover(t, e.item, e.almox, "OUT", 1, 0, ""); !ehValidacao(err) {
		t.Errorf("item em inventário bloqueado: esperava recusa, veio %v", err)
	}
	exigir(t, e.mover(t, e.outro, e.almox, "OUT", 1, 0, "")) // fora do inventário parcial
	e.contar(t, inv.ID, "", "", 8)
	_, err := e.repo.FecharInventario(e.ctx, inv.ID, e.ator, false, "")
	exigir(t, err) // o próprio ajuste passa
	exigir(t, e.mover(t, e.item, e.almox, "OUT", 1, 0, ""))
	q, _ := e.saldo(t, e.item, e.almox)
	if q != 7 {
		t.Errorf("saldo %v, esperado 7", q)
	}
}

func TestFaseA_RecontagemVoltaSoAsForaDaTolerancia(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.mover(t, e.item, e.almox, "IN", 100, 1, "A1"))
	exigir(t, e.mover(t, e.item, e.almox, "IN", 100, 1, "A2"))
	pct := decimal.NewFromInt(10)
	inv := e.abrirInventario(t, func(i *stockentity.PhysicalInventory) { i.TolerancePct = &pct })
	e.contar(t, inv.ID, "A1", "", 95)
	e.contar(t, inv.ID, "A2", "", 80)
	exigir(t, e.repo.EncerrarContagem(e.ctx, inv.ID))
	lido, err := e.repo.GetInventory(e.ctx, inv.ID)
	exigir(t, err)
	r, err := e.repo.ResumoDoFechamento(e.ctx, lido, false)
	exigir(t, err)
	if r.ComDiferenca != 2 || r.ForaTolerancia != 1 {
		t.Fatalf("resumo %+v", r)
	}
	n, err := e.repo.MarcarRecontagem(e.ctx, lido, nil)
	exigir(t, err)
	if n != 1 {
		t.Fatalf("linhas a recontar: %d", n)
	}
	e.contar(t, inv.ID, "A2", "", 98)
	itens, err := e.repo.ListInventoryItems(e.ctx, inv.ID)
	exigir(t, err)
	for _, it := range itens {
		if it.Address == "A2" && (it.NeedsRecount || it.CountNumber != 2 || !it.PreviousCountedQty.Equal(decimal.NewFromInt(80))) {
			t.Errorf("recontagem não ficou registrada: %+v", it)
		}
	}
}

func TestFaseA_InconsistenciasEAcerto(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM stock_inconsistency_fixes WHERE enterprise_id=$1`, e.empresa)
		testutil.Exec(t, pool, `DELETE FROM stock_reservations WHERE enterprise_id=$1`, e.empresa)
	})
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "A1", "L1"))
	l, err := e.repo.ListarInconsistencias(e.ctx, &e.almox, nil)
	exigir(t, err)
	if len(l) != 0 {
		t.Fatalf("saldo correto apontado: %+v", l)
	}
	testutil.Exec(t, pool, `UPDATE stock_balances SET quantity=7, reserved_qty=2 WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item)
	l, err = e.repo.ListarInconsistencias(e.ctx, &e.almox, &e.item)
	exigir(t, err)
	if len(l) != 3 {
		t.Fatalf("esperava 3 inconsistências, veio %+v", l)
	}
	for _, tipo := range []string{stockentity.InconsistenciaReservado, stockentity.InconsistenciaSaldo} {
		exigir(t, e.repo.CorrigirInconsistencia(e.ctx, tipo, e.item, "", e.almox, "teste de acerto", e.ator))
	}
	l, err = e.repo.ListarInconsistencias(e.ctx, &e.almox, &e.item)
	exigir(t, err)
	if len(l) != 0 {
		t.Errorf("sobrou: %+v", l)
	}
	igual(t, "A1 + acerto", e.somar(t, `SELECT SUM(quantity) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item), "10")
	igual(t, "registros do acerto", e.somar(t, `SELECT COUNT(*) FROM stock_inconsistency_fixes WHERE enterprise_id=$1`, e.empresa), "2")
	// Lote acima do saldo.
	testutil.Exec(t, pool, `UPDATE stock_lot_balances SET quantity=30 WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item)
	exigir(t, e.repo.CorrigirInconsistencia(e.ctx, stockentity.InconsistenciaLote, e.item, "", e.almox, "teste de acerto", e.ator))
	igual(t, "lote acertado", e.saldoLote(t, e.item, e.almox, "L1"), "7")
}

func TestFaseA_MovimentosPaginados(t *testing.T) {
	e := novoAmbienteEstoque(t)
	for i := 0; i < 3; i++ {
		exigir(t, e.mover(t, e.item, e.almox, "IN", 1, 1, ""))
	}
	l, err := e.repo.ListMovementsPage(e.ctx, stockentity.FiltroMovimentos{Almoxarifado: &e.almox, Limite: 2})
	exigir(t, err)
	if len(l) != 2 {
		t.Fatalf("página: %d", len(l))
	}
	l2, err := e.repo.ListMovementsPage(e.ctx, stockentity.FiltroMovimentos{Almoxarifado: &e.almox, Limite: 2, Deslocamento: 2})
	exigir(t, err)
	if len(l2) != 1 || l2[0].ID == l[0].ID {
		t.Fatalf("segunda página: %+v", l2)
	}
}

// ---- Fase B: controle de material ----

func (e ambienteEstoque) limparFaseB(t *testing.T) {
	t.Helper()
	pool := testutil.Pool(t)
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM stock_lot_status_events WHERE enterprise_id=$1`,
			`DELETE FROM stock_lots WHERE enterprise_id=$1`,
			`DELETE FROM stock_lot_statuses WHERE enterprise_id=$1`,
			`DELETE FROM stock_parameters WHERE enterprise_id=$1`,
			`DELETE FROM manufacturing_stock_item_controls WHERE enterprise_id=$1`,
		} {
			testutil.Exec(t, pool, q, e.empresa)
		}
	})
}

func (e ambienteEstoque) situacao(t *testing.T, codigo string, saida, transf, conta bool) {
	t.Helper()
	exigir(t, e.repo.SalvarSituacaoLote(e.ctx, stockentity.SituacaoLote{Code: codigo, Description: codigo, AllowsIssue: saida, AllowsTransfer: transf, CountsAvailable: conta, IsActive: true}))
}

func TestFaseB_ValidadeDoMovimentoChegaAoLote(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseB(t)
	lote := "L1"
	validade := time.Now().AddDate(0, 0, 30)
	_, err := e.repo.CreateMovement(e.ctx, &stockentity.StockMovement{ItemCode: e.item, WarehouseID: e.almox, MovementType: "IN",
		Quantity: decimal.NewFromInt(5), UnitPrice: decimal.NewFromInt(1), Lot: &lote, ExpirationDate: &validade, CreatedBy: e.ator})
	exigir(t, err)
	l, err := e.repo.GetLot(e.ctx, e.item, "L1")
	exigir(t, err)
	if l == nil || l.ExpiresAt == nil {
		t.Fatalf("a validade não chegou ao cadastro do lote: %+v", l)
	}
}

func TestFaseB_LancamentoManualExigeLoteEEndereco(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseB(t)
	pool := testutil.Pool(t)
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM manufacturing_warehouse_addresses WHERE enterprise_id=$1`, e.empresa)
	})
	testutil.Exec(t, pool, `INSERT INTO manufacturing_warehouse_addresses(enterprise_id,warehouse_id,address) VALUES($1,$2,'A1')`, e.empresa, e.almox)
	exigir(t, e.repo.SalvarControleDoItem(e.ctx, stockentity.ControleDoItem{ItemCode: e.item, ControlsLot: true, ControlsAddress: true, LotIssueRule: "FEFO"}))
	manual := func(lote, endereco string) error {
		m := &stockentity.StockMovement{ItemCode: e.item, WarehouseID: e.almox, MovementType: "IN", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(1), CreatedBy: e.ator, ExigirRastreio: true}
		if lote != "" {
			m.Lot = &lote
		}
		if endereco != "" {
			m.Address = &endereco
		}
		_, err := e.repo.CreateMovement(e.ctx, m)
		return err
	}
	if err := manual("", "A1"); !ehValidacao(err) {
		t.Errorf("sem lote: esperava recusa, veio %v", err)
	}
	if err := manual("L1", ""); !ehValidacao(err) {
		t.Errorf("sem endereço: esperava recusa, veio %v", err)
	}
	exigir(t, manual("L1", "A1"))
	// Processo automático segue pela baixa automática.
	exigir(t, e.mover(t, e.item, e.almox, "OUT", 1, 0, ""))
}

func TestFaseB_LoteSemSaidaFicaRetido(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseB(t)
	e.situacao(t, "QUARENTENA", false, true, false)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "", "L1"))
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 4, 5, "", "L2"))
	st := "QUARENTENA"
	exigir(t, e.repo.MudarSituacaoDoLote(e.ctx, stockentity.MudancaSituacaoLote{ItemCode: e.item, Lot: "L1", Status: &st, Reason: "laudo pendente"}, e.ator))

	if err := e.moverLote(t, e.item, e.almox, "OUT", 1, 0, "", "L1"); !ehValidacao(err) {
		t.Errorf("saída de lote em quarentena: esperava recusa, veio %v", err)
	}
	// Saída sem lote só alcança o que pode sair.
	if err := e.mover(t, e.item, e.almox, "OUT", 5, 0, ""); !ehValidacao(err) {
		t.Errorf("saída maior que o livre: esperava recusa, veio %v", err)
	}
	exigir(t, e.mover(t, e.item, e.almox, "OUT", 3, 0, ""))
	igual(t, "L1 intacto", e.saldoLote(t, e.item, e.almox, "L1"), "10")
	igual(t, "L2", e.saldoLote(t, e.item, e.almox, "L2"), "1")

	s, err := e.repo.SaldosComSituacao(e.ctx, e.item)
	exigir(t, err)
	igual(t, "disponível sem a quarentena", s[0].AvailableQty, "1")

	r, err := e.repo.SugerirSeparacaoFEFO(e.ctx, e.item, "", e.almox, decimal.NewFromInt(5), "FEFO")
	exigir(t, err)
	for _, l := range r.Linhas {
		if l.Lot == "L1" {
			t.Error("sugeriu separar o lote em quarentena")
		}
	}
	// Transferir é permitido pela situação; o descarte é por ajuste.
	outro := e.livre
	_, err = e.repo.TransferirEntreAlmoxarifados(e.ctx, stockentity.TransferenciaEstoque{ItemCode: e.item, DeAlmox: e.almox, ParaAlmox: outro, Quantidade: decimal.NewFromInt(2), Lote: strPtr("L1"), Ator: e.ator})
	exigir(t, err)
	exigir(t, e.moverLote(t, e.item, e.almox, "ADJUSTMENT", -1, 0, "", "L1"))
	igual(t, "L1 depois", e.saldoLote(t, e.item, e.almox, "L1"), "7")
	igual(t, "registro da mudança", e.somar(t, `SELECT COUNT(*) FROM stock_lot_status_events WHERE enterprise_id=$1`, e.empresa), "1")
}

func strPtr(s string) *string { return &s }

func TestFaseB_LoteVencido(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseB(t)
	pool := testutil.Pool(t)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "", "L1"))
	testutil.Exec(t, pool, `UPDATE stock_lots SET expires_at=$3::date WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item, time.Now().AddDate(0, 0, -10))
	if err := e.moverLote(t, e.item, e.almox, "OUT", 1, 0, "", "L1"); !ehValidacao(err) {
		t.Errorf("lote vencido: esperava recusa, veio %v", err)
	}
	if err := e.mover(t, e.item, e.almox, "OUT", 1, 0, ""); !ehValidacao(err) {
		t.Errorf("saída sem lote de saldo vencido: esperava recusa, veio %v", err)
	}
	exigir(t, e.repo.SalvarParametrosDeEstoque(e.ctx, stockentity.ParametrosEstoque{ExpiredLotAction: "AVISAR"}, e.ator))
	exigir(t, e.moverLote(t, e.item, e.almox, "OUT", 1, 0, "", "L1"))
}

func TestFaseB_BaixaAutomaticaSegueARegraDoItem(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseB(t)
	pool := testutil.Pool(t)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "", "ANTIGO"))
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "", "VENCE-ANTES"))
	testutil.Exec(t, pool, `UPDATE stock_lots SET received_at=$3::date, expires_at=$4::date WHERE enterprise_id=$1 AND item_code=$2 AND lot='ANTIGO'`, e.empresa, e.item, time.Now().AddDate(0, 0, -30), time.Now().AddDate(0, 0, 90))
	testutil.Exec(t, pool, `UPDATE stock_lots SET received_at=$3::date, expires_at=$4::date WHERE enterprise_id=$1 AND item_code=$2 AND lot='VENCE-ANTES'`, e.empresa, e.item, time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 10))
	exigir(t, e.mover(t, e.item, e.almox, "OUT", 3, 0, ""))
	igual(t, "FEFO tira do que vence antes", e.saldoLote(t, e.item, e.almox, "VENCE-ANTES"), "7")
	exigir(t, e.repo.SalvarControleDoItem(e.ctx, stockentity.ControleDoItem{ItemCode: e.item, LotIssueRule: "FIFO"}))
	exigir(t, e.mover(t, e.item, e.almox, "OUT", 3, 0, ""))
	igual(t, "FIFO tira do que chegou antes", e.saldoLote(t, e.item, e.almox, "ANTIGO"), "7")
}

// A sugestão de separação sem regra escolhida segue a regra do item: o item FIFO
// era sugerido em FEFO, porque a tela e o repositório partiam sempre de FEFO.
func TestFaseB_SeparacaoSegueARegraDoItem(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseB(t)
	pool := testutil.Pool(t)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "", "ANTIGO"))
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "", "VENCE-ANTES"))
	testutil.Exec(t, pool, `UPDATE stock_lots SET received_at=$3::date, expires_at=$4::date WHERE enterprise_id=$1 AND item_code=$2 AND lot='ANTIGO'`, e.empresa, e.item, time.Now().AddDate(0, 0, -30), time.Now().AddDate(0, 0, 90))
	testutil.Exec(t, pool, `UPDATE stock_lots SET received_at=$3::date, expires_at=$4::date WHERE enterprise_id=$1 AND item_code=$2 AND lot='VENCE-ANTES'`, e.empresa, e.item, time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 20))
	primeiro := func(regra string) (string, string) {
		r, err := e.repo.SugerirSeparacaoFEFO(e.ctx, e.item, "", e.almox, decimal.NewFromInt(3), regra)
		exigir(t, err)
		if len(r.Linhas) == 0 {
			t.Fatalf("sem sugestão para a regra %q", regra)
		}
		return r.Linhas[0].Lot, r.Regra
	}
	if lote, regra := primeiro(""); lote != "VENCE-ANTES" || regra != "FEFO" {
		t.Fatalf("item FEFO sem regra pedida: %s (%s)", lote, regra)
	}
	exigir(t, e.repo.SalvarControleDoItem(e.ctx, stockentity.ControleDoItem{ItemCode: e.item, ControlsLot: true, LotIssueRule: "FIFO"}))
	if lote, regra := primeiro(""); lote != "ANTIGO" || regra != "FIFO" {
		t.Fatalf("item FIFO sem regra pedida: %s (%s)", lote, regra)
	}
	if lote, _ := primeiro("FEFO"); lote != "VENCE-ANTES" {
		t.Fatalf("FEFO pedido vence a regra do item: %s", lote)
	}
}

func TestFaseB_ImplantarLote(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseB(t)
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 5, "A1"))
	i := stockentity.ImplantacaoLote{ItemCode: e.item, WarehouseID: e.almox, Address: "A1", Lot: "L1", Quantity: decimal.NewFromInt(6)}
	exigir(t, e.repo.ImplantarLote(e.ctx, i, e.ator))
	igual(t, "lote", e.saldoLote(t, e.item, e.almox, "L1"), "6")
	igual(t, "saldo não muda", e.saldoEndereco(t, e.item, e.almox, "A1"), "10")
	i.Quantity = decimal.NewFromInt(5)
	if err := e.repo.ImplantarLote(e.ctx, i, e.ator); !ehValidacao(err) {
		t.Errorf("implantar além do saldo sem lote: esperava recusa, veio %v", err)
	}
	l, err := e.repo.ListarInconsistencias(e.ctx, &e.almox, &e.item)
	exigir(t, err)
	if len(l) != 0 {
		t.Errorf("implantação criou inconsistência: %+v", l)
	}
}

// ---- Fase C: data do documento e fechamento ----

func (e ambienteEstoque) limparFaseC(t *testing.T) {
	t.Helper()
	pool := testutil.Pool(t)
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM stock_cost_variances WHERE enterprise_id=$1`,
			`DELETE FROM stock_closings WHERE enterprise_id=$1`,
			`DELETE FROM manufacturing_stock_closed_periods WHERE enterprise_id=$1`,
		} {
			testutil.Exec(t, pool, q, e.empresa)
		}
	})
}

func (e ambienteEstoque) moverEm(t *testing.T, data time.Time, tipo string, qtd, preco int64) error {
	t.Helper()
	_, err := e.repo.CreateMovement(e.ctx, &stockentity.StockMovement{ItemCode: e.item, WarehouseID: e.almox, MovementType: tipo,
		Quantity: decimal.NewFromInt(qtd), UnitPrice: decimal.NewFromInt(preco), MovementDate: &data, CreatedBy: e.ator})
	return err
}

func mesPassado() (time.Time, time.Time) {
	agora := time.Now()
	inicio := time.Date(agora.Year(), agora.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	return inicio, inicio.AddDate(0, 1, -1)
}

func TestFaseC_FechamentoRecalculaORetroativo(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseC(t)
	inicio, _ := mesPassado()
	dia := func(n int) time.Time { return inicio.AddDate(0, 0, n-1) }
	exigir(t, e.moverEm(t, dia(5), "IN", 10, 100))
	exigir(t, e.moverEm(t, dia(10), "OUT", 4, 0))
	// Nota lançada depois, com a data de antes da saída.
	exigir(t, e.moverEm(t, dia(7), "IN", 10, 200))

	sim, err := e.repo.FecharMes(e.ctx, inicio, e.ator, true)
	exigir(t, err)
	if len(sim.Pendencias) != 0 || sim.MovimentosAlterados != 1 {
		t.Fatalf("simulação: %+v", sim)
	}
	igual(t, "diferença simulada", sim.DiferencaRecalculo, "200")
	igual(t, "simulação não grava", e.somar(t, `SELECT COUNT(*) FROM stock_closings WHERE enterprise_id=$1`, e.empresa), "0")
	igual(t, "saída ainda no valor antigo", e.somar(t, `SELECT total_price FROM stock_movements WHERE enterprise_id=$1 AND movement_type='OUT'`, e.empresa), "400")

	res, err := e.repo.FecharMes(e.ctx, inicio, e.ator, false)
	exigir(t, err)
	igual(t, "saída recalculada", e.somar(t, `SELECT total_price FROM stock_movements WHERE enterprise_id=$1 AND movement_type='OUT'`, e.empresa), "600")
	igual(t, "valor da foto", res.ValorTotal, "2400")
	igual(t, "foto", e.somar(t, `SELECT quantity FROM stock_closing_balances b JOIN stock_closings c ON c.id=b.closing_id WHERE c.enterprise_id=$1`, e.empresa), "16")
	q, custo := e.saldo(t, e.item, e.almox)
	if q != 16 || custo != 150 {
		t.Errorf("saldo %v a %v, esperado 16 a 150", q, custo)
	}
	if !strings.Contains(strings.Join(res.Avisos, " "), "contas") {
		t.Errorf("sem contas configuradas devia avisar: %+v", res.Avisos)
	}
	l, err := e.repo.ListarInconsistencias(e.ctx, &e.almox, &e.item)
	exigir(t, err)
	if len(l) != 0 {
		t.Errorf("fechamento criou inconsistência: %+v", l)
	}
	if err := e.moverEm(t, inicio.AddDate(0, 0, 14), "IN", 1, 1); !ehValidacao(err) {
		t.Errorf("mês fechado: esperava recusa, veio %v", err)
	}
	// Documento que já existe, de mês fechado, entra hoje.
	data := inicio.AddDate(0, 0, 14)
	m := &stockentity.StockMovement{ItemCode: e.item, WarehouseID: e.almox, MovementType: "IN", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(150), MovementDate: &data, DataAjustavel: true, CreatedBy: e.ator}
	_, err = e.repo.CreateMovement(e.ctx, m)
	exigir(t, err)
	if m.MovementDate.Before(inicio.AddDate(0, 1, 0)) {
		t.Errorf("documento de mês fechado ficou em %v", m.MovementDate)
	}
	// Reabrir devolve o período.
	f, err := e.repo.Fechamentos(e.ctx)
	exigir(t, err)
	exigir(t, e.repo.ReabrirMes(e.ctx, f[0].ID, "lançamento esquecido", e.ator))
	exigir(t, e.moverEm(t, inicio.AddDate(0, 0, 14), "IN", 1, 150))
}

func TestFaseC_SemRetroativoNaoMexeNoCusto(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseC(t)
	inicio, _ := mesPassado()
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 100, ""))
	r, err := e.repo.FecharMes(e.ctx, inicio, e.ator, false)
	exigir(t, err)
	if r.ItensRecalculados != 0 || r.MovimentosAlterados != 0 {
		t.Errorf("recalculou sem precisar: %+v", r)
	}
}

func TestFaseC_PendenciasImpedemFechar(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseC(t)
	pool := testutil.Pool(t)
	inicio, _ := mesPassado()
	exigir(t, e.mover(t, e.item, e.almox, "IN", 10, 100, ""))
	mesCorrente := inicio.AddDate(0, 1, 0)
	if _, err := e.repo.FecharMes(e.ctx, mesCorrente, e.ator, false); !ehValidacao(err) {
		t.Errorf("mês em curso: esperava recusa, veio %v", err)
	}
	testutil.Exec(t, pool, `UPDATE stock_balances SET quantity=9 WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item)
	s, err := e.repo.FecharMes(e.ctx, inicio, e.ator, true)
	exigir(t, err)
	if len(s.Pendencias) != 1 {
		t.Fatalf("simulação devia apontar a inconsistência: %+v", s.Pendencias)
	}
	if _, err := e.repo.FecharMes(e.ctx, inicio, e.ator, false); !ehValidacao(err) {
		t.Errorf("com inconsistência: esperava recusa, veio %v", err)
	}
}

func TestFaseC_EntradaSobreSaldoNegativoVaiParaODiferenca(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseC(t)
	exigir(t, e.mover(t, e.item, e.livre, "OUT", 5, 0, ""))
	exigir(t, e.mover(t, e.item, e.livre, "IN", 10, 100, ""))
	igual(t, "diferença", e.somar(t, `SELECT value FROM stock_cost_variances WHERE enterprise_id=$1 AND kind='SALDO_NEGATIVO'`, e.empresa), "500")
}

func TestFaseC_KardexEPosicaoUsamADataDoDocumento(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseC(t)
	inicio, fim := mesPassado()
	exigir(t, e.moverEm(t, inicio.AddDate(0, 0, 3), "IN", 10, 100))
	exigir(t, e.mover(t, e.item, e.almox, "IN", 5, 100, ""))
	p, err := e.repo.PosicaoEmData(e.ctx, fim, &e.almox)
	exigir(t, err)
	if len(p) != 1 {
		t.Fatalf("posição: %+v", p)
	}
	igual(t, "posição no fim do mês passado", p[0].Quantidade, "10")
	k, err := e.repo.Kardex(e.ctx, e.item, &e.almox, inicio, fim)
	exigir(t, err)
	if len(k.Linhas) != 1 || k.Linhas[0].DataDocumento != inicio.AddDate(0, 0, 3).Format("2006-01-02") {
		t.Errorf("kardex: %+v", k.Linhas)
	}
}

func TestFaseC_DiferencaVaiParaAContabilidadeEEstornaAoReabrir(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseC(t)
	pool := testutil.Pool(t)
	bg := context.Background()
	u := testutil.UniqueCode()
	var plano int64
	exigir(t, pool.QueryRow(bg, `INSERT INTO accounting_plans(plan_number,description,valid_from,status,enterprise_id) VALUES($1,'Estoque','2020-01-01','A',$2) RETURNING id`, u%1_000_000+700_000, e.empresa).Scan(&plano))
	conta := func(numero string) int64 {
		var id int64
		exigir(t, pool.QueryRow(bg, `INSERT INTO accounting_accounts(plan_id,account_number,description,nature_code,valid_from,is_analytic) VALUES($1,$2,$2,'D','2020-01-01',TRUE) RETURNING id`, plano, numero).Scan(&id))
		return id
	}
	cForn, cCMV, cEstoque := conta("2.1.1"), conta("3.1 CMV"), conta("1.1.3 ESTOQUE")
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM accounting_journal_entries WHERE empresa_id=$1`, e.empresa)
		_, _ = pool.Exec(bg, `DELETE FROM accounting_posting_params WHERE enterprise_id=$1`, e.empresa)
		_, _ = pool.Exec(bg, `DELETE FROM accounting_accounts WHERE plan_id=$1`, plano)
		_, _ = pool.Exec(bg, `DELETE FROM accounting_plans WHERE id=$1`, plano)
	})
	testutil.Exec(t, pool, `INSERT INTO accounting_posting_params(enterprise_id,plan_id,fornecedores_account_id,cmv_account_id,estoque_account_id,updated_by)
		VALUES($1,$2,$3,$4,$5,$6)`, e.empresa, plano, cForn, cCMV, cEstoque, e.ator)

	inicio, fim := mesPassado()
	dia := func(n int) time.Time { return inicio.AddDate(0, 0, n-1) }
	exigir(t, e.moverEm(t, dia(5), "IN", 10, 100))
	exigir(t, e.moverEm(t, dia(10), "OUT", 4, 0))
	exigir(t, e.moverEm(t, dia(7), "IN", 10, 200))
	r, err := e.repo.FecharMes(e.ctx, inicio, e.ator, false)
	exigir(t, err)
	if !r.Contabilizado {
		t.Fatalf("não contabilizou: %+v", r)
	}
	lancamentos := func() decimal.Decimal {
		return e.somar(t, `SELECT COUNT(*) FROM accounting_journal_entries WHERE empresa_id=$1 AND source_type='FECHAMENTO_ESTOQUE' AND debit_account_id=$2 AND credit_account_id=$3 AND value=200 AND entry_date=$4::date`,
			e.empresa, cCMV, cEstoque, fim.Format("2006-01-02"))
	}
	igual(t, "D CMV / C Estoque de 200", lancamentos(), "1")
	f, err := e.repo.Fechamentos(e.ctx)
	exigir(t, err)
	exigir(t, e.repo.ReabrirMes(e.ctx, f[0].ID, "lançamento esquecido", e.ator))
	igual(t, "estorno", e.somar(t, `SELECT COUNT(*) FROM accounting_journal_entries WHERE empresa_id=$1 AND source_type='FECH_ESTOQUE_ESTORNO' AND debit_account_id=$2 AND credit_account_id=$3`, e.empresa, cEstoque, cCMV), "1")
}

// ---------- Fase D: reposição e requisição ----------

func decp(s string) *decimal.Decimal {
	v := dec(s)
	return &v
}

func reservadoDe(t *testing.T, e ambienteEstoque, item, almox int64) decimal.Decimal {
	t.Helper()
	var q decimal.Decimal
	if err := testutil.Pool(t).QueryRow(context.Background(),
		`SELECT COALESCE(SUM(reserved_qty),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3`,
		e.empresa, item, almox).Scan(&q); err != nil {
		t.Fatal(err)
	}
	return q
}

// A requisição reserva ao abrir, até o que estiver livre: outro pedido não leva o
// material. Cancelar devolve o que restava reservado.
func TestFaseD_RequisicaoReservaAoAbrirECancelarDevolve(t *testing.T) {
	e := novoAmbienteEstoque(t)
	_ = e.mover(t, e.item, e.almox, "IN", 10, 2, "")
	req, err := e.repo.CriarRequisicaoMaterial(e.ctx, &stockentity.RequisicaoMaterial{
		WarehouseID: e.almox, CreatedBy: e.ator, Itens: []stockentity.RequisicaoMaterialItem{{ItemCode: e.item, Quantity: dec("12")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if r := reservadoDe(t, e, e.item, e.almox); !r.Equal(dec("10")) || !req.Itens[0].ReservedQty.Equal(dec("10")) {
		t.Fatalf("reserva até o livre, 10 dos 12 pedidos: %s %+v", r, req.Itens[0])
	}
	if _, err := e.repo.AtenderRequisicaoMaterial(e.ctx, req.ID, []stockentity.AtendimentoLinha{{ItemID: req.Itens[0].ID, Quantity: dec("4")}}, e.ator); err != nil {
		t.Fatalf("a própria reserva não bloqueia a entrega: %v", err)
	}
	if r := reservadoDe(t, e, e.item, e.almox); !r.Equal(dec("6")) {
		t.Fatalf("a entrega consome a reserva: %s", r)
	}
	lida, err := e.repo.ObterRequisicaoMaterial(e.ctx, req.ID)
	if err != nil || !lida.Itens[0].ReservedQty.Equal(dec("6")) || lida.Status != stockentity.RequisicaoParcial {
		t.Fatalf("a linha mostra o que segue reservado: %+v %v", lida, err)
	}
	if err := e.repo.CancelarRequisicaoMaterial(e.ctx, req.ID); err != nil {
		t.Fatal(err)
	}
	if r := reservadoDe(t, e, e.item, e.almox); !r.IsZero() {
		t.Fatalf("cancelar devolve a reserva: %s", r)
	}
	if err := e.repo.CancelarRequisicaoMaterial(e.ctx, req.ID); !ehValidacao(err) {
		t.Fatalf("cancelada não se cancela de novo: %v", err)
	}
}

// Enquanto aguarda a alçada nada fica reservado nem pode ser entregue.
func TestFaseD_RequisicaoSoReservaDepoisDeAprovada(t *testing.T) {
	e := novoAmbienteEstoque(t)
	_ = e.mover(t, e.item, e.almox, "IN", 10, 2, "")
	nova := func() *stockentity.RequisicaoMaterial {
		r, err := e.repo.CriarRequisicaoMaterial(e.ctx, &stockentity.RequisicaoMaterial{
			WarehouseID: e.almox, CreatedBy: e.ator, AguardaAprovacao: true,
			Itens: []stockentity.RequisicaoMaterialItem{{ItemCode: e.item, Quantity: dec("3")}},
		})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	req := nova()
	if req.Status != stockentity.RequisicaoPendente || !req.Valor.Equal(dec("6")) || !reservadoDe(t, e, e.item, e.almox).IsZero() {
		t.Fatalf("pendente, valor 3 × 2, sem reserva: %+v", req)
	}
	if _, err := e.repo.AtenderRequisicaoMaterial(e.ctx, req.ID, []stockentity.AtendimentoLinha{{ItemID: req.Itens[0].ID, Quantity: dec("1")}}, e.ator); !ehValidacao(err) {
		t.Fatalf("pendente não se atende: %v", err)
	}
	for i := 0; i < 2; i++ { // aprovar de novo não reserva em dobro
		if err := e.repo.LiberarRequisicaoAprovada(e.ctx, req.ID); err != nil {
			t.Fatal(err)
		}
	}
	if r := reservadoDe(t, e, e.item, e.almox); !r.Equal(dec("3")) {
		t.Fatalf("aprovada reserva uma vez: %s", r)
	}
	if err := e.repo.ReprovarRequisicaoMaterial(e.ctx, req.ID); err != nil {
		t.Fatal(err)
	}
	lida, _ := e.repo.ObterRequisicaoMaterial(e.ctx, req.ID)
	if lida.Status != stockentity.RequisicaoAberta {
		t.Fatalf("decisão tardia não muda a que já foi aberta: %s", lida.Status)
	}
	outra := nova()
	if err := e.repo.ReprovarRequisicaoMaterial(e.ctx, outra.ID); err != nil {
		t.Fatal(err)
	}
	lida, _ = e.repo.ObterRequisicaoMaterial(e.ctx, outra.ID)
	if lida.Status != stockentity.RequisicaoReprovada {
		t.Fatalf("reprovada: %s", lida.Status)
	}
}

// A ordem de reposição é a requisição de transferência: entrega move entre
// almoxarifados, ao custo da origem, e o pendente conta como a caminho.
func TestFaseD_OrdemDeReposicaoTransfereEConsumaAReserva(t *testing.T) {
	e := novoAmbienteEstoque(t)
	_ = e.mover(t, e.item, e.almox, "IN", 10, 3, "")
	if _, err := e.repo.SalvarPoliticaDeReposicao(e.ctx, stockentity.PoliticaReposicao{
		ItemCode: e.item, WarehouseID: e.livre, Minimo: decp("8"), Maximo: decp("8"), Origem: stockentity.OrigemTransferencia,
		AlmoxOrigem: &e.almox, Ativa: true, CreatedBy: e.ator,
	}); err != nil {
		t.Fatal(err)
	}
	sug, err := e.repo.SugestoesDeReposicao(e.ctx)
	if err != nil || len(sug) != 1 || !sug[0].Sugerido.Equal(dec("8")) || sug[0].Origem != stockentity.OrigemTransferencia {
		t.Fatalf("sugere transferir 8: %+v %v", sug, err)
	}
	req, err := e.repo.CriarRequisicaoMaterial(e.ctx, &stockentity.RequisicaoMaterial{
		WarehouseID: e.almox, DestinoID: &e.livre, CreatedBy: e.ator, Itens: []stockentity.RequisicaoMaterialItem{{ItemCode: e.item, Quantity: dec("8")}},
	})
	if err != nil || req.Kind != stockentity.RequisicaoTransferencia {
		t.Fatalf("%+v %v", req, err)
	}
	if sug, err = e.repo.SugestoesDeReposicao(e.ctx); err != nil || len(sug) != 0 {
		t.Fatalf("o que está a caminho não se pede de novo: %+v %v", sug, err)
	}
	if _, err := e.repo.AtenderRequisicaoMaterial(e.ctx, req.ID, []stockentity.AtendimentoLinha{{ItemID: req.Itens[0].ID, Quantity: dec("5")}}, e.ator); err != nil {
		t.Fatal(err)
	}
	if q, _ := e.saldo(t, e.item, e.livre); q != 5 {
		t.Fatalf("chegaram 5 ao destino: %v", q)
	}
	if q, _ := e.saldo(t, e.item, e.almox); q != 5 {
		t.Fatalf("saíram 5 da origem: %v", q)
	}
	if r := reservadoDe(t, e, e.item, e.almox); !r.Equal(dec("3")) {
		t.Fatalf("seguem 3 reservados na origem: %s", r)
	}
	var custo decimal.Decimal
	if err := testutil.Pool(t).QueryRow(context.Background(),
		`SELECT MAX(avg_cost) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3`, e.empresa, e.item, e.livre).Scan(&custo); err != nil || !custo.Equal(dec("3")) {
		t.Fatalf("o destino recebe ao custo da origem: %s %v", custo, err)
	}
}

// Item que não pode ser comprado nem produzido não entra, e o pedido com destino
// conta só no almoxarifado dele.
func TestFaseD_ReposicaoFiltraItemEOlhaOAlmoxarifado(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	testutil.Exec(t, pool, `UPDATE items SET warehouse_minimum_stock=5, engineering_type=2 WHERE code=$1`, e.item)
	testutil.Exec(t, pool, `UPDATE items SET warehouse_minimum_stock=5, engineering_type=1 WHERE code=$1`, e.outro)
	sug, err := e.repo.SugestoesDeReposicao(e.ctx)
	if err != nil || len(sug) != 1 || sug[0].ItemCode != e.outro || sug[0].Origem != stockentity.OrigemCompra {
		t.Fatalf("item de terceiro fora, comprado dentro: %+v %v", sug, err)
	}
	testutil.Exec(t, pool, `INSERT INTO item_governanca (enterprise_id, item_code, status, motivo) VALUES ($1,$2,'BLOQUEADO','Teste')
		ON CONFLICT (enterprise_id, item_code) DO UPDATE SET status='BLOQUEADO'`, e.empresa, e.outro)
	if sug, err = e.repo.SugestoesDeReposicao(e.ctx); err != nil || len(sug) != 0 {
		t.Fatalf("item bloqueado não se repõe: %+v %v", sug, err)
	}
	testutil.Exec(t, pool, `UPDATE item_governanca SET status='LIBERADO' WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.outro)
	if _, err := e.repo.SalvarPoliticaDeReposicao(e.ctx, stockentity.PoliticaReposicao{
		ItemCode: e.outro, WarehouseID: e.almox, Origem: stockentity.OrigemProducao, Ativa: true, CreatedBy: e.ator,
	}); err == nil {
		t.Fatalf("item comprado não se repõe por produção: %v", err)
	}
	if _, err := e.repo.SalvarPoliticaDeReposicao(e.ctx, stockentity.PoliticaReposicao{
		ItemCode: e.outro, WarehouseID: e.almox, Minimo: decp("4"), Origem: stockentity.OrigemCompra, Ativa: true, CreatedBy: e.ator,
	}); err != nil {
		t.Fatal(err)
	}
	_ = e.mover(t, e.outro, e.livre, "IN", 50, 1, "")
	sug, err = e.repo.SugestoesDeReposicao(e.ctx)
	if err != nil || len(sug) != 1 || *sug[0].WarehouseID != e.almox || !sug[0].Sugerido.Equal(dec("4")) {
		t.Fatalf("o saldo de outro almoxarifado não cobre a política deste: %+v %v", sug, err)
	}
}

// ---------- Fase E: trânsito e onda do pedido ----------

func novoAlmoxDeTransito(t *testing.T, e ambienteEstoque) int64 {
	t.Helper()
	var id int64
	if err := testutil.Pool(t).QueryRow(context.Background(),
		`INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		 VALUES($1,'Em trânsito',$2,'TRANSITO','NORMAL',true,false,$3) RETURNING id`,
		"TR"+strings.ReplaceAll(e.ator.String(), "-", "")[:6]+time.Now().Format("150405.000"), e.ator, e.empresa).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// O material sai da origem e só entra no destino quando recebido: até lá conta
// como a caminho na reposição e não conta para promessa.
func TestFaseE_TransitoEmDuasEtapas(t *testing.T) {
	e := novoAmbienteEstoque(t)
	_ = e.mover(t, e.item, e.almox, "IN", 10, 3, "")
	envio := stockentity.EnvioTransito{ItemCode: e.item, DeAlmox: e.almox, ParaAlmox: e.livre, Quantidade: dec("6"), Ator: e.ator}
	if _, err := e.repo.EnviarParaTransito(e.ctx, envio); !ehValidacao(err) {
		t.Fatalf("sem almoxarifado de trânsito é recusado: %v", err)
	}
	transito := novoAlmoxDeTransito(t, e)
	env, err := e.repo.EnviarParaTransito(e.ctx, envio)
	if err != nil {
		t.Fatal(err)
	}
	if q, _ := e.saldo(t, e.item, e.almox); q != 4 {
		t.Fatalf("saíram 6 da origem: %v", q)
	}
	if q, _ := e.saldo(t, e.item, transito); q != 6 {
		t.Fatalf("estão 6 no trânsito: %v", q)
	}
	if _, err := e.repo.SalvarPoliticaDeReposicao(e.ctx, stockentity.PoliticaReposicao{
		ItemCode: e.item, WarehouseID: e.livre, Minimo: decp("6"), Maximo: decp("6"), Origem: stockentity.OrigemCompra, Ativa: true, CreatedBy: e.ator,
	}); err != nil {
		t.Fatal(err)
	}
	if sug, err := e.repo.SugestoesDeReposicao(e.ctx); err != nil || len(sug) != 0 {
		t.Fatalf("o que está na estrada não se pede de novo: %+v %v", sug, err)
	}
	baixa := func(acao, qtd string) (*stockentity.Transito, error) {
		return e.repo.BaixarTransito(e.ctx, stockentity.BaixaTransito{ID: env.ID, Acao: acao, Quantidade: dec(qtd), Motivo: "Teste", Ator: e.ator})
	}
	if _, err := baixa(stockentity.TransitoReceber, "7"); !ehValidacao(err) {
		t.Fatalf("não se recebe mais que o enviado: %v", err)
	}
	if out, err := baixa(stockentity.TransitoReceber, "3"); err != nil || out.Status != stockentity.TransitoAberto {
		t.Fatalf("recebimento parcial segue aberto: %+v %v", out, err)
	}
	if q, _ := e.saldo(t, e.item, e.livre); q != 3 {
		t.Fatalf("chegaram 3 ao destino: %v", q)
	}
	if _, err := baixa(stockentity.TransitoDevolver, "2"); err != nil {
		t.Fatal(err)
	}
	if q, _ := e.saldo(t, e.item, e.almox); q != 6 {
		t.Fatalf("voltaram 2 à origem: %v", q)
	}
	out, err := baixa(stockentity.TransitoPerda, "1")
	if err != nil || out.Status != stockentity.TransitoEncerrado || !out.Pendente().IsZero() {
		t.Fatalf("a perda do resto encerra o envio: %+v %v", out, err)
	}
	if q, _ := e.saldo(t, e.item, transito); q != 0 {
		t.Fatalf("nada fica no trânsito: %v", q)
	}
	if _, err := baixa(stockentity.TransitoReceber, "1"); !ehValidacao(err) {
		t.Fatalf("envio encerrado não se baixa: %v", err)
	}
	var mov int64
	if err := testutil.Pool(t).QueryRow(context.Background(),
		`SELECT id FROM stock_movements WHERE enterprise_id=$1 AND reference_type='TRANSITO' AND reference_code=$2 LIMIT 1`, e.empresa, env.Code).Scan(&mov); err != nil {
		t.Fatal(err)
	}
	if _, err := e.repo.EstornarMovimento(e.ctx, mov, e.ator, "teste"); !ehValidacao(err) {
		t.Fatalf("movimento do envio não se estorna solto: %v", err)
	}
}

// A entrega de ordem de reposição pode ir pelo trânsito: a reserva é consumida no
// envio e a linha fica entregue.
func TestFaseE_RequisicaoViaTransito(t *testing.T) {
	e := novoAmbienteEstoque(t)
	_ = e.mover(t, e.item, e.almox, "IN", 10, 3, "")
	transito := novoAlmoxDeTransito(t, e)
	req, err := e.repo.CriarRequisicaoMaterial(e.ctx, &stockentity.RequisicaoMaterial{
		WarehouseID: e.almox, DestinoID: &e.livre, CreatedBy: e.ator, Itens: []stockentity.RequisicaoMaterialItem{{ItemCode: e.item, Quantity: dec("4")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	r, err := e.repo.AtenderRequisicaoMaterial(e.ctx, req.ID, []stockentity.AtendimentoLinha{{ItemID: req.Itens[0].ID, Quantity: dec("4"), ViaTransito: true}}, e.ator)
	if err != nil || r.Status != stockentity.RequisicaoAtendida {
		t.Fatalf("%+v %v", r, err)
	}
	if q, _ := e.saldo(t, e.item, transito); q != 4 {
		t.Fatalf("foram 4 para o trânsito: %v", q)
	}
	if q, _ := e.saldo(t, e.item, e.livre); q != 0 {
		t.Fatalf("o destino só recebe na chegada: %v", q)
	}
	if res := reservadoDe(t, e, e.item, e.almox); !res.IsZero() {
		t.Fatalf("a reserva foi consumida: %s", res)
	}
	lista, err := e.repo.ListarTransitos(e.ctx, true)
	if err != nil || len(lista) != 1 || lista[0].RequisicaoID == nil || *lista[0].RequisicaoID != req.ID {
		t.Fatalf("o envio guarda a requisição: %+v %v", lista, err)
	}
}

func novoPedido(t *testing.T, e ambienteEstoque) int64 {
	t.Helper()
	pool := testutil.Pool(t)
	var pedido int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO sales_orders (order_number, enterprise_code, status, created_by)
		 SELECT 1, en.code, 'R', $2 FROM enterprise en WHERE en.id=$1 RETURNING code`, e.empresa, e.ator).Scan(&pedido); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM sales_orders WHERE code=$1`, pedido) })
	return pedido
}

// A onda do pedido separa o que o próprio pedido reservou e não baixa o estoque:
// a saída é da nota. Antes a confirmação baixava, e a nota baixaria de novo.
func TestFaseE_OndaDoPedidoNaoBaixaEDevolveAReserva(t *testing.T) {
	e := novoAmbienteEstoque(t)
	_ = e.mover(t, e.item, e.almox, "IN", 10, 3, "")
	pedido := novoPedido(t, e)
	if _, err := e.repo.CreateReservation(e.ctx, &stockentity.StockReservation{
		ItemCode: e.item, WarehouseID: e.almox, Quantity: dec("10"), ReferenceType: stockentity.ReferenceTypeSalesOrder,
		ReferenceCode: pedido, ReservationDate: time.Now(), Status: "ACTIVE", CreatedBy: e.ator,
	}); err != nil {
		t.Fatal(err)
	}
	ref := stockentity.ReferenceTypeSalesOrder
	nec := []stockentity.NecessidadeOnda{{ItemCode: e.item, Quantity: dec("6"), ReferenceType: &ref, ReferenceCode: &pedido}}
	onda, err := e.repo.CriarOndaDeSeparacao(e.ctx, 0, e.almox, "FEFO", nec, e.ator)
	if err != nil || onda.Code <= 0 || len(onda.EmFalta) != 0 || len(onda.Linhas) != 1 {
		t.Fatalf("a onda enxerga o saldo reservado para o pedido: %+v %v", onda, err)
	}
	if r := reservadoDe(t, e, e.item, e.almox); !r.Equal(dec("10")) {
		t.Fatalf("o total reservado não muda: %s", r)
	}
	out, err := e.repo.ConfirmarOndaDeSeparacao(e.ctx, onda.Code, map[int64]decimal.Decimal{onda.Linhas[0].ID: dec("4")}, e.ator)
	if err != nil || out.Status != "SEPARADA" || out.Linhas[0].PickedQty != 4 {
		t.Fatalf("separação parcial: %+v %v", out, err)
	}
	if q, _ := e.saldo(t, e.item, e.almox); q != 10 {
		t.Fatalf("a onda do pedido não baixa: %v", q)
	}
	if r := reservadoDe(t, e, e.item, e.almox); !r.Equal(dec("10")) {
		t.Fatalf("a reserva volta inteira ao pedido, com a falta: %s", r)
	}
	var doPedido decimal.Decimal
	if err := testutil.Pool(t).QueryRow(context.Background(),
		`SELECT COALESCE(SUM(quantity),0) FROM stock_reservations WHERE enterprise_id=$1 AND reference_type='SALES_ORDER' AND reference_code=$2 AND status='ACTIVE'`,
		e.empresa, pedido).Scan(&doPedido); err != nil || !doPedido.Equal(dec("10")) {
		t.Fatalf("a nota acha a reserva do pedido: %s %v", doPedido, err)
	}
	cancelar, err := e.repo.CriarOndaDeSeparacao(e.ctx, 0, e.almox, "FEFO", nec, e.ator)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.repo.CancelarOndaDeSeparacao(e.ctx, cancelar.Code, e.ator); err != nil {
		t.Fatal(err)
	}
	if r := reservadoDe(t, e, e.item, e.almox); !r.Equal(dec("10")) {
		t.Fatalf("cancelar devolve a reserva ao pedido: %s", r)
	}
	testutil.Exec(t, testutil.Pool(t), `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,mask,requested_qty,warehouse_code)
		VALUES($1,1,$2,'',10,$3)`, pedido, e.item, e.almox)
	falta, err := e.repo.LinhasDePedidoParaOnda(e.ctx, pedido, e.almox)
	if err != nil || len(falta) != 1 || !falta[0].Quantity.Equal(dec("6")) {
		t.Fatalf("pediu 10 e separou 4: faltam 6. %+v %v", falta, err)
	}
	lista, err := e.repo.ListarOndas(e.ctx, "SEPARADA")
	if err != nil || len(lista) != 1 || len(lista[0].Linhas) != 1 {
		t.Fatalf("a onda fica na lista: %+v %v", lista, err)
	}
}

// Onda sem pedido continua baixando, agora só o separado de fato.
func TestFaseE_SeparacaoParcialBaixaSoOSeparado(t *testing.T) {
	e := novoAmbienteEstoque(t)
	_ = e.mover(t, e.item, e.almox, "IN", 10, 3, "")
	onda, err := e.repo.CriarOndaDeSeparacao(e.ctx, 0, e.almox, "FEFO", []stockentity.NecessidadeOnda{{ItemCode: e.item, Quantity: dec("5")}}, e.ator)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.repo.ConfirmarOndaDeSeparacao(e.ctx, onda.Code, map[int64]decimal.Decimal{onda.Linhas[0].ID: dec("9")}, e.ator); !ehValidacao(err) {
		t.Fatalf("não se separa mais que a linha: %v", err)
	}
	if _, err := e.repo.ConfirmarOndaDeSeparacao(e.ctx, onda.Code, map[int64]decimal.Decimal{onda.Linhas[0].ID: dec("2")}, e.ator); err != nil {
		t.Fatal(err)
	}
	if q, _ := e.saldo(t, e.item, e.almox); q != 8 {
		t.Fatalf("baixaram só 2: %v", q)
	}
	if r := reservadoDe(t, e, e.item, e.almox); !r.IsZero() {
		t.Fatalf("o resto volta a ficar livre: %s", r)
	}
}

// O coletor lê endereço, código do item, código de barras do fornecedor, EAN das
// notas, GS1 e lote. Código de outra empresa não aparece.
func TestFaseE_LeituraDoColetor(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	testutil.Exec(t, pool, `INSERT INTO manufacturing_warehouse_addresses(enterprise_id,warehouse_id,address) VALUES($1,$2,'R1-P2')`, e.empresa, e.almox)
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM manufacturing_warehouse_addresses WHERE enterprise_id=$1`, e.empresa)
	})
	testutil.Exec(t, pool, `UPDATE items SET business_code='COL-'||code WHERE code=$1`, e.item)
	var codigo string
	if err := pool.QueryRow(context.Background(), `SELECT business_code FROM items WHERE code=$1`, e.item).Scan(&codigo); err != nil {
		t.Fatal(err)
	}
	l, err := e.repo.ResolverLeitura(e.ctx, "R1-P2", &e.almox)
	if err != nil || l.Tipo != stockentity.LeituraEndereco || *l.WarehouseID != e.almox {
		t.Fatalf("endereço: %+v %v", l, err)
	}
	if l, err = e.repo.ResolverLeitura(e.ctx, codigo+"\r\n", &e.almox); err != nil || *l.ItemCode != e.item {
		t.Fatalf("item, com o Enter do leitor: %+v %v", l, err)
	}
	testutil.Exec(t, pool, `INSERT INTO stock_lots(enterprise_id,item_code,lot,created_by) VALUES($1,$2,'L-COL',$3)`, e.empresa, e.outro, e.ator)
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM stock_lots WHERE enterprise_id=$1`, e.empresa) })
	if l, err = e.repo.ResolverLeitura(e.ctx, "L-COL", nil); err != nil || *l.ItemCode != e.outro || l.Lote != "L-COL" {
		t.Fatalf("lote leva ao item: %+v %v", l, err)
	}
	if _, err = e.repo.ResolverLeitura(e.ctx, "NADA-DISSO", nil); err == nil {
		t.Fatal("código desconhecido é recusado")
	}
}

// ---------- Fase F: análise ----------

// Ajuste de inventário não renova o item: o estoque parado continua parado.
func TestFaseF_AjusteNaoEscondeItemParado(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	_ = e.mover(t, e.item, e.almox, "IN", 10, 3, "")
	testutil.Exec(t, pool, `UPDATE stock_movements SET movement_date = CURRENT_DATE - 200, created_at = NOW() - INTERVAL '200 days' WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item)
	if _, err := e.repo.TransferirEntreAlmoxarifados(e.ctx, stockentity.TransferenciaEstoque{ItemCode: e.item, DeAlmox: e.almox, ParaAlmox: e.livre, Quantidade: dec("2"), Ator: e.ator}); err != nil {
		t.Fatal(err)
	}
	an, err := e.repo.AnaliseDeEstoque(e.ctx, 6, 90)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range an {
		if a.ItemCode == e.item && (!a.Parado || a.DiasSemMovimento == nil || *a.DiasSemMovimento < 199) {
			t.Fatalf("transferir não tira o item de parado: %+v", a)
		}
	}
}

// Inventário fechado e contagem cíclica entram na acuracidade; aberto e de outro
// almoxarifado, não.
func TestFaseF_ContagensDaAcuracidade(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	inventario := func(status string, almox int64, sistema, contado string) {
		var id int64
		if err := pool.QueryRow(context.Background(),
			`INSERT INTO physical_inventories(code,description,warehouse_id,start_date,status,created_by,enterprise_id)
			 VALUES($1,'Teste',$2,$3::date,$4,$5,$6) RETURNING id`, testutil.UniqueCode(), almox, time.Now(), status, e.ator, e.empresa).Scan(&id); err != nil {
			t.Fatal(err)
		}
		testutil.Exec(t, pool, `INSERT INTO physical_inventory_items(inventory_id,item_code,warehouse_id,system_qty,counted_qty,unit_cost,address)
			VALUES($1,$2,$3,$4,$5,10,'A1')`, id, e.item, almox, dec(sistema), dec(contado))
	}
	inventario("CLOSED", e.almox, "10", "8")
	inventario("OPEN", e.almox, "10", "1")
	inventario("CLOSED", e.livre, "10", "10")
	lista, err := e.repo.ContagensParaAcuracidade(e.ctx, 12, &e.almox)
	if err != nil || len(lista) != 1 || !lista[0].Contado.Equal(dec("8")) || lista[0].Mes == "" {
		t.Fatalf("só o fechado deste almoxarifado: %+v %v", lista, err)
	}
	if lista, err = e.repo.ContagensParaAcuracidade(e.ctx, 12, nil); err != nil || len(lista) != 2 {
		t.Fatalf("todos os almoxarifados: %+v %v", lista, err)
	}
}

// A reserva da requisição é do item, não do endereço: entregar de um endereço não
// pode esbarrar na parte da reserva que caiu nele.
func TestRevisao_EntregaDeEnderecoComReservaRepartida(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	for _, a := range []string{"A1", "A2"} {
		testutil.Exec(t, pool, `INSERT INTO manufacturing_warehouse_addresses(enterprise_id,warehouse_id,address) VALUES($1,$2,$3)`, e.empresa, e.almox, a)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM manufacturing_warehouse_addresses WHERE enterprise_id=$1`, e.empresa)
	})
	_ = e.mover(t, e.item, e.almox, "IN", 5, 2, "A1")
	_ = e.mover(t, e.item, e.almox, "IN", 5, 2, "A2")
	req, err := e.repo.CriarRequisicaoMaterial(e.ctx, &stockentity.RequisicaoMaterial{
		WarehouseID: e.almox, CreatedBy: e.ator, Itens: []stockentity.RequisicaoMaterialItem{{ItemCode: e.item, Quantity: dec("10")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	a2 := "A2"
	if _, err := e.repo.AtenderRequisicaoMaterial(e.ctx, req.ID, []stockentity.AtendimentoLinha{{ItemID: req.Itens[0].ID, Quantity: dec("5"), Address: &a2}}, e.ator); err != nil {
		t.Fatalf("entregar de A2 com a reserva repartida: %v", err)
	}
	var q, res decimal.Decimal
	if err := pool.QueryRow(context.Background(), `SELECT SUM(quantity), SUM(reserved_qty) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item).Scan(&q, &res); err != nil {
		t.Fatal(err)
	}
	if !q.Equal(dec("5")) || !res.Equal(dec("5")) {
		t.Fatalf("sobram 5 com 5 reservados: %s %s", q, res)
	}
	var negativo int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND reserved_qty > quantity`, e.empresa, e.item).Scan(&negativo); err != nil || negativo != 0 {
		t.Fatalf("reserva maior que o saldo em %d endereço(s): %v", negativo, err)
	}
}

// Entrega errada se desfaz: o material volta e a linha fica pendente de novo.
func TestRevisao_EstornoDaEntregaDeRequisicao(t *testing.T) {
	e := novoAmbienteEstoque(t)
	_ = e.mover(t, e.item, e.almox, "IN", 10, 2, "")
	req, err := e.repo.CriarRequisicaoMaterial(e.ctx, &stockentity.RequisicaoMaterial{
		WarehouseID: e.almox, CreatedBy: e.ator, Itens: []stockentity.RequisicaoMaterialItem{{ItemCode: e.item, Quantity: dec("4")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.repo.AtenderRequisicaoMaterial(e.ctx, req.ID, []stockentity.AtendimentoLinha{{ItemID: req.Itens[0].ID, Quantity: dec("4")}}, e.ator); err != nil {
		t.Fatal(err)
	}
	var mov int64
	if err := testutil.Pool(t).QueryRow(context.Background(),
		`SELECT id FROM stock_movements WHERE enterprise_id=$1 AND reference_type='REQUISICAO_MATERIAL' AND reference_code=$2`, e.empresa, req.Code).Scan(&mov); err != nil {
		t.Fatal(err)
	}
	if _, err := e.repo.EstornarMovimento(e.ctx, mov, e.ator, "Entrega errada"); err != nil {
		t.Fatal(err)
	}
	if q, _ := e.saldo(t, e.item, e.almox); q != 10 {
		t.Fatalf("o material volta: %v", q)
	}
	lida, err := e.repo.ObterRequisicaoMaterial(e.ctx, req.ID)
	if err != nil || lida.Status != stockentity.RequisicaoAberta || !lida.Itens[0].IssuedQty.IsZero() {
		t.Fatalf("a requisição reabre: %+v %v", lida, err)
	}
	igual(t, "a linha volta a ter reserva", lida.Itens[0].ReservedQty, "4")
	igual(t, "reservado no saldo", e.somar(t, `SELECT SUM(reserved_qty) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item), "4")
}

// Estorno de entrega parcial soma na reserva que já existe: duas reservas na
// mesma linha deixariam a segunda presa depois da nova entrega.
func TestRevisao_EstornoParcialMantemUmaReservaPorLinha(t *testing.T) {
	e := novoAmbienteEstoque(t)
	_ = e.mover(t, e.item, e.almox, "IN", 10, 2, "")
	req, err := e.repo.CriarRequisicaoMaterial(e.ctx, &stockentity.RequisicaoMaterial{
		WarehouseID: e.almox, CreatedBy: e.ator, Itens: []stockentity.RequisicaoMaterialItem{{ItemCode: e.item, Quantity: dec("10")}},
	})
	exigir(t, err)
	linha := req.Itens[0].ID
	_, err = e.repo.AtenderRequisicaoMaterial(e.ctx, req.ID, []stockentity.AtendimentoLinha{{ItemID: linha, Quantity: dec("4")}}, e.ator)
	exigir(t, err)
	var mov int64
	exigir(t, testutil.Pool(t).QueryRow(context.Background(),
		`SELECT id FROM stock_movements WHERE enterprise_id=$1 AND reference_type='REQUISICAO_MATERIAL' AND reference_code=$2`, e.empresa, req.Code).Scan(&mov))
	_, err = e.repo.EstornarMovimento(e.ctx, mov, e.ator, "Entrega errada")
	exigir(t, err)
	igual(t, "reservas ativas da linha", e.somar(t, `SELECT COUNT(*) FROM stock_reservations WHERE enterprise_id=$1 AND reference_item_code=$2 AND status='ACTIVE'`, e.empresa, linha), "1")
	_, err = e.repo.AtenderRequisicaoMaterial(e.ctx, req.ID, []stockentity.AtendimentoLinha{{ItemID: linha, Quantity: dec("10")}}, e.ator)
	exigir(t, err)
	igual(t, "nada fica reservado", e.somar(t, `SELECT SUM(reserved_qty) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item), "0")
}

// Sem o lote, só anda o que está fora de lote: senão o lote fica no endereço
// antigo e o material no novo.
func TestRevisao_TransferenciaEntreEnderecosNaoAbandonaOLote(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 5, 4, "A1", "L1"))
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 2, 4, "A1", ""))
	mov := func(q float64, lote string) error {
		m := &stockentity.StockMovement{ItemCode: e.item, WarehouseID: e.almox, Quantity: decimal.NewFromFloat(q), CreatedBy: e.ator}
		if lote != "" {
			m.Lot = &lote
		}
		_, err := e.repo.TransferirEntreEnderecos(e.ctx, m, "A1", "A2")
		return err
	}
	if err := mov(3, ""); err == nil || !strings.Contains(err.Error(), "informe o lote") {
		t.Fatalf("3 sem lote com só 2 fora de lote: %v", err)
	}
	exigir(t, mov(2, ""))
	exigir(t, mov(1, "L1"))
	igual(t, "lote L1 em A1", e.somar(t, `SELECT quantity FROM stock_lot_balances WHERE enterprise_id=$1 AND item_code=$2 AND address='A1' AND lot='L1'`, e.empresa, e.item), "4")
	igual(t, "saldo A1", e.saldoEndereco(t, e.item, e.almox, "A1"), "4")
	igual(t, "saldo A2", e.saldoEndereco(t, e.item, e.almox, "A2"), "3")
}

// A transferência não recalcula custo: o material é o mesmo, e o médio é um só no
// almoxarifado.
func TestRevisao_TransferenciaEntreEnderecosMantemOCustoUnico(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 3, 1, "A1", ""))
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 7, 2, "A2", ""))
	if _, err := e.repo.TransferirEntreEnderecos(e.ctx, &stockentity.StockMovement{ItemCode: e.item, WarehouseID: e.almox, Quantity: dec("1"), CreatedBy: e.ator}, "A1", "A2"); err != nil {
		t.Fatal(err)
	}
	igual(t, "custos distintos", e.somar(t, `SELECT COUNT(DISTINCT avg_cost) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item), "1")
	igual(t, "valor do almoxarifado", e.somar(t, `SELECT ROUND(SUM(total_cost),2) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item), "17")
}

// Sem saldo em lote no almoxarifado, o lote da saída é referência do documento.
func TestRevisao_SaidaComLoteDeReferenciaNaoNegativaOLote(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 4, "", ""))
	exigir(t, e.moverLote(t, e.item, e.almox, "OUT", 3, 0, "", "NF-123"))
	igual(t, "saldo", e.saldoEndereco(t, e.item, e.almox, ""), "7")
	igual(t, "linhas de lote", e.somar(t, `SELECT COUNT(*) FROM stock_lot_balances WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item), "0")
}

// Duas pessoas no mesmo item, em sentidos opostos: nenhuma pode levar "erro
// interno". Antes o banco derrubava uma das duas por deadlock.
func TestRevisao_OperacoesSimultaneasNoMesmoItem(t *testing.T) {
	e := novoAmbienteEstoque(t)
	_ = e.mover(t, e.item, e.almox, "IN", 500, 2, "")
	_ = e.mover(t, e.item, e.livre, "IN", 500, 2, "")
	var wg sync.WaitGroup
	erros := make(chan error, 80)
	for i := 0; i < 40; i++ {
		wg.Add(2)
		de, para := e.almox, e.livre
		if i%2 == 1 {
			de, para = para, de
		}
		go func() {
			defer wg.Done()
			_, err := e.repo.TransferirEntreAlmoxarifados(e.ctx, stockentity.TransferenciaEstoque{ItemCode: e.item, DeAlmox: de, ParaAlmox: para, Quantidade: dec("1"), Ator: e.ator})
			erros <- err
		}()
		go func() {
			defer wg.Done()
			r, err := e.reservar(t, e.item, de, 1)
			if err == nil {
				err = e.repo.CancelReservation(e.ctx, r.ID)
			}
			erros <- err
		}()
	}
	wg.Wait()
	close(erros)
	for err := range erros {
		if err != nil {
			t.Errorf("operação simultânea: %v", err)
		}
	}
	igual(t, "total do item", e.somar(t, `SELECT SUM(quantity) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item), "1000")
	igual(t, "reservado", e.somar(t, `SELECT SUM(reserved_qty) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.item), "0")
}

// Numeração de requisição sem colisão entre duas pessoas abrindo ao mesmo tempo.
func TestRevisao_RequisicoesSimultaneasNaoColidemNoNumero(t *testing.T) {
	e := novoAmbienteEstoque(t)
	_ = e.mover(t, e.item, e.almox, "IN", 100, 2, "")
	var wg sync.WaitGroup
	erros := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.repo.CriarRequisicaoMaterial(e.ctx, &stockentity.RequisicaoMaterial{
				WarehouseID: e.almox, CreatedBy: e.ator, Itens: []stockentity.RequisicaoMaterialItem{{ItemCode: e.item, Quantity: dec("1")}},
			})
			erros <- err
		}()
	}
	wg.Wait()
	close(erros)
	for err := range erros {
		if err != nil {
			t.Errorf("requisição simultânea: %v", err)
		}
	}
}

// O ajuste de contagem é sobre a foto: se o lote saiu depois dela, a diferença não
// pode deixar o lote negativo.
func TestRevisao_AjusteDeContagemNaoNegativaOLote(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 5, 4, "A1", "L1"))
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 5, 4, "A1", ""))
	lote, endereco, ref := "L1", "A1", stockentity.ReferenceTypeInventory
	if _, err := e.repo.CreateMovement(e.ctx, &stockentity.StockMovement{ItemCode: e.item, WarehouseID: e.almox, MovementType: stockentity.MovementTypeAdjustment,
		Quantity: dec("-7"), Lot: &lote, Address: &endereco, ReferenceType: &ref, CreatedBy: e.ator, PermiteSaldoNegativo: true}); err != nil {
		t.Fatal(err)
	}
	igual(t, "lote", e.saldoLote(t, e.item, e.almox, "L1"), "0")
	igual(t, "endereço", e.saldoEndereco(t, e.item, e.almox, "A1"), "3")
}

// Parâmetros da empresa: o limite do retroativo vale para o que se digita.
func TestPendencias_LimiteDoRetroativo(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM stock_parameters WHERE enterprise_id=$1`, e.empresa) })
	zero := 0
	exigir(t, e.repo.SalvarParametrosDeEstoque(e.ctx, stockentity.ParametrosEstoque{ExpiredLotAction: stockentity.LoteVencidoBloqueia, RetroactiveMonths: &zero}, e.ator))
	antes := time.Now().AddDate(0, -2, 0)
	_, err := e.repo.CreateMovement(e.ctx, &stockentity.StockMovement{ItemCode: e.item, WarehouseID: e.almox, MovementType: "IN", Quantity: dec("1"), UnitPrice: dec("1"), CreatedBy: e.ator, MovementDate: &antes})
	if err == nil || !strings.Contains(err.Error(), "aceita lançamento") {
		t.Fatalf("data de dois meses atrás com limite zero: %v", err)
	}
	_ = e.mover(t, e.item, e.almox, "IN", 1, 1, "")
}

func TestPendencias_DesligarLoteComSaldoERecusado(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.repo.SalvarControleDoItem(e.ctx, stockentity.ControleDoItem{ItemCode: e.item, ControlsLot: true, LotIssueRule: stockentity.BaixaFEFO}))
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 5, 1, "", "L1"))
	if err := e.repo.SalvarControleDoItem(e.ctx, stockentity.ControleDoItem{ItemCode: e.item, LotIssueRule: stockentity.BaixaFEFO}); err == nil {
		t.Fatal("desligar o lote com saldo em lote")
	}
}

func TestPendencias_KanbanEAcuracidadeDosParametros(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM kanban_cards WHERE enterprise_id=$1`, e.empresa) })
	c, err := e.repo.SalvarCartaoKanban(e.ctx, stockentity.CartaoKanban{ItemCode: e.item, WarehouseID: &e.almox, CardCount: 2, QuantityPerCard: dec("10"), ReorderPoint: dec("5"), Status: "ACTIVE"}, e.ator)
	exigir(t, err)
	_ = e.mover(t, e.item, e.almox, "IN", 7, 2, "")
	igual(t, "saldo no cartão", c.Saldo, "0")
	lista, err := e.repo.ListarCartoesKanban(e.ctx)
	exigir(t, err)
	if len(lista) != 1 || !lista[0].Saldo.Equal(dec("7")) {
		t.Fatalf("cartões: %+v", lista)
	}
}

// A idade conta do recebimento, e o nível sugerido vem do consumo real.
func TestPendencias_IdadeDoLoteENiveis(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 100, 2, "", "L1"))
	_ = e.mover(t, e.item, e.almox, "OUT", 30, 0, "")
	lotes, err := e.repo.IdadeDosLotes(e.ctx, nil)
	exigir(t, err)
	if len(lotes) != 1 || !lotes[0].Quantidade.Equal(dec("70")) || lotes[0].Dias == nil {
		t.Fatalf("lotes: %+v", lotes)
	}
	n, err := e.repo.NiveisPeloConsumo(e.ctx, e.item, "", nil, 1, 10, 5)
	exigir(t, err)
	if !n.ConsumoDiario.IsPositive() || !n.Minimo.GreaterThan(n.Seguranca) || !n.Maximo.GreaterThan(n.Minimo) {
		t.Fatalf("níveis: %+v", n)
	}
}

// A provisão entra pela regra da empresa e a demanda aberta vira falta.
func TestPendencias_ProvisaoDeObsoletos(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM stock_parameters WHERE enterprise_id=$1`, e.empresa) })
	dias, pct := 1, dec("50")
	exigir(t, e.repo.SalvarParametrosDeEstoque(e.ctx, stockentity.ParametrosEstoque{ExpiredLotAction: stockentity.LoteVencidoBloqueia, ObsoleteDays: &dias, ObsoleteProvisionPct: &pct}, e.ator))
	lanc := time.Now().AddDate(0, 0, -20)
	_, err := e.repo.CreateMovement(e.ctx, &stockentity.StockMovement{ItemCode: e.item, WarehouseID: e.almox, MovementType: "IN", Quantity: dec("10"), UnitPrice: dec("4"), CreatedBy: e.ator, MovementDate: &lanc})
	exigir(t, err)
	out, err := e.repo.AnaliseDeEstoque(e.ctx, 6, 5)
	exigir(t, err)
	var achou bool
	for _, a := range out {
		if a.ItemCode == e.item {
			achou = true
			igual(t, "provisão", a.Provisao, "20")
		}
	}
	if !achou {
		t.Fatal("item fora da análise")
	}
}

// A nota do pedido sai no lote e no endereço separados pela onda.
func TestPendencias_SeparacaoDaOndaVaiParaANota(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 5, 2, "A1", "L1"))
	var onda, linha int64
	exigir(t, pool.QueryRow(context.Background(), `INSERT INTO stock_picking_waves(enterprise_id,code,warehouse_id,status,rule,created_by) VALUES($1,9001,$2,'SEPARADA','FEFO',$3) RETURNING id`, e.empresa, e.almox, e.ator).Scan(&onda))
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM stock_picking_wave_lines WHERE wave_id=$1`, onda)
		testutil.Exec(t, pool, `DELETE FROM stock_picking_waves WHERE id=$1`, onda)
	})
	exigir(t, pool.QueryRow(context.Background(), `INSERT INTO stock_picking_wave_lines(enterprise_id,wave_id,item_code,mask,lot,address,quantity,picked_qty,reference_type,reference_code)
		VALUES($1,$2,$3,'','L1','A1',4,4,'SALES_ORDER',777) RETURNING id`, e.empresa, onda, e.item).Scan(&linha))
	sep, err := e.repo.SeparadoParaFaturar(e.ctx, 777, e.item, "", e.almox)
	exigir(t, err)
	if len(sep) != 1 || sep[0].Lote != "L1" || !sep[0].Quantidade.Equal(dec("4")) {
		t.Fatalf("separado: %+v", sep)
	}
	lote, end := "L1", "A1"
	_, err = e.repo.CreateMovement(e.ctx, &stockentity.StockMovement{ItemCode: e.item, WarehouseID: e.almox, MovementType: "OUT", Quantity: dec("4"), Lot: &lote, Address: &end, LinhaDaOnda: &linha, CreatedBy: e.ator})
	exigir(t, err)
	sep, err = e.repo.SeparadoParaFaturar(e.ctx, 777, e.item, "", e.almox)
	exigir(t, err)
	if len(sep) != 0 {
		t.Fatalf("a separação foi toda faturada: %+v", sep)
	}
}

// O retroativo no componente chega ao produto: a entrada da ordem acompanha o novo
// custo do consumo, e a diferença não vai para o resultado.
func TestPendencias_RecalculoEmNiveis(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseC(t)
	inicio, fim := mesPassado()
	ordem := int64(990001)
	ref := stockentity.ReferenceTypeProductionOrder
	mov := func(item int64, tipo string, dia int, qtd, preco string, codigo *int64) {
		data := inicio.AddDate(0, 0, dia)
		m := &stockentity.StockMovement{ItemCode: item, WarehouseID: e.almox, MovementType: tipo, Quantity: dec(qtd), UnitPrice: dec(preco), MovementDate: &data, CreatedBy: e.ator}
		if codigo != nil {
			m.ReferenceType, m.ReferenceCode = &ref, codigo
		}
		_, err := e.repo.CreateMovement(e.ctx, m)
		exigir(t, err)
	}
	mov(e.item, "IN", 0, "10", "2", nil)
	mov(e.item, "OUT", 4, "5", "0", &ordem)
	mov(e.outro, "EP", 5, "5", "2", &ordem)
	// Nota esquecida, lançada depois com a data de antes do consumo.
	mov(e.item, "IN", 1, "10", "4", nil)
	_, err := e.repo.FecharMes(e.ctx, fim, e.ator, false)
	exigir(t, err)
	igual(t, "custo do produto", e.somar(t, `SELECT unit_price FROM stock_movements WHERE enterprise_id=$1 AND item_code=$2 AND movement_type='EP'`, e.empresa, e.outro), "3")
	igual(t, "saldo do produto", e.somar(t, `SELECT avg_cost FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, e.empresa, e.outro), "3")
	igual(t, "diferença no resultado", e.somar(t, `SELECT COALESCE(SUM(value),0) FROM stock_cost_variances WHERE enterprise_id=$1 AND kind='RECALCULO'`, e.empresa), "0")
}

// Desligar o lote de item com validade é recusado: a validade é do lote.
func TestFaseB_DesligarLoteComValidadeERecusado(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseB(t)
	pool := testutil.Pool(t)
	exigir(t, e.repo.SalvarControleDoItem(e.ctx, stockentity.ControleDoItem{ItemCode: e.item, ControlsLot: true, LotIssueRule: "FEFO"}))
	testutil.Exec(t, pool, `UPDATE items SET warehouse_shelf_life_days=90 WHERE enterprise_id=$1 AND code=$2`, e.empresa, e.item)
	if err := e.repo.SalvarControleDoItem(e.ctx, stockentity.ControleDoItem{ItemCode: e.item, ControlsLot: false, LotIssueRule: "FEFO"}); !ehValidacao(err) {
		t.Fatalf("desligar o lote com validade: esperava recusa, veio %v", err)
	}
	testutil.Exec(t, pool, `UPDATE items SET warehouse_shelf_life_days=NULL WHERE enterprise_id=$1 AND code=$2`, e.empresa, e.item)
	exigir(t, e.repo.SalvarControleDoItem(e.ctx, stockentity.ControleDoItem{ItemCode: e.item, ControlsLot: false, LotIssueRule: "FEFO"}))
}

// O MRP usa este cálculo no modo por dia trabalhado: consumo por dia × dias que a
// empresa trabalha no prazo (calendário industrial) + segurança.
func TestPontoDePedidoDoItemPorDiaTrabalhado(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	testutil.Exec(t, pool, `UPDATE items SET planning_reorder_point='{"TR":5,"CM":10,"CR":1,"ES":20,"modo":"DIA_TRABALHADO"}'::jsonb WHERE enterprise_id=$1 AND code=$2`, e.empresa, e.item)
	hoje, err := e.repo.HojeDaEmpresa(e.ctx)
	exigir(t, err)
	quer := stockentity.PontoPorDiaTrabalhado(decimal.NewFromInt(10), stockentity.CalendarioDeTrabalho{}.DiasDeConsumoNoPrazo(hoje, 5), decimal.NewFromInt(20))
	ponto, ok, err := e.repo.PontoDePedidoDoItem(e.ctx, e.item)
	exigir(t, err)
	if !ok || !ponto.Equal(quer) {
		t.Fatalf("ponto %s (ok=%v), queria %s", ponto, ok, quer)
	}
	if _, ok, _ := e.repo.PontoDePedidoDoItem(e.ctx, e.outro); ok {
		t.Fatal("item sem ponto de pedido não devia ter ponto")
	}
}

// A onda solta a reserva do pedido por produto; na volta ela é repartida pelas
// linhas do produto (gravava o código do produto, e cancelar uma das linhas não
// achava a reserva dela).
func TestOndaDevolveAReservaParaAsLinhasDoPedido(t *testing.T) {
	e := novoAmbienteEstoque(t)
	_ = e.mover(t, e.item, e.almox, "IN", 10, 3, "")
	pedido := novoPedido(t, e)
	pool := testutil.Pool(t)
	var l1, l2 int64
	if err := pool.QueryRow(context.Background(), `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,mask,requested_qty,warehouse_code)
		VALUES($1,1,$2,'',6,$3) RETURNING code`, pedido, e.item, e.almox).Scan(&l1); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,mask,requested_qty,warehouse_code)
		VALUES($1,2,$2,'',4,$3) RETURNING code`, pedido, e.item, e.almox).Scan(&l2); err != nil {
		t.Fatal(err)
	}
	for _, l := range []struct {
		code int64
		qtd  string
	}{{l1, "6"}, {l2, "4"}} {
		linha := l.code
		if _, err := e.repo.CreateReservation(e.ctx, &stockentity.StockReservation{
			ItemCode: e.item, WarehouseID: e.almox, Quantity: dec(l.qtd), ReferenceType: stockentity.ReferenceTypeSalesOrder,
			ReferenceCode: pedido, ReferenceItemCode: &linha, ReservationDate: time.Now(), Status: "ACTIVE", CreatedBy: e.ator,
		}); err != nil {
			t.Fatal(err)
		}
	}
	ref := stockentity.ReferenceTypeSalesOrder
	nec := []stockentity.NecessidadeOnda{{ItemCode: e.item, Quantity: dec("10"), ReferenceType: &ref, ReferenceCode: &pedido}}
	onda, err := e.repo.CriarOndaDeSeparacao(e.ctx, 0, e.almox, "FEFO", nec, e.ator)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.repo.CancelarOndaDeSeparacao(e.ctx, onda.Code, e.ator); err != nil {
		t.Fatal(err)
	}
	if r := reservadoDe(t, e, e.item, e.almox); !r.Equal(dec("10")) {
		t.Fatalf("cancelar devolve a reserva ao pedido: %s", r)
	}
	rows, err := pool.Query(context.Background(),
		`SELECT reference_item_code, SUM(quantity) FROM stock_reservations
		  WHERE enterprise_id=$1 AND reference_type='SALES_ORDER' AND reference_code=$2 AND status='ACTIVE'
		  GROUP BY reference_item_code`, e.empresa, pedido)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	porLinha := map[int64]decimal.Decimal{}
	for rows.Next() {
		var refLinha int64
		var q decimal.Decimal
		if err := rows.Scan(&refLinha, &q); err != nil {
			t.Fatal(err)
		}
		porLinha[refLinha] = q
	}
	if len(porLinha) != 2 || !porLinha[l1].Equal(dec("6")) || !porLinha[l2].Equal(dec("4")) {
		t.Fatalf("reserva por linha = %v, esperado %d→6 e %d→4", porLinha, l1, l2)
	}
}
