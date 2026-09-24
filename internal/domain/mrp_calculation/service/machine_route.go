package service

import (
	"context"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	routing "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"math"
	"sort"
	"time"
)

type machineStep struct {
	profile *entity.MachineTimeInfo
	// alternativas são as máquinas EQUIVALENTES a `profile` — mesma máscara e
	// mesma prioridade. Três serras iguais no chão de fábrica não são uma
	// preferida e duas reservas: são três recursos que fazem o mesmo, e quem
	// estiver livre antes deve pegar a ordem. Prioridade diferente continua
	// mandando, então "prefira a serra 1" segue valendo.
	alternativas                       []*entity.MachineTimeInfo
	minutes                            float64
	cycles, cycleMinutes, setupMinutes float64
	operationID                        *int64
	before, after                      time.Duration
}
type machineRouteReader interface {
	MachineRoute(context.Context, int64, string) ([]*routing.RouteOperation, []*routing.NetworkEdge, error)
}

// Route-specific times take precedence. Otherwise the measured item/machine
// productivity supplies the operation's processing time. Every internal step
// needs an eligible machine; missing data must not produce a partial schedule.
func (s *MRPServiceImpl) machineSteps(ctx context.Context, sug *entity.PlannedOrderSuggestion, profiles []*entity.MachineTimeInfo) ([]machineStep, error) {
	var ops []*routing.RouteOperation
	var edges []*routing.NetworkEdge
	if reader, ok := s.MRPRepo.(machineRouteReader); ok {
		var err error
		ops, edges, err = reader.MachineRoute(ctx, sug.ItemCode, sug.Mask)
		if err != nil {
			return nil, err
		}
	}
	if len(ops) == 0 {
		mt := selectMachineTime(profiles, sug.Mask)
		if mt == nil {
			return nil, nil
		}
		minutes, err := machineMinutes(mt, sug.Quantity)
		step := measuredMachineStep(mt, sug.Quantity, minutes)
		step.alternativas = equivalentes(profiles, mt)
		return []machineStep{step}, err
	}
	ordered, err := orderMachineRoute(ops, edges)
	if err != nil {
		return nil, err
	}

	// Com refugo, cada etapa processa uma quantidade diferente: para entregar a
	// quantidade da ordem, a primeira operação precisa rodar mais peças que a
	// última. Reservar a capacidade pela quantidade do pedido subestimaria
	// justamente o começo do roteiro — e a máquina apareceria livre num horário
	// em que estará ocupada. É a mesma conta que o roteiro mostra na tela.
	entraPorEtapa := routing.QuantidadePorOperacao(ops, edges, sug.Quantity)
	quantidadeDa := func(op *routing.RouteOperation) float64 {
		if q, ok := entraPorEtapa[op.ID]; ok && q > 0 {
			return q
		}
		return sug.Quantity
	}

	out := []machineStep{}
	delay := time.Duration(0)
	for _, op := range ordered {
		if op.Situation == routing.RouteOpGhost || op.Situation == routing.RouteOpInactive {
			continue
		}
		if op.OperationOrigin == routing.OriginExternal || op.OperationOrigin == routing.OriginThirdPart {
			if op.LeadTimeDays == nil || *op.LeadTimeDays <= 0 {
				return nil, fmt.Errorf("informe o prazo da operação externa %d antes de planejar a capacidade", op.ID)
			}
			delay += time.Duration(*op.LeadTimeDays) * 24 * time.Hour
			continue
		}
		if op.EffectiveWorkCenterID == nil {
			return nil, fmt.Errorf("operação %d do item %d sem centro de trabalho", op.ID, sug.ItemCode)
		}
		candidates := []*entity.MachineTimeInfo{}
		for _, profile := range profiles {
			if profile.WorkCenterID == *op.EffectiveWorkCenterID {
				candidates = append(candidates, profile)
			}
		}
		mt := selectMachineTime(candidates, sug.Mask)
		if mt == nil {
			return nil, fmt.Errorf("cadastre a produtividade do item %d em uma máquina do centro %d antes de planejar", sug.ItemCode, *op.EffectiveWorkCenterID)
		}
		qtdeDaEtapa := quantidadeDa(op)
		minutes, err := machineMinutes(mt, qtdeDaEtapa)
		if err != nil {
			return nil, err
		}
		peloRoteiro := op.RunTime != nil || op.StandardTime != nil
		if peloRoteiro {
			minutes = minutosDoRoteiro(mt, op.EffTime, qtdeDaEtapa)
		}
		if minutes <= 0 {
			return nil, fmt.Errorf("tempo inválido na operação %d", op.ID)
		}
		id := op.ID
		step := measuredMachineStep(mt, qtdeDaEtapa, minutes)
		step.alternativas = equivalentes(candidates, mt)
		if peloRoteiro {
			step.cycles = op.EffTime.Batches(qtdeDaEtapa)
			step.setupMinutes = op.EffTime.Setup * 60
			// O ciclo carrega o que sobra depois da preparação — inclusive a
			// parada de consumível, que acontece no meio da usinagem. Dividir o
			// total é o que mantém `ciclos × ciclo + preparação = total`, a
			// identidade de que AllocateMachineCycles depende para não cortar
			// uma peça no fim do turno.
			if step.cycles > 0 {
				step.cycleMinutes = (minutes - step.setupMinutes) / step.cycles
			}
			if step.cycleMinutes <= 0 {
				step.cycles = 0
			}
		}
		step.operationID = &id
		step.before = delay + time.Duration(op.EffTime.Queue*float64(time.Hour))
		step.after = time.Duration((op.EffTime.Wait + op.EffTime.Move) * float64(time.Hour))
		out = append(out, step)
		delay = 0
	}
	if len(out) > 0 {
		out[len(out)-1].after += delay
	}
	return out, nil
}

// Stable topological order preserves all dependencies. Parallel branches are
// conservatively sequenced; the finite scheduler never starts a successor early.
func orderMachineRoute(ops []*routing.RouteOperation, edges []*routing.NetworkEdge) ([]*routing.RouteOperation, error) {
	sorted := append([]*routing.RouteOperation(nil), ops...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Sequence < sorted[j].Sequence })
	if len(edges) == 0 {
		return sorted, nil
	}
	done := map[int64]bool{}
	out := make([]*routing.RouteOperation, 0, len(ops))
	for len(out) < len(ops) {
		progress := false
		for _, op := range sorted {
			if done[op.ID] {
				continue
			}
			ready := true
			for _, edge := range edges {
				if edge.SuccessorID == op.ID && !done[edge.PredecessorID] {
					ready = false
					break
				}
			}
			if ready {
				done[op.ID] = true
				out = append(out, op)
				progress = true
			}
		}
		if !progress {
			return nil, fmt.Errorf("roteiro com ciclo ou precedência inválida")
		}
	}
	return out, nil
}

func measuredMachineStep(mt *entity.MachineTimeInfo, qty, minutes float64) machineStep {
	step := machineStep{profile: mt, minutes: minutes}
	if mt.TimeBasis != "PROPORTIONAL" && mt.ProductionBaseQty > 0 {
		step.cycles = math.Ceil(qty / float64(mt.ProductionBaseQty))
		step.setupMinutes = mt.SetupTime
		if step.cycles > 0 {
			step.cycleMinutes = (minutes - mt.SetupTime) / step.cycles
		}
	}
	return step
}
