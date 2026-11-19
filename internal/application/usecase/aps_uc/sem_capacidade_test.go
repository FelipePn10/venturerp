package aps_uc

import "testing"

func TestMensagemSemCapacidadeDizOFEtapaEHoras(t *testing.T) {
	got := semCapacidadeNoAno(603, 10, 6250)
	want := "a etapa 10 da OF 603 precisa de 6250,0 h (preparação + operação) e não cabe em um ano de capacidade das máquinas do centro: confira o tempo e a unidade do roteiro do item"
	if got != want {
		t.Fatalf("mensagem:\n%s\nesperado:\n%s", got, want)
	}
}
