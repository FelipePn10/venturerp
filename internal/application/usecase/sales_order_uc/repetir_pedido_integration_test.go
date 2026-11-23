//go:build integration

package sales_order_uc_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	customerrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	salesorder "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Melhoria 13 (B2B): repetir o pedido cria um rascunho com as mesmas linhas e o
// preço de HOJE da tabela; a empresa B não repete o pedido da A.
func TestRepetirPedidoComPrecoDeHoje(t *testing.T) {
	q, pool := testutil.Queries(t)
	bg := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		pool.Exec(bg, `DELETE FROM sales_order_items WHERE sales_order_code IN (SELECT code FROM sales_orders WHERE enterprise_code=$1)`, a.Code)
		pool.Exec(bg, `DELETE FROM sales_orders WHERE enterprise_code=$1`, a.Code)
		pool.Exec(bg, `DELETE FROM sales_table_prices WHERE sales_table_id IN (SELECT id FROM sales_tables WHERE enterprise_id=$1)`, a.ID)
		pool.Exec(bg, `DELETE FROM sales_tables WHERE enterprise_id=$1`, a.ID)
		pool.Exec(bg, `DELETE FROM customers WHERE enterprise_id=$1`, a.ID)
	})
	item := testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, item)
	cliente, tabela := testutil.UniqueCode(), testutil.UniqueCode()%1_000_000+70_000_000
	testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'CLIENTE B2B','11444777000161',$2,$3)`, cliente, ator, a.ID)
	var tabelaID, pedido int64
	if err := pool.QueryRow(bg, `INSERT INTO sales_tables(code,description,enterprise_id) VALUES($1,'B2B',$2) RETURNING id`, tabela, a.ID).Scan(&tabelaID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO sales_table_prices(sales_table_id,item_code,price) VALUES($1,$2,12)`, tabelaID, fmt.Sprint(item))
	if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by,price_table_code,currency_code) VALUES($1,$2,'F',$3,$4,$5,'BRL') RETURNING code`,
		testutil.UniqueCode()%900_000, a.Code, cliente, ator, tabela).Scan(&pedido); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price) VALUES($1,1,$2,3,9)`, pedido, item)
	repo := salesorder.NewSalesOrderRepositorySQLC(q)
	az := &auth.AuthService{}
	uc := &sales_order_uc.RepetirPedidoUseCase{
		Pedido: &sales_order_uc.CreateSalesOrderUseCase{Repo: repo, Auth: az},
		Linha:  &sales_order_uc.CreateSalesOrderItemUseCase{Repo: repo, Auth: az, Items: itemrepo.NewRepositoryItemSQLC(q).ComPool(pool), Prices: customerrepo.New(q, pool)},
	}
	if _, err := uc.Repetir(b.Ctx, pedido); err == nil {
		t.Fatal("a empresa B repetiu o pedido da A")
	}
	res, err := uc.Repetir(a.Ctx, pedido)
	if err != nil {
		t.Fatal(err)
	}
	if res.Linhas != 1 || res.Pedido.Code == pedido {
		t.Fatalf("resultado: %+v", res)
	}
	var preco, qtd decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT unit_price, requested_qty FROM sales_order_items WHERE sales_order_code=$1`, res.Pedido.Code).Scan(&preco, &qtd); err != nil {
		t.Fatal(err)
	}
	if !preco.Equal(decimal.NewFromInt(12)) || !qtd.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("linha repetida: preço %s (quer o de hoje, 12), quantidade %s", preco, qtd)
	}

	// Capa sem tabela, linha com tabela (pedido antigo): repete pela tabela da linha.
	var semCapa int64
	if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by,currency_code) VALUES($1,$2,'F',$3,$4,'BRL') RETURNING code`,
		testutil.UniqueCode()%900_000, a.Code, cliente, ator).Scan(&semCapa); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,price_table_code) VALUES($1,1,$2,2,9,$3)`, semCapa, item, tabela)
	if res, err := uc.Repetir(a.Ctx, semCapa); err != nil || res.Linhas != 1 {
		t.Fatalf("repetição pela tabela da linha: %+v err=%v", res, err)
	}

	// Nenhuma linha válida: erro, e o rascunho criado não fica aberto na carteira.
	var vazio int64
	if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by,currency_code) VALUES($1,$2,'F',$3,$4,'BRL') RETURNING code`,
		testutil.UniqueCode()%900_000, a.Code, cliente, ator).Scan(&vazio); err != nil {
		t.Fatal(err)
	}
	var antes int64
	_ = pool.QueryRow(bg, `SELECT COALESCE(MAX(code),0) FROM sales_orders`).Scan(&antes)
	if _, err := uc.Repetir(a.Ctx, vazio); err == nil {
		t.Fatal("repetir pedido sem linha deveria falhar")
	}
	var abertos int
	_ = pool.QueryRow(bg, `SELECT COUNT(*) FROM sales_orders WHERE code > $1 AND status <> 'CANCELLED'`, antes).Scan(&abertos)
	if abertos != 0 {
		t.Fatalf("ficaram %d rascunho(s) vazio(s) abertos", abertos)
	}
}
