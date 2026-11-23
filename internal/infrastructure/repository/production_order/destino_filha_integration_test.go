//go:build integration

package production_order_test

import (
	"context"
	"testing"
	"time"

	productionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Bug AB: a OF filha entrega no almoxarifado em que a OF pai baixa a peça.
func TestDestinoDaOFFilhaEOndeAPaiBaixa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, nil)
	pai, filho := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, actor, pai, filho)
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	hoje := time.Now()
	var paiPlan, filhoPlan, ofPai int64
	for _, p := range []struct {
		item int64
		dst  *int64
	}{{pai, &paiPlan}, {filho, &filhoPlan}} {
		if err := pool.QueryRow(base, `INSERT INTO planned_orders(order_number,item_code,quantity,order_type,demand_type,need_date,created_by,demand_code,enterprise_id,is_firm)
			VALUES($1,$2,10,'PRODUCTION','SALES_ORDER',$3::date,$4,1,$5,TRUE) RETURNING id`, testutil.UniqueCode()%1_000_000, p.item, hoje, actor, a.ID).Scan(p.dst); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(base, `INSERT INTO production_chain_links(enterprise_id,child_planned_order_id,parent_planned_order_id,quantity,created_by) VALUES($1,$2,$3,10,$4)`, a.ID, filhoPlan, paiPlan, actor); err != nil {
		t.Fatal(err)
	}
	repo := productionrepo.NewProductionOrderRepositoryPGX(pool)
	if got, err := repo.DestinoDaOFFilha(a.Ctx, filhoPlan, filho); err != nil || got != nil {
		t.Fatalf("sem OF pai ainda, sem destino: %v %v", got, err)
	}
	if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,start_date,created_by,enterprise_id,planned_order_id)
		VALUES($1,$2,10,'OPEN',$3::date,$4,$5,$6) RETURNING id`, testutil.UniqueCode()%1_000_000, pai, hoje, actor, a.ID, paiPlan).Scan(&ofPai); err != nil {
		t.Fatal(err)
	}
	// Linha do mesmo item em outra máscara e outro almoxarifado, gravada antes:
	// não é a que a filha (sem máscara) abastece.
	outroAlmox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	for _, l := range []struct {
		mask  string
		almox int64
	}{{"AZUL", outroAlmox}, {"", almox}} {
		if _, err := pool.Exec(base, `INSERT INTO production_order_materials(production_order_id,enterprise_id,material_kind,item_code,mask,quantity,attended_quantity,warehouse_id,created_by)
			VALUES($1,$2,'DEMAND',$3,$4,10,0,$5,$6)`, ofPai, a.ID, filho, l.mask, l.almox, actor); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		pool.Exec(base, `DELETE FROM production_order_materials WHERE production_order_id=$1`, ofPai)
		pool.Exec(base, `DELETE FROM production_orders WHERE id=$1`, ofPai)
		pool.Exec(base, `DELETE FROM production_chain_links WHERE child_planned_order_id=$1`, filhoPlan)
		pool.Exec(base, `DELETE FROM planned_orders WHERE id = ANY($1)`, []int64{paiPlan, filhoPlan})
	})
	if got, err := repo.DestinoDaOFFilha(a.Ctx, filhoPlan, filho); err != nil || got == nil || *got != almox {
		t.Fatalf("destino da filha = %v (%v), quer %d", got, err, almox)
	}
	if got, err := repo.DestinoDaOFFilha(b.Ctx, filhoPlan, filho); err != nil || got != nil {
		t.Fatalf("a empresa B leu o destino da cadeia da A: %v %v", got, err)
	}
}
