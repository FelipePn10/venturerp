package sales_quotation_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
)

// PlanoDePagamento resolve a condição de pagamento do orçamento em parcelas com
// valor e data.
//
// É a pergunta que o cliente faz antes de aprovar — "quanto eu pago e quando?"
// — e que o sistema não sabia responder: a condição guardava só os dias de cada
// parcela, sem percentual e sem dizer de que evento os dias contavam. Com
// "30% de entrada, 20% na entrega e o restante em 28/56 dias", a proposta saía
// com o nome da condição e nenhum número.
//
// O plano é calculado, não gravado: ele acompanha o total e a data de entrega do
// orçamento, que mudam enquanto a proposta está sendo montada. Vira título de
// verdade no faturamento.
func (uc *UseCase) PlanoDePagamento(ctx context.Context, code int64) (*response.PlanoDePagamentoResponse, error) {
	q, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if q.PaymentTermCode == nil {
		return nil, errorsuc.NewValidationError("este orçamento ainda não tem condição de pagamento")
	}
	cond, err := uc.Customers.GetPaymentConditionByCode(ctx, *q.PaymentTermCode)
	if err != nil {
		return nil, errorsuc.NewValidationError("a condição de pagamento do orçamento não foi encontrada")
	}

	datas := customerentity.DatasBase{Emissao: q.EmissionDate}
	if q.DeliveryDate != nil {
		datas.Entrega = q.DeliveryDate
	}

	parcelas, err := customerentity.CalcularPlano(cond, q.TotalNet, datas)
	if err != nil {
		// Percentual que não fecha é problema do CADASTRO da condição, não do
		// orçamento: dizer isso aqui evita o usuário procurar o erro na proposta.
		return nil, errorsuc.NewValidationError(err.Error())
	}

	out := &response.PlanoDePagamentoResponse{
		CondicaoCode:      cond.Code,
		CondicaoDescricao: cond.Description,
		Total:             q.TotalNet.InexactFloat64(),
		Parcelas:          make([]response.ParcelaPlanoResponse, 0, len(parcelas)),
	}
	estimadas := 0
	for _, p := range parcelas {
		if p.Estimado {
			estimadas++
		}
		out.Parcelas = append(out.Parcelas, response.ParcelaPlanoResponse{
			Numero:       p.Numero,
			Percentual:   p.Percentual.InexactFloat64(),
			Valor:        p.Valor.InexactFloat64(),
			Vencimento:   p.Vencimento.Format("2006-01-02"),
			DiasPrazo:    p.DiasPrazo,
			Evento:       string(p.Evento),
			EventoRotulo: p.Evento.Rotulo(),
			Descricao:    p.Descricao,
			Estimado:     p.Estimado,
			DocumentType: p.DocumentType,
		})
	}
	if estimadas > 0 {
		out.Aviso = "há parcela contada da entrega ou do faturamento e o orçamento ainda não tem essa data: o vencimento mostrado é projetado pela emissão e muda quando a data for definida"
	}
	return out, nil
}
