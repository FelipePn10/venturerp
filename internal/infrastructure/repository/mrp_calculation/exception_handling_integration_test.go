//go:build integration

package mrp_calculation

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	plannedrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planned_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A exceção guarda a data sugerida, lista a ordem de origem e registra o
// tratamento; reprogramar desloca início e fim junto; cancelar tira a ordem do
// suprimento; a OF gerada é achada pelo ID interno da planejada. Tudo por empresa.
func TestTratamentoDeExcecaoNoBanco(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM mrp_exception_messages WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM production_orders WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM planned_orders WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
	})
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, actor)
	hoje := time.Now()
	nova := func() (id, code int64) {
		if err := pool.QueryRow(base, `INSERT INTO planned_orders(order_number,item_code,quantity,order_type,status,demand_type,demand_code,need_date,start_date,end_date,created_by,enterprise_id,is_active,is_firm)
			VALUES($1,$2,1,'PRODUCTION','RELEASED','INDEPENDENT',0,$3::date,$3::date-3,$3::date,$4,$5,true,true) RETURNING id, code`,
			testutil.UniqueCode(), item, hoje.AddDate(0, 0, 10), actor, a.ID).Scan(&id, &code); err != nil {
			t.Fatal(err)
		}
		return
	}
	_, ordem := nova()
	idComOF, comOF := nova()

	repo := NewMRPCalculationRepositorySQLC(sqlc.New(pool), pool)
	plano := testutil.UniqueCode()
	sugerida := time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 4)
	origem := "PLANNED_ORDER"
	if err := repo.CreateExceptionMessage(a.Ctx, &entity.MRPExceptionMessage{PlanCode: plano, ItemCode: item, MessageType: entity.ExceptionRescheduleIn,
		SourceCode: &ordem, SourceType: &origem, Description: "antecipar", SuggestedDate: &sugerida}); err != nil {
		t.Fatal(err)
	}
	lista, err := repo.ListExceptionsByPlan(a.Ctx, plano)
	if err != nil || len(lista) != 1 || lista[0].SuggestedDate == nil || lista[0].SourceOrderNumber == nil || lista[0].HandledAt != nil {
		t.Fatalf("exceção listada: %+v (%v)", lista, err)
	}
	if outra, _ := repo.ListExceptionsByPlan(b.Ctx, plano); len(outra) != 0 {
		t.Fatal("empresa B vê a exceção da A")
	}
	if _, err := repo.GetException(b.Ctx, lista[0].Code); err == nil {
		t.Fatal("empresa B lê a exceção da A")
	}

	pr := plannedrepo.NewPlannedOrderRepositorySQLC(sqlc.New(pool))
	if _, err := pr.Reschedule(b.Ctx, ordem, sugerida); err == nil {
		t.Fatal("empresa B reprogramou a ordem da A")
	}
	o, err := pr.Reschedule(a.Ctx, ordem, sugerida)
	if err != nil || !o.NeedDate.Equal(sugerida) || o.StartDate == nil || !o.StartDate.Equal(sugerida.AddDate(0, 0, -3)) {
		t.Fatalf("reprogramação: %+v (%v)", o, err)
	}
	marcada, err := repo.MarkExceptionHandled(a.Ctx, lista[0].Code, entity.ExceptionHandledRescheduled, nil, actor)
	if err != nil || marcada.HandledAt == nil || marcada.HandlingAction != entity.ExceptionHandledRescheduled {
		t.Fatalf("marcação: %+v (%v)", marcada, err)
	}
	// Segunda marcação (dois cliques ao mesmo tempo): conflito, sem sobrescrever.
	if _, err := repo.MarkExceptionHandled(a.Ctx, lista[0].Code, entity.ExceptionHandledAcknowledged, nil, actor); err == nil {
		t.Fatal("marcou duas vezes a mesma exceção")
	}
	if de, _ := repo.GetException(a.Ctx, lista[0].Code); de.HandlingAction != entity.ExceptionHandledRescheduled {
		t.Fatalf("a segunda marcação sobrescreveu a primeira: %+v", de)
	}

	testutil.Exec(t, pool, `INSERT INTO production_orders(order_number,item_code,planned_qty,created_by,enterprise_id,planned_order_id) VALUES($1,$2,1,$3,$4,$5)`,
		testutil.UniqueCode(), item, actor, a.ID, idComOF)
	if n, err := pr.ActiveProductionOrderNumber(a.Ctx, comOF); err != nil || n == nil {
		t.Fatalf("OF da planejada não encontrada: %v %v", n, err)
	}
	if n, err := pr.ActiveProductionOrderNumber(a.Ctx, ordem); err != nil || n != nil {
		t.Fatalf("planejada sem OF apontou OF: %v %v", n, err)
	}
	if _, err := pr.Cancel(a.Ctx, ordem); err != nil {
		t.Fatal(err)
	}
	if _, err := pr.Cancel(a.Ctx, ordem); err == nil {
		t.Fatal("cancelou duas vezes")
	}
}
