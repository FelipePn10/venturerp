package production_order_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	mrpentity "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
)

type parametrosFixos struct {
	p   *mrpentity.TypedPlanningParams
	err error
}

func (f parametrosFixos) LoadTypedPlanningParams(context.Context) (*mrpentity.TypedPlanningParams, error) {
	return f.p, f.err
}

// A OF segue a fórmula de perdas do parâmetro 20, como o MRP: com 5% de perda
// sobre 100, a fórmula 1 pede 105 e a 2 (padrão) pede 105,2632.
func TestOFUsaAFormulaDePerdaDoMRP(t *testing.T) {
	ctx := context.Background()
	multiplica := mrpentity.DefaultTypedPlanningParams()
	multiplica.FormulaPerdasEstrutura = structentity.FormulaPerdaMultiplica

	casos := []struct {
		nome     string
		leitor   LeitorDaFormulaDePerda
		esperado string
	}{
		{"sem leitor: padrão", nil, "105.2632"},
		{"parâmetro 1: multiplica", parametrosFixos{p: multiplica}, "105"},
		{"parâmetro vazio: padrão", parametrosFixos{p: &mrpentity.TypedPlanningParams{}}, "105.2632"},
	}
	for _, c := range casos {
		f, err := formulaDePerda(ctx, c.leitor)
		if err != nil {
			t.Fatalf("%s: %v", c.nome, err)
		}
		got := structentity.QuantidadeDoComponente(decimal.NewFromInt(100), decimal.NewFromInt(1), decimal.NewFromInt(5), f)
		if !got.Equal(decimal.RequireFromString(c.esperado)) {
			t.Errorf("%s: %s, esperado %s", c.nome, got, c.esperado)
		}
	}
	if _, err := formulaDePerda(ctx, parametrosFixos{err: errors.New("banco fora")}); err == nil {
		t.Fatal("falha ao ler o parâmetro não pode virar a fórmula padrão em silêncio")
	}
}
