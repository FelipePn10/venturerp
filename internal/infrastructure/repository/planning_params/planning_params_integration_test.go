//go:build integration

package planning_params_test

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planning_params"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestPlanningParamsAreSeededPerEnterprise: empresa criada depois da 000221
// ficava sem parâmetros — tela vazia, nada editável. Agora cada empresa nasce
// com o catálogo, cujos valores são os padrões do código: o MRP calcula igual.
func TestPlanningParamsAreSeededPerEnterprise(t *testing.T) {
	q, pool := testutil.Queries(t)
	a, b := testutil.DuasEmpresas(t, pool, nil)
	repo := planning_params.NewPlanningParamRepositorySQLC(q)

	lista, err := repo.List(a.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 21 {
		t.Fatalf("empresa nova com %d parâmetro(s), esperado 21", len(lista))
	}

	mrp := mrp_calculation.NewMRPCalculationRepositorySQLC(q, pool)
	tipados, err := mrp.LoadTypedPlanningParams(a.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tipadosUsados(tipados), tipadosUsados(entity.DefaultTypedPlanningParams())) {
		t.Fatalf("o catálogo mudou o cálculo:\n semeado %+v\n padrão  %+v", tipados, entity.DefaultTypedPlanningParams())
	}

	// Alterar na A não mexe na B.
	if _, err := repo.Update(a.Ctx, 8, "9", testutil.Actor(t, pool)); err != nil {
		t.Fatalf("parâmetro da empresa não pôde ser alterado: %v", err)
	}
	p, err := repo.GetByNumber(b.Ctx, 8)
	if err != nil || p.Value != "5" {
		t.Fatalf("a alteração da A chegou na B: %+v (%v)", p, err)
	}
	if _, err := repo.Update(b.Ctx, 999, "S", uuid.New()); err == nil {
		t.Fatal("parâmetro inexistente foi alterado")
	}
}

// tipadosUsados zera os campos que o MRP não lê (posição padrão, proporção de
// entrega e cálculo do MPS): o catálogo traz texto neles e o código, vazio.
func tipadosUsados(p *entity.TypedPlanningParams) entity.TypedPlanningParams {
	c := *p
	c.DefaultPosicao, c.ProporcaoEntrega, c.UtilizaCalculoMPS = "", "", false
	return c
}
