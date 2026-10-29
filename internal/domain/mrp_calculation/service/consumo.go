package service

import (
	"context"
	"fmt"
	"time"

	inddemandentity "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	forecastentity "github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity"
)

// SalesForConsumption devolve a quantidade vendida dos itens por linha de
// pedido (aberta ou já atendida, sem a cancelada).
type SalesForConsumption interface {
	Vendas(ctx context.Context, itens []int64) ([]inddemandentity.Venda, error)
}

// consumirPelaVenda aplica o consumo da previsão e da demanda de estoque pelos
// pedidos de venda (parâmetros 46/47) e o saldo das demandas de finalidade
// própria. O pedido de venda continua entrando inteiro, pela fonte própria:
// o que sai daqui é só a parte da expectativa que a venda ainda não cobriu.
// No plano de cenário, ajustes e pedidosLidos (os pedidos antes do cenário)
// levam as alterações de pedido do cenário para as vendas que consomem.
func (s *MRPServiceImpl) consumirPelaVenda(
	ctx context.Context,
	params *entity.TypedPlanningParams,
	demands []*inddemandentity.IndependentDemand,
	previsao []*entity.MRPInput,
	ajustes *entity.ScenarioAdjustments,
	pedidosLidos []*entity.MRPInput,
) ([]*inddemandentity.IndependentDemand, []*entity.MRPInput, error) {
	if len(demands) == 0 && len(previsao) == 0 {
		return demands, previsao, nil
	}
	vistos := map[int64]bool{}
	var itens []int64
	for _, d := range demands {
		if !vistos[d.ItemCode] {
			vistos[d.ItemCode] = true
			itens = append(itens, d.ItemCode)
		}
	}
	prev := make([]inddemandentity.Consumivel, len(previsao))
	for i, p := range previsao {
		prev[i] = inddemandentity.Consumivel{Chave: inddemandentity.ChaveItem(p.ItemCode, p.Mask), Item: p.ItemCode, Data: p.NeedDate, Quantidade: p.Quantity}
		if !vistos[p.ItemCode] {
			vistos[p.ItemCode] = true
			itens = append(itens, p.ItemCode)
		}
	}
	var vendas []inddemandentity.Venda
	if s.Vendas != nil {
		var errVendas error
		// Sem as vendas o consumo não acontece: o erro volta e o cálculo
		// para (a previsão somaria com o pedido).
		vendas, errVendas = s.Vendas.Vendas(ctx, itens)
		if errVendas != nil {
			return demands, previsao, fmt.Errorf("lendo as vendas que consomem a previsão: %w", errVendas)
		}
		vendas = ajustarVendasDoCenario(ajustes, pedidosLidos, vendas)
	}
	// A janela e a estratégia (sob encomenda) são as da política de cada item.
	pos, posPrev, _ := inddemandentity.PosicionarComPolitica(demands, prev, vendas, PoliticaDeConsumo(params), today())

	outD := make([]*inddemandentity.IndependentDemand, 0, len(demands))
	for i, d := range demands {
		if !pos[i].APlanejar.IsPositive() {
			continue
		}
		c := *d
		c.Quantity = pos[i].APlanejar
		outD = append(outD, &c)
	}
	outP := make([]*entity.MRPInput, 0, len(previsao))
	for i, p := range previsao {
		if !posPrev[i].Resto.IsPositive() {
			continue
		}
		c := *p
		c.Quantity = posPrev[i].Resto
		outP = append(outP, &c)
	}
	return outD, outP, nil
}

// semSemanasPassadas tira a previsão das semanas que já terminaram
// (parâmetro 11): previsão de semana passada não é necessidade.
func semSemanasPassadas(previsao []*entity.MRPInput, hoje time.Time) []*entity.MRPInput {
	out := previsao[:0]
	for _, p := range previsao {
		// NeedDate é a segunda-feira da semana; a semana termina no domingo.
		if !p.NeedDate.AddDate(0, 0, 7).After(hoje) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// ConsumiveisDaPrevisao é a previsão de vendas como o MRP a enxerga (semana →
// segunda-feira, sem as semanas passadas quando o parâmetro 11 manda), para a
// tela da demanda mostrar o mesmo consumo do cálculo.
func ConsumiveisDaPrevisao(fs []*forecastentity.SalesForecast, descontarPassadas bool, hoje time.Time) []inddemandentity.Consumivel {
	out := make([]inddemandentity.Consumivel, 0, len(fs))
	for _, f := range fs {
		data := mrpWeekToDate(f.Year, f.Week)
		if descontarPassadas && !data.AddDate(0, 0, 7).After(hoje) {
			continue
		}
		mask := ""
		if f.Mask != nil {
			mask = *f.Mask
		}
		out = append(out, inddemandentity.Consumivel{Chave: inddemandentity.ChaveItem(f.ItemCode, mask), Item: f.ItemCode, Data: data, Quantidade: dq(f.Quantity)})
	}
	return out
}

// Hoje é a data do cálculo, para quem precisa da mesma referência do MRP.
func Hoje() time.Time { return today() }
