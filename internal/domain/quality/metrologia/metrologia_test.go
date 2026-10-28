package metrologia

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func dp(s string) *decimal.Decimal {
	d := decimal.RequireFromString(s)
	return &d
}
func sp(s string) *string { return &s }

var hoje = time.Date(2026, 10, 21, 15, 0, 0, 0, time.UTC)

func paquimetro() *Instrumento {
	i := &Instrumento{Codigo: " pq-01 ", Descricao: "Paquímetro 150 mm", Grupo: "Paquímetro", FrequenciaDias: 180, EMA: dp("0.03")}
	if err := i.Validar(); err != nil {
		panic(err)
	}
	return i
}

func TestCadastroNormaliza(t *testing.T) {
	i := paquimetro()
	if i.Codigo != "PQ-01" || i.Grupo != "PAQUIMETRO" || i.Status != "ATIVO" {
		t.Fatalf("normalização: %+v", i)
	}
	if NormalizarGrupo("  relógio   comparador ") != "RELOGIO COMPARADOR" {
		t.Fatal("grupo com acento e espaços")
	}
	if err := (&Instrumento{Codigo: "X", Descricao: "y", Grupo: "z"}).Validar(); err == nil {
		t.Fatal("sem frequência deveria ser recusado")
	}
	if err := (&Instrumento{Codigo: "X", Descricao: "y", Grupo: "z", FrequenciaDias: 30, EMA: dp("0")}).Validar(); err == nil {
		t.Fatal("EMA zero deveria ser recusado")
	}
}

func TestAvaliacaoDaCalibracao(t *testing.T) {
	i := paquimetro()
	casos := []struct {
		erro, u string
		quer    string
	}{
		{"0.02", "0.005", "APROVADO"},
		{"-0.025", "0.005", "APROVADO"}, // |−0,025| + 0,005 = 0,03 ≤ EMA
		{"0.028", "0.005", "REPROVADO"}, // dentro do EMA, mas a incerteza leva para fora
		{"0.04", "0", "REPROVADO"},
	}
	for _, c := range casos {
		cal := &Calibracao{Data: hoje.AddDate(0, 0, -1), Executor: "INTERNO", ErroEncontrado: dp(c.erro), Incerteza: dp(c.u), Resultado: "APROVADO"}
		if err := cal.Avaliar(i, hoje); err != nil || cal.Resultado != c.quer {
			t.Errorf("erro %s ± %s: %s %v, esperado %s", c.erro, c.u, cal.Resultado, err, c.quer)
		}
	}
	ok := &Calibracao{Data: hoje, Executor: "INTERNO", ErroEncontrado: dp("0.01")}
	if err := ok.Avaliar(i, hoje); err != nil || ok.Proxima == nil || !ok.Proxima.Equal(time.Date(2027, 4, 19, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("próxima = data + 180 dias: %v %v", ok.Proxima, err)
	}
	for _, c := range []*Calibracao{
		{Data: hoje.AddDate(0, 0, 1), Executor: "INTERNO", Resultado: "APROVADO"},
		{Data: hoje, Executor: "EXTERNO", Resultado: "APROVADO"},
		{Data: hoje, Executor: "INTERNO"},
		{Executor: "INTERNO", Resultado: "APROVADO"},
	} {
		if err := c.Avaliar(&Instrumento{FrequenciaDias: 30}, hoje); err == nil {
			t.Errorf("deveria recusar %+v", c)
		}
	}
}

func TestAplicarEPodeMedir(t *testing.T) {
	i := paquimetro()
	if err := i.PodeMedir("PAQUIMETRO", hoje); err == nil || !strings.Contains(err.Error(), "não tem calibração") {
		t.Fatalf("sem calibração: %v", err)
	}
	ok := &Calibracao{Data: hoje.AddDate(0, 0, -10), Executor: "INTERNO", ErroEncontrado: dp("0.01")}
	_ = ok.Avaliar(i, hoje)
	i.Aplicar(ok)
	if err := i.PodeMedir("paquímetro", hoje); err != nil {
		t.Fatalf("calibrado: %v", err)
	}
	if err := i.PodeMedir("MICROMETRO", hoje); err == nil || !strings.Contains(err.Error(), "grupo") {
		t.Fatalf("grupo errado: %v", err)
	}
	if err := i.PodeMedir("", hoje.AddDate(0, 0, 175)); err == nil || !strings.Contains(err.Error(), "venceu") {
		t.Fatalf("vencido: %v", err)
	}
	ruim := &Calibracao{Data: hoje, Executor: "INTERNO", ErroEncontrado: dp("0.05")}
	_ = ruim.Avaliar(i, hoje)
	i.Aplicar(ruim)
	if i.Status != "BLOQUEADO" || i.ProximaCalibracao != nil {
		t.Fatalf("reprovada bloqueia: %+v", i)
	}
	if err := i.PodeMedir("", hoje); err == nil || !strings.Contains(err.Error(), "bloqueado") {
		t.Fatalf("bloqueado: %v", err)
	}
	// Calibração antiga registrada depois não mexe nas datas.
	velha := &Calibracao{Data: hoje.AddDate(0, -3, 0), Executor: "INTERNO", ErroEncontrado: dp("0.01")}
	_ = velha.Avaliar(i, hoje)
	i.Aplicar(velha)
	if i.Status != "BLOQUEADO" {
		t.Fatal("calibração antiga não desbloqueia")
	}
	// Ajustado e recalibrado: volta a ativo.
	nova := &Calibracao{Data: hoje, Executor: "EXTERNO", Certificado: sp("RBC 123"), ErroEncontrado: dp("0.005"), Ajustado: true}
	_ = nova.Avaliar(i, hoje)
	i.Aplicar(nova)
	if i.Status != "ATIVO" || i.MotivoStatus != nil {
		t.Fatalf("recalibrado: %+v", i)
	}
}
