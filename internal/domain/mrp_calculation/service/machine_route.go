package service

import (
	"context"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	routing "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/shopspring/decimal"
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

// machineRouteQtyReader escolhe o roteiro considerando a faixa de lote.
type machineRouteQtyReader interface {
	MachineRouteQty(context.Context, int64, string, decimal.Decimal) ([]*routing.RouteOperation, []*routing.NetworkEdge, error)
}

// Route-specific times take precedence. Otherwise the measured item/machine
// productivity supplies the operation's processing time. Every internal step
// needs an eligible machine; missing data must not produce a partial schedule.
func (s *MRPServiceImpl) machineSteps(ctx context.Context, sug *entity.PlannedOrderSuggestion, profiles []*entity.MachineTimeInfo) ([]machineStep, error) {
	var ops []*routing.RouteOperation
	var edges []*routing.NetworkEdge
	if reader, ok := s.MRPRepo.(machineRouteQtyReader); ok {
		// Com a quantidade da sugestão, a faixa de lote dos roteiros entra na
		// escolha — a capacidade reserva a máquina do roteiro que a OP vai usar.
		var err error
		ops, edges, err = reader.MachineRouteQty(ctx, sug.ItemCode, sug.Mask, sug.Quantity)
		if err != nil {
			return nil, err
		}
	} else if reader, ok := s.MRPRepo.(machineRouteReader); ok {
		var err error
		ops, edges, err = reader.MachineRoute(ctx, sug.ItemCode, sug.Mask)
		if err != nil {
			return nil, err
		}
	}
	// A quantidade é decimal no MRP; o tempo de máquina (minutos) é que é
	// calculado em float, na fronteira.
	qtdOrdem := sug.Quantity.InexactFloat64()
	if len(ops) == 0 {
		mt := selectMachineTime(profiles, sug.Mask)
		if mt == nil {
			return nil, nil
		}
		minutes, err := machineMinutes(mt, qtdOrdem)
		if err != nil {
			// A mensagem de machineMinutes cita o código interno do item.
			return nil, domainerr.NewValidation(fmt.Sprintf(
				"o %s não tem roteiro e a produtividade dele na máquina %d não tem tempo válido: informe o tempo de produção, a quantidade base e a unidade de tempo na produtividade",
				s.rotuloDoItem(ctx, sug.ItemCode), mt.MachineCode))
		}
		step := measuredMachineStep(mt, qtdOrdem, minutes)
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
	entraPorEtapa := routing.QuantidadePorOperacao(ops, edges, qtdOrdem)
	quantidadeDa := func(op *routing.RouteOperation) float64 {
		if q, ok := entraPorEtapa[op.ID]; ok && q > 0 {
			return q
		}
		return qtdOrdem
	}

	out := []machineStep{}
	delay := time.Duration(0)
	for _, op := range ordered {
		if op.Situation == routing.RouteOpGhost || op.Situation == routing.RouteOpInactive {
			continue
		}
		if op.OperationOrigin == routing.OriginExternal || op.OperationOrigin == routing.OriginThirdPart {
			// O prazo vale da etapa OU da operação de biblioteca (o mesmo
			// COALESCE do resto do sistema). Antes só a etapa contava: com o
			// prazo cadastrado na biblioteca, a capacidade recusava pedindo um
			// prazo que estava preenchido.
			if op.EffectiveSubcontractDays <= 0 {
				return nil, domainerr.NewValidation(fmt.Sprintf(
					"informe o prazo do terceiro na etapa %d (%s) do roteiro do item antes de planejar a capacidade",
					op.Sequence, op.OperationName))
			}
			delay += time.Duration(op.EffectiveSubcontractDays) * 24 * time.Hour
			continue
		}
		if op.EffectiveWorkCenterID == nil {
			// A sequência e o código de negócio são o que o usuário vê; o id da
			// etapa e a chave do item não dizem nada na tela.
			return nil, domainerr.NewValidation(fmt.Sprintf(
				"etapa %d (%s) do roteiro do %s sem centro de trabalho: escolha o centro na etapa ou defina o centro padrão da operação",
				op.Sequence, op.OperationName, s.rotuloDoItem(ctx, sug.ItemCode)))
		}
		candidates := []*entity.MachineTimeInfo{}
		for _, profile := range profiles {
			if profile.WorkCenterID == *op.EffectiveWorkCenterID {
				candidates = append(candidates, profile)
			}
		}
		// Regra única de tempo: o ROTEIRO manda (sobreposição da etapa ∘
		// operação de biblioteca). A produtividade da VMAQ0200 só dimensiona a
		// etapa quando o roteiro não tem tempo. Com tempo no roteiro, da
		// produtividade só valem a máquina (que diz onde a etapa roda) e o
		// rendimento; antes o tempo dela era conferido primeiro e, zerado,
		// recusava o plano com "produtividade inválida" — o usuário tinha de
		// digitar o tempo duas vezes, e o da produtividade nunca era usado.
		if op.EffTime.SemMedida {
			return nil, routing.ErroSemMedida(op)
		}
		peloRoteiro := op.EffTime.Run > 0
		mt := selectMachineTime(candidates, sug.Mask)
		if mt == nil {
			if peloRoteiro {
				return nil, domainerr.NewValidation(fmt.Sprintf(
					"cadastre o %s na produtividade de uma máquina de cada centro do roteiro (o tempo vem do roteiro) — falta no %s, etapa %d",
					s.rotuloDoItem(ctx, sug.ItemCode), nomeDoCentro(op), op.Sequence))
			}
			return nil, domainerr.NewValidation(fmt.Sprintf(
				"cadastre o %s na produtividade de uma máquina do %s com o tempo de produção: a etapa %d (%s) do roteiro não tem tempo",
				s.rotuloDoItem(ctx, sug.ItemCode), nomeDoCentro(op), op.Sequence, op.OperationName))
		}
		qtdeDaEtapa := quantidadeDa(op)
		var minutes float64
		if peloRoteiro {
			if err := eficienciaValida(mt); err != nil {
				return nil, domainerr.NewValidation(fmt.Sprintf("%s do %s na máquina %d", err.Error(), s.rotuloDoItem(ctx, sug.ItemCode), mt.MachineCode))
			}
			minutes = minutosDoRoteiro(mt, op.EffTime, qtdeDaEtapa)
		} else {
			var err error
			if minutes, err = machineMinutes(mt, qtdeDaEtapa); err != nil {
				return nil, domainerr.NewValidation(fmt.Sprintf(
					"a etapa %d (%s) do roteiro não tem tempo e a produtividade do %s na máquina %d não tem tempo válido: informe o tempo no roteiro ou na produtividade",
					op.Sequence, op.OperationName, s.rotuloDoItem(ctx, sug.ItemCode), mt.MachineCode))
			}
		}
		if minutes <= 0 {
			return nil, domainerr.NewValidation(fmt.Sprintf("tempo inválido na etapa %d (%s) do roteiro", op.Sequence, op.OperationName))
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
			return nil, domainerr.NewValidation("roteiro com ciclo ou precedência inválida")
		}
	}
	return out, nil
}

// rotuloDoItem é o código que o usuário conhece (o comercial, RN-01001), não a
// chave interna. Só é buscado nas mensagens de erro; sem ele, sai "item N".
func (s *MRPServiceImpl) rotuloDoItem(ctx context.Context, itemCode int64) string {
	if l, ok := s.MRPRepo.(itemLabeler); ok {
		if codigos, err := l.ItemBusinessCodes(ctx, []int64{itemCode}); err == nil && codigos[itemCode] != "" {
			return codigos[itemCode]
		}
	}
	return fmt.Sprintf("item %d", itemCode)
}

func nomeDoCentro(op *routing.RouteOperation) string {
	if op.WorkCenterName != "" {
		return fmt.Sprintf("centro %s", op.WorkCenterName)
	}
	return fmt.Sprintf("centro %d", *op.EffectiveWorkCenterID)
}

func measuredMachineStep(mt *entity.MachineTimeInfo, qty, minutes float64) machineStep {
	step := machineStep{profile: mt, minutes: minutes}
	if mt.TimeBasis != "PROPORTIONAL" && mt.ProductionBaseQty > 0 {
		step.cycles = divisoesParaCima(qty, float64(mt.ProductionBaseQty))
		step.setupMinutes = mt.SetupTime
		if step.cycles > 0 {
			step.cycleMinutes = (minutes - mt.SetupTime) / step.cycles
		}
	}
	return step
}
