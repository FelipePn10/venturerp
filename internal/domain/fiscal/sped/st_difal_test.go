package sped

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func df(v float64) decimal.Decimal { return decimal.NewFromFloat(v) }

// Saída com DIFAL para GO e com ST para SP: C101 na nota, E300/E310/E316 de GO
// e E200/E210/E250 de SP, com as contagens do bloco 9 batendo.
func TestEFDApuraSTEDIFALPorUF(t *testing.T) {
	ini := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	fin := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	d := DadosEFD{
		Empresa: EFDEmpresa{CNPJ: "52454668000102", Nome: "TECNOFER", UF: "PR", IE: "9103144679", CodigoMunicipio: "4114203"},
		Periodo: EFDPeriodo{DataInicial: ini, DataFinal: fin}, VencimentoICMS: time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC),
		CodReceitaICMS: "1015",
		Saidas: []NotaSaida{
			{Part: Participante{}, Serie: "1", Numero: "1", Chave: "41261052454668000102550010000000011000000011", Emissao: ini, Saida: ini,
				Total: df(1050), Produtos: df(1000), IPI: df(50), UF: "GO",
				Itens: []ItemSaida{{CST: "000", CFOP: "6108", AliqICMS: df(7), BaseICMS: df(1050), ICMS: df(73.5), BaseIPI: df(1000), IPI: df(50), Total: df(1000), DIFAL: df(126), FCP: df(0)}}},
			{Part: Participante{}, Serie: "1", Numero: "2", Chave: "41261052454668000102550010000000021000000022", Emissao: ini, Saida: ini,
				Total: df(1213.96), Produtos: df(1000), IPI: df(50), ST: df(163.96), BaseST: df(1577.56), UF: "SP",
				Itens: []ItemSaida{{CST: "010", CFOP: "6401", AliqICMS: df(12), BaseICMS: df(1000), ICMS: df(120), BaseST: df(1577.56), ST: df(163.96), BaseIPI: df(1000), IPI: df(50), Total: df(1000)}}},
			{Part: Participante{}, Serie: "1", Numero: "3", Chave: "41261052454668000102550010000000031000000033", Emissao: ini, Saida: ini,
				Total: df(1050), Produtos: df(1000), IPI: df(50), UF: "PR",
				Itens: []ItemSaida{{CST: "051", CFOP: "5102", AliqICMS: df(19.5), BaseICMS: df(1000), ICMS: df(120), BaseIPI: df(1000), IPI: df(50), Total: df(1000)}}},
		},
	}
	arq := Generate(Montar(d))
	tem := func(trecho string) {
		t.Helper()
		if !strings.Contains(arq, trecho) {
			t.Fatalf("EFD sem %q:\n%s", trecho, arq)
		}
	}
	tem("|C101|0,00|126,00|0,00|")
	tem("|E200|SP|01102026|31102026|")
	tem("|E210|1|0,00|0,00|0,00|0,00|0,00|163,96|0,00|0,00|163,96|0,00|163,96|0,00|0,00|")
	tem("|E250|999|163,96|10112026|100099|")
	tem("|E300|GO|01102026|31102026|")
	tem("|E310|1|0,00|126,00|0,00|0,00|0,00|126,00|0,00|126,00|0,00|0,00|0,00|0,00|0,00|0,00|0,00|0,00|0,00|0,00|0,00|0,00|")
	tem("|E316|000|126,00|10112026|100102|")
	// ICMS próprio: 73,50 + 120 + 120 (o da nota com diferimento já é o destacado).
	tem("|E110|313,50|")
	tem("|9900|C101|1|")
	tem("|9900|E310|1|")
	if strings.Contains(arq, "|E316|006|") {
		t.Fatal("sem FCP não pode haver obrigação de FCP")
	}
}
