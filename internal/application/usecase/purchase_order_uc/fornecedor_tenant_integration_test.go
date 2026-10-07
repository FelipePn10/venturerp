//go:build integration

package purchase_order_uc_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/supplier_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_order"
	supplierrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/supplier"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O fornecedor do pedido tem de ser da empresa autenticada e estar ativo.
//
// suppliers.code é único no banco inteiro e a chave estrangeira do pedido não
// olha a empresa: informar o código de um fornecedor de OUTRA empresa gravava
// o pedido. Depois dele gravado, o documento saía sem fornecedor (o leitor
// filtra por empresa), não havia e-mail para enviar — e o pedido ainda podia
// receber material.
func TestFornecedorDoPedidoTemDeSerDaEmpresa(t *testing.T) {
	q, pool := testutil.Queries(t)
	ctx := testutil.TenantContext(t, pool)
	bg := context.Background()
	uid := testutil.Actor(t, pool)

	var outraID int64
	outroCode := int32(testutil.UniqueCode()%1000000) + 900000
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Empresa vizinha') RETURNING id`, outroCode).Scan(&outraID); err != nil {
		t.Fatalf("criando a empresa vizinha: %v", err)
	}
	// Um fornecedor da empresa vizinha e um desta empresa, mas inativo.
	doVizinho, inativo := testutil.UniqueCode(), testutil.UniqueCode()
	for code, empresa := range map[int64]int64{doVizinho: outraID, inativo: testutil.EnterpriseID(t, ctx)} {
		if _, err := pool.Exec(bg, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id,is_active)
			VALUES($1,'Fornecedor de teste',$2,$3,$4,$5)`,
			code, fmt.Sprintf("EXT%d", code), uid, empresa, code != inativo); err != nil {
			t.Fatalf("criando fornecedor %d: %v", code, err)
		}
	}
	var pedidos []int64
	t.Cleanup(func() {
		pool.Exec(bg, `DELETE FROM purchase_order_items WHERE purchase_order_code = ANY($1)`, pedidos)
		pool.Exec(bg, `DELETE FROM purchase_orders WHERE code = ANY($1)`, pedidos)
		pool.Exec(bg, `DELETE FROM suppliers WHERE code = ANY($1)`, []int64{doVizinho, inativo})
		pool.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, outraID)
	})

	repo := purchase_order.NewPurchaseOrderRepositorySQLC(pool)
	auth := authCompras{uid: uid}
	supUC := supplier_uc.NewSupplierUseCase(supplierrepo.New(q, pool), auth)
	criar := &purchase_order_uc.CreatePurchaseOrderUseCase{Repo: repo, Auth: auth, SupplierDefaults: supUC, Almoxarifados: repo}
	alterar := &purchase_order_uc.UpdatePurchaseOrderUseCase{Repo: repo, Auth: auth, SupplierDefaults: supUC}

	for _, caso := range []struct {
		nome       string
		fornecedor int64
		espera     string
	}{
		{"fornecedor de outra empresa", doVizinho, "não encontrado no cadastro desta empresa"},
		{"fornecedor inativo", inativo, "está inativo"},
	} {
		t.Run(caso.nome, func(t *testing.T) {
			f := caso.fornecedor
			out, err := criar.Execute(ctx, request.CreatePurchaseOrderDTO{SupplierCode: &f})
			if err == nil {
				pedidos = append(pedidos, out.Code)
				t.Fatalf("criar aceitou o fornecedor %d; o pedido %d não deveria existir", f, out.Code)
			}
			if !strings.Contains(err.Error(), caso.espera) {
				t.Fatalf("criar: esperava %q, veio %q", caso.espera, err.Error())
			}
		})
	}

	// A alteração da capa também não pode trocar o fornecedor por um de fora.
	proprio := testutil.UniqueCode()
	if _, err := pool.Exec(bg, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id)
		VALUES($1,'Fornecedor desta empresa',$2,$3,$4)`,
		proprio, fmt.Sprintf("EXT%d", proprio), uid, testutil.EnterpriseID(t, ctx)); err != nil {
		t.Fatalf("criando fornecedor próprio: %v", err)
	}
	t.Cleanup(func() { pool.Exec(bg, `DELETE FROM suppliers WHERE code=$1`, proprio) })

	criado, err := criar.Execute(ctx, request.CreatePurchaseOrderDTO{SupplierCode: &proprio})
	if err != nil {
		t.Fatalf("criar com fornecedor próprio: %v", err)
	}
	pedidos = append(pedidos, criado.Code)

	if _, err := alterar.Execute(ctx, request.UpdatePurchaseOrderDTO{Code: criado.Code, SupplierCode: &doVizinho}); err == nil {
		t.Fatal("alterar aceitou trocar o fornecedor por um de outra empresa")
	} else if !strings.Contains(err.Error(), "não encontrado no cadastro desta empresa") {
		t.Fatalf("alterar: mensagem inesperada %q", err.Error())
	}
}
