package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func pct(v float64) *float64 { return &v }

func parcela(n int16, dias int16, evento string, percentual *float64) *PaymentInstallment {
	return &PaymentInstallment{InstallmentNumber: n, DueDays: dias, BaseEvent: evento, Percentage: percentual, IsActive: true}
}

// A condição que a indústria mais usa e que não cabia no modelo anterior.
func TestPlano30EntradaVinteNaEntregaRestanteEm28e56(t *testing.T) {
	emissao := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	entrega := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	cond := &PaymentCondition{Installments: []*PaymentInstallment{
		parcela(1, 0, "ENTRADA", pct(30)),
		parcela(2, 0, "ENTREGA", pct(20)),
		parcela(3, 28, "EMISSAO", pct(25)),
		parcela(4, 56, "EMISSAO", pct(25)),
	}}

	plano, err := CalcularPlano(cond, decimal.NewFromInt(10000), DatasBase{Emissao: emissao, Entrega: &entrega})
	if err != nil {
		t.Fatal(err)
	}
	if len(plano) != 4 {
		t.Fatalf("esperava 4 parcelas, veio %d", len(plano))
	}
	esperado := []struct {
		valor string
		data  time.Time
	}{
		{"3000", emissao},
		{"2000", entrega},
		{"2500", emissao.AddDate(0, 0, 28)},
		{"2500", emissao.AddDate(0, 0, 56)},
	}
	for i, e := range esperado {
		if !plano[i].Valor.Equal(decimal.RequireFromString(e.valor)) {
			t.Fatalf("parcela %d: valor %s, esperado %s", i+1, plano[i].Valor, e.valor)
		}
		if !plano[i].Vencimento.Equal(e.data) {
			t.Fatalf("parcela %d: vence %s, esperado %s", i+1, plano[i].Vencimento, e.data)
		}
	}
	// A entrada vence no ato, não em "0 dias da entrega".
	if plano[0].Evento != BaseEntrada || plano[0].DiasPrazo != 0 {
		t.Fatalf("a entrada tem de vencer no ato: %+v", plano[0])
	}
}

// Centavo não pode sumir no arredondamento: a última parcela absorve a diferença.
func TestPlanoNaoPerdeCentavoNoArredondamento(t *testing.T) {
	cond := &PaymentCondition{Installments: []*PaymentInstallment{
		parcela(1, 0, "ENTRADA", pct(30)),
		parcela(2, 30, "EMISSAO", pct(30)),
		parcela(3, 60, "EMISSAO", pct(40)),
	}}
	total := decimal.RequireFromString("1000.01")
	plano, err := CalcularPlano(cond, total, DatasBase{Emissao: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	soma := decimal.Zero
	for _, p := range plano {
		soma = soma.Add(p.Valor)
	}
	if !soma.Equal(total) {
		t.Fatalf("o plano soma %s e o pedido é %s", soma, total)
	}
}

// Sem percentual em lugar nenhum, a condição continua significando o que sempre
// significou: partes iguais.
func TestPlanoSemPercentualDivideEmPartesIguais(t *testing.T) {
	cond := &PaymentCondition{Installments: []*PaymentInstallment{
		parcela(1, 30, "EMISSAO", nil),
		parcela(2, 60, "EMISSAO", nil),
	}}
	plano, err := CalcularPlano(cond, decimal.NewFromInt(900), DatasBase{Emissao: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range plano {
		if !p.Valor.Equal(decimal.NewFromInt(450)) {
			t.Fatalf("esperava 450 por parcela, veio %s", p.Valor)
		}
	}
}

// Percentual pela metade é pior que nenhum: não há como adivinhar o resto.
func TestPlanoRecusaPercentualIncompletoOuQueNaoFecha(t *testing.T) {
	meio := &PaymentCondition{Installments: []*PaymentInstallment{
		parcela(1, 0, "ENTRADA", pct(30)),
		parcela(2, 30, "EMISSAO", nil),
	}}
	if _, err := CalcularPlano(meio, decimal.NewFromInt(100), DatasBase{Emissao: time.Now()}); err == nil {
		t.Fatal("percentual em parte das parcelas deveria ser recusado")
	}
	naoFecha := &PaymentCondition{Installments: []*PaymentInstallment{
		parcela(1, 0, "ENTRADA", pct(30)),
		parcela(2, 30, "EMISSAO", pct(50)),
	}}
	_, err := CalcularPlano(naoFecha, decimal.NewFromInt(100), DatasBase{Emissao: time.Now()})
	if err == nil {
		t.Fatal("percentuais que somam 80% deveriam ser recusados")
	}
}

// Sem data de entrega o vencimento é estimado pela emissão — e a tela precisa
// saber disso para não apresentar palpite como compromisso.
func TestPlanoMarcaVencimentoEstimadoSemDataDeEntrega(t *testing.T) {
	cond := &PaymentCondition{Installments: []*PaymentInstallment{parcela(1, 10, "ENTREGA", pct(100))}}
	plano, err := CalcularPlano(cond, decimal.NewFromInt(500), DatasBase{Emissao: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if !plano[0].Estimado {
		t.Fatal("vencimento contado da entrega sem data de entrega é estimativa")
	}
}

// Condição sem parcela nenhuma é à vista — não é erro nem plano vazio.
func TestPlanoSemParcelasEhAVista(t *testing.T) {
	plano, err := CalcularPlano(&PaymentCondition{}, decimal.NewFromInt(250), DatasBase{Emissao: time.Now()})
	if err != nil || len(plano) != 1 || !plano[0].Valor.Equal(decimal.NewFromInt(250)) {
		t.Fatalf("esperava uma parcela à vista: %+v %v", plano, err)
	}
}
