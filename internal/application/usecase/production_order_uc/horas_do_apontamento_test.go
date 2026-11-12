package production_order_uc

import "testing"

func TestConferirHorasDoApontamento(t *testing.T) {
	p := func(s string) *string { return &s }
	validos := [][2]*string{
		{nil, nil},
		{p(""), p("  ")},
		{p("08:00"), p("16:30")},
		{p("08:00:15"), nil},
		{p("22:00"), p("06:00")}, // turno que vira a noite
	}
	for _, c := range validos {
		if err := conferirHorasDoApontamento(c[0], c[1]); err != nil {
			t.Fatalf("recusou hora válida: %v", err)
		}
	}
	// Formatos que o banco recusava com erro interno.
	invalidos := [][2]*string{
		{p("2026-11-11T08:00:00-03:00"), nil},
		{nil, p("8h")},
		{p("25:00"), nil},
	}
	for _, c := range invalidos {
		if err := conferirHorasDoApontamento(c[0], c[1]); err == nil {
			t.Fatal("aceitou hora em formato inválido")
		}
	}
}
