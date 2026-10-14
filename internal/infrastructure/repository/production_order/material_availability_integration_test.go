//go:build integration

package production_order_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	productionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O saldo livre é do depósito do material, menos o reservado; o comprometido é
// o que falta baixar nas OFs abertas que estão na frente da fila. Outra empresa
// não entra em nenhuma das contas.
func TestDisponibilidadeDeMaterialDaOF(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM production_order_materials WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM production_orders WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM stock_balances WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
	})
	chapa := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, chapa, actor)
	hoje := time.Now()
	of := func(e testutil.Empresa, status string, inicio time.Time) int64 {
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,start_date,created_by,enterprise_id)
			VALUES($1,$2,1,$3,$4::date,$5,$6) RETURNING id`, testutil.UniqueCode(), chapa, status, inicio, actor, e.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	material := func(e testutil.Empresa, ofID int64, qtd, baixado int64, deposito int64) {
		testutil.Exec(t, pool, `INSERT INTO production_order_materials(production_order_id,enterprise_id,material_kind,item_code,mask,quantity,attended_quantity,warehouse_id,created_by)
			VALUES($1,$2,'DEMAND',$3,'',$4,$5,$6,$7)`, ofID, e.ID, chapa, qtd, baixado, deposito, actor)
	}
	anterior := of(a, "OPEN", hoje)
	material(a, anterior, 12, 2, 1) // falta baixar 10
	alvo := of(a, "OPEN", hoje.AddDate(0, 0, 3))
	material(a, alvo, 15, 0, 1)
	posterior := of(a, "OPEN", hoje.AddDate(0, 0, 9))
	material(a, posterior, 50, 0, 1) // atrás na fila: não conta
	cancelada := of(a, "CANCELLED", hoje.AddDate(0, 0, -5))
	material(a, cancelada, 50, 0, 1) // cancelada: não conta
	// A outra empresa com o MESMO item é dado legado cruzado (o banco recusa
	// gravar isso hoje): prova que a consulta não o deixa vazar.
	var outra int64
	if err := pool.QueryRow(base, `SELECT nextval(pg_get_serial_sequence('production_orders','id'))`).Scan(&outra); err != nil {
		t.Fatal(err)
	}
	testutil.DadoLegadoCruzado(t, pool, "production_orders", `INSERT INTO production_orders(id,order_number,item_code,planned_qty,status,start_date,created_by,enterprise_id)
		VALUES($1,$2,$3,1,'OPEN',$4::date,$5,$6)`, outra, testutil.UniqueCode(), chapa, hoje.AddDate(0, 0, -5), actor, b.ID)
	testutil.DadoLegadoCruzado(t, pool, "production_order_materials", `INSERT INTO production_order_materials(production_order_id,enterprise_id,material_kind,item_code,mask,quantity,attended_quantity,warehouse_id,created_by)
		VALUES($1,$2,'DEMAND',$3,'',50,0,1,$4)`, outra, b.ID, chapa, actor) // outra empresa: não conta
	testutil.Exec(t, pool, `INSERT INTO stock_balances(item_code,warehouse_id,quantity,reserved_qty,enterprise_id) VALUES($1,1,30,4,$2),($1,2,99,0,$2)`, chapa, a.ID)
	testutil.DadoLegadoCruzado(t, pool, "stock_balances", `INSERT INTO stock_balances(item_code,warehouse_id,quantity,reserved_qty,enterprise_id) VALUES($1,1,500,0,$2)`, chapa, b.ID)

	repo := productionrepo.NewProductionOrderRepositoryPGX(pool)
	linhas, err := repo.MaterialAvailability(a.Ctx, []int64{alvo, outra})
	if err != nil {
		t.Fatal(err)
	}
	if len(linhas) != 1 {
		t.Fatalf("linhas = %d, esperado 1 (a OF da outra empresa não pode aparecer): %+v", len(linhas), linhas)
	}
	l := linhas[0]
	if !l.FreeStock.Equal(decimal.NewFromInt(26)) {
		t.Fatalf("saldo livre = %s, esperado 26 (depósito 1, menos o reservado)", l.FreeStock)
	}
	if !l.CommittedEarlier.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("comprometido = %s, esperado 10 (só a OF anterior aberta da mesma empresa)", l.CommittedEarlier)
	}
}
