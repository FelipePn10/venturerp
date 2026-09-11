package entity

import (
	"testing"
	"time"
)

func TestVigenteEm(t *testing.T) {
	d := func(s string) *time.Time { v, _ := time.Parse("2006-01-02", s); return &v }
	casos := []struct {
		nome        string
		inicio, fim *time.Time
		em          string
		vale        bool
	}{
		{"sem datas vale sempre", nil, nil, "2026-09-11", true},
		{"antes do início não vale", d("2026-10-01"), nil, "2026-09-30", false},
		{"no dia do início vale", d("2026-10-01"), nil, "2026-10-01", true},
		{"no dia do fim ainda vale", nil, d("2026-08-12"), "2026-08-12", true},
		{"depois do fim não vale", nil, d("2026-08-12"), "2026-08-13", false},
		{"hora do dia não importa", nil, d("2026-08-12"), "2026-08-12T23:59:00Z", true},
	}
	for _, c := range casos {
		em, err := time.Parse("2006-01-02", c.em)
		if err != nil {
			em, _ = time.Parse(time.RFC3339, c.em)
		}
		s := &ItemStructure{StartDate: c.inicio, EndDate: c.fim}
		if got := s.VigenteEm(em); got != c.vale {
			t.Errorf("%s: esperado %v, obtido %v", c.nome, c.vale, got)
		}
	}
}
