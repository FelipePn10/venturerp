package financial_uc

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
)

// LeitorDePrevisoes lê títulos e pedidos em aberto (repositório PG).
type LeitorDePrevisoes interface {
	PrevisoesDeCaixa(ctx context.Context, ate time.Time) ([]entity.PrevisaoDeCaixa, error)
}

// SemanaDoFluxo é uma semana (segunda a domingo) do fluxo projetado.
type SemanaDoFluxo struct {
	Inicio         string          `json:"week_start"`
	ReceberTitulos decimal.Decimal `json:"receivable_titles"`
	ReceberPedidos decimal.Decimal `json:"receivable_orders"`
	PagarTitulos   decimal.Decimal `json:"payable_titles"`
	PagarPedidos   decimal.Decimal `json:"payable_orders"`
	Saldo          decimal.Decimal `json:"net"`
	Acumulado      decimal.Decimal `json:"cumulative"`
}

// FluxoComPedidosUseCase projeta o caixa com os títulos E os pedidos de compra
// e venda em aberto (melhoria 10): o fluxo projetado só via títulos, e o pedido
// confirmado ainda sem nota não aparecia.
type FluxoComPedidosUseCase struct{ Repo LeitorDePrevisoes }

func inicioDaSemana(d time.Time) time.Time {
	d = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
	for d.Weekday() != time.Monday {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

func (uc *FluxoComPedidosUseCase) Projetar(ctx context.Context, dias int) ([]SemanaDoFluxo, error) {
	if dias <= 0 || dias > 366 {
		return nil, errorsuc.NewValidationError("informe de 1 a 366 dias")
	}
	hoje := time.Now()
	lista, err := uc.Repo.PrevisoesDeCaixa(ctx, hoje.AddDate(0, 0, dias))
	if err != nil {
		return nil, err
	}
	return SemanasDoFluxo(hoje, dias, lista), nil
}

// SemanasDoFluxo agrupa as previsões por semana, com o acumulado.
func SemanasDoFluxo(hoje time.Time, dias int, lista []entity.PrevisaoDeCaixa) []SemanaDoFluxo {
	primeira := inicioDaSemana(hoje)
	var semanas []SemanaDoFluxo
	idx := map[string]int{}
	for d := primeira; !d.After(hoje.AddDate(0, 0, dias)); d = d.AddDate(0, 0, 7) {
		idx[d.Format("2006-01-02")] = len(semanas)
		semanas = append(semanas, SemanaDoFluxo{Inicio: d.Format("2006-01-02")})
	}
	for _, p := range lista {
		i, ok := idx[inicioDaSemana(p.Data).Format("2006-01-02")]
		if !ok {
			continue
		}
		s := &semanas[i]
		switch p.Origem {
		case "RECEBER_TITULO":
			s.ReceberTitulos = s.ReceberTitulos.Add(p.Valor)
		case "RECEBER_PEDIDO":
			s.ReceberPedidos = s.ReceberPedidos.Add(p.Valor)
		case "PAGAR_TITULO":
			s.PagarTitulos = s.PagarTitulos.Add(p.Valor)
		case "PAGAR_PEDIDO":
			s.PagarPedidos = s.PagarPedidos.Add(p.Valor)
		}
	}
	acumulado := decimal.Zero
	for i := range semanas {
		s := &semanas[i]
		s.ReceberTitulos, s.ReceberPedidos, s.PagarTitulos, s.PagarPedidos = s.ReceberTitulos.Round(2), s.ReceberPedidos.Round(2), s.PagarTitulos.Round(2), s.PagarPedidos.Round(2)
		s.Saldo = s.ReceberTitulos.Add(s.ReceberPedidos).Sub(s.PagarTitulos).Sub(s.PagarPedidos)
		acumulado = acumulado.Add(s.Saldo)
		s.Acumulado = acumulado
	}
	return semanas
}
