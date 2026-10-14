//go:build integration

package mrp_calculation

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O pegging é trocado inteiro a cada cálculo e é da empresa: outra empresa não
// lê o da primeira nem com o mesmo código de plano.
func TestPeggingGravadoPorPlanoEEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM mrp_pegging WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM mrp_planned_suggestions WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
	})
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, actor)
	plano := testutil.UniqueCode()
	var sug int64
	if err := pool.QueryRow(base, `INSERT INTO mrp_planned_suggestions(plan_code,item_code,quantity,need_date,start_date,order_type,enterprise_id,order_number) VALUES($1,$2,7,$3::date,$3::date,'COMPRA',$4,$5) RETURNING code`,
		plano, item, time.Now(), a.ID, testutil.UniqueCode()).Scan(&sug); err != nil {
		t.Fatal(err)
	}
	repo := NewMRPCalculationRepositorySQLC(sqlc.New(pool), pool)
	dia := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	demanda := int64(55)
	grava := func(n int) {
		recs := []entity.PeggingRecord{}
		for i := 0; i < n; i++ {
			recs = append(recs, entity.PeggingRecord{ItemCode: item, DemandKind: entity.PegDemandIndependent, DemandCode: &demanda, NeedDate: dia,
				SupplyKind: entity.PegSupplySuggestion, SupplyCode: &sug, SupplyDate: &dia, Quantity: 3.5})
		}
		if err := repo.ReplacePegging(a.Ctx, plano, recs); err != nil {
			t.Fatal(err)
		}
	}
	grava(3)
	grava(2) // o segundo cálculo substitui o primeiro
	rows, err := repo.PeggingRows(a.Ctx, plano)
	if err != nil || len(rows) != 2 || rows[0].Quantity != 3.5 || *rows[0].SupplyCode != sug || !rows[0].NeedDate.Equal(dia) {
		t.Fatalf("pegging lido: %+v (%v)", rows, err)
	}
	if outra, err := repo.PeggingRows(b.Ctx, plano); err != nil || len(outra) != 0 {
		t.Fatalf("empresa B lê o pegging da A: %+v (%v)", outra, err)
	}
	labels, err := repo.PeggingLabels(a.Ctx, entity.PeggingRefs{Suggestions: []int64{sug}, Items: []int64{item}})
	if err != nil || labels.Suggestions[sug].OrderNumber == nil || labels.Suggestions[sug].Quantity != 7 || labels.Items[item].BusinessCode == "" {
		t.Fatalf("rótulos: %+v (%v)", labels, err)
	}
	if outros, err := repo.PeggingLabels(b.Ctx, entity.PeggingRefs{Suggestions: []int64{sug}, Items: []int64{item}}); err != nil || len(outros.Suggestions) != 0 || len(outros.Items) != 0 {
		t.Fatalf("empresa B lê rótulos da A: %+v (%v)", outros, err)
	}
	if cen, err := repo.IsScenarioPlan(a.Ctx, plano); err != nil || cen {
		t.Fatalf("plano comum marcado como cenário: %v %v", cen, err)
	}
}
