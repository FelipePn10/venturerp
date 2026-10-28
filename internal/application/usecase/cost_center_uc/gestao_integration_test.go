//go:build integration

package cost_center_uc_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/cost_center_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_requisition_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/cost_center"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	purchaseReqRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_requisition"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// Centro de custo único: gestão (código da empresa, aprovador), requisição
// aprovada pelo centro antes de virar pedido, orçamento × realizado na árvore,
// conta que exige centro e exclusão pela rota antiga barrada pelo uso.
func TestCentroDeCustoUnicoComGestao(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	ctx := testutil.TenantContext(t, pool)
	empresa := testutil.EnterpriseID(t, ctx)
	eu := testutil.Actor(t, pool)
	u := ctx.Value(contextkey.UserKey).(*security.AuthUser)
	az := &auth.AuthService{}

	// Aprovador: outro usuário da empresa.
	aprovador := uuid.New()
	sufixo := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO users (id, name, email, password, role) VALUES ($1, 'Aprovador CC', $2, 'x', 'USER')`, aprovador, fmt.Sprintf("aprov-cc-%d@teste.local", sufixo))
	testutil.Exec(t, pool, `INSERT INTO user_enterprises (user_id, enterprise_id, role) VALUES ($1, $2, 'USER')`, aprovador, empresa)
	ctxAprovador := context.WithValue(bg, contextkey.UserKey, &security.AuthUser{ID: aprovador.String(), Role: "USER", EnterpriseID: u.EnterpriseID, EnterpriseCode: u.EnterpriseCode})

	var pai, filho int32
	if err := pool.QueryRow(bg, `INSERT INTO cost_centers (code, description, type, start_date, created_by, enterprise_id)
		VALUES ((SELECT COALESCE(MAX(code),0)+1 FROM cost_centers), 'FABRICA TESTE', 'PRODUCTIVE', CURRENT_DATE - 30, $1, $2) RETURNING code`, eu, empresa).Scan(&pai); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO cost_centers (code, description, parent_code, type, start_date, created_by, enterprise_id)
		VALUES ((SELECT COALESCE(MAX(code),0)+1 FROM cost_centers), 'CORTE TESTE', $3, 'PRODUCTIVE', CURRENT_DATE - 30, $1, $2) RETURNING code`, eu, empresa, pai).Scan(&filho); err != nil {
		t.Fatal(err)
	}
	var filhoID, contaID int64
	_ = pool.QueryRow(bg, `SELECT id FROM cost_centers WHERE code = $1`, filho).Scan(&filhoID)
	if err := pool.QueryRow(bg, `INSERT INTO plano_contas (codigo, descricao, tipo, natureza, nivel, is_active, enterprise_id, exige_centro_custo)
		VALUES ($1, 'DESPESA COM CENTRO', 'ANALITICA', 'DEVEDORA', 3, TRUE, $2, TRUE) RETURNING id`, fmt.Sprintf("9.9.%d", sufixo), empresa).Scan(&contaID); err != nil {
		t.Fatal(err)
	}
	item := 9_300_000_000 + sufixo%100_000_000
	testutil.SeedItem(t, pool, ctx, item, eu)
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM contas_pagar_rateios WHERE centro_custo_id = $1`, filhoID)
		_, _ = pool.Exec(bg, `DELETE FROM contas_pagar WHERE numero_documento LIKE 'CC-TESTE-%'`)
		_, _ = pool.Exec(bg, `DELETE FROM requisicao_aprovacoes_centro WHERE cost_center_id = $1`, filhoID)
		_, _ = pool.Exec(bg, `DELETE FROM purchase_requisition_items WHERE item_code = $1`, item)
		_, _ = pool.Exec(bg, `DELETE FROM cost_centers WHERE code IN ($1, $2)`, filho, pai)
		_, _ = pool.Exec(bg, `DELETE FROM plano_contas WHERE id = $1`, contaID)
		_, _ = pool.Exec(bg, `DELETE FROM user_enterprises WHERE user_id = $1`, aprovador)
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id = $1`, aprovador)
	})

	repo := cost_center.NewGestaoPG(pool)
	uc := &cost_center_uc.GestaoUseCase{Repo: repo, Auth: az}

	// 1. Gestão: o código da empresa e o aprovador.
	if _, err := uc.SalvarGestao(ctx, filho, cost_center_uc.GestaoDTO{Codigo: "CC-T-CORTE", AprovaRequisicoes: true}); err == nil {
		t.Fatal("aprovar requisições exige aprovador")
	}
	c, err := uc.SalvarGestao(ctx, filho, cost_center_uc.GestaoDTO{Codigo: fmt.Sprintf("CC-T-CORTE-%d", sufixo), ResponsavelID: &eu, AprovadorID: &aprovador, AprovaRequisicoes: true, ContaPadraoID: &contaID})
	if err != nil {
		t.Fatal(err)
	}
	if c.Aprovador != "Aprovador CC" || !c.AprovaRequisicoes {
		t.Fatalf("gestão gravada: %+v", c)
	}
	if _, err := uc.SalvarGestao(ctx, pai, cost_center_uc.GestaoDTO{Codigo: c.Codigo}); err == nil {
		t.Fatal("código repetido na empresa deveria ser recusado")
	}

	// 2. Requisição com item do centro: espera o aprovador antes de virar pedido.
	reqs := purchaseReqRepo.New(sqlc.New(pool), pool)
	rUC := purchase_requisition_uc.NewPurchaseRequisitionUseCase(reqs, az).ComAprovacoesDoCentro(repo)
	cc := int64(filho)
	req, err := rUC.Create(ctx, request.CreatePurchaseRequisitionDTO{Items: []request.RequisitionItemInput{{ItemCode: item, Quantity: 5, CostCenterCode: &cc}}})
	if err != nil {
		t.Fatal(err)
	}
	pend, err := uc.Aprovacoes(ctxAprovador, "PENDENTE", true, &req.Code)
	if err != nil || len(pend) != 1 || !pend[0].PodeDecidir {
		t.Fatalf("aprovação pendente para o aprovador: %v %+v", err, pend)
	}
	gerar := &purchase_requisition_uc.GeneratePurchaseOrdersUseCase{Reqs: reqs, Auth: az, Aprovacoes: repo}
	sup := int64(1)
	res, err := gerar.Execute(ctx, request.GeneratePurchaseOrdersDTO{Selections: []request.GenerationSelection{{RequisitionItemID: req.Items[0].ID, QtyToAttend: 5, SupplierCode: &sup}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) != 1 || !strings.Contains(res.Skipped[0], "aguardando o aprovador do centro") {
		t.Fatalf("a requisição pendente não vira pedido: %+v", res.Skipped)
	}
	if err := uc.Decidir(ctx, pend[0].ID, true, ""); err == nil {
		t.Fatal("só o aprovador do centro decide")
	}
	if err := uc.Decidir(ctxAprovador, pend[0].ID, false, ""); err == nil {
		t.Fatal("reprovar exige motivo")
	}
	if err := uc.Decidir(ctxAprovador, pend[0].ID, true, ""); err != nil {
		t.Fatal(err)
	}
	if b, _ := repo.RequisicoesBloqueadas(ctx, []int64{req.Code}); len(b) != 0 {
		t.Fatalf("aprovada não bloqueia: %v", b)
	}
	// Reprovada: bloqueia; corrigida com item novo, volta para o aprovador.
	req2, err := rUC.Create(ctx, request.CreatePurchaseRequisitionDTO{Items: []request.RequisitionItemInput{{ItemCode: item, Quantity: 9, CostCenterCode: &cc}}})
	if err != nil {
		t.Fatal(err)
	}
	p2, _ := uc.Aprovacoes(ctxAprovador, "PENDENTE", true, &req2.Code)
	if len(p2) != 1 {
		t.Fatalf("segunda requisição pendente: %+v", p2)
	}
	if err := uc.Decidir(ctxAprovador, p2[0].ID, false, "quantidade acima do necessário"); err != nil {
		t.Fatal(err)
	}
	if b, _ := repo.RequisicoesBloqueadas(ctx, []int64{req2.Code}); !strings.Contains(b[req2.Code], "reprovada") {
		t.Fatalf("reprovada bloqueia: %v", b)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := rUC.AddItem(ctx, request.AddRequisitionItemDTO{RequisitionCode: req2.Code, RequisitionItemInput: request.RequisitionItemInput{ItemCode: item, Quantity: 3, CostCenterCode: &cc}}); err != nil {
		t.Fatal(err)
	}
	if p3, _ := uc.Aprovacoes(ctxAprovador, "PENDENTE", true, &req2.Code); len(p3) != 1 {
		t.Fatalf("corrigida volta a pendente: %+v", p3)
	}

	// 3. Contas a pagar: a conta exige centro; com o centro, entra no realizado.
	fin := financial.NewFinancialRepositoryPG(pool)
	ap := &financial_uc.CreateContaPagarUseCase{Repo: fin, Auth: az}
	hoje := time.Now().Format(time.DateOnly)
	base := request.CreateContaPagarDTO{TipoDocumento: "NF", DataEmissao: hoje, DataVencimento: hoje, ValorBruto: 1500, ParcelaNumero: 1, ParcelaTotal: 1}
	semCentro := base
	semCentro.NumeroDocumento = fmt.Sprintf("CC-TESTE-%d-A", sufixo)
	semCentro.Rateios = []request.RateioContaPagarDTO{{PlanoContasID: contaID, Valor: decimal.NewFromInt(1500)}}
	if _, err := ap.Execute(ctx, semCentro); err == nil || !strings.Contains(err.Error(), "exige centro de custo") {
		t.Fatalf("conta que exige centro sem centro deveria ser recusada: %v", err)
	}
	comCentro := base
	comCentro.NumeroDocumento = fmt.Sprintf("CC-TESTE-%d-B", sufixo)
	comCentro.Rateios = []request.RateioContaPagarDTO{{PlanoContasID: contaID, CentroCustoID: &filhoID, Valor: decimal.NewFromInt(1500)}}
	if _, err := ap.Execute(ctx, comCentro); err != nil {
		t.Fatal(err)
	}

	// 4. Orçamento do mês e a árvore consolidada: o filho estourou, o pai soma.
	agora := time.Now()
	if err := uc.SalvarOrcamento(ctx, cost_center_uc.OrcamentoDTO{Ano: agora.Year(), Linhas: []cost_center_uc.OrcamentoLinhaDTO{{CostCenterID: filhoID, Mes: int(agora.Month()), Valor: "1000,00"}}}); err == nil {
		t.Fatal("valor com vírgula vai em texto decimal com ponto")
	}
	if err := uc.SalvarOrcamento(ctx, cost_center_uc.OrcamentoDTO{Ano: agora.Year(), Linhas: []cost_center_uc.OrcamentoLinhaDTO{{CostCenterID: filhoID, Mes: int(agora.Month()), Valor: "1000.00"}}}); err != nil {
		t.Fatal(err)
	}
	arvore, err := uc.Arvore(ctx, agora.Year(), int(agora.Month()))
	if err != nil {
		t.Fatal(err)
	}
	var achou bool
	for _, n := range arvore {
		if n.Code != pai {
			continue
		}
		achou = true
		if len(n.Filhos) != 1 || !n.Filhos[0].Estourado || !n.RealizadoConsolidado.Equal(decimal.NewFromInt(1500)) || !n.OrcadoConsolidado.Equal(decimal.NewFromInt(1000)) {
			t.Fatalf("árvore consolidada: %+v", n)
		}
	}
	if !achou {
		t.Fatal("o centro-pai deveria ser raiz da árvore")
	}

	// 5. Excluir pela rota antiga (VFIN0130) confere o uso dos dois lados.
	usos, err := fin.UsosCentroCusto(ctx, filhoID)
	if err != nil {
		t.Fatal(err)
	}
	if len(usos) == 0 {
		t.Fatal("o centro usado no rateio e na requisição tem uso")
	}
}
