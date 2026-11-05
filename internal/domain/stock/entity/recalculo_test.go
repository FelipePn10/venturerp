package entity

import (
	"testing"

	"github.com/shopspring/decimal"
)

func n(v int64) decimal.Decimal { return decimal.NewFromInt(v) }

func entrada(id int64, q, p int64) MovimentoCusto {
	return MovimentoCusto{ID: id, Almoxarifado: 1, Tipo: MovementTypeIn, Quantidade: n(q), Unitario: n(p), Total: n(q * p)}
}

func saida(id int64, q, p int64) MovimentoCusto {
	return MovimentoCusto{ID: id, Almoxarifado: 1, Tipo: MovementTypeOut, Quantidade: n(q), Unitario: n(p), Total: n(q * p)}
}

// Uma nota de 10 a R$ 200 lançada depois, com a data de antes da saída: a saída
// tinha ido a R$ 100 e, em ordem, sai a R$ 150.
func TestRecalculoCorrigeASaidaDepoisDoRetroativo(t *testing.T) {
	estados := map[ChaveCusto]CostingState{}
	m := RecalcularCusto(estados, []MovimentoCusto{entrada(1, 10, 100), entrada(3, 10, 200), saida(2, 4, 100)})
	if len(m) != 1 || m[0].MovimentoID != 2 {
		t.Fatalf("mudanças: %+v", m)
	}
	if !m[0].UnitarioDepois.Equal(n(150)) || !m[0].DiferencaResultado.Equal(n(200)) {
		t.Errorf("saída: %+v", m[0])
	}
	st := estados[ChaveCusto{Almoxarifado: 1}]
	if !st.Quantity.Equal(n(16)) || !st.TotalCost.Equal(n(2400)) {
		t.Errorf("estado final: %+v", st)
	}
}

func TestRecalculoSemForaDeOrdemNaoMudaNada(t *testing.T) {
	m := RecalcularCusto(map[ChaveCusto]CostingState{}, []MovimentoCusto{entrada(1, 10, 100), saida(2, 4, 100), entrada(3, 6, 100), saida(4, 2, 100)})
	if len(m) != 0 {
		t.Errorf("mudou o que estava certo: %+v", m)
	}
}

func TestRecalculoTransferenciaLevaOCustoNovo(t *testing.T) {
	estados := map[ChaveCusto]CostingState{}
	out := MovimentoCusto{ID: 4, Almoxarifado: 1, Tipo: MovementTypeTransferOut, Quantidade: n(2), Unitario: n(100), Total: n(200)}
	in := MovimentoCusto{ID: 5, Almoxarifado: 2, Tipo: MovementTypeTransferIn, Quantidade: n(2), Unitario: n(100), Total: n(200)}
	m := RecalcularCusto(estados, []MovimentoCusto{entrada(1, 10, 100), entrada(3, 10, 200), out, in})
	if len(m) != 2 {
		t.Fatalf("mudanças: %+v", m)
	}
	if !m[1].UnitarioDepois.Equal(n(150)) {
		t.Errorf("entrada da transferência: %+v", m[1])
	}
	if !m[0].DiferencaResultado.IsZero() {
		t.Errorf("transferência não vai para o resultado: %+v", m[0])
	}
}

func TestRecalculoEntradaDesfeitaSaiPeloPreco(t *testing.T) {
	estados := map[ChaveCusto]CostingState{}
	cancelamento := MovimentoCusto{ID: 3, Almoxarifado: 1, Tipo: MovementTypeOut, Quantidade: n(10), Unitario: n(200), Total: n(2000), Referencia: ReferenceTypeNFEntry}
	m := RecalcularCusto(estados, []MovimentoCusto{entrada(1, 10, 100), entrada(2, 10, 200), cancelamento})
	if len(m) != 0 {
		t.Errorf("cancelamento de nota mudou: %+v", m)
	}
	if st := estados[ChaveCusto{Almoxarifado: 1}]; !st.AvgCost.Equal(n(100)) {
		t.Errorf("médio devia voltar a 100: %+v", st)
	}
}
