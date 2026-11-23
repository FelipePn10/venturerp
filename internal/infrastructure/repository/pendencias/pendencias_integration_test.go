//go:build integration

package pendencias_test

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/pendencias"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func TestCentralDePendenciasPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		pool.Exec(base, `DELETE FROM production_orders WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
	})
	item := testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, item)
	// Datas pelo relógio do processo (AGENTS.md: dois relógios).
	vencida := time.Now().AddDate(0, 0, -5)
	testutil.Exec(t, pool, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,start_date,end_date,created_by,enterprise_id)
		VALUES($1,$2,10,'OPEN',$3::date,$3::date,$4,$5)`, testutil.UniqueCode()%1_000_000, item, vencida, ator, a.ID)
	repo := pendencias.New(pool)
	ca, err := repo.Contagens(a.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ca["ofs_atrasadas"].Quantidade != 1 {
		t.Fatalf("empresa A: %+v", ca["ofs_atrasadas"])
	}
	// "mais antiga de" é dia puro: o texto do JSON tem de trazer o mesmo dia,
	// sem virar 21h da véspera (banco em UTC, API em São Paulo).
	if j, _ := ca["ofs_atrasadas"].MaisAntiga.MarshalJSON(); string(j[1:11]) != vencida.Format("2006-01-02") {
		t.Fatalf("mais antiga = %s, quer %s", j, vencida.Format("2006-01-02"))
	}
	cb, err := repo.Contagens(b.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cb["ofs_atrasadas"].Quantidade != 0 {
		t.Fatalf("a empresa B viu a OF atrasada da A: %+v", cb["ofs_atrasadas"])
	}
}
