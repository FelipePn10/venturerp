package planned_order_uc

import (
	"context"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	routingrepo "github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
)

// unidadeDoServico: em que unidade o item de serviço é comprado (unidade de
// compra do cadastro, senão a de estoque). A requisição gravava "SERV" fixo, e
// a geração do pedido recusava: "unidade de compra SERV fora do catálogo".
type unidadeDoServico interface {
	UnidadeDeCompraDoItem(ctx context.Context, itemCode int64) (string, error)
}

// redeDoRoteiro é o que o firmar precisa do roteiro para datar as operações.
type redeDoRoteiro interface {
	GetRouteForItem(ctx context.Context, itemCode int64, mask string) (*routingentity.ManufacturingRoute, error)
	GetRouteOperations(ctx context.Context, routeID int64) ([]*routingentity.RouteOperation, error)
	GetNetworkEdges(ctx context.Context, routeID int64) ([]*routingentity.NetworkEdge, error)
}

// datasDoServico data a linha de serviço pela posição da operação externa na
// ordem, não por "hoje + prazo do fornecedor" (que mandava pedir para amanhã o
// serviço da última etapa de uma OF de dois meses, ou para depois da entrega o
// da primeira).
//
// A operação começa na fração do processo em que o caminho crítico a põe —
// início mais cedo ÷ duração total, para a quantidade da ordem — aplicada à
// janela da ordem (início → necessidade), como o MRP data a necessidade de um
// componente por etapa. A peça vai ao terceiro nesse dia e volta "prazo" dias
// depois: entrega = início da operação + prazo (no máximo a necessidade da
// ordem); enviar/pedir até = entrega − prazo.
func (uc *FirmPlannedOrderUseCase) datasDoServico(ctx context.Context, order *entity.PlannedOrder, ops []*routingentity.ExternalOp) map[int64][2]time.Time {
	out := map[int64][2]time.Time{}
	fim := order.NeedDate
	if fim.IsZero() && order.EndDate != nil {
		fim = *order.EndDate
	}
	inicio := fim
	if order.StartDate != nil && !order.StartDate.IsZero() {
		inicio = *order.StartDate
	}
	hoje := time.Now().Truncate(24 * time.Hour)
	fracoes := uc.fracoesDoRoteiro(ctx, order)
	for _, op := range ops {
		prazo := int(op.LeadTimeDays)
		if fim.IsZero() {
			// Ordem sem data: o melhor que dá é hoje + prazo.
			entrega := hoje.AddDate(0, 0, prazo)
			out[op.RouteOpID] = [2]time.Time{hoje, entrega}
			continue
		}
		dia := inicio
		if f, ok := fracoes[op.RouteOpID]; ok && fim.After(inicio) {
			dia = inicio.Add(time.Duration(f * float64(fim.Sub(inicio))))
		}
		entrega := dia.AddDate(0, 0, prazo)
		if entrega.After(fim) {
			entrega = fim
		}
		out[op.RouteOpID] = [2]time.Time{entrega.AddDate(0, 0, -prazo), entrega}
	}
	return out
}

// fracoesDoRoteiro: etapa do roteiro → em que fração do processo começa.
func (uc *FirmPlannedOrderUseCase) fracoesDoRoteiro(ctx context.Context, order *entity.PlannedOrder) map[int64]float64 {
	out := map[int64]float64{}
	rede, ok := uc.ExternalOps.(redeDoRoteiro)
	if !ok {
		return out
	}
	mascara := ""
	if order.Mask != nil {
		mascara = *order.Mask
	}
	route, err := routingrepo.RoteiroParaQuantidade(ctx, rede, order.ItemCode, mascara, order.Quantity)
	if err != nil || route == nil {
		return out
	}
	ops, err := rede.GetRouteOperations(ctx, route.ID)
	if err != nil {
		return out
	}
	edges, err := rede.GetNetworkEdges(ctx, route.ID)
	if err != nil {
		return out
	}
	qtd := order.Quantity
	if !qtd.IsPositive() {
		qtd = decimal.NewFromInt(1)
	}
	caminho := routingentity.CriticalPath(ops, edges, qtd.InexactFloat64())
	if caminho.HasCycle() || caminho.TotalHours <= 0 {
		return out
	}
	for _, op := range ops {
		out[op.ID] = caminho.EarlyStartHours[op.ID] / caminho.TotalHours
	}
	return out
}

func (uc *FirmPlannedOrderUseCase) unidadeDoItemDeServico(ctx context.Context, itemCode int64) (*string, error) {
	if uc.UnidadesDeServico == nil {
		return nil, nil
	}
	u, err := uc.UnidadesDeServico.UnidadeDeCompraDoItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	u = strings.ToUpper(strings.TrimSpace(u))
	if u == "" {
		return nil, nil
	}
	return &u, nil
}
