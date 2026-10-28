package service

import "testing"

// TestAplicaPoliticaDeLote trava o comportamento dos campos "lote mínimo" e
// "lote múltiplo" do cadastro do item, que eram gravados na tela e ignorados
// pelo planejamento.
func TestAplicaPoliticaDeLote(t *testing.T) {
	casos := []struct {
		nome                          string
		necessidade, minimo, multiplo float64
		esperado                      float64
	}{
		{"sem política devolve a necessidade", 37, 0, 0, 37},
		{"abaixo do mínimo sobe para o mínimo", 12, 50, 0, 50},
		{"acima do mínimo não é reduzido", 80, 50, 0, 80},
		{"arredonda para cima no múltiplo", 37, 0, 12, 48},
		{"múltiplo exato não infla", 48, 0, 12, 48},
		{"mínimo primeiro, múltiplo depois", 12, 50, 12, 60},
		{"necessidade zero não vira lote", 0, 50, 12, 0},
		{"necessidade negativa é preservada", -5, 50, 12, -5},
		// Frações: em float, 0,3 ÷ 0,1 e somas como 0,1 + 0,2 deixam resíduo
		// binário que o Ceil transformava em quantidade torta ou lote a mais.
		{"múltiplo fracionário exato", 0.3, 0, 0.1, 0.3},
		{"resíduo de soma não infla o lote", 0.1 + 0.2, 0, 0.1, 0.3},
		{"resíduo de soma não pula para o próximo inteiro", 1.1 + 0.9 + 1e-12, 0, 1, 2},
		{"fração sobe para o próximo múltiplo", 2.31, 0, 0.1, 2.4},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := aplicaPoliticaDeLote(dq(c.necessidade), dq(c.minimo), dq(c.multiplo)); !got.Equal(dq(c.esperado)) {
				t.Fatalf("necessidade %v (mín %v, múlt %v) = %v, esperado %v",
					c.necessidade, c.minimo, c.multiplo, got, c.esperado)
			}
		})
	}
}

func TestDivisoesParaCimaSemResiduoBinario(t *testing.T) {
	casos := []struct{ a, b, esperado float64 }{
		{0.69, 0.23, 3},
		{0.3, 0.1, 3},
		{10, 4, 3},
		{12, 4, 3},
		{0.1 + 0.2, 0.3, 1},
	}
	for _, c := range casos {
		if got := divisoesParaCima(c.a, c.b); got != c.esperado {
			t.Fatalf("⌈%v ÷ %v⌉ = %v, esperado %v", c.a, c.b, got, c.esperado)
		}
	}
}
