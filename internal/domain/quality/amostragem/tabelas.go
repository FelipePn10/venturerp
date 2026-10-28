// Package amostragem implementa a amostragem por atributos da NBR 5426, que
// reproduz a ISO 2859-1: planos de amostragem simples indexados pelo NQA (nível
// de qualidade aceitável, o AQL da norma), para inspeção lote a lote.
//
// As tabelas foram conferidas contra a IS 2500-1:2000, idêntica à
// ISO 2859-1:1999: tabela 1 (letra-código) e tabelas 2-A, 2-B e 2-C (planos
// simples para inspeção normal, apertada e atenuada).
package amostragem

import (
	"strings"
)

// Nivel de inspeção (tabela 1). O nível geral II é o padrão da norma.
type Nivel string

const (
	NivelS1  Nivel = "S1"
	NivelS2  Nivel = "S2"
	NivelS3  Nivel = "S3"
	NivelS4  Nivel = "S4"
	NivelI   Nivel = "I"
	NivelII  Nivel = "II"
	NivelIII Nivel = "III"
)

var colunaDoNivel = map[Nivel]int{NivelS1: 0, NivelS2: 1, NivelS3: 2, NivelS4: 3, NivelI: 4, NivelII: 5, NivelIII: 6}

// faixasDeLote: limite superior de cada faixa de tamanho de lote da tabela 1 e
// a letra por nível (S-1, S-2, S-3, S-4, I, II, III).
var faixasDeLote = []struct {
	ate    int64
	letras string
}{
	{8, "AAAAAAB"},
	{15, "AAAAABC"},
	{25, "AABBBCD"},
	{50, "ABBCCDE"},
	{90, "BBCCCEF"},
	{150, "BBCDDFG"},
	{280, "BCDEEGH"},
	{500, "BCDEFHJ"},
	{1200, "CCEFGJK"},
	{3200, "CDEGHKL"},
	{10000, "CDFGJLM"},
	{35000, "CDFHKMN"},
	{150000, "DEGJLNP"},
	{500000, "DEGJMPQ"},
	{1 << 62, "DEHKNQR"},
}

// Severidade da inspeção (cláusula 9).
type Severidade string

const (
	Normal   Severidade = "NORMAL"
	Apertada Severidade = "APERTADA"
	Atenuada Severidade = "ATENUADA"
)

// NQAs da norma, do mais exigente ao mais folgado (colunas das tabelas 2).
var NQAs = []string{
	"0.010", "0.015", "0.025", "0.040", "0.065", "0.10", "0.15", "0.25", "0.40", "0.65",
	"1.0", "1.5", "2.5", "4.0", "6.5", "10", "15", "25", "40", "65",
	"100", "150", "250", "400", "650", "1000",
}

// letras na ordem das linhas das tabelas 2 (sem I e O, como na norma).
const letras = "ABCDEFGHJKLMNPQRS"

var tamanhoNormal = []int64{2, 3, 5, 8, 13, 20, 32, 50, 80, 125, 200, 315, 500, 800, 1250, 2000}
var tamanhoApertada = []int64{2, 3, 5, 8, 13, 20, 32, 50, 80, 125, 200, 315, 500, 800, 1250, 2000, 3150}
var tamanhoAtenuada = []int64{2, 2, 2, 3, 5, 8, 13, 20, 32, 50, 80, 125, 200, 315, 500, 800}

// celula de uma tabela 2: um plano (ac/re), uma seta ou vazio.
type celula struct {
	plano  bool
	ac, re int
	seta   int // +1 seta para baixo, -1 seta para cima, 0 sem seta
}

var (
	baixo = celula{seta: +1}
	cima  = celula{seta: -1}
	vazio = celula{}
)

func ar(ac, re int) celula { return celula{plano: true, ac: ac, re: re} }

// As tabelas normal e apertada seguem a diagonal da norma: a posição p na
// diagonal é coluna + linha − 14 (normal) e coluna + linha − 15 (apertada); a
// sequência abaixo é o que aparece em cada p. Antes do primeiro plano há setas
// para baixo; passado o último plano da linha, setas para cima. As linhas A a
// E vão mais longe na diagonal (30/31 e 44/45 na normal; 27/28 e 41/42 na
// apertada) do que as linhas de F em diante.
var (
	sequenciaNormal   = []celula{ar(0, 1), cima, baixo, ar(1, 2), ar(2, 3), ar(3, 4), ar(5, 6), ar(7, 8), ar(10, 11), ar(14, 15), ar(21, 22), ar(30, 31), ar(44, 45)}
	sequenciaApertada = []celula{ar(0, 1), baixo, baixo, ar(1, 2), ar(2, 3), ar(3, 4), ar(5, 6), ar(8, 9), ar(12, 13), ar(18, 19), ar(27, 28), ar(41, 42)}
	// Atenuada, linhas C em diante (as linhas A e B, que têm a mesma amostra
	// de 2 da C, são transcritas à parte).
	sequenciaAtenuada = []celula{ar(0, 1), cima, baixo, baixo, ar(1, 2), ar(2, 3), ar(3, 4), ar(5, 6), ar(6, 7), ar(8, 9), ar(10, 11), ar(14, 15), ar(21, 22)}
)

var tabelaNormal, tabelaApertada, tabelaAtenuada [][]celula

func init() {
	tabelaNormal = diagonal(16, sequenciaNormal, 14, 12, 10)
	tabelaApertada = diagonal(16, sequenciaApertada, 15, 11, 9)
	// Linha S da apertada: só existe o plano 1/2 de 3150 sob o NQA 0,025.
	linhaS := make([]celula, len(NQAs))
	linhaS[2] = ar(1, 2)
	tabelaApertada = append(tabelaApertada, linhaS)

	tabelaAtenuada = diagonal(16, sequenciaAtenuada, 14, 12, 10)
	// Linhas A e B da atenuada, transcritas da tabela 2-C.
	a := make([]celula, len(NQAs))
	b := make([]celula, len(NQAs))
	cauda := []celula{ar(1, 2), ar(2, 3), ar(3, 4), ar(5, 6), ar(7, 8), ar(10, 11), ar(14, 15), ar(21, 22), ar(30, 31)}
	for j := range NQAs {
		switch {
		case j < 14:
			a[j] = baixo
		case j == 14:
			a[j] = ar(0, 1)
		case j < 17:
			a[j] = baixo
		default:
			a[j] = cauda[j-17]
		}
		switch {
		case j < 13:
			b[j] = baixo
		case j == 13:
			b[j] = ar(0, 1)
		case j == 14:
			b[j] = cima
		case j < 17:
			b[j] = baixo
		default:
			b[j] = cauda[j-17]
		}
	}
	tabelaAtenuada[0], tabelaAtenuada[1] = a, b
}

// diagonal monta uma tabela de linhas × NQAs a partir da sequência da
// diagonal. maxAE e maxF são o último índice da sequência usado nas linhas A–E
// e nas linhas de F em diante.
func diagonal(linhas int, seq []celula, deslocamento, maxAE, maxF int) [][]celula {
	t := make([][]celula, linhas)
	for i := range t {
		t[i] = make([]celula, len(NQAs))
		limite := maxF
		if i <= 4 {
			limite = maxAE
		}
		for j := range NQAs {
			p := j + i - deslocamento
			switch {
			case p < 0:
				t[i][j] = baixo
			case p > limite || p >= len(seq):
				t[i][j] = cima
			default:
				t[i][j] = seq[p]
			}
		}
	}
	return t
}

func tabelaDe(s Severidade) ([][]celula, []int64) {
	switch s {
	case Apertada:
		return tabelaApertada, tamanhoApertada
	case Atenuada:
		return tabelaAtenuada, tamanhoAtenuada
	default:
		return tabelaNormal, tamanhoNormal
	}
}

func indiceDoNQA(nqa string) int {
	n := strings.ReplaceAll(strings.TrimSpace(nqa), ",", ".")
	for i, v := range NQAs {
		if v == n {
			return i
		}
	}
	// Aceita "0.01", "1", "4" etc.: compara pelo valor numérico normalizado.
	for i, v := range NQAs {
		if normalizar(v) == normalizar(n) {
			return i
		}
	}
	return -1
}

func normalizar(s string) string {
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	return s
}
