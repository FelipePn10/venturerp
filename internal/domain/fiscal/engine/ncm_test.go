package engine

import "testing"

func TestNormalizarNCM(t *testing.T) {
	casos := map[string]string{
		"8466.20.90":  "84662090",
		"84662090":    "84662090",
		" 8466 2090 ": "84662090",
		"8466-20-90":  "84662090",
		"":            "",
		"abc":         "",
	}
	for entrada, esperado := range casos {
		if got := NormalizarNCM(entrada); got != esperado {
			t.Errorf("NormalizarNCM(%q) = %q, esperado %q", entrada, got, esperado)
		}
	}
}

// O defeito que originou isto: a tabela tributária cadastrada COM máscara e o item
// COM a classificação sem máscara (ou o contrário) não casavam, e o motor não
// reclamava — devolvia a alíquota padrão. A nota saía com imposto zerado.
//
// O teste cobre as duas direções, porque arrumar só uma deixaria metade do defeito.
func TestMascaraDoNCMNaoDecideSeOItemETributado(t *testing.T) {
	tributacao := &NcmTaxConfig{
		AliqIPI: 0.05, AliqPis: 0.0165, AliqCofins: 0.076,
		CstPis: "01", CstCofins: "01", CstIPI: "50",
	}

	casos := []struct {
		nome        string
		ncmDaTabela string
		ncmDoItem   string
	}{
		{"tabela com máscara, item sem", "8471.49.00", "84714900"},
		{"tabela sem máscara, item com", "84714900", "8471.49.00"},
		{"as duas com máscara", "8471.49.00", "8471.49.00"},
		{"as duas sem máscara", "84714900", "84714900"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			params := TaxCalculationParams{
				Itens:      []TaxItem{{Ncm: c.ncmDoItem, ValorUnitario: 100, Quantidade: 10}},
				EmitenteUF: "PR", DestinoUF: "PR", DestinoTipo: "contribuinte", Cfop: "5101",
			}
			tabela := map[string]*NcmTaxConfig{c.ncmDaTabela: tributacao}

			res, err := CalcularImpostos(params, tabela, defaultInterstateTable(),
				defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
			if err != nil {
				t.Fatalf("CalcularImpostos: %v", err)
			}
			item := res.Itens[0]

			// base 1000: IPI 5% = 50, PIS 1,65% = 16,50, COFINS 7,6% = 76.
			near(t, "ValorIPI", 50.00, item.ValorIPI)
			near(t, "ValorPIS", 16.50, item.ValorPIS)
			near(t, "ValorCOFINS", 76.00, item.ValorCOFINS)
			if item.CSTIPI != "50" {
				t.Errorf("CSTIPI = %q, esperado 50 (veio da tabela?)", item.CSTIPI)
			}
		})
	}
}

// Controle negativo: um NCM que realmente não está na tabela tem de continuar
// caindo na alíquota padrão. Sem isto, uma normalização frouxa demais (por exemplo
// casar por prefixo) passaria no teste de cima e tributaria errado.
func TestNCMAusenteContinuaSemTributacaoDaTabela(t *testing.T) {
	params := TaxCalculationParams{
		Itens:      []TaxItem{{Ncm: "99999999", ValorUnitario: 100, Quantidade: 10}},
		EmitenteUF: "PR", DestinoUF: "PR", DestinoTipo: "contribuinte", Cfop: "5101",
	}
	tabela := map[string]*NcmTaxConfig{"8471.49.00": {AliqIPI: 0.05, CstIPI: "50"}}

	res, err := CalcularImpostos(params, tabela, defaultInterstateTable(),
		defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
	if err != nil {
		t.Fatalf("CalcularImpostos: %v", err)
	}
	near(t, "ValorIPI", 0.00, res.Itens[0].ValorIPI)
}

// Duas linhas que colapsam na mesma chave não podem fazer o resultado depender da
// ordem de iteração do mapa (que em Go é aleatória).
func TestTabelaComMascaraDuplicadaEDeterministica(t *testing.T) {
	tabela := map[string]*NcmTaxConfig{
		"8471.49.00": {AliqIPI: 0.05, CstIPI: "50"},
		"84714900":   {AliqIPI: 0.10, CstIPI: "50"},
	}
	primeira := NormalizarTabelaNCM(tabela)["84714900"].AliqIPI
	for i := 0; i < 50; i++ {
		if got := NormalizarTabelaNCM(tabela)["84714900"].AliqIPI; got != primeira {
			t.Fatalf("resultado variou entre execuções: %v e %v", primeira, got)
		}
	}
}
