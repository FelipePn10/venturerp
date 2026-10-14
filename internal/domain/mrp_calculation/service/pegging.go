package service

import (
	"context"
	"sort"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
)

// peggingWriter grava o pegging do cálculo, substituindo o anterior do plano.
type peggingWriter interface {
	ReplacePegging(ctx context.Context, planCode int64, records []entity.PeggingRecord) error
}

// pegOrigin diz de onde vem a necessidade: a origem explícita (sugestão do
// pai, MPS, estoque de segurança, demanda simulada) ou a do tipo de demanda.
func pegOrigin(in *entity.MRPInput) (string, *int64) {
	if in.PegKind != "" {
		return in.PegKind, in.PegCode
	}
	switch in.DemandType {
	case "SALES_ORDER":
		return entity.PegDemandSalesOrder, in.SourceCode
	case "INDEPENDENT_DEMAND":
		return entity.PegDemandIndependent, in.SourceCode
	case "FORECAST":
		return entity.PegDemandForecast, nil
	}
	return entity.PegDemandOther, nil
}

// pegging vincula cada necessidade ao suprimento que a cobriu, na mesma ordem
// do netting do MRP: primeiro o estoque disponível, depois as ordens firmes
// que chegam até a data da necessidade, depois as sugestões do cálculo com
// necessidade até essa data. O que nada cobre fica como FALTA (item de
// terceiro, projeto, restrito, ou ordem firme atrasada sem sugestão).
func (b *exceptionBasis) pegging() []entity.PeggingRecord {
	items := make([]int64, 0, len(b.requirements))
	for item := range b.requirements {
		if !b.skip[item] {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i] < items[j] })

	out := []entity.PeggingRecord{}
	for _, item := range items {
		reqs := append([]requirement(nil), b.requirements[item]...)
		sort.SliceStable(reqs, func(i, j int) bool { return reqs[i].date.Before(reqs[j].date) })

		estoque := b.stock[item]
		firmes := append(b.supply[item][:0:0], b.supply[item]...)
		sort.SliceStable(firmes, func(i, j int) bool { return firmes[i].ArrivalDate.Before(firmes[j].ArrivalDate) })
		restanteFirme := make([]float64, len(firmes))
		for i, f := range firmes {
			restanteFirme[i] = f.Quantity
		}
		sugestoes := append([]suggestedSupply(nil), b.suggestions[item]...)
		sort.SliceStable(sugestoes, func(i, j int) bool {
			if sugestoes[i].needDate.Equal(sugestoes[j].needDate) {
				return sugestoes[i].code < sugestoes[j].code
			}
			return sugestoes[i].needDate.Before(sugestoes[j].needDate)
		})
		restanteSug := make([]float64, len(sugestoes))
		for i, s := range sugestoes {
			restanteSug[i] = s.quantity
		}

		for _, r := range reqs {
			falta := r.quantity
			add := func(kind string, code *int64, data *time.Time, qtd float64) {
				out = append(out, entity.PeggingRecord{
					ItemCode: item, DemandKind: r.kind, DemandCode: r.code, NeedDate: r.date,
					SupplyKind: kind, SupplyCode: code, SupplyDate: data, Quantity: qtd,
				})
				falta -= qtd
			}
			if estoque > quantityEpsilon && falta > quantityEpsilon {
				usado := minF(estoque, falta)
				estoque -= usado
				add(entity.PegSupplyStock, nil, nil, usado)
			}
			for i := range firmes {
				if falta <= quantityEpsilon {
					break
				}
				if restanteFirme[i] <= quantityEpsilon || firmes[i].ArrivalDate.After(r.date) {
					continue
				}
				usado := minF(restanteFirme[i], falta)
				restanteFirme[i] -= usado
				code, chegada := firmes[i].SourceCode, firmes[i].ArrivalDate
				add(entity.PegSupplyFirmOrder, &code, &chegada, usado)
			}
			for i := range sugestoes {
				if falta <= quantityEpsilon {
					break
				}
				if restanteSug[i] <= quantityEpsilon || sugestoes[i].needDate.After(r.date) {
					continue
				}
				usado := minF(restanteSug[i], falta)
				restanteSug[i] -= usado
				code, data := sugestoes[i].code, sugestoes[i].needDate
				add(entity.PegSupplySuggestion, &code, &data, usado)
			}
			if falta > quantityEpsilon {
				add(entity.PegSupplyShortage, nil, nil, falta)
			}
		}
	}
	return out
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
