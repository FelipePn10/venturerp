package main

import "testing"

func TestRegimeDoCadastro(t *testing.T) {
	for entrada, esperado := range map[string]string{
		"1": "1", "simples_nacional": "1", "Simples Nacional": "1",
		"2": "2", "lucro_presumido": "2",
		"3": "3", "lucro_real": "3", "Lucro Real": "3", "real": "3",
	} {
		if got, err := regimeDoCadastro(entrada); err != nil || got != esperado {
			t.Errorf("regimeDoCadastro(%q) = %q, %v; esperado %q", entrada, got, err, esperado)
		}
	}
	for _, ruim := range []string{"", "4", "mei", "irreal"} {
		if _, err := regimeDoCadastro(ruim); err == nil {
			t.Errorf("regimeDoCadastro(%q) aceitou um regime inválido", ruim)
		}
	}
}
