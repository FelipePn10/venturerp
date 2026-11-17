//go:build integration

package fiscal_uc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	customerpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	salesorderpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// Venda do PR para Manaus com a regra "Zona Franca" e SUFRAMA válida: CST 40,
// ICMS desonerado abatido do total, motivo 7, cBenef PR810175, PIS/COFINS 06,
// CFOP 6109 e o enquadramento do IPI da regra — tudo gravado e lido de volta.
// Com a SUFRAMA vencida a mesma venda é tributada (CST 00, CFOP 6101).
func TestIntegration_FaturarPedido_ZonaFranca(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	queries, _ := testutil.Queries(t)
	actor := uuid.New()
	u := testutil.UniqueCode()
	enterpriseCode := int64(1_900_000_000 + u%90_000_000)
	var enterpriseID int64
	if _, err := pool.Exec(bg, `INSERT INTO users(id,name,email,password) VALUES($1,'ZFM',$2,'x')`, actor, actor.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Zona Franca') RETURNING id`, enterpriseCode).Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: enterpriseID, EnterpriseCode: enterpriseCode})

	customerCode, item := 800_000_000+u%100_000_000, u
	var customerID int64
	testutil.Exec(t, pool, `INSERT INTO fiscal_configs(cnpj_empresa,razao_social,uf_empresa,updated_by,enterprise_id) VALUES('11222333000181','EMPRESA TESTE','PR',$1,$2)`, actor, enterpriseID)
	testutil.Exec(t, pool, `INSERT INTO fiscal_classifications(code,description,ncm,enterprise_id,created_by) VALUES($1,'Teste','73089010',$2,$3)`, u%1_000_000_000, enterpriseID, actor)
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name,accounting_sale_fiscal_classification_code)
		VALUES($1,($1::bigint)::text,$1,$2,$3,'ESTRUTURA '||$1::text,$4)`, item, actor, enterpriseID, fmt.Sprint(u%1_000_000_000))
	if err := pool.QueryRow(bg, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id,state_registration,ie_indicator,finalidade_compra,suframa_code,suframa_expiry)
		VALUES($1,'CLIENTE MANAUS','12345678000195',$2,$3,'041234561','CONTRIBUINTE','REVENDA','200100122',$4::date) RETURNING id`,
		customerCode, actor, enterpriseID, time.Now().AddDate(1, 0, 0).Format("2006-01-02")).Scan(&customerID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO customer_addresses(customer_id,address_type,zip_code,street,number,neighborhood,city,uf,is_default)
		VALUES($1,'COMERCIAL','69050010','AV DJALMA BATISTA','1661','CHAPADA','MANAUS','AM',true)`, customerID)
	testutil.Exec(t, pool, `INSERT INTO icms_ipi_tax_params(enterprise_id,uf,operation_type,ncm_code,uses_icms_zona_franca,cst_ipi_exit,ipi_codigo_enquadramento,is_active)
		VALUES($1,'AM','SAIDA','73089010',true,'52','324',true)`, enterpriseID)
	pedido := func(n int) (int64, int64) {
		var order, line int64
		if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,total_net,created_by) VALUES($1,$2,'P',$3,500,$4) RETURNING code`,
			n, enterpriseCode, customerCode, actor).Scan(&order); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(bg, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,total_net,sales_uom) VALUES($1,1,$2,5,100,500,'PC') RETURNING code`,
			order, item).Scan(&line); err != nil {
			t.Fatal(err)
		}
		return order, line
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM notification_outbox WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(bg, `DELETE FROM fiscal_exits WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(bg, `DELETE FROM sales_order_items WHERE sales_order_code IN (SELECT code FROM sales_orders WHERE enterprise_code=$1)`, enterpriseCode)
		_, _ = pool.Exec(bg, `DELETE FROM sales_orders WHERE enterprise_code=$1`, enterpriseCode)
		_, _ = pool.Exec(bg, `DELETE FROM icms_ipi_tax_params WHERE enterprise_id=$1`, enterpriseID)
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
		Create: &fiscal_uc.CreateFiscalExitUseCase{Repo: fiscalRepo, Auth: auth, Customers: clientes, SalesOrders: pedidos,
			Cadastro: fiscalpg.NewCadastroTributarioPG(pool), Parametros: fiscalpg.NewParametroICMSIPIPG(pool)},
		Fiscal: fiscalRepo, Faturamento: fiscalpg.NewSalesOrderInvoicingRepositoryPG(fiscalRepo),
		Itens: fiscalpg.NewFiscalEntryDocumentRepositoryPG(fiscalRepo), Pedidos: pedidos, Clientes: clientes, Auth: auth,
	}
	faturar := func(order int64) (int64, error) {
		n, err := uc.Execute(ctx, fiscal_uc.FaturarPedidoDTO{SalesOrderCode: order, DataEmissao: time.Now().Format("2006-01-02"), Cfop: "6101", NaturezaOperacao: "Venda"})
		if err != nil {
			return 0, err
		}
		return n.ID, nil
	}

	order, _ := pedido(1)
	id, err := faturar(order)
	if err != nil {
		t.Fatal(err)
	}
	nota, err := fiscalRepo.GetExitByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	itens, err := fiscalRepo.GetExitItems(ctx, id)
	if err != nil || len(itens) != 1 {
		t.Fatalf("itens: %v %d", err, len(itens))
	}
	it := itens[0]
	cb := ""
	if it.CodigoBeneficioFiscal != nil {
		cb = *it.CodigoBeneficioFiscal
	}
	if *it.CstICMS != "40" || !it.ICMSValorDesonerado.Equal(decimal.NewFromInt(35)) || it.ICMSMotivoDesoneracao != "7" ||
		it.Cfop != "6109" || cb != fiscal_uc.CBenefZonaFrancaPR || *it.CstPIS != "06" || *it.CstCOFINS != "06" ||
		*it.CstIPI != "52" || it.IPIEnquadramento != "324" || !it.ValorICMS.IsZero() {
		t.Fatalf("linha ZFM: CST %s deson %s motivo %q CFOP %s cBenef %s PIS %s COFINS %s IPI %s cEnq %s ICMS %s",
			*it.CstICMS, it.ICMSValorDesonerado, it.ICMSMotivoDesoneracao, it.Cfop, cb, *it.CstPIS, *it.CstCOFINS, *it.CstIPI, it.IPIEnquadramento, it.ValorICMS)
	}
	if nota.Cfop != "6109" || !nota.ValorTotal.Equal(decimal.NewFromInt(465)) || nota.SuframaDestinatario == nil || *nota.SuframaDestinatario != "200100122" {
		t.Fatalf("nota ZFM: CFOP %s total %s SUFRAMA %v", nota.Cfop, nota.ValorTotal, nota.SuframaDestinatario)
	}

	// SUFRAMA vencida: tributa normalmente.
	testutil.Exec(t, pool, `UPDATE customers SET suframa_expiry=$2::date WHERE id=$1`, customerID, time.Now().AddDate(0, 0, -5).Format("2006-01-02"))
	order2, _ := pedido(2)
	id2, err := faturar(order2)
	if err != nil {
		t.Fatal(err)
	}
	itens2, _ := fiscalRepo.GetExitItems(ctx, id2)
	if *itens2[0].CstICMS == "40" || itens2[0].ICMSValorDesonerado.IsPositive() || itens2[0].Cfop != "6101" || !itens2[0].ValorICMS.Equal(decimal.NewFromInt(35)) {
		t.Fatalf("sem SUFRAMA válida devia tributar: CST %s deson %s CFOP %s ICMS %s", *itens2[0].CstICMS, itens2[0].ICMSValorDesonerado, itens2[0].Cfop, itens2[0].ValorICMS)
	}
}
