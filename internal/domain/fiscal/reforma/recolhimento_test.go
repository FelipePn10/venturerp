package reforma

import (
	"testing"
	"time"
)

func TestRecolhivelSoAPartirDe2027(t *testing.T) {
	casos := []struct {
		data time.Time
		quer bool
	}{
		{time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC), false},
		{time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{time.Date(2033, 6, 1, 0, 0, 0, 0, time.UTC), true},
	}
	for _, c := range casos {
		if got := Recolhivel(c.data); got != c.quer {
			t.Fatalf("Recolhivel(%s) = %v, esperado %v", c.data.Format("2006-01-02"), got, c.quer)
		}
	}
	for comp, quer := range map[string]bool{"12/2026": false, "01/2027": true, "13/2027": false, "": false} {
		if got := RecolhivelNaCompetencia(comp); got != quer {
			t.Fatalf("RecolhivelNaCompetencia(%q) = %v, esperado %v", comp, got, quer)
		}
	}
}
