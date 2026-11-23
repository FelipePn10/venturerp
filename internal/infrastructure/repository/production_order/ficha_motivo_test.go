package production_order

import "testing"

func TestMotivoDoEvento(t *testing.T) {
	notas := "2026-11-23 13:51:58 UTC · Em andamento\n2026-11-23 13:52:10 UTC · Pausada: falta de material: chapa"
	if got := motivoDoEvento(notas); got != "falta de material: chapa" {
		t.Fatalf("motivo = %q", got)
	}
	for _, n := range []string{"", "2026-11-23 13:51:58 UTC · Concluída", "texto livre"} {
		if got := motivoDoEvento(n); got != "" {
			t.Fatalf("motivo de %q = %q, quer vazio", n, got)
		}
	}
}
