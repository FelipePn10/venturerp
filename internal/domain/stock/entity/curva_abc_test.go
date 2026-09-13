package entity

import "testing"

func TestClassificarABC(t *testing.T) {
	const corteA, corteB = 80.0, 95.0
	casos := []struct {
		nome                    string
		acumulado, participacao float64
		esperado                string
	}{
		// O caso que expôs o defeito: item único, 100% do consumo.
		{"item único responde por tudo", 100, 100, "A"},
		{"primeiro item de muitos", 40, 40, "A"},
		{"item que cruza o corte dos 80% ainda é A", 82, 10, "A"},
		{"logo depois do corte vira B", 90, 8, "B"},
		{"item que cruza os 95% ainda é B", 96, 5, "B"},
		{"cauda é C", 99, 1, "C"},
		{"último item irrelevante", 100, 0.1, "C"},
	}
	for _, caso := range casos {
		if got := ClassificarABC(caso.acumulado, caso.participacao, corteA, corteB); got != caso.esperado {
			t.Errorf("%s (acum=%.1f part=%.1f): esperava %s, veio %s",
				caso.nome, caso.acumulado, caso.participacao, caso.esperado, got)
		}
	}
}
