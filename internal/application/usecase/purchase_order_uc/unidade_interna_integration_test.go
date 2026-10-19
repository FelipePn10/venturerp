//go:build integration

package purchase_order_uc_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/google/uuid"
)

// kgParaMetro: o fator do tubo 601 da Tecnofer (1 KG = 0,15530965 M).
type kgParaMetro struct{}

func (kgParaMetro) Factor(context.Context, int64, string, string) (float64, bool, error) {
	return 0.15530965, true, nil
}
func (kgParaMetro) ConvertQuantity(_ context.Context, _ int64, q float64, _, _ string) (float64, bool, error) {
	return q * 0.15530965, true, nil
}
func (kgParaMetro) ConvertUnitPrice(_ context.Context, _ int64, p float64, _, _ string) (float64, bool, error) {
	return p / 0.15530965, true, nil
}

// A quantidade interna gravada é a da conversão (15,530965 M), não 15,5310: o
// recebimento lança no estoque received × internal_qty / requested_qty, e com 4
// casas o estoque recebia uma quantidade que não é a da conversão (migração 408).
func TestPedidoDeCompraGuardaUnidadeInternaComPrecisao(t *testing.T) {
	_, pool := testutil.Queries(t)
	ctx := testutil.TenantContext(t, pool)
	bg := context.Background()
	uid := testutil.Actor(t, pool)
	enterpriseID := testutil.EnterpriseID(t, ctx)
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, ctx, item, uid)
	testutil.Exec(t, pool, `UPDATE items SET warehouse_unit_of_measurement='M', supplies_purchase_uom='KG' WHERE code=$1`, item)
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id) VALUES($1,'Almox tubo',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`,
		"UI-"+uuid.NewString()[:8], uid, enterpriseID).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	fornecedor := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'FORNECEDOR TUBO',$2,$3,$4)`, fornecedor, uuid.NewString()[:14], uid, enterpriseID)
	var pedidos []int64
	t.Cleanup(func() {
		pool.Exec(bg, `DELETE FROM purchase_order_items WHERE purchase_order_code = ANY($1)`, pedidos)
		pool.Exec(bg, `DELETE FROM purchase_orders WHERE code = ANY($1)`, pedidos)
		pool.Exec(bg, `DELETE FROM suppliers WHERE code=$1`, fornecedor)
		pool.Exec(bg, `DELETE FROM items WHERE code=$1`, item)
		pool.Exec(bg, `DELETE FROM warehouse WHERE id=$1`, almox)
	})

	repo := purchase_order.NewPurchaseOrderRepositorySQLC(pool)
	auth := authCompras{uid: uid}
	criar := &purchase_order_uc.CreatePurchaseOrderUseCase{Repo: repo, Auth: auth, Almoxarifados: repo}
	incluir := &purchase_order_uc.AddPurchaseOrderItemUseCase{Repo: repo, Auth: auth, Almoxarifados: repo,
		Unidade: &purchase_order_uc.UnidadeInterna{Unidades: repo, Conversor: kgParaMetro{}}}
	po, err := criar.Execute(ctx, request.CreatePurchaseOrderDTO{SupplierCode: &fornecedor})
	if err != nil {
		t.Fatal(err)
	}
	pedidos = append(pedidos, po.Code)
	kg := "KG"
	if _, err := incluir.Execute(ctx, request.CreatePurchaseOrderItemDTO{PurchaseOrderCode: po.Code, ItemCode: item, RequestedQty: 100, UnitPrice: 6.246, PurchaseUOM: &kg, WarehouseID: &almox}); err != nil {
		t.Fatal(err)
	}
	var qtd, preco string
	if err := pool.QueryRow(bg, `SELECT internal_qty::text, internal_price::text FROM purchase_order_items WHERE purchase_order_code=$1`, po.Code).Scan(&qtd, &preco); err != nil {
		t.Fatal(err)
	}
	if qtd != "15.530965" {
		t.Fatalf("100 KG × 0,15530965 = 15,530965 M; gravado %s", qtd)
	}
	if preco != "40.2164321406" {
		t.Fatalf("R$ 624,60 ÷ 15,530965 M = 40.2164321406; gravado %s", preco)
	}
}
