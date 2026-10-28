package lote

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/quality/amostragem"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }
func dp(s string) *decimal.Decimal {
	v := d(s)
	return &v
}
func ip(v int64) *int64   { return &v }
func sp(v string) *string { return &v }
func bp(v bool) *bool     { return &v }

const (
	inspecao  = 10
	almox     = 20
	rejeicao  = 30
	sucata    = 40
	semDispos = 50
)

var almoxs = map[int64]Almoxarifado{
	inspecao:  {ID: inspecao, Localizacao: "INSPECAO"},
	almox:     {ID: almox, Localizacao: "INTERNO", Disposicao: true},
	rejeicao:  {ID: rejeicao, Localizacao: "REJEICAO"},
	sucata:    {ID: sucata, Localizacao: "INTERNO", Disposicao: false},
	semDispos: {ID: semDispos, Localizacao: "INTERNO", Disposicao: false},
}

// Lote de 100 com amostra de 3 (Ac 0, Re 1), uma variável e uma crítica.
func loteDeTeste() *Lote {
	return &Lote{
		ID: 1, Numero: 7, Status: Aberto, Quantidade: d("100"), WarehouseID: ip(inspecao),
		Plano: &amostragem.Plano{N: 3, Ac: 0, Re: 1},
		Caracteristicas: []Caracteristica{
			{ID: 1, Nome: "Espessura", Tipo: Variavel, Obrigatoria: true, Minimo: dp("2.9"), Maximo: dp("3.1")},
			{ID: 2, Nome: "Trinca", Tipo: Atributo, Obrigatoria: true, Critica: true},
			{ID: 3, Nome: "Etiqueta", Tipo: Atributo, Obrigatoria: false},
		},
	}
}

func todosConformes(l *Lote) {
	for i := 1; i <= 3; i++ {
		l.Resultados = append(l.Resultados, Resultado{CaracteristicaID: 1, AmostraIndice: i, Valor: dp("3.0"), Conforme: true},
			Resultado{CaracteristicaID: 2, AmostraIndice: i, Conforme: true})
	}
}

func TestConformidadePeloServidor(t *testing.T) {
	c := loteDeTeste().Caracteristicas[0]
	for v, quer := range map[string]bool{"2.9": true, "3.1": true, "3.0": true, "2.899": false, "3.1001": false} {
		ok, err := c.Conformidade(dp(v), bp(!quer)) // a tela mandar o contrário não importa
		if err != nil || ok != quer {
			t.Fatalf("%s: %v %v", v, ok, err)
		}
	}
	if _, err := c.Conformidade(nil, bp(true)); err == nil {
		t.Fatal("variável sem valor deveria ser recusada")
	}
	if _, err := loteDeTeste().Caracteristicas[1].Conformidade(nil, nil); err == nil {
		t.Fatal("atributo sem conforme deveria ser recusado")
	}
}

func TestAvaliacao(t *testing.T) {
	l := loteDeTeste()
	a := l.Avaliar()
	if a.Completa || len(a.Faltantes) != 2 {
		t.Fatalf("sem resultados: %+v", a)
	}
	todosConformes(l)
	a = l.Avaliar()
	if !a.Completa || a.NaoConformes != 0 || !a.AceitoPelaAmostragem {
		t.Fatalf("tudo conforme: %+v", a)
	}
	// Unidade 2 com espessura e etiqueta não conformes conta uma vez.
	l.Resultados[2].Conforme = false
	l.Resultados = append(l.Resultados, Resultado{CaracteristicaID: 3, AmostraIndice: 2, Conforme: false})
	a = l.Avaliar()
	if a.NaoConformes != 1 || a.AceitoPelaAmostragem {
		t.Fatalf("d conta unidades, não resultados: %+v", a)
	}
	// Crítica reprovada recusa mesmo com Ac folgado.
	l2 := loteDeTeste()
	l2.Plano.Ac, l2.Plano.Re = 5, 6
	todosConformes(l2)
	l2.Resultados[1].Conforme = false
	if a := l2.Avaliar(); a.AceitoPelaAmostragem || !a.CriticaReprovada {
		t.Fatalf("crítica reprovada: %+v", a)
	}
}

func TestDecisaoSomaELocais(t *testing.T) {
	l := loteDeTeste()
	todosConformes(l)
	ok := Decisao{Aceita: d("100"), DestinoAceito: ip(almox)}
	if err := l.PrepararDecisao(&ok, almoxs); err != nil || ok.Tipo != DecisaoAceito || ok.AceitoAmostragem == nil || !*ok.AceitoAmostragem {
		t.Fatalf("aceite total: %+v %v", ok, err)
	}
	casos := []struct {
		nome string
		dec  Decisao
		erro string
	}{
		{"soma menor", Decisao{Aceita: d("99"), DestinoAceito: ip(almox)}, "distribua o lote inteiro"},
		{"negativa", Decisao{Aceita: d("110"), Rejeitada: d("-10"), DestinoAceito: ip(almox)}, "negativas"},
		{"sem destino", Decisao{Aceita: d("100")}, "destino da quantidade aceita"},
		{"aceito para rejeição", Decisao{Aceita: d("100"), DestinoAceito: ip(rejeicao)}, "almoxarifado disponível"},
		{"aceito para a própria inspeção", Decisao{Aceita: d("100"), DestinoAceito: ip(inspecao)}, "próprio almoxarifado"},
		{"rejeitado para disponível", Decisao{Aceita: d("90"), Rejeitada: d("10"), DestinoAceito: ip(almox), DestinoRejeitado: ip(almox), Motivo: sp("x")}, "rejeição"},
		{"parcial sem motivo", Decisao{Aceita: d("90"), Rejeitada: d("10"), DestinoAceito: ip(almox), DestinoRejeitado: ip(rejeicao)}, "motivo"},
		{"destino de outra empresa", Decisao{Aceita: d("100"), DestinoAceito: ip(999)}, "não existe nesta empresa"},
		{"tratamento inválido", Decisao{Aceita: d("100"), DestinoAceito: ip(almox), Tratamento: sp("JOGAR FORA")}, "tratamento inválido"},
	}
	for _, c := range casos {
		dec := c.dec
		if err := loteDeTeste().copiaComResultados().PrepararDecisao(&dec, almoxs); err == nil || !strings.Contains(err.Error(), c.erro) {
			t.Errorf("%s: %v (esperado %q)", c.nome, err, c.erro)
		}
	}
	p := Decisao{Aceita: d("80"), Restrita: d("5"), Retrabalho: d("5"), Rejeitada: d("5"), Sucata: d("5"),
		DestinoAceito: ip(almox), DestinoRestrito: ip(almox), DestinoRetrabalho: ip(semDispos), DestinoRejeitado: ip(rejeicao), Motivo: sp("misto")}
	if err := loteDeTeste().copiaComResultados().PrepararDecisao(&p, almoxs); err != nil || p.Tipo != DecisaoParcial {
		t.Fatalf("parcial com sucata baixada: %+v %v", p, err)
	}
	if pernas := p.Pernas(); len(pernas) != 5 || pernas[4].Destino != nil {
		t.Fatalf("pernas: %+v", pernas)
	}
}

func (l *Lote) copiaComResultados() *Lote {
	todosConformes(l)
	return l
}

func TestAceitarExigeInspecaoCompletaMasRejeitarNao(t *testing.T) {
	l := loteDeTeste()
	l.Resultados = []Resultado{{CaracteristicaID: 1, AmostraIndice: 1, Valor: dp("3"), Conforme: true}}
	ac := Decisao{Aceita: d("100"), DestinoAceito: ip(almox)}
	if err := l.PrepararDecisao(&ac, almoxs); err == nil || !strings.Contains(err.Error(), "Espessura (1 de 3)") {
		t.Fatalf("aceitar incompleto: %v", err)
	}
	rej := Decisao{Rejeitada: d("100"), DestinoRejeitado: ip(rejeicao), Tratamento: sp("devolver"), Motivo: sp("avaria no transporte")}
	if err := l.PrepararDecisao(&rej, almoxs); err != nil || rej.Tipo != DecisaoRejeitado || *rej.Tratamento != "DEVOLVER" {
		t.Fatalf("rejeitar sem concluir é permitido: %+v %v", rej, err)
	}
}

func TestConcessaoParaAceitarLoteRecusado(t *testing.T) {
	l := loteDeTeste()
	todosConformes(l)
	l.Resultados[0].Conforme = false // espessura fora na unidade 1: d=1 > Ac 0
	sem := Decisao{Aceita: d("100"), DestinoAceito: ip(almox)}
	if err := l.PrepararDecisao(&sem, almoxs); err == nil || !strings.Contains(err.Error(), "CONCESSAO") {
		t.Fatalf("aceitar lote recusado sem concessão: %v", err)
	}
	com := Decisao{Aceita: d("100"), DestinoAceito: ip(almox), Tratamento: sp("CONCESSAO"), Motivo: sp("desvio aprovado pelo cliente")}
	if err := l.PrepararDecisao(&com, almoxs); err != nil || *com.AceitoAmostragem {
		t.Fatalf("concessão: %+v %v", com, err)
	}
	// Separar a parte ruim não precisa de concessão.
	sel := Decisao{Aceita: d("95"), Rejeitada: d("5"), DestinoAceito: ip(almox), DestinoRejeitado: ip(rejeicao), Tratamento: sp("SELECIONAR"), Motivo: sp("triagem 100%")}
	if err := loteDeTeste().copiaComResultados().PrepararDecisao(&sel, almoxs); err != nil {
		t.Fatal(err)
	}
}

func TestLotePuladoELoteDeProcesso(t *testing.T) {
	l := loteDeTeste()
	l.Pulado = true
	dec := Decisao{Aceita: d("100"), DestinoAceito: ip(almox)}
	if err := l.PrepararDecisao(&dec, almoxs); err != nil {
		t.Fatalf("skip-lot aceita sem resultados: %v", err)
	}
	p := loteDeTeste()
	p.WarehouseID = nil
	todosConformes(p)
	dec = Decisao{Aceita: d("100"), DestinoAceito: ip(almox)}
	if err := p.PrepararDecisao(&dec, almoxs); err != nil || dec.DestinoAceito != nil {
		t.Fatalf("processo não move estoque: %+v %v", dec, err)
	}
	l.Status = Decidido
	if err := l.PrepararDecisao(&Decisao{Aceita: d("100")}, almoxs); err == nil {
		t.Fatal("lote decidido não recebe outra decisão")
	}
}
