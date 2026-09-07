package entity

import (
	"math"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/structure/formula"
)

// Modos de arredondamento do resultado da fórmula de quantidade.
const (
	RoundNone    = "NONE"
	RoundUp      = "UP"
	RoundDown    = "DOWN"
	RoundNearest = "NEAREST"
)

// ValidRounding informa se o modo de arredondamento é conhecido.
func ValidRounding(mode string) bool {
	switch strings.ToUpper(strings.TrimSpace(mode)) {
	case RoundNone, RoundUp, RoundDown, RoundNearest, "":
		return true
	default:
		return false
	}
}

// NormalizeRounding devolve o modo em caixa alta, caindo em NONE quando vazio.
func NormalizeRounding(mode string) string {
	value := strings.ToUpper(strings.TrimSpace(mode))
	if value == "" {
		return RoundNone
	}
	return value
}

// HasQuantityFormula informa se o componente calcula a quantidade por fórmula.
func (s *ItemStructure) HasQuantityFormula() bool {
	return s.QuantityFormula != nil && strings.TrimSpace(*s.QuantityFormula) != ""
}

// applyRounding aplica o modo de arredondamento configurado na casa decimal
// pedida. Sem modo definido o valor sai como calculado.
func applyRounding(value float64, mode string, scale int16) float64 {
	if scale < 0 {
		scale = 0
	}
	factor := math.Pow(10, float64(scale))
	switch NormalizeRounding(mode) {
	case RoundUp:
		return math.Ceil(value*factor) / factor
	case RoundDown:
		return math.Floor(value*factor) / factor
	case RoundNearest:
		return math.Round(value*factor) / factor
	default:
		return value
	}
}

// ResolvedQuantity devolve a quantidade do componente por unidade do pai. Com
// fórmula cadastrada, ela é avaliada com as variáveis da configuração (as
// "perguntas" do configurador, por exemplo COMPRIMENTO e PROFUNDIDADE); sem
// fórmula, ou quando ela não puder ser avaliada, vale a quantidade fixa.
// O segundo retorno indica se a fórmula foi de fato aplicada.
func (s *ItemStructure) ResolvedQuantity(vars map[string]float64) (float64, bool) {
	if !s.HasQuantityFormula() {
		return s.Quantity, false
	}
	value, ok := formula.EvaluateSafe(*s.QuantityFormula, vars)
	if !ok || value < 0 {
		return s.Quantity, false
	}
	return applyRounding(value, s.QuantityRounding, s.QuantityScale), true
}

// EffectiveQuantityWith aplica a fórmula (quando houver) e, em cima do
// resultado, o percentual de perda — a quantidade que a OF precisa consumir.
func (s *ItemStructure) EffectiveQuantityWith(vars map[string]float64) float64 {
	quantity, _ := s.ResolvedQuantity(vars)
	return quantity * (1 + s.LossPercentage/100.0)
}

// FormulaVariables lista, em ordem de aparição, as variáveis usadas pela fórmula
// de quantidade. Serve para a tela apresentar o que precisa ser respondido.
func (s *ItemStructure) FormulaVariables() []string {
	if !s.HasQuantityFormula() {
		return nil
	}
	return formula.Variables(*s.QuantityFormula)
}

// Tipos de perda de custo aceitos.
const (
	CostLossPercent  = "PERCENTUAL"
	CostLossQuantity = "QUANTIDADE"
)

// NormalizeCostLossType devolve o tipo de perda de custo em maiúsculas, caindo
// em PERCENTUAL quando não informado — que é como a maioria das empresas
// cadastra a perda.
func NormalizeCostLossType(mode string) string {
	value := strings.ToUpper(strings.TrimSpace(mode))
	if value != CostLossQuantity {
		return CostLossPercent
	}
	return value
}

// CostQuantity devolve a quantidade que entra no cálculo do custo.
//
// A perda de custo é separada da perda de engenharia de propósito: a de
// engenharia dimensiona a necessidade de compra e produção, a de custo entra
// no valor do produto. Somar as duas é o que o FoccoERP faz na resposta 3 do
// parâmetro 20; aqui as duas ficam explícitas.
func (s *ItemStructure) CostQuantity(baseQuantity float64) float64 {
	if s.CostLoss <= 0 {
		return baseQuantity
	}
	if NormalizeCostLossType(s.CostLossType) == CostLossQuantity {
		return baseQuantity + s.CostLoss
	}
	return baseQuantity * (1 + s.CostLoss/100)
}
