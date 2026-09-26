package customer_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	"github.com/shopspring/decimal"
)

// ListInstallments devolve as parcelas da condição. A tela precisava disso para
// mostrar o que já foi montado: sem a lista, quem cadastra "30% entrada, 20%
// entrega e o restante em 28/56" grava às cegas e só descobre o erro no pedido.
func (uc *CustomerUseCase) ListInstallments(ctx context.Context, conditionCode int64) ([]response.InstallmentResponse, error) {
	pc, err := uc.repo.GetPaymentConditionByCode(ctx, conditionCode)
	if err != nil {
		return nil, errorsuc.NewNotFoundError("condição de pagamento não encontrada")
	}
	out := make([]response.InstallmentResponse, 0, len(pc.Installments))
	for _, i := range pc.Installments {
		out = append(out, response.InstallmentResponse{
			ID:                i.ID,
			InstallmentNumber: i.InstallmentNumber,
			DueDays:           i.DueDays,
			Description:       i.Description,
			DocumentType:      i.DocumentType,
			MovementType:      i.MovementType,
			CarrierID:         i.CarrierID,
			Percentage:        i.Percentage,
			BaseEvent:         i.BaseEvent,
		})
	}
	return out, nil
}

// DeleteInstallment apaga uma parcela da condição. Conferimos que a parcela é da
// condição informada: sem isso, um id de outra condição apagaria a parcela
// errada.
func (uc *CustomerUseCase) DeleteInstallment(ctx context.Context, conditionCode, installmentID int64) error {
	pc, err := uc.repo.GetPaymentConditionByCode(ctx, conditionCode)
	if err != nil {
		return errorsuc.NewNotFoundError("condição de pagamento não encontrada")
	}
	encontrada := false
	for _, i := range pc.Installments {
		if i.ID == installmentID {
			encontrada = true
			break
		}
	}
	if !encontrada {
		return errorsuc.NewNotFoundError("esta parcela não pertence à condição informada")
	}
	return uc.repo.DeleteInstallment(ctx, installmentID)
}

// SimularPlano mostra a condição em dinheiro e em datas antes de ela ser usada
// em qualquer pedido. É a conferência do cadastro: 30% de R$ 10.000 na entrada,
// 20% na entrega, 25% em 28 dias e 25% em 56 dias.
func (uc *CustomerUseCase) SimularPlano(ctx context.Context, conditionCode int64, total decimal.Decimal, emissao time.Time, entrega *time.Time) (*response.PlanoDePagamentoResponse, error) {
	pc, err := uc.repo.GetPaymentConditionByCode(ctx, conditionCode)
	if err != nil {
		return nil, errorsuc.NewNotFoundError("condição de pagamento não encontrada")
	}
	if total.IsNegative() {
		return nil, errorsuc.NewValidationError("o valor da simulação não pode ser negativo")
	}
	if total.IsZero() {
		// Um valor de referência deixa a simulação legível sem obrigar a tela a
		// inventar número.
		total = decimal.NewFromInt(1000)
	}
	if emissao.IsZero() {
		emissao = time.Now()
	}
	parcelas, err := entity.CalcularPlano(pc, total, entity.DatasBase{Emissao: emissao, Entrega: entrega})
	if err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}
	out := &response.PlanoDePagamentoResponse{
		CondicaoCode:      pc.Code,
		CondicaoDescricao: pc.Description,
		Total:             total.InexactFloat64(),
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
		out.Aviso = "há parcelas presas à entrega: a data mostrada é estimada e muda quando a entrega for confirmada"
	}
	return out, nil
}
