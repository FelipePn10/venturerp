//go:build integration

package planning_policy_uc_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planning_policy_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	mrprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	politicapg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planning_policy"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func dias(v int) *int        { return &v }
func texto(v string) *string { return &v }

func TestPoliticaDePlanejamentoPorItemEClassificacao(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	code := testutil.UniqueCode() % 1_000_000_000
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, q := range []string{
			`DELETE FROM politicas_planejamento WHERE enterprise_id = ANY($1)`,
			`DELETE FROM item_classification_assignments WHERE enterprise_id = ANY($1)`,
			`DELETE FROM item_classifications WHERE mask_id IN (SELECT id FROM item_classification_masks WHERE enterprise_id = ANY($1))`,
			`DELETE FROM item_classification_masks WHERE enterprise_id = ANY($1)`,
			`DELETE FROM items WHERE enterprise_id = ANY($1)`,
			`DELETE FROM planning_params_semeados WHERE planning_param_id IN (SELECT id FROM planning_params WHERE enterprise_id = ANY($1))`,
			`DELETE FROM planning_params WHERE enterprise_id = ANY($1)`,
		} {
			_, _ = pool.Exec(bg, q, []int64{a.ID, b.ID})
		}
	})
	scalar := func(q string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := pool.QueryRow(bg, q, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		return v
	}
	testutil.Exec(t, pool, `SELECT semear_parametros_planejamento($1)`, a.ID)
	// Parâmetros da empresa: janela 30/0, horizonte 4, segurança 1.
	testutil.Exec(t, pool, `UPDATE planning_params SET value='4' WHERE enterprise_id=$1 AND param_number=48`, a.ID)
	testutil.Exec(t, pool, `UPDATE planning_params SET value='1' WHERE enterprise_id=$1 AND param_number=49`, a.ID)

	itemA, itemB, itemC := code, code+1, code+2
	for _, it := range []int64{itemA, itemB, itemC} {
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,health,created_by,enterprise_id) VALUES($1::bigint,'PP-'||$1::bigint::text,0,'ATIVO',$2,$3)`, it, actor, a.ID)
	}
	// Árvore 10 → 10.100; itemA na folha, itemB na raiz, itemC sem classificação.
	mask := scalar(`INSERT INTO item_classification_masks(code,mask,description,enterprise_id) VALUES($1,$2,'Política',$3) RETURNING id`, code, "PP"+strconv.FormatInt(code, 10), a.ID)
	raiz := scalar(`INSERT INTO item_classifications(code,mask_id,level,description) VALUES('10',$1,1,'Família') RETURNING id`, mask)
	folha := scalar(`INSERT INTO item_classifications(code,mask_id,parent_id,level,description) VALUES('10.100',$1,$2,2,'Subfamília') RETURNING id`, mask, raiz)
	testutil.Exec(t, pool, `INSERT INTO item_classification_assignments(enterprise_id,item_code,classification_id) VALUES($1,$2,$3),($1,$4,$5)`, a.ID, itemA, folha, itemB, raiz)
	maskB := scalar(`INSERT INTO item_classification_masks(code,mask,description,enterprise_id) VALUES($1,$2,'Outra',$3) RETURNING id`, code+1, "PQ"+strconv.FormatInt(code, 10), b.ID)
	classeB := scalar(`INSERT INTO item_classifications(code,mask_id,level,description) VALUES('20',$1,1,'De B') RETURNING id`, maskB)

	q := sqlc.New(pool)
	repo := politicapg.New(pool)
	uc := &planning_policy_uc.UseCase{Repo: repo, Params: mrprepo.NewMRPCalculationRepositorySQLC(q, pool), Auth: &auth.AuthService{}, Items: itemrepo.NewRepositoryItemSQLC(q)}
	txt := func(item int64) request.TextCode { return request.TextCode("PP-" + strconv.FormatInt(item, 10)) }
	sob := entity.EstrategiaSobEncomenda

	// Validações: escopo, campos, classificação de outra empresa.
	if _, err := uc.Criar(a.Ctx, request.PoliticaPlanejamentoDTO{ItemCode: txt(itemA)}); err == nil {
		t.Fatal("política sem nenhum campo")
	}
	if _, err := uc.Criar(a.Ctx, request.PoliticaPlanejamentoDTO{ClassificationID: &classeB, HorizonteFirmeDias: dias(3)}); err == nil || !strings.Contains(err.Error(), "classificação") {
		t.Fatalf("classificação de outra empresa: %v", err)
	}
	if _, err := uc.Criar(a.Ctx, request.PoliticaPlanejamentoDTO{ItemCode: txt(itemA), TempoSegurancaDias: dias(400)}); err == nil {
		t.Fatal("dias acima de 365")
	}

	// Raiz: sob encomenda e horizonte 10. Folha: horizonte 20. Item A: janela
	// só para a frente e estratégia de consumo.
	pr, err := uc.Criar(a.Ctx, request.PoliticaPlanejamentoDTO{ClassificationID: &raiz, Estrategia: &sob, HorizonteFirmeDias: dias(10), Observacao: texto("família sob encomenda")})
	if err != nil || pr.Escopo != "CLASSIFICACAO" || pr.ClassificationCode != "10" {
		t.Fatalf("política da raiz: %+v %v", pr, err)
	}
	if _, err := uc.Criar(a.Ctx, request.PoliticaPlanejamentoDTO{ClassificationID: &folha, HorizonteFirmeDias: dias(20)}); err != nil {
		t.Fatal(err)
	}
	cons := entity.EstrategiaConsumo
	pa, err := uc.Criar(a.Ctx, request.PoliticaPlanejamentoDTO{ItemCode: txt(itemA), Estrategia: &cons, ConsumoDiasAntes: dias(0), ConsumoDiasDepois: dias(15)})
	if err != nil || pa.Escopo != "ITEM" || pa.ItemBusinessCode != string(txt(itemA)) {
		t.Fatalf("política do item: %+v %v", pa, err)
	}
	if _, err := uc.Criar(a.Ctx, request.PoliticaPlanejamentoDTO{ItemCode: txt(itemA), TempoSegurancaDias: dias(2)}); err == nil || !strings.Contains(err.Error(), "já tem política") {
		t.Fatalf("uma política por item: %v", err)
	}

	// Efetiva campo a campo.
	ef, err := uc.Efetiva(a.Ctx, txt(itemA))
	if err != nil || ef.Estrategia != cons || ef.Origem.Estrategia != "do item" || ef.ConsumoDiasDepois != 15 ||
		ef.HorizonteFirmeDias != 20 || ef.Origem.HorizonteFirme != "da classificação 10.100" || ef.TempoSegurancaDias != 1 || ef.Origem.TempoSeguranca != "parâmetro da empresa" {
		t.Fatalf("efetiva do item A: %+v %v", ef, err)
	}
	if ef, err = uc.Efetiva(a.Ctx, txt(itemB)); err != nil || ef.Estrategia != sob || ef.HorizonteFirmeDias != 10 || ef.ConsumoDiasAntes != 30 {
		t.Fatalf("efetiva do item B (raiz): %+v %v", ef, err)
	}
	if ef, err = uc.Efetiva(a.Ctx, txt(itemC)); err != nil || ef.Estrategia != cons || ef.HorizonteFirmeDias != 4 || ef.Origem.HorizonteFirme != "parâmetro da empresa" {
		t.Fatalf("efetiva do item C (empresa): %+v %v", ef, err)
	}
	// O mapa do MRP traz A e B, e não C.
	params, err := mrprepo.NewMRPCalculationRepositorySQLC(q, pool).LoadTypedPlanningParams(a.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	mapa, err := repo.PoliticasDePlanejamento(a.Ctx, params.PoliticaDaEmpresa())
	if err != nil || len(mapa) != 2 || !mapa[itemB].SobEncomenda() || mapa[itemA].SobEncomenda() {
		t.Fatalf("mapa do MRP: %+v %v", mapa, err)
	}

	// Alterar: limpa a estratégia (volta a herdar a da raiz).
	alt, err := uc.Alterar(a.Ctx, pa.ID, request.PoliticaPlanejamentoDTO{ConsumoDiasAntes: dias(5)})
	if err != nil || alt.Estrategia != nil || alt.ConsumoDiasAntes == nil || *alt.ConsumoDiasAntes != 5 || alt.Escopo != "ITEM" {
		t.Fatalf("alterar: %+v %v", alt, err)
	}
	if ef, _ = uc.Efetiva(a.Ctx, txt(itemA)); ef.Estrategia != sob || ef.Origem.Estrategia != "da classificação 10" {
		t.Fatalf("estratégia volta a herdar: %+v", ef)
	}

	// Lista com filtro e isolamento.
	if l, err := uc.Listar(a.Ctx, "CLASSIFICACAO", ""); err != nil || len(l) != 2 {
		t.Fatalf("lista por escopo: %d %v", len(l), err)
	}
	if l, err := uc.Listar(a.Ctx, "", "subfam"); err != nil || len(l) != 1 {
		t.Fatalf("busca: %d %v", len(l), err)
	}
	if l, err := uc.Listar(b.Ctx, "", ""); err != nil || len(l) != 0 {
		t.Fatalf("a outra empresa não vê: %d %v", len(l), err)
	}
	if _, err := uc.Obter(b.Ctx, pa.ID); err == nil {
		t.Fatal("a outra empresa não abre")
	}
	if err := uc.Excluir(b.Ctx, pa.ID); err == nil {
		t.Fatal("a outra empresa não exclui")
	}
	if err := uc.Excluir(a.Ctx, pa.ID); err != nil {
		t.Fatal(err)
	}
	if ef, _ = uc.Efetiva(a.Ctx, txt(itemA)); ef.ConsumoDiasAntes != 30 || ef.HorizonteFirmeDias != 20 {
		t.Fatalf("sem a do item, herda tudo: %+v", ef)
	}
}
