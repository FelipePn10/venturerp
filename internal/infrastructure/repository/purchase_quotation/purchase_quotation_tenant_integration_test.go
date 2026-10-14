//go:build integration

package purchase_quotation_test

import (
	"context"
	"fmt"
	"testing"

	qentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/entity"
	reqentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_quotation"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_requisition"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestPurchaseQuotationAndRequisitionAreIsolatedByTenant: o código da cotação e
// da requisição é único no banco inteiro, mas a numeração era por empresa — a
// segunda empresa recebia um código já usado e a criação falhava. E os filhos
// (itens, fornecedores convidados, preços) eram lidos e gravados só pelo id ou
// pelo código, sem olhar a empresa da cotação/requisição.
func TestPurchaseQuotationAndRequisitionAreIsolatedByTenant(t *testing.T) {
	q, pool := testutil.Queries(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		codes, ids := []int64{a.Code, b.Code}, []int64{a.ID, b.ID}
		_, _ = pool.Exec(base, `DELETE FROM purchase_quotations WHERE enterprise_code = ANY($1)`, codes)
		_, _ = pool.Exec(base, `DELETE FROM purchase_requisitions WHERE enterprise_code = ANY($1)`, codes)
		_, _ = pool.Exec(base, `DELETE FROM suppliers WHERE enterprise_id = ANY($1)`, ids)
		_, _ = pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
	})
	fornecedor := func(e testutil.Empresa) int64 {
		code := testutil.UniqueCode()
		testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'FORNECEDOR',$2,$3,$4)`,
			code, fmt.Sprintf("%014d", code%100_000_000_000_000), actor, e.ID)
		return code
	}
	item := func(e testutil.Empresa) int64 {
		code := testutil.UniqueCode()
		testutil.SeedItem(t, pool, e.Ctx, code, actor)
		return code
	}
	fornA, fornB := fornecedor(a), fornecedor(b)
	itemA, itemB := item(a), item(b)

	cot := purchase_quotation.New(q, pool)
	cria := func(e testutil.Empresa) *qentity.PurchaseQuotation {
		code, err := cot.NextCode(e.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		nova, err := qentity.NewPurchaseQuotation(code, e.Code, actor)
		if err != nil {
			t.Fatal(err)
		}
		criada, err := cot.Create(e.Ctx, nova)
		if err != nil {
			t.Fatalf("empresa %d não criou a cotação %d: %v", e.Code, code, err)
		}
		return criada
	}
	cotA := cria(a)
	cotB := cria(b) // antes: colidia com o código da A
	if cotA.Code == cotB.Code {
		t.Fatalf("as duas empresas receberam o código %d", cotA.Code)
	}

	itA, err := cot.AddItem(a.Ctx, &qentity.PurchaseQuotationItem{QuotationCode: cotA.Code, Sequence: 1, ItemCode: itemA, Quantity: 10, SourceType: qentity.SourceManual})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cot.AddItem(b.Ctx, &qentity.PurchaseQuotationItem{QuotationCode: cotA.Code, Sequence: 2, ItemCode: itemB, Quantity: 1, SourceType: qentity.SourceManual}); err == nil {
		t.Fatal("empresa B incluiu item na cotação da A")
	}
	if _, err := cot.GetItem(b.Ctx, itA.ID); err == nil {
		t.Fatal("empresa B leu o item da cotação da A")
	}
	if l, err := cot.ListItems(b.Ctx, cotA.Code); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d item(ns) da cotação da A (%v)", len(l), err)
	}

	if _, err := cot.AddSupplier(b.Ctx, &qentity.PurchaseQuotationSupplier{QuotationCode: cotA.Code, SupplierCode: fornB}); err == nil {
		t.Fatal("empresa B convidou fornecedor na cotação da A")
	}
	if _, err := cot.AddSupplier(a.Ctx, &qentity.PurchaseQuotationSupplier{QuotationCode: cotA.Code, SupplierCode: fornB}); err == nil {
		t.Fatal("a cotação da A aceitou o fornecedor da B")
	}
	if _, err := cot.AddSupplier(a.Ctx, &qentity.PurchaseQuotationSupplier{QuotationCode: cotA.Code, SupplierCode: fornA}); err != nil {
		t.Fatalf("a cotação da A recusou o próprio fornecedor: %v", err)
	}
	if l, err := cot.ListSuppliers(b.Ctx, cotA.Code); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d fornecedor(es) da cotação da A (%v)", len(l), err)
	}

	if _, err := cot.UpsertPrice(b.Ctx, &qentity.PurchaseQuotationPrice{QuotationItemID: itA.ID, SupplierCode: fornB, UnitPrice: 1}); err == nil {
		t.Fatal("empresa B gravou preço no item da cotação da A")
	}
	if _, err := cot.UpsertPrice(a.Ctx, &qentity.PurchaseQuotationPrice{QuotationItemID: itA.ID, SupplierCode: fornB, UnitPrice: 1}); err == nil {
		t.Fatal("a cotação da A aceitou preço do fornecedor da B")
	}
	preco, err := cot.UpsertPrice(a.Ctx, &qentity.PurchaseQuotationPrice{QuotationItemID: itA.ID, SupplierCode: fornA, UnitPrice: 12.5})
	if err != nil {
		t.Fatalf("a cotação da A recusou o preço do próprio fornecedor: %v", err)
	}
	if l, err := cot.ListPricesByItem(b.Ctx, itA.ID); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d preço(s) da cotação da A (%v)", len(l), err)
	}
	if _, err := cot.SelectPrice(b.Ctx, preco.ID); err == nil {
		t.Fatal("empresa B selecionou o preço da cotação da A")
	}
	if _, err := cot.SelectPrice(a.Ctx, preco.ID); err != nil {
		t.Fatalf("a A não selecionou o próprio preço: %v", err)
	}
	if l, err := cot.ListSelectedPrices(b.Ctx, cotA.Code); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d preço(s) selecionado(s) da A (%v)", len(l), err)
	}
	if l, err := cot.ListSelectedPrices(a.Ctx, cotA.Code); err != nil || len(l) != 1 {
		t.Fatalf("a A listou %d preço(s) selecionado(s), esperado 1 (%v)", len(l), err)
	}

	req := purchase_requisition.New(q, pool)
	criaReq := func(e testutil.Empresa) *reqentity.PurchaseRequisition {
		code, err := req.NextCode(e.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		nova, err := reqentity.NewPurchaseRequisition(code, e.Code, actor)
		if err != nil {
			t.Fatal(err)
		}
		criada, err := req.Create(e.Ctx, nova)
		if err != nil {
			t.Fatalf("empresa %d não criou a requisição %d: %v", e.Code, code, err)
		}
		return criada
	}
	reqA := criaReq(a)
	criaReq(b)

	riA, err := req.AddItem(a.Ctx, &reqentity.PurchaseRequisitionItem{RequisitionCode: reqA.Code, Sequence: 1, ItemCode: itemA, Quantity: 5})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := req.AddItem(b.Ctx, &reqentity.PurchaseRequisitionItem{RequisitionCode: reqA.Code, Sequence: 2, ItemCode: itemB, Quantity: 1}); err == nil {
		t.Fatal("empresa B incluiu item na requisição da A")
	}
	if _, err := req.GetItem(b.Ctx, riA.ID); err == nil {
		t.Fatal("empresa B leu o item da requisição da A — e o levaria para a cotação dela")
	}
	if l, err := req.ListItems(b.Ctx, reqA.Code); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d item(ns) da requisição da A (%v)", len(l), err)
	}
	if _, err := req.RegisterAttendance(b.Ctx, riA.ID, 5); err == nil {
		t.Fatal("empresa B deu atendimento no item da requisição da A")
	}
	if got, err := req.RegisterAttendance(a.Ctx, riA.ID, 5); err != nil || got.AttendedQty != 5 {
		t.Fatalf("a A não atendeu o próprio item: %+v (%v)", got, err)
	}
}

// TestPurchaseQuotationCreateWithChildrenIsAtomic: a cotação era gravada antes
// dos itens e fornecedores; uma falha no meio deixava uma cotação vazia.
func TestPurchaseQuotationCreateWithChildrenIsAtomic(t *testing.T) {
	q, pool := testutil.Queries(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		codes, ids := []int64{a.Code, b.Code}, []int64{a.ID, b.ID}
		_, _ = pool.Exec(base, `DELETE FROM purchase_quotations WHERE enterprise_code = ANY($1)`, codes)
		_, _ = pool.Exec(base, `DELETE FROM suppliers WHERE enterprise_id = ANY($1)`, ids)
		_, _ = pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
	})
	fornecedor := func(e testutil.Empresa) int64 {
		code := testutil.UniqueCode()
		testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'FORNECEDOR',$2,$3,$4)`,
			code, fmt.Sprintf("%014d", code%100_000_000_000_000), actor, e.ID)
		return code
	}
	fornA, fornB := fornecedor(a), fornecedor(b)
	itemA := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, itemA, actor)

	cot := purchase_quotation.New(q, pool)
	nova := func() *qentity.PurchaseQuotation {
		code, err := cot.NextCode(a.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		n, err := qentity.NewPurchaseQuotation(code, a.Code, actor)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	itens := func() []*qentity.PurchaseQuotationItem {
		return []*qentity.PurchaseQuotationItem{{Sequence: 1, ItemCode: itemA, Quantity: 3, SourceType: qentity.SourceManual}}
	}

	falha := nova()
	if _, err := cot.CreateWithChildren(a.Ctx, falha, itens(), []int64{fornA, fornB}); err == nil {
		t.Fatal("a cotação aceitou o fornecedor da outra empresa")
	}
	var sobras int
	if err := pool.QueryRow(base, `SELECT COUNT(*) FROM purchase_quotations WHERE code=$1`, falha.Code).Scan(&sobras); err != nil {
		t.Fatal(err)
	}
	if sobras != 0 {
		t.Fatalf("a falha deixou %d cotação(ões) %d gravada(s)", sobras, falha.Code)
	}

	criada, err := cot.CreateWithChildren(a.Ctx, nova(), itens(), []int64{fornA})
	if err != nil {
		t.Fatalf("a cotação válida não foi criada: %v", err)
	}
	if len(criada.Items) != 1 || len(criada.Suppliers) != 1 || criada.Items[0].QuotationCode != criada.Code {
		t.Fatalf("cotação criada sem os filhos: %d item(ns), %d fornecedor(es)", len(criada.Items), len(criada.Suppliers))
	}
}
