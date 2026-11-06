//go:build integration

package production_order_uc_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	prodrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	stockrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	structrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

type fakeCompleteAuth struct {
	ports.AuthService
	ator uuid.UUID
}

func (fakeCompleteAuth) CanUpdateSalesOrder(context.Context) bool    { return true }
func (f fakeCompleteAuth) UserID(context.Context) (uuid.UUID, error) { return f.ator, nil }

// Completing an OF receives the item's BOM co-products / returnable scrap into
// stock: an IN movement of coproduct_qty × produced.
func TestIntegration_CompleteReceivesCoproductScrap(t *testing.T) {
	q, pool := testutil.Queries(t)
	ctx := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(ctx, "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	uid := uuid.New()

	finished := testutil.UniqueCode()
	scrap := testutil.UniqueCode()
	for _, code := range []int64{finished, scrap} {
		testutil.Exec(t, pool, "INSERT INTO items (code, business_code, warehouse_code, created_by, enterprise_id) VALUES ($1,($1::bigint)::text,$2,$3,$4)", code, code, uid, enterpriseID)
	}
	defer testutil.Exec(t, pool, "DELETE FROM items WHERE code IN ($1,$2)", finished, scrap)

	// BOM: finished → scrap (co-product, 0.5 per unit).
	testutil.Exec(t, pool, "INSERT INTO item_structures (parent_code, child_code, quantity, sequence, is_coproduct, created_by) VALUES ($1,$2,0.5,1,TRUE,$3)", finished, scrap, uid)
	defer testutil.Exec(t, pool, "DELETE FROM item_structures WHERE parent_code = $1", finished)

	// Production order that produced 100 units.
	var poID int64
	if err := pool.QueryRow(ctx,
		"INSERT INTO production_orders (order_number,item_code,planned_qty,produced_qty,status,created_by,enterprise_id) VALUES ($1,$2,$3,$4,'IN_PROGRESS',$5,$6) RETURNING id",
		testutil.UniqueCode(), finished, 100, 100, uid, enterpriseID).Scan(&poID); err != nil {
		t.Fatalf("seed production_order: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM production_orders WHERE id = $1", poID)
	defer testutil.Exec(t, pool, "DELETE FROM stock_movements WHERE reference_code = $1 AND reference_type = 'PRODUCTION_ORDER'", poID)

	wh := testutil.NovoAlmoxarifado(t, pool, enterpriseID)
	uc := &production_order_uc.CompleteProductionOrderUseCase{
		Repo:      prodrepo.NewProductionOrderRepositoryPGX(pool),
		Auth:      fakeCompleteAuth{ator: testutil.Actor(t, pool)},
		StockRepo: stockrepo.NewStockRepositorySQLC(pool),
		Structure: structrepo.NewItemStructureRepository(q),
	}
	if _, err := uc.Execute(ctx, request.CompleteProductionOrderDTO{ID: poID, WarehouseID: &wh}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	movs, err := uc.StockRepo.ListMovementsByItem(ctx, scrap)
	if err != nil {
		t.Fatalf("ListMovementsByItem: %v", err)
	}
	var received decimal.Decimal
	for _, m := range movs {
		if m.MovementType == "IN" {
			received = received.Add(m.Quantity)
		}
	}
	if !received.Equal(decimal.NewFromInt(50)) {
		t.Fatalf("scrap received = %v, want 50 (0.5 × 100)", received)
	}
}

// A entrada do produto vale o que a ordem gastou. Antes ela entrava a custo zero:
// diluía o médio e o produto saía na venda sem custo.
func TestIntegration_EntregaValorizaPeloCustoDaOrdem(t *testing.T) {
	_, pool := testutil.Queries(t)
	ctx := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(ctx, "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	uid := uuid.New()
	produto, material := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, enterpriseID, uid, produto, material)
	almox := testutil.NovoAlmoxarifado(t, pool, enterpriseID)
	var ordem int64
	if err := pool.QueryRow(ctx,
		"INSERT INTO production_orders (order_number,item_code,planned_qty,produced_qty,status,created_by,enterprise_id,warehouse_id) VALUES ($1,$2,10,0,'IN_PROGRESS',$3,$4,$5) RETURNING id",
		testutil.UniqueCode(), produto, uid, enterpriseID, almox).Scan(&ordem); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM stock_movements WHERE enterprise_id=$1 AND item_code = ANY($2)", enterpriseID, []int64{produto, material})
		pool.Exec(ctx, "DELETE FROM stock_balances WHERE enterprise_id=$1 AND item_code = ANY($2)", enterpriseID, []int64{produto, material})
		pool.Exec(ctx, "DELETE FROM production_orders WHERE id=$1", ordem)
	})
	stock := stockrepo.NewStockRepositorySQLC(pool)
	ref := "PRODUCTION_ORDER"
	for _, m := range []*stockentity.StockMovement{
		{ItemCode: material, WarehouseID: almox, MovementType: "IN", Quantity: decimal.NewFromInt(20), UnitPrice: decimal.NewFromInt(5), CreatedBy: uid},
		{ItemCode: material, WarehouseID: almox, MovementType: "OUT", Quantity: decimal.NewFromInt(10), ReferenceType: &ref, ReferenceCode: &ordem, CreatedBy: uid},
	} {
		if _, err := stock.CreateMovement(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	repo := prodrepo.NewProductionOrderRepositoryPGX(pool)
	o, err := repo.GetByCode(ctx, ordem)
	if err != nil {
		t.Fatal(err)
	}
	unit, err := repo.CustoUnitarioDaEntrega(ctx, o, decimal.NewFromInt(4), false)
	if err != nil {
		t.Fatal(err)
	}
	if !unit.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("entrega parcial: 50 gastos em 10 a produzir valem 5, veio %s", unit)
	}
	unit, err = repo.CustoUnitarioDaEntrega(ctx, o, decimal.NewFromInt(5), true)
	if err != nil {
		t.Fatal(err)
	}
	if !unit.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("entrega final leva tudo: 50 em 5 valem 10, veio %s", unit)
	}
}

// Foi o que aconteceu na OP 2 da Tecnofer: o apontamento baixou o componente e a
// entrega baixou de novo. A entrega só tira o que falta.
func TestIntegration_EntregaNaoBaixaDeNovoOQueOApontamentoBaixou(t *testing.T) {
	_, pool := testutil.Queries(t)
	ctx := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(ctx, "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	uid := uuid.New()
	produto, material := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, enterpriseID, uid, produto, material)
	almox := testutil.NovoAlmoxarifado(t, pool, enterpriseID)
	testutil.Exec(t, pool, "INSERT INTO item_structures (parent_code, child_code, quantity, sequence, created_by) VALUES ($1,$2,1,1,$3)", produto, material, uid)
	testutil.Exec(t, pool, `UPDATE items SET warehouse_automatic_low=TRUE, warehouse_code=$2 WHERE code=$1`, material, almox)
	var ordem int64
	if err := pool.QueryRow(ctx,
		"INSERT INTO production_orders (order_number,item_code,planned_qty,produced_qty,status,created_by,enterprise_id,warehouse_id) VALUES ($1,$2,2,0,'IN_PROGRESS',$3,$4,$5) RETURNING id",
		testutil.UniqueCode(), produto, uid, enterpriseID, almox).Scan(&ordem); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM production_delivery_lines WHERE production_delivery_id IN (SELECT id FROM production_deliveries WHERE production_order_id=$1)", ordem)
		pool.Exec(ctx, "DELETE FROM production_deliveries WHERE production_order_id=$1", ordem)
		pool.Exec(ctx, "DELETE FROM stock_movements WHERE enterprise_id=$1 AND item_code = ANY($2)", enterpriseID, []int64{produto, material})
		pool.Exec(ctx, "DELETE FROM stock_balances WHERE enterprise_id=$1 AND item_code = ANY($2)", enterpriseID, []int64{produto, material})
		pool.Exec(ctx, "DELETE FROM production_orders WHERE id=$1", ordem)
		pool.Exec(ctx, "DELETE FROM item_structures WHERE parent_code=$1", produto)
	})
	stock := stockrepo.NewStockRepositorySQLC(pool)
	ref := "PRODUCTION_ORDER"
	for _, m := range []*stockentity.StockMovement{
		{ItemCode: material, WarehouseID: almox, MovementType: "IN", Quantity: decimal.NewFromInt(10), UnitPrice: decimal.NewFromInt(7), CreatedBy: uid},
		// O apontamento já baixou 1 dos 2.
		{ItemCode: material, WarehouseID: almox, MovementType: "OUT", Quantity: decimal.NewFromInt(1), ReferenceType: &ref, ReferenceCode: &ordem, CreatedBy: uid},
	} {
		if _, err := stock.CreateMovement(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	q2, _ := testutil.Queries(t)
	uc := &production_order_uc.CompleteProductionOrderUseCase{
		Repo: prodrepo.NewProductionOrderRepositoryPGX(pool), Auth: fakeCompleteAuth{ator: testutil.Actor(t, pool)},
		StockRepo: stock, Structure: structrepo.NewItemStructureRepository(q2),
	}
	dois := decimal.NewFromInt(2)
	if _, err := uc.Execute(ctx, request.CompleteProductionOrderDTO{ID: ordem, WarehouseID: &almox, Quantity: &dois}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var rep, saldo, unit decimal.Decimal
	if err := pool.QueryRow(ctx, `SELECT COALESCE(SUM(quantity) FILTER (WHERE movement_type='REP'),0) FROM stock_movements WHERE reference_code=$1 AND reference_type='PRODUCTION_ORDER' AND item_code=$2`, ordem, material).Scan(&rep); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT SUM(quantity) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, enterpriseID, material).Scan(&saldo); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT unit_price FROM stock_movements WHERE reference_code=$1 AND reference_type='PRODUCTION_ORDER' AND item_code=$2`, ordem, produto).Scan(&unit); err != nil {
		t.Fatal(err)
	}
	if !rep.Equal(decimal.NewFromInt(1)) || !saldo.Equal(decimal.NewFromInt(8)) {
		t.Fatalf("baixa na entrega %s, saldo %s: faltava 1 dos 2, sobram 8", rep, saldo)
	}
	if !unit.Equal(decimal.NewFromInt(7)) {
		t.Fatalf("o produto vale o material que a ordem gastou, incluindo a baixa desta entrega: %s", unit)
	}
}

// O fechamento lê a ordem pela referência que a entrega real grava: se a referência
// não for a do id, o consumo some da contabilização sem erro nenhum.
func TestIntegration_FechamentoApuraAOrdemEntregueDeVerdade(t *testing.T) {
	_, pool := testutil.Queries(t)
	bg := context.Background()
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		pool.Exec(bg, `DELETE FROM accounting_posting_params WHERE enterprise_id=$1`, a.ID)
	})
	ctx := a.Ctx
	ator := testutil.Actor(t, pool)
	produto, material := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, produto, material)
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	if _, err := pool.Exec(bg, "INSERT INTO item_structures (parent_code, child_code, quantity, sequence, created_by) VALUES ($1,$2,1,1,$3)", produto, material, ator); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec(bg, "DELETE FROM item_structures WHERE parent_code=$1", produto) })
	if _, err := pool.Exec(bg, `UPDATE items SET warehouse_automatic_low=TRUE, warehouse_code=$2 WHERE code=$1`, material, almox); err != nil {
		t.Fatal(err)
	}
	var ordem int64
	if err := pool.QueryRow(bg, "INSERT INTO production_orders (order_number,item_code,planned_qty,produced_qty,status,created_by,enterprise_id,warehouse_id) VALUES ($1,$2,2,0,'RELEASED',$3,$4,$5) RETURNING id",
		900_000+testutil.UniqueCode()%90_000, produto, ator, a.ID, almox).Scan(&ordem); err != nil {
		t.Fatal(err)
	}
	stock := stockrepo.NewStockRepositorySQLC(pool)
	if _, err := stock.CreateMovement(ctx, &stockentity.StockMovement{ItemCode: material, WarehouseID: almox, MovementType: "IN", Quantity: decimal.NewFromInt(10), UnitPrice: decimal.NewFromInt(7), CreatedBy: ator}); err != nil {
		t.Fatal(err)
	}
	q2, _ := testutil.Queries(t)
	uc := &production_order_uc.CompleteProductionOrderUseCase{
		Repo: prodrepo.NewProductionOrderRepositoryPGX(pool), Auth: fakeCompleteAuth{ator: ator},
		StockRepo: stock, Structure: structrepo.NewItemStructureRepository(q2),
	}
	dois := decimal.NewFromInt(2)
	if _, err := uc.Execute(ctx, request.CompleteProductionOrderDTO{ID: ordem, WarehouseID: &almox, Quantity: &dois}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	inicio := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	if _, err := pool.Exec(bg, `UPDATE stock_movements SET movement_date=$2::date, backdated=FALSE WHERE enterprise_id=$1`, a.ID, inicio.AddDate(0, 0, 9)); err != nil {
		t.Fatal(err)
	}
	var plano int64
	if err := pool.QueryRow(bg, `INSERT INTO accounting_plans(plan_number,description,valid_from,status,enterprise_id) VALUES($1,'Fech','2020-01-01','A',$2) RETURNING id`, testutil.UniqueCode()%1_000_000+500_000, a.ID).Scan(&plano); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(bg, `DELETE FROM accounting_posting_params WHERE enterprise_id=$1`, a.ID)
		pool.Exec(bg, `DELETE FROM accounting_accounts WHERE plan_id=$1`, plano)
		pool.Exec(bg, `DELETE FROM accounting_plans WHERE id=$1`, plano)
	})
	conta := func(n string) int64 {
		var id int64
		if err := pool.QueryRow(bg, `INSERT INTO accounting_accounts(plan_id,account_number,description,nature_code,valid_from,is_analytic) VALUES($1,$2,$2,'D','2020-01-01',TRUE) RETURNING id`, plano, n).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	if _, err := pool.Exec(bg, `INSERT INTO accounting_posting_params(enterprise_id,plan_id,fornecedores_account_id,estoque_account_id,cmv_account_id,producao_processo_account_id,custos_absorvidos_account_id,contabilizar_producao)
		VALUES($1,$2,$3,$4,$5,$6,$7,TRUE)`, a.ID, plano, conta("1"), conta("2"), conta("3"), conta("4"), conta("5")); err != nil {
		t.Fatal(err)
	}
	res, err := stock.FecharMes(ctx, inicio, ator, true)
	if err != nil {
		t.Fatal(err)
	}
	if !res.ProducaoConsumo.Equal(decimal.NewFromInt(14)) || !res.ProducaoEntrega.Equal(decimal.NewFromInt(14)) {
		t.Fatalf("consumo %s, entrega %s; quero 14 e 14: %+v", res.ProducaoConsumo, res.ProducaoEntrega, res)
	}
}
