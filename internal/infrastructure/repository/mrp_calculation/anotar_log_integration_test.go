//go:build integration

package mrp_calculation_test

import (
	"context"
	"testing"

	mrprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O aviso das estruturas sem conversão é apurado antes do cálculo e o log já
// foi gravado quando ele chega: a anotação junta a chave sem apagar o que o
// cálculo gravou, e a empresa B não anota o log da A.
func TestAnotarNoLogDoMRP(t *testing.T) {
	q, pool := testutil.Queries(t)
	base := context.Background()
	plano := int64(1_880_000_000 + testutil.UniqueCode()%10_000_000)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		_, _ = pool.Exec(base, `DELETE FROM production_plans WHERE code = $1`, plano)
	})
	actor := testutil.Actor(t, pool)
	if _, err := pool.Exec(base, `INSERT INTO production_plans(code,name,created_by,enterprise_id) VALUES($1,'Anotação',$2,$3)`, plano, actor, a.ID); err != nil {
		t.Fatal(err)
	}
	var log int64
	if err := pool.QueryRow(base, `INSERT INTO mrp_calculation_logs(plan_code,status,errors,enterprise_id,finished_at)
		VALUES($1,'COMPLETED','{"itens":"2 sem lead time"}',$2,NOW()) RETURNING code`, plano, a.ID).Scan(&log); err != nil {
		t.Fatal(err)
	}
	repo := mrprepo.NewMRPCalculationRepositorySQLC(q, pool)

	if err := repo.AnotarNoLog(b.Ctx, log, "estrutura_sem_conversao", []string{"invasão"}); err == nil {
		t.Fatal("a empresa B anotou o log da A")
	}
	aviso := []string{"estrutura do item RN-1: o componente TUBO é estocado em M e a linha está em KG"}
	if err := repo.AnotarNoLog(a.Ctx, log, "estrutura_sem_conversao", aviso); err != nil {
		t.Fatal(err)
	}
	var itens, estrutura string
	if err := pool.QueryRow(base, `SELECT errors->>'itens', errors->'estrutura_sem_conversao'->>0 FROM mrp_calculation_logs WHERE code=$1`, log).Scan(&itens, &estrutura); err != nil {
		t.Fatal(err)
	}
	if itens != "2 sem lead time" || estrutura != aviso[0] {
		t.Fatalf("log anotado: itens=%q estrutura=%q", itens, estrutura)
	}
}
