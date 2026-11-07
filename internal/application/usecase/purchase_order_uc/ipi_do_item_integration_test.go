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
	"github.com/shopspring/decimal"
)

type ipiPorClassificacao map[int64]decimal.Decimal

func (m ipiPorClassificacao) GetIPIRate(_ context.Context, code int64) (decimal.Decimal, bool, error) {
	v, ok := m[code]
	return v, ok, nil
}

// TestLinhaDoPedidoHerdaOIPIDoItem: sem IPI e sem classificação na linha, vale
// a classificação de compra do cadastro do item — antes a linha ficava com IPI
// zero, embora a tela dissesse "da classificação".
func TestLinhaDoPedidoHerdaOIPIDoItem(t *testing.T) {
	_, pool := testutil.Queries(t)
	ctx := testutil.TenantContext(t, pool)
	bg := context.Background()
	uid := testutil.Actor(t, pool)
	enterpriseID := testutil.EnterpriseID(t, ctx)

	comClassif, semClassif := testutil.UniqueCode(), testutil.UniqueCode()
	classif := testutil.UniqueCode() % 1_000_000_000
	testutil.SeedItem(t, pool, ctx, comClassif, uid)
	testutil.SeedItem(t, pool, ctx, semClassif, uid)
	testutil.Exec(t, pool, `INSERT INTO fiscal_classifications(code,description,ncm,enterprise_id,created_by) VALUES($1,'Tintas','32089010',$2,$3)`, classif, enterpriseID, uid)
	testutil.Exec(t, pool, `UPDATE items SET accounting_purchase_fiscal_classification_code=$2::bigint::text WHERE code=$1`, comClassif, classif)
	testutil.Exec(t, pool, `UPDATE items SET accounting_purchase_fiscal_classification_code=NULL, accounting_sale_fiscal_classification_code=NULL WHERE code=$1`, semClassif)
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id) VALUES($1,'Almox IPI',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`,
		"IPI-"+uuid.NewString()[:8], uid, enterpriseID).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `UPDATE items SET supplies_warehouse_code=$2 WHERE code = ANY($1)`, []int64{comClassif, semClassif}, almox)
	fornecedor := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'FORNECEDOR IPI',$2,$3,$4)`, fornecedor, uuid.NewString()[:14], uid, enterpriseID)
	var pedidos []int64
	t.Cleanup(func() {
		pool.Exec(bg, `DELETE FROM purchase_order_items WHERE purchase_order_code = ANY($1)`, pedidos)
		pool.Exec(bg, `DELETE FROM purchase_orders WHERE code = ANY($1)`, pedidos)
		pool.Exec(bg, `DELETE FROM suppliers WHERE code=$1`, fornecedor)
		pool.Exec(bg, `DELETE FROM items WHERE code = ANY($1)`, []int64{comClassif, semClassif})
		pool.Exec(bg, `DELETE FROM fiscal_classifications WHERE code=$1 AND enterprise_id=$2`, classif, enterpriseID)
		pool.Exec(bg, `DELETE FROM warehouse WHERE id=$1`, almox)
	})

	repo := purchase_order.NewPurchaseOrderRepositorySQLC(pool)
	auth := authCompras{uid: uid}
	criar := &purchase_order_uc.CreatePurchaseOrderUseCase{Repo: repo, Auth: auth, Almoxarifados: repo}
	incluir := &purchase_order_uc.AddPurchaseOrderItemUseCase{Repo: repo, Auth: auth, Almoxarifados: repo,
		FiscalClass: ipiPorClassificacao{classif: decimal.RequireFromString("3.25")}, Classificacoes: repo}

	po, err := criar.Execute(ctx, request.CreatePurchaseOrderDTO{SupplierCode: &fornecedor})
	if err != nil {
		t.Fatal(err)
	}
	pedidos = append(pedidos, po.Code)

	linha, err := incluir.Execute(ctx, request.CreatePurchaseOrderItemDTO{PurchaseOrderCode: po.Code, ItemCode: comClassif, RequestedQty: decimal.NewFromInt(2), UnitPrice: decimal.NewFromInt(50)})
	if err != nil {
		t.Fatal(err)
	}
	if linha.IPIPct != 3.25 || linha.FiscalClassificationCode == nil || *linha.FiscalClassificationCode != classif {
		t.Fatalf("linha sem IPI: ipi=%v classificação=%v; esperava 3,25%% da classificação %d do item", linha.IPIPct, linha.FiscalClassificationCode, classif)
	}

	informado := decimal.NewFromInt(0)
	linha, err = incluir.Execute(ctx, request.CreatePurchaseOrderItemDTO{PurchaseOrderCode: po.Code, ItemCode: comClassif, RequestedQty: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(50), IPIPct: &informado})
	if err != nil {
		t.Fatal(err)
	}
	if linha.IPIPct != 0 {
		t.Fatalf("IPI digitado vale sobre o do item: ipi=%v", linha.IPIPct)
	}

	linha, err = incluir.Execute(ctx, request.CreatePurchaseOrderItemDTO{PurchaseOrderCode: po.Code, ItemCode: semClassif, RequestedQty: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(50)})
	if err != nil {
		t.Fatal(err)
	}
	if linha.IPIPct != 0 || linha.FiscalClassificationCode != nil {
		t.Fatalf("item sem classificação: ipi=%v classificação=%v", linha.IPIPct, linha.FiscalClassificationCode)
	}
}
