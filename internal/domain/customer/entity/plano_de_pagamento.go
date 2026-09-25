package entity

import (
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

// PaymentBaseEvent diz a partir de QUANDO os dias de uma parcela contam.
type PaymentBaseEvent string

const (
	// BaseEmissao conta da emissão do documento — o padrão.
	BaseEmissao PaymentBaseEvent = "EMISSAO"
	// BaseEntrada é o pagamento no ato: não conta dias, vence na própria emissão.
	BaseEntrada PaymentBaseEvent = "ENTRADA"
	// BaseEntrega conta da entrega da mercadoria.
	BaseEntrega PaymentBaseEvent = "ENTREGA"
	// BaseFaturamento conta do faturamento.
	BaseFaturamento PaymentBaseEvent = "FATURAMENTO"
)

func (e PaymentBaseEvent) IsValid() bool {
	switch e {
	case BaseEmissao, BaseEntrada, BaseEntrega, BaseFaturamento:
		return true
	}
	return false
}

// Rotulo devolve o evento em português, do jeito que a tela mostra.
func (e PaymentBaseEvent) Rotulo() string {
	switch e {
	case BaseEntrada:
		return "entrada (no ato)"
	case BaseEntrega:
		return "entrega"
	case BaseFaturamento:
		return "faturamento"
	default:
		return "emissão"
	}
}

// DatasBase são os marcos que os eventos de vencimento usam. Entrega e
// faturamento caem na emissão quando ainda não se sabe a data deles — é o
// melhor palpite possível e a tela diz que é estimativa.
type DatasBase struct {
	Emissao     time.Time
	Entrega     *time.Time
	Faturamento *time.Time
}

func (d DatasBase) para(evento PaymentBaseEvent) time.Time {
	switch evento {
	case BaseEntrega:
		if d.Entrega != nil {
			return *d.Entrega
		}
	case BaseFaturamento:
		if d.Faturamento != nil {
			return *d.Faturamento
		}
	}
	return d.Emissao
}

// ParcelaCalculada é uma linha do plano de pagamento já resolvida em dinheiro e
// data — o que o cliente lê na proposta e o que vira título no financeiro.
type ParcelaCalculada struct {
	Numero       int16
	Percentual   decimal.Decimal
	Valor        decimal.Decimal
	Vencimento   time.Time
	DiasPrazo    int16
	Evento       PaymentBaseEvent
	Descricao    string
	Estimado     bool
	DocumentType *string
}

// CalcularPlano transforma a condição de pagamento num plano concreto para um
// valor total.
//
// Duas regras cuidam do dinheiro:
//
//   - Sem percentual em nenhuma parcela, o total é dividido em partes iguais —
//     é o que a condição significava antes de existir percentual, e nenhuma
//     condição já cadastrada muda de comportamento.
//   - O arredondamento não pode sumir nem inventar centavo: as parcelas são
//     arredondadas a duas casas e a DIFERENÇA vai inteira para a última. Somar
//     30%, 20% e 50% de R$ 1.000,01 dá R$ 1.000,00 se cada uma for arredondada
//     isoladamente, e um centavo perdido por pedido vira divergência no
//     fechamento do mês.
func CalcularPlano(cond *PaymentCondition, total decimal.Decimal, datas DatasBase) ([]ParcelaCalculada, error) {
	if cond == nil {
		return nil, fmt.Errorf("informe a condição de pagamento")
	}
	ativas := make([]*PaymentInstallment, 0, len(cond.Installments))
	for _, p := range cond.Installments {
		if p != nil && p.IsActive {
			ativas = append(ativas, p)
		}
	}
	// Condição sem parcelas cadastradas é à vista: uma parcela na emissão.
	if len(ativas) == 0 {
		return []ParcelaCalculada{{
			Numero: 1, Percentual: decimal.NewFromInt(100), Valor: total.Round(2),
			Vencimento: datas.Emissao, Evento: BaseEmissao, Descricao: "à vista",
		}}, nil
	}
	sort.Slice(ativas, func(i, j int) bool { return ativas[i].InstallmentNumber < ativas[j].InstallmentNumber })

	if err := ValidarPercentuais(ativas); err != nil {
		return nil, err
	}

	cem := decimal.NewFromInt(100)
	usaPercentual := false
	for _, p := range ativas {
		if p.Percentage != nil {
			usaPercentual = true
			break
		}
	}

	out := make([]ParcelaCalculada, 0, len(ativas))
	somado := decimal.Zero
	for i, p := range ativas {
		pct := decimal.Zero
		if usaPercentual {
			if p.Percentage != nil {
				pct = decimal.NewFromFloat(*p.Percentage)
			}
		} else {
			pct = cem.Div(decimal.NewFromInt(int64(len(ativas))))
		}
		valor := total.Mul(pct).Div(cem).Round(2)

		evento := PaymentBaseEvent(p.BaseEvent)
		if !evento.IsValid() {
			evento = BaseEmissao
		}
		dias := p.DueDays
		if evento == BaseEntrada {
			dias = 0
		}
		base := datas.para(evento)
		estimado := (evento == BaseEntrega && datas.Entrega == nil) ||
			(evento == BaseFaturamento && datas.Faturamento == nil)

		linha := ParcelaCalculada{
			Numero:       p.InstallmentNumber,
			Percentual:   pct.Round(4),
			Valor:        valor,
			Vencimento:   base.AddDate(0, 0, int(dias)),
			DiasPrazo:    dias,
			Evento:       evento,
			Estimado:     estimado,
			DocumentType: p.DocumentType,
		}
		if p.Description != nil {
			linha.Descricao = *p.Description
		}
		if linha.Descricao == "" {
			linha.Descricao = descricaoPadrao(evento, dias)
		}
		// A última absorve a diferença de arredondamento.
		if i == len(ativas)-1 {
			linha.Valor = total.Round(2).Sub(somado)
		}
		somado = somado.Add(linha.Valor)
		out = append(out, linha)
	}
	return out, nil
}

// ValidarPercentuais recusa a condição cujos percentuais não fecham 100.
//
// Meio informado é pior que nenhum: com 30% numa parcela e nada na outra, o
// sistema não tem como saber se a segunda leva 70% ou metade do total, e
// qualquer palpite vira dinheiro errado no título.
func ValidarPercentuais(parcelas []*PaymentInstallment) error {
	informadas, soma := 0, decimal.Zero
	for _, p := range parcelas {
		if p.Percentage == nil {
			continue
		}
		informadas++
		soma = soma.Add(decimal.NewFromFloat(*p.Percentage))
	}
	if informadas == 0 {
		return nil
	}
	if informadas != len(parcelas) {
		return fmt.Errorf("informe o percentual de todas as %d parcelas ou de nenhuma: com parte delas em branco não há como saber quanto cada uma leva", len(parcelas))
	}
	if !soma.Round(4).Equal(decimal.NewFromInt(100)) {
		return fmt.Errorf("os percentuais das parcelas somam %s%%; precisam somar 100%%", soma.Round(4).String())
	}
	return nil
}

func descricaoPadrao(evento PaymentBaseEvent, dias int16) string {
	if evento == BaseEntrada {
		return "entrada"
	}
	if dias == 0 {
		return "na " + evento.Rotulo()
	}
	return fmt.Sprintf("%d dias da %s", dias, evento.Rotulo())
}
