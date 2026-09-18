package entity

import (
	"sort"

	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
)

// EffectiveScrapPct devolve o refugo que vale para esta etapa: o da etapa
// quando informado, senão o da operação de biblioteca.
func (ro *RouteOperation) EffectiveScrapPct(defaultScrap float64) float64 {
	if ro.ScrapPct != nil {
		return *ro.ScrapPct
	}
	return defaultScrap
}

// QuantidadePorOperacao responde a pergunta que o planejador faz todo dia:
// para entregar `boas` peças no fim do roteiro, quantas precisam ENTRAR em
// cada operação?
//
// A conta vai de trás para frente. A última operação precisa receber mais do
// que entrega, porque parte do que entra ali refuga; a anterior precisa
// receber mais ainda, e assim por diante até a primeira. A quantidade a soltar
// é a que entra na primeira operação.
//
//	entra = sai / (1 − refugo/100)
//
// É a MESMA fórmula da perda de estrutura (structentity.QuantidadeComPerda,
// fórmula "divide"). Duas contas de perda no mesmo sistema já produziram uma
// divergência silenciosa entre o que o MRP comprava e o que a ordem consumia;
// aqui existe uma só.
//
// Numa rede que converge (duas operações alimentam uma terceira), o que sai de
// uma operação é o maior do que seus sucessores precisam — todos precisam ser
// alimentados, e a peça é física: não dá para mandar metade para cada lado.
// Numa rede linear isso se reduz ao encadeamento simples 10 → 20 → 30.
//
// Operações presas num ciclo de precedência ficam de fora do mapa; quem chama
// já recusa planejar uma rede com ciclo (ver CriticalPath).
func QuantidadePorOperacao(ops []*RouteOperation, edges []*NetworkEdge, boas float64) map[int64]float64 {
	entra := make(map[int64]float64, len(ops))
	if len(ops) == 0 {
		return entra
	}
	if boas <= 0 {
		boas = 1
	}
	if len(edges) == 0 && len(ops) > 1 {
		edges = linearEdgesBySequence(ops)
	}

	successores := make(map[int64][]int64)
	grauDeSaida := make(map[int64]int, len(ops))
	for _, e := range edges {
		successores[e.PredecessorID] = append(successores[e.PredecessorID], e.SuccessorID)
	}
	for _, op := range ops {
		grauDeSaida[op.ID] = len(successores[op.ID])
	}

	// Kahn ao contrário: começa pelas operações que não alimentam ninguém (o
	// fim do roteiro) e caminha para trás.
	predecessores := make(map[int64][]int64)
	for _, e := range edges {
		predecessores[e.SuccessorID] = append(predecessores[e.SuccessorID], e.PredecessorID)
	}

	fila := make([]int64, 0, len(ops))
	for _, op := range ops {
		if grauDeSaida[op.ID] == 0 {
			fila = append(fila, op.ID)
		}
	}

	opPorID := make(map[int64]*RouteOperation, len(ops))
	for _, op := range ops {
		opPorID[op.ID] = op
	}

	sai := make(map[int64]float64, len(ops))
	for _, id := range fila {
		sai[id] = boas // operação final: o que sai dela é o que o roteiro entrega
	}

	for len(fila) > 0 {
		atual := fila[0]
		fila = fila[1:]

		op := opPorID[atual]
		if op == nil {
			continue
		}
		entra[atual] = structentity.QuantidadeComPerda(
			sai[atual], op.EffectiveScrap, structentity.FormulaPerdaDivide)

		for _, predID := range predecessores[atual] {
			// O predecessor precisa entregar o suficiente para o sucessor mais
			// exigente.
			if entra[atual] > sai[predID] {
				sai[predID] = entra[atual]
			}
			grauDeSaida[predID]--
			if grauDeSaida[predID] == 0 {
				fila = append(fila, predID)
			}
		}
	}
	return entra
}

// QuantidadeASoltar é quanto precisa entrar na PRIMEIRA operação para o roteiro
// entregar `boas` peças. É a quantidade que a ordem de produção libera.
func QuantidadeASoltar(ops []*RouteOperation, edges []*NetworkEdge, boas float64) float64 {
	porOp := QuantidadePorOperacao(ops, edges, boas)
	if len(porOp) == 0 {
		return boas
	}
	// A primeira operação é a de menor sequência entre as que têm quantidade
	// calculada; ela é a que recebe o maior número, porque acumula o refugo de
	// todas as seguintes.
	ordenadas := make([]*RouteOperation, 0, len(ops))
	for _, op := range ops {
		if _, ok := porOp[op.ID]; ok {
			ordenadas = append(ordenadas, op)
		}
	}
	if len(ordenadas) == 0 {
		return boas
	}
	sort.SliceStable(ordenadas, func(i, j int) bool { return ordenadas[i].Sequence < ordenadas[j].Sequence })
	return porOp[ordenadas[0].ID]
}
