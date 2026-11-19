//go:build integration

package production_order_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	productionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/shopspring/decimal"
)

// TestCompletarMateriaisDaOFSemLista: a OF aberta sem lista entra na fila da
// rotina; completar grava os materiais com a etapa do roteiro vinda da
// estrutura, e rodar de novo não duplica. A empresa B não completa a OF da A,
// e OF encerrada não é mexida.
func TestCompletarMateriaisDaOFSemLista(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, nil)
	repo := productionrepo.NewProductionOrderRepositoryPGX(pool)

	rn, bu, tp := testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, actor, rn, bu, tp)
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	testutil.Exec(t, pool, `INSERT INTO item_structures(parent_code,child_code,quantity,sequence,route_operation_sequence,created_by)
		VALUES ($1,$2,2,1,20,$4), ($1,$3,1,2,NULL,$4)`, rn, bu, tp, actor)
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM item_structures WHERE parent_code = $1`, rn) })

	of := func(status string) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,created_by,enterprise_id)
			VALUES($1,$2,10,$3,$4,$5) RETURNING id`, testutil.UniqueCode()%1_000_000, rn, status, actor, a.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	aberta, encerrada := of("OPEN"), of("CLOSED")
	// Em produção com peças já apontadas: entra na fila marcada JaProduziu,
	// para a lista entrar com baixa manual (não lançar de uma vez o passado).
	produzindo := of("IN_PROGRESS")
	testutil.Exec(t, pool, `UPDATE production_orders SET produced_qty = 4 WHERE id = $1`, produzindo)

	fila, err := repo.OFsSemMateriais(base, 100000)
	if err != nil {
		t.Fatal(err)
	}
	naFila := map[int64]entity.OFSemMateriais{}
	for _, f := range fila {
		naFila[f.ID] = f
	}
	if f, ok := naFila[aberta]; !ok || f.EnterpriseID != a.ID || f.EnterpriseCode != a.Code || !f.PlannedQty.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("a OF aberta sem lista deveria estar na fila da empresa A: %+v", f)
	}
	if f, ok := naFila[produzindo]; !ok || !f.JaProduziu {
		t.Fatalf("a OF com peças apontadas deveria vir marcada JaProduziu: %+v", f)
	}
	if naFila[aberta].JaProduziu {
		t.Fatal("OF aberta sem nada produzido veio marcada JaProduziu")
	}
	if _, ok := naFila[encerrada]; ok {
		t.Fatal("OF encerrada não entra na fila")
	}

	materiais := func() []*entity.ProductionOrderMaterial {
		return []*entity.ProductionOrderMaterial{
			{Kind: entity.MaterialDemand, ItemCode: bu, Quantity: decimal.NewFromInt(20), WarehouseID: almox, AutomaticIssue: true, CreatedBy: actor},
			{Kind: entity.MaterialDemand, ItemCode: tp, Quantity: decimal.NewFromInt(10), WarehouseID: almox, AutomaticIssue: true, CreatedBy: actor},
		}
	}
	// A empresa B não completa a OF da A.
	if n, err := repo.CompletarMateriais(b.Ctx, aberta, materiais()); err != nil || n != 0 {
		t.Fatalf("empresa B completou a OF da A: n=%d err=%v", n, err)
	}
	if n, err := repo.CompletarMateriais(a.Ctx, aberta, materiais()); err != nil || n != 2 {
		t.Fatalf("completar: n=%d err=%v", n, err)
	}
	// De novo: a OF já tem lista, nada entra.
	if n, err := repo.CompletarMateriais(a.Ctx, aberta, materiais()); err != nil || n != 0 {
		t.Fatalf("completar de novo duplicou: n=%d err=%v", n, err)
	}
	if n, err := repo.CompletarMateriais(a.Ctx, encerrada, materiais()); err != nil || n != 0 {
		t.Fatalf("OF encerrada foi completada: n=%d err=%v", n, err)
	}
	var etapaBucha, etapaTubo *int16
	var total int
	if err := pool.QueryRow(base, `SELECT count(*),
		max(route_operation_sequence) FILTER (WHERE item_code = $2),
		max(route_operation_sequence) FILTER (WHERE item_code = $3)
		FROM production_order_materials WHERE production_order_id = $1 AND enterprise_id = $4`, aberta, bu, tp, a.ID).Scan(&total, &etapaBucha, &etapaTubo); err != nil {
		t.Fatal(err)
	}
	if total != 2 || etapaBucha == nil || *etapaBucha != 20 || etapaTubo != nil {
		t.Fatalf("lista gravada: %d materiais, etapa da bucha %v, do tubo %v", total, etapaBucha, etapaTubo)
	}
	fila, err = repo.OFsSemMateriais(base, 100000)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fila {
		if f.ID == aberta {
			t.Fatal("a OF completada não sai da fila")
		}
	}
}
