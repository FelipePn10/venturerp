//go:build integration

package mrp_calculation

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/shopspring/decimal"
)

// Lote em quarentena está no saldo, mas não abastece a necessidade: sem descontar,
// o MRP deixava de comprar contando com material que não pode sair.
func TestLoteRetidoNaoContaParaOMRP(t *testing.T) {
	pool := testutil.Pool(t)
	ator := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, q := range []string{
			`DELETE FROM stock_lot_balances WHERE enterprise_id=$1`, `DELETE FROM stock_lots WHERE enterprise_id=$1`,
			`DELETE FROM stock_lot_statuses WHERE enterprise_id=$1`, `DELETE FROM stock_balances WHERE enterprise_id=$1`,
			`DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`,
		} {
			testutil.Exec(t, pool, q, a.ID)
		}
	})
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	item := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO items(code,warehouse_code,created_by,enterprise_id) VALUES($1,$2,$3,$4)`, item, almox, ator, a.ID)
	testutil.Exec(t, pool, `INSERT INTO stock_lot_statuses(enterprise_id,code,description,allows_issue,allows_transfer,counts_available)
		VALUES($1,'QUAR','Quarentena',false,true,false)`, a.ID)
	testutil.Exec(t, pool, `INSERT INTO stock_lots(enterprise_id,item_code,lot,status_code,created_by) VALUES($1,$2,'L1','QUAR',$3)`, a.ID, item, ator)
	testutil.Exec(t, pool, `INSERT INTO stock_balances(enterprise_id,item_code,warehouse_id,mask,quantity) VALUES($1,$2,$3,'',10)`, a.ID, item, almox)
	testutil.Exec(t, pool, `INSERT INTO stock_lot_balances(enterprise_id,item_code,warehouse_id,mask,lot,quantity) VALUES($1,$2,$3,'','L1',4)`, a.ID, item, almox)

	repo := NewMRPCalculationRepositorySQLC(sqlc.New(pool), pool)
	todos, err := repo.ListAllStockSnapshots(a.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	if s := todos[item]; s == nil || !s.Quantity.Equal(decimal.NewFromInt(6)) {
		t.Fatalf("10 no saldo, 4 retidos: sobram 6. %+v", s)
	}
	um, err := repo.GetStockSnapshot(a.Ctx, item)
	if err != nil || um == nil || !um.Quantity.Equal(decimal.NewFromInt(6)) {
		t.Fatalf("a consulta do item segue a mesma regra: %+v %v", um, err)
	}
}
