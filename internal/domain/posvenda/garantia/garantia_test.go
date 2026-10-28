package garantia

import (
	"testing"
	"time"
)

func d(s string) *time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return &t
}

func TestDeterminar(t *testing.T) {
	r12 := Regra{Meses: 12, Inicio: InicioVenda, CobrePecas: true, CobreMaoObra: true}
	casos := []struct {
		nome   string
		r      Regra
		b      Base
		hoje   string
		em     bool
		ate    string
		estend bool
	}{
		{"12 meses da venda, último dia ainda vale", r12, Base{Venda: d("2025-03-10")}, "2026-03-09", true, "2026-03-09", false},
		{"12 meses da venda, dia seguinte vencida", r12, Base{Venda: d("2025-03-10")}, "2026-03-10", false, "2026-03-09", false},
		{"instalação manda quando registrada", Regra{Meses: 6, Inicio: InicioInstalacao}, Base{Venda: d("2025-01-05"), Instalacao: d("2025-03-01")}, "2025-08-20", true, "2025-08-31", false},
		{"instalação ausente: conta da venda", Regra{Meses: 6, Inicio: InicioInstalacao}, Base{Venda: d("2025-01-05")}, "2025-08-20", false, "2025-07-04", false},
		{"dias do item quando o perfil não tem meses", Regra{DiasDoItem: 90}, Base{Venda: d("2025-01-01")}, "2025-03-31", true, "2025-03-31", false},
		{"estendida prevalece", r12, Base{Venda: d("2024-01-01"), EstendidaAte: d("2026-12-31")}, "2026-06-01", true, "2026-12-31", true},
		{"estendida menor que a normal não muda", r12, Base{Venda: d("2025-06-01"), EstendidaAte: d("2025-07-01")}, "2026-01-01", true, "2026-05-31", false},
	}
	for _, c := range casos {
		got := Determinar(c.r, c.b, *d(c.hoje))
		if got.EmGarantia != c.em || got.Ate == nil || got.Ate.Format("2006-01-02") != c.ate || got.Estendida != c.estend {
			t.Errorf("%s: %+v", c.nome, got)
		}
	}
	if got := Determinar(r12, Base{}, *d("2025-01-01")); got.EmGarantia || got.Motivo == "" {
		t.Errorf("sem data de venda não há garantia: %+v", got)
	}
	if got := Determinar(Regra{}, Base{Venda: d("2025-01-01")}, *d("2025-01-02")); got.EmGarantia || got.Ate != nil {
		t.Errorf("sem prazo não há garantia: %+v", got)
	}
	if got := Determinar(r12, Base{Venda: d("2020-01-01")}, *d("2025-01-01")); got.CobrePecas || got.CobreMaoObra {
		t.Errorf("garantia vencida não cobre nada: %+v", got)
	}
}
