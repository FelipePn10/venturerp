//go:build integration

package indicadores_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/indicadores"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O OEE é da máquina da empresa: a B não vê o apontamento da A.
func TestOEEPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		pool.Exec(base, `DELETE FROM production_appointments WHERE production_order_id IN (SELECT id FROM production_orders WHERE enterprise_id=$1)`, a.ID)
		pool.Exec(base, `DELETE FROM production_order_operations WHERE enterprise_id=$1`, a.ID)
		pool.Exec(base, `DELETE FROM production_orders WHERE enterprise_id=$1`, a.ID)
		pool.Exec(base, `DELETE FROM machines WHERE enterprise_id=$1`, a.ID)
		pool.Exec(base, `DELETE FROM machine_types WHERE enterprise_id=$1`, a.ID)
	})
	item := testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, item)
	var maquina, ordem int64
	tipo := testutil.UniqueCode() % 1_000_000
	testutil.Exec(t, pool, `INSERT INTO machine_types(code,name,type,created_by,enterprise_id) VALUES($1,'Dobra',(enum_range(NULL::machine_type_enum))[1],$2,$3)`, tipo, ator, a.ID)
	if err := pool.QueryRow(base, `INSERT INTO machines(code,name,machine_type_code,created_by,enterprise_id,available_hours_per_day,efficiency_rate) VALUES($1,'Dobradeira',$4,$2,$3,8,1) RETURNING id`,
		testutil.UniqueCode()%1_000_000, ator, a.ID, tipo).Scan(&maquina); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,created_by,enterprise_id) VALUES($1,$2,10,'IN_PROGRESS',$3,$4) RETURNING id`,
		testutil.UniqueCode()%1_000_000, item, ator, a.ID).Scan(&ordem); err != nil {
		t.Fatal(err)
	}
	hoje := time.Now()
	// Etapa com 8 h previstas para as 10 peças da OF: 0,8 h/peça.
	var etapa int64
	if err := pool.QueryRow(base, `INSERT INTO production_order_operations(production_order_id,sequence,operation_name,planned_hours,enterprise_id) VALUES($1,10,'Dobra',8,$2) RETURNING id`, ordem, a.ID).Scan(&etapa); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO production_appointments(production_order_id,operation_id,machine_id,appointment_date,start_time,end_time,produced_qty,scrapped_qty,created_by)
		VALUES($1,$5,$2,$3::date,'08:00','12:00',4,1,$4)`, ordem, maquina, hoje, ator, etapa)
	repo := indicadores.New(pool)
	oa, err := repo.OEE(a.Ctx, hoje.AddDate(0, 0, -1), hoje)
	if err != nil {
		t.Fatal(err)
	}
	// 5 peças × 0,8 h = 4 h padrão em 4 h apontadas: 100%.
	if len(oa) != 1 || !oa[0].HorasApontadas.Equal(decimal.NewFromInt(4)) || !oa[0].Boas.Equal(decimal.NewFromInt(4)) ||
		oa[0].Desempenho == nil || !oa[0].Desempenho.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("OEE da A: %+v", oa)
	}
	if ob, err := repo.OEE(b.Ctx, hoje.AddDate(0, 0, -1), hoje); err != nil || len(ob) != 0 {
		t.Fatalf("a empresa B viu a máquina da A: %+v err=%v", ob, err)
	}
	for _, e := range []testutil.Empresa{a, b} {
		if _, err := repo.Giro(e.Ctx, hoje.AddDate(0, -1, 0), hoje); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.AcertoDaPromessa(e.Ctx, hoje.AddDate(0, -1, 0), hoje); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.PontualidadeFornecedores(e.Ctx, hoje.AddDate(0, -1, 0), hoje); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Vendas(e.Ctx, hoje.AddDate(0, -1, 0), hoje); err != nil {
			t.Fatal(err)
		}
	}
}
