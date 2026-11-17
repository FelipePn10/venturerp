package focusnfe

import "testing"

func TestBasePISCOFINS(t *testing.T) {
	dif := 46.15
	casos := []struct {
		nome string
		it   NFEItem
		want float64
	}{
		{"Tecnofer: 6.950,00 − ICMS 7% (486,50)", NFEItem{ValorBruto: 6950, CodigoSituacaoTributariaICMS: "00", ValorICMS: 486.50,
			AliquotaPIS: 1.65, ValorPIS: 106.65, AliquotaCOFINS: 7.6, ValorCOFINS: 491.23}, 6463.50},
		{"CST 51: sai só o destacado (operação − diferido)", NFEItem{ValorBruto: 1000, CodigoSituacaoTributariaICMS: "51", ValorICMS: 120, ValorICMSDiferido: &dif,
			AliquotaPIS: 1.65, ValorPIS: 15.28}, 926.15},
		{"frete e desconto entram como no motor", NFEItem{ValorBruto: 1000, ValorFrete: 50, ValorDesconto: 10, CodigoSituacaoTributariaICMS: "00", ValorICMS: 124.80,
			AliquotaPIS: 1.65, ValorPIS: 15.10}, 915.20},
		{"CST sem ICMS: base cheia", NFEItem{ValorBruto: 1000, CodigoSituacaoTributariaICMS: "40", ValorICMS: 0, AliquotaPIS: 1.65, ValorPIS: 16.50}, 1000},
		{"rascunho calculado antes da exclusão mantém a base que reproduz o valor", NFEItem{ValorBruto: 6950, CodigoSituacaoTributariaICMS: "00", ValorICMS: 486.50,
			AliquotaPIS: 1.65, ValorPIS: 114.68, AliquotaCOFINS: 7.6, ValorCOFINS: 528.20}, 6950},
		{"valor proporcional de devolução de nota antiga (arredondamento próprio) fica com a base cheia", NFEItem{ValorBruto: 2316.67, CodigoSituacaoTributariaICMS: "00", ValorICMS: 162.17,
			AliquotaPIS: 1.65, ValorPIS: 38.23}, 2316.67},
		{"só COFINS: confere pela COFINS", NFEItem{ValorBruto: 6950, CodigoSituacaoTributariaICMS: "00", ValorICMS: 486.50,
			AliquotaCOFINS: 7.6, ValorCOFINS: 491.23}, 6463.50},
	}
	for _, c := range casos {
		if got := basePISCOFINS(c.it); got != c.want {
			t.Errorf("%s: base = %v, want %v", c.nome, got, c.want)
		}
	}
}
