//go:build integration

package production_order_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	projentity "github.com/FelipePn10/panossoerp/internal/domain/project/entity"
	projrepo "github.com/FelipePn10/panossoerp/internal/domain/project/repository"
	productionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	projectrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/project"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestOFGuardaClienteEProjeto: a OF herda cliente e projeto do pedido de
// venda (pela ordem planejada), da demanda independente e da OF pai — também
// quando o elo com a pai chega depois de a filha existir —, acompanha a troca
// feita na pai, não sobrescreve o que foi informado, e a identificação mostra
// pai, raiz e pedido. A empresa B, com o mesmo código de cliente e de projeto,
// não lê nem altera nada da A.
func TestOFGuardaClienteEProjeto(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, nil)
	repo := productionrepo.NewProductionOrderRepositoryPGX(pool)
	projetos := projectrepo.New(pool)

	rn, bu, dm := testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, actor, rn, bu, dm)
	cliente := testutil.UniqueCode() % 1_000_000_000
	// O mesmo código de cliente nas duas empresas, com nomes diferentes.
	testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES
		($1,'CLIENTE DA A','11444777000161',$2,$3), ($1,'CLIENTE DA B','11444777000161',$2,$4)`, cliente, actor, a.ID, b.ID)

	var pedido, numeroPedido, linha int64
	numeroPedido = testutil.UniqueCode() % 1_000_000
	if err := pool.QueryRow(base, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,project_code,project_name,created_by)
		VALUES($1,$2,'P',$3,'obra-x','Obra X',$4) RETURNING code`, numeroPedido, a.Code, cliente, actor).Scan(&pedido); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price)
		VALUES($1,1,$2,10,100) RETURNING code`, pedido, rn).Scan(&linha); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM sales_order_planned_orders WHERE sales_order_item_code = $1`, linha)
		testutil.Exec(t, pool, `DELETE FROM sales_order_items WHERE code = $1`, linha)
		testutil.Exec(t, pool, `DELETE FROM sales_orders WHERE code = $1`, pedido)
	})
	hoje := time.Now()
	planejada := func(item int64) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO planned_orders(order_number,item_code,quantity,order_type,demand_type,need_date,created_by,demand_code,enterprise_id,is_firm)
			VALUES($1,$2,10,'PRODUCTION','SALES_ORDER',$3::date,$4,1,$5,TRUE) RETURNING id`, testutil.UniqueCode()%1_000_000, item, hoje, actor, a.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	of := func(empresa int64, item int64, planned *int64) (int64, int64) {
		t.Helper()
		var id, numero int64
		if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,created_by,enterprise_id,planned_order_id)
			VALUES($1,$2,10,'OPEN',$3,$4,$5) RETURNING id, order_number`, testutil.UniqueCode()%1_000_000, item, actor, empresa, planned).Scan(&id, &numero); err != nil {
			t.Fatal(err)
		}
		return id, numero
	}
	origem := func(id int64) (cli, proj *int64) {
		t.Helper()
		if err := pool.QueryRow(base, `SELECT customer_code, project_id FROM production_orders WHERE id = $1`, id).Scan(&cli, &proj); err != nil {
			t.Fatal(err)
		}
		return
	}

	// 1. Pedido → OF do RN pela ordem planejada.
	rnPl, buPl := planejada(rn), planejada(bu)
	testutil.Exec(t, pool, `INSERT INTO sales_order_planned_orders(enterprise_id,sales_order_item_code,planned_order_id,quantity,created_by) VALUES($1,$2,$3,10,$4)`, a.ID, linha, rnPl, actor)
	// A bucha vira OF ANTES de o elo com o RN existir (de baixo para cima).
	buOF, _ := of(a.ID, bu, &buPl)
	if c, p := origem(buOF); c != nil || p != nil {
		t.Fatalf("a bucha sem elo não tem de onde herdar: cliente=%v projeto=%v", c, p)
	}
	rnOF, rnNumero := of(a.ID, rn, &rnPl)
	c, p := origem(rnOF)
	if c == nil || *c != cliente || p == nil {
		t.Fatalf("a OF do RN tem de herdar do pedido: cliente=%v projeto=%v", c, p)
	}
	obra, err := projetos.GetByCode(a.Ctx, "OBRA-X")
	if err != nil || obra.ID != *p || obra.Name != "Obra X" || obra.CustomerCode == nil || *obra.CustomerCode != cliente {
		t.Fatalf("o projeto do pedido entra no cadastro (código sem diferenciar maiúsculas): %+v err=%v", obra, err)
	}
	// O elo chega: a bucha herda do RN.
	testutil.Exec(t, pool, `INSERT INTO production_chain_links(enterprise_id,child_planned_order_id,parent_planned_order_id,quantity,created_by) VALUES($1,$2,$3,20,$4)`, a.ID, buPl, rnPl, actor)
	if c, p := origem(buOF); c == nil || *c != cliente || p == nil || *p != obra.ID {
		t.Fatalf("a bucha tem de herdar da OF pai quando o elo chega: cliente=%v projeto=%v", c, p)
	}

	// 2. Identificação: pai, raiz, pedido, nome do cliente da PRÓPRIA empresa.
	ident, err := repo.IdentificacaoDasOFs(a.Ctx, []int64{rnOF, buOF})
	if err != nil {
		t.Fatal(err)
	}
	x := ident[buOF]
	if x.CustomerName != "CLIENTE DA A" || x.ProjectCode != "obra-x" || x.Level != 1 ||
		x.ParentOrderNumber == nil || *x.ParentOrderNumber != rnNumero || x.RootOrderNumber == nil || *x.RootOrderNumber != rnNumero ||
		len(x.SalesOrders) != 1 || x.SalesOrders[0] != numeroPedido {
		t.Fatalf("identificação da bucha: %+v", x)
	}
	if r := ident[rnOF]; r.Level != 0 || r.Children != 1 || r.ParentOrderID != nil || len(r.SalesOrders) != 1 {
		t.Fatalf("identificação do RN (produto final com 1 filha): %+v", r)
	}

	// 3. A pai troca de projeto: a filha que herdou acompanha.
	outro, err := projetos.Create(a.Ctx, &projentity.Project{Code: "OBRA-Y", Name: "Obra Y", Status: projentity.StatusAtivo, CreatedBy: &actor})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.DefinirOrigem(a.Ctx, rnOF, &cliente, &outro.ID); err != nil {
		t.Fatal(err)
	}
	if _, p := origem(buOF); p == nil || *p != outro.ID {
		t.Fatalf("a filha herdou o projeto e tem de acompanhar a troca da pai: %v", p)
	}
	if lido, _ := projetos.GetByCode(a.Ctx, "obra-y"); lido == nil || lido.OpenOrders != 2 {
		t.Fatalf("o projeto conta as OFs abertas ligadas: %+v", lido)
	}

	// 4. Demanda independente → OF ligada pela OF (projeto digitado na demanda).
	var demanda int64
	if err := pool.QueryRow(base, `INSERT INTO independent_demands(code,item_code,quantity,demand_date,created_by,enterprise_id,customer_code,projeto)
		VALUES($1,$2,5,$3::date,$4,$5,$6,'ESTOQUE-26') RETURNING id`, testutil.UniqueCode()%1_000_000, dm, hoje, actor, a.ID, cliente).Scan(&demanda); err != nil {
		t.Fatal(err)
	}
	dmOF, _ := of(a.ID, dm, nil)
	testutil.Exec(t, pool, `INSERT INTO independent_demand_orders(enterprise_id,demand_id,production_order_id,quantity,created_by) VALUES($1,$2,$3,5,$4)`, a.ID, demanda, dmOF, actor)
	cDm, pDm := origem(dmOF)
	if cDm == nil || *cDm != cliente || pDm == nil {
		t.Fatalf("a OF da demanda herda cliente e projeto: cliente=%v projeto=%v", cDm, pDm)
	}
	if est, err := projetos.GetByCode(a.Ctx, "ESTOQUE-26"); err != nil || est.ID != *pDm {
		t.Fatalf("o projeto da demanda entra no cadastro: %+v err=%v", est, err)
	}

	// 5. Valor informado não é sobrescrito pela herança.
	proprio, _ := of(a.ID, bu, nil)
	if err := repo.DefinirOrigem(a.Ctx, proprio, nil, &obra.ID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `SELECT erp_of_herdar_origem($1)`, proprio)
	if _, p := origem(proprio); p == nil || *p != obra.ID {
		t.Fatalf("projeto informado na OF fica: %v", p)
	}

	// 6. Empresa B: mesmo código de projeto é outro cadastro; nada da A.
	if _, err := projetos.GetByCode(b.Ctx, "OBRA-X"); !errors.As(err, new(*domainerr.NotFoundError)) {
		t.Fatalf("a empresa B não enxerga o projeto da A: %v", err)
	}
	if _, err := projetos.Create(b.Ctx, &projentity.Project{Code: "OBRA-X", Name: "Obra da B", Status: projentity.StatusAtivo, CreatedBy: &actor}); err != nil {
		t.Fatalf("o código do projeto se repete entre empresas: %v", err)
	}
	if _, err := projetos.Create(a.Ctx, &projentity.Project{Code: "Obra-X", Name: "Duplicada", Status: projentity.StatusAtivo}); !errors.As(err, new(*domainerr.ConflictError)) {
		t.Fatalf("na mesma empresa o código não repete (sem diferenciar maiúsculas): %v", err)
	}
	if lista, err := projetos.List(b.Ctx, projrepo.Filtro{Busca: "obra"}); err != nil || len(lista) != 1 || lista[0].Name != "Obra da B" {
		t.Fatalf("a lista da B só traz o da B: %+v err=%v", lista, err)
	}
	if got, err := repo.IdentificacaoDasOFs(b.Ctx, []int64{rnOF, buOF}); err != nil || len(got) != 0 {
		t.Fatalf("a B não lê a identificação das OFs da A: %+v err=%v", got, err)
	}
	if err := repo.DefinirOrigem(b.Ctx, rnOF, nil, nil); !errors.As(err, new(*domainerr.NotFoundError)) {
		t.Fatalf("a B não altera a OF da A: %v", err)
	}
	itemB := testutil.UniqueCode()
	testutil.NovosItens(t, pool, b.ID, actor, itemB)
	bOF, _ := of(b.ID, itemB, nil)
	if err := repo.DefinirOrigem(b.Ctx, bOF, nil, &obra.ID); err == nil {
		t.Fatal("a OF da B não pode apontar para o projeto da A")
	}
	if nome, existe, err := repo.ClienteDaEmpresa(b.Ctx, cliente); err != nil || !existe || nome != "CLIENTE DA B" {
		t.Fatalf("o cliente com o mesmo código é o da própria empresa: %q %v", nome, err)
	}
	if c, p := origem(rnOF); c == nil || *c != cliente || p == nil || *p != outro.ID {
		t.Fatalf("a OF da A ficou como estava: cliente=%v projeto=%v", c, p)
	}
}
