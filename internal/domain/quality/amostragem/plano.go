package amostragem

import (
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// Plano de amostragem simples: inspeciona N unidades; aceita o lote com até Ac
// não conformes e recusa com Re ou mais.
type Plano struct {
	Letra      string     `json:"letra"`
	Severidade Severidade `json:"severidade"`
	NQA        string     `json:"nqa"`
	N          int64      `json:"n"`
	Ac         int        `json:"ac"`
	Re         int        `json:"re"`
	// CemPorCento: a amostra alcança o tamanho do lote, então o lote inteiro é
	// inspecionado (N passa a ser o lote) com o mesmo critério de aceitação.
	CemPorCento bool `json:"cem_por_cento"`
	// LetraDaTabela é a letra do plano usado depois de seguir as setas — pode
	// ser diferente da letra do lote.
	LetraDaTabela string `json:"letra_da_tabela"`
}

// LetraCodigo devolve a letra da tabela 1 para o tamanho do lote e o nível.
func LetraCodigo(tamanhoLote int64, nivel Nivel) (string, error) {
	if tamanhoLote < 2 {
		return "", domainerr.NewValidation("o lote precisa ter ao menos 2 unidades para amostragem pela NBR 5426")
	}
	col, ok := colunaDoNivel[nivel]
	if !ok {
		return "", domainerr.NewValidation(fmt.Sprintf("nível de inspeção %q inválido (use S1, S2, S3, S4, I, II ou III)", nivel))
	}
	for _, f := range faixasDeLote {
		if tamanhoLote <= f.ate {
			return string(f.letras[col]), nil
		}
	}
	return "", domainerr.NewValidation("tamanho de lote fora da tabela 1")
}

// PlanoDaTabela resolve letra × NQA × severidade nas tabelas 2, seguindo as
// setas: seta para baixo usa o primeiro plano abaixo; para cima, o primeiro
// acima. Na última linha não há plano abaixo e vale o primeiro acima, como a
// norma desenha.
func PlanoDaTabela(letra, nqa string, sev Severidade) (Plano, error) {
	i := indiceDaLetra(letra)
	if i < 0 || (sev != Apertada && i > 15) {
		return Plano{}, domainerr.NewValidation(fmt.Sprintf("letra-código %q inválida", letra))
	}
	j := indiceDoNQA(nqa)
	if j < 0 {
		return Plano{}, domainerr.NewValidation(fmt.Sprintf("NQA %q não é um valor da norma", nqa))
	}
	tab, tamanhos := tabelaDe(sev)
	linha, ok := seguir(tab, i, j)
	if !ok {
		return Plano{}, domainerr.NewValidation("não há plano na tabela para essa combinação")
	}
	c := tab[linha][j]
	return Plano{
		Letra: letra, LetraDaTabela: string(letras[linha]), Severidade: sev, NQA: NQAs[j],
		N: tamanhos[linha], Ac: c.ac, Re: c.re,
	}, nil
}

func seguir(tab [][]celula, i, j int) (int, bool) {
	c := tab[i][j]
	if c.plano {
		return i, true
	}
	abaixo := func() (int, bool) {
		for k := i + 1; k < len(tab); k++ {
			if tab[k][j].plano {
				return k, true
			}
		}
		return 0, false
	}
	acima := func() (int, bool) {
		for k := i - 1; k >= 0; k-- {
			if tab[k][j].plano {
				return k, true
			}
		}
		return 0, false
	}
	// Seta sem plano no seu sentido (topo ou fundo da tabela) aponta, no
	// desenho da norma, para o plano do outro lado.
	primeiro, segundo := abaixo, acima
	if c.seta < 0 {
		primeiro, segundo = acima, abaixo
	}
	if k, ok := primeiro(); ok {
		return k, true
	}
	return segundo()
}

func indiceDaLetra(l string) int {
	if len(l) != 1 {
		return -1
	}
	for i := 0; i < len(letras); i++ {
		if letras[i] == l[0] {
			return i
		}
	}
	return -1
}

// PlanoParaLote: letra pela tabela 1, plano pela tabela 2 da severidade e, se
// a amostra alcança o lote, inspeção 100 %.
func PlanoParaLote(tamanhoLote int64, nivel Nivel, nqa string, sev Severidade) (Plano, error) {
	letra, err := LetraCodigo(tamanhoLote, nivel)
	if err != nil {
		return Plano{}, err
	}
	p, err := PlanoDaTabela(letra, nqa, sev)
	if err != nil {
		return Plano{}, err
	}
	if p.N >= tamanhoLote {
		p.N, p.CemPorCento = tamanhoLote, true
	}
	return p, nil
}

// Aceita diz se o lote é aceito com d não conformes na amostra.
func (p Plano) Aceita(d int) bool { return d <= p.Ac }

// NQAMaisApertado devolve o NQA um passo mais exigente (usado na pontuação de
// troca para atenuada); vazio quando já é o primeiro da série.
func NQAMaisApertado(nqa string) string {
	j := indiceDoNQA(nqa)
	if j <= 0 {
		return ""
	}
	return NQAs[j-1]
}
