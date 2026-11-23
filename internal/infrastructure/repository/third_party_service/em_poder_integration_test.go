//go:build integration

package third_party_service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	domain "github.com/FelipePn10/panossoerp/internal/domain/third_party_service"
	stockpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	repoimpl "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/third_party_service"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Bug I: a remessa para a TRATE era só um registro — a peça continuava no
// almoxarifado. A remessa de estoque transfere para o almoxarifado do terceiro,
// o recebimento traz de volta, e o saldo em poder aparece por fornecedor. A
// empresa B não enxerga nada da A.
func TestRemessaLevaAoAlmoxarifadoDoTerceiro(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []int64{a.ID, b.ID} {
			pool.Exec(base, `DELETE FROM third_party_service_order_history WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM third_party_service_movements WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM third_party_service_orders WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM stock_movements WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM stock_balances WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM warehouse WHERE enterprise_id=$1 AND supplier_code IS NOT NULL`, e)
			pool.Exec(base, `DELETE FROM fiscal_exits WHERE enterprise_id=$1`, e)
		}
	})
	item := testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, item)
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	fornecedor := testutil.UniqueCode()
	var fornecedorID, operacao, roteiro, rotaOp, of int64
	if err := pool.QueryRow(base, `INSERT INTO suppliers(code,name,document_type,document_number,created_by,enterprise_id) VALUES($1,'TRATE Tratamentos','CNPJ',$2,$3,$4) RETURNING id`,
		fornecedor, fmt.Sprint(fornecedor), ator, a.ID).Scan(&fornecedorID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec(base, `DELETE FROM suppliers WHERE id=$1`, fornecedorID) })
	if err := pool.QueryRow(base, `INSERT INTO operations(code,name,origin,created_by,enterprise_id) VALUES($1,'Têmpera','TERCEIROS',$2,$3) RETURNING id`, testutil.UniqueCode(), ator, a.ID).Scan(&operacao); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO manufacturing_routes(code,item_code,alternative,is_standard,created_by,enterprise_id) VALUES($1,$2,1,FALSE,$3,$4) RETURNING id`, testutil.UniqueCode(), item, ator, a.ID).Scan(&roteiro); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO route_operations(route_id,sequence,operation_id) VALUES($1,10,$2) RETURNING id`, roteiro, operacao).Scan(&rotaOp); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,mask,planned_qty,status,route_id,enterprise_id,created_by) VALUES($1,$2,'',10,'OPEN',$3,$4,$5) RETURNING id`,
		testutil.UniqueCode()%1_000_000, item, roteiro, a.ID, ator).Scan(&of); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(base, `DELETE FROM production_orders WHERE id=$1`, of)
		pool.Exec(base, `DELETE FROM route_operations WHERE id=$1`, rotaOp)
		pool.Exec(base, `DELETE FROM manufacturing_routes WHERE id=$1`, roteiro)
		pool.Exec(base, `DELETE FROM operations WHERE id=$1`, operacao)
	})
	testutil.FonteDeSuprimento(t, pool, item, fornecedor)
	hoje := time.Now()
	var os int64
	if err := pool.QueryRow(base, `INSERT INTO third_party_service_orders(code,enterprise_id,production_order_id,route_operation_id,operation_id,item_code,supplier_code,uom,quantity,start_date,due_date,status,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,'UN',10,$8::date,$8::date + 10,'RELEASED_WITHOUT_PO',$9) RETURNING id`,
		testutil.UniqueCode()%1_000_000, a.ID, of, rotaOp, operacao, item, fornecedor, hoje, ator).Scan(&os); err != nil {
		t.Fatal(err)
	}
	estoque := stockpg.NewStockRepositorySQLC(pool)
	if _, err := estoque.CreateMovement(a.Ctx, &stockentity.StockMovement{ItemCode: item, WarehouseID: almox, MovementType: "IN",
		Quantity: decimal.NewFromInt(10), UnitPrice: decimal.NewFromInt(5), CreatedBy: ator}); err != nil {
		t.Fatal(err)
	}
	saldo := func(w int64) string {
		var q decimal.Decimal
		if err := pool.QueryRow(base, `SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3`, a.ID, item, w).Scan(&q); err != nil {
			t.Fatal(err)
		}
		return q.String()
	}
	repo := repoimpl.New(pool).(*repoimpl.Repo)
	mover := func(tipo string, qtd int64, w *int64) {
		t.Helper()
		if _, err := repo.AddMovement(a.Ctx, os, domain.Movement{MovementType: tipo, Quantity: decimal.NewFromInt(qtd), OccurredAt: hoje,
			IdempotencyKey: fmt.Sprintf("%s-%d-%d", tipo, os, testutil.UniqueCode()), WarehouseID: w, CreatedBy: ator}); err != nil {
			t.Fatalf("%s: %v", tipo, err)
		}
	}
	mover("REMITTANCE", 4, &almox)
	var terceiro int64
	if err := pool.QueryRow(base, `SELECT id FROM warehouse WHERE enterprise_id=$1 AND supplier_code=$2 AND location='TERCEIRO'`, a.ID, fornecedor).Scan(&terceiro); err != nil {
		t.Fatalf("o almoxarifado do terceiro não foi criado: %v", err)
	}
	if saldo(almox) != "6" || saldo(terceiro) != "4" {
		t.Fatalf("depois da remessa: fábrica %s, terceiro %s; quer 6 e 4", saldo(almox), saldo(terceiro))
	}
	em, err := repo.EmPoder(a.Ctx, domain.FiltroEmPoder{})
	if err != nil || len(em) != 1 || !em[0].InHands.Equal(decimal.NewFromInt(4)) || !em[0].Value.Equal(decimal.NewFromInt(20)) || em[0].SupplierCode != fornecedor {
		t.Fatalf("em poder = %+v err=%v", em, err)
	}
	if emB, err := repo.EmPoder(b.Ctx, domain.FiltroEmPoder{}); err != nil || len(emB) != 0 {
		t.Fatalf("a empresa B enxergou o saldo em poder da A: %+v err=%v", emB, err)
	}
	if _, err := repo.RemessaDaOS(b.Ctx, os, nil); err == nil {
		t.Fatal("a empresa B leu a OS da A para emitir a remessa")
	}
	if r, err := repo.RemessaDaOS(a.Ctx, os, nil); err != nil || r.Fornecedor != fornecedor || !r.CustoMedio.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("remessa da OS = %+v err=%v", r, err)
	}

	mover("RECEIPT", 4, nil)
	if saldo(almox) != "10" || saldo(terceiro) != "0" {
		t.Fatalf("depois do recebimento: fábrica %s, terceiro %s; quer 10 e 0", saldo(almox), saldo(terceiro))
	}
	if em, _ := repo.EmPoder(a.Ctx, domain.FiltroEmPoder{}); len(em) != 0 {
		t.Fatalf("nada mais com o terceiro: %+v", em)
	}

	// NF-e de remessa cancelada: a peça não saiu e volta para a fábrica (uma vez só).
	mover("REMITTANCE", 3, &almox)
	var nota, remessa int64
	if err := pool.QueryRow(base, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,created_by,enterprise_id,status)
		VALUES($1,$2::date,'5901','Remessa',$3,$4,'CANCELLED') RETURNING id`, testutil.UniqueCode()%1_000_000, hoje, ator, a.ID).Scan(&nota); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `SELECT MAX(id) FROM third_party_service_movements WHERE service_order_id=$1 AND movement_type='REMITTANCE'`, os).Scan(&remessa); err != nil {
		t.Fatal(err)
	}
	if err := repo.LigarNotaAoMovimento(a.Ctx, os, remessa, nota); err != nil {
		t.Fatal(err)
	}
	if saldo(almox) != "7" {
		t.Fatalf("depois da segunda remessa a fábrica tem 7, veio %s", saldo(almox))
	}
	for range 2 {
		if err := repo.EstornarRemessasDaNota(a.Ctx, nota); err != nil {
			t.Fatal(err)
		}
	}
	if saldo(almox) != "10" || saldo(terceiro) != "0" {
		t.Fatalf("depois do cancelamento da remessa: fábrica %s, terceiro %s; quer 10 e 0", saldo(almox), saldo(terceiro))
	}

	// Retorno pela nota do terceiro (5902): a aprovação recebe na OS e a peça
	// volta; aprovar de novo a mesma linha não recebe duas vezes; a empresa B
	// não acha a OS da A; o que passa do que está com o terceiro fica de fora.
	mover("REMITTANCE", 3, &almox)
	if rec, _, err := repo.ReceberRetornoDaNota(b.Ctx, fornecedor, item, decimal.NewFromInt(2), 777, 1, ator, "RECEIPT"); err != nil || !rec.IsZero() {
		t.Fatalf("a empresa B recebeu na OS da A: %s err=%v", rec, err)
	}
	rec, oss, err := repo.ReceberRetornoDaNota(a.Ctx, fornecedor, item, decimal.NewFromInt(2), 777, 1, ator, "RECEIPT")
	if err != nil || !rec.Equal(decimal.NewFromInt(2)) || len(oss) != 1 {
		t.Fatalf("retorno pela nota: %s %v err=%v", rec, oss, err)
	}
	if saldo(almox) != "9" || saldo(terceiro) != "1" {
		t.Fatalf("depois do retorno pela nota: fábrica %s, terceiro %s; quer 9 e 1", saldo(almox), saldo(terceiro))
	}
	if _, _, err := repo.ReceberRetornoDaNota(a.Ctx, fornecedor, item, decimal.NewFromInt(2), 777, 1, ator, "RECEIPT"); err != nil {
		t.Fatalf("reaprovação da mesma linha: %v", err)
	}
	if saldo(almox) != "9" {
		t.Fatalf("a mesma linha da nota recebeu duas vezes: fábrica %s", saldo(almox))
	}
	if rec, _, err := repo.ReceberRetornoDaNota(a.Ctx, fornecedor, item, decimal.NewFromInt(9), 778, 1, ator, "RECEIPT"); err != nil || !rec.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("retorno maior que o que está com o terceiro: recebeu %s err=%v (quer 1)", rec, err)
	}
	if saldo(almox) != "10" || saldo(terceiro) != "0" {
		t.Fatalf("fim: fábrica %s, terceiro %s; quer 10 e 0", saldo(almox), saldo(terceiro))
	}
}

// Bug Q: a OS guarda o item da OF (o acabado), mas o que vai ao terceiro é o
// componente da etapa externa. A remessa transferia o acabado (−40 a custo zero
// no processo) e o valor em poder saía R$ 0.
func TestRemessaLevaOComponenteDaEtapa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []int64{a.ID, b.ID} {
			pool.Exec(base, `DELETE FROM third_party_service_order_history WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM third_party_service_movements WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM third_party_service_orders WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM production_order_materials WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM stock_movements WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM stock_balances WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM warehouse WHERE enterprise_id=$1 AND supplier_code IS NOT NULL`, e)
		}
	})
	acabado, peca, parafuso := testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, acabado, peca, parafuso)
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	fornecedor := testutil.UniqueCode()
	var fornecedorID, operacao, roteiro, rotaOp, of int64
	if err := pool.QueryRow(base, `INSERT INTO suppliers(code,name,document_type,document_number,created_by,enterprise_id) VALUES($1,'TRATE','CNPJ',$2,$3,$4) RETURNING id`,
		fornecedor, fmt.Sprint(fornecedor), ator, a.ID).Scan(&fornecedorID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec(base, `DELETE FROM suppliers WHERE id=$1`, fornecedorID) })
	if err := pool.QueryRow(base, `INSERT INTO operations(code,name,origin,created_by,enterprise_id) VALUES($1,'Zincagem','TERCEIROS',$2,$3) RETURNING id`, testutil.UniqueCode(), ator, a.ID).Scan(&operacao); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO manufacturing_routes(code,item_code,alternative,is_standard,created_by,enterprise_id) VALUES($1,$2,1,FALSE,$3,$4) RETURNING id`, testutil.UniqueCode(), acabado, ator, a.ID).Scan(&roteiro); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO route_operations(route_id,sequence,operation_id) VALUES($1,20,$2) RETURNING id`, roteiro, operacao).Scan(&rotaOp); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,mask,planned_qty,status,route_id,enterprise_id,created_by) VALUES($1,$2,'',40,'OPEN',$3,$4,$5) RETURNING id`,
		testutil.UniqueCode()%1_000_000, acabado, roteiro, a.ID, ator).Scan(&of); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(base, `DELETE FROM production_orders WHERE id=$1`, of)
		pool.Exec(base, `DELETE FROM route_operations WHERE id=$1`, rotaOp)
		pool.Exec(base, `DELETE FROM manufacturing_routes WHERE id=$1`, roteiro)
		pool.Exec(base, `DELETE FROM operations WHERE id=$1`, operacao)
	})
	// A peça entra na etapa 20 (a externa); o parafuso, no início.
	for _, m := range []struct {
		item  int64
		etapa *int
	}{{peca, func() *int { v := 20; return &v }()}, {parafuso, nil}} {
		if _, err := pool.Exec(base, `INSERT INTO production_order_materials(production_order_id,enterprise_id,material_kind,item_code,quantity,warehouse_id,created_by,route_operation_sequence)
			VALUES($1,$2,'DEMAND',$3,40,$4,$5,$6)`, of, a.ID, m.item, almox, ator, m.etapa); err != nil {
			t.Fatal(err)
		}
	}
	testutil.FonteDeSuprimento(t, pool, acabado, fornecedor)
	hoje := time.Now()
	var os int64
	if err := pool.QueryRow(base, `INSERT INTO third_party_service_orders(code,enterprise_id,production_order_id,route_operation_id,operation_id,item_code,supplier_code,uom,quantity,start_date,due_date,status,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,'UN',40,$8::date,$8::date + 10,'RELEASED_WITHOUT_PO',$9) RETURNING id`,
		testutil.UniqueCode()%1_000_000, a.ID, of, rotaOp, operacao, acabado, fornecedor, hoje, ator).Scan(&os); err != nil {
		t.Fatal(err)
	}
	estoque := stockpg.NewStockRepositorySQLC(pool)
	if _, err := estoque.CreateMovement(a.Ctx, &stockentity.StockMovement{ItemCode: peca, WarehouseID: almox, MovementType: "IN",
		Quantity: decimal.NewFromInt(40), UnitPrice: decimal.NewFromInt(7), CreatedBy: ator}); err != nil {
		t.Fatal(err)
	}
	repo := repoimpl.New(pool).(*repoimpl.Repo)
	if _, err := repo.AddMovement(a.Ctx, os, domain.Movement{MovementType: "REMITTANCE", Quantity: decimal.NewFromInt(40), OccurredAt: hoje,
		IdempotencyKey: fmt.Sprintf("q-%d", os), WarehouseID: &almox, CreatedBy: ator}); err != nil {
		t.Fatal(err)
	}
	var acabadoNoProcesso, pecaNaFabrica decimal.Decimal
	pool.QueryRow(base, `SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, a.ID, acabado).Scan(&acabadoNoProcesso)
	pool.QueryRow(base, `SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3`, a.ID, peca, almox).Scan(&pecaNaFabrica)
	if !acabadoNoProcesso.IsZero() || !pecaNaFabrica.IsZero() {
		t.Fatalf("a remessa mexeu no acabado (%s) ou não levou a peça (fábrica ficou com %s)", acabadoNoProcesso, pecaNaFabrica)
	}
	em, err := repo.EmPoder(a.Ctx, domain.FiltroEmPoder{})
	if err != nil || len(em) != 1 || em[0].ItemCode != peca || !em[0].Value.Equal(decimal.NewFromInt(280)) {
		t.Fatalf("em poder = %+v err=%v (quer a peça, R$ 280)", em, err)
	}
	if r, err := repo.RemessaDaOS(a.Ctx, os, nil); err != nil || r.ItemCode != peca || !r.CustoMedio.Equal(decimal.NewFromInt(7)) {
		t.Fatalf("a NF-e de remessa leva %+v err=%v (quer a peça a R$ 7)", r, err)
	}
}

func TestCorrigirRemessaAntigaDoItemErrado(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []int64{a.ID, b.ID} {
			pool.Exec(base, `DELETE FROM third_party_service_order_history WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM third_party_service_movements WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM third_party_service_orders WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM production_order_materials WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM stock_movements WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM stock_balances WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM warehouse WHERE enterprise_id=$1 AND supplier_code IS NOT NULL`, e)
		}
	})
	acabado, peca, parafuso := testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, acabado, peca, parafuso)
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	fornecedor := testutil.UniqueCode()
	var fornecedorID, operacao, roteiro, rotaOp, of int64
	if err := pool.QueryRow(base, `INSERT INTO suppliers(code,name,document_type,document_number,created_by,enterprise_id) VALUES($1,'TRATE','CNPJ',$2,$3,$4) RETURNING id`,
		fornecedor, fmt.Sprint(fornecedor), ator, a.ID).Scan(&fornecedorID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec(base, `DELETE FROM suppliers WHERE id=$1`, fornecedorID) })
	if err := pool.QueryRow(base, `INSERT INTO operations(code,name,origin,created_by,enterprise_id) VALUES($1,'Zincagem','TERCEIROS',$2,$3) RETURNING id`, testutil.UniqueCode(), ator, a.ID).Scan(&operacao); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO manufacturing_routes(code,item_code,alternative,is_standard,created_by,enterprise_id) VALUES($1,$2,1,FALSE,$3,$4) RETURNING id`, testutil.UniqueCode(), acabado, ator, a.ID).Scan(&roteiro); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO route_operations(route_id,sequence,operation_id) VALUES($1,20,$2) RETURNING id`, roteiro, operacao).Scan(&rotaOp); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,mask,planned_qty,status,route_id,enterprise_id,created_by) VALUES($1,$2,'',40,'OPEN',$3,$4,$5) RETURNING id`,
		testutil.UniqueCode()%1_000_000, acabado, roteiro, a.ID, ator).Scan(&of); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(base, `DELETE FROM production_orders WHERE id=$1`, of)
		pool.Exec(base, `DELETE FROM route_operations WHERE id=$1`, rotaOp)
		pool.Exec(base, `DELETE FROM manufacturing_routes WHERE id=$1`, roteiro)
		pool.Exec(base, `DELETE FROM operations WHERE id=$1`, operacao)
	})
	// A peça entra na etapa 20 (a externa); o parafuso, no início.
	for _, m := range []struct {
		item  int64
		etapa *int
	}{{peca, func() *int { v := 20; return &v }()}, {parafuso, nil}} {
		if _, err := pool.Exec(base, `INSERT INTO production_order_materials(production_order_id,enterprise_id,material_kind,item_code,quantity,warehouse_id,created_by,route_operation_sequence)
			VALUES($1,$2,'DEMAND',$3,40,$4,$5,$6)`, of, a.ID, m.item, almox, ator, m.etapa); err != nil {
			t.Fatal(err)
		}
	}
	testutil.FonteDeSuprimento(t, pool, acabado, fornecedor)
	hoje := time.Now()
	var os int64
	if err := pool.QueryRow(base, `INSERT INTO third_party_service_orders(code,enterprise_id,production_order_id,route_operation_id,operation_id,item_code,supplier_code,uom,quantity,start_date,due_date,status,created_by,remittance_type)
		VALUES($1,$2,$3,$4,$5,$6,$7,'UN',40,$8::date,$8::date + 10,'RELEASED_WITHOUT_PO',$9,'ORDER_ITEM') RETURNING id`,
		testutil.UniqueCode()%1_000_000, a.ID, of, rotaOp, operacao, acabado, fornecedor, hoje, ator).Scan(&os); err != nil {
		t.Fatal(err)
	}
	estoque := stockpg.NewStockRepositorySQLC(pool)
	if _, err := estoque.CreateMovement(a.Ctx, &stockentity.StockMovement{ItemCode: peca, WarehouseID: almox, MovementType: "IN",
		Quantity: decimal.NewFromInt(40), UnitPrice: decimal.NewFromInt(7), CreatedBy: ator}); err != nil {
		t.Fatal(err)
	}
	if _, err := estoque.CreateMovement(a.Ctx, &stockentity.StockMovement{ItemCode: acabado, WarehouseID: almox, MovementType: "IN",
		Quantity: decimal.NewFromInt(40), UnitPrice: decimal.NewFromInt(50), CreatedBy: ator}); err != nil {
		t.Fatal(err)
	}
	repo := repoimpl.New(pool).(*repoimpl.Repo)
	if _, err := repo.AddMovement(a.Ctx, os, domain.Movement{MovementType: "REMITTANCE", Quantity: decimal.NewFromInt(40), OccurredAt: hoje,
		IdempotencyKey: fmt.Sprintf("q-%d", os), WarehouseID: &almox, CreatedBy: ator}); err != nil {
		t.Fatal(err)
	}
	// A remessa antiga: o acabado foi ao terceiro. Depois a OS volta a ser de
	// itens demandados (o que ela sempre deveria ter sido).
	if _, err := pool.Exec(base, `UPDATE third_party_service_orders SET remittance_type='DEMAND_ITEMS' WHERE id=$1`, os); err != nil {
		t.Fatal(err)
	}
	saldo := func(item, w int64) string {
		var q decimal.Decimal
		pool.QueryRow(base, `SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3`, a.ID, item, w).Scan(&q)
		return q.String()
	}
	var terceiro int64
	if err := pool.QueryRow(base, `SELECT id FROM warehouse WHERE enterprise_id=$1 AND supplier_code=$2`, a.ID, fornecedor).Scan(&terceiro); err != nil {
		t.Fatal(err)
	}
	if saldo(acabado, terceiro) != "40" {
		t.Fatalf("a remessa antiga deveria ter levado o acabado: %s", saldo(acabado, terceiro))
	}
	if l, err := repo.CorrigirRemessasDoItemErrado(b.Ctx, ator, true); err != nil || len(l) != 0 {
		t.Fatalf("a empresa B corrigiu a OS da A: %+v %v", l, err)
	}
	sim, err := repo.CorrigirRemessasDoItemErrado(a.Ctx, ator, false)
	if err != nil || len(sim) != 1 || sim[0].Situacao != "SIMULADA" || sim[0].ItemCerto != peca || !sim[0].Quantidade.Equal(decimal.NewFromInt(40)) {
		t.Fatalf("simulação = %+v err=%v", sim, err)
	}
	if saldo(acabado, terceiro) != "40" {
		t.Fatal("a simulação gravou")
	}
	feita, err := repo.CorrigirRemessasDoItemErrado(a.Ctx, ator, true)
	if err != nil || len(feita) != 1 || feita[0].Situacao != "CORRIGIDA" {
		t.Fatalf("correção = %+v err=%v", feita, err)
	}
	if saldo(acabado, terceiro) != "0" || saldo(acabado, almox) != "40" || saldo(peca, terceiro) != "40" || saldo(peca, almox) != "0" {
		t.Fatalf("depois da correção: acabado %s/%s, peça %s/%s (terceiro/fábrica)", saldo(acabado, terceiro), saldo(acabado, almox), saldo(peca, terceiro), saldo(peca, almox))
	}
	if deNovo, err := repo.CorrigirRemessasDoItemErrado(a.Ctx, ator, true); err != nil || len(deNovo) != 0 {
		t.Fatalf("repetir corrigiu de novo: %+v %v", deNovo, err)
	}
	em, err := repo.EmPoder(a.Ctx, domain.FiltroEmPoder{})
	if err != nil || len(em) != 1 || em[0].ItemCode != peca || !em[0].Value.Equal(decimal.NewFromInt(280)) {
		t.Fatalf("em poder = %+v err=%v", em, err)
	}
}

// Bug AA: a peça reservada para a OF (componente da OF filha) ia à TRATE pela OS
// da própria OF e a remessa era recusada pela reserva. A remessa usa a reserva
// da OF da OS, e a peça que volta fica reservada de novo para ela.
func TestRemessaUsaAReservaDaPropriaOF(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []int64{a.ID, b.ID} {
			pool.Exec(base, `DELETE FROM third_party_service_order_history WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM third_party_service_movements WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM third_party_service_orders WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM production_order_materials WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM stock_movements WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM stock_balances WHERE enterprise_id=$1`, e)
			pool.Exec(base, `DELETE FROM warehouse WHERE enterprise_id=$1 AND supplier_code IS NOT NULL`, e)
		}
	})
	acabado, peca, parafuso := testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, acabado, peca, parafuso)
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	fornecedor := testutil.UniqueCode()
	var fornecedorID, operacao, roteiro, rotaOp, of int64
	if err := pool.QueryRow(base, `INSERT INTO suppliers(code,name,document_type,document_number,created_by,enterprise_id) VALUES($1,'TRATE','CNPJ',$2,$3,$4) RETURNING id`,
		fornecedor, fmt.Sprint(fornecedor), ator, a.ID).Scan(&fornecedorID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec(base, `DELETE FROM suppliers WHERE id=$1`, fornecedorID) })
	if err := pool.QueryRow(base, `INSERT INTO operations(code,name,origin,created_by,enterprise_id) VALUES($1,'Zincagem','TERCEIROS',$2,$3) RETURNING id`, testutil.UniqueCode(), ator, a.ID).Scan(&operacao); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO manufacturing_routes(code,item_code,alternative,is_standard,created_by,enterprise_id) VALUES($1,$2,1,FALSE,$3,$4) RETURNING id`, testutil.UniqueCode(), acabado, ator, a.ID).Scan(&roteiro); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO route_operations(route_id,sequence,operation_id) VALUES($1,20,$2) RETURNING id`, roteiro, operacao).Scan(&rotaOp); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,mask,planned_qty,status,route_id,enterprise_id,created_by) VALUES($1,$2,'',40,'OPEN',$3,$4,$5) RETURNING id`,
		testutil.UniqueCode()%1_000_000, acabado, roteiro, a.ID, ator).Scan(&of); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(base, `DELETE FROM production_orders WHERE id=$1`, of)
		pool.Exec(base, `DELETE FROM route_operations WHERE id=$1`, rotaOp)
		pool.Exec(base, `DELETE FROM manufacturing_routes WHERE id=$1`, roteiro)
		pool.Exec(base, `DELETE FROM operations WHERE id=$1`, operacao)
	})
	// A peça entra na etapa 20 (a externa); o parafuso, no início.
	for _, m := range []struct {
		item  int64
		etapa *int
	}{{peca, func() *int { v := 20; return &v }()}, {parafuso, nil}} {
		if _, err := pool.Exec(base, `INSERT INTO production_order_materials(production_order_id,enterprise_id,material_kind,item_code,quantity,warehouse_id,created_by,route_operation_sequence)
			VALUES($1,$2,'DEMAND',$3,40,$4,$5,$6)`, of, a.ID, m.item, almox, ator, m.etapa); err != nil {
			t.Fatal(err)
		}
	}
	testutil.FonteDeSuprimento(t, pool, acabado, fornecedor)
	hoje := time.Now()
	var os int64
	if err := pool.QueryRow(base, `INSERT INTO third_party_service_orders(code,enterprise_id,production_order_id,route_operation_id,operation_id,item_code,supplier_code,uom,quantity,start_date,due_date,status,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,'UN',40,$8::date,$8::date + 10,'RELEASED_WITHOUT_PO',$9) RETURNING id`,
		testutil.UniqueCode()%1_000_000, a.ID, of, rotaOp, operacao, acabado, fornecedor, hoje, ator).Scan(&os); err != nil {
		t.Fatal(err)
	}
	estoque := stockpg.NewStockRepositorySQLC(pool)
	if _, err := estoque.CreateMovement(a.Ctx, &stockentity.StockMovement{ItemCode: peca, WarehouseID: almox, MovementType: "IN",
		Quantity: decimal.NewFromInt(40), UnitPrice: decimal.NewFromInt(7), CreatedBy: ator}); err != nil {
		t.Fatal(err)
	}
	var mat int64
	if err := pool.QueryRow(base, `SELECT id FROM production_order_materials WHERE production_order_id=$1 AND item_code=$2`, of, peca).Scan(&mat); err != nil {
		t.Fatal(err)
	}
	if _, err := estoque.CreateReservation(a.Ctx, &stockentity.StockReservation{ItemCode: peca, WarehouseID: almox, Quantity: decimal.NewFromInt(40),
		ReferenceType: stockentity.ReferenceTypeComponenteDaOF, ReferenceCode: of, ReferenceItemCode: &mat, ReservationDate: hoje, Status: "ACTIVE", CreatedBy: ator}); err != nil {
		t.Fatal(err)
	}
	repo := repoimpl.New(pool).(*repoimpl.Repo)
	if _, err := repo.AddMovement(a.Ctx, os, domain.Movement{MovementType: "REMITTANCE", Quantity: decimal.NewFromInt(40), OccurredAt: hoje,
		IdempotencyKey: fmt.Sprintf("q-%d", os), WarehouseID: &almox, CreatedBy: ator}); err != nil {
		t.Fatalf("remessa da peça reservada para a própria OF: %v", err)
	}
	reservadoParaOF := func() decimal.Decimal {
		var q decimal.Decimal
		pool.QueryRow(base, `SELECT COALESCE(SUM(quantity),0) FROM stock_reservations WHERE enterprise_id=$1 AND reference_type='OF_COMPONENTE'
			AND reference_code=$2 AND status='ACTIVE' AND warehouse_id=$3`, a.ID, of, almox).Scan(&q)
		return q
	}
	if got := reservadoParaOF(); !got.IsZero() {
		t.Fatalf("a remessa não consumiu a reserva da OF: %s ainda reservado na fábrica", got)
	}
	if _, err := repo.AddMovement(a.Ctx, os, domain.Movement{MovementType: "RECEIPT", Quantity: decimal.NewFromInt(40), OccurredAt: hoje,
		IdempotencyKey: fmt.Sprintf("q-volta-%d", os), CreatedBy: ator}); err != nil {
		t.Fatalf("retorno da TRATE: %v", err)
	}
	if got := reservadoParaOF(); !got.Equal(decimal.NewFromInt(40)) {
		t.Fatalf("a peça voltou sem a reserva da OF: %s reservado", got)
	}
	// Voltando a outro almoxarifado, a peça segue reservada para a OF.
	outro := testutil.NovoAlmoxarifado(t, pool, a.ID)
	if _, err := pool.Exec(base, `UPDATE third_party_service_orders SET quantity=80, status='RELEASED_WITHOUT_PO' WHERE id=$1`, os); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddMovement(a.Ctx, os, domain.Movement{MovementType: "REMITTANCE", Quantity: decimal.NewFromInt(40), OccurredAt: hoje,
		IdempotencyKey: fmt.Sprintf("q2-%d", os), WarehouseID: &almox, CreatedBy: ator}); err != nil {
		t.Fatalf("segunda remessa: %v", err)
	}
	if _, err := repo.AddMovement(a.Ctx, os, domain.Movement{MovementType: "RECEIPT", Quantity: decimal.NewFromInt(40), OccurredAt: hoje,
		IdempotencyKey: fmt.Sprintf("q2-volta-%d", os), WarehouseID: &outro, CreatedBy: ator}); err != nil {
		t.Fatalf("retorno a outro almoxarifado: %v", err)
	}
	var noOutro decimal.Decimal
	pool.QueryRow(base, `SELECT COALESCE(SUM(quantity),0) FROM stock_reservations WHERE enterprise_id=$1 AND reference_type='OF_COMPONENTE'
		AND reference_code=$2 AND status='ACTIVE' AND warehouse_id=$3`, a.ID, of, outro).Scan(&noOutro)
	if !noOutro.Equal(decimal.NewFromInt(40)) {
		t.Fatalf("de volta a outro almoxarifado, reservado para a OF = %s, quer 40", noOutro)
	}
}
