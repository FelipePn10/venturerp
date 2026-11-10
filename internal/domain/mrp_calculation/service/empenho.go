package service

import (
	"context"
	"time"

	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports"
)

// DemandTypeOrderAllocation marca a necessidade que vem do empenho de uma OF.
const DemandTypeOrderAllocation = "ORDER_ALLOCATION"

// empenhoDasOrdensAbertas transforma o material ainda não consumido das OFs
// abertas em demanda dependente. A data é a do início da OF ou, quando o
// componente entra numa etapa do roteiro, a fração do prazo da OF em que a
// etapa começa — a mesma regra das sugestões (necessidadeNaEtapa).
func (s *MRPServiceImpl) empenhoDasOrdensAbertas(ctx context.Context) ([]*entity.MRPInput, error) {
	porta, ok := s.SupplyPort.(ports.OrderAllocationsPort)
	if !ok {
		return nil, nil
	}
	empenhos, err := porta.ListOpenOrderAllocations(ctx)
	if err != nil {
		return nil, err
	}
	cache := map[string]map[int16]float64{}
	out := make([]*entity.MRPInput, 0, len(empenhos))
	for _, e := range empenhos {
		if !e.Quantity.IsPositive() {
			continue
		}
		data, errEtapa := s.dataDoEmpenho(ctx, cache, e)
		if errEtapa != nil {
			// Sem o roteiro, o componente é necessário no início da OF: cedo,
			// nunca tarde.
			data = e.OrderStart
		}
		pai, ordem := e.ParentItemCode, e.ProductionOrderID
		out = append(out, &entity.MRPInput{
			ItemCode:           e.ItemCode,
			Mask:               e.Mask,
			Quantity:           e.Quantity,
			NeedDate:           data,
			DemandType:         DemandTypeOrderAllocation,
			ParentItemCode:     &pai,
			ParentStepSequence: e.Step,
			SourceCode:         &ordem,
			PegKind:            entity.PegDemandProductionOrder,
			PegCode:            &ordem,
		})
	}
	return out, nil
}

func (s *MRPServiceImpl) dataDoEmpenho(ctx context.Context, cache map[string]map[int16]float64, e ports.OrderAllocation) (time.Time, error) {
	if e.Step == nil || s.RoutingRepo == nil || !e.OrderEnd.After(e.OrderStart) {
		return e.OrderStart, nil
	}
	inicio := e.OrderStart
	ordem := &entity.PlannedOrderSuggestion{StartDate: &inicio, NeedDate: e.OrderEnd, Quantity: e.ParentQuantity}
	return s.necessidadeNaEtapa(ctx, cache, e.ParentItemCode, e.ParentMask, ordem, e.Step)
}

// DemandTypeFirmPlanned marca a necessidade de componente de planejada firme sem OF.
const DemandTypeFirmPlanned = "FIRM_PLANNED_ORDER"

func (s *MRPServiceImpl) planejadasFirmesSemOF(ctx context.Context) ([]ports.FirmPlannedWithoutOrder, error) {
	porta, ok := s.SupplyPort.(ports.FirmPlannedWithoutOrderPort)
	if !ok {
		return nil, nil
	}
	return porta.ListFirmPlannedProductionWithoutOrder(ctx)
}

// demandaDasPlanejadasSemOF explode a estrutura da planejada firme sem OF: os
// componentes são necessários na data da etapa em que entram (ou no início).
func (s *MRPServiceImpl) demandaDasPlanejadasSemOF(ctx context.Context, planejadas []ports.FirmPlannedWithoutOrder, bomMap map[int64][]*structentity.ItemStructure, formula int, maskVars *maskVarCache) []*entity.MRPInput {
	cache := map[string]map[int16]float64{}
	var out []*entity.MRPInput
	for _, p := range planejadas {
		if !p.Quantity.IsPositive() {
			continue
		}
		inicio := p.Start
		ordem := &entity.PlannedOrderSuggestion{StartDate: &inicio, NeedDate: p.Need, Quantity: p.Quantity}
		pai, codigo := p.ItemCode, p.Code
		for _, filho := range explodeFromBOMWithVars(bomMap, p.ItemCode, p.Mask, p.Quantity, 1, formula, maskVars.vars(ctx, p.ItemCode, p.Mask), inicio) {
			data, err := s.necessidadeNaEtapa(ctx, cache, p.ItemCode, p.Mask, ordem, filho.ParentStepSequence)
			if err != nil {
				data = inicio
			}
			filho.NeedDate = data
			filho.DemandType = DemandTypeFirmPlanned
			filho.ParentItemCode = &pai
			filho.SourceCode = &codigo
			filho.PegKind, filho.PegCode = entity.PegDemandFirmPlanned, &codigo
			out = append(out, filho)
		}
	}
	return out
}
