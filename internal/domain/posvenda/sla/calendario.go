package sla

import (
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// Intervalo de expediente num dia da semana, em minutos desde a meia-noite.
type Intervalo struct {
	DiaSemana int `json:"dia_semana"` // 0 = domingo … 6 = sábado (time.Weekday)
	Inicio    int `json:"inicio"`
	Fim       int `json:"fim"`
}

// Hora formata minutos como HH:MM.
func Hora(min int) string { return fmt.Sprintf("%02d:%02d", min/60, min%60) }

// LerHora lê HH:MM em minutos.
func LerHora(s string) (int, error) {
	t, err := time.Parse("15:04", strings.TrimSpace(s))
	if err != nil {
		return 0, domainerr.NewValidation(fmt.Sprintf("horário inválido %q (use HH:MM)", s))
	}
	return t.Hour()*60 + t.Minute(), nil
}

// ExpedientePadrao: segunda a sexta, 08:00–12:00 e 13:00–18:00 — vale enquanto
// a empresa não cadastrar o seu.
func ExpedientePadrao() []Intervalo {
	out := make([]Intervalo, 0, 10)
	for d := 1; d <= 5; d++ {
		out = append(out, Intervalo{DiaSemana: d, Inicio: 8 * 60, Fim: 12 * 60}, Intervalo{DiaSemana: d, Inicio: 13 * 60, Fim: 18 * 60})
	}
	return out
}

// ConferirExpediente: intervalos válidos e sem sobreposição no mesmo dia.
func ConferirExpediente(iv []Intervalo) error {
	if len(iv) == 0 {
		return domainerr.NewValidation("informe ao menos um intervalo de expediente")
	}
	por := map[int][]Intervalo{}
	for _, x := range iv {
		if x.DiaSemana < 0 || x.DiaSemana > 6 {
			return domainerr.NewValidation("dia da semana inválido (0 = domingo … 6 = sábado)")
		}
		if x.Inicio < 0 || x.Fim > 24*60 || x.Fim <= x.Inicio {
			return domainerr.NewValidation(fmt.Sprintf("intervalo %s–%s inválido: o fim vem depois do início", Hora(x.Inicio), Hora(x.Fim)))
		}
		for _, y := range por[x.DiaSemana] {
			if x.Inicio < y.Fim && y.Inicio < x.Fim {
				return domainerr.NewValidation(fmt.Sprintf("intervalos sobrepostos no mesmo dia: %s–%s e %s–%s", Hora(y.Inicio), Hora(y.Fim), Hora(x.Inicio), Hora(x.Fim)))
			}
		}
		por[x.DiaSemana] = append(por[x.DiaSemana], x)
	}
	return nil
}

// Calendario conta horas úteis: expediente por dia da semana, no fuso da
// empresa, menos as folgas (feriados do calendário industrial).
type Calendario struct {
	Loc        *time.Location
	expediente [7][]Intervalo
	folgas     map[string]bool
}

// NovoCalendario: sem intervalos, usa o expediente padrão.
func NovoCalendario(loc *time.Location, iv []Intervalo, folgas []time.Time) *Calendario {
	if loc == nil {
		loc = time.Local
	}
	if len(iv) == 0 {
		iv = ExpedientePadrao()
	}
	c := &Calendario{Loc: loc, folgas: map[string]bool{}}
	for _, x := range iv {
		c.expediente[x.DiaSemana] = append(c.expediente[x.DiaSemana], x)
	}
	for d := range c.expediente {
		l := c.expediente[d]
		for i := 1; i < len(l); i++ { // ordena por início (poucos itens)
			for j := i; j > 0 && l[j].Inicio < l[j-1].Inicio; j-- {
				l[j], l[j-1] = l[j-1], l[j]
			}
		}
	}
	for _, f := range folgas {
		c.folgas[f.Format("2006-01-02")] = true
	}
	return c
}

// limiteDeDias: o prazo mais longo que se procura (evita laço sem fim num
// calendário sem expediente).
const limiteDeDias = 3660

// janelas devolve, dia a dia a partir de t, os trechos de expediente.
func (c *Calendario) janelas(dia time.Time, f func(ini, fim time.Time) bool) {
	d := time.Date(dia.Year(), dia.Month(), dia.Day(), 0, 0, 0, 0, c.Loc)
	for n := 0; n < limiteDeDias; n++ {
		if !c.folgas[d.Format("2006-01-02")] {
			for _, x := range c.expediente[d.Weekday()] {
				ini := time.Date(d.Year(), d.Month(), d.Day(), 0, x.Inicio, 0, 0, c.Loc)
				fim := time.Date(d.Year(), d.Month(), d.Day(), 0, x.Fim, 0, 0, c.Loc)
				if !f(ini, fim) {
					return
				}
			}
		}
		d = d.AddDate(0, 0, 1)
	}
}

func (c *Calendario) temExpediente() bool {
	for _, l := range c.expediente {
		if len(l) > 0 {
			return true
		}
	}
	return false
}

// Somar d de expediente a partir de t. O prazo que cai no fim de um trecho
// fica no fim dele (18:00), não no início do seguinte.
func (c *Calendario) Somar(t time.Time, d time.Duration) time.Time {
	if d <= 0 || !c.temExpediente() {
		return t.Add(d)
	}
	t = t.In(c.Loc)
	resto, fimEm := d, t.Add(d)
	c.janelas(t, func(ini, fim time.Time) bool {
		if !fim.After(t) {
			return true
		}
		if ini.Before(t) {
			ini = t
		}
		if dur := fim.Sub(ini); dur >= resto {
			fimEm = ini.Add(resto)
			resto = 0
			return false
		} else {
			resto -= dur
		}
		return true
	})
	return fimEm
}

// Entre: quanto de expediente há entre a e b.
func (c *Calendario) Entre(a, b time.Time) time.Duration {
	if !b.After(a) {
		return 0
	}
	if !c.temExpediente() {
		return b.Sub(a)
	}
	a, b = a.In(c.Loc), b.In(c.Loc)
	var total time.Duration
	c.janelas(a, func(ini, fim time.Time) bool {
		if !ini.Before(b) {
			return false
		}
		if ini.Before(a) {
			ini = a
		}
		if fim.After(b) {
			fim = b
		}
		if fim.After(ini) {
			total += fim.Sub(ini)
		}
		return true
	})
	return total
}
