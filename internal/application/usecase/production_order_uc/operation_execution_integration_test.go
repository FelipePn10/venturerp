//go:build integration

package production_order_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

func TestOperationExecutionLifecycleIntegration(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var enterprise int64
	if err := tx.QueryRow(ctx, "SELECT MIN(id) FROM enterprise").Scan(&enterprise); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterprise})
	q := sqlc.New(tx)
	code := testutil.UniqueCode()
	user := uuid.New()
	if _, err := tx.Exec(ctx, "INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,$2,$1,$3,$4)", code, strings.ToUpper(uuid.NewString()), user, enterprise); err != nil {
		t.Fatal(err)
	}
	var order int64
	if err := tx.QueryRow(ctx, "INSERT INTO production_orders(order_number,item_code,planned_qty,status,created_by,enterprise_id) VALUES($1,$1,10,'IN_PROGRESS',$2,$3) RETURNING id", code, user, enterprise).Scan(&order); err != nil {
		t.Fatal(err)
	}
	var first, second int64
	for i, id := range []*int64{&first, &second} {
		if err := tx.QueryRow(ctx, "INSERT INTO production_order_operations(production_order_id,sequence,operation_name,enterprise_id) VALUES($1,$2,'Etapa de teste',$3) RETURNING id", order, (i+1)*10, enterprise).Scan(id); err != nil {
			t.Fatal(err)
		}
	}
	uc := OrderOperationsUseCase{Q: q, Auth: scannerAuth{}, WithinTransaction: func(ctx context.Context, fn func(*sqlc.Queries) error) error {
		nested, err := tx.Begin(ctx)
		if err != nil {
			return err
		}
		defer nested.Rollback(ctx)
		if err := fn(sqlc.New(nested)); err != nil {
			return err
		}
		return nested.Commit(ctx)
	}}
	advance := func(id int64, status, reason string, want bool) {
		t.Helper()
		_, err := uc.AdvanceOperation(ctx, request.AdvanceOperationDTO{OperationID: id, Status: status, Reason: reason})
		if (err == nil) != want {
			t.Fatalf("%d %s success=%v: %v", id, status, want, err)
		}
	}
	advance(second, "IN_PROGRESS", "", false)
	advance(first, "DONE", "", false)
	advance(first, "IN_PROGRESS", "", true)
	before, err := q.GetProductionOrderOperation(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	advance(first, "PAUSED", "", false)
	advance(first, "PAUSED", "intervalo", true)
	advance(first, "DONE", "", false)
	advance(first, "IN_PROGRESS", "", true)
	advance(first, "INTERRUPTED", "manutenção", true)
	advance(first, "IN_PROGRESS", "", true)
	after, err := q.GetProductionOrderOperation(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if !before.StartedAt.Time.Equal(after.StartedAt.Time) {
		t.Fatal("retomar alterou o início original")
	}
	advance(first, "DONE", "", true)
	advance(first, "DONE", "", false)
	advance(first, "IN_PROGRESS", "", false)
	list, err := uc.ListOperations(ctx, order)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || !list[1].CanStart || list[0].CanStart {
		t.Fatalf("liberação incorreta: %+v", list)
	}
	pending, err := q.ProductionOperationsPending(ctx, order)
	if err != nil || !pending {
		t.Fatalf("pendência não detectada: %v", err)
	}
	advance(second, "IN_PROGRESS", "", true)
	advance(second, "DONE", "", true)
	pending, err = q.ProductionOperationsPending(ctx, order)
	if err != nil || pending {
		t.Fatalf("roteiro não concluiu: %v", err)
	}
	foreign := context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterprise + 999999})
	if _, err := uc.AdvanceOperation(foreign, request.AdvanceOperationDTO{OperationID: first, Status: "IN_PROGRESS"}); err == nil {
		t.Fatal("acesso entre empresas permitido")
	}
	if _, err := uc.ListOperations(foreign, order); err == nil {
		t.Fatal("leitura entre empresas permitida")
	}
}

// Cronômetro da etapa: sem horas digitadas, concluir grava o tempo corrido
// desde a última entrada em andamento e as peças da ordem; digitar continua
// valendo. A tela mostra o "em andamento desde" do trecho atual.
func TestCronometroDaEtapaIntegration(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var enterprise int64
	if err := tx.QueryRow(ctx, "SELECT MIN(id) FROM enterprise").Scan(&enterprise); err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterprise})
	q := sqlc.New(tx)
	code := testutil.UniqueCode()
	user := uuid.New()
	if _, err := tx.Exec(ctx, "INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,$2,$1,$3,$4)", code, strings.ToUpper(uuid.NewString()), user, enterprise); err != nil {
		t.Fatal(err)
	}
	var order int64
	if err := tx.QueryRow(ctx, "INSERT INTO production_orders(order_number,item_code,planned_qty,status,created_by,enterprise_id) VALUES($1,$1,12,'IN_PROGRESS',$2,$3) RETURNING id", code, user, enterprise).Scan(&order); err != nil {
		t.Fatal(err)
	}
	var corte, solda int64
	for i, id := range []*int64{&corte, &solda} {
		if err := tx.QueryRow(ctx, "INSERT INTO production_order_operations(production_order_id,sequence,operation_name,enterprise_id) VALUES($1,$2,'Etapa cronometrada',$3) RETURNING id", order, (i+1)*10, enterprise).Scan(id); err != nil {
			t.Fatal(err)
		}
	}
	uc := OrderOperationsUseCase{Q: q, Auth: scannerAuth{}, WithinTransaction: func(ctx context.Context, fn func(*sqlc.Queries) error) error {
		nested, err := tx.Begin(ctx)
		if err != nil {
			return err
		}
		defer nested.Rollback(ctx)
		if err := fn(sqlc.New(nested)); err != nil {
			return err
		}
		return nested.Commit(ctx)
	}}
	if _, err := uc.AdvanceOperation(ctx, request.AdvanceOperationDTO{OperationID: corte, Status: "IN_PROGRESS"}); err != nil {
		t.Fatal(err)
	}
	// O posto começou há 90 minutos (a trilha é imutável: o evento de início entra como nova linha).
	if _, err := tx.Exec(ctx, `INSERT INTO production_operation_execution_events(enterprise_id,production_order_id,operation_id,old_status,new_status,actual_hours_delta,source,occurred_at)
		VALUES($1,$2,$3,'PAUSED','IN_PROGRESS',0,'TESTE',clock_timestamp() - interval '90 minutes')`, enterprise, order, corte); err != nil {
		t.Fatal(err)
	}
	lista, err := uc.ListOperations(ctx, order)
	if err != nil {
		t.Fatal(err)
	}
	if lista[0].RunningSince == nil || time.Since(*lista[0].RunningSince) < 89*time.Minute {
		t.Fatalf("a etapa em andamento diz desde quando: %+v", lista[0].RunningSince)
	}
	r, err := uc.AdvanceOperation(ctx, request.AdvanceOperationDTO{OperationID: corte, Status: "DONE"})
	if err != nil {
		t.Fatal(err)
	}
	if !r.HoursFromClock || r.RecordedHours < 1.49 || r.RecordedHours > 1.6 {
		t.Fatalf("concluir sem digitar grava o tempo corrido (1,5 h): %+v", r)
	}
	if !r.QtyProposed || r.RecordedQty != 12 {
		t.Fatalf("concluir sem digitar peças usa a quantidade da ordem (12): %+v", r)
	}
	var horas float64
	if err := tx.QueryRow(ctx, "SELECT actual_hours::float8 FROM production_order_operations WHERE id=$1", corte).Scan(&horas); err != nil || horas < 1.49 || horas > 1.6 {
		t.Fatalf("horas gravadas na etapa: %v %v", horas, err)
	}
	// Digitado vale e corrige o relógio.
	if _, err := uc.AdvanceOperation(ctx, request.AdvanceOperationDTO{OperationID: solda, Status: "IN_PROGRESS"}); err != nil {
		t.Fatal(err)
	}
	r, err = uc.AdvanceOperation(ctx, request.AdvanceOperationDTO{OperationID: solda, Status: "DONE", ActualHours: 0.25, ProducedQty: 11})
	if err != nil {
		t.Fatal(err)
	}
	if r.HoursFromClock || r.RecordedHours != 0.25 || r.QtyProposed || r.RecordedQty != 11 {
		t.Fatalf("o digitado vale: %+v", r)
	}
	// Etapa esquecida em andamento (15 h): sem horas é recusada; com horas, conclui.
	var esquecida int64
	if err := tx.QueryRow(ctx, "INSERT INTO production_order_operations(production_order_id,sequence,operation_name,status,started_at,enterprise_id) VALUES($1,30,'Esquecida','IN_PROGRESS',clock_timestamp() - interval '15 hours',$2) RETURNING id", order, enterprise).Scan(&esquecida); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.AdvanceOperation(ctx, request.AdvanceOperationDTO{OperationID: esquecida, Status: "DONE"}); err == nil || !strings.Contains(err.Error(), "ficou aberta") {
		t.Fatalf("etapa aberta há 15 h deveria pedir as horas: %v", err)
	}
	if r, err = uc.AdvanceOperation(ctx, request.AdvanceOperationDTO{OperationID: esquecida, Status: "DONE", ActualHours: 7.5}); err != nil || r.RecordedHours != 7.5 {
		t.Fatalf("com horas digitadas conclui: %+v %v", r, err)
	}
	foreign := context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterprise + 999999})
	if _, err := uc.ListOperations(foreign, order); err == nil {
		t.Fatal("leitura do cronômetro entre empresas permitida")
	}
}
