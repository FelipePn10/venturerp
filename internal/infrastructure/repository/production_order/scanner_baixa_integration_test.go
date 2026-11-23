//go:build integration

package production_order_test

import (
	"context"
	"crypto/sha256"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	productionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	stockpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Melhoria 7 (achado): a leitura APONTAR do coletor só gravava o apontamento.
// Agora baixa o material automático na mesma transação, grava o consumo e
// devolve ao operador o que baixou.
func TestLeituraApontaEBaixaOMaterial(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	user := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, nil)
	produto, chapa := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, user, produto, chapa)
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	var ordem int64
	if err := pool.QueryRow(base, "INSERT INTO production_orders(order_number,item_code,planned_qty,status,created_by,enterprise_id) VALUES($1,$2,10,'IN_PROGRESS',$3,$4) RETURNING id",
		testutil.UniqueCode()%1_000_000, produto, user, a.ID).Scan(&ordem); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO production_order_materials(production_order_id,enterprise_id,material_kind,item_code,mask,quantity,warehouse_id,automatic_issue,created_by)
		VALUES ($1,$2,'DEMAND',$3,'',20,$4,TRUE,$5)`, ordem, a.ID, chapa, almox, user)
	device := "baixa-" + uuid.NewString()
	t.Cleanup(func() {
		pool.Exec(base, "DELETE FROM production_scan_events WHERE device_id=$1", device)
		pool.Exec(base, "DELETE FROM production_consumptions WHERE production_order_id=$1", ordem)
		pool.Exec(base, "DELETE FROM production_appointments WHERE production_order_id=$1", ordem)
		pool.Exec(base, "DELETE FROM production_order_materials WHERE production_order_id=$1", ordem)
		pool.Exec(base, "DELETE FROM stock_movements WHERE enterprise_id=$1", a.ID)
		pool.Exec(base, "DELETE FROM stock_balances WHERE enterprise_id=$1", a.ID)
		pool.Exec(base, "DELETE FROM production_scan_tokens WHERE production_order_id=$1", ordem)
		pool.Exec(base, "DELETE FROM production_orders WHERE id=$1", ordem)
	})
	if _, err := stockpg.NewStockRepositorySQLC(pool).CreateMovement(a.Ctx, &stockentity.StockMovement{ItemCode: chapa, WarehouseID: almox, MovementType: "IN",
		Quantity: decimal.NewFromInt(50), UnitPrice: decimal.NewFromInt(3), CreatedBy: user}); err != nil {
		t.Fatal(err)
	}
	repo := productionrepo.NewProductionOrderRepositoryPGX(pool)
	h := sha256.Sum256([]byte(uuid.NewString()))
	if err := repo.CreateScanToken(a.Ctx, &entity.ScanToken{EnterpriseID: a.ID, ProductionOrderID: ordem, TokenHash: h[:], CreatedBy: user}); err != nil {
		t.Fatal(err)
	}
	res, err := repo.ExecuteScan(a.Ctx, entity.ScanCommand{EnterpriseID: a.ID, UserID: user, TokenHash: h[:], Action: entity.ScanAppoint,
		IdempotencyKey: uuid.NewString(), DeviceID: device, Fingerprint: []byte{1}, GoodQuantity: decimal.NewFromInt(5)})
	if err != nil {
		t.Fatal(err)
	}
	// 5 de 10 = metade dos 20.
	if len(res.Baixas) != 1 || res.Baixas[0].ItemCode != chapa || !res.Baixas[0].Quantidade.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("baixas devolvidas ao operador: %+v", res.Baixas)
	}
	var saldo, consumido decimal.Decimal
	if err := pool.QueryRow(base, `SELECT (SELECT SUM(quantity) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2),
		(SELECT SUM(consumed_qty) FROM production_consumptions WHERE production_order_id=$3 AND appointment_id IS NOT NULL)`, a.ID, chapa, ordem).Scan(&saldo, &consumido); err != nil {
		t.Fatal(err)
	}
	if !saldo.Equal(decimal.NewFromInt(40)) || !consumido.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("saldo %s (quer 40), consumo %s (quer 10)", saldo, consumido)
	}
}
