//go:build integration

package purchase_order_uc_test

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_requisition_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pedidocompra"
	customerRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_order"
	reqRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_requisition"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

type authCompleto struct{ authCompras }

func (authCompleto) CanListPurchaseOrders(context.Context) bool { return true }
func (authCompleto) EnterpriseID(ctx context.Context) (int64, error) {
	return tenant.ID(ctx)
}

type emailCapturado struct{ msgs []ports.EmailMessage }

func (e *emailCapturado) Send(_ context.Context, m ports.EmailMessage) error {
	e.msgs = append(e.msgs, m)
	return nil
}

// TestComprasConsultasIntegracao: recebimento só com pedido aprovado,
// acompanhamento de entrega com promessa do fornecedor, destinatários, PDF e
// envio registrado, previsão de pagamento pela condição cadastrada e geração
// de pedido a partir da requisição passando pela alçada.
func TestComprasConsultasIntegracao(t *testing.T) {
	q, pool := testutil.Queries(t)
	ctx := testutil.TenantContext(t, pool)
	bg := context.Background()
	uid := testutil.Actor(t, pool)
	enterpriseID := testutil.EnterpriseID(t, ctx)
	enterpriseCode, _ := tenant.Code(ctx)

	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id) VALUES($1,'Almox consultas',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`,
		"PCQ-"+uuid.NewString()[:8], uid, enterpriseID).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, ctx, item, uid)
	testutil.Exec(t, pool, `UPDATE items SET supplies_warehouse_code=$2, name='PARAFUSO TESTE' WHERE code=$1`, item, almox)
	fornecedor := testutil.UniqueCode()
	var supplierID int64
	if err := pool.QueryRow(bg, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'FORNECEDOR CONSULTAS',$2,$3,$4) RETURNING id`,
		fornecedor, uuid.NewString()[:14], uid, enterpriseID).Scan(&supplierID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO supplier_emails(supplier_id,email,ranking) VALUES($1,'geral@forn.test',1)`, supplierID)
	var contato int64
	if err := pool.QueryRow(bg, `INSERT INTO supplier_contacts(supplier_id,name,purchase_order_tag) VALUES($1,'Vendedor',$2) RETURNING id`, supplierID, "PEDIDO").Scan(&contato); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO supplier_contact_emails(contact_id,value) VALUES($1,'Vendas@Forn.test')`, contato)
	condCode := 900_000 + testutil.UniqueCode()%90_000
	var condID int64
	if err := pool.QueryRow(bg, `INSERT INTO payment_conditions(code,description,enterprise_id) VALUES($1,'28/56 TESTE',$2) RETURNING id`, condCode, enterpriseID).Scan(&condID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO payment_condition_installments(payment_condition_id,installment_number,due_days,base_event) VALUES($1,1,28,'FATURAMENTO'),($1,2,56,'FATURAMENTO')`, condID)
	var reqCode int64 = testutil.UniqueCode()
	var pedidos []int64
	t.Cleanup(func() {
		pool.Exec(bg, `DELETE FROM purchase_order_envios WHERE purchase_order_code = ANY($1)`, pedidos)
		pool.Exec(bg, `DELETE FROM purchase_order_item_followups WHERE purchase_order_code = ANY($1)`, pedidos)
		pool.Exec(bg, `DELETE FROM purchase_order_items WHERE purchase_order_code = ANY($1)`, pedidos)
		pool.Exec(bg, `DELETE FROM purchase_orders WHERE code = ANY($1)`, pedidos)
		pool.Exec(bg, `DELETE FROM purchase_requisition_items WHERE requisition_code=$1`, reqCode)
		pool.Exec(bg, `DELETE FROM purchase_requisitions WHERE code=$1`, reqCode)
		pool.Exec(bg, `DELETE FROM payment_conditions WHERE id=$1`, condID)
		pool.Exec(bg, `DELETE FROM suppliers WHERE id=$1`, supplierID)
		pool.Exec(bg, `DELETE FROM items WHERE code=$1`, item)
		pool.Exec(bg, `DELETE FROM warehouse WHERE id=$1`, almox)
	})

	repo := purchase_order.NewPurchaseOrderRepositorySQLC(pool)
	consultas := purchase_order.NewConsultasPG(pool, nil)
	auth := authCompleto{authCompras{uid: uid}}
	criar := &purchase_order_uc.CreatePurchaseOrderUseCase{Repo: repo, Auth: auth, Almoxarifados: repo}
	incluir := &purchase_order_uc.AddPurchaseOrderItemUseCase{Repo: repo, Auth: auth, Almoxarifados: repo}
	aprovar := &purchase_order_uc.ApprovePurchaseOrderUseCase{Repo: repo, Auth: auth, Policy: alcadaDe100{}}
	acomp := &purchase_order_uc.AcompanhamentoUseCase{Repo: repo, Consultas: consultas, Auth: auth}
	previsao := &purchase_order_uc.PrevisaoPagamentosUseCase{Repo: repo, Condicoes: customerRepo.New(q, pool), Auth: auth}
	mail := &emailCapturado{}
	doc := &purchase_order_uc.DocumentoPedidoUseCase{Repo: repo, Consultas: consultas, Previsao: previsao, Gerador: pedidocompra.Gerador{}, Email: mail, Auth: auth}

	hoje := time.Now()
	atrasada := hoje.AddDate(0, 0, -5).Format("2006-01-02")
	po, err := criar.Execute(ctx, request.CreatePurchaseOrderDTO{SupplierCode: &fornecedor, PaymentTermCode: &condCode})
	if err != nil {
		t.Fatal(err)
	}
	pedidos = append(pedidos, po.Code)
	linha, err := incluir.Execute(ctx, request.CreatePurchaseOrderItemDTO{PurchaseOrderCode: po.Code, ItemCode: item, RequestedQty: 8, UnitPrice: 10, DeliveryDate: &atrasada})
	if err != nil {
		t.Fatal(err)
	}

	// Recebimento antes da aprovação é recusado (a guarda está no repositório).
	if _, err := repo.RegisterItemReceipts(ctx, po.Code, map[int64]float64{linha.Code: 1}); err == nil || !strings.Contains(err.Error(), "aprove") {
		t.Fatalf("receber rascunho: %v", err)
	}
	if _, err := (&purchase_order_uc.ReceivePurchaseOrderUseCase{Repo: repo, Auth: authReceber{auth}}).Execute(ctx,
		request.ReceivePurchaseOrderDTO{PurchaseOrderCode: po.Code, Items: []request.ReceivePurchaseOrderItemDTO{{PurchaseOrderItemCode: linha.Code, Quantity: 1, WarehouseID: almox}}}); err == nil {
		t.Fatal("recebimento direto de rascunho deveria ser recusado")
	}
	// Rascunho não vai ao fornecedor.
	if _, err := doc.Enviar(ctx, po.Code, purchase_order_uc.EnviarDTO{Para: []string{"x@y.com"}}); err == nil {
		t.Fatal("enviar rascunho deveria ser recusado")
	}
	if _, err := aprovar.Execute(ctx, po.Code); err != nil {
		t.Fatal(err)
	}

	// Acompanhamento: a linha está 5 dias atrasada; a promessa do fornecedor a põe em dia.
	atrasadas, err := acomp.Linhas(ctx, purchase_order_uc.AcompanhamentoAtrasadas, &fornecedor, hoje)
	if err != nil || len(atrasadas) != 1 || atrasadas[0].DiasAtraso != 5 || atrasadas[0].ItemName != "PARAFUSO TESTE" || atrasadas[0].SupplierName != "FORNECEDOR CONSULTAS" {
		t.Fatalf("atrasadas: %+v (%v)", atrasadas, err)
	}
	prometida := hoje.AddDate(0, 0, 3).Format("2006-01-02")
	f, err := acomp.RegistrarFollowup(ctx, po.Code, linha.Code, purchase_order_uc.FollowupDTO{DataPrometida: prometida, Contato: "Vendedor", Observacao: "faltou matéria-prima"})
	if err != nil || f.DataPrometida == nil || f.RegistradoPor == "" {
		t.Fatalf("follow-up: %+v (%v)", f, err)
	}
	if l, _ := acomp.Linhas(ctx, purchase_order_uc.AcompanhamentoAtrasadas, &fornecedor, hoje); len(l) != 0 {
		t.Fatalf("com promessa futura não está atrasada: %+v", l)
	}
	prox, _ := acomp.Linhas(ctx, purchase_order_uc.AcompanhamentoProximos7, &fornecedor, hoje)
	if len(prox) != 1 || prox[0].UltimoContato == nil || prox[0].UltimoContato.Contato != "Vendedor" || prox[0].PromisedDate == nil {
		t.Fatalf("próximos 7 dias: %+v", prox)
	}
	if h, _ := acomp.Followups(ctx, po.Code, linha.Code); len(h) != 1 {
		t.Fatalf("histórico de contatos: %+v", h)
	}
	if _, err := acomp.Followups(ctx, po.Code+999999, linha.Code); err == nil {
		t.Fatal("linha de outro pedido não deveria ser lida")
	}

	// Previsão pela condição 28/56, contada da data prometida (a nota vem com o material).
	prev, err := previsao.DoPedido(ctx, po.Code)
	if err != nil || len(prev.Parcelas) != 2 || prev.Total != 80 || prev.Parcelas[0].Condicao != "28/56 TESTE" {
		t.Fatalf("previsão: %+v (%v)", prev, err)
	}
	base, _ := time.Parse("2006-01-02", prometida)
	if !prev.Parcelas[0].Vencimento.Equal(base.AddDate(0, 0, 28)) || !prev.Parcelas[1].Vencimento.Equal(base.AddDate(0, 0, 56)) {
		t.Fatalf("vencimentos: %v / %v", prev.Parcelas[0].Vencimento, prev.Parcelas[1].Vencimento)
	}

	// Destinatários: o contato marcado para pedido vem sugerido e em minúsculas.
	dest, err := doc.Destinatarios(ctx, po.Code)
	if err != nil || len(dest) != 2 || dest[0].Email != "vendas@forn.test" || !dest[0].Sugerido || dest[1].Sugerido {
		t.Fatalf("destinatários: %+v (%v)", dest, err)
	}
	pdf, nome, err := doc.PDF(ctx, po.Code)
	if err != nil || !strings.HasPrefix(string(pdf), "%PDF") || nome != "pedido-compra-"+strconv.FormatInt(po.OrderNumber, 10)+".pdf" {
		t.Fatalf("PDF: %q %v", nome, err)
	}
	env, err := doc.Enviar(ctx, po.Code, purchase_order_uc.EnviarDTO{Para: []string{dest[0].Email}})
	if err != nil || env.Situacao != "ENVIADO" || len(mail.msgs) != 1 || len(mail.msgs[0].Attachments) != 1 {
		t.Fatalf("envio: %+v (%v)", env, err)
	}
	if lista, _ := doc.Envios(ctx, po.Code); len(lista) != 1 || lista[0].EnviadoPorNome == "" {
		t.Fatalf("envios registrados: %+v", lista)
	}

	// Requisição → pedido: nasce em rascunho, passa pela alçada (80 < 100 aprova)
	// e a linha recebe o almoxarifado do cadastro e o vínculo com a requisição.
	testutil.Exec(t, pool, `INSERT INTO purchase_requisitions(code,enterprise_code,created_by) VALUES($1,$2,$3)`, reqCode, enterpriseCode, uid)
	var reqItem int64
	if err := pool.QueryRow(bg, `INSERT INTO purchase_requisition_items(requisition_code,sequence,item_code,quantity,suggested_price) VALUES($1,1,$2,4,20) RETURNING id`, reqCode, item).Scan(&reqItem); err != nil {
		t.Fatal(err)
	}
	gerar := &purchase_requisition_uc.GeneratePurchaseOrdersUseCase{Reqs: reqRepo.New(q, pool), POs: repo, Auth: auth,
		Geracao: &purchase_order_uc.Geracao{Repo: repo, Almoxarifados: repo, Aprovacao: aprovar}}
	res, err := gerar.Execute(ctx, request.GeneratePurchaseOrdersDTO{EnterpriseCode: enterpriseCode, CreatedBy: uid,
		Selections: []request.GenerationSelection{{RequisitionItemID: reqItem, QtyToAttend: 4, SupplierCode: &fornecedor}}})
	if err != nil || len(res.Orders) != 1 {
		t.Fatalf("gerar pela requisição: %+v (%v)", res, err)
	}
	gerado := res.Orders[0]
	pedidos = append(pedidos, gerado.Code)
	if gerado.Status != "APPROVED" || gerado.AlcadaStatus != "A" || gerado.TotalNet != 80 {
		t.Fatalf("pedido gerado: %s/%s total %v avisos %v", gerado.Status, gerado.AlcadaStatus, gerado.TotalNet, res.Skipped)
	}
	linhas, _ := repo.ListItems(ctx, gerado.Code)
	if len(linhas) != 1 || linhas[0].WarehouseID == nil || *linhas[0].WarehouseID != almox || linhas[0].PurchaseRequisitionItemID == nil || *linhas[0].PurchaseRequisitionItemID != reqItem {
		t.Fatalf("linha gerada: %+v", linhas[0])
	}
}

type authReceber struct{ authCompleto }

func (authReceber) CanCreateStockMovement(context.Context) bool { return true }
