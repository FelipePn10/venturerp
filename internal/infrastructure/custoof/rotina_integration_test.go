//go:build integration

package custoof_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/custoof"
	productionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

type logMudo struct{ erros []string }

func (l *logMudo) Info(string, ...any)           {}
func (l *logMudo) Error(msg string, args ...any) { l.erros = append(l.erros, msg) }

// TestRotinaApuraOCustoDasOFsAntigas: OF concluída com horas de etapa e sem
// custo (movida antes do custo automático) sai da rotina com o custo real
// gravado, na empresa dela; rodar de novo não refaz nada.
func TestRotinaApuraOCustoDasOFsAntigas(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, nil)
	itA, itB := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, actor, itA)
	testutil.NovosItens(t, pool, b.ID, actor, itB)
	of := func(emp, item int64) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,produced_qty,status,created_by,enterprise_id)
			VALUES($1,$2,10,10,'COMPLETED',$3,$4) RETURNING id`, testutil.UniqueCode()%1_000_000, item, actor, emp).Scan(&id); err != nil {
			t.Fatal(err)
		}
		testutil.Exec(t, pool, `INSERT INTO production_order_operations(production_order_id,sequence,operation_name,actual_hours,enterprise_id) VALUES($1,10,'Corte',2,$2)`, id, emp)
		t.Cleanup(func() {
			testutil.Exec(t, pool, `DELETE FROM production_order_costs WHERE production_order_id = $1`, id)
			testutil.Exec(t, pool, `DELETE FROM production_orders WHERE id = $1`, id)
		})
		return id
	}
	ofA, ofB := of(a.ID, itA), of(b.ID, itB)
	repo := productionrepo.NewProductionOrderRepositoryPGX(pool)
	log := &logMudo{}
	rotina := custoof.NewRotina(repo, &production_order_uc.SettleProductionCostUseCase{Repo: repo, Auth: &infraauth.AuthService{}}, log)
	if n := rotina.Run(base); n < 2 {
		t.Fatalf("a rotina apurou %d OF(s); esperava pelo menos as 2 do teste (erros: %v)", n, log.erros)
	}
	for _, id := range []int64{ofA, ofB} {
		var qtd float64
		if err := pool.QueryRow(base, `SELECT produced_qty::float8 FROM production_order_costs WHERE production_order_id = $1`, id).Scan(&qtd); err != nil || qtd != 10 {
			t.Fatalf("OF %d sem custo depois da rotina: %v %v", id, qtd, err)
		}
	}
	fila, err := repo.OFsSemCusto(base, 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fila {
		if f.ID == ofA || f.ID == ofB {
			t.Fatalf("OF %d apurada continua na fila", f.ID)
		}
	}
}
