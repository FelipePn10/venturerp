package sla

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func sp(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Skip("sem tzdata")
	}
	return loc
}

// Seg–sex 08–12 e 13–18 (padrão): aberto sexta 17h com 4 h úteis vence
// segunda 11h; com a segunda feriado, terça 11h.
func TestSomarHorasUteis(t *testing.T) {
	loc := sp(t)
	sexta17 := time.Date(2026, 10, 23, 17, 0, 0, 0, loc)
	cal := NovoCalendario(loc, nil, nil)
	if got := cal.Somar(sexta17, 4*time.Hour); !got.Equal(time.Date(2026, 10, 26, 11, 0, 0, 0, loc)) {
		t.Fatalf("seg 11h: %v", got)
	}
	feriado := NovoCalendario(loc, nil, []time.Time{time.Date(2026, 10, 26, 0, 0, 0, 0, loc)})
	if got := feriado.Somar(sexta17, 4*time.Hour); !got.Equal(time.Date(2026, 10, 27, 11, 0, 0, 0, loc)) {
		t.Fatalf("ter 11h: %v", got)
	}
	// Aberto fora do expediente (sábado) começa a contar segunda 08h.
	if got := cal.Somar(time.Date(2026, 10, 24, 10, 0, 0, 0, loc), time.Hour); !got.Equal(time.Date(2026, 10, 26, 9, 0, 0, 0, loc)) {
		t.Fatalf("sábado: %v", got)
	}
	// Termina exatamente no fim do trecho: fica 12:00, não 13:00.
	if got := cal.Somar(time.Date(2026, 10, 26, 8, 0, 0, 0, loc), 4*time.Hour); !got.Equal(time.Date(2026, 10, 26, 12, 0, 0, 0, loc)) {
		t.Fatalf("12h: %v", got)
	}
	// O resultado não depende do fuso do processo.
	if got := cal.Somar(sexta17.UTC(), 4*time.Hour); !got.Equal(time.Date(2026, 10, 26, 11, 0, 0, 0, loc)) {
		t.Fatalf("entrada em UTC: %v", got)
	}
}

func TestEntreHorasUteis(t *testing.T) {
	loc := sp(t)
	cal := NovoCalendario(loc, nil, nil)
	// Sexta 17h → segunda 10h: 1 h na sexta + 2 h na segunda.
	if d := cal.Entre(time.Date(2026, 10, 23, 17, 0, 0, 0, loc), time.Date(2026, 10, 26, 10, 0, 0, 0, loc)); d != 3*time.Hour {
		t.Fatalf("3h: %v", d)
	}
	if d := cal.Entre(time.Date(2026, 10, 26, 12, 0, 0, 0, loc), time.Date(2026, 10, 26, 13, 0, 0, 0, loc)); d != 0 {
		t.Fatalf("almoço: %v", d)
	}
}

func TestPrazosEmHorasUteisERetomada(t *testing.T) {
	loc := sp(t)
	cal := NovoCalendario(loc, nil, nil)
	r := Regra{ID: 9, Prioridade: "HIGH", HorasResposta: decimal.NewFromInt(2), HorasSolucao: decimal.NewFromInt(18), AlertaPct: 50, HorasUteis: true}
	ab := time.Date(2026, 10, 26, 16, 0, 0, 0, loc) // segunda 16h
	p := r.Prazos(ab, cal)
	// 2 h: segunda 18h. 18 h: seg 2 h + ter 9 h + qua 7 h → quarta 16h. Risco 9 h: ter 16h.
	if !p.RespostaAte.Equal(time.Date(2026, 10, 26, 18, 0, 0, 0, loc)) || !p.SolucaoAte.Equal(time.Date(2026, 10, 28, 16, 0, 0, 0, loc)) ||
		!p.SolucaoRisco.Equal(time.Date(2026, 10, 27, 16, 0, 0, 0, loc)) || !p.HorasUteis {
		t.Fatalf("prazos úteis: %v %v %v", p.RespostaAte, p.SolucaoAte, p.SolucaoRisco)
	}
	// Parado de terça 10h a quarta 10h (9 h úteis): a solução vai a quinta 16h;
	// a resposta, já dada, não muda.
	desde := time.Date(2026, 10, 27, 10, 0, 0, 0, loc)
	e := EstadoDoRelogio{PausadoDesde: &desde, HorasUteis: true, Respondido: true, RespostaAte: p.RespostaAte, SolucaoAte: p.SolucaoAte, SolucaoRisco: p.SolucaoRisco}
	n, parado := e.Retomada(time.Date(2026, 10, 28, 10, 0, 0, 0, loc), cal)
	if parado != 9*time.Hour || !n.SolucaoAte.Equal(time.Date(2026, 10, 29, 16, 0, 0, 0, loc)) || !n.RespostaAte.Equal(*p.RespostaAte) {
		t.Fatalf("retomada: parado %v solução %v resposta %v", parado, n.SolucaoAte, n.RespostaAte)
	}
	// Horas corridas: a pausa de 24 h empurra 24 h.
	e.HorasUteis = false
	n, parado = e.Retomada(desde.Add(24*time.Hour), cal)
	if parado != 24*time.Hour || !n.SolucaoAte.Equal(p.SolucaoAte.Add(24*time.Hour)) {
		t.Fatalf("corrida: %v %v", parado, n.SolucaoAte)
	}
}

func TestConferirExpediente(t *testing.T) {
	if err := ConferirExpediente(ExpedientePadrao()); err != nil {
		t.Fatal(err)
	}
	for nome, iv := range map[string][]Intervalo{
		"vazio":      nil,
		"invertido":  {{DiaSemana: 1, Inicio: 600, Fim: 500}},
		"sobreposto": {{DiaSemana: 1, Inicio: 480, Fim: 720}, {DiaSemana: 1, Inicio: 700, Fim: 800}},
		"dia":        {{DiaSemana: 7, Inicio: 480, Fim: 720}},
	} {
		if ConferirExpediente(iv) == nil {
			t.Errorf("%s: devia recusar", nome)
		}
	}
}
