package contabilizacao

import (
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }
func c(v int64) *int64           { return &v }

const (
	banco, fornecedores, clientes, jurosPag, descObt, jurosRec, descConc                      = 1, 2, 3, 4, 5, 6, 7
	receita, ipiRec, stRec, icmsVend, icmsRec, pisVend, pisRec, cofVend, cofRec, cmv, estoque = 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20
	despMP, despEPI                                                                           = 30, 31
)

func fecha(t *testing.T, ls []Lancamento) map[int64]decimal.Decimal {
	t.Helper()
	s := Saldo(ls)
	total := decimal.Zero
	for _, v := range s {
		total = total.Add(v)
	}
	if !total.IsZero() {
		t.Fatalf("débito ≠ crédito: diferença %s", total)
	}
	return s
}

// Pagamento de título de NF já contabilizada, com atraso e desconto: o banco
// sai pelo principal + juros; Fornecedores baixa principal + desconto.
func TestPagamentoComJurosEDesconto(t *testing.T) {
	ls, falta := Pagamento(Baixa{
		Banco: c(banco), Partes: []Parte{{Conta: c(fornecedores), Peso: d("1")}},
		Principal: d("980"), Juros: d("25.50"), Desconto: d("20"),
		ContaJuros: c(jurosPag), ContaDesconto: c(descObt), Historico: "Pagamento NF-1/1",
	})
	if len(falta) > 0 {
		t.Fatal(falta)
	}
	s := fecha(t, ls)
	if !s[fornecedores].Equal(d("1000")) || !s[banco].Equal(d("-1005.50")) || !s[jurosPag].Equal(d("25.50")) || !s[descObt].Equal(d("-20")) {
		t.Fatalf("saldos: %v", s)
	}
}

// Título avulso (sem nota contabilizada): o principal vai para a despesa de
// cada plano do rateio, na proporção; os centavos fecham na última parte.
func TestPagamentoAvulsoRateadoPelosPlanos(t *testing.T) {
	cc := int64(9)
	ls, falta := Pagamento(Baixa{
		Banco: c(banco), Principal: d("100"),
		Partes: []Parte{{Conta: c(despMP), Peso: d("1")}, {Conta: c(despEPI), CC: &cc, Peso: d("2")}},
	})
	if len(falta) > 0 {
		t.Fatal(falta)
	}
	s := fecha(t, ls)
	if !s[despMP].Equal(d("33.33")) || !s[despEPI].Equal(d("66.67")) {
		t.Fatalf("rateio: MP %s EPI %s", s[despMP], s[despEPI])
	}
	if ls[1].DebitoCC == nil || *ls[1].DebitoCC != 9 {
		t.Fatal("centro de custo do rateio perdido")
	}
}

func TestPagamentoSemContasListaOQueFalta(t *testing.T) {
	_, falta := Pagamento(Baixa{Principal: d("10"), Juros: d("1"), Partes: []Parte{{Nome: "o plano EPI", Peso: d("1")}}})
	if len(falta) != 3 {
		t.Fatalf("faltando: %v", falta)
	}
}

func TestRecebimentoComJurosEDesconto(t *testing.T) {
	ls, falta := Recebimento(Baixa{
		Banco: c(banco), Partes: []Parte{{Conta: c(clientes), Peso: d("1")}},
		Principal: d("490"), Juros: d("5"), Desconto: d("10"),
		ContaJuros: c(jurosRec), ContaDesconto: c(descConc),
	})
	if len(falta) > 0 {
		t.Fatal(falta)
	}
	s := fecha(t, ls)
	if !s[clientes].Equal(d("-500")) || !s[banco].Equal(d("495")) || !s[jurosRec].Equal(d("-5")) || !s[descConc].Equal(d("10")) {
		t.Fatalf("saldos: %v", s)
	}
}

func contasSaida() ContasSaida {
	return ContasSaida{
		Clientes: c(clientes), Receita: c(receita), IPIRecolher: c(ipiRec), ICMSSTRecolher: c(stRec),
		ICMSVendas: c(icmsVend), ICMSRecolher: c(icmsRec), PISVendas: c(pisVend), PISRecolher: c(pisRec),
		COFINSVendas: c(cofVend), COFINSRec: c(cofRec), CMV: c(cmv), Estoque: c(estoque),
	}
}

// Venda de R$ 1.150 (1.000 de produto + 100 de IPI + 50 de ST): Clientes deve
// 1.150, a receita é 1.000, IPI e ST são repasse; ICMS/PIS/COFINS são despesa
// contra o passivo; CMV pelo custo.
func TestNotaSaida(t *testing.T) {
	ls, falta := NotaSaida(Saida{
		Total: d("1150"), IPI: d("100"), ICMSST: d("50"), ICMS: d("120"), PIS: d("16.50"), COFINS: d("76"),
		CMV: d("610.40"), GeraReceita: true,
	}, contasSaida())
	if len(falta) > 0 {
		t.Fatal(falta)
	}
	s := fecha(t, ls)
	casos := map[int64]string{clientes: "1150", receita: "-1000", ipiRec: "-100", stRec: "-50", icmsVend: "120", icmsRec: "-120",
		pisVend: "16.5", pisRec: "-16.5", cofVend: "76", cofRec: "-76", cmv: "610.4", estoque: "-610.4"}
	for conta, v := range casos {
		if !s[conta].Equal(d(v)) {
			t.Fatalf("conta %d: %s, esperado %s", conta, s[conta], v)
		}
	}
}

// Remessa/bonificação: sem receita nem Clientes; o custo e os impostos ficam.
func TestNotaSaidaSemReceita(t *testing.T) {
	ls, _ := NotaSaida(Saida{Total: d("500"), ICMS: d("60"), CMV: d("300")}, contasSaida())
	s := fecha(t, ls)
	if !s[clientes].IsZero() || !s[receita].IsZero() || !s[cmv].Equal(d("300")) || !s[icmsVend].Equal(d("60")) {
		t.Fatalf("saldos: %v", s)
	}
}

func TestNotaSaidaContaFaltando(t *testing.T) {
	cs := contasSaida()
	cs.CMV = nil
	_, falta := NotaSaida(Saida{Total: d("10"), CMV: d("5"), GeraReceita: true}, cs)
	if len(falta) != 1 || falta[0] != "falta a conta de CMV" {
		t.Fatalf("faltando: %v", falta)
	}
}

func TestEstornoZeraOsSaldos(t *testing.T) {
	ls, _ := NotaSaida(Saida{Total: d("1150"), IPI: d("100"), ICMS: d("120"), CMV: d("600"), GeraReceita: true}, contasSaida())
	todos := append(ls, Estorno(ls, "Estorno")...)
	for conta, v := range Saldo(todos) {
		if !v.IsZero() {
			t.Fatalf("conta %d não zerou: %s", conta, v)
		}
	}
}

func TestContaRetencao(t *testing.T) {
	if *ContaRetencao("IRRF", c(1), c(2), c(3), c(4)) != 1 || *ContaRetencao("CSLL", c(1), c(2), c(3), c(4)) != 2 ||
		*ContaRetencao("INSS", c(1), c(2), c(3), c(4)) != 3 || *ContaRetencao("ISS", c(1), c(2), c(3), c(4)) != 4 ||
		ContaRetencao("XYZ", c(1), c(2), c(3), c(4)) != nil {
		t.Fatal("conta de retenção errada")
	}
}

// Devolução de 5.000 (custo médio 3.800 + créditos 1.050) e 150 de diferença.
func TestNotaDevolucao(t *testing.T) {
	icmsRec, ipiRec, estoqueMP, variacao := int64(40), int64(41), int64(42), int64(43)
	ls, falta := NotaDevolucao(Devolucao{
		Total: d("5000"), Fornecedores: c(fornecedores), Variacao: &variacao,
		Linhas:   []LinhaDevolucao{{Conta: &estoqueMP, Custo: d("3800"), Nome: "chapa"}},
		Creditos: []CreditoEstornado{{Conta: &icmsRec, Valor: d("900"), Nome: "ICMS"}, {Conta: &ipiRec, Valor: d("150"), Nome: "IPI"}, {Valor: d("10"), Nome: "PIS"}},
	})
	if len(falta) > 0 {
		t.Fatal(falta)
	}
	s := fecha(t, ls)
	if !s[fornecedores].Equal(d("5000")) || !s[estoqueMP].Equal(d("-3800")) || !s[icmsRec].Equal(d("-900")) || !s[variacao].Equal(d("-150")) {
		t.Fatalf("saldos: %v", s)
	}
	// Custo médio acima do da compra: a diferença é despesa.
	ls, _ = NotaDevolucao(Devolucao{Total: d("100"), Fornecedores: c(fornecedores), Variacao: &variacao,
		Linhas: []LinhaDevolucao{{Conta: &estoqueMP, Custo: d("120")}}})
	if s := fecha(t, ls); !s[variacao].Equal(d("20")) || !s[fornecedores].Equal(d("100")) {
		t.Fatalf("diferença negativa: %v", s)
	}
	if _, falta := NotaDevolucao(Devolucao{Total: d("10"), Linhas: []LinhaDevolucao{{Custo: d("10"), Nome: "x"}}}); len(falta) != 3 {
		t.Fatalf("faltando: %v", falta)
	}
}
