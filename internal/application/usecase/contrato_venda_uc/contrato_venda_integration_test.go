//go:build integration

package contrato_venda_uc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/contrato_venda_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/contratovenda"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	cvpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/contratovenda"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	item_ "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	notificationrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/notification"
	salesorder "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// O contrato só recebe chamada ativo e na vigência; cada chamada gera o
// pedido de venda pelo preço do contrato e consome o saldo do item e do teto
// de valor; pedido cancelado devolve o saldo; a parcela programada fica
// atendida; com chamada, o contrato encerra mas não cancela; saldo baixo gera
// alerta. Outra empresa não vê nada.
func TestContratoDeVenda(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	item := 87_000_000 + u%1_000_000
	cliente := 865_000 + u%90_000
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []testutil.Empresa{a, b} {
			for _, q := range []string{
				`DELETE FROM notification_alerts WHERE enterprise_id=$1`, `DELETE FROM notification_outbox WHERE enterprise_id=$1`,
				`DELETE FROM enterprise_notification_subscriptions WHERE enterprise_id=$1`,
				`DELETE FROM contratos_venda_chamadas WHERE enterprise_id=$1`, `DELETE FROM contratos_venda WHERE enterprise_id=$1`,
				`DELETE FROM sales_order_items WHERE sales_order_code IN (SELECT code FROM sales_orders WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1))`,
				`DELETE FROM sales_orders WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1)`,
				`DELETE FROM customers WHERE enterprise_id=$1`, `DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`,
			} {
				if _, err := pool.Exec(bg, q, e.ID); err != nil {
					t.Logf("limpeza %s: %v", q, err)
				}
			}
		}
	})
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		VALUES($1,'Acabado',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, fmt.Sprintf("CV-%d", u), actor, a.ID).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name) VALUES($1,'CV'||($1::bigint)::text,$2,$3,$4,'CHAPA CONTRATO')`, item, almox, actor, a.ID)
	testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'CLIENTE CONTRATO','11444777000161',$2,$3)`, cliente, actor, a.ID)

	uc := &contrato_venda_uc.UseCase{Repo: cvpg.New(pool), Auth: &auth.AuthService{}, Items: item_.NewRepositoryItemSQLC(sqlc.New(pool))}
	hoje := time.Now()
	teto := d("850")
	c, err := uc.Criar(a.Ctx, contrato_venda_uc.CriarDTO{CustomerCode: cliente, Descricao: "Fornecimento anual de chapas",
		VigenciaInicio: hoje.AddDate(0, 0, -10).Format("2006-01-02"), VigenciaFim: hoje.AddDate(0, 0, 20).Format("2006-01-02"),
		ValorContratado: &teto, Itens: []contrato_venda_uc.ItemDTO{{ItemCode: request.TextCode(fmt.Sprintf("CV%d", item)), Quantidade: d("100"), PrecoUnitario: d("10")}}})
	if err != nil {
		t.Fatal(err)
	}
	ci := c.Itens[0].ID
	entrega := hoje.AddDate(0, 0, 5).Format("2006-01-02")
	if _, err := uc.Chamar(a.Ctx, c.ID, contrato_venda_uc.ChamadaDTO{Linhas: []dom.LinhaChamada{{ContratoItemID: ci, Quantidade: d("1")}}, DataEntrega: entrega}); err == nil {
		t.Fatal("rascunho não recebe chamada")
	}
	if _, err := uc.Mudar(a.Ctx, c.ID, "ATIVO", ""); err != nil {
		t.Fatal(err)
	}
	c, err = uc.Programar(a.Ctx, c.ID, contrato_venda_uc.ParcelaDTO{ContratoItemID: ci, Data: entrega, Quantidade: d("30")})
	if err != nil {
		t.Fatal(err)
	}
	parcela := c.Programacao[0].ID

	r, err := uc.Chamar(a.Ctx, c.ID, contrato_venda_uc.ChamadaDTO{Linhas: []dom.LinhaChamada{{ContratoItemID: ci, Quantidade: d("30"), ParcelaID: &parcela}}, DataEntrega: entrega})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Total.Equal(d("300")) || !r.Contrato.Itens[0].Saldo.Equal(d("70")) || !r.Contrato.ValorConsumido.Equal(d("300")) || r.Contrato.Programacao[0].SalesOrderCode == nil {
		t.Fatalf("chamada: %+v", r)
	}
	var preco decimal.Decimal
	var qtd decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT unit_price, requested_qty FROM sales_order_items WHERE sales_order_code=$1`, r.SalesOrderCode).Scan(&preco, &qtd); err != nil {
		t.Fatal(err)
	}
	if !preco.Equal(d("10")) || !qtd.Equal(d("30")) {
		t.Fatalf("pedido pelo preço do contrato: %s × %s", qtd, preco)
	}
	var totalLinha, totalPedido decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT i.total_gross, o.total_gross FROM sales_order_items i JOIN sales_orders o ON o.code=i.sales_order_code WHERE i.sales_order_code=$1`, r.SalesOrderCode).Scan(&totalLinha, &totalPedido); err != nil {
		t.Fatal(err)
	}
	if !totalLinha.Equal(d("300")) || !totalPedido.Equal(d("300")) {
		t.Fatalf("o pedido de chamada nasce totalizado: linha %s pedido %s", totalLinha, totalPedido)
	}
	// Alterar a linha de chamada mantém o preço do contrato (o pedido nem tem
	// tabela de preço) — antes repreçava pela tabela.
	var linha int64
	if err := pool.QueryRow(bg, `SELECT code FROM sales_order_items WHERE sales_order_code=$1`, r.SalesOrderCode).Scan(&linha); err != nil {
		t.Fatal(err)
	}
	q := sqlc.New(pool)
	alterar := &sales_order_uc.UpdateSalesOrderItemUseCase{Repo: salesorder.NewSalesOrderRepositorySQLC(q), Auth: &auth.AuthService{},
		Prices: customer.New(q, pool), Items: item_.NewRepositoryItemSQLC(q), Contratos: cvpg.New(pool)}
	if _, err := alterar.Execute(a.Ctx, request.UpdateSalesOrderItemDTO{Code: linha, RequestedQty: d("30"), UnitPrice: d("999"), TributosManuais: true}); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `SELECT unit_price FROM sales_order_items WHERE code=$1`, linha).Scan(&preco); err != nil || !preco.Equal(d("10")) {
		t.Fatalf("a alteração mantém o preço do contrato: %s %v", preco, err)
	}
	if _, err := uc.Chamar(a.Ctx, c.ID, contrato_venda_uc.ChamadaDTO{Linhas: []dom.LinhaChamada{{ContratoItemID: ci, Quantidade: d("71")}}, DataEntrega: entrega}); err == nil {
		t.Fatal("71 passa do saldo de 70")
	}
	if _, err := uc.Chamar(a.Ctx, c.ID, contrato_venda_uc.ChamadaDTO{Linhas: []dom.LinhaChamada{{ContratoItemID: ci, Quantidade: d("60")}}, DataEntrega: entrega}); err == nil {
		t.Fatal("600 passa dos 550 de teto restante")
	}
	if _, err := uc.Chamar(a.Ctx, c.ID, contrato_venda_uc.ChamadaDTO{Linhas: []dom.LinhaChamada{{ContratoItemID: ci, Quantidade: d("1")}}, DataEntrega: hoje.AddDate(0, 0, 30).Format("2006-01-02")}); err == nil {
		t.Fatal("entrega fora da vigência")
	}
	r2, err := uc.Chamar(a.Ctx, c.ID, contrato_venda_uc.ChamadaDTO{Linhas: []dom.LinhaChamada{{ContratoItemID: ci, Quantidade: d("50")}}, DataEntrega: entrega})
	if err != nil {
		t.Fatal(err)
	}
	if !r2.Contrato.Itens[0].Saldo.Equal(d("20")) {
		t.Fatalf("saldo 20: %s", r2.Contrato.Itens[0].Saldo)
	}

	// Saldo 20 de 100 = 20% → alerta de saldo baixo só abaixo de 20%: ainda não.
	testutil.Exec(t, pool, `INSERT INTO enterprise_notification_subscriptions(enterprise_id,event_key,cadence,created_by) VALUES($1,'COMERCIAL_CONTRATO_SALDO_BAIXO','RESUMO_DIARIO',$2),($1,'COMERCIAL_CONTRATO_VENCENDO','RESUMO_DIARIO',$2)`, a.ID, actor)
	notif := notificationrepo.New(pool)
	contar := func(ev string) int {
		var n int
		_ = pool.QueryRow(bg, `SELECT COUNT(*) FROM notification_outbox WHERE enterprise_id=$1 AND event_key=$2`, a.ID, ev).Scan(&n)
		return n
	}
	if err := notif.ScheduleOperationalAlerts(bg); err != nil {
		t.Fatal(err)
	}
	if contar("COMERCIAL_CONTRATO_SALDO_BAIXO") != 0 || contar("COMERCIAL_CONTRATO_VENCENDO") != 1 {
		t.Fatalf("alertas: saldo %d vencendo %d (vigência acaba em 20 dias, alerta de 30)", contar("COMERCIAL_CONTRATO_SALDO_BAIXO"), contar("COMERCIAL_CONTRATO_VENCENDO"))
	}
	if _, err := uc.Chamar(a.Ctx, c.ID, contrato_venda_uc.ChamadaDTO{Linhas: []dom.LinhaChamada{{ContratoItemID: ci, Quantidade: d("5")}}, DataEntrega: entrega}); err != nil {
		t.Fatal(err)
	}
	if err := notif.ScheduleOperationalAlerts(bg); err != nil {
		t.Fatal(err)
	}
	if contar("COMERCIAL_CONTRATO_SALDO_BAIXO") != 1 {
		t.Fatal("saldo 15 de 100: alerta")
	}

	// Pedido cancelado devolve o saldo.
	testutil.Exec(t, pool, `UPDATE sales_order_items SET status='CANCELLED' WHERE sales_order_code=$1`, r2.SalesOrderCode)
	c, _ = uc.Obter(a.Ctx, c.ID)
	if !c.Itens[0].Saldo.Equal(d("65")) || !c.ValorConsumido.Equal(d("350")) {
		t.Fatalf("cancelado devolve o saldo: %s / %s", c.Itens[0].Saldo, c.ValorConsumido)
	}

	if _, err := uc.Mudar(a.Ctx, c.ID, "CANCELADO", "erro"); err == nil {
		t.Fatal("com chamada não cancela")
	}
	if _, err := uc.Mudar(a.Ctx, c.ID, "ENCERRADO", "fim do fornecimento"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Chamar(a.Ctx, c.ID, contrato_venda_uc.ChamadaDTO{Linhas: []dom.LinhaChamada{{ContratoItemID: ci, Quantidade: d("1")}}, DataEntrega: entrega}); err == nil {
		t.Fatal("encerrado não recebe chamada")
	}

	// Isolamento.
	if _, err := uc.Obter(b.Ctx, c.ID); err == nil {
		t.Fatal("a outra empresa não lê o contrato")
	}
	if l, _ := uc.Listar(b.Ctx, dom.Filtro{}); len(l) != 0 {
		t.Fatal("a outra empresa não lista os contratos")
	}
	if _, err := uc.Criar(b.Ctx, contrato_venda_uc.CriarDTO{CustomerCode: cliente, Descricao: "x", VigenciaInicio: entrega, VigenciaFim: entrega,
		Itens: []contrato_venda_uc.ItemDTO{{ItemCode: request.TextCode(fmt.Sprintf("CV%d", item)), Quantidade: d("1"), PrecoUnitario: d("1")}}}); err == nil {
		t.Fatal("a outra empresa não usa cliente/item desta")
	}
}
