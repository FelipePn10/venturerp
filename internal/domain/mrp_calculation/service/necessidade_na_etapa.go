package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	routingrepo "github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
	"github.com/shopspring/decimal"
)

// necessidadeNaEtapa data a necessidade de um componente pela etapa do roteiro
// do pai em que ele entra.
//
// Sem etapa, vale o de sempre: o componente é necessário no início da ordem do
// pai. Com etapa, ele é necessário quando a etapa começa. A etapa ocupa uma
// FRAÇÃO do processo do pai — início mais cedo dela ÷ duração do caminho
// crítico, as duas calculadas para a quantidade da ordem — e essa fração é
// aplicada à janela que o MRP deu ao pai (do início à entrega, em dias úteis).
// Assim a data fica coerente com o prazo do pai qualquer que seja a regra de
// prazo (roteiro, regra configurada, cenário). É o que SAP e Oracle chamam de
// alocação de componente à operação: a ferragem da montagem não precisa estar
// na fábrica no dia do corte.
//
// Os dias de terceiro antes da etapa não entram na fração (conservador: só
// antecipam a necessidade). Etapa que não existe no roteiro vigente do pai cai
// no início da ordem — o planejamento não perde a necessidade por um cadastro
// desencontrado; a tela do roteiro mostra a pendência.
func (s *MRPServiceImpl) necessidadeNaEtapa(
	ctx context.Context,
	cache map[string]map[int16]float64,
	pai int64, mascara string,
	sugestao *entity.PlannedOrderSuggestion,
	etapa *int16,
) (time.Time, error) {
	inicio := *sugestao.StartDate
	if etapa == nil || s.RoutingRepo == nil || sugestao.NeedDate.IsZero() || !sugestao.NeedDate.After(inicio) {
		return inicio, nil
	}
	qtd := sugestao.Quantity.InexactFloat64()
	chave := fmt.Sprintf("%d|%s|%g", pai, mascara, qtd)
	fracoes, ok := cache[chave]
	if !ok {
		var err error
		fracoes, err = s.fracoesDasEtapas(ctx, pai, mascara, qtd)
		if err != nil {
			return inicio, err
		}
		cache[chave] = fracoes
	}
	fracao, ok := fracoes[*etapa]
	if !ok || fracao <= 0 {
		return inicio, nil
	}
	janela, err := s.diasUteisEntre(ctx, inicio, sugestao.NeedDate)
	if err != nil {
		return inicio, err
	}
	dias := int(math.Floor(fracao * float64(janela)))
	data, err := s.addWorkdays(ctx, inicio, dias)
	if err != nil {
		return inicio, err
	}
	if data.After(sugestao.NeedDate) {
		return sugestao.NeedDate, nil
	}
	return data, nil
}

// fracoesDasEtapas: sequência da etapa → em que fração do processo ela começa
// (0 = no início, 0,5 = na metade), pelo roteiro vigente do item para a
// quantidade. Item sem roteiro devolve mapa vazio.
func (s *MRPServiceImpl) fracoesDasEtapas(ctx context.Context, item int64, mascara string, qtd float64) (map[int16]float64, error) {
	out := map[int16]float64{}
	var route *routingentity.ManufacturingRoute
	var err error
	if qtd > 0 {
		route, err = routingrepo.RoteiroParaQuantidade(ctx, s.RoutingRepo, item, mascara, decimal.NewFromFloat(qtd))
	} else {
		route, err = s.RoutingRepo.GetRouteForItem(ctx, item, mascara)
	}
	if err != nil {
		if domainerr.EhNaoEncontrado(err) {
			return out, nil
		}
		return out, fmt.Errorf("lendo o roteiro do item %d para datar os componentes: %w", item, err)
	}
	if route == nil {
		return out, nil
	}
	ops, err := s.RoutingRepo.GetRouteOperations(ctx, route.ID)
	if err != nil {
		return out, err
	}
	edges, err := s.RoutingRepo.GetNetworkEdges(ctx, route.ID)
	if err != nil {
		return out, err
	}
	if qtd <= 0 {
		qtd = 1
	}
	caminho := routingentity.CriticalPath(ops, edges, qtd)
	if caminho.HasCycle() || caminho.TotalHours <= 0 {
		return out, nil
	}
	for _, op := range ops {
		out[op.Sequence] = caminho.EarlyStartHours[op.ID] / caminho.TotalHours
	}
	return out, nil
}

// diasUteisEntre conta os dias úteis de `de` (exclusive) até `ate` (inclusive).
func (s *MRPServiceImpl) diasUteisEntre(ctx context.Context, de, ate time.Time) (int, error) {
	n, atual := 0, de
	for atual.Before(ate) && n < 3650 {
		prox, err := s.addWorkdays(ctx, atual, 1)
		if err != nil {
			return 0, err
		}
		if prox.After(ate) {
			break
		}
		atual = prox
		n++
	}
	return n, nil
}

// addWorkdays avança `dias` dias úteis no calendário industrial (sem
// calendário, dias corridos).
func (s *MRPServiceImpl) addWorkdays(ctx context.Context, de time.Time, dias int) (time.Time, error) {
	if dias <= 0 {
		return de, nil
	}
	if s.CalRepo == nil {
		return de.AddDate(0, 0, dias), nil
	}
	atual := de
	for i := 0; i < dias; i++ {
		prox, err := s.CalRepo.GetNextWorkday(ctx, atual.Year(), int(atual.Month()), atual.Day())
		if err != nil {
			return de.AddDate(0, 0, dias), nil
		}
		if !prox.After(atual) {
			prox = atual.AddDate(0, 0, 1)
		}
		atual = prox
	}
	return atual, nil
}
