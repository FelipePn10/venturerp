package amostragem

import "testing"

// Valores lidos das tabelas 1, 2-A, 2-B e 2-C da IS 2500-1:2000 (= ISO
// 2859-1:1999), não derivados da fórmula da diagonal: o teste é a conferência
// independente da transcrição.
func TestLetraCodigo(t *testing.T) {
	casos := []struct {
		lote  int64
		nivel Nivel
		letra string
	}{
		{2, NivelII, "A"}, {8, NivelIII, "B"}, {50, NivelII, "D"}, {51, NivelII, "E"},
		{1000, NivelII, "J"}, {1201, NivelII, "K"}, {600, NivelS3, "E"}, {10000, NivelI, "J"},
		{35001, NivelS1, "D"}, {500001, NivelIII, "R"}, {150000, NivelS4, "J"},
	}
	for _, c := range casos {
		got, err := LetraCodigo(c.lote, c.nivel)
		if err != nil || got != c.letra {
			t.Errorf("lote %d nível %s = %q (%v), esperado %q", c.lote, c.nivel, got, err, c.letra)
		}
	}
	if _, err := LetraCodigo(1, NivelII); err == nil {
		t.Error("lote de 1 unidade deveria ser recusado")
	}
	if _, err := LetraCodigo(10, "IV"); err == nil {
		t.Error("nível inválido deveria ser recusado")
	}
}

func TestPlanosDasTabelas(t *testing.T) {
	casos := []struct {
		letra, nqa string
		sev        Severidade
		n          int64
		ac, re     int
	}{
		// 2-A normal
		{"A", "6.5", Normal, 2, 0, 1},
		{"A", "10", Normal, 5, 1, 2},       // seta longa até C
		{"A", "0.010", Normal, 1250, 0, 1}, // seta longa até Q
		{"R", "0.010", Normal, 1250, 0, 1}, // seta para cima
		{"R", "0.015", Normal, 800, 0, 1},  // sem plano abaixo: primeiro acima
		{"K", "1.0", Normal, 125, 3, 4},
		{"M", "0.65", Normal, 315, 5, 6},
		{"G", "2.5", Normal, 32, 2, 3},
		{"E", "150", Normal, 13, 30, 31},
		{"E", "250", Normal, 13, 44, 45},
		{"E", "400", Normal, 8, 44, 45}, // seta para cima até D
		{"F", "100", Normal, 13, 21, 22},
		{"J", "4.0", Normal, 80, 7, 8},
		{"N", "0.10", Normal, 500, 1, 2},
		// 2-B apertada
		{"K", "1.0", Apertada, 125, 2, 3},
		{"A", "15", Apertada, 5, 1, 2},
		{"R", "0.015", Apertada, 1250, 0, 1},
		{"R", "0.025", Apertada, 3150, 1, 2},
		{"E", "250", Apertada, 13, 41, 42},
		{"F", "100", Apertada, 13, 18, 19},
		{"H", "2.5", Apertada, 50, 2, 3},
		// 2-C atenuada
		{"A", "6.5", Atenuada, 2, 0, 1},
		{"A", "10", Atenuada, 3, 1, 2},
		{"B", "6.5", Atenuada, 2, 0, 1},
		{"B", "150", Atenuada, 2, 7, 8},
		{"C", "150", Atenuada, 2, 8, 9},
		{"K", "1.0", Atenuada, 50, 2, 3},
		{"R", "0.025", Atenuada, 200, 0, 1},
		{"Q", "0.025", Atenuada, 200, 0, 1},
		{"Q", "0.040", Atenuada, 800, 1, 2},
		{"F", "100", Atenuada, 5, 10, 11},
		{"D", "400", Atenuada, 3, 21, 22},
	}
	for _, c := range casos {
		p, err := PlanoDaTabela(c.letra, c.nqa, c.sev)
		if err != nil {
			t.Errorf("%s %s %s: %v", c.letra, c.nqa, c.sev, err)
			continue
		}
		if p.N != c.n || p.Ac != c.ac || p.Re != c.re {
			t.Errorf("%s NQA %s %s = n %d Ac %d Re %d, esperado n %d Ac %d Re %d",
				c.letra, c.nqa, c.sev, p.N, p.Ac, p.Re, c.n, c.ac, c.re)
		}
	}
}

// Toda combinação válida resolve para um plano com Re > Ac.
func TestTodaCombinacaoTemPlano(t *testing.T) {
	for _, sev := range []Severidade{Normal, Apertada, Atenuada} {
		for i := 0; i < 16; i++ {
			for _, nqa := range NQAs {
				p, err := PlanoDaTabela(string(letras[i]), nqa, sev)
				if err != nil || p.Re <= p.Ac || p.N < 2 {
					t.Fatalf("%c %s %s: %+v %v", letras[i], nqa, sev, p, err)
				}
			}
		}
	}
}

func TestNQAComVirgulaEFormatosEquivalentes(t *testing.T) {
	for _, nqa := range []string{"0,65", "0.65", "0.650"} {
		if p, err := PlanoDaTabela("K", nqa, Normal); err != nil || p.Ac != 2 {
			t.Fatalf("NQA %q: %+v %v", nqa, p, err)
		}
	}
	for _, nqa := range []string{"1", "1,0", "4", "0.01"} {
		if _, err := PlanoDaTabela("K", nqa, Normal); err != nil {
			t.Fatalf("NQA %q: %v", nqa, err)
		}
	}
	if _, err := PlanoDaTabela("K", "0.5", Normal); err == nil {
		t.Fatal("0,5 não é NQA da norma")
	}
}

func TestPlanoParaLoteCemPorCento(t *testing.T) {
	// Lote 6, nível II → A; NQA 0,65 leva à amostra de 20 (F), maior que o lote.
	p, err := PlanoParaLote(6, NivelII, "0.65", Normal)
	if err != nil {
		t.Fatal(err)
	}
	if !p.CemPorCento || p.N != 6 || p.Ac != 0 {
		t.Fatalf("esperado 100 %% do lote de 6 com Ac 0: %+v", p)
	}
	p, err = PlanoParaLote(1000, NivelII, "1.0", Normal)
	if err != nil || p.CemPorCento || p.Letra != "J" || p.N != 80 || p.Ac != 2 {
		t.Fatalf("lote 1000 NQA 1,0: %+v %v", p, err)
	}
}

func TestTrocaNormalParaApertadaEVolta(t *testing.T) {
	e := Novo()
	regras := Regras{ProducaoRegular: true}
	e, tr := e.Registrar(Resultado{Aceito: false, Ac: 2}, regras)
	if tr != nil || e.Severidade != Normal {
		t.Fatalf("uma recusa não troca: %+v", e)
	}
	e, _ = e.Registrar(Resultado{Aceito: true, Ac: 2}, regras)
	e, tr = e.Registrar(Resultado{Aceito: false, Ac: 2}, regras)
	if tr == nil || e.Severidade != Apertada {
		t.Fatalf("2 recusas em 3 lotes trocam para apertada: %+v", e)
	}
	for k := 0; k < 4; k++ {
		e, tr = e.Registrar(Resultado{Aceito: true, Ac: 1}, regras)
		if tr != nil {
			t.Fatalf("só 5 aceitos seguidos voltam a normal (lote %d)", k+1)
		}
	}
	e, tr = e.Registrar(Resultado{Aceito: true, Ac: 1}, regras)
	if tr == nil || e.Severidade != Normal || e.Pontuacao != 0 {
		t.Fatalf("5 aceitos voltam a normal com pontuação zerada: %+v", e)
	}
}

func TestDuasRecusasForaDaJanelaDeCincoNaoTrocam(t *testing.T) {
	e := Novo()
	r := Regras{ProducaoRegular: true}
	e, _ = e.Registrar(Resultado{Aceito: false}, r)
	for k := 0; k < 4; k++ {
		e, _ = e.Registrar(Resultado{Aceito: true}, r)
	}
	e, tr := e.Registrar(Resultado{Aceito: false}, r)
	if tr != nil || e.Severidade != Normal {
		t.Fatalf("a 1ª recusa saiu da janela de 5: %+v", e)
	}
}

func TestPontuacaoLevaAAtenuadaSoComAutorizacao(t *testing.T) {
	e := Novo()
	// Ac ≥ 2 e aceito com o NQA mais apertado: +3 por lote → 10 lotes = 30.
	for k := 0; k < 9; k++ {
		e, _ = e.Registrar(Resultado{Aceito: true, Ac: 3, AceitoComNQAMaisApertado: true}, Regras{ProducaoRegular: true})
	}
	if e.Pontuacao != 27 || e.Severidade != Normal {
		t.Fatalf("9 lotes = 27 pontos: %+v", e)
	}
	sem, tr := e.Registrar(Resultado{Aceito: true, Ac: 3, AceitoComNQAMaisApertado: true}, Regras{ProducaoRegular: true})
	if tr != nil || sem.Pontuacao != 30 {
		t.Fatalf("sem autorização a pontuação chega a 30 mas não troca: %+v", sem)
	}
	com, tr := e.Registrar(Resultado{Aceito: true, Ac: 3, AceitoComNQAMaisApertado: true}, Regras{PermiteAtenuada: true, ProducaoRegular: true})
	if tr == nil || com.Severidade != Atenuada {
		t.Fatalf("com autorização vai para atenuada: %+v", com)
	}
	// Aceito, mas não com o NQA mais apertado: zera.
	z, _ := e.Registrar(Resultado{Aceito: true, Ac: 3}, Regras{ProducaoRegular: true})
	if z.Pontuacao != 0 {
		t.Fatalf("Ac ≥ 2 sem passar no NQA mais apertado zera: %d", z.Pontuacao)
	}
	// Ac 0/1: +2 por aceito.
	d, _ := Novo().Registrar(Resultado{Aceito: true, Ac: 1}, Regras{})
	if d.Pontuacao != 2 {
		t.Fatalf("Ac 1 aceito soma 2: %d", d.Pontuacao)
	}
}

func TestAtenuadaVoltaANormal(t *testing.T) {
	e := Estado{Severidade: Atenuada}
	if n, tr := e.Registrar(Resultado{Aceito: false}, Regras{ProducaoRegular: true}); tr == nil || n.Severidade != Normal {
		t.Fatalf("recusa em atenuada volta a normal: %+v", n)
	}
	if n, tr := e.Registrar(Resultado{Aceito: true}, Regras{ProducaoRegular: false}); tr == nil || n.Severidade != Normal {
		t.Fatalf("produção irregular volta a normal: %+v", n)
	}
}

func TestSuspensaoComCincoRecusasEmApertada(t *testing.T) {
	e := Estado{Severidade: Apertada}
	r := Regras{ProducaoRegular: true}
	var tr *Transicao
	// Recusas intercaladas com aceitos: não chegam a 5 aceitos seguidos (que
	// voltariam a normal) e acumulam até a 5ª.
	for k := 0; k < 4; k++ {
		e, tr = e.Registrar(Resultado{Aceito: false}, r)
		if e.Suspensa {
			t.Fatalf("suspensão só na 5ª recusa (lote %d)", k+1)
		}
		e, _ = e.Registrar(Resultado{Aceito: true}, r)
	}
	e, tr = e.Registrar(Resultado{Aceito: false}, r)
	if tr == nil || !tr.Suspensao || !e.Suspensa {
		t.Fatalf("5 recusas acumuladas suspendem: %+v", e)
	}
	l := e.Liberar()
	if l.Suspensa || l.Severidade != Apertada || l.RecusasEmApertada != 0 {
		t.Fatalf("liberação recomeça em apertada: %+v", l)
	}
}

func TestSkipLot(t *testing.T) {
	s := SkipLot{AposAceitos: 3, Frequencia: 4}
	e := Novo()
	for k := 0; k < 3; k++ {
		if !e.DeveInspecionar(s) {
			t.Fatalf("antes de 3 aceitos todo lote é inspecionado (lote %d)", k+1)
		}
		e, _ = e.Registrar(Resultado{Aceito: true, Ac: 1}, Regras{ProducaoRegular: true})
	}
	// Agora 1 a cada 4: pula 3, inspeciona o 4º.
	for k := 0; k < 3; k++ {
		if e.DeveInspecionar(s) {
			t.Fatalf("lote %d deveria ser pulado", k+1)
		}
		e = e.Pular()
	}
	if !e.DeveInspecionar(s) {
		t.Fatal("o 4º lote é inspecionado")
	}
	e, _ = e.Registrar(Resultado{Aceito: false, Ac: 1}, Regras{ProducaoRegular: true})
	if !e.DeveInspecionar(s) {
		t.Fatal("recusa volta à inspeção de todo lote")
	}
	if !(Estado{Severidade: Apertada, AceitosSeguidos: 10}).DeveInspecionar(s) {
		t.Fatal("apertada nunca pula")
	}
	if !Novo().DeveInspecionar(SkipLot{}) {
		t.Fatal("sem regra, todo lote é inspecionado")
	}
}
