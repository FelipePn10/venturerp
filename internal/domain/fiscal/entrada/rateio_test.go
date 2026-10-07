package entrada

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }
func p64(v int64) *int64         { return &v }

const (
	planoMP  = int64(10)
	planoEPI = int64(20)
)

// O exemplo do pedido: nota de 100 mil, 50 mil de matéria-prima e 50 mil de
// EPI, em três parcelas.
func notaExemplo() ([]*entity.FiscalEntryItem, []*entity.FiscalEntryInstallment) {
	code := int64(1)
	itens := []*entity.FiscalEntryItem{
		{Sequence: 1, ItemCode: &code, PlanoContasID: p64(planoMP), ValorContabil: d("50000")},
		{Sequence: 2, ItemCode: &code, PlanoContasID: p64(planoEPI), ValorContabil: d("50000")},
	}
	hoje := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	parcelas := []*entity.FiscalEntryInstallment{
		{Numero: 1, Valor: d("33333.33"), DataVencimento: hoje.AddDate(0, 0, 30)},
		{Numero: 2, Valor: d("33333.33"), DataVencimento: hoje.AddDate(0, 0, 60)},
		{Numero: 3, Valor: d("33333.34"), DataVencimento: hoje.AddDate(0, 0, 90)},
	}
	return itens, parcelas
}

func TestDistribuirProporcional_FechaLinhasEColunas(t *testing.T) {
	itens, parcelas := notaExemplo()
	if err := DistribuirProporcional(parcelas, TotalPorConta(itens)); err != nil {
		t.Fatal(err)
	}
	pend := Conferir(d("100000"), itens, parcelas, false)
	if TemImpedimento(pend) {
		t.Fatalf("rateio proporcional deveria fechar, pendências: %+v", pend)
	}
	// Cada parcela meio a meio.
	for _, p := range parcelas {
		if len(p.Distribuicao) != 2 {
			t.Fatalf("parcela %d: esperava 2 contas, veio %d", p.Numero, len(p.Distribuicao))
		}
		dif := p.Distribuicao[0].Valor.Sub(p.Distribuicao[1].Valor).Abs()
		if dif.GreaterThan(d("0.02")) {
			t.Fatalf("parcela %d não ficou meio a meio: %v", p.Numero, p.Distribuicao)
		}
	}
}

func TestConferir_DistribuicaoManualPorParcela(t *testing.T) {
	itens, parcelas := notaExemplo()
	// Primeira parcela inteira em EPI, a segunda inteira em MP e a terceira
	// completando cada conta.
	parcelas[0].Distribuicao = []entity.InstallmentAllocation{{PlanoContasID: planoEPI, Valor: d("33333.33")}}
	parcelas[1].Distribuicao = []entity.InstallmentAllocation{{PlanoContasID: planoMP, Valor: d("33333.33")}}
	parcelas[2].Distribuicao = []entity.InstallmentAllocation{
		{PlanoContasID: planoMP, Valor: d("16666.67")},
		{PlanoContasID: planoEPI, Valor: d("16666.67")},
	}
	if pend := Conferir(d("100000"), itens, parcelas, false); TemImpedimento(pend) {
		t.Fatalf("distribuição manual válida foi recusada: %+v", pend)
	}
}

func TestConferir_ContaComTotalErradoImpede(t *testing.T) {
	itens, parcelas := notaExemplo()
	for _, p := range parcelas {
		p.Distribuicao = []entity.InstallmentAllocation{{PlanoContasID: planoMP, Valor: p.Valor}}
	}
	pend := Conferir(d("100000"), itens, parcelas, false)
	if !TemImpedimento(pend) {
		t.Fatal("tudo em MP não fecha com 50/50 dos itens e deveria impedir")
	}
}

func TestConferir_ParcelaComDistribuicaoIncompletaImpede(t *testing.T) {
	itens, parcelas := notaExemplo()
	_ = DistribuirProporcional(parcelas, TotalPorConta(itens))
	parcelas[0].Distribuicao[0].Valor = parcelas[0].Distribuicao[0].Valor.Sub(d("100"))
	if !TemImpedimento(Conferir(d("100000"), itens, parcelas, false)) {
		t.Fatal("parcela que não soma o seu valor deveria impedir")
	}
}

func TestConferir_ItemSemConciliacaoOuSemPlanoImpede(t *testing.T) {
	itens, parcelas := notaExemplo()
	_ = DistribuirProporcional(parcelas, TotalPorConta(itens))
	itens[0].ItemCode = nil
	pend := Conferir(d("100000"), itens, parcelas, false)
	if !TemImpedimento(pend) {
		t.Fatal("item sem conciliação deveria impedir a aprovação")
	}
	itens[0].ItemCode = p64(1)
	itens[1].PlanoContasID = nil
	if !TemImpedimento(Conferir(d("100000"), itens, parcelas, false)) {
		t.Fatal("item sem plano de contas deveria impedir a aprovação")
	}
}

func TestConferir_ParcelasQueNaoFechamComONota(t *testing.T) {
	itens, parcelas := notaExemplo()
	parcelas[2].Valor = d("30000")
	_ = DistribuirProporcional(parcelas, TotalPorConta(itens))
	if !TemImpedimento(Conferir(d("100000"), itens, parcelas, false)) {
		t.Fatal("parcelas que não somam o total da nota deveriam impedir")
	}
}

func TestConferir_SemPagamentoNaoExigeParcela(t *testing.T) {
	itens, _ := notaExemplo()
	if TemImpedimento(Conferir(d("100000"), itens, nil, true)) {
		t.Fatal("nota sem pagamento (tPag 90) não deve exigir parcela")
	}
	if !TemImpedimento(Conferir(d("100000"), itens, nil, false)) {
		t.Fatal("nota com pagamento e sem parcela deveria impedir")
	}
}

func TestAjustarValorContabil_FechaComTotalDaNota(t *testing.T) {
	itens := []*entity.FiscalEntryItem{
		{ValorContabil: d("10.004")},
		{ValorContabil: d("20.004")},
	}
	AjustarValorContabil(itens, d("30.02"))
	soma := itens[0].ValorContabil.Add(itens[1].ValorContabil)
	if !soma.Equal(d("30.02")) {
		t.Fatalf("soma %s != 30.02", soma)
	}
}

func TestProporcaoPaga(t *testing.T) {
	if v := ProporcaoPaga(d("600"), d("500"), d("1000")); !v.Equal(d("300")) {
		t.Fatalf("esperava 300, veio %s", v)
	}
	if v := ProporcaoPaga(d("600"), d("1000"), d("1000")); !v.Equal(d("600")) {
		t.Fatalf("pago integral deveria devolver o rateio inteiro, veio %s", v)
	}
}

// Com só parte dos itens classificada, a parcela não pode ser empurrada
// inteira para os planos que já existem: cada plano recebe a sua proporção e o
// restante fica sem distribuição (pendência), sem valor negativo nem parcela
// zerada.
func TestDistribuirProporcional_ClassificacaoParcial(t *testing.T) {
	itens, parcelas := notaExemplo()
	itens[0].PlanoContasID = nil // a matéria-prima ainda sem plano
	if err := DistribuirProporcional(parcelas, TotalPorConta(itens)); err != nil {
		t.Fatal(err)
	}
	soma := decimal.Zero
	for _, p := range parcelas {
		if len(p.Distribuicao) != 1 {
			t.Fatalf("parcela %d: %+v", p.Numero, p.Distribuicao)
		}
		v := p.Distribuicao[0].Valor
		if v.IsNegative() || v.IsZero() {
			t.Fatalf("parcela %d recebeu %s", p.Numero, v)
		}
		if v.Sub(p.Valor.Div(decimal.NewFromInt(2))).Abs().GreaterThan(d("0.02")) {
			t.Fatalf("parcela %d deveria ter metade em EPI, tem %s de %s", p.Numero, v, p.Valor)
		}
		soma = soma.Add(v)
	}
	if !soma.Equal(d("50000")) {
		t.Fatalf("EPI recebe %s nas parcelas, esperado 50000", soma)
	}
}

func TestMoeda(t *testing.T) {
	casos := map[string]string{"66666.66": "R$ 66.666,66", "0.5": "R$ 0,50", "1234567.891": "R$ 1.234.567,89", "-10": "R$ -10,00"}
	for in, want := range casos {
		if got := Moeda(d(in)); got != want {
			t.Errorf("Moeda(%s) = %q, esperado %q", in, got, want)
		}
	}
}
