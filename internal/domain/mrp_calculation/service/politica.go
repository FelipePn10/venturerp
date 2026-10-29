package service

import (
	"context"
	"fmt"
	"time"

	inddemandentity "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
)

// PlanningPolicyReader devolve a política de planejamento efetiva dos itens
// que têm regra própria ou de classificação (VPLA0310). empresa é a política
// dos parâmetros, que os campos vazios herdam.
type PlanningPolicyReader interface {
	PoliticasDePlanejamento(ctx context.Context, empresa entity.PoliticaEfetiva) (map[int64]entity.PoliticaEfetiva, error)
}

// carregarPoliticas preenche params.Politicas. Sem leitor, todos os itens
// seguem os parâmetros da empresa.
func (s *MRPServiceImpl) carregarPoliticas(ctx context.Context, params *entity.TypedPlanningParams) error {
	if s.Politicas == nil {
		return nil
	}
	pols, err := s.Politicas.PoliticasDePlanejamento(ctx, params.PoliticaDaEmpresa())
	if err != nil {
		return fmt.Errorf("lendo as políticas de planejamento: %w", err)
	}
	params.Politicas = pols
	return nil
}

// PoliticaDeConsumo é a janela e a estratégia de consumo de cada item,
// como o consumo pela venda (independent_demand) as entende.
func PoliticaDeConsumo(params *entity.TypedPlanningParams) inddemandentity.PoliticaDeConsumo {
	return func(item int64) inddemandentity.Consumo {
		p := params.Politica(item)
		return inddemandentity.Consumo{Antes: p.ConsumoDiasAntes, Depois: p.ConsumoDiasDepois, SobEncomenda: p.SobEncomenda()}
	}
}

// datasDaSugestao calcula o recebimento e o início da sugestão do item:
//   - tempo de segurança: o recebimento vem esse tanto (dias úteis) antes da
//     necessidade;
//   - prazo: o início é o recebimento menos o prazo (dias úteis);
//   - horizonte firme: o início não cai antes de hoje + horizonte. Se caía, vai
//     para o fim do horizonte e o recebimento acompanha (o prazo fica o
//     mesmo); inicioPedido devolve o início que a necessidade pedia.
func (s *MRPServiceImpl) datasDaSugestao(ctx context.Context, necessidade time.Time, prazo int, pol entity.PoliticaEfetiva) (recebimento, inicio time.Time, inicioPedido *time.Time, err error) {
	recebimento = necessidade
	if pol.TempoSegurancaDias > 0 {
		r, errCal := s.subtractWorkdays(ctx, necessidade, pol.TempoSegurancaDias)
		if errCal != nil || r.IsZero() {
			err = errCal
			r = necessidade.AddDate(0, 0, -pol.TempoSegurancaDias)
		}
		recebimento = r
	}
	inicio, errCal := s.subtractWorkdays(ctx, recebimento, prazo)
	if errCal != nil || inicio.IsZero() {
		if err == nil {
			err = errCal
		}
		inicio = recebimento.AddDate(0, 0, -prazo)
	}
	inicio, recebimento, inicioPedido = aplicarHorizonteFirme(inicio, recebimento, pol.HorizonteFirmeDias, today())
	return recebimento, inicio, inicioPedido, err
}

// aplicarHorizonteFirme: dentro do horizonte firme (de hoje até hoje + dias)
// o MRP não cria ordem nova. A sugestão que começaria nele começa no fim dele,
// e o recebimento anda o mesmo tanto. Devolve o início pedido quando mexeu.
func aplicarHorizonteFirme(inicio, recebimento time.Time, dias int, hoje time.Time) (time.Time, time.Time, *time.Time) {
	if dias <= 0 {
		return inicio, recebimento, nil
	}
	fim := hoje.AddDate(0, 0, dias)
	if !inicio.Before(fim) {
		return inicio, recebimento, nil
	}
	pedido := inicio
	recebimento = recebimento.Add(fim.Sub(inicio))
	return fim, recebimento, &pedido
}
