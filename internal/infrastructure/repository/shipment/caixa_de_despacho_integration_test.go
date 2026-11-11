//go:build integration

package shipment_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	shipmentrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/shipment"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A caixa de despacho é cadastro da empresa: cada uma tem a sua DOCA-1, e
// gravar de novo o mesmo código atualiza a caixa em vez de cair em erro
// interno (o ON CONFLICT pedia uma unicidade por empresa que não existia).
func TestCaixaDeDespachoPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		pool.Exec(base, `DELETE FROM shipment_dispatch_boxes WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
	})
	repo := shipmentrepo.NewShipmentRepositoryPG(pool)
	desc := func(s string) *string { return &s }

	for _, e := range []testutil.Empresa{a, b} {
		if _, err := repo.CreateDispatchBox(e.Ctx, &entity.DispatchBox{Code: "DOCA-1", Description: desc("Doca"), Active: true}); err != nil {
			t.Fatalf("empresa %d não cadastrou a DOCA-1: %v", e.Code, err)
		}
	}
	if _, err := repo.CreateDispatchBox(a.Ctx, &entity.DispatchBox{Code: "DOCA-1", Description: desc("Doca norte"), Active: true}); err != nil {
		t.Fatalf("regravar a caixa deveria atualizar: %v", err)
	}
	caixasA, err := repo.ListDispatchBoxes(a.Ctx, false)
	if err != nil || len(caixasA) != 1 || caixasA[0].Description == nil || *caixasA[0].Description != "Doca norte" {
		t.Fatalf("empresa A deveria ter 1 caixa atualizada: %+v err=%v", caixasA, err)
	}
	caixasB, err := repo.ListDispatchBoxes(b.Ctx, false)
	if err != nil || len(caixasB) != 1 || *caixasB[0].Description != "Doca" {
		t.Fatalf("a alteração da A vazou para a B: %+v err=%v", caixasB, err)
	}
}
