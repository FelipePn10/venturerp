//go:build integration

package portal_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/portal_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	customerrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/portal"
	salesorder "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

type documentosFalsos struct{}

func (documentosFalsos) Execute(_ context.Context, id int64) (*fiscal_uc.DANFEResult, error) {
	return &fiscal_uc.DANFEResult{ExitID: id, DanfeURL: "https://emissora/danfe.pdf", XMLURL: "https://emissora/nfe.xml"}, nil
}

// Portal fase 0: o link do cliente X da empresa A só mostra o que é de X em A;
// vencido ou revogado não abre; o pedido de outro cliente não se repete; a
// repetição vira rascunho PORTAL; a empresa B não revoga nem lista o link da A.
func TestPortalIsolaClienteEEmpresa(t *testing.T) {
	q, pool := testutil.Queries(t)
	bg := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []testutil.Empresa{a, b} {
			pool.Exec(bg, `DELETE FROM portal_links WHERE enterprise_id=$1`, e.ID)
			pool.Exec(bg, `DELETE FROM notification_outbox WHERE enterprise_id=$1`, e.ID)
			pool.Exec(bg, `DELETE FROM fiscal_exits WHERE enterprise_id=$1`, e.ID)
			pool.Exec(bg, `DELETE FROM sales_order_items WHERE sales_order_code IN (SELECT code FROM sales_orders WHERE enterprise_code=$1)`, e.Code)
			pool.Exec(bg, `DELETE FROM sales_orders WHERE enterprise_code=$1`, e.Code)
			pool.Exec(bg, `DELETE FROM sales_table_prices WHERE sales_table_id IN (SELECT id FROM sales_tables WHERE enterprise_id=$1)`, e.ID)
			pool.Exec(bg, `DELETE FROM sales_tables WHERE enterprise_id=$1`, e.ID)
			pool.Exec(bg, `DELETE FROM customers WHERE enterprise_id=$1`, e.ID)
			pool.Exec(bg, `DELETE FROM user_enterprises WHERE enterprise_id=$1`, e.ID)
		}
	})
	// O portal grava em nome de um ADMIN da empresa.
	testutil.Exec(t, pool, `INSERT INTO user_enterprises(user_id,enterprise_id,role) VALUES($1,$2,'ADMIN') ON CONFLICT DO NOTHING`, ator, a.ID)
	item := testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, item)
	x, y, tabela := testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode()%1_000_000+71_000_000
	for _, c := range []int64{x, y} {
		testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'CLIENTE PORTAL','11444777000161',$2,$3)`, c, ator, a.ID)
	}
	var tabelaID int64
	if err := pool.QueryRow(bg, `INSERT INTO sales_tables(code,description,enterprise_id) VALUES($1,'PORTAL',$2) RETURNING id`, tabela, a.ID).Scan(&tabelaID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO sales_table_prices(sales_table_id,item_code,price) VALUES($1,$2,15)`, tabelaID, fmt.Sprint(item))
	pedido := func(cliente int64) int64 {
		var code int64
		if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by,price_table_code,currency_code,emission_date) VALUES($1,$2,'F',$3,$4,$5,'BRL',$6::date) RETURNING code`,
			testutil.UniqueCode()%900_000, a.Code, cliente, ator, tabela, time.Now()).Scan(&code); err != nil {
			t.Fatal(err)
		}
		testutil.Exec(t, pool, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price) VALUES($1,1,$2,4,10)`, code, item)
		return code
	}
	doX, doY := pedido(x), pedido(y)

	repo := salesorder.NewSalesOrderRepositorySQLC(q)
	az := &auth.AuthService{}
	uc := &portal_uc.UseCase{Repo: portal.New(pool), Auth: az, BaseURL: "https://portal.exemplo",
		Repetir: &sales_order_uc.RepetirPedidoUseCase{
			Pedido: &sales_order_uc.CreateSalesOrderUseCase{Repo: repo, Auth: az},
			Linha:  &sales_order_uc.CreateSalesOrderItemUseCase{Repo: repo, Auth: az, Items: itemrepo.NewRepositoryItemSQLC(q).ComPool(pool), Prices: customerrepo.New(q, pool)},
		}}

	if _, err := uc.CriarLink(b.Ctx, x, 30, ""); err == nil {
		t.Fatal("a empresa B criou link para o cliente da A")
	}
	link, err := uc.CriarLink(a.Ctx, x, 30, "compras")
	if err != nil {
		t.Fatal(err)
	}
	uc.BaseURL = ""
	if _, err := uc.CriarLink(a.Ctx, x, 1, ""); err == nil {
		t.Fatal("sem PORTAL_BASE_URL o link não teria onde abrir: precisa recusar")
	}
	uc.BaseURL = "https://portal.exemplo"
	uc.Empresa = "usimac"
	if l2, err := uc.CriarLink(a.Ctx, x, 1, ""); err != nil || l2.URL != "https://portal.exemplo/#/portal/usimac/"+l2.Token {
		t.Fatalf("link com a empresa do catálogo: %+v %v", l2, err)
	}
	uc.Empresa = ""
	if link.URL != "https://portal.exemplo/#/portal/"+link.Token || len(link.Token) < 40 {
		t.Fatalf("link: %+v", link)
	}
	var hashGuardado []byte
	_ = pool.QueryRow(bg, `SELECT token_hash FROM portal_links WHERE id=$1`, link.ID).Scan(&hashGuardado)
	if string(hashGuardado) == link.Token {
		t.Fatal("o banco guardou o token em claro")
	}

	s, err := uc.Abrir(bg, link.Token, "127.0.0.1", "teste")
	if err != nil {
		t.Fatal(err)
	}
	if s.EnterpriseID != a.ID || s.CustomerCode != x {
		t.Fatalf("sessão: %+v", s)
	}
	res, err := uc.Resumo(a.Ctx, s, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Pedidos) != 1 || res.Pedidos[0].Code != doX || len(res.Pedidos[0].Linhas) != 1 {
		t.Fatalf("o cliente X precisa ver só o pedido dele: %+v", res.Pedidos)
	}
	if _, err := uc.PedirDeNovo(a.Ctx, s, doY, "", "", ""); !errors.Is(err, portal_uc.ErrNaoEncontrado) {
		t.Fatalf("o cliente X repetiu o pedido do Y: %v", err)
	}
	rep, err := uc.PedirDeNovo(a.Ctx, s, doX, "entregar na filial", "", "")
	if err != nil || rep.Linhas != 1 {
		t.Fatalf("repetição pelo portal: %+v err=%v", rep, err)
	}
	var origem, status, notas string
	_ = pool.QueryRow(bg, `SELECT origin, status, COALESCE(notes,'') FROM sales_orders WHERE code=$1`, rep.Pedido.Code).Scan(&origem, &status, &notas)
	if origem != "PORTAL" || status != "R" || notas == "" {
		t.Fatalf("rascunho do portal: origem %s situação %s notas %q", origem, status, notas)
	}
	var eventos, alertas int
	_ = pool.QueryRow(bg, `SELECT COUNT(*) FROM portal_events WHERE link_id=$1`, link.ID).Scan(&eventos)
	if eventos != 2 {
		t.Fatalf("auditoria: %d evento(s), quer acesso + pedido", eventos)
	}
	_ = pool.QueryRow(bg, `SELECT COUNT(*) FROM notification_outbox WHERE enterprise_id=$1 AND event_key='COMERCIAL_PEDIDO_PORTAL'`, a.ID).Scan(&alertas)
	if alertas != 1 {
		t.Fatalf("o comercial precisa ser avisado do pedido do portal: %d alerta(s)", alertas)
	}

	// Pedido antigo sem tabela de preço: o pedido do portal NÃO se perde — fica
	// o rascunho com a lista do que o cliente pediu, e o comercial é avisado.
	var semTabela int64
	if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by,currency_code,emission_date) VALUES($1,$2,'F',$3,$4,'BRL',$5::date) RETURNING code`,
		testutil.UniqueCode()%900_000, a.Code, x, ator, time.Now()).Scan(&semTabela); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price) VALUES($1,1,$2,7,10)`, semTabela, item)
	repSem, err := uc.PedirDeNovo(a.Ctx, s, semTabela, "urgente", "", "")
	if err != nil || repSem.Linhas != 0 {
		t.Fatalf("pedido do portal sem tabela precisa ser recebido mesmo assim: %+v err=%v", repSem, err)
	}
	var stSem, notasSem string
	_ = pool.QueryRow(bg, `SELECT status, COALESCE(notes,'') FROM sales_orders WHERE code=$1`, repSem.Pedido.Code).Scan(&stSem, &notasSem)
	if stSem != "R" || !strings.Contains(notasSem, "Itens pedidos: 7 ×") || !strings.Contains(notasSem, "urgente") {
		t.Fatalf("rascunho do portal sem tabela: situação %s notas %q", stSem, notasSem)
	}
	// O cliente vê o que pediu no rascunho ainda sem itens, com o preço a confirmar.
	if res2, err := uc.Resumo(a.Ctx, s, "", ""); err != nil {
		t.Fatal(err)
	} else {
		achou := false
		for _, p := range res2.Pedidos {
			if p.Code == repSem.Pedido.Code {
				achou = len(p.Linhas) == 1 && p.Linhas[0].APreco && p.Linhas[0].Quantidade.Equal(decimal.NewFromInt(7))
			}
		}
		if !achou {
			t.Fatal("o rascunho do portal precisa mostrar ao cliente os itens que ele pediu")
		}
	}
	_ = pool.QueryRow(bg, `SELECT COUNT(*) FROM notification_outbox WHERE enterprise_id=$1 AND event_key='COMERCIAL_PEDIDO_PORTAL'`, a.ID).Scan(&alertas)
	if alertas != 2 {
		t.Fatalf("o comercial precisa ser avisado também deste pedido: %d alerta(s)", alertas)
	}

	// Dois cliques no mesmo instante: só um rascunho nasce.
	var duplo int64
	if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by,currency_code,emission_date) VALUES($1,$2,'F',$3,$4,'BRL',$5::date) RETURNING code`,
		testutil.UniqueCode()%900_000, a.Code, x, ator, time.Now()).Scan(&duplo); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price) VALUES($1,1,$2,3,10)`, duplo, item)
	var wg sync.WaitGroup
	var certos atomic.Int32
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := uc.PedirDeNovo(a.Ctx, s, duplo, "", "", ""); err == nil {
				certos.Add(1)
			}
		}()
	}
	wg.Wait()
	if certos.Load() != 1 {
		t.Fatalf("cliques simultâneos criaram %d rascunhos (esperado 1)", certos.Load())
	}

	// Orçamento e rascunho do vendedor não se repetem pelo portal.
	for _, st := range []string{"OF", "R"} {
		var cod int64
		if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by,currency_code,emission_date) VALUES($1,$2,$3,$4,$5,'BRL',$6::date) RETURNING code`,
			testutil.UniqueCode()%900_000, a.Code, st, x, ator, time.Now()).Scan(&cod); err != nil {
			t.Fatal(err)
		}
		if _, err := uc.PedirDeNovo(a.Ctx, s, cod, "", "", ""); !errors.Is(err, portal_uc.ErrNaoEncontrado) {
			t.Fatalf("pedido %s não pode ser repetido pelo portal: %v", st, err)
		}
	}

	// Limites: o mesmo pedido não é pedido de novo enquanto o anterior aguarda,
	// e cliente inativado perde o acesso.
	if _, err := uc.PedirDeNovo(a.Ctx, s, semTabela, "", "", ""); err == nil || !strings.Contains(err.Error(), "já pediu este pedido de novo") {
		t.Fatalf("repetição em dobro do mesmo pedido precisa ser barrada: %v", err)
	}

	// Nota do cliente: baixa; nota de outro cliente: não existe para ele.
	nota := func(pedido int64) int64 {
		var id int64
		if err := pool.QueryRow(bg, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,created_by,enterprise_id,status,sales_order_code,danfe_path)
			VALUES($1,$2::date,'5101','Venda',$3,$4,'AUTHORIZED',$5,'/arquivos/danfe.pdf') RETURNING id`, testutil.UniqueCode()%1_000_000, time.Now(), ator, a.ID, pedido).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	notaX, notaY := nota(doX), nota(doY)
	uc.Documentos = documentosFalsos{}
	if url, nome, err := uc.Documento(a.Ctx, s, notaX, "danfe", "", ""); err != nil || url == "" || nome == "" {
		t.Fatalf("DANFE do cliente: %q %q %v", url, nome, err)
	}
	if _, _, err := uc.Documento(a.Ctx, s, notaY, "xml", "", ""); !errors.Is(err, portal_uc.ErrNaoEncontrado) {
		t.Fatalf("o cliente X baixou a nota do Y: %v", err)
	}

	testutil.Exec(t, pool, `UPDATE customers SET is_active = false WHERE enterprise_id = $1 AND code = $2`, a.ID, x)
	if _, err := uc.Abrir(bg, link.Token, "", ""); !errors.Is(err, portal_uc.ErrNaoEncontrado) {
		t.Fatalf("cliente inativo não pode abrir o portal: %v", err)
	}
	if _, err := uc.CriarLink(a.Ctx, x, 30, ""); err == nil {
		t.Fatal("link novo para cliente inativo")
	}
	testutil.Exec(t, pool, `UPDATE customers SET is_active = true WHERE enterprise_id = $1 AND code = $2`, a.ID, x)

	if l, _ := uc.ListarLinks(b.Ctx, x); len(l) != 0 {
		t.Fatal("a empresa B listou o link da A")
	}
	if err := uc.RevogarLink(b.Ctx, link.ID); err == nil {
		t.Fatal("a empresa B revogou o link da A")
	}
	if err := uc.RevogarLink(a.Ctx, link.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Abrir(bg, link.Token, "", ""); !errors.Is(err, portal_uc.ErrNaoEncontrado) {
		t.Fatalf("link revogado abriu: %v", err)
	}
	vencido, _ := uc.CriarLink(a.Ctx, x, 1, "")
	testutil.Exec(t, pool, `UPDATE portal_links SET expires_at = now() - interval '1 day' WHERE id=$1`, vencido.ID)
	if _, err := uc.Abrir(bg, vencido.Token, "", ""); !errors.Is(err, portal_uc.ErrNaoEncontrado) {
		t.Fatalf("link vencido abriu: %v", err)
	}
	if _, err := uc.Abrir(bg, "token-que-nao-existe-de-jeito-nenhum-xx", "", ""); !errors.Is(err, portal_uc.ErrNaoEncontrado) {
		t.Fatalf("token inventado: %v", err)
	}
}
