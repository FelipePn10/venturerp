package sped

import (
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/ciap"
)

// O CIAP do mês sai no Bloco G (G110, G125 e, na imobilização, G130/G140), com
// os cadastros 0300/0305/0500/0600, e o crédito entra no E111.
func TestMontar_CIAP(t *testing.T) {
	d := dadosPeriodo()
	set := func(day int) time.Time { return time.Date(2026, time.September, day, 0, 0, 0, 0, time.UTC) }
	novo := ciap.Bem{ID: 1, Codigo: "TORNO-01", Descricao: "Torno CNC", Funcao: "Usinagem", ContaCodigo: "1.2.3.01", ContaNome: "Máquinas", ContaNivel: 4,
		CentroCodigo: "CC-FAB-USIN", CentroNome: "USINAGEM", DataEntrada: set(3), ICMSOp: dec("4800"), NumeroParcelas: 48,
		Documento: &ciap.Documento{Modelo: "55", Serie: "1", Numero: "77102", Chave: strings.Repeat("4", 44), Emissao: set(2),
			CodPart: "F7", NumItem: 1, CodItem: "1001", Unidade: "UN", Quantidade: dec("1")}}
	velho := ciap.Bem{ID: 2, Codigo: "PRENSA-07", Descricao: "Prensa", DataEntrada: time.Date(2022, time.October, 10, 0, 0, 0, 0, time.UTC),
		ICMSOp: dec("960"), NumeroParcelas: 48} // sem conta nem centro: centro padrão "3"
	a, err := ciap.Apurar([]ciap.Bem{novo, velho}, 2026, 9, dec("75000"), dec("100000"))
	if err != nil {
		t.Fatal(err)
	}
	d.CIAP = &DadosCIAP{Apuracao: *a, CodigoAjuste: "PR020001"}
	out := Generate(Montar(d))
	validar(t, out)

	g110 := todasLinhas(out, "G110")
	if len(g110) != 1 || campo(g110[0], 5) != "120,00" || campo(g110[0], 8) != "0,75000000" || campo(g110[0], 9) != "90,00" {
		t.Fatalf("G110: %v", g110) // parcelas 100 + 20 = 120; índice 0,75 → 90,00
	}
	g125 := todasLinhas(out, "G125")
	if len(g125) != 3 {
		t.Fatalf("G125: %v", g125) // IM do torno, SI da prensa (48ª) e BA
	}
	if campo(todasLinhas(out, "G130")[0], 3) != "F7" || len(todasLinhas(out, "G140")) != 1 {
		t.Fatalf("G130/G140: %v %v", todasLinhas(out, "G130"), todasLinhas(out, "G140"))
	}
	if len(todasLinhas(out, "0300")) != 2 || len(todasLinhas(out, "0305")) != 2 || len(todasLinhas(out, "0500")) != 1 || len(todasLinhas(out, "0600")) != 2 {
		t.Fatalf("cadastros: 0300 %v 0500 %v 0600 %v", todasLinhas(out, "0300"), todasLinhas(out, "0500"), todasLinhas(out, "0600"))
	}
	var e111 bool
	for _, l := range todasLinhas(out, "E111") {
		if campo(l, 2) == "PR020001" && campo(l, 4) == "90,00" {
			e111 = true
		}
	}
	if !e111 {
		t.Fatalf("o crédito do CIAP no E111: %v", todasLinhas(out, "E111"))
	}
	// Ordem do bloco 0: o 0300 depois do 0200, o 0600 antes do 0990.
	if strings.Index(out, "|0200|") > strings.Index(out, "|0300|") || strings.Index(out, "|0600|") > strings.Index(out, "|0990|") {
		t.Fatal("ordem dos registros do bloco 0")
	}
	if g990 := todasLinhas(out, "G990"); campo(g990[0], 2) != "8" { // G001, G110, 3 G125, G130, G140, G990
		t.Fatalf("G990: %v", g990)
	}
}
