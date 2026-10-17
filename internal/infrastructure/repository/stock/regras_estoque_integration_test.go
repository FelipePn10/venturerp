//go:build integration

package stock_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

type ambienteEstoque struct {
	ctx          context.Context
	empresa      int64
	ator         uuid.UUID
	repo         *stockrepo.StockRepositorySQLC
	almox, livre int64 // livre = permite saldo negativo
	item, outro  int64
}

func novoAmbienteEstoque(t *testing.T) ambienteEstoque {
	t.Helper()
	pool := testutil.Pool(t)
	ator := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, q := range []string{
			`DELETE FROM material_requisition_items WHERE requisition_id IN (SELECT id FROM material_requisitions WHERE enterprise_id=$1)`,
			`DELETE FROM material_requisitions WHERE enterprise_id=$1`,
			`DELETE FROM item_unit_conversions WHERE item_code IN (SELECT code FROM items WHERE enterprise_id=$1)`,
			`DELETE FROM item_consumption_averages WHERE enterprise_id=$1`,
			`UPDATE stock_movements SET reversed_by_movement_id=NULL, reverses_movement_id=NULL WHERE enterprise_id=$1`,
			`DELETE FROM physical_inventory_items WHERE inventory_id IN (SELECT id FROM physical_inventories WHERE enterprise_id=$1)`,
			`DELETE FROM physical_inventories WHERE enterprise_id=$1`,
			`DELETE FROM stock_lot_balances WHERE enterprise_id=$1`,
			`DELETE FROM stock_movements WHERE enterprise_id=$1`,
			`DELETE FROM stock_balances WHERE enterprise_id=$1`,
			`DELETE FROM manufacturing_stock_closed_periods WHERE enterprise_id=$1`,
			`DELETE FROM items WHERE enterprise_id=$1`,
			`DELETE FROM warehouse WHERE enterprise_id=$1`,
		} {
			testutil.Exec(t, pool, q, a.ID)
		}
	})
	e := ambienteEstoque{ctx: a.Ctx, empresa: a.ID, ator: ator, repo: stockrepo.NewStockRepositorySQLC(pool)}
	novoAlmox := func(codigo string, permite bool) int64 {
		var id int64
		if err := pool.QueryRow(context.Background(),
			`INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id,allow_negative_stock)
			 VALUES($1,$1,$2,'INTERNO','NORMAL',true,true,$3,$4) RETURNING id`, codigo, ator, a.ID, permite).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	u := testutil.UniqueCode()
	e.almox = novoAlmox("EST-"+uuid.NewString()[:8], false)
	e.livre = novoAlmox("LIN-"+uuid.NewString()[:8], true)
	e.item, e.outro = u, u+1
	for _, c := range []int64{e.item, e.outro} {
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,$2,$3,$4)`, c, e.almox, ator, a.ID)
	}
	return e
}

func (e ambienteEstoque) mover(t *testing.T, item, almox int64, tipo string, qtd, preco float64, endereco string) error {
	t.Helper()
	m := &stockentity.StockMovement{ItemCode: item, WarehouseID: almox, MovementType: tipo, Quantity: qtd, UnitPrice: preco, CreatedBy: e.ator}
	if endereco != "" {
		m.Address = &endereco
	}
	_, err := e.repo.CreateMovement(e.ctx, m)
	return err
}

func (e ambienteEstoque) saldo(t *testing.T, item, almox int64) (float64, float64) {
	t.Helper()
	var q, custo float64
	if err := testutil.Pool(t).QueryRow(context.Background(),
		`SELECT COALESCE(SUM(quantity),0)::float8, COALESCE(MAX(avg_cost),0)::float8 FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3`,
		e.empresa, item, almox).Scan(&q, &custo); err != nil {
		t.Fatal(err)
	}
	return q, custo
}

func ehValidacao(err error) bool {
	_, ok := errorsuc.AsValidation(err)
	return ok
}

// Saída que deixaria o saldo negativo é recusada, salvo no almoxarifado que
// permite; a recusa não deixa movimento nem saldo pela metade.
func TestIntegration_SaldoNegativoBloqueado(t *testing.T) {
	e := novoAmbienteEstoque(t)
	if err := e.mover(t, e.item, e.almox, "IN", 5, 10, ""); err != nil {
		t.Fatal(err)
	}
	err := e.mover(t, e.item, e.almox, "OUT", 8, 0, "")
	if !ehValidacao(err) || !strings.Contains(err.Error(), "saldo insuficiente") {
		t.Fatalf("saída maior que o saldo deveria ser recusada: %v", err)
	}
	if q, _ := e.saldo(t, e.item, e.almox); q != 5 {
		t.Fatalf("a recusa não pode mexer no saldo: %v", q)
	}
	if err := e.mover(t, e.item, e.almox, "OUT", 5, 0, ""); err != nil {
		t.Fatalf("saída do saldo exato tem de passar: %v", err)
	}
	if err := e.mover(t, e.item, e.livre, "OUT", 3, 0, ""); err != nil {
		t.Fatalf("almoxarifado que permite negativo aceita: %v", err)
	}
	if q, _ := e.saldo(t, e.item, e.livre); q != -3 {
		t.Fatalf("saldo negativo esperado no almoxarifado livre: %v", q)
	}
	// Endereço sem saldo, mesmo com saldo no almoxarifado.
	if err := e.mover(t, e.outro, e.almox, "IN", 10, 1, "A-01"); err != nil {
		t.Fatal(err)
	}
	if err := e.mover(t, e.outro, e.almox, "OUT", 2, 0, "B-01"); !ehValidacao(err) || !strings.Contains(err.Error(), "endereço B-01") {
		t.Fatalf("saída de endereço vazio deveria ser recusada pelo endereço: %v", err)
	}
}

// O custo médio é um só para o item no almoxarifado, somando os endereços, e a
// saída sem preço é valorizada por ele.
func TestIntegration_CustoMedioUnicoEntreEnderecos(t *testing.T) {
	e := novoAmbienteEstoque(t)
	if err := e.mover(t, e.item, e.almox, "IN", 10, 10, "A-01"); err != nil {
		t.Fatal(err)
	}
	if err := e.mover(t, e.item, e.almox, "IN", 10, 20, "B-01"); err != nil {
		t.Fatal(err)
	}
	var custos []float64
	rows, err := testutil.Pool(t).Query(context.Background(),
		`SELECT avg_cost::float8 FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3 ORDER BY address`, e.empresa, e.item, e.almox)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var c float64
		_ = rows.Scan(&c)
		custos = append(custos, c)
	}
	rows.Close()
	if len(custos) != 2 || custos[0] != 15 || custos[1] != 15 {
		t.Fatalf("os dois endereços têm de ter o custo comum 15: %v", custos)
	}
	saida := &stockentity.StockMovement{ItemCode: e.item, WarehouseID: e.almox, MovementType: "OUT", Quantity: 4, CreatedBy: e.ator}
	addr := "A-01"
	saida.Address = &addr
	if _, err := e.repo.CreateMovement(e.ctx, saida); err != nil {
		t.Fatal(err)
	}
	if saida.UnitPrice != 15 || saida.TotalPrice != 60 {
		t.Fatalf("a saída vale pelo custo comum (15): %v / %v", saida.UnitPrice, saida.TotalPrice)
	}
	var total float64
	if err := testutil.Pool(t).QueryRow(context.Background(),
		`SELECT SUM(total_cost)::float8 FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3`, e.empresa, e.item, e.almox).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 240 {
		t.Fatalf("valor do estoque: 300 − 60 = 240, veio %v", total)
	}
}

// Lançamento com data num período de estoque fechado é recusado.
func TestIntegration_PeriodoDeEstoqueFechado(t *testing.T) {
	e := novoAmbienteEstoque(t)
	hoje := time.Now()
	testutil.Exec(t, testutil.Pool(t), `INSERT INTO manufacturing_stock_closed_periods(enterprise_id,period_from,period_to) VALUES($1,$2::date - 3,$2::date + 3)`, e.empresa, hoje)
	if err := e.mover(t, e.item, e.almox, "IN", 1, 1, ""); !ehValidacao(err) || !strings.Contains(err.Error(), "está fechado") {
		t.Fatalf("período fechado deveria recusar: %v", err)
	}
}

// Inventário: a abertura grava a foto, a contagem calcula a diferença (com
// recontagem e item achado fora da foto) e o fechamento lança os ajustes.
func TestIntegration_InventarioFisicoAjustaOSaldo(t *testing.T) {
	e := novoAmbienteEstoque(t)
	if err := e.mover(t, e.item, e.almox, "IN", 10, 8, ""); err != nil {
		t.Fatal(err)
	}
	inv, err := e.repo.CreateInventory(e.ctx, &stockentity.PhysicalInventory{
		Code: testutil.UniqueCode() % 1_000_000, Description: "Inventário geral", WarehouseID: e.almox,
		StartDate: time.Now(), Status: "OPEN", CreatedBy: e.ator,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.repo.CreateInventory(e.ctx, &stockentity.PhysicalInventory{
		Code: inv.Code + 1, Description: "outro", WarehouseID: e.almox, StartDate: time.Now(), Status: "OPEN", CreatedBy: e.ator,
	}); !errors.As(err, new(*errorsuc.ConflictError)) {
		t.Fatalf("segundo inventário aberto no mesmo almoxarifado deveria dar conflito: %v", err)
	}
	itens, _ := e.repo.ListInventoryItems(e.ctx, inv.ID)
	if len(itens) != 1 || itens[0].SystemQty != 10 || itens[0].UnitCost == nil || *itens[0].UnitCost != 8 {
		t.Fatalf("a foto deveria ter o item com 10 a custo 8: %+v", itens)
	}
	// Saída depois da abertura: não pode ser desfeita pelo ajuste.
	if err := e.mover(t, e.item, e.almox, "OUT", 2, 0, ""); err != nil {
		t.Fatal(err)
	}
	contar := func(item int64, qtd float64) {
		t.Helper()
		if err := e.repo.CountInventoryItem(e.ctx, &stockentity.PhysicalInventoryItem{
			InventoryID: inv.ID, ItemCode: item, WarehouseID: e.almox, CountedQty: &qtd, CountedBy: &e.ator,
		}); err != nil {
			t.Fatal(err)
		}
	}
	contar(e.item, 9)
	contar(e.item, 7) // recontagem
	contar(e.outro, 4)
	itens, _ = e.repo.ListInventoryItems(e.ctx, inv.ID)
	if len(itens) != 2 {
		t.Fatalf("item achado fora da foto entra no inventário: %+v", itens)
	}
	for _, it := range itens {
		if it.ItemCode == e.item && (it.CountNumber != 2 || *it.PreviousCountedQty != 9 || *it.DifferenceQty != -3) {
			t.Fatalf("recontagem: contagem 2, anterior 9, diferença 7−10 = −3: %+v", it)
		}
		if it.ItemCode == e.outro && *it.DifferenceQty != 4 {
			t.Fatalf("item fora da foto: diferença 4: %+v", it)
		}
	}
	res, err := e.repo.FecharInventario(e.ctx, inv.ID, e.ator, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ajustes != 2 || res.Faltas != 3 || res.Sobras != 4 {
		t.Fatalf("resultado do fechamento: %+v", res)
	}
	if q, _ := e.saldo(t, e.item, e.almox); q != 5 {
		t.Fatalf("saldo = 10 − 2 (saída depois da abertura) − 3 (falta) = 5, veio %v", q)
	}
	if q, _ := e.saldo(t, e.outro, e.almox); q != 4 {
		t.Fatalf("sobra do item fora da foto: 4, veio %v", q)
	}
	var refs int
	if err := testutil.Pool(t).QueryRow(context.Background(),
		`SELECT COUNT(*) FROM stock_movements WHERE enterprise_id=$1 AND reference_type='INVENTORY' AND reference_code=$2 AND movement_type='ADJUSTMENT'`,
		e.empresa, inv.Code).Scan(&refs); err != nil || refs != 2 {
		t.Fatalf("os dois ajustes têm de referenciar o inventário: %d %v", refs, err)
	}
	if _, err := e.repo.FecharInventario(e.ctx, inv.ID, e.ator, false); !ehValidacao(err) {
		t.Fatalf("fechar de novo deveria ser recusado: %v", err)
	}
}

// Fechar com item sem contagem é recusado, a menos que se peça para zerar.
func TestIntegration_InventarioZeraNaoContados(t *testing.T) {
	e := novoAmbienteEstoque(t)
	_ = e.mover(t, e.item, e.almox, "IN", 6, 2, "")
	_ = e.mover(t, e.outro, e.almox, "IN", 3, 2, "")
	inv, err := e.repo.CreateInventory(e.ctx, &stockentity.PhysicalInventory{
		Code: testutil.UniqueCode() % 1_000_000, Description: "parcial", WarehouseID: e.almox,
		StartDate: time.Now(), Status: "OPEN", CreatedBy: e.ator, ItemCodes: []int64{e.item},
	})
	if err != nil {
		t.Fatal(err)
	}
	itens, _ := e.repo.ListInventoryItems(e.ctx, inv.ID)
	if len(itens) != 1 || itens[0].ItemCode != e.item {
		t.Fatalf("inventário parcial só com o item pedido: %+v", itens)
	}
	if _, err := e.repo.FecharInventario(e.ctx, inv.ID, e.ator, false); !ehValidacao(err) || !strings.Contains(err.Error(), "sem contagem") {
		t.Fatalf("item sem contagem deveria impedir o fechamento: %v", err)
	}
	res, err := e.repo.FecharInventario(e.ctx, inv.ID, e.ator, true)
	if err != nil || res.NaoContadosZerados != 1 {
		t.Fatalf("zerando os não contados: %+v %v", res, err)
	}
	if q, _ := e.saldo(t, e.item, e.almox); q != 0 {
		t.Fatalf("item não contado zerado: %v", q)
	}
	if q, _ := e.saldo(t, e.outro, e.almox); q != 3 {
		t.Fatalf("item fora do inventário parcial não muda: %v", q)
	}
}
