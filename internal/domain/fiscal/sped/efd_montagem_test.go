package sped

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func campo(linha string, i int) string {
	p := strings.Split(strings.TrimRight(linha, "\r"), "|")
	if i >= len(p) {
		return ""
	}
	return p[i]
}

func todasLinhas(out, reg string) []string {
	var r []string
	for _, ln := range linhas(out) {
		if strings.HasPrefix(ln, "|"+reg+"|") {
			r = append(r, ln)
		}
	}
	return r
}

func dadosPeriodo() DadosEFD {
	d := func(day int) time.Time { return time.Date(2026, time.September, day, 0, 0, 0, 0, time.UTC) }
	forn := Participante{Cod: "F7", Nome: "Aço Sul", CNPJ: "11222333000181", IE: "1234567", CodMun: "4106902", End: "Rua A", Num: "10"}
	cli := Participante{Cod: "C9", Nome: "Cliente B", CNPJ: "44555666000190", CodMun: "4314902"}
	chapa := ItemCadastro{Cod: "1001", Desc: "Chapa aço", UnidInv: "KG", Tipo: "01", NCM: "72085100"}
	return DadosEFD{
		Empresa: EFDEmpresa{CNPJ: "52454668000102", Nome: "Tecnofer", UF: "PR", IE: "9012345678", CodigoMunicipio: "4106902",
			RegimeTributario: "A", ContabilistaNome: "Contador", ContabilistaCPF: "11122233344"},
		Periodo:         EFDPeriodo{DataInicial: d(1), DataFinal: d(30), IndicadorSituacaoEspecial: "0"},
		ContribuinteIPI: true,
		CodReceitaICMS:  "1015",
		VencimentoICMS:  time.Date(2026, time.October, 12, 0, 0, 0, 0, time.UTC),
		Entradas: []NotaEntrada{{
			Part: forn, Serie: "1", Numero: "77102", Chave: strings.Repeat("4", 44), Emissao: d(2), Entrada: d(3),
			Total: dec("1310.00"), Produtos: dec("1200.00"), Frete: dec("50.00"), IPI: dec("60.00"),
			Itens: []ItemEntrada{
				// Em TN (unidade do fornecedor): 0220 TN→KG fator 1000.
				{Item: chapa, Seq: 1, Unid: "tn", Qtd: dec("1"), Fator: dec("1000"), Total: dec("1000.00"),
					ValorContabil: dec("1091.67"), CST: "000", CFOP: "1101", BaseICMS: dec("1041.67"), AliqICMS: dec("12"),
					ICMS: dec("125.00"), CreditaICMS: true, BaseIPI: dec("1000"), AliqIPI: dec("5"), IPI: dec("50.00"), CreditaIPI: true,
					MovEstoque: true},
				// Material de uso e consumo: sem crédito de ICMS.
				{Item: ItemCadastro{Cod: "2002", Desc: "Luva", UnidInv: "PC", Tipo: "07"}, Seq: 2, Unid: "PC", Qtd: dec("10"),
					Total: dec("200.00"), ValorContabil: dec("218.33"), CST: "000", CFOP: "1556", BaseICMS: dec("208.33"),
					AliqICMS: dec("12"), ICMS: dec("25.00"), BaseIPI: dec("200"), AliqIPI: dec("5"), IPI: dec("10.00"), MovEstoque: true},
			},
		}},
		Saidas: []NotaSaida{
			{Part: cli, Serie: "1", Numero: "501", Chave: strings.Repeat("5", 44), Emissao: d(10), Saida: d(10),
				Total: dec("2010.00"), Produtos: dec("2000.00"), Frete: dec("30.00"), Desconto: dec("20.00"),
				Itens: []ItemSaida{
					{CST: "000", CFOP: "6101", AliqICMS: dec("12"), BaseICMS: dec("1000.00"), ICMS: dec("120.00"), Total: dec("1000.00")},
					{CST: "000", CFOP: "6101", AliqICMS: dec("12"), BaseICMS: dec("700.00"), ICMS: dec("84.00"), Total: dec("700.00")},
					{CST: "040", CFOP: "6102", Total: dec("300.00")},
				}},
			{Serie: "1", Numero: "502", Chave: strings.Repeat("6", 44), Cancelada: true, Total: dec("99")},
		},
		Fretes: []Frete{{Part: Participante{Cod: "T3", Nome: "Transp", CNPJ: "77888999000100", CodMun: "4106902"}, Serie: "1",
			Numero: "900", Chave: strings.Repeat("7", 44), CST: "000", CFOP: "1353", Emissao: d(3), Lancamento: d(4),
			Valor: dec("50.00"), Base: dec("50.00"), Aliq: dec("12"), ICMS: dec("6.00"), CreditaICMS: true,
			MunOrig: "4106902", MunDest: "4106902"}},
		SaldoCredorAnteriorICMS: dec("10.00"),
		Itens:                   []ItemCadastro{chapa, {Cod: "3003", Desc: "Suporte", UnidInv: "UN", Tipo: "04"}},
	}
}

func TestMontar_ArquivoValidoEApuracao(t *testing.T) {
	out := Generate(Montar(dadosPeriodo()))
	validar(t, out)

	// C100: 1 entrada, 1 saída regular, 1 cancelada.
	c100 := todasLinhas(out, "C100")
	if len(c100) != 3 {
		t.Fatalf("C100 = %d, want 3", len(c100))
	}
	// Entrada só credita o ICMS do item com crédito (125,00), não o da luva.
	if campo(c100[0], 22) != "125,00" {
		t.Errorf("VL_ICMS da entrada = %q, want 125,00 (%s)", campo(c100[0], 22), c100[0])
	}
	// A saída não leva C170; só os 2 itens da entrada.
	if n := len(todasLinhas(out, "C170")); n != 2 {
		t.Errorf("C170 = %d, want 2", n)
	}
	if campo(c100[2], 6) != "02" {
		t.Errorf("COD_SIT da cancelada = %q", campo(c100[2], 6))
	}

	// C190 da saída fecha no VL_DOC (frete e desconto rateados).
	soma := decimal.Zero
	for _, l := range todasLinhas(out, "C190") {
		if strings.HasPrefix(campo(l, 3), "6") {
			soma = soma.Add(dec(strings.ReplaceAll(campo(l, 5), ",", ".")))
		}
	}
	if !soma.Equal(dec("2010")) {
		t.Errorf("Σ VL_OPR da saída = %s, want 2010", soma)
	}

	// E110: débitos 204; créditos 125 (NF-e) + 6 (CT-e) = 131; saldo anterior 10 → 63 a recolher.
	e110 := todasLinhas(out, "E110")[0]
	want := map[int]string{2: "204,00", 6: "131,00", 10: "10,00", 11: "63,00", 13: "63,00", 14: "0,00"}
	for i, w := range want {
		if campo(e110, i) != w {
			t.Errorf("E110 campo %d = %q, want %q (%s)", i, campo(e110, i), w, e110)
		}
	}
	e116 := todasLinhas(out, "E116")
	if len(e116) != 1 || campo(e116[0], 3) != "63,00" || campo(e116[0], 5) != "1015" || campo(e116[0], 10) != "092026" {
		t.Errorf("E116 = %v", e116)
	}

	// 0220 TN→KG fator 1000; unidades da nota e do estoque no 0190.
	if l := todasLinhas(out, "0220"); len(l) != 1 || campo(l[0], 2) != "TN" || campo(l[0], 3) != "1000,000000" {
		t.Errorf("0220 = %v", l)
	}
	for _, u := range []string{"KG", "TN", "PC", "UN"} {
		ok := false
		for _, l := range todasLinhas(out, "0190") {
			if campo(l, 2) == u {
				ok = true
			}
		}
		if !ok {
			t.Errorf("0190 sem %s", u)
		}
	}
	// Participantes referenciados: fornecedor, cliente e transportadora.
	if n := len(todasLinhas(out, "0150")); n != 3 {
		t.Errorf("0150 = %d, want 3", n)
	}
	if n := len(todasLinhas(out, "D100")); n != 1 {
		t.Errorf("D100 = %d", n)
	}

	// IPI: crédito só do item com crédito (50), sem débito nas saídas.
	e520 := todasLinhas(out, "E520")
	if len(e520) != 1 || campo(e520[0], 3) != "0,00" || campo(e520[0], 4) != "50,00" {
		t.Errorf("E520 = %v", e520)
	}
}

func TestMontar_SaldoCredorTransportado(t *testing.T) {
	d := dadosPeriodo()
	d.Saidas = nil
	out := Generate(Montar(d))
	validar(t, out)
	e110 := todasLinhas(out, "E110")[0]
	// Créditos 131 + saldo anterior 10, sem débitos → 141 transportados.
	if campo(e110, 13) != "0,00" || campo(e110, 14) != "141,00" {
		t.Errorf("E110 = %s", e110)
	}
	if n := len(todasLinhas(out, "E116")); n != 0 {
		t.Errorf("E116 sem ICMS a recolher = %d", n)
	}
}

func TestMontar_SemContribuinteIPI(t *testing.T) {
	d := dadosPeriodo()
	d.ContribuinteIPI = false
	p := Montar(d)
	if p.ApuracaoIPI != nil {
		t.Fatal("ApuracaoIPI deveria ser nil")
	}
	validar(t, Generate(p))
}

func TestMontar_PeriodoVazio(t *testing.T) {
	d := dadosPeriodo()
	d.Entradas, d.Saidas, d.Fretes, d.Itens, d.SaldoCredorAnteriorICMS = nil, nil, nil, nil, decimal.Zero
	out := Generate(Montar(d))
	validar(t, out)
	if n := len(todasLinhas(out, "C100")); n != 0 {
		t.Errorf("C100 = %d", n)
	}
}

func TestTipoItemCFOPECST(t *testing.T) {
	fab, comp, serv, ind, cons, imob := 0, 1, 3, 0, 1, 2
	casos := []struct {
		c    CadastroDoItem
		cfop string
		want string
	}{
		{CadastroDoItem{}, "1102", "00"},
		{CadastroDoItem{TipoEngenharia: &comp, TipoUso: &ind}, "2556", "07"},
		{CadastroDoItem{TipoEngenharia: &fab}, "1551", "08"},
		{CadastroDoItem{}, "1353", "09"},
		{CadastroDoItem{TipoEngenharia: &fab, TipoUso: &ind}, "1101", "04"},
		{CadastroDoItem{TipoEngenharia: &comp, TipoUso: &ind}, "1101", "01"},
		{CadastroDoItem{}, "1949", "99"},
		// O cadastro manda: consumo/imobilizado declarados vencem o CFOP.
		{CadastroDoItem{TipoEngenharia: &comp, TipoUso: &cons}, "1101", "07"},
		{CadastroDoItem{TipoEngenharia: &comp, TipoUso: &imob}, "1101", "08"},
		{CadastroDoItem{TipoEngenharia: &comp, TipoUso: &ind, Revenda: true}, "", "00"},
		{CadastroDoItem{TipoEngenharia: &comp, Embalagem: true}, "1101", "02"},
		{CadastroDoItem{TipoEngenharia: &serv}, "", "09"},
	}
	for _, c := range casos {
		if got := TipoItem(c.c, c.cfop); got != c.want {
			t.Errorf("TipoItem(%+v,%s) = %s, want %s", c.c, c.cfop, got, c.want)
		}
	}
	for in, want := range map[string]string{"5102": "1102", "6101": "2101", "7101": "3101", "1556": "1556", "": ""} {
		if got := CFOPEntrada(in); got != want {
			t.Errorf("CFOPEntrada(%s) = %s", in, got)
		}
	}
	for _, c := range [][3]string{{"0", "00", "000"}, {"2", "60", "260"}, {"0", "101", "090"}, {"1", "500", "190"}, {"", "", "090"}, {"0", "040", "040"}, {"5", "500", "500"}, {"0", "900", "090"}} {
		if got := CSTICMS(c[0], c[1]); got != c[2] {
			t.Errorf("CSTICMS(%s,%s) = %s, want %s", c[0], c[1], got, c[2])
		}
	}
}

func TestMontar_AjustesDaApuracao(t *testing.T) {
	d := dadosPeriodo()
	// Débitos 204, créditos 131, saldo anterior 10 (cenário base: 63 a recolher).
	d.AjustesApuracao = []AjusteApuracao{
		{Codigo: "PR000001", Descricao: "outro débito", Valor: dec("7")},
		{Codigo: "PR010001", Descricao: "estorno de crédito", Valor: dec("3")},
		{Codigo: "PR020001", Descricao: "crédito presumido", Valor: dec("20")},
		{Codigo: "PR030001", Descricao: "estorno de débito", Valor: dec("1")},
		{Codigo: "PR040001", Descricao: "dedução", Valor: dec("2")},
		{Codigo: "PR050001", Descricao: "débito especial", Valor: dec("4")},
		{Codigo: "PR100001", Descricao: "ajuste de ST: fica fora", Valor: dec("99")},
	}
	out := Generate(Montar(d))
	validar(t, out)
	e110 := todasLinhas(out, "E110")[0]
	// saldo = 204 + 7 + 3 − (131 + 20 + 1 + 10) = 52; recolher = 52 − 2 = 50.
	want := map[int]string{2: "204,00", 4: "7,00", 5: "3,00", 6: "131,00", 8: "20,00", 9: "1,00", 10: "10,00",
		11: "52,00", 12: "2,00", 13: "50,00", 14: "0,00", 15: "4,00"}
	for i, w := range want {
		if campo(e110, i) != w {
			t.Errorf("E110 campo %d = %q, want %q (%s)", i, campo(e110, i), w, e110)
		}
	}
	if n := len(todasLinhas(out, "E111")); n != 6 {
		t.Errorf("E111 = %d, want 6 (o de ST fica fora)", n)
	}
	e116 := todasLinhas(out, "E116")
	if len(e116) != 2 || campo(e116[0], 2) != "000" || campo(e116[0], 3) != "50,00" || campo(e116[1], 2) != "090" || campo(e116[1], 3) != "4,00" {
		t.Errorf("E116 = %v", e116)
	}
}

func TestTipoAjuste(t *testing.T) {
	for c, want := range map[string]bool{"PR020001": true, "SP050123": true, "PR100001": false, "PR060001": false, "PR0201": false, "": false} {
		if _, ok := TipoAjuste(c); ok != want {
			t.Errorf("TipoAjuste(%q) ok = %v", c, ok)
		}
	}
}

func TestMontar_Inventario(t *testing.T) {
	d := dadosPeriodo()
	d.InventarioItens = []ItemInventario{
		{Item: ItemCadastro{Cod: "5005", Desc: "Parafuso", UnidInv: "PC", Tipo: "01", NCM: "73181500"}, Quantidade: dec("3"), Valor: dec("10.00")},
		{Item: ItemCadastro{Cod: "5006", Desc: "Zerado"}, Quantidade: dec("0"), Valor: dec("0")},
	}
	d.DataInventario = time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	d.ContaEstoque = "1.1.4.01"
	out := Generate(Montar(d))
	validar(t, out)
	h010 := todasLinhas(out, "H010")
	if len(h010) != 1 || campo(h010[0], 5) != "3,333333" || campo(h010[0], 10) != "1.1.4.01" {
		t.Fatalf("H010 = %v", h010)
	}
	if h005 := todasLinhas(out, "H005"); campo(h005[0], 2) != "31122025" || campo(h005[0], 3) != "10,00" || campo(h005[0], 4) != "01" {
		t.Errorf("H005 = %v", h005)
	}
	achou := false
	for _, l := range todasLinhas(out, "0200") {
		if campo(l, 2) == "5005" {
			achou = true
		}
	}
	if !achou {
		t.Error("item do inventário sem 0200")
	}
}
