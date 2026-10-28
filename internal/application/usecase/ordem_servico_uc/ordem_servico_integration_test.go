//go:build integration

package ordem_servico_uc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/ordem_servico_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/serie_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sla_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/technical_assistance_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/posvenda/ordem"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	item_ "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	ospg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/ordemservico"
	salesorder "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	seriepg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/serie"
	slapg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/slaassistencia"
	tapg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/technical_assistance"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// OS na fábrica de um equipamento em garantia: a série entra e sai da
// assistência, a peça em garantia sai do estoque pela OS (e volta se for
// removida), o deslocamento não coberto é cobrado e o faturamento gera o
// pedido de venda só do que é cobrado. Fora da garantia, a peça não sai do
// estoque pela OS: vai para o pedido e sai pela NF. Outra empresa não vê nada.
func TestOrdemDeServicoGarantiaEstoqueEFaturamento(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	equip, peca, servico := 93_000_000+u%1_000_000, 94_000_000+u%1_000_000, 95_000_000+u%1_000_000
	cliente := 810_000 + u%90_000
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []testutil.Empresa{a, b} {
			for _, q := range []string{
				`DELETE FROM os_parametros WHERE enterprise_id=$1`,
				`DELETE FROM ordens_servico WHERE enterprise_id=$1`,
				`DELETE FROM sales_order_items WHERE sales_order_code IN (SELECT code FROM sales_orders WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1))`,
				`DELETE FROM sales_orders WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1)`,
				`DELETE FROM technical_assistance_calls WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1)`,
				`DELETE FROM stock_movements WHERE enterprise_id=$1`, `DELETE FROM stock_balances WHERE enterprise_id=$1`,
				`DELETE FROM numeros_serie WHERE enterprise_id=$1`, `DELETE FROM item_pos_venda WHERE enterprise_id=$1`,
				`DELETE FROM employees WHERE enterprise_id=$1`, `DELETE FROM customers WHERE enterprise_id=$1`,
				`DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`,
			} {
				if _, err := pool.Exec(bg, q, e.ID); err != nil {
					t.Logf("limpeza %s: %v", q, err)
				}
			}
		}
	})
	var almox, tecnico int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		VALUES($1,'Assistência',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, fmt.Sprintf("OS-%d", u), actor, a.ID).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	for _, it := range []struct {
		code int64
		nome string
	}{{equip, "BALANÇA"}, {peca, "CÉLULA DE CARGA"}, {servico, "SERVIÇO DE ASSISTÊNCIA"}} {
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name,commercial_warranty_days) VALUES($1,'OS'||($1::bigint)::text,$5,$2,$3,$4,0)`, it.code, actor, a.ID, it.nome, almox)
	}
	txt := func(c int64) string { return fmt.Sprintf("OS%d", c) }
	testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'CLIENTE OS','11444777000161',$2,$3)`, cliente, actor, a.ID)
	testutil.Exec(t, pool, `INSERT INTO stock_balances(item_code,warehouse_id,quantity,reserved_qty,avg_cost,total_cost,enterprise_id) VALUES($1,$2,10,0,25,250,$3)`, peca, almox, a.ID)
	if err := pool.QueryRow(bg, `INSERT INTO employees(code,name,created_by,enterprise_id) VALUES($1,'Técnico',$2,$3) RETURNING code`, u%1_000_000_000, actor, a.ID).Scan(&tecnico); err != nil {
		t.Fatal(err)
	}

	items := item_.NewRepositoryItemSQLC(sqlc.New(pool))
	su := &serie_uc.UseCase{Repo: seriepg.New(pool), Auth: &auth.AuthService{}, Items: items}
	if _, err := su.SalvarPerfil(a.Ctx, txt(equip), serie_uc.PerfilDTO{ControlaSerie: true, GarantiaMeses: 12, CobrePecas: true, CobreMaoObra: true}); err != nil {
		t.Fatal(err)
	}
	venda := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	ss, err := su.Cadastrar(a.Ctx, serie_uc.CadastroDTO{ItemCode: request.TextCode(txt(equip)), Series: []string{"OS-1"}, Situacao: "NO_CLIENTE", CustomerCode: &cliente, DataVenda: &venda})
	if err != nil {
		t.Fatal(err)
	}
	situacaoSerie := func() string {
		var s string
		if err := pool.QueryRow(bg, `SELECT situacao FROM numeros_serie WHERE id=$1`, ss[0].ID).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	saldo := func() decimal.Decimal {
		var q decimal.Decimal
		if err := pool.QueryRow(bg, `SELECT quantity FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3`, a.ID, peca, almox).Scan(&q); err != nil {
			t.Fatal(err)
		}
		return q
	}

	ta := &technical_assistance_uc.UseCase{Repo: tapg.New(pool), Auth: &auth.AuthService{}, Garantia: su}
	serie := "OS-1"
	call, err := ta.CreateCall(a.Ctx, request.CreateTechnicalAssistanceCallDTO{EnterpriseCode: a.Code, CustomerCode: cliente, Subject: "Balança descalibrada", Priority: "HIGH",
		Items: []request.CreateTechnicalAssistanceCallItemDTO{
			{ItemCode: equip, SerialNumber: &serie, Quantity: decimal.NewFromInt(1)},
			{ItemCode: peca, Quantity: decimal.NewFromInt(1)},
		}})
	if err != nil {
		t.Fatal(err)
	}
	var itemGarantia, itemFora int64
	if err := pool.QueryRow(bg, `SELECT code FROM technical_assistance_call_items WHERE call_code=$1 AND item_code=$2`, call.Code, equip).Scan(&itemGarantia); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `SELECT code FROM technical_assistance_call_items WHERE call_code=$1 AND item_code=$2`, call.Code, peca).Scan(&itemFora); err != nil {
		t.Fatal(err)
	}

	uc := &ordem_servico_uc.UseCase{Repo: ospg.New(pool), Auth: &auth.AuthService{}, SalesOrders: salesorder.NewSalesOrderRepositorySQLC(sqlc.New(pool)), Items: items,
		SLA: &sla_uc.UseCase{Repo: slapg.New(pool), Auth: &auth.AuthService{}}}
	if _, err := uc.SalvarParametros(a.Ctx, ordem_servico_uc.ParametrosDTO{CustoHora: d("50"), PrecoHora: d("120"), CustoKM: d("1"), PrecoKM: d("2"),
		WarehouseID: &almox, ItemServico: request.TextCode(txt(servico))}); err != nil {
		t.Fatal(err)
	}
	if p, _ := uc.Parametros(a.Ctx); p.ItemServico != txt(servico) {
		t.Fatalf("o item de serviço volta pelo código de negócio: %+v", p)
	}

	o, err := uc.Abrir(a.Ctx, ordem_servico_uc.AberturaDTO{CallCode: call.Code, CallItemCode: &itemGarantia, Local: "FABRICA"})
	if err != nil {
		t.Fatal(err)
	}
	var respondido *time.Time
	if err := pool.QueryRow(bg, `SELECT respondido_em FROM technical_assistance_calls WHERE code=$1`, call.Code).Scan(&respondido); err != nil || respondido == nil {
		t.Fatalf("abrir a OS é a primeira resposta do chamado: %v %v", respondido, err)
	}
	if !o.EmGarantia || !o.CobrePecas || !o.CobreMaoObra || o.CobreDeslocamento || o.SerieID == nil || o.Situacao != dom.Aberta {
		t.Fatalf("a OS herda a garantia e a série do item do chamado: %+v", o.OS)
	}
	if _, err := uc.Iniciar(a.Ctx, o.ID, nil); err == nil {
		t.Fatal("sem técnico a OS não inicia")
	}
	if _, err := uc.Iniciar(a.Ctx, o.ID, &tecnico); err != nil {
		t.Fatal(err)
	}
	if s := situacaoSerie(); s != "EM_ASSISTENCIA" {
		t.Fatalf("OS na fábrica em execução: série em assistência, veio %s", s)
	}

	hoje := time.Now().Format("2006-01-02")
	if _, err := uc.AddHora(a.Ctx, o.ID, ordem_servico_uc.HoraDTO{Data: hoje, Tipo: "TRABALHO", Horas: d("2")}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.AddHora(a.Ctx, o.ID, ordem_servico_uc.HoraDTO{Data: hoje, Tipo: "DESLOCAMENTO", Horas: d("1")}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.AddDespesa(a.Ctx, o.ID, ordem_servico_uc.DespesaDTO{Tipo: "KM", Quantidade: d("100")}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.AddPeca(a.Ctx, o.ID, ordem_servico_uc.PecaDTO{ItemCode: request.TextCode(txt(peca)), Quantidade: d("2")}); err != nil {
		t.Fatal(err)
	}
	r, err := uc.AddPeca(a.Ctx, o.ID, ordem_servico_uc.PecaDTO{ItemCode: request.TextCode(txt(peca)), Quantidade: d("1")})
	if err != nil {
		t.Fatal(err)
	}
	if !saldo().Equal(d("7")) {
		t.Fatalf("peça em garantia sai do estoque pela OS: saldo %s", saldo())
	}
	if _, err := uc.Remover(a.Ctx, o.ID, "pecas", r.Pecas[len(r.Pecas)-1].ID); err != nil {
		t.Fatal(err)
	}
	if !saldo().Equal(d("8")) {
		t.Fatalf("peça removida volta ao estoque: saldo %s", saldo())
	}
	if _, err := uc.Cancelar(a.Ctx, o.ID, "desistência"); err == nil {
		t.Fatal("OS com peça baixada do estoque não cancela")
	}
	if _, err := uc.Concluir(a.Ctx, o.ID, ordem_servico_uc.ConclusaoDTO{Solucao: "trocada"}); err == nil {
		t.Fatal("sem o defeito constatado a OS não conclui")
	}
	r, err = uc.Concluir(a.Ctx, o.ID, ordem_servico_uc.ConclusaoDTO{DefeitoConstatado: "célula de carga danificada", Solucao: "célula trocada e balança calibrada"})
	if err != nil {
		t.Fatal(err)
	}
	if s := situacaoSerie(); s != "NO_CLIENTE" {
		t.Fatalf("OS concluída: a série volta ao cliente, veio %s", s)
	}
	// Trabalho 2h×50 + deslocamento 1h×50 + 100 km×1 + 2 peças×25 = 300 de custo;
	// cobra-se só o deslocamento (1h×120) e o km (100×2) = 320.
	if !r.Totais.CustoTotal.Equal(d("300")) || !r.Totais.ACobrar.Equal(d("320")) || !r.Totais.CustoGarantia.Equal(d("150")) {
		t.Fatalf("totais: %+v", r.Totais)
	}

	if _, err := uc.Faturar(b.Ctx, o.ID, ordem_servico_uc.FaturamentoDTO{}); err == nil {
		t.Fatal("a outra empresa não fatura a OS")
	}
	r, err = uc.Faturar(a.Ctx, o.ID, ordem_servico_uc.FaturamentoDTO{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Situacao != dom.Faturada || r.SalesOrderCode == nil {
		t.Fatalf("OS faturada com o pedido: %+v", r.OS)
	}
	var linhas int
	var total decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT COUNT(*), COALESCE(SUM(requested_qty*unit_price),0) FROM sales_order_items WHERE sales_order_code=$1`, *r.SalesOrderCode).Scan(&linhas, &total); err != nil {
		t.Fatal(err)
	}
	if linhas != 1 || !total.Equal(d("320")) {
		t.Fatalf("pedido da OS em garantia: só a linha de serviço de 320 — %d linhas, %s", linhas, total)
	}
	// O pedido nasce totalizado (linha e capa) e a OS não se fatura de novo.
	var linhaTotal, capaTotal decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT i.total_gross, o.total_gross FROM sales_order_items i JOIN sales_orders o ON o.code=i.sales_order_code WHERE o.code=$1`, *r.SalesOrderCode).Scan(&linhaTotal, &capaTotal); err != nil {
		t.Fatal(err)
	}
	if !linhaTotal.Equal(d("320")) || !capaTotal.Equal(d("320")) {
		t.Fatalf("pedido totalizado: linha %s capa %s", linhaTotal, capaTotal)
	}
	var pedidosAntes, pedidosDepois int
	_ = pool.QueryRow(bg, `SELECT COUNT(*) FROM sales_orders WHERE enterprise_code=$1`, a.Code).Scan(&pedidosAntes)
	if _, err := uc.Faturar(a.Ctx, o.ID, ordem_servico_uc.FaturamentoDTO{}); err == nil {
		t.Fatal("OS já faturada não fatura de novo")
	}
	_ = pool.QueryRow(bg, `SELECT COUNT(*) FROM sales_orders WHERE enterprise_code=$1`, a.Code).Scan(&pedidosDepois)
	if pedidosDepois != pedidosAntes {
		t.Fatalf("a segunda tentativa não deixa pedido: %d → %d", pedidosAntes, pedidosDepois)
	}

	// Fora da garantia, no cliente: a peça cobrada exige preço e não sai do estoque pela OS.
	o2, err := uc.Abrir(a.Ctx, ordem_servico_uc.AberturaDTO{CallCode: call.Code, CallItemCode: &itemFora, Local: "CLIENTE", TecnicoCode: &tecnico})
	if err != nil {
		t.Fatal(err)
	}
	if o2.EmGarantia {
		t.Fatal("peça sem perfil nem NF: fora da garantia")
	}
	if _, err := uc.Iniciar(a.Ctx, o2.ID, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.AddPeca(a.Ctx, o2.ID, ordem_servico_uc.PecaDTO{ItemCode: request.TextCode(txt(peca)), Quantidade: d("1")}); err == nil {
		t.Fatal("peça cobrada sem preço")
	}
	r2, err := uc.AddPeca(a.Ctx, o2.ID, ordem_servico_uc.PecaDTO{ItemCode: request.TextCode(txt(peca)), Quantidade: d("1"), PrecoUnitario: d("90")})
	if err != nil {
		t.Fatal(err)
	}
	if !saldo().Equal(d("8")) || r2.Pecas[0].StockMovementID != nil || !r2.Pecas[0].Cobrado || !r2.Pecas[0].CustoUnitario.Equal(d("25")) {
		t.Fatalf("peça cobrada fica a faturar: saldo %s, %+v", saldo(), r2.Pecas[0])
	}
	if _, err := uc.AddHora(a.Ctx, o2.ID, ordem_servico_uc.HoraDTO{Data: hoje, Horas: d("1.5")}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Concluir(a.Ctx, o2.ID, ordem_servico_uc.ConclusaoDTO{DefeitoConstatado: "desgaste", Solucao: "troca"}); err != nil {
		t.Fatal(err)
	}
	r2, err = uc.Faturar(a.Ctx, o2.ID, ordem_servico_uc.FaturamentoDTO{})
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `SELECT COUNT(*), COALESCE(SUM(requested_qty*unit_price),0) FROM sales_order_items WHERE sales_order_code=$1`, *r2.SalesOrderCode).Scan(&linhas, &total); err != nil {
		t.Fatal(err)
	}
	if linhas != 2 || !total.Equal(d("270")) {
		t.Fatalf("pedido fora da garantia: peça 90 + 1,5h×120 = 270 — %d linhas, %s", linhas, total)
	}

	// Isolamento: a outra empresa não lista, não lê e não abre OS pelo chamado alheio.
	if l, err := uc.Listar(b.Ctx, dom.Filtro{}); err != nil || len(l) != 0 {
		t.Fatalf("a outra empresa não vê as OS: %d %v", len(l), err)
	}
	if _, err := uc.Obter(b.Ctx, o.ID); err == nil {
		t.Fatal("a outra empresa não lê a OS")
	}
	if _, err := uc.Abrir(b.Ctx, ordem_servico_uc.AberturaDTO{CallCode: call.Code, Local: "CLIENTE"}); err == nil {
		t.Fatal("a outra empresa não abre OS pelo chamado alheio")
	}
}
