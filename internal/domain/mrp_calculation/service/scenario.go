package service

import (
	"context"
	"github.com/shopspring/decimal"
	"strconv"

	inddemandentity "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
)

// ScenarioReader devolve os ajustes do cenário dono do plano. O motor só o
// consulta quando o plano é de cenário (production_plans.is_scenario).
type ScenarioReader interface {
	ScenarioAdjustments(ctx context.Context, planCode int64) (*entity.ScenarioAdjustments, error)
}

// applyScenarioDemands aplica os ajustes de demanda do cenário: altera ou
// retira (quantidade 0) demanda independente e pedido de venda pelo código, e
// acrescenta as demandas novas. Devolve cópias — os registros lidos do banco
// não são alterados.
func applyScenarioDemands(adj *entity.ScenarioAdjustments, demands []*inddemandentity.IndependentDemand, inputs []*entity.MRPInput, planCode int64) ([]*inddemandentity.IndependentDemand, []*entity.MRPInput) {
	if adj == nil || len(adj.Demands) == 0 {
		return demands, inputs
	}
	porIndependente := map[int64]entity.ScenarioDemandChange{}
	porPedido := map[int64]entity.ScenarioDemandChange{}
	var novas []entity.ScenarioDemandChange
	for _, c := range adj.Demands {
		switch c.RefKind {
		case entity.ScenarioDemandIndependent:
			if c.RefCode != nil {
				porIndependente[*c.RefCode] = c
			}
		case entity.ScenarioDemandSalesOrder:
			if c.RefCode != nil {
				porPedido[*c.RefCode] = c
			}
		case entity.ScenarioDemandNew:
			novas = append(novas, c)
		}
	}

	outD := make([]*inddemandentity.IndependentDemand, 0, len(demands))
	for _, d := range demands {
		c, ok := porIndependente[d.CodeDemand]
		if !ok {
			outD = append(outD, d)
			continue
		}
		copia := *d
		if c.Quantity != nil {
			copia.Quantity = *c.Quantity
		}
		if c.NeedDate != nil {
			copia.DemandDate = *c.NeedDate
		}
		if copia.Quantity.IsPositive() {
			outD = append(outD, &copia)
		}
	}

	outI := make([]*entity.MRPInput, 0, len(inputs)+len(novas))
	for _, in := range inputs {
		if in.DemandType != "SALES_ORDER" || in.SourceCode == nil {
			outI = append(outI, in)
			continue
		}
		c, ok := porPedido[*in.SourceCode]
		if !ok {
			outI = append(outI, in)
			continue
		}
		copia := *in
		if c.Quantity != nil {
			copia.Quantity = *c.Quantity
		}
		if c.NeedDate != nil {
			copia.NeedDate = *c.NeedDate
		}
		if copia.Quantity.IsPositive() {
			outI = append(outI, &copia)
		}
	}
	for _, c := range novas {
		if c.Quantity == nil || !c.Quantity.IsPositive() || c.NeedDate == nil || c.ItemCode <= 0 {
			continue
		}
		id := c.ID
		outI = append(outI, &entity.MRPInput{
			PlanCode:   planCode,
			ItemCode:   c.ItemCode,
			Mask:       c.Mask,
			Quantity:   *c.Quantity,
			NeedDate:   *c.NeedDate,
			DemandType: "INDEPENDENT_DEMAND",
			PegKind:    entity.PegDemandSimulated,
			PegCode:    &id,
		})
	}
	return outD, outI
}

// ajustarVendasDoCenario leva a alteração de pedido do cenário para a venda
// que consome a previsão e a demanda de estoque. A venda é a quantidade
// vendida (atendida + aberta) e o cenário altera a aberta: a diferença entre
// a aberta nova e a lida do banco vai para a venda; a data nova também.
func ajustarVendasDoCenario(adj *entity.ScenarioAdjustments, pedidosLidos []*entity.MRPInput, vendas []inddemandentity.Venda) []inddemandentity.Venda {
	if adj == nil || len(adj.Demands) == 0 || len(vendas) == 0 {
		return vendas
	}
	porLinha := map[int64]entity.ScenarioDemandChange{}
	for _, c := range adj.Demands {
		if c.RefKind == entity.ScenarioDemandSalesOrder && c.RefCode != nil {
			porLinha[*c.RefCode] = c
		}
	}
	if len(porLinha) == 0 {
		return vendas
	}
	abertaLida := map[int64]decimal.Decimal{}
	for _, in := range pedidosLidos {
		if in.DemandType == "SALES_ORDER" && in.SourceCode != nil {
			abertaLida[*in.SourceCode] = abertaLida[*in.SourceCode].Add(in.Quantity)
		}
	}
	out := make([]inddemandentity.Venda, 0, len(vendas))
	for _, v := range vendas {
		if v.Linha == nil {
			out = append(out, v)
			continue
		}
		c, ok := porLinha[*v.Linha]
		if !ok {
			out = append(out, v)
			continue
		}
		if c.Quantity != nil {
			v.Quantidade = decimal.Max(decimal.Zero, v.Quantidade.Sub(abertaLida[*v.Linha]).Add(*c.Quantity))
		}
		if c.NeedDate != nil {
			v.Data = *c.NeedDate
		}
		out = append(out, v)
	}
	return out
}

// applyScenarioStock soma a entrada/baixa simulada ao saldo do item. O mapa é
// do cálculo em curso; o snapshot é copiado para não alterar o lido do banco.
func applyScenarioStock(adj *entity.ScenarioAdjustments, snapshots map[int64]*entity.StockSnapshot) {
	if adj == nil {
		return
	}
	for item, delta := range adj.StockDelta {
		var copia entity.StockSnapshot
		if atual := snapshots[item]; atual != nil {
			copia = *atual
		} else {
			copia = entity.StockSnapshot{ItemCode: item}
		}
		copia.Quantity = decimal.Max(decimal.Zero, copia.Quantity.Add(delta))
		snapshots[item] = &copia
	}
}

// scenarioRuleTable marca a regra de prazo criada pelo cenário.
const scenarioRuleTable = "CENARIO"

// applyScenarioLeadTimes acrescenta, por último, uma regra de prazo sem
// condição de máscara: no cálculo a última regra que casa vence, então ela
// substitui o prazo do item para todas as máscaras.
func applyScenarioLeadTimes(adj *entity.ScenarioAdjustments, rules map[int64][]*entity.ConfiguredItemRule) {
	if adj == nil {
		return
	}
	for item, dias := range adj.LeadTimeDays {
		copia := append([]*entity.ConfiguredItemRule(nil), rules[item]...)
		copia = append(copia, &entity.ConfiguredItemRule{ItemCode: item, TableType: scenarioRuleTable, FieldName: "lead_time", RuleValue: strconv.Itoa(dias), IsActive: true})
		rules[item] = copia
	}
}
