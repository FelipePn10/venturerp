package devolucao

import (
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestCFOP(t *testing.T) {
	casos := []struct {
		entrada        string
		mesmaUF, exter bool
		want           string
	}{
		{"1101", true, false, "5201"}, {"2102", false, false, "6202"}, {"1403", true, false, "5411"},
		{"2401", false, false, "6410"}, {"1556", true, false, "5556"}, {"2551", false, false, "6553"},
		{"3102", false, true, "7202"}, {"1910", true, false, "5949"}, {"", true, false, "5949"},
	}
	for _, c := range casos {
		if got := CFOP(c.entrada, c.mesmaUF, c.exter); got != c.want {
			t.Errorf("CFOP(%s) = %s, esperado %s", c.entrada, got, c.want)
		}
	}
}

func origem() Origem {
	return Origem{
		Descricao: "chapa", Quantidade: d("10000"), ValorUnitario: d("4.80"), Total: d("48000"), Frete: d("500"),
		BaseICMS: d("48500"), ICMS: d("5820"), BaseIPI: d("48000"), IPI: d("1500"), PIS: d("792"), COFINS: d("3648"),
		CustoAquisicao: d("38240"), CreditoICMS: d("5820"), CreditoIPI: d("1500"), CreditoPIS: d("792"), CreditoCOFI: d("3648"),
	}
}

// Devolver 1.000 kg (10%): todos os valores a 10%; o fornecedor deve 5.000
// (4.800 + 50 de frete + 150 de IPI), que é o custo devolvido + os créditos estornados.
func TestMontarProporcional(t *testing.T) {
	l, err := Montar(origem(), d("1000"))
	if err != nil {
		t.Fatal(err)
	}
	if !l.Total.Equal(d("4800")) || !l.ICMS.Equal(d("582")) || !l.IPI.Equal(d("150")) || !l.Frete.Equal(d("50")) {
		t.Fatalf("linha: %+v", l)
	}
	if !l.ValorContabil().Equal(d("5000")) {
		t.Fatalf("valor contábil %s", l.ValorContabil())
	}
	creditos := l.CreditoICMS.Add(l.CreditoIPI).Add(l.CreditoPIS).Add(l.CreditoCOFI)
	if !l.CustoAquisicao.Add(creditos).Equal(l.ValorContabil()) {
		t.Fatalf("custo %s + créditos %s ≠ valor %s", l.CustoAquisicao, creditos, l.ValorContabil())
	}
}

func TestMontarRespeitaOSaldo(t *testing.T) {
	o := origem()
	o.Devolvida = d("9500")
	if _, err := Montar(o, d("600")); err == nil {
		t.Fatal("devolver mais do que resta deveria ser recusado")
	}
	if _, err := Montar(o, d("0")); err == nil {
		t.Fatal("quantidade zero")
	}
	l, err := Montar(o, d("500"))
	if err != nil || !l.Quantidade.Equal(d("500")) {
		t.Fatalf("resto: %v %+v", err, l)
	}
}

func TestCSTICMSDaDevolucao(t *testing.T) {
	for _, c := range []struct {
		cst     string
		simples bool
		want    string
	}{
		{"00", false, "00"}, {"20", false, "20"}, {"102", false, "90"}, {"101", false, "90"}, {"900", false, "90"},
		{"", false, "90"}, {"060", false, "60"}, {"00", true, "900"}, {"102", true, "900"},
	} {
		if got := CSTICMS(c.cst, c.simples); got != c.want {
			t.Errorf("CSTICMS(%q, simples=%v) = %s, want %s", c.cst, c.simples, got, c.want)
		}
	}
}
