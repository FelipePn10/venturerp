package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	machineentity "github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	machinesvc "github.com/FelipePn10/panossoerp/internal/domain/machine/service"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	mrprepo "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	routing "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"math"
	"sort"
	"strings"
	"time"
)

func selectMachineTime(list []*entity.MachineTimeInfo, mask string) *entity.MachineTimeInfo {
	var best *entity.MachineTimeInfo
	for _, mt := range list {
		if mt.Mask != mask && mt.Mask != "" {
			continue
		}
		if best == nil || (mt.Mask == mask && best.Mask != mask) || (mt.Mask == best.Mask && (mt.Priority < best.Priority || mt.Priority == best.Priority && mt.MachineID < best.MachineID)) {
			best = mt
		}
	}
	return best
}

// equivalentes devolve as máquinas que fazem o mesmo trabalho que `escolhida`:
// mesma máscara e mesma prioridade. É o que permite distribuir uma fila entre
// três máquinas iguais em vez de empilhar tudo na primeira.
func equivalentes(list []*entity.MachineTimeInfo, escolhida *entity.MachineTimeInfo) []*entity.MachineTimeInfo {
	out := []*entity.MachineTimeInfo{escolhida}
	for _, mt := range list {
		if mt.MachineID == escolhida.MachineID {
			continue
		}
		if mt.Mask == escolhida.Mask && mt.Priority == escolhida.Priority {
			out = append(out, mt)
		}
	}
	return out
}

// eficienciaDe é o rendimento aplicável do item naquela máquina: o valor do
// item (VMAQ0200 · Produtividade) sobrepõe o da máquina, e valor fora de (0,1]
// é tratado como 100 % — a mesma regra de CalculateProductionTime.
func eficienciaDe(mt *entity.MachineTimeInfo) float64 {
	eff := mt.MachineEfficiencyRate
	if mt.EfficiencyRate != nil {
		eff = *mt.EfficiencyRate
	}
	if eff <= 0 || eff > 1 {
		eff = 1
	}
	return eff
}

// minutosDeTroca é a parada para recarregar o consumível durante a usinagem.
// A primeira carga já está montada, então o número de PARADAS é o de cargas
// menos uma — idêntico a CalculateProductionTime.
func minutosDeTroca(mt *entity.MachineTimeInfo, usinagemMin float64) float64 {
	if mt.ConsumptionPerHour == nil || mt.ConsumableCapacity == nil || mt.ConsumableSwapMinutes == nil {
		return 0
	}
	if *mt.ConsumptionPerHour <= 0 || *mt.ConsumableCapacity <= 0 {
		return 0
	}
	gasto := usinagemMin / 60 * *mt.ConsumptionPerHour
	if cargas := divisoesParaCima(gasto, *mt.ConsumableCapacity); cargas > 1 {
		return (cargas - 1) * *mt.ConsumableSwapMinutes
	}
	return 0
}

// minutosDoRoteiro converte a ocupação prevista pelo ROTEIRO em minutos reais
// daquela máquina.
//
// Roteiro e produtividade por máquina não são a mesma informação em dois
// lugares: o roteiro diz quanto a operação leva no ritmo nominal; a VMAQ0200
// diz o que aquele equipamento entrega — quanto o item rende nele e quantas
// paradas para trocar consumível a ordem obriga. Até aqui, bastava a etapa ter
// tempo para o plano jogar fora os dois, e "Eficiência deste item = 85 %" não
// mudava um minuto sequer do planejamento. As camadas agora se somam: o tempo
// vem do roteiro, o rendimento e as paradas vêm da máquina.
func minutosDoRoteiro(mt *entity.MachineTimeInfo, t routing.OperationTime, qty float64) float64 {
	usinagem := t.Run * t.Batches(qty) * 60 / eficienciaDe(mt)
	return t.Setup*60 + usinagem + minutosDeTroca(mt, usinagem)
}

func machineMinutes(mt *entity.MachineTimeInfo, qty float64) (float64, error) {
	if mt.ProductionTime <= 0 || mt.ProductionBaseQty <= 0 || mt.SetupTime < 0 {
		return 0, domainerr.NewValidation(fmt.Sprintf("produtividade inválida do item %d na máquina %d", mt.ItemCode, mt.MachineCode))
	}
	switch mt.ProductionTimeUnit {
	case "SEGUNDO", "MINUTO", "HORA", "DIA":
	default:
		return 0, domainerr.NewValidation(fmt.Sprintf("unidade de tempo inválida do item %d", mt.ItemCode))
	}
	imt := &machineentity.ItemMachineTime{ProductionTime: mt.ProductionTime, ProductionTimeUnit: types.CapacityPeriod(mt.ProductionTimeUnit), ProductionBaseQty: mt.ProductionBaseQty, SetupTime: mt.SetupTime, EfficiencyRate: mt.EfficiencyRate, TimeBasis: mt.TimeBasis}
	// Trocar o cilindro para a máquina. Sem contar essas paradas a ordem cabe no
	// turno na conta e estoura no chão — é o erro clássico de quem planeja corte
	// a laser só pelo tempo de corte.
	if mt.ConsumptionPerHour != nil && mt.ConsumableCapacity != nil {
		uso := &machineentity.ConsumableUsage{PerHour: *mt.ConsumptionPerHour, CapacityPerRefill: *mt.ConsumableCapacity}
		if mt.ConsumableSwapMinutes != nil {
			uso.ReplacementMinutes = *mt.ConsumableSwapMinutes
		}
		if mt.ConsumableUnit != nil {
			uso.Unit = *mt.ConsumableUnit
		}
		if mt.ConsumableName != nil {
			uso.Description = *mt.ConsumableName
		}
		imt.Consumable = uso
	}
	m := &machineentity.Machine{EfficiencyRate: mt.MachineEfficiencyRate}
	result := machinesvc.CalculateProductionTime(imt, m, qty, 1, mt.WorkingHoursPerDay*60)
	if math.IsNaN(result.TotalMinutes) || math.IsInf(result.TotalMinutes, 0) || result.TotalMinutes <= 0 {
		return 0, domainerr.NewValidation(fmt.Sprintf("duração inválida do item %d", mt.ItemCode))
	}
	return result.TotalMinutes, nil
}

// escolherMaquinaQueTerminaAntes distribui a ordem entre máquinas equivalentes.
//
// Três serras iguais não são uma preferida e duas reservas: são três recursos
// que fazem o mesmo. Escolher sempre a primeira empilha a fila nela e deixa as
// outras ociosas — o pior jeito de errar, porque o prazo estica com capacidade
// sobrando ao lado e nada na tela denuncia isso.
//
// `alocar` é injetado para esta decisão poder ser testada sem banco: o laço de
// escolha é a regra, a alocação é o detalhe.
func escolherMaquinaQueTerminaAntes(
	step machineStep,
	alocar func(*entity.MachineTimeInfo) ([]machinesvc.CapacityWindow, error),
) ([]machinesvc.CapacityWindow, *entity.MachineTimeInfo, error) {
	candidatas := step.alternativas
	if len(candidatas) == 0 {
		candidatas = []*entity.MachineTimeInfo{step.profile}
	}
	var melhores []machinesvc.CapacityWindow
	var escolhida *entity.MachineTimeInfo
	var primeiroErro error

	for _, cand := range candidatas {
		slots, err := alocar(cand)
		if err != nil {
			// O primeiro motivo explica a recusa melhor que "nenhuma disponível".
			if primeiroErro == nil {
				primeiroErro = fmt.Errorf("máquina %d: %w", cand.MachineCode, err)
			}
			continue
		}
		if len(slots) == 0 {
			continue
		}
		if melhores == nil || slots[len(slots)-1].End.Before(melhores[len(melhores)-1].End) {
			melhores, escolhida = slots, cand
		}
	}
	if melhores == nil {
		if primeiroErro != nil {
			return nil, nil, primeiroErro
		}
		return nil, nil, domainerr.NewValidation("nenhuma máquina equivalente tem janela disponível")
	}
	return melhores, escolhida, nil
}

type machinePlanningCalendar interface {
	MachinePlanningWindows(context.Context, int64, int64, time.Time, time.Time) ([]machinesvc.CapacityWindow, []machinesvc.CapacityWindow, error)
	SaveMachineSlots(context.Context, int64, int64, *int64, []machinesvc.CapacityWindow, bool) error
}

func (s *MRPServiceImpl) processMachineIntegration(ctx context.Context, planCode int64, allCodes []int64) error {
	if tx, ok := s.MRPRepo.(interface {
		WithMachinePlanningTransaction(context.Context, func(mrprepo.MRPCalculationRepository) error) error
	}); ok {
		var semAgenda []string
		err := tx.WithMachinePlanningTransaction(ctx, func(repo mrprepo.MRPCalculationRepository) error {
			copy := *s
			copy.MRPRepo = repo
			falhas, programadas, err := copy.scheduleMachineIntegration(ctx, planCode, allCodes)
			if err != nil {
				return err
			}
			// Nenhuma ordem coube: o replanejamento não trouxe nada. Devolver o
			// erro aqui desfaz a transação e mantém a programação anterior
			// completa, em vez de trocá-la por um plano vazio.
			if programadas == 0 && len(falhas) > 0 {
				return nenhumaOrdemProgramada(falhas)
			}
			semAgenda = falhas
			return nil
		})
		if err != nil {
			return err
		}
		return ordensSemAgenda(semAgenda)
	}
	semAgenda, _, err := s.scheduleMachineIntegration(ctx, planCode, allCodes)
	if err != nil {
		return err
	}
	return ordensSemAgenda(semAgenda)
}

// nenhumaOrdemProgramada é a falha total do replanejamento de máquinas.
func nenhumaOrdemProgramada(falhas []string) error {
	return domainerr.NewValidation(fmt.Sprintf(
		"nenhuma ordem de fabricação pôde ser programada nas máquinas; a programação anterior foi mantida: %s", falhas[0]))
}

// ordensSemAgenda resume as ordens que não couberam na agenda das máquinas.
// As demais ficam programadas: antes uma única ordem sem janela (ou com o
// cadastro incompleto) desfazia a programação de máquina do plano inteiro.
func ordensSemAgenda(falhas []string) error {
	if len(falhas) == 0 {
		return nil
	}
	const mostrar = 5
	texto := strings.Join(falhas[:min(len(falhas), mostrar)], "; ")
	if len(falhas) > mostrar {
		texto += fmt.Sprintf("; e mais %d", len(falhas)-mostrar)
	}
	return domainerr.NewValidation(fmt.Sprintf(
		"%d ordem(ns) de fabricação ficaram sem agenda de máquina (as demais foram programadas): %s", len(falhas), texto))
}

// falhaDeCadastroOuCapacidade diz se o erro é da ordem (capacidade que não
// cabe, cadastro que falta) e não da infraestrutura. Só o primeiro deixa o
// planejamento seguir com as outras ordens.
func falhaDeCadastroOuCapacidade(err error) bool {
	var v *domainerr.ValidationError
	return errors.As(err, &v)
}

// scheduleMachineIntegration programa as sugestões de fabricação nas máquinas.
// Devolve as ordens que ficaram sem agenda (uma frase por ordem), quantas foram
// programadas e, à parte, o erro que impede o planejamento inteiro.
func (s *MRPServiceImpl) scheduleMachineIntegration(ctx context.Context, planCode int64, allCodes []int64) ([]string, int, error) {
	var semAgenda []string
	programadas := 0
	times, err := s.MRPRepo.ListItemMachineTimes(ctx, allCodes)
	if err != nil {
		return nil, 0, fmt.Errorf("carregando produtividade por item: %w", err)
	}

	suggestions, err := s.MRPRepo.ListSuggestionsByPlan(ctx, planCode)
	if err != nil {
		return nil, 0, err
	}
	if cleaner, ok := s.MRPRepo.(interface {
		ClearUnreleasedMachineAllocations(context.Context, int64) error
	}); ok {
		if err := cleaner.ClearUnreleasedMachineAllocations(ctx, planCode); err != nil {
			return nil, 0, err
		}
	}
	sort.SliceStable(suggestions, func(i, j int) bool {
		if suggestions[i].NeedDate.Equal(suggestions[j].NeedDate) {
			return suggestions[i].Code < suggestions[j].Code
		}
		return suggestions[i].NeedDate.Before(suggestions[j].NeedDate)
	})
	suggestions, err = orderMachineSuggestions(suggestions)
	if err != nil {
		return nil, 0, err
	}
	ready := materialReadiness{}
	reserved := map[int64][]machinesvc.CapacityWindow{}
	calendar, finite := s.MRPRepo.(machinePlanningCalendar)
	for _, sug := range suggestions {
		if sug.OrderType != "FABRICACAO" {
			ready.record(sug, sug.NeedDate)
			continue
		}
		if reader, ok := s.MRPRepo.(interface {
			SuggestionHasOrder(context.Context, int64) (bool, error)
		}); ok {
			exists, err := reader.SuggestionHasOrder(ctx, sug.Code)
			if err != nil {
				return nil, 0, err
			}
			if exists {
				end := sug.NeedDate
				if sug.EstimatedEndAt != nil {
					end = *sug.EstimatedEndAt
				}
				ready.record(sug, end)
				continue
			}
		}
		steps, err := s.machineSteps(ctx, sug, times[sug.ItemCode])
		if err != nil {
			if !falhaDeCadastroOuCapacidade(err) {
				return nil, 0, err
			}
			semAgenda = append(semAgenda, err.Error())
			ready.record(sug, sug.NeedDate)
			continue
		}
		if len(steps) == 0 {
			ready.record(sug, sug.NeedDate)
			continue
		}
		totalMinutes := 0.0
		for _, step := range steps {
			totalMinutes += step.minutes
		}
		day := sug.NeedDate
		if sug.StartDate != nil {
			day = *sug.StartDate
		}
		if sug.RequestedStartDate != nil {
			day = *sug.RequestedStartDate
		}
		if pronto := ready.readyFor(sug.ItemCode, day); pronto.After(day) {
			day = pronto
		}
		firstStart := day
		// Primeiro calcula-se a agenda da ordem inteira, sem gravar: se uma
		// etapa não couber, a ordem fica sem agenda e nada dela é gravado pela
		// metade. As reservas das etapas anteriores da própria ordem entram
		// como ocupação das seguintes.
		type etapaAlocada struct {
			slots   []machinesvc.CapacityWindow
			maquina *entity.MachineTimeInfo
		}
		alocadas := make([]etapaAlocada, len(steps))
		daOrdem := map[int64][]machinesvc.CapacityWindow{}
		var naoCoube error
		for i, step := range steps {
			day = day.Add(step.before)
			if !finite {
				continue
			}
			inicio := day
			slots, escolhida, err := escolherMaquinaQueTerminaAntes(step, func(cand *entity.MachineTimeInfo) ([]machinesvc.CapacityWindow, error) {
				windows, busy, err := calendar.MachinePlanningWindows(ctx, cand.MachineID, planCode, inicio, inicio.AddDate(1, 0, 0))
				if err != nil {
					return nil, err
				}
				ocupado := make([]machinesvc.CapacityWindow, 0, len(busy)+len(reserved[cand.MachineID])+len(daOrdem[cand.MachineID]))
				ocupado = append(append(append(ocupado, busy...), reserved[cand.MachineID]...), daOrdem[cand.MachineID]...)
				if step.cycles > 0 {
					return machinesvc.AllocateMachineCycles(inicio, step.cycles, step.cycleMinutes, step.setupMinutes, windows, ocupado)
				}
				return machinesvc.AllocateMachineTime(inicio, step.minutes, windows, ocupado)
			})
			if err != nil {
				if !falhaDeCadastroOuCapacidade(err) {
					return nil, 0, fmt.Errorf("item %d: %w", sug.ItemCode, err)
				}
				naoCoube = err
				break
			}
			alocadas[i] = etapaAlocada{slots: slots, maquina: escolhida}
			daOrdem[escolhida.MachineID] = append(daOrdem[escolhida.MachineID], slots...)
			day = slots[len(slots)-1].End.Add(step.after)
		}
		if naoCoube != nil {
			semAgenda = append(semAgenda, fmt.Sprintf("item %d: %v", sug.ItemCode, naoCoube))
			ready.record(sug, sug.NeedDate)
			continue
		}
		if finite {
			for i, a := range alocadas {
				mt := a.maquina
				// Persist the parent allocation before slots: INSERT ... SELECT
				// cannot create reservations until this row exists.
				if i == 0 {
					if err := s.MRPRepo.UpdatePlannedOrderMachine(ctx, sug.Code, mt.MachineID, totalMinutes); err != nil {
						return nil, 0, err
					}
				}
				if err = calendar.SaveMachineSlots(ctx, sug.Code, mt.MachineID, steps[i].operationID, a.slots, i == 0); err != nil {
					return nil, 0, err
				}
				steps[i].profile = mt // a etapa passa a apontar a máquina REALMENTE alocada
				reserved[mt.MachineID] = append(reserved[mt.MachineID], a.slots...)
				if i == 0 {
					firstStart = a.slots[0].Start
				}
			}
		}
		// Só agora se sabe qual máquina levou a primeira etapa.
		if err = s.MRPRepo.UpdatePlannedOrderMachine(ctx, sug.Code, steps[0].profile.MachineID, totalMinutes); err != nil {
			return nil, 0, err
		}
		if err = s.MRPRepo.CreateMachineSchedule(ctx, &entity.MachineScheduleInfo{PlanCode: planCode, PlannedOrderCode: sug.Code, MachineID: steps[0].profile.MachineID, ScheduleDate: firstStart, ProductionTime: totalMinutes}); err != nil {
			return nil, 0, err
		}

		if writer, ok := s.MRPRepo.(interface {
			SaveMachineCompletion(context.Context, int64, time.Time) error
		}); ok {
			if err := writer.SaveMachineCompletion(ctx, sug.Code, day); err != nil {
				return nil, 0, err
			}
		}
		ready.record(sug, day)
		programadas++
	}
	return semAgenda, programadas, nil
}

// Components are planned before their consuming parent. When several orders
// share a component, waiting for all its dependent suggestions is conservative.
func orderMachineSuggestions(in []*entity.PlannedOrderSuggestion) ([]*entity.PlannedOrderSuggestion, error) {
	out := make([]*entity.PlannedOrderSuggestion, 0, len(in))
	done := map[int64]bool{}
	for len(out) < len(in) {
		progress := false
		for _, s := range in {
			if done[s.Code] {
				continue
			}
			ready := true
			for _, child := range in {
				if child.ParentItemCode != nil && *child.ParentItemCode == s.ItemCode && !done[child.Code] {
					ready = false
					break
				}
			}
			if ready {
				out = append(out, s)
				done[s.Code] = true
				progress = true
			}
		}
		if !progress {
			return nil, fmt.Errorf("dependências de materiais em ciclo nas sugestões do MRP")
		}
	}
	return out, nil
}

// materialReadiness guarda, por item pai, quando cada lote de componente fica
// pronto e para quando ele foi pedido. Antes era um único horário por item pai —
// o do componente mais tardio de TODAS as ordens —, e a ordem que vence amanhã
// esperava a compra pedida para a ordem do mês que vem.
type materialReadiness map[int64][]componentReady

type componentReady struct {
	need time.Time
	end  time.Time
}

func (r materialReadiness) record(s *entity.PlannedOrderSuggestion, end time.Time) {
	if s.ParentItemCode != nil {
		r[*s.ParentItemCode] = append(r[*s.ParentItemCode], componentReady{need: s.NeedDate, end: end})
	}
}

// readyFor devolve quando ficam prontos os componentes que a ordem do pai que
// começa em `start` consome: os pedidos para até essa data. Componente pedido
// para depois abastece uma ordem posterior do mesmo item.
func (r materialReadiness) readyFor(parent int64, start time.Time) time.Time {
	var pronto time.Time
	for _, c := range r[parent] {
		if !c.need.After(start) && c.end.After(pronto) {
			pronto = c.end
		}
	}
	return pronto
}
