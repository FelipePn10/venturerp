package entity

// Fórmulas de perda da estrutura. O parâmetro de planejamento
// FormulaPerdasEstrutura escolhe qual vale; o padrão do sistema é a 2.
const (
	FormulaPerdaMultiplica = 1 // base × (1 + p/100)
	FormulaPerdaDivide     = 2 // base ÷ (1 − p/100)  ← padrão
	FormulaPerdaIgnora     = 3 // base
	FormulaPerdaPadrao     = FormulaPerdaDivide
)

// QuantidadeComPerda aplica a fórmula de perdas sobre a quantidade bruta.
//
// Fonte única da conta. Antes cada rotina tinha a sua: o MRP respeitava o
// parâmetro (padrão 2, divisão) e a criação da ordem, o encerramento, o
// apontamento e o custo fixavam a fórmula 1 (multiplicação) no código. Para uma
// perda de 5% sobre 2250, o MRP comprava 2368,42 e a ordem consumia 2362,50 —
// a fábrica ficava permanentemente com uma sobra que ninguém explicava, e o
// custo do produto saía calculado sobre uma terceira quantidade.
//
// A divisão é a leitura correta de "perde-se p% do que foi requisitado": para
// obter `base` peças boas é preciso requisitar base/(1−p).
func QuantidadeComPerda(base, perda float64, formula int) float64 {
	if perda <= 0 {
		return base
	}
	switch formula {
	case FormulaPerdaMultiplica:
		return base * (1 + perda/100)
	case FormulaPerdaIgnora:
		return base
	default: // FormulaPerdaDivide
		if denominador := 1 - perda/100; denominador > 0 {
			return base / denominador
		}
		return base
	}
}
