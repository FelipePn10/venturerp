//go:build integration

package fiscal_uc_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	customerpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	salesorderpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// Faturamento do pedido em duas notas. A primeira leva parte de uma linha; a
// prévia passa a mostrar o que está em nota e o que falta; quantidade maior
// que o pendente é recusada; a segunda nota leva o resto e só então o pedido
// fica todo atendido. Cancelar a nota devolve a quantidade ao pedido.
//
// Usa uma empresa própria com código ≠ id, para pegar qualquer comparação de
// `enterprise_code` (pedido) com `enterprise_id` (nota).
func TestIntegration_FaturarPedidoParcialmente(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	queries, _ := testutil.Queries(t)
	actor := uuid.New()
	u := testutil.UniqueCode()
	enterpriseCode := int64(1_900_000_000 + u%90_000_000)
	var enterpriseID int64
	if _, err := pool.Exec(bg, `INSERT INTO users(id,name,email,password) VALUES($1,'Faturamento',$2,'x')`, actor, actor.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Faturamento do pedido') RETURNING id`, enterpriseCode).Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: enterpriseID, EnterpriseCode: enterpriseCode})

	customerCode := 800_000_000 + u%100_000_000
	itemA, itemB := u, u+1
	var customerID, orderCode, lineA, lineB int64
	testutil.Exec(t, pool, `INSERT INTO fiscal_configs(cnpj_empresa,razao_social,uf_empresa,updated_by,enterprise_id) VALUES('11222333000181','EMPRESA TESTE','PR',$1,$2)`, actor, enterpriseID)
	testutil.Exec(t, pool, `INSERT INTO fiscal_classifications(code,description,ncm,enterprise_id,created_by) VALUES($1,'Teste','73089010',$2,$3)`, u%1_000_000_000, enterpriseID, actor)
	for _, code := range []int64{itemA, itemB} {
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name,accounting_sale_fiscal_classification_code)
			VALUES($1,($1::bigint)::text,$1,$2,$3,'PRODUTO '||$1::text,$4)`, code, actor, enterpriseID, fmt.Sprint(u%1_000_000_000))
	}
	if err := pool.QueryRow(bg, `INSERT INTO customers(code,name,document_number,created_by) VALUES($1,'CLIENTE DO PEDIDO','12345678000195',$2) RETURNING id`, customerCode, actor).Scan(&customerID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO customer_addresses(customer_id,address_type,zip_code,street,number,neighborhood,city,uf,is_default)
		VALUES($1,'ENTREGA','80000000','RUA DO CLIENTE','100','CENTRO','CURITIBA','PR',true)`, customerID)
	if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,representative_code,commission_pct,total_net,freight_value,created_by)
		VALUES(1,$1,'P',$2,77,5,1150,115,$3) RETURNING code`, enterpriseCode, customerCode, actor).Scan(&orderCode); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,discount_pct,total_net,sales_uom)
		VALUES($1,1,$2,10,100,10,900,'PC') RETURNING code`, orderCode, itemA).Scan(&lineA); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,total_net)
		VALUES($1,2,$2,5,50,250) RETURNING code`, orderCode, itemB).Scan(&lineB); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM notification_outbox WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(bg, `DELETE FROM commercial_commission_ledger WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(bg, `DELETE FROM fiscal_exits WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(bg, `DELETE FROM sales_order_items WHERE sales_order_code=$1`, orderCode)
		_, _ = pool.Exec(bg, `DELETE FROM sales_orders WHERE code=$1`, orderCode)
		_, _ = pool.Exec(bg, `DELETE FROM customers WHERE id=$1`, customerID)
		_, _ = pool.Exec(bg, `DELETE FROM items WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(bg, `DELETE FROM fiscal_classifications WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(bg, `DELETE FROM fiscal_configs WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, enterpriseID)
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id=$1`, actor)
	})

	auth := &infraauth.AuthService{}
	fiscalRepo := fiscalpg.NewFiscalRepositoryPG(pool)
	faturamento := fiscalpg.NewSalesOrderInvoicingRepositoryPG(fiscalRepo)
	pedidos := salesorderpg.NewSalesOrderRepositorySQLC(queries)
	clientes := customerpg.New(queries, pool)
	uc := &fiscal_uc.FaturarPedidoUseCase{
		Create:      &fiscal_uc.CreateFiscalExitUseCase{Repo: fiscalRepo, Auth: auth, Customers: clientes, SalesOrders: pedidos},
		Fiscal:      fiscalRepo,
		Faturamento: faturamento,
		Itens:       fiscalpg.NewFiscalEntryDocumentRepositoryPG(fiscalRepo),
		Pedidos:     pedidos,
		Clientes:    clientes,
		Auth:        auth,
	}

	previa, err := uc.Previa(ctx, orderCode)
	if err != nil {
		t.Fatalf("prévia: %v", err)
	}
	if !previa.PodeFaturar || len(previa.Itens) != 2 {
		t.Fatalf("pedido deveria poder ser faturado: %+v", previa.Impedimentos)
	}
	if previa.CustomerUF != "PR" || previa.CfopSugerido != "5101" || previa.Itens[0].NCM != "73089010" {
		t.Fatalf("destinatário/CFOP/NCM da prévia: UF %q CFOP %q NCM %q", previa.CustomerUF, previa.CfopSugerido, previa.Itens[0].NCM)
	}
	if previa.Itens[0].PrecoUnitario != 90 {
		t.Fatalf("preço líquido deveria ser 90 (100 com 10%% de desconto), veio %v", previa.Itens[0].PrecoUnitario)
	}

	hoje := time.Now().Format("2006-01-02")
	nota1, err := uc.Execute(ctx, fiscal_uc.FaturarPedidoDTO{
		SalesOrderCode: orderCode, DataEmissao: hoje, Cfop: "5101", NaturezaOperacao: "Venda",
		Itens: []fiscal_uc.FaturarPedidoItem{{SalesOrderItemCode: lineA, Quantidade: 4}},
	})
	if err != nil {
		t.Fatalf("primeira nota: %v", err)
	}
	if nota1.ValorProdutos != 360 || nota1.ValorFrete != 36 {
		t.Fatalf("primeira nota: produtos %v (esperado 360) frete %v (esperado 36, proporcional)", nota1.ValorProdutos, nota1.ValorFrete)
	}
	if nota1.SalesOrderCode == nil || *nota1.SalesOrderCode != orderCode {
		t.Fatal("a nota tem de levar o pedido: é por ele que nasce a comissão")
	}

	previa, _ = uc.Previa(ctx, orderCode)
	if previa.Itens[0].QuantidadeEmNota != 4 || previa.Itens[0].QuantidadePendente != 6 {
		t.Fatalf("depois da 1ª nota: em nota %v pendente %v", previa.Itens[0].QuantidadeEmNota, previa.Itens[0].QuantidadePendente)
	}
	if _, err := uc.Execute(ctx, fiscal_uc.FaturarPedidoDTO{
		SalesOrderCode: orderCode, DataEmissao: hoje, Cfop: "5101", NaturezaOperacao: "Venda",
		Itens: []fiscal_uc.FaturarPedidoItem{{SalesOrderItemCode: lineA, Quantidade: 7}},
	}); err == nil {
		t.Fatal("faturar mais que o pendente deveria ser recusado")
	}

	// Autorização da 1ª nota (sem SEFAZ no teste: o que a autorização faz no pedido).
	testutil.Exec(t, pool, `UPDATE fiscal_exits SET status='AUTHORIZED' WHERE id=$1`, nota1.ID)
	if ligada, atendido, err := faturamento.RegistrarFaturamento(ctx, nota1.ID, false); err != nil || !ligada || atendido {
		t.Fatalf("1ª nota: ligada=%v atendido=%v err=%v (esperado ligada e pedido ainda aberto)", ligada, atendido, err)
	}

	nota2, err := uc.Execute(ctx, fiscal_uc.FaturarPedidoDTO{SalesOrderCode: orderCode, DataEmissao: hoje, Cfop: "5101", NaturezaOperacao: "Venda"})
	if err != nil {
		t.Fatalf("segunda nota (resto): %v", err)
	}
	if nota2.ValorProdutos != 790 || len(nota2.Itens) != 2 {
		t.Fatalf("segunda nota: produtos %v (esperado 6×90 + 5×50 = 790), itens %d", nota2.ValorProdutos, len(nota2.Itens))
	}
	testutil.Exec(t, pool, `UPDATE fiscal_exits SET status='AUTHORIZED' WHERE id=$1`, nota2.ID)
	if _, atendido, err := faturamento.RegistrarFaturamento(ctx, nota2.ID, false); err != nil || !atendido {
		t.Fatalf("depois da 2ª nota o pedido deveria estar todo atendido (err=%v)", err)
	}
	previa, _ = uc.Previa(ctx, orderCode)
	if previa.PodeFaturar {
		t.Fatal("pedido todo atendido não pode ser faturado de novo")
	}

	// Cancelamento devolve a quantidade ao pedido.
	if _, atendido, err := faturamento.RegistrarFaturamento(ctx, nota2.ID, true); err != nil || atendido {
		t.Fatalf("estorno: atendido=%v err=%v", atendido, err)
	}
	testutil.Exec(t, pool, `UPDATE fiscal_exits SET status='CANCELLED' WHERE id=$1`, nota2.ID)
	previa, _ = uc.Previa(ctx, orderCode)
	if previa.Itens[0].QuantidadePendente != 6 || previa.Itens[1].QuantidadePendente != 5 {
		t.Fatalf("depois do cancelamento: pendente A %v B %v", previa.Itens[0].QuantidadePendente, previa.Itens[1].QuantidadePendente)
	}

	// Doze usuários faturando ao mesmo tempo as 5 unidades pendentes da linha
	// B: sem o bloqueio do pedido, todos liam o mesmo saldo e o pedido saía
	// faturado em dobro. Só um pode conseguir.
	var wg sync.WaitGroup
	largada := make(chan struct{})
	resultados := make([]error, 12)
	for i := range resultados {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-largada
			_, resultados[i] = uc.Execute(ctx, fiscal_uc.FaturarPedidoDTO{
				SalesOrderCode: orderCode, DataEmissao: hoje, Cfop: "5101", NaturezaOperacao: "Venda",
				Itens: []fiscal_uc.FaturarPedidoItem{{SalesOrderItemCode: lineB, Quantidade: 5}},
			})
		}(i)
	}
	close(largada)
	wg.Wait()
	sucessos := 0
	for _, err := range resultados {
		var conflito *errorsuc.ConflictError
		var validacao *errorsuc.ValidationError
		switch {
		case err == nil:
			sucessos++
		case errors.As(err, &conflito), errors.As(err, &validacao):
			// em faturamento por outro (conflito) ou já sem saldo (validação)
		default:
			t.Fatalf("erro inesperado no faturamento concorrente: %v", err)
		}
	}
	if sucessos != 1 {
		t.Fatalf("faturamento concorrente: %d nota(s) para as mesmas 5 unidades (erros: %v)", sucessos, resultados)
	}
	previa, _ = uc.Previa(ctx, orderCode)
	if previa.Itens[1].QuantidadePendente != 0 {
		t.Fatalf("depois do faturamento concorrente: pendente B %v", previa.Itens[1].QuantidadePendente)
	}
}
