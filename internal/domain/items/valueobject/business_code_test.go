package valueobject

import "testing"

func TestBusinessCodeNormalizesAndPreservesMeaningfulCharacters(t *testing.T) {
	code, err := NewBusinessCode(" tea452-0 ")
	if err != nil {
		t.Fatal(err)
	}
	if code != "TEA452-0" {
		t.Fatalf("codigo=%q", code)
	}
	zero, err := NewBusinessCode("0007-A")
	if err != nil || zero != "0007-A" {
		t.Fatalf("zeros significativos nao preservados: %q %v", zero, err)
	}
}

func TestBusinessCodeRejectsUnsafeCharacters(t *testing.T) {
	for _, code := range []string{"", "A B", "A@B", "-ABC"} {
		if _, err := NewBusinessCode(code); err == nil {
			t.Fatalf("codigo invalido aceito: %q", code)
		}
	}
}
