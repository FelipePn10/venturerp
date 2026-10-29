//go:build integration

package planning_scenario_test

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planning_scenario"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Promover segue as regras da VPLA0102: a demanda que já teve saída não é
// cancelada (encerra o saldo) nem fica com quantidade abaixo do que saiu.
func TestPromoverRespeitaOQueJaSaiu(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {})
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, actor)
	plano := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO production_plans(code,name,created_by,enterprise_id) VALUES($1,'Base',$2,$3)`, plano, actor, a.ID)
	dia := time.Now().AddDate(0, 0, 10)
	// Demanda 1 (amostra) com 3 de 5 já saídos; demanda 2 sem saída.
	testutil.Exec(t, pool, `INSERT INTO independent_demands(code,item_code,quantity,demand_date,created_by,enterprise_id,motivo) VALUES
		(1,$1,5,$2::date,$3,$4,'AMOSTRA'),(2,$1,4,$2::date,$3,$4,'AMOSTRA')`, item, dia, actor, a.ID)
	testutil.Exec(t, pool, `INSERT INTO stock_movements(item_code,warehouse_id,movement_type,quantity,created_by,enterprise_id,reference_type,reference_code)
		VALUES($1,1,'OUT',3,$2,$3,'DEMANDA_INDEPENDENTE',1)`, item, actor, a.ID)

	repo := planning_scenario.New(pool)
	ajuste := func(cenario, demanda int64, qtd int) {
		testutil.Exec(t, pool, `INSERT INTO planning_scenario_changes(enterprise_id,scenario_id,kind,demand_ref_kind,demand_ref_code,quantity,created_by)
			VALUES($1,$2,'DEMANDA','DEMANDA_INDEPENDENTE',$3,$4,$5)`, a.ID, cenario, demanda, qtd, actor)
	}
	situacao := func(code int) (s string, qtd int) {
		if err := pool.QueryRow(base, `SELECT situacao, quantity::int FROM independent_demands WHERE enterprise_id=$1 AND code=$2`, a.ID, code).Scan(&s, &qtd); err != nil {
			t.Fatal(err)
		}
		return
	}

	// Baixar a demanda 1 para 2 (saíram 3) é recusado, e nada da promoção fica.
	c1, err := repo.Create(a.Ctx, "Abaixo do atendido", "", plano, actor)
	if err != nil {
		t.Fatal(err)
	}
	ajuste(c1.ID, 2, 0)
	ajuste(c1.ID, 1, 2)
	if _, err := repo.Promote(a.Ctx, c1.ID, actor); err == nil {
		t.Fatal("promoveu a quantidade abaixo do que já saiu")
	}
	if s, q := situacao(1); s != "ABERTA" || q != 5 {
		t.Fatalf("demanda 1 mudou: %s %d", s, q)
	}
	if s, _ := situacao(2); s != "ABERTA" {
		t.Fatalf("a promoção recusada deixou a demanda 2 %s", s)
	}

	// Zerar: a demanda com saída encerra; a sem saída cancela.
	c2, err := repo.Create(a.Ctx, "Remove as duas", "", plano, actor)
	if err != nil {
		t.Fatal(err)
	}
	ajuste(c2.ID, 1, 0)
	ajuste(c2.ID, 2, 0)
	if res, err := repo.Promote(a.Ctx, c2.ID, actor); err != nil || res.Removed != 2 {
		t.Fatalf("promoção: %+v %v", res, err)
	}
	if s, _ := situacao(1); s != "ENCERRADA" {
		t.Fatalf("demanda com saída: %s, esperado ENCERRADA", s)
	}
	if s, _ := situacao(2); s != "CANCELADA" {
		t.Fatalf("demanda sem saída: %s, esperado CANCELADA", s)
	}
}
