package sped

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// Melhoria 4: o K200 separa a peça que está com o terceiro (industrialização por
// encomenda) — IND_EST 1 e o fornecedor como participante.
func TestBlocoKSaldosComPosse(t *testing.T) {
	trate := Participante{Cod: "11222333000181", Nome: "TRATE", CNPJ: "11222333000181"}
	item := ItemCadastro{Cod: "601", Desc: "Eixo", UnidInv: "UN"}
	fim := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)
	p := Montar(DadosEFD{Periodo: EFDPeriodo{DataInicial: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), DataFinal: fim}, EstoqueK: []ItemInventario{
		{Item: item, Quantidade: decimal.NewFromInt(14)},
		{Item: item, Quantidade: decimal.NewFromInt(4), IndProp: "1", Parte: &trate},
	}})
	arq := Generate(p)
	for _, want := range []string{"|K001|0|", "|K010|2|", "|K100|01112026|30112026|", "|K200|30112026|601|14,00000|0||", "|K200|30112026|601|4,00000|1|" + trate.Cod + "|", "|K990|6|"} {
		if !strings.Contains(arq, want) {
			t.Errorf("faltou %s no arquivo:\n%s", want, blocoK(arq))
		}
	}
	vazio := Generate(Montar(DadosEFD{Periodo: EFDPeriodo{DataInicial: fim, DataFinal: fim}}))
	if !strings.Contains(vazio, "|K001|1|") || !strings.Contains(vazio, "|K990|2|") {
		t.Errorf("sem Bloco K: K001 1 e K990 2\n%s", blocoK(vazio))
	}
}

func blocoK(arq string) string {
	var out []string
	for _, l := range strings.Split(arq, "\r\n") {
		if strings.HasPrefix(l, "|K") {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}
