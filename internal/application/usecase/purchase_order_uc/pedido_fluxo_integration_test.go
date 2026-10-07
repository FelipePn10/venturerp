//go:build integration

package purchase_order_uc_test

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	procuremententity "github.com/FelipePn10/panossoerp/internal/domain/procurement/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

type authCompras struct {
	ports.AuthService
	uid uuid.UUID
}

func (authCompras) CanCreatePurchaseOrder(context.Context) bool { return true }
func (authCompras) CanUpdatePurchaseOrder(context.Context) bool { return true }
func (authCompras) CanGetPurchaseOrder(context.Context) bool    { return true }
func (a authCompras) UserID(context.Context) (uuid.UUID, error) { return a.uid, nil }
func (authCompras) EnterpriseCode(ctx context.Context) (int64, error) {
	return tenant.Code(ctx)
}

// alcadaDe100 libera até 100 e bloqueia até 1000.
type alcadaDe100 struct{}

func (alcadaDe100) EvaluatePurchaseApproval(_ context.Context, _ int64, _ *int64, valor float64) (*procuremententity.ApprovalDecision, error) {
	teto, id := 1000.0, int64(1)
	return &procuremententity.ApprovalDecision{AutoApprove: valor <= 100, Blocked: valor > teto, Ceiling: 100, HardCeiling: &teto, LimitID: &id}, nil
}

// TestFluxoDoPedidoDeCompra cobre os defeitos achados no pedido de compra:
// totais nunca somados (a alçada avaliava 0), alçada não gravada (autorizar
// não achava o pedido), alteração da capa apagando origem e forjando
// situação, linha sem almoxarifado, e cancelamento com recebimento.
func TestFluxoDoPedidoDeCompra(t *testing.T) {
	_, pool := testutil.Queries(t)
	ctx := testutil.TenantContext(t, pool)
	bg := context.Background()
	uid := testutil.Actor(t, pool)
	enterpriseID := testutil.EnterpriseID(t, ctx)
	enterpriseCode, _ := tenant.Code(ctx)

	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id) VALUES($1,'Almox compras',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`,
		"PC-"+uuid.NewString()[:8], uid, enterpriseID).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	comAlmox, semAlmox := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.SeedItem(t, pool, ctx, comAlmox, uid)
	testutil.SeedItem(t, pool, ctx, semAlmox, uid)
	// SeedItem grava warehouse_code = código do item (sem almoxarifado real).
	testutil.Exec(t, pool, `UPDATE items SET supplies_warehouse_code=$2 WHERE code=$1`, comAlmox, almox)
	testutil.Exec(t, pool, `UPDATE items SET supplies_warehouse_code=NULL WHERE code=$1`, semAlmox)
	var pedidos []int64
	t.Cleanup(func() {
		pool.Exec(bg, `DELETE FROM purchase_order_items WHERE purchase_order_code = ANY($1)`, pedidos)
		pool.Exec(bg, `DELETE FROM purchase_orders WHERE code = ANY($1)`, pedidos)
		pool.Exec(bg, `DELETE FROM items WHERE code = ANY($1)`, []int64{comAlmox, semAlmox})
		pool.Exec(bg, `DELETE FROM warehouse WHERE id=$1`, almox)
	})

	repo := purchase_order.NewPurchaseOrderRepositorySQLC(pool)
	auth := authCompras{uid: uid}
	criar := &purchase_order_uc.CreatePurchaseOrderUseCase{Repo: repo, Auth: auth, Almoxarifados: repo}
	alterar := &purchase_order_uc.UpdatePurchaseOrderUseCase{Repo: repo, Auth: auth}
	incluir := &purchase_order_uc.AddPurchaseOrderItemUseCase{Repo: repo, Auth: auth, Almoxarifados: repo}
	alterarLinha := &purchase_order_uc.UpdatePurchaseOrderItemUseCase{Repo: repo, Auth: auth}
	cancelarLinha := &purchase_order_uc.CancelPurchaseOrderItemUseCase{Repo: repo, Auth: auth}
	aprovar := &purchase_order_uc.ApprovePurchaseOrderUseCase{Repo: repo, Auth: auth, Policy: alcadaDe100{}}
	cancelar := &purchase_order_uc.CancelPurchaseOrderUseCase{Repo: repo, Auth: auth}
	obter := &purchase_order_uc.GetPurchaseOrderUseCase{Repo: repo, Auth: auth}

	fornecedor := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'FORNECEDOR PEDIDO',$2,$3,$4)`, fornecedor, uuid.NewString()[:14], uid, enterpriseID)
	t.Cleanup(func() {
		// Roda antes da limpeza dos pedidos (LIFO): solta-os do fornecedor primeiro.
		pool.Exec(bg, `DELETE FROM purchase_order_items WHERE purchase_order_code = ANY($1)`, pedidos)
		pool.Exec(bg, `DELETE FROM purchase_orders WHERE code = ANY($1)`, pedidos)
		pool.Exec(bg, `DELETE FROM suppliers WHERE code=$1`, fornecedor)
	})
	if _, err := criar.Execute(ctx, request.CreatePurchaseOrderDTO{SupplierCode: &fornecedor, Status: "APPROVED"}); err == nil {
		t.Fatal("criar pedido já aprovado deveria ser recusado")
	}
	po, err := criar.Execute(ctx, request.CreatePurchaseOrderDTO{SupplierCode: &fornecedor, FreightType: "FOB", FreightValue: 30, TotalNet: 999})
	if err != nil {
		t.Fatal(err)
	}
	pedidos = append(pedidos, po.Code)
	if po.Status != "DRAFT" || po.TotalNet != 30 {
		t.Fatalf("pedido novo: status=%s total=%v (o total do corpo não vale; só o frete FOB)", po.Status, po.TotalNet)
	}

	// Linha sem almoxarifado: o do cadastro do item completa; sem ele, a
	// mensagem diz onde cadastrar.
	l1, err := incluir.Execute(ctx, request.CreatePurchaseOrderItemDTO{PurchaseOrderCode: po.Code, ItemCode: comAlmox, RequestedQty: 10, UnitPrice: 5, DiscountPct: 10, IPIPct: f(5)})
	if err != nil {
		t.Fatal(err)
	}
	if l1.WarehouseID == nil || *l1.WarehouseID != almox {
		t.Fatalf("almoxarifado da linha = %v, quer o do cadastro %d", l1.WarehouseID, almox)
	}
	if _, err := incluir.Execute(ctx, request.CreatePurchaseOrderItemDTO{PurchaseOrderCode: po.Code, ItemCode: semAlmox, RequestedQty: 1, UnitPrice: 1}); err == nil || !strings.Contains(err.Error(), "VENT0200") {
		t.Fatalf("item sem almoxarifado: err=%v", err)
	}
	got, _ := obter.Execute(ctx, po.Code)
	if got.TotalGross != 50 || got.TotalNet != 77.25 { // 45 + IPI 2,25 + frete 30
		t.Fatalf("totais depois da linha: bruto=%v líquido=%v", got.TotalGross, got.TotalNet)
	}

	// Capa: situação forjada é recusada; origem e totais sobrevivem.
	if _, err := alterar.Execute(ctx, request.UpdatePurchaseOrderDTO{Code: po.Code, SupplierCode: &fornecedor, Status: "APPROVED"}); err == nil {
		t.Fatal("PUT com situação APPROVED deveria ser recusado")
	}
	alterado, err := alterar.Execute(ctx, request.UpdatePurchaseOrderDTO{Code: po.Code, SupplierCode: &fornecedor, Status: "DRAFT", FreightType: "FOB", FreightValue: 0, TotalNet: 1})
	if err != nil {
		t.Fatal(err)
	}
	if alterado.Origin != "NORMAL" || alterado.TotalNet != 47.25 || alterado.FreightType != "FOB" {
		t.Fatalf("capa alterada: origem=%q total=%v frete=%q", alterado.Origin, alterado.TotalNet, alterado.FreightType)
	}

	// Linha: 40 × 5 − 10% = 180 + IPI 9 = 189 → acima da alçada de 100.
	if _, err := alterarLinha.Execute(ctx, request.UpdatePurchaseOrderItemDTO{PurchaseOrderCode: po.Code, ItemLineCode: l1.Code, RequestedQty: f(40)}); err != nil {
		t.Fatal(err)
	}
	r, err := aprovar.Execute(ctx, po.Code)
	if err != nil {
		t.Fatal(err)
	}
	if r.AppliedAmount != 189 || r.AlcadaStatus != "B" {
		t.Fatalf("aprovação: valor=%v alçada=%s", r.AppliedAmount, r.AlcadaStatus)
	}
	got, _ = obter.Execute(ctx, po.Code)
	if got.AlcadaStatus != "B" || got.Status != "REQUESTED" {
		t.Fatalf("alçada não gravada: %s/%s", got.Status, got.AlcadaStatus)
	}
	if _, err := aprovar.Authorize(ctx, po.Code); err != nil {
		t.Fatalf("autorizar pedido bloqueado: %v", err)
	}

	// Aprovado: não muda mais; recebido em parte, não cancela; elimina saldo.
	if _, err := alterarLinha.Execute(ctx, request.UpdatePurchaseOrderItemDTO{PurchaseOrderCode: po.Code, ItemLineCode: l1.Code, RequestedQty: f(1)}); err == nil {
		t.Fatal("alterar linha de pedido aprovado deveria ser recusado")
	}
	if _, err := repo.RegisterItemReceipts(ctx, po.Code, map[int64]float64{l1.Code: 15}); err != nil {
		t.Fatal(err)
	}
	if err := cancelar.Execute(ctx, po.Code); err == nil {
		t.Fatal("cancelar pedido com recebimento deveria ser recusado")
	}
	if _, err := cancelarLinha.Execute(ctx, request.CancelPurchaseOrderItemDTO{PurchaseOrderCode: po.Code, ItemLineCode: l1.Code}); err == nil {
		t.Fatal("eliminar saldo sem motivo deveria ser recusado")
	}
	fim, err := cancelarLinha.Execute(ctx, request.CancelPurchaseOrderItemDTO{PurchaseOrderCode: po.Code, ItemLineCode: l1.Code, Motivo: "fornecedor sem estoque"})
	if err != nil {
		t.Fatal(err)
	}
	if fim.Status != "RECEIVED" || len(fim.Items) != 1 || fim.Items[0].CancelledQty != 25 || fim.TotalNet != 70.88 {
		t.Fatalf("depois de eliminar o saldo: status=%s total=%v itens=%+v", fim.Status, fim.TotalNet, fim.Items)
	}

	// Rascunho: remove linha e cancela levando as linhas junto.
	p2, err := criar.Execute(ctx, request.CreatePurchaseOrderDTO{SupplierCode: &fornecedor})
	if err != nil {
		t.Fatal(err)
	}
	pedidos = append(pedidos, p2.Code)
	a, _ := incluir.Execute(ctx, request.CreatePurchaseOrderItemDTO{PurchaseOrderCode: p2.Code, ItemCode: comAlmox, RequestedQty: 2, UnitPrice: 3})
	if _, err := incluir.Execute(ctx, request.CreatePurchaseOrderItemDTO{PurchaseOrderCode: p2.Code, ItemCode: comAlmox, RequestedQty: 1, UnitPrice: 10}); err != nil {
		t.Fatal(err)
	}
	if _, err := cancelarLinha.Execute(ctx, request.CancelPurchaseOrderItemDTO{PurchaseOrderCode: p2.Code, ItemLineCode: a.Code}); err != nil {
		t.Fatal(err)
	}
	c, err := incluir.Execute(ctx, request.CreatePurchaseOrderItemDTO{PurchaseOrderCode: p2.Code, ItemCode: comAlmox, RequestedQty: 1, UnitPrice: 1})
	if err != nil || c.Sequence != 3 {
		t.Fatalf("sequência depois de remover a linha 1: %v err=%v", c, err)
	}
	if err := cancelar.Execute(ctx, p2.Code); err != nil {
		t.Fatal(err)
	}
	var abertas int
	pool.QueryRow(bg, `SELECT count(*) FROM purchase_order_items WHERE purchase_order_code=$1 AND status<>'CANCELLED'`, p2.Code).Scan(&abertas)
	if abertas != 0 {
		t.Fatalf("pedido cancelado ficou com %d linhas abertas", abertas)
	}

	// O almoxarifado padrão respeita a empresa (items usa enterprise_id): na
	// sessão de outra empresa o item não existe.
	var outraID int64
	outraCode := 880_000_000 + (testutil.UniqueCode() % 100_000_000)
	if err := pool.QueryRow(bg, `INSERT INTO enterprise (code, name, created_by) VALUES ($1, 'COMPRAS VIZINHA', $2) RETURNING id`, outraCode, uid).Scan(&outraID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, outraID) })
	outra := context.WithValue(bg, contextkey.UserKey, &security.AuthUser{ID: uid.String(), Role: "ADMIN", EnterpriseID: outraID, EnterpriseCode: outraCode})
	if wh, err := repo.AlmoxarifadoPadraoDoItem(outra, comAlmox); err != nil || wh != nil {
		t.Fatalf("almoxarifado do item vazou para outra empresa: %v err=%v", wh, err)
	}
	if _, err := obter.Execute(outra, po.Code); err == nil {
		t.Fatalf("pedido da empresa %d visível na empresa %d", enterpriseCode, outraCode)
	}
}

func f(v float64) *float64 { return &v }
