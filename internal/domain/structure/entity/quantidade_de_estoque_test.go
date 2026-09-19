package entity

import (
	"math"
	"testing"
)

func quase(t *testing.T, nome string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s: %.9f, esperado %.9f", nome, got, want)
	}
}

// O caso do usuário: a chapa é estocada em KG e a engenharia desenhou em M².
// 2 m² de uma chapa que pesa 15,7 kg/m² são 31,4 kg — e é isso que o MRP tem
// de reservar, não 2.
func TestQuantidadeSaiNaUnidadeDeEstoque(t *testing.T) {
	linha := &ItemStructure{Quantity: 2, ConversionFactor: 15.7}
	quase(t, "quantidade em estoque", linha.QuantidadeNaUnidadeDeEstoque(), 31.4)

	resolvida, usouFormula := linha.ResolvedQuantity(nil)
	quase(t, "quantidade resolvida", resolvida, 31.4)
	if usouFormula {
		t.Error("não havia fórmula; não deveria dizer que aplicou uma")
	}
}

// Linha gravada antes da migração 354: sem fator. A unidade da estrutura era
// sempre a de estoque, então o número já está certo. Devolver zero aqui faria a
// ordem nascer sem componente — o pior desfecho possível.
func TestLinhaAntigaSemFatorNaoZeraAQuantidade(t *testing.T) {
	linha := &ItemStructure{Quantity: 7}
	quase(t, "fator", linha.FatorParaEstoque(), 1)
	quase(t, "quantidade", linha.QuantidadeNaUnidadeDeEstoque(), 7)
}

// Fator negativo ou zero é dado corrompido; cair em 1 mantém o comportamento
// anterior à conversão em vez de multiplicar tudo por zero.
func TestFatorInvalidoCaiEmUm(t *testing.T) {
	for _, f := range []float64{0, -3} {
		linha := &ItemStructure{Quantity: 5, ConversionFactor: f}
		quase(t, "quantidade", linha.QuantidadeNaUnidadeDeEstoque(), 5)
	}
}

// Com fórmula, o resultado dela também está na unidade da estrutura e precisa
// da mesma conversão. Sem isso, o item configurado escaparia da regra.
func TestFormulaTambemPassaPelaConversao(t *testing.T) {
	f := "COMPRIMENTO * LARGURA"
	linha := &ItemStructure{
		Quantity: 1, ConversionFactor: 15.7,
		QuantityFormula: &f, QuantityRounding: "NONE", QuantityScale: 4,
	}
	resolvida, usouFormula := linha.ResolvedQuantity(map[string]float64{"COMPRIMENTO": 2, "LARGURA": 1.5})
	if !usouFormula {
		t.Fatal("a fórmula deveria ter sido aplicada")
	}
	quase(t, "3 m² convertidos", resolvida, 3*15.7)
}

// Unidades iguais: fator 1, nada muda. É o caso de praticamente toda estrutura
// já cadastrada, e a garantia de que a migração não mexeu em número nenhum.
func TestUnidadesIguaisNaoMudamNada(t *testing.T) {
	linha := &ItemStructure{Quantity: 12.5, ConversionFactor: 1}
	quase(t, "quantidade", linha.QuantidadeNaUnidadeDeEstoque(), 12.5)
}

// A quantidade convertida gravada é a autoridade, não o fator arredondado.
// Caso real: 104 mm de uma barra de 6000 mm. O fator inverso é 0,000166666…;
// arredondado a seis casas vira 0,000167 e devolve 0,017368 em vez de 0,017333
// — meio por cento a mais, sempre para o mesmo lado.
func TestQuantidadeGravadaVenceOFatorArredondado(t *testing.T) {
	linha := &ItemStructure{
		Quantity:         104,
		QuantityStockUOM: 0.0173333, // calculado na gravação, precisão inteira
		ConversionFactor: 0.000167,  // o mesmo fator depois do arredondamento
	}
	quase(t, "quantidade em estoque", linha.QuantidadeNaUnidadeDeEstoque(), 0.0173333)

	resolvida, usouFormula := linha.ResolvedQuantity(nil)
	quase(t, "quantidade resolvida", resolvida, 0.0173333)
	if usouFormula {
		t.Error("não havia fórmula")
	}
}

// Com fórmula a coluna gravada vale para a quantidade FIXA e não serve; aí o
// fator é aplicado sobre o resultado da fórmula.
func TestComFormulaOFatorEhAplicadoSobreOResultado(t *testing.T) {
	f := "COMPRIMENTO"
	linha := &ItemStructure{
		Quantity: 1, QuantityStockUOM: 15.7, ConversionFactor: 15.7,
		QuantityFormula: &f, QuantityRounding: "NONE", QuantityScale: 4,
	}
	resolvida, usouFormula := linha.ResolvedQuantity(map[string]float64{"COMPRIMENTO": 3})
	if !usouFormula {
		t.Fatal("a fórmula deveria ter sido aplicada")
	}
	quase(t, "3 × 15,7", resolvida, 47.1)
}
