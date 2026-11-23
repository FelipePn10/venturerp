//go:build integration

package delivery_promise_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_promise"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Bug V: a promessa usava o prazo 0 do fornecedor no lugar dos 5 dias do item
// (o MRP usa o do item) e ignorava a etapa no terceiro.
func TestPromessaUsaOPrazoDoMRPEAEtapaNoTerceiro(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {})
	comprado, fabricado := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, comprado, fabricado)
	testutil.Exec(t, pool, `UPDATE items SET engineering_type=1, planning_purchase_lead_time_days=5 WHERE code=$1`, comprado)
	testutil.Exec(t, pool, `UPDATE items SET engineering_type=0, planning_manufacturing_lead_time_days=2 WHERE code=$1`, fabricado)
	testutil.FonteDeSuprimento(t, pool, comprado, 0) // fornecedor sem prazo (0)

	var operacao, roteiro int64
	if err := pool.QueryRow(base, `INSERT INTO operations(code,name,origin,lead_time_days,created_by,enterprise_id) VALUES($1,'Zincagem','TERCEIROS',7,$2,$3) RETURNING id`,
		testutil.UniqueCode(), ator, a.ID).Scan(&operacao); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO manufacturing_routes(code,item_code,alternative,is_standard,created_by,enterprise_id) VALUES($1,$2,1,TRUE,$3,$4) RETURNING id`,
		testutil.UniqueCode(), fabricado, ator, a.ID).Scan(&roteiro); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO route_operations(route_id,sequence,operation_id) VALUES($1,10,$2)`, roteiro, operacao)
	t.Cleanup(func() {
		pool.Exec(base, `DELETE FROM route_operations WHERE route_id=$1`, roteiro)
		pool.Exec(base, `DELETE FROM manufacturing_routes WHERE id=$1`, roteiro)
		pool.Exec(base, `DELETE FROM operations WHERE id=$1`, operacao)
	})

	repo := delivery_promise.New(pool)
	s, err := repo.SituacaoParaPromessa(a.Ctx, comprado)
	if err != nil || s.PrazoCompraDias != 5 {
		t.Fatalf("prazo de compra = %d (%v), quer os 5 dias do item", s.PrazoCompraDias, err)
	}
	s, err = repo.SituacaoParaPromessa(a.Ctx, fabricado)
	if err != nil || s.DiasNoTerceiro != 5 { // 7 corridos ≈ 5 úteis
		t.Fatalf("dias no terceiro = %d (%v), quer 5", s.DiasNoTerceiro, err)
	}
}
