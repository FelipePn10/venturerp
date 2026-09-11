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
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := aplicaPoliticaDeLote(c.necessidade, c.minimo, c.multiplo); got != c.esperado {
				t.Fatalf("necessidade %.0f (mín %.0f, múlt %.0f) = %.0f, esperado %.0f",
					c.necessidade, c.minimo, c.multiplo, got, c.esperado)
			}
		})
	}
}
