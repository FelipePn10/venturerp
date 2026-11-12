package production_order_uc

import (
	"context"
	"fmt"

	mrpentity "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
)

// LeitorDaFormulaDePerda lê os parâmetros de planejamento; o parâmetro 20 é a
// fórmula de perdas da estrutura. O MRP já planejava por ele, mas a OF, o
// apontamento e o encerramento fixavam a fórmula padrão: com o parâmetro em 1
// (multiplica), o MRP comprava por uma conta e a fábrica consumia por outra.
type LeitorDaFormulaDePerda interface {
	LoadTypedPlanningParams(ctx context.Context) (*mrpentity.TypedPlanningParams, error)
}

// formulaDePerda devolve a fórmula configurada; sem leitor, a padrão do sistema.
func formulaDePerda(ctx context.Context, leitor LeitorDaFormulaDePerda) (int, error) {
	if leitor == nil {
		return structentity.FormulaPerdaPadrao, nil
	}
	p, err := leitor.LoadTypedPlanningParams(ctx)
	if err != nil {
		return 0, fmt.Errorf("lendo a fórmula de perdas da estrutura (parâmetro 20): %w", err)
	}
	if p == nil || p.FormulaPerdasEstrutura == 0 {
		return structentity.FormulaPerdaPadrao, nil
	}
	return p.FormulaPerdasEstrutura, nil
}
