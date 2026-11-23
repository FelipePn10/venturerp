package production_order_uc

import (
	"testing"
	"time"
)

func TestInicioAPartirDeHoje(t *testing.T) {
	sp, _ := time.LoadLocation("America/Sao_Paulo")
	agora := time.Date(2026, 11, 23, 22, 30, 0, 0, sp) // já é 24/11 em UTC: vale o dia local
	d := func(dia int) *time.Time { x := time.Date(2026, 11, dia, 0, 0, 0, 0, time.UTC); return &x }
	casos := []struct {
		nome             string
		ini, fim         *time.Time
		querIni, querFim string
	}{
		{"vazio vira hoje", nil, d(30), "23/11", "30/11"},
		{"passado vira hoje", d(20), d(30), "23/11", "30/11"},
		{"término vencido acompanha", d(20), d(21), "23/11", "23/11"},
		{"futuro fica", d(25), nil, "25/11", ""},
	}
	for _, c := range casos {
		i, f := inicioAPartirDeHoje(c.ini, c.fim, agora)
		gotF := ""
		if f != nil {
			gotF = f.Format("02/01")
		}
		if i.Format("02/01") != c.querIni || gotF != c.querFim {
			t.Errorf("%s: %s → %s", c.nome, i.Format("02/01"), gotF)
		}
	}
}
