//go:build integration

// Testes do esquema de rateio de indiretos e do isolamento do custo por empresa
// (migração 000373).
//
// Dois defeitos motivam este arquivo:
//
//  1. `overhead_cost` era gravado SEMPRE ZERO. A coluna existia, o comentário do
//     código dizia "currently 0 unless configured" e não havia como configurar —
//     custo indireto simplesmente não entrava no custo do produto.
//
//  2. `item_standard_costs` e `item_purchase_costs` não têm coluna de empresa e o
//     repositório não conferia posse. Uma empresa podia LER e SOBRESCREVER o custo
//     de um item de outra passando o código dele.
//
//     TEST_DATABASE_URL=... go test -tags=integration -run "Rateio|CustoIsola" ./internal/application/usecase/cost_uc/
package cost_uc_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/cost_uc"
	standardCostRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/standard_cost"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/jackc/pgx/v5/pgxpool"
)

func pertoDe(a, b float64) bool { return math.Abs(a-b) < 0.01 }

// empresaNova cria uma empresa e devolve a sessão dela. testutil.TenantContext
// devolve sempre a PRIMEIRA empresa do banco, então dois contextos dele são a
// MESMA empresa — e um teste de isolamento com eles não prova nada.
func empresaNova(t *testing.T, pool *pgxpool.Pool) context.Context {
	t.Helper()
	ator := testutil.Actor(t, pool)
	code := 870_000_000 + (testutil.UniqueCode() % 100_000_000)
	var id int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO enterprise (code, name, created_by) VALUES ($1, 'CUSTO VIZINHA', $2) RETURNING id`,
		code, ator).Scan(&id); err != nil {
		t.Fatalf("criando empresa vizinha: %v", err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM enterprise WHERE id = $1`, id) })
	return context.WithValue(context.Background(), contextkey.UserKey,
		&security.AuthUser{ID: ator.String(), Role: "ADMIN", EnterpriseID: id, EnterpriseCode: code})
}

// TestRateioEntraNoCustoDoProduto: o teste do defeito principal. Sem regra, o
// indireto é zero; com regra, o indireto aparece no custo e no rastro.
func TestRateioEntraNoCustoDoProduto(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := standardCostRepo.New(q, pool)
	uc := cost_uc.New(repo)
	ctx := testutil.TenantContext(t, pool)
	uid := testutil.Actor(t, pool)

	// Item folha com custo de compra de 200: é a base do percentual.
	itemCode := testutil.UniqueCode()
	testutil.SeedItem(t, pool, ctx, itemCode, uid)
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM item_standard_cost_history WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM cost_rollup_log WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM item_standard_costs WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM item_purchase_costs WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM items WHERE code = $1", itemCode)
	})
	if _, err := uc.UpsertItemPurchaseCost(ctx, request.UpsertItemPurchaseCostDTO{
		ItemCode: itemCode, UnitCost: 200, Currency: "BRL", UpdatedBy: uid.String(),
	}); err != nil {
		t.Fatalf("gravando custo de compra: %v", err)
	}

	// Controle negativo: SEM regra cadastrada, o indireto tem de ser zero. Sem esta
	// medida, um rateio que aplicasse sempre passaria pelo teste seguinte.
	semRegra, err := uc.RollUp(ctx, request.CostRollupDTO{ItemCode: itemCode, CalculatedBy: uid.String()})
	if err != nil {
		t.Fatalf("apuração sem regra: %v", err)
	}
	if semRegra.OverheadCost != 0 {
		t.Fatalf("sem regra de rateio o indireto deveria ser 0, veio %.4f", semRegra.OverheadCost)
	}
	if !pertoDe(semRegra.TotalCost, 200) {
		t.Fatalf("total sem regra = %.4f, esperado 200", semRegra.TotalCost)
	}

	regra, err := uc.CriarRegraDeRateio(ctx, request.CostOverheadRuleDTO{
		Code: "CIF-MAT", Description: "Indiretos de armazenagem sobre material",
		Base: "MATERIAL", Method: "PERCENTUAL", Rate: 0.12,
		ValidFrom: time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
		CreatedBy: uid.String(),
	})
	if err != nil {
		t.Fatalf("criando regra de rateio: %v", err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, "DELETE FROM cost_overhead_rules WHERE id = $1", regra.ID) })

	comRegra, err := uc.RollUp(ctx, request.CostRollupDTO{ItemCode: itemCode, CalculatedBy: uid.String()})
	if err != nil {
		t.Fatalf("apuração com regra: %v", err)
	}
	if !pertoDe(comRegra.OverheadCost, 24) {
		t.Fatalf("indireto = %.4f, esperado 24 (12%% de 200)", comRegra.OverheadCost)
	}
	if !pertoDe(comRegra.TotalCost, 224) {
		t.Fatalf("total = %.4f, esperado 224 — o indireto tem de entrar no total", comRegra.TotalCost)
	}
	// O rastro é o que torna o indireto explicável. Um número no total sem rastro é
	// exatamente o que não se consegue defender numa análise de custo.
	if len(comRegra.Overheads) != 1 {
		t.Fatalf("rastro do rateio com %d linha(s), esperado 1", len(comRegra.Overheads))
	}
	rastro := comRegra.Overheads[0]
	if rastro.Code != "CIF-MAT" || !pertoDe(rastro.BaseValue, 200) || !pertoDe(rastro.Applied, 24) {
		t.Fatalf("rastro do rateio errado: %+v", rastro)
	}

	// Regra fora de vigência não pode aplicar: custo errado que sai plausível é o
	// que ninguém questiona.
	fim := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if _, err := uc.AtualizarRegraDeRateio(ctx, regra.ID, request.CostOverheadRuleDTO{
		Code: "CIF-MAT", Description: "Indiretos de armazenagem sobre material",
		Base: "MATERIAL", Method: "PERCENTUAL", Rate: 0.12,
		ValidFrom: time.Now().AddDate(0, 0, -30).Format("2006-01-02"), ValidTo: &fim,
		CreatedBy: uid.String(),
	}); err != nil {
		t.Fatalf("encerrando a vigência: %v", err)
	}
	expirada, err := uc.RollUp(ctx, request.CostRollupDTO{ItemCode: itemCode, CalculatedBy: uid.String()})
	if err != nil {
		t.Fatalf("apuração com regra expirada: %v", err)
	}
	if expirada.OverheadCost != 0 {
		t.Fatalf("regra vencida ontem aplicou %.4f de indireto", expirada.OverheadCost)
	}

	// Desativar não apaga: o histórico aponta para a regra, e apagá-la deixaria a
	// apuração antiga sem explicação.
	if err := uc.DesativarRegraDeRateio(ctx, regra.ID); err != nil {
		t.Fatalf("desativando regra: %v", err)
	}
	regras, err := uc.ListarRegrasDeRateio(ctx)
	if err != nil {
		t.Fatalf("listando regras: %v", err)
	}
	var achou bool
	for _, r := range regras {
		if r.ID == regra.ID {
			achou = true
			if r.IsActive {
				t.Fatal("a regra desativada continua ativa")
			}
		}
	}
	if !achou {
		t.Fatal("a regra desativada desapareceu da listagem — o histórico ficaria sem explicação")
	}
}

// TestRateioGravaHistoricoPorApuracao: sem histórico não se responde "por que o
// custo subiu 12% este mês".
func TestRateioGravaHistoricoPorApuracao(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := cost_uc.New(standardCostRepo.New(q, pool))
	ctx := testutil.TenantContext(t, pool)
	uid := testutil.Actor(t, pool)

	itemCode := testutil.UniqueCode()
	testutil.SeedItem(t, pool, ctx, itemCode, uid)
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM item_standard_cost_history WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM cost_rollup_log WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM item_standard_costs WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM item_purchase_costs WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM items WHERE code = $1", itemCode)
	})

	if _, err := uc.UpsertItemPurchaseCost(ctx, request.UpsertItemPurchaseCostDTO{
		ItemCode: itemCode, UnitCost: 100, UpdatedBy: uid.String(),
	}); err != nil {
		t.Fatalf("custo de compra: %v", err)
	}
	if _, err := uc.RollUp(ctx, request.CostRollupDTO{ItemCode: itemCode, CalculatedBy: uid.String()}); err != nil {
		t.Fatalf("1ª apuração: %v", err)
	}
	// Material sobe 50%: é a variação que o histórico tem de mostrar.
	if _, err := uc.UpsertItemPurchaseCost(ctx, request.UpsertItemPurchaseCostDTO{
		ItemCode: itemCode, UnitCost: 150, UpdatedBy: uid.String(),
	}); err != nil {
		t.Fatalf("custo de compra novo: %v", err)
	}
	if _, err := uc.RollUp(ctx, request.CostRollupDTO{ItemCode: itemCode, CalculatedBy: uid.String()}); err != nil {
		t.Fatalf("2ª apuração: %v", err)
	}

	historico, err := uc.HistoricoDeCusto(ctx, itemCode, "", 0)
	if err != nil {
		t.Fatalf("lendo histórico: %v", err)
	}
	if len(historico) != 2 {
		t.Fatalf("histórico com %d apuração(ões), esperado 2 — o upsert não pode sobrescrever o histórico", len(historico))
	}
	// Mais recente primeiro: é a ordem que a tela usa para comparar com a anterior.
	if !pertoDe(historico[0].TotalCost, 150) || !pertoDe(historico[1].TotalCost, 100) {
		t.Fatalf("histórico fora de ordem ou com valores errados: %.2f e %.2f", historico[0].TotalCost, historico[1].TotalCost)
	}

	// Histórico é fato consumado: o banco recusa alteração.
	if _, err := pool.Exec(ctx, "UPDATE item_standard_cost_history SET total_cost = 1 WHERE id = $1", historico[0].ID); err == nil {
		t.Fatal("o histórico de custo aceitou ser alterado")
	}
	if _, err := pool.Exec(ctx, "DELETE FROM item_standard_cost_history WHERE id = $1", historico[0].ID); err == nil {
		t.Fatal("o histórico de custo aceitou ser apagado")
	}
}

// TestCustoIsolaEntreEmpresas: o item de uma empresa não pode ter o custo lido nem
// sobrescrito por outra. As tabelas não têm coluna de empresa — a posse vem de
// items.enterprise_id, e é essa conferência que os testes cobram.
func TestCustoIsolaEntreEmpresas(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := cost_uc.New(standardCostRepo.New(q, pool))
	dona := testutil.TenantContext(t, pool)
	vizinha := empresaNova(t, pool)
	uid := testutil.Actor(t, pool)

	itemCode := testutil.UniqueCode()
	testutil.SeedItem(t, pool, dona, itemCode, uid)
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM item_standard_cost_history WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM cost_rollup_log WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM item_standard_costs WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM item_purchase_costs WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM items WHERE code = $1", itemCode)
	})
	if _, err := uc.UpsertItemPurchaseCost(dona, request.UpsertItemPurchaseCostDTO{
		ItemCode: itemCode, UnitCost: 777, UpdatedBy: uid.String(),
	}); err != nil {
		t.Fatalf("a dona não conseguiu gravar o custo do próprio item: %v", err)
	}

	// Leitura pela vizinha: recusada.
	if _, err := uc.GetItemPurchaseCost(vizinha, itemCode); err == nil {
		t.Fatal("a empresa vizinha leu o custo de compra de um item alheio")
	}
	if _, err := uc.GetStandardCost(vizinha, itemCode, ""); err == nil {
		t.Fatal("a empresa vizinha leu o custo-padrão de um item alheio")
	}
	if _, err := uc.HistoricoDeCusto(vizinha, itemCode, "", 0); err == nil {
		t.Fatal("a empresa vizinha leu o histórico de custo de um item alheio")
	}
	// Apuração pela vizinha: recusada. Sem isso, ela recalcularia e SOBRESCREVERIA o
	// custo-padrão do item de outra empresa.
	if _, err := uc.RollUp(vizinha, request.CostRollupDTO{ItemCode: itemCode, CalculatedBy: uid.String()}); err == nil {
		t.Fatal("a empresa vizinha apurou o custo de um item alheio")
	}
	// Sobrescrita direta pela vizinha: recusada.
	if _, err := uc.UpsertItemPurchaseCost(vizinha, request.UpsertItemPurchaseCostDTO{
		ItemCode: itemCode, UnitCost: 1, UpdatedBy: uid.String(),
	}); err == nil {
		t.Fatal("a empresa vizinha sobrescreveu o custo de compra de um item alheio")
	}

	// Controle positivo: o custo da dona continua intacto e legível por ela — senão
	// as recusas acima passariam com o repositório simplesmente quebrado.
	atual, err := uc.GetItemPurchaseCost(dona, itemCode)
	if err != nil {
		t.Fatalf("a dona perdeu acesso ao próprio custo: %v", err)
	}
	if !pertoDe(atual.UnitCost, 777) {
		t.Fatalf("custo da dona = %.4f, esperado 777 — a vizinha conseguiu alterar", atual.UnitCost)
	}
}

// TestRegraDeRateioIsolaEntreEmpresas: a regra é da empresa por coluna própria.
func TestRegraDeRateioIsolaEntreEmpresas(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := cost_uc.New(standardCostRepo.New(q, pool))
	dona := testutil.TenantContext(t, pool)
	vizinha := empresaNova(t, pool)
	uid := testutil.Actor(t, pool)

	criada, err := uc.CriarRegraDeRateio(dona, request.CostOverheadRuleDTO{
		Code: "ISOLA-1", Description: "Energia da usinagem", Base: "MAQUINA",
		Method: "VALOR_POR_HORA", Rate: 18,
		ValidFrom: time.Now().Format("2006-01-02"), CreatedBy: uid.String(),
	})
	if err != nil {
		t.Fatalf("criando regra: %v", err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, "DELETE FROM cost_overhead_rules WHERE id = $1", criada.ID) })

	daVizinha, err := uc.ListarRegrasDeRateio(vizinha)
	if err != nil {
		t.Fatalf("vizinha listando regras: %v", err)
	}
	for _, r := range daVizinha {
		if r.ID == criada.ID {
			t.Fatal("a empresa vizinha viu a regra de rateio da outra")
		}
	}
	if err := uc.DesativarRegraDeRateio(vizinha, criada.ID); err == nil {
		t.Fatal("a empresa vizinha desativou a regra de rateio da outra")
	}
	// Controle positivo.
	daDona, err := uc.ListarRegrasDeRateio(dona)
	if err != nil {
		t.Fatalf("dona listando regras: %v", err)
	}
	achou := false
	for _, r := range daDona {
		if r.ID == criada.ID {
			achou = true
		}
	}
	if !achou {
		t.Fatal("a dona não viu a própria regra de rateio")
	}
}

// TestRateioRecusaTaxaEmPercentualInteiro: 12 em vez de 0,12 é o erro de digitação
// mais provável, e multiplicaria o custo por 13.
func TestRateioRecusaTaxaEmPercentualInteiro(t *testing.T) {
	q, pool := testutil.Queries(t)
	uc := cost_uc.New(standardCostRepo.New(q, pool))
	ctx := testutil.TenantContext(t, pool)
	uid := uuid.New()

	_, err := uc.CriarRegraDeRateio(ctx, request.CostOverheadRuleDTO{
		Code: "RUIM", Description: "Percentual inteiro", Base: "MATERIAL",
		Method: "PERCENTUAL", Rate: 12,
		ValidFrom: time.Now().Format("2006-01-02"), CreatedBy: uid.String(),
	})
	if err == nil {
		t.Fatal("taxa de 12 (1200%) foi aceita como percentual")
	}
	// A mensagem precisa dizer o que digitar, não só recusar.
	if !contemTexto(err.Error(), "0.12") && !contemTexto(err.Error(), "0,12") {
		t.Fatalf("a mensagem não sugere a fração correta: %v", err)
	}

	// Valor por hora sobre base que não é medida em horas: não faz sentido.
	if _, err := uc.CriarRegraDeRateio(ctx, request.CostOverheadRuleDTO{
		Code: "RUIM2", Description: "Hora sobre material", Base: "MATERIAL",
		Method: "VALOR_POR_HORA", Rate: 10,
		ValidFrom: time.Now().Format("2006-01-02"), CreatedBy: uid.String(),
	}); err == nil {
		t.Fatal("valor por hora sobre base MATERIAL foi aceito")
	}
}

func contemTexto(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
