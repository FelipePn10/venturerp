//go:build integration

package fiscal_uc_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	customerpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	salesorderpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// O tipo de NF do pedido e o das linhas valem no faturamento (antes eram
// gravados e ignorados): a prévia devolve o tipo, a nota sem tipo escolhido o
// leva, tipo inativo impede com o motivo e linhas de tipos diferentes não vão
// na mesma nota.
func TestIntegration_FaturarPedido_TipoDeNFDoPedido(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	queries, _ := testutil.Queries(t)
	actor := uuid.New()
	u := testutil.UniqueCode()
	enterpriseCode := int64(1_900_000_000 + u%90_000_000)
	var enterpriseID int64
	if _, err := pool.Exec(bg, `INSERT INTO users(id,name,email,password) VALUES($1,'Tipo NF',$2,'x')`, actor, actor.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Tipo de NF do pedido') RETURNING id`, enterpriseCode).Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: enterpriseID, EnterpriseCode: enterpriseCode})

	customerCode, item, tipoCodigo := 800_000_000+u%100_000_000, u, int64(77)
	var customerID, orderCode, tipoID, tipoBonif, linha1, linha2 int64
	testutil.Exec(t, pool, `INSERT INTO fiscal_configs(cnpj_empresa,razao_social,uf_empresa,updated_by,enterprise_id) VALUES('11222333000181','EMPRESA TESTE','PR',$1,$2)`, actor, enterpriseID)
	testutil.Exec(t, pool, `INSERT INTO fiscal_classifications(code,description,ncm,enterprise_id,created_by) VALUES($1,'Teste','73089010',$2,$3)`, u%1_000_000_000, enterpriseID, actor)
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name,accounting_sale_fiscal_classification_code)
		VALUES($1,($1::bigint)::text,$1,$2,$3,'PRODUTO '||$1::text,$4)`, item, actor, enterpriseID, fmt.Sprint(u%1_000_000_000))
	if err := pool.QueryRow(bg, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'CLIENTE TIPO NF','12345678000195',$2,$3) RETURNING id`, customerCode, actor, enterpriseID).Scan(&customerID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO customer_addresses(customer_id,address_type,zip_code,street,number,neighborhood,city,uf,is_default)
		VALUES($1,'COMERCIAL','80000000','RUA DO CLIENTE','100','CENTRO','CURITIBA','PR',true)`, customerID)
	if err := pool.QueryRow(bg, `INSERT INTO invoice_types(code,description,enterprise_id,updates_inventory,generates_financial_title) VALUES($1,'VENDA PADRÃO DO PEDIDO',$2,false,false) RETURNING id`, tipoCodigo, enterpriseID).Scan(&tipoID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO invoice_types(code,description,enterprise_id,updates_inventory,generates_financial_title) VALUES(78,'BONIFICAÇÃO',$1,false,false) RETURNING id`, enterpriseID).Scan(&tipoBonif); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,total_net,created_by,default_nf_type)
		VALUES(1,$1,'P',$2,500,$3,$4) RETURNING code`, enterpriseCode, customerCode, actor, fmt.Sprint(tipoCodigo)).Scan(&orderCode); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,total_net,sales_uom) VALUES($1,1,$2,5,100,500,'PC') RETURNING code`, orderCode, item).Scan(&linha1); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,total_net,sales_uom,nf_type) VALUES($1,2,$2,1,100,100,'PC','78') RETURNING code`, orderCode, item).Scan(&linha2); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM notification_outbox WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(bg, `DELETE FROM fiscal_exits WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(bg, `DELETE FROM sales_order_items WHERE sales_order_code=$1`, orderCode)
		_, _ = pool.Exec(bg, `DELETE FROM sales_orders WHERE code=$1`, orderCode)
		_, _ = pool.Exec(bg, `DELETE FROM invoice_types WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(bg, `DELETE FROM customers WHERE id=$1`, customerID)
		_, _ = pool.Exec(bg, `DELETE FROM items WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(bg, `DELETE FROM fiscal_classifications WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(bg, `DELETE FROM fiscal_configs WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, enterpriseID)
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id=$1`, actor)
	})

	auth := &infraauth.AuthService{}
	fiscalRepo := fiscalpg.NewFiscalRepositoryPG(pool)
	pedidos := salesorderpg.NewSalesOrderRepositorySQLC(queries)
	clientes := customerpg.New(queries, pool)
	uc := &fiscal_uc.FaturarPedidoUseCase{
		Create:      &fiscal_uc.CreateFiscalExitUseCase{Repo: fiscalRepo, Auth: auth, Customers: clientes, SalesOrders: pedidos, Cadastro: fiscalpg.NewCadastroTributarioPG(pool)},
		Fiscal:      fiscalRepo,
		Faturamento: fiscalpg.NewSalesOrderInvoicingRepositoryPG(fiscalRepo),
		Itens:       fiscalpg.NewFiscalEntryDocumentRepositoryPG(fiscalRepo),
		Pedidos:     pedidos,
		Clientes:    clientes,
		Auth:        auth,
	}

	hoje := time.Now().Format("2006-01-02")
	so := func(linha int64, qtd float64) fiscal_uc.FaturarPedidoDTO {
		return fiscal_uc.FaturarPedidoDTO{SalesOrderCode: orderCode, DataEmissao: hoje, Cfop: "5101", NaturezaOperacao: "Venda",
			Itens: []fiscal_uc.FaturarPedidoItem{{SalesOrderItemCode: linha, Quantidade: qtd}}}
	}

	// Saldo inteiro mistura o tipo do pedido (77) e a linha de bonificação (78).
	previa, err := uc.Previa(ctx, orderCode)
	if err != nil {
		t.Fatal(err)
	}
	if !previa.PodeFaturar || previa.InvoiceTypeID != nil || !strings.Contains(strings.Join(previa.Avisos, ";"), "tipos de NF diferentes") {
		t.Fatalf("tipos misturados: avisa sem travar o faturamento por linha (pode %v, tipo %v, avisos %v, impedimentos %v)", previa.PodeFaturar, previa.InvoiceTypeID, previa.Avisos, previa.Impedimentos)
	}
	if _, err := uc.Execute(ctx, fiscal_uc.FaturarPedidoDTO{SalesOrderCode: orderCode, DataEmissao: hoje, Cfop: "5101", NaturezaOperacao: "Venda"}); err == nil {
		t.Fatal("faturar tudo numa nota com tipos diferentes deveria ser recusado")
	}
	bonif, err := uc.Execute(ctx, so(linha2, 1))
	if err != nil {
		t.Fatal(err)
	}
	if bonif.InvoiceTypeID == nil || *bonif.InvoiceTypeID != tipoBonif {
		t.Fatalf("a linha de bonificação devia sair com o tipo 78 (%d), saiu %v", tipoBonif, bonif.InvoiceTypeID)
	}

	// Sem a bonificação pendente, sobra o tipo do pedido.
	previa, err = uc.Previa(ctx, orderCode)
	if err != nil {
		t.Fatal(err)
	}
	if previa.InvoiceTypeID == nil || *previa.InvoiceTypeID != tipoID || !previa.PodeFaturar {
		t.Fatalf("prévia: tipo %v (quer %d), pode faturar %v %v", previa.InvoiceTypeID, tipoID, previa.PodeFaturar, previa.Impedimentos)
	}

	testutil.Exec(t, pool, `UPDATE invoice_types SET is_active=false WHERE id=$1`, tipoID)
	previa, err = uc.Previa(ctx, orderCode)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(previa.Avisos, ";"), "inativo") {
		t.Fatalf("tipo inativo deveria ser avisado na prévia: %v", previa.Avisos)
	}
	dto := so(linha1, 5)
	if _, err := uc.Execute(ctx, dto); err == nil || !strings.Contains(err.Error(), "inativo") {
		t.Fatalf("faturar com tipo inativo: %v", err)
	}

	testutil.Exec(t, pool, `UPDATE invoice_types SET is_active=true WHERE id=$1`, tipoID)

	// Tipo de nota que exige SUFRAMA: com a inscrição vencida a nota não sai.
	testutil.Exec(t, pool, `UPDATE invoice_types SET exige_suframa=true WHERE id=$1`, tipoID)
	testutil.Exec(t, pool, `UPDATE customers SET suframa_code='20.0100.12-2', suframa_expiry=$2::date WHERE id=$1`, customerID, time.Now().AddDate(0, 0, -10).Format("2006-01-02"))
	if _, err := uc.Execute(ctx, dto); err == nil || !strings.Contains(err.Error(), "SUFRAMA") {
		t.Fatalf("SUFRAMA vencida com tipo que exige: %v", err)
	}
	testutil.Exec(t, pool, `UPDATE customers SET suframa_expiry=$2::date WHERE id=$1`, customerID, time.Now().AddDate(1, 0, 0).Format("2006-01-02"))
	nota, err := uc.Execute(ctx, dto)
	if err != nil {
		t.Fatal(err)
	}
	if nota.InvoiceTypeID == nil || *nota.InvoiceTypeID != tipoID {
		t.Fatalf("a nota devia levar o tipo do pedido %d, levou %v", tipoID, nota.InvoiceTypeID)
	}
	gravada, err := fiscalRepo.GetExitByID(ctx, nota.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gravada.SuframaDestinatario == nil || *gravada.SuframaDestinatario != "200100122" {
		t.Fatalf("a nota devia guardar a SUFRAMA válida do cliente: %v", gravada.SuframaDestinatario)
	}
}
