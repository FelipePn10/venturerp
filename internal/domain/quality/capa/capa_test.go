package capa

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func sp(s string) *string { return &s }

func nova(m Metodo) *Capa {
	c := &Capa{Metodo: m, Origem: "INTERNA", Titulo: "Rebarba na furação", Responsavel: uuid.New()}
	if err := c.ValidarAbertura(); err != nil {
		panic(err)
	}
	return c
}

func avancarOuFalhar(t *testing.T, c *Capa, para string) {
	t.Helper()
	if _, p, err := c.Avancar(); err != nil || p != para {
		t.Fatalf("avançar para %s: %s %v (pendências %v)", para, p, err, c.Pendencias())
	}
}

func TestAbertura(t *testing.T) {
	casos := []Capa{
		{Metodo: "X", Origem: "INTERNA", Titulo: "t", Responsavel: uuid.New()},
		{Metodo: Metodo8D, Origem: "NC", Titulo: "t", Responsavel: uuid.New()},
		{Metodo: Metodo8D, Origem: "FORNECEDOR", Titulo: "t", Responsavel: uuid.New()},
		{Metodo: Metodo8D, Origem: "INTERNA", Titulo: " ", Responsavel: uuid.New()},
		{Metodo: Metodo8D, Origem: "INTERNA", Titulo: "t"},
	}
	for i, c := range casos {
		if err := c.ValidarAbertura(); err == nil {
			t.Errorf("caso %d deveria ser recusado", i)
		}
	}
	c := nova(Metodo8D)
	if c.Etapa != "D0" || c.Status != Aberta || c.Ciclo != 1 || c.Tipo != "CORRETIVA" {
		t.Fatalf("abertura: %+v", c)
	}
}

// O 8D inteiro: cada etapa só avança com o seu conteúdo.
func Test8DCompleto(t *testing.T) {
	c := nova(Metodo8D)
	avancarOuFalhar(t, c, "D1")
	if _, _, err := c.Avancar(); err == nil || !strings.Contains(err.Error(), "líder") {
		t.Fatalf("D1 sem equipe: %v", err)
	}
	c.Equipe = []Membro{{UserID: uuid.New(), Papel: "LIDER"}, {UserID: uuid.New(), Papel: "MEMBRO"}}
	avancarOuFalhar(t, c, "D2")
	c.Problema = Problema{OQue: "rebarba", Onde: "furo Ø8", Quando: "turno B"}
	if _, _, err := c.Avancar(); err == nil || !strings.Contains(err.Error(), "quanto") {
		t.Fatalf("D2 sem quanto: %v", err)
	}
	c.Problema.Quanto = "12% das peças"
	avancarOuFalhar(t, c, "D3")
	if _, _, err := c.Avancar(); err == nil || !strings.Contains(err.Error(), "contenção") {
		t.Fatalf("D3 sem contenção: %v", err)
	}
	c.Acoes = append(c.Acoes, Acao{ID: 1, Tipo: "CONTENCAO", Descricao: "segregar estoque", Status: "PENDENTE", Ciclo: 1})
	avancarOuFalhar(t, c, "D4")
	c.MetodoCausa = sp("5PORQUES")
	c.CincoPorques = []string{"broca gasta", "troca por tempo"}
	c.CausaRaiz = sp("vida da broca não controlada")
	if _, _, err := c.Avancar(); err == nil || !strings.Contains(err.Error(), "3 porquês") || !strings.Contains(err.Error(), "ponto de escape") {
		t.Fatalf("D4 incompleto: %v", err)
	}
	c.CincoPorques = append(c.CincoPorques, "sem contador de golpes")
	c.PontoDeEscape = sp("inspeção visual sem lupa")
	avancarOuFalhar(t, c, "D5")
	if _, _, err := c.Avancar(); err == nil {
		t.Fatal("D5 sem ação corretiva")
	}
	c.Acoes = append(c.Acoes, Acao{ID: 2, Tipo: "CORRETIVA", Descricao: "contador de vida da ferramenta", Status: "PENDENTE", Ciclo: 1})
	avancarOuFalhar(t, c, "D6")
	if _, _, err := c.Avancar(); err == nil || !strings.Contains(err.Error(), "contador de vida") || !strings.Contains(err.Error(), "segregar") {
		t.Fatalf("D6 com ações abertas: %v", err)
	}
	for i := range c.Acoes {
		c.Acoes[i].Status = "CONCLUIDA"
	}
	avancarOuFalhar(t, c, "D7")
	c.Prevencao = sp("vida da ferramenta no plano de manutenção")
	avancarOuFalhar(t, c, "D8")
	if _, _, err := c.Avancar(); err == nil {
		t.Fatal("D8 não avança: encerra")
	}
	if _, err := c.Encerrar(true, " ", nil, uuid.New(), time.Now()); err == nil {
		t.Fatal("encerrar sem verificação")
	}
	ok, err := c.Encerrar(true, "3 lotes seguintes sem rebarba", sp("controlar vida de toda ferramenta"), uuid.New(), time.Now())
	if err != nil || !ok || c.Status != Encerrada || c.ClosedAt == nil {
		t.Fatalf("encerramento: %v %+v", err, c)
	}
	if _, _, err := c.Avancar(); err == nil {
		t.Fatal("CAPA encerrada não avança")
	}
}

func TestIneficazVoltaACausaNumCicloNovo(t *testing.T) {
	c := nova(MetodoSimples)
	c.Problema.OQue = "trinca na solda"
	c.MetodoCausa = sp("ISHIKAWA")
	c.Ishikawa = map[string][]string{"METODO": {"pré-aquecimento não feito"}}
	c.CausaRaiz = sp("sem pré-aquecimento")
	avancarOuFalhar(t, c, "ACAO")
	c.Acoes = []Acao{{ID: 1, Tipo: "CORRETIVA", Descricao: "IT de pré-aquecimento", Status: "CONCLUIDA", Ciclo: 1}}
	avancarOuFalhar(t, c, "VERIFICACAO")
	if _, err := (&Capa{Metodo: MetodoSimples, Status: Aberta, Etapa: "ACAO"}).Encerrar(true, "x", nil, uuid.Nil, time.Now()); err == nil {
		t.Fatal("só se encerra na etapa final")
	}
	ok, err := c.Encerrar(false, "trinca voltou no lote 12", nil, uuid.New(), time.Now())
	if err != nil || ok || c.Etapa != "ANALISE" || c.Ciclo != 2 || c.CausaRaiz != nil || c.Status != Aberta {
		t.Fatalf("ineficaz: %v %+v", err, c)
	}
	// A ação do ciclo 1 não conta para o ciclo 2.
	c.CausaRaiz = sp("material com carbono alto")
	avancarOuFalhar(t, c, "ACAO")
	if _, _, err := c.Avancar(); err == nil {
		t.Fatal("o ciclo 2 precisa de ação corretiva própria")
	}
}

func TestIshikawaVazioEContencaoDispensada(t *testing.T) {
	c := nova(MetodoSimples)
	c.Problema.OQue = "x"
	c.MetodoCausa = sp("ISHIKAWA")
	c.CausaRaiz = sp("y")
	if _, _, err := c.Avancar(); err == nil || !strings.Contains(err.Error(), "Ishikawa") {
		t.Fatalf("Ishikawa vazio: %v", err)
	}
	d := nova(Metodo8D)
	d.Etapa = "D3"
	d.ContencaoDispensada = sp("peça ainda não expedida, nada em campo")
	avancarOuFalhar(t, d, "D4")
}

func TestValidarAcao(t *testing.T) {
	if err := ValidarAcao(&Acao{Tipo: "CORRETIVA", Descricao: "x", Responsavel: uuid.New()}); err == nil {
		t.Fatal("ação sem prazo")
	}
	if err := ValidarAcao(&Acao{Tipo: "OUTRA", Descricao: "x", Responsavel: uuid.New(), Prazo: time.Now()}); err == nil {
		t.Fatal("tipo inválido")
	}
	if err := ValidarAcao(&Acao{Tipo: "CONTENCAO", Descricao: " separar ", Responsavel: uuid.New(), Prazo: time.Now()}); err != nil {
		t.Fatal(err)
	}
}
