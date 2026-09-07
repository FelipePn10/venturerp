package structure_uc

import (
	"context"
	"strings"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	strentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/formula"
)

// SimulateFormulaInput são a expressão e os valores das perguntas.
type SimulateFormulaInput struct {
	Formula   string             `json:"formula"`
	Rounding  string             `json:"quantity_rounding,omitempty"`
	Scale     int16              `json:"quantity_scale,omitempty"`
	Variables map[string]float64 `json:"variables,omitempty"`
	// LossPercentage e SetupLoss entram no resultado para a tela mostrar a
	// quantidade que a ordem vai realmente consumir, não só o valor bruto.
	LossPercentage float64 `json:"loss_percentage,omitempty"`
	SetupLoss      float64 `json:"setup_loss,omitempty"`
}

// SimulateFormulaOutput é o que a tela mostra antes de gravar.
type SimulateFormulaOutput struct {
	Valid            bool     `json:"valid"`
	Error            string   `json:"error,omitempty"`
	Variables        []string `json:"variables"`
	MissingVariables []string `json:"missing_variables,omitempty"`
	RawResult        float64  `json:"raw_result"`
	RoundedResult    float64  `json:"rounded_result"`
	QuantityWithLoss float64  `json:"quantity_with_loss"`
	QuantityPerOrder float64  `json:"quantity_per_order"`
	Explanation      string   `json:"explanation"`
}

// SimulateQuantityFormulaUseCase responde "quanto isso dá?" antes de salvar.
//
// Uma fórmula errada na estrutura só aparece depois, como necessidade errada no
// MRP — quando o erro já custou compra ou produção. Nem FoccoERP nem SAP
// oferecem essa conferência na própria tela de estrutura; aqui o usuário
// preenche as perguntas, vê o número e só então grava.
type SimulateQuantityFormulaUseCase struct{}

func (uc *SimulateQuantityFormulaUseCase) Execute(
	_ context.Context,
	in SimulateFormulaInput,
) (*SimulateFormulaOutput, error) {
	expr := strings.TrimSpace(in.Formula)
	if expr == "" {
		return nil, errorsuc.NewValidationError("informe a fórmula que deseja simular")
	}
	if err := formula.Validate(expr); err != nil {
		return &SimulateFormulaOutput{
			Valid:       false,
			Error:       err.Error(),
			Explanation: "A fórmula não pôde ser interpretada. Confira os parênteses e os operadores.",
		}, nil
	}

	usadas := formula.Variables(expr)
	faltando := []string{}
	for _, v := range usadas {
		if _, ok := in.Variables[v]; !ok {
			faltando = append(faltando, v)
		}
	}
	if len(faltando) > 0 {
		return &SimulateFormulaOutput{
			Valid:            false,
			Variables:        usadas,
			MissingVariables: faltando,
			Explanation:      "Responda " + strings.Join(faltando, ", ") + " para ver a quantidade resultante.",
		}, nil
	}

	bruto, err := formula.Evaluate(expr, in.Variables)
	if err != nil {
		return &SimulateFormulaOutput{
			Valid:       false,
			Error:       err.Error(),
			Variables:   usadas,
			Explanation: "A fórmula é válida, mas não pôde ser calculada com estes valores.",
		}, nil
	}

	// O arredondamento e a perda são aplicados como na explosão da estrutura,
	// para o número simulado ser o mesmo que o MRP vai usar.
	componente := &strentity.ItemStructure{
		QuantityFormula:  &expr,
		QuantityRounding: strentity.NormalizeRounding(in.Rounding),
		QuantityScale:    in.Scale,
		LossPercentage:   in.LossPercentage,
		SetupLoss:        in.SetupLoss,
	}
	arredondado, _ := componente.ResolvedQuantity(in.Variables)
	comPerda := componente.EffectiveQuantityWith(in.Variables)

	return &SimulateFormulaOutput{
		Valid:            true,
		Variables:        usadas,
		RawResult:        bruto,
		RoundedResult:    arredondado,
		QuantityWithLoss: comPerda,
		QuantityPerOrder: comPerda + in.SetupLoss,
		Explanation: "Resultado da fórmula, já com o arredondamento, a perda e a perda de preparação " +
			"— é a quantidade que a ordem vai consumir.",
	}, nil
}
