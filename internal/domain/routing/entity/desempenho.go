package entity

import (
	"math"
	"time"

	"github.com/shopspring/decimal"
)

// RouteMaterial é um componente da estrutura do item do roteiro e a etapa em
// que ele entra (nil = no início da ordem).
type RouteMaterial struct {
	StructureID   int64
	ChildCode     int64
	ChildBusiness string
	ChildName     string
	Quantity      decimal.Decimal
	Unit          string
	StepSequence  *int16
	ParentMask    *string
	IsCoproduct   bool
}

// StepActuals é o que as OPs concluídas de uma etapa planejaram e apontaram.
type StepActuals struct {
	RouteOperationID int64
	Orders           int
	ActualHours      float64
	PlannedHours     float64 // tempo de produção planejado (sem preparação)
	SetupHours       float64
	Quantity         float64
	LastCompleted    *time.Time
}

// AmostraMinimaParaSugerir: abaixo disso a diferença é ruído de uma OP só.
const AmostraMinimaParaSugerir = 3

// ComparacaoDeTempo compara o padrão com o real de uma etapa.
type ComparacaoDeTempo struct {
	Orders            int
	PlannedHours      float64 // o que o padrão atual prevê para o volume apontado
	ActualHours       float64
	DeviationPct      float64 // (real − planejado) / planejado × 100
	ActualPerPiece    float64 // horas reais por peça
	PlannedPerPiece   float64
	SuggestedRunHours *float64 // novo tempo de máquina, em HORAS por ciclo
	SuggestionReason  string
	LastCompleted     *time.Time
}

// CompararTempo diz quanto a etapa realmente leva e, com amostra suficiente e
// desvio relevante (≥ 10%), sugere o tempo de máquina que reproduz o real.
//
// A comparação é com o que o padrão ATUAL (`t`) prevê para o mesmo volume —
// preparação uma vez por ordem mais o tempo de máquina pelos ciclos das peças
// apontadas — e não com o planejado gravado nas ordens antigas. Comparar com o
// gravado fazia a sugestão se repetir depois de aplicada (as ordens antigas não
// mudam), e cada "Aplicar" inflava o tempo de novo.
//
// A preparação fica de fora da sugestão: o apontamento não a separa da produção,
// então ela é descontada pelo valor padrão e o resto é atribuído à máquina.
func CompararTempo(a StepActuals, t OperationTime) ComparacaoDeTempo {
	c := ComparacaoDeTempo{Orders: a.Orders, ActualHours: a.ActualHours, LastCompleted: a.LastCompleted}
	base := t.RunBaseQty
	if base <= 0 {
		base = 1
	}
	ciclos := a.Quantity * t.unidadesPorPeca() / base // agregado: sem arredondar por ordem
	setupTotal := float64(a.Orders) * t.Setup
	padrao := setupTotal + t.Run*ciclos
	c.PlannedHours = padrao
	if padrao > 0 {
		c.DeviationPct = (a.ActualHours - padrao) / padrao * 100
	}
	if a.Quantity > 0 {
		c.ActualPerPiece = a.ActualHours / a.Quantity
		c.PlannedPerPiece = padrao / a.Quantity
	}
	switch {
	case a.Orders < AmostraMinimaParaSugerir:
		c.SuggestionReason = "poucas ordens concluídas para sugerir um novo tempo"
	case a.Quantity <= 0 || ciclos <= 0:
		c.SuggestionReason = "as ordens não registram a quantidade da etapa para comparar"
	case t.Run <= 0:
		c.SuggestionReason = "a etapa não tem tempo de máquina padrão para comparar"
	case math.Abs(c.DeviationPct) < 10:
		c.SuggestionReason = "o real está dentro de 10% do padrão"
	default:
		producaoReal := a.ActualHours - setupTotal
		if producaoReal <= 0 {
			c.SuggestionReason = "o apontado não passa da preparação padrão"
			break
		}
		novo := math.Round(producaoReal/ciclos*1e6) / 1e6
		c.SuggestedRunHours = &novo
		c.SuggestionReason = "o real se afasta mais de 10% do padrão nas últimas ordens"
	}
	return c
}
