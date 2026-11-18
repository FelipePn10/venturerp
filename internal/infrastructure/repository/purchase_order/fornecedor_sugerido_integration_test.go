//go:build integration

package purchase_order

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O fornecedor que o MRP escolheu (VSUP0130) chega à aprovação da sugestão e à
// lista de sugestões — e só para a empresa da ordem planejada.
func TestFornecedorSugeridoPeloMRPDuasEmpresas(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM planned_orders WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM mrp_planned_suggestions WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM suppliers WHERE enterprise_id = ANY($1)`, ids)
	})
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, ator)
	forn := func(e testutil.Empresa) int64 {
		c := 1_700_000_000 + testutil.UniqueCode()%100_000_000
		testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,$2,$3,$4,$5)`,
			c, fmt.Sprintf("FORN %d", c), fmt.Sprintf("%014d", c), ator, e.ID)
		return c
	}
	fornecedor, fornecedorB := forn(a), forn(b)
	hoje := time.Now()

	var sugestao, planejada, semFornecedor int64
	if err := pool.QueryRow(base, `INSERT INTO mrp_planned_suggestions(plan_code,item_code,quantity,need_date,order_type,supplier_code,enterprise_id)
		VALUES(1,$1,10,$2::date,'COMPRA',$3,$4) RETURNING code`, item, hoje, fornecedor, a.ID).Scan(&sugestao); err != nil {
		t.Fatal(err)
	}
	planejadaDe := func(sug *int64) int64 {
		t.Helper()
		var code int64
		if err := pool.QueryRow(base, `INSERT INTO planned_orders(order_number,item_code,quantity,order_type,demand_type,need_date,created_by,demand_code,mrp_suggestion_code,enterprise_id)
			VALUES($1,$2,10,'PURCHASE','REPLENISHMENT',$3::date,$4,0,$5,$6) RETURNING code`,
			testutil.UniqueCode()%1_000_000_000, item, hoje, ator, sug, a.ID).Scan(&code); err != nil {
			t.Fatal(err)
		}
		return code
	}
	planejada = planejadaDe(&sugestao)
	semFornecedor = planejadaDe(nil)

	r := NewPurchaseOrderRepositorySQLC(pool)
	if got, ok, err := r.FornecedorSugerido(a.Ctx, planejada); err != nil || !ok || got != fornecedor {
		t.Fatalf("fornecedor sugerido: %d %v %v (esperado %d)", got, ok, err, fornecedor)
	}
	if _, ok, err := r.FornecedorSugerido(a.Ctx, semFornecedor); err != nil || ok {
		t.Fatalf("ordem sem sugestão de origem devolveu fornecedor: %v %v", ok, err)
	}
	lista, err := r.FornecedoresSugeridos(a.Ctx)
	if err != nil || lista[planejada] != fornecedor || len(lista) != 1 {
		t.Fatalf("lista de fornecedores sugeridos: %v %v", lista, err)
	}
	// Gravado na ordem ao firmar: sobrevive ao recálculo, que apaga a sugestão.
	if err := r.GravarFornecedor(a.Ctx, planejada, fornecedor); err != nil {
		t.Fatal(err)
	}
	// B não grava na ordem da A; a A não grava fornecedor da B.
	if err := r.GravarFornecedor(b.Ctx, semFornecedor, fornecedorB); err != nil {
		t.Fatal(err)
	}
	if err := r.GravarFornecedor(a.Ctx, semFornecedor, fornecedorB); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := r.FornecedorSugerido(a.Ctx, semFornecedor); err != nil || ok {
		t.Fatalf("empresa B gravou fornecedor na ordem da A: %v %v", ok, err)
	}
	testutil.Exec(t, pool, `DELETE FROM mrp_planned_suggestions WHERE code=$1`, sugestao)
	if got, ok, err := r.FornecedorSugerido(a.Ctx, planejada); err != nil || !ok || got != fornecedor {
		t.Fatalf("fornecedor perdido depois do recálculo: %d %v %v", got, ok, err)
	}
	if lista, err := r.FornecedoresSugeridos(a.Ctx); err != nil || lista[planejada] != fornecedor {
		t.Fatalf("lista perdeu o fornecedor depois do recálculo: %v %v", lista, err)
	}

	// Empresa B não enxerga a sugestão da A.
	if _, ok, err := r.FornecedorSugerido(b.Ctx, planejada); err != nil || ok {
		t.Fatalf("empresa B leu o fornecedor sugerido da A: %v %v", ok, err)
	}
	if lb, err := r.FornecedoresSugeridos(b.Ctx); err != nil || len(lb) != 0 {
		t.Fatalf("empresa B listou sugestões da A: %v %v", lb, err)
	}
}
