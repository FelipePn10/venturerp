package routing_uc

import "testing"

// A mensagem de erro mandava "use minuto, hora ou dia" e o sistema recusava
// justamente "MINUTO": quem seguisse a instrução tomava o mesmo erro de novo.
// Máquina grava MINUTO, APS aceita HOUR/MINUTE, o roteiro só gravava MIN.
func TestUnidadeDeTempoAceitaOsTresVocabularios(t *testing.T) {
	canonico := map[string]string{
		"MIN": "MIN", "MINUTO": "MIN", "minuto": "MIN", "MINUTE": "MIN", " MINUTOS ": "MIN",
		"HORA": "HORA", "HOUR": "HORA", "h": "HORA",
		"DIA": "DIA", "DAY": "DIA", "dias": "DIA",
		"": "",
	}
	for entrada, esperado := range canonico {
		got, ok := normalizaUnidadeDeTempo(entrada)
		if !ok || got != esperado {
			t.Errorf("%q: esperava %q, veio %q (ok=%v)", entrada, esperado, got, ok)
		}
	}
	for _, invalida := range []string{"SEMANA", "XPTO", "12"} {
		if _, ok := normalizaUnidadeDeTempo(invalida); ok {
			t.Errorf("%q deveria ser recusada", invalida)
		}
	}
}
