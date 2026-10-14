package fiscal_classification_uc

import "testing"

// Data mal digitada não pode sumir em silêncio: vira erro com o formato certo.
func TestDataOpcional(t *testing.T) {
	vazio, ok, ruim := "  ", "2026-10-13", "13/10/2026"
	if d, err := dataOpcional(nil, "início"); d != nil || err != nil {
		t.Fatalf("nil: %v %v", d, err)
	}
	if d, err := dataOpcional(&vazio, "início"); d != nil || err != nil {
		t.Fatalf("vazio: %v %v", d, err)
	}
	if d, err := dataOpcional(&ok, "início"); err != nil || d.Format("2006-01-02") != ok {
		t.Fatalf("válida: %v %v", d, err)
	}
	if _, err := dataOpcional(&ruim, "início da vigência"); err == nil || err.Error() != "início da vigência inválida: use o formato AAAA-MM-DD" {
		t.Fatalf("inválida: %v", err)
	}
}
