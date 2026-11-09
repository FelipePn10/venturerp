// Package service resolve conversões de unidade como CAMINHO num grafo.
//
// Antes cada tela procurava um fator direto (ou a inversa dele) com o seu
// próprio SQL — cinco implementações, cada uma olhando uma parte diferente do
// cadastro. "Compro em KG, estoco em UN e a estrutura consome em MM" exigia
// cadastrar KG→MM à mão, mesmo com KG→UN e UN→MM já cadastrados.
//
// Aqui as conversões do lote, do item (com e sem máscara) e as padrão da
// empresa são arestas; a resposta é o menor caminho, e a fração é carregada
// exata (numerador × numerador / denominador × denominador) até o fim, para
// que 1/6000 × 6000 volte a ser 1 e não 0,9999999.
package service

import (
	"fmt"
	"sort"
	"strings"

	"github.com/shopspring/decimal"
)

// Origem diz de que cadastro veio uma aresta. A ordem é a de precedência: o
// mais específico ganha quando dois cadastros falam do mesmo par.
type Origem string

const (
	OrigemLote        Origem = "LOTE"
	OrigemItemMascara Origem = "ITEM_MASCARA"
	OrigemItem        Origem = "ITEM"
	// OrigemSegundaUnidade: o fator da segunda unidade do cadastro do item
	// (VENT0200). Perde para a conversão explícita da VSUP0110 do mesmo par e
	// ganha da padrão da empresa.
	OrigemSegundaUnidade Origem = "SEGUNDA_UNIDADE"
	// OrigemEmbalagem: o nível de embalagem do cadastro do item (VENT0200 ›
	// Embalagens e GTIN): "1 CX = 12 UN". É a mesma informação de uma
	// conversão; sem ela no grafo, a nota em CX ficava sem fator mesmo com a
	// caixa cadastrada no item.
	OrigemEmbalagem Origem = "EMBALAGEM"
	OrigemPadrao    Origem = "PADRAO"
)

func (o Origem) peso() int {
	switch o {
	case OrigemLote:
		return 0
	case OrigemItemMascara:
		return 1
	case OrigemItem:
		return 2
	case OrigemSegundaUnidade:
		return 3
	case OrigemEmbalagem:
		return 4
	default:
		return 5
	}
}

// Rotulo é o nome da origem como o usuário a conhece.
func (o Origem) Rotulo() string {
	switch o {
	case OrigemLote:
		return "lote"
	case OrigemItemMascara:
		return "item (máscara)"
	case OrigemItem:
		return "item"
	case OrigemSegundaUnidade:
		return "segunda unidade do item"
	case OrigemEmbalagem:
		return "embalagem do item"
	default:
		return "padrão da empresa"
	}
}

// Aresta: QtdDe unidades De = QtdPara unidades Para.
type Aresta struct {
	ID      int64
	De      string
	Para    string
	QtdDe   decimal.Decimal
	QtdPara decimal.Decimal
	Origem  Origem
	// Política de arredondamento (só conversões do item a têm).
	ArredondamentoPct decimal.Decimal
	Tolerancia        decimal.Decimal
	TipoTolerancia    string
}

// Fator: 1 De = Fator × Para.
func (a Aresta) Fator() decimal.Decimal {
	if !a.QtdDe.IsPositive() {
		return decimal.Zero
	}
	return a.QtdPara.DivRound(a.QtdDe, 10)
}

func (a Aresta) valida() bool {
	return a.De != "" && a.Para != "" && a.De != a.Para && a.QtdDe.IsPositive() && a.QtdPara.IsPositive()
}

// Passo é uma aresta percorrida, já no sentido do caminho.
type Passo struct {
	ID        int64
	De        string
	Para      string
	Numerador decimal.Decimal // QtdPara no sentido do passo
	Denomin   decimal.Decimal // QtdDe no sentido do passo
	Origem    Origem
	Invertido bool
	// Política de arredondamento herdada da aresta.
	ArredondamentoPct decimal.Decimal
	Tolerancia        decimal.Decimal
	TipoTolerancia    string
}

// Fator do passo: 1 De = Fator × Para.
func (p Passo) Fator() decimal.Decimal { return p.Numerador.DivRound(p.Denomin, 10) }

// Caminho é a resposta: 1 De = Numerador/Denominador × Para.
type Caminho struct {
	De          string
	Para        string
	Numerador   decimal.Decimal
	Denominador decimal.Decimal
	Passos      []Passo
}

// Fator com dez casas (a escala das colunas de fator).
func (c Caminho) Fator() decimal.Decimal {
	if c.Denominador.IsZero() {
		return decimal.Zero
	}
	return c.Numerador.DivRound(c.Denominador, 10)
}

// Converter aplica a fração inteira de uma vez: q × N / D.
func (c Caminho) Converter(q decimal.Decimal) decimal.Decimal {
	if c.Denominador.IsZero() {
		return decimal.Zero
	}
	return q.Mul(c.Numerador).DivRound(c.Denominador, 10)
}

// ConverterPreco: preço por De → preço por Para (p × D / N).
func (c Caminho) ConverterPreco(p decimal.Decimal) decimal.Decimal {
	if c.Numerador.IsZero() {
		return decimal.Zero
	}
	return p.Mul(c.Denominador).DivRound(c.Numerador, 10)
}

// OrigemPrincipal: a origem mais específica entre os passos (o que decide se
// a conversão é "do item" ou só "padrão").
func (c Caminho) OrigemPrincipal() Origem {
	if len(c.Passos) == 0 {
		return OrigemItem
	}
	melhor := c.Passos[0].Origem
	for _, p := range c.Passos[1:] {
		if p.Origem.peso() < melhor.peso() {
			melhor = p.Origem
		}
	}
	return melhor
}

// Explicacao: "1 KG = 0,0425531915 UN (item) · 1 UN = 6000 MM (item)".
func (c Caminho) Explicacao() string {
	if len(c.Passos) == 0 {
		return fmt.Sprintf("%s = %s", c.De, c.Para)
	}
	partes := make([]string, 0, len(c.Passos))
	for _, p := range c.Passos {
		partes = append(partes, fmt.Sprintf("1 %s = %s %s (%s)", p.De, Numero(p.Fator()), p.Para, p.Origem.Rotulo()))
	}
	return strings.Join(partes, " · ")
}

// Numero formata sem zeros à direita e com vírgula decimal.
func Numero(d decimal.Decimal) string {
	s := d.String()
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return strings.Replace(s, ".", ",", 1)
}

// MaxPassos limita o caminho: conversão que precisa de mais de quatro saltos
// é sinal de cadastro confuso, não de engenharia.
const MaxPassos = 4

type chavePar struct{ a, b string }

func par(a, b string) chavePar {
	if a > b {
		a, b = b, a
	}
	return chavePar{a, b}
}

// deduplicar mantém, para cada par de unidades (em qualquer sentido), só a
// aresta mais específica. Lote ganha do item, item com máscara ganha do item
// genérico, que ganha da padrão.
func deduplicar(arestas []Aresta) []Aresta {
	melhor := map[chavePar]Aresta{}
	ordem := []chavePar{}
	for _, a := range arestas {
		a.De, a.Para = Normalizar(a.De), Normalizar(a.Para)
		if !a.valida() {
			continue
		}
		k := par(a.De, a.Para)
		atual, ok := melhor[k]
		if !ok {
			ordem = append(ordem, k)
			melhor[k] = a
			continue
		}
		if a.Origem.peso() < atual.Origem.peso() {
			melhor[k] = a
		}
	}
	out := make([]Aresta, 0, len(ordem))
	for _, k := range ordem {
		out = append(out, melhor[k])
	}
	return out
}

// Normalizar põe o código da unidade na forma do catálogo (maiúsculas, sem
// espaços). Sinônimos de texto (UND, PÇ) são outro assunto: ver NormalizarTexto.
func Normalizar(u string) string { return strings.ToUpper(strings.TrimSpace(u)) }

type estado struct {
	unidade string
	custo   int // soma dos pesos de origem: desempate entre caminhos do mesmo tamanho
	passos  []Passo
}

// Resolver encontra o menor caminho de `de` para `para`. Entre caminhos com o
// mesmo número de passos, ganha o que usa cadastros mais específicos.
func Resolver(arestas []Aresta, de, para string) (Caminho, bool) {
	de, para = Normalizar(de), Normalizar(para)
	um := decimal.NewFromInt(1)
	if de == "" || para == "" {
		return Caminho{}, false
	}
	if de == para {
		return Caminho{De: de, Para: para, Numerador: um, Denominador: um}, true
	}
	adj := map[string][]Passo{}
	for _, a := range deduplicar(arestas) {
		ida := Passo{ID: a.ID, De: a.De, Para: a.Para, Numerador: a.QtdPara, Denomin: a.QtdDe, Origem: a.Origem,
			ArredondamentoPct: a.ArredondamentoPct, Tolerancia: a.Tolerancia, TipoTolerancia: a.TipoTolerancia}
		volta := ida
		volta.De, volta.Para, volta.Numerador, volta.Denomin, volta.Invertido = a.Para, a.De, a.QtdDe, a.QtdPara, true
		adj[a.De] = append(adj[a.De], ida)
		adj[a.Para] = append(adj[a.Para], volta)
	}
	for k := range adj {
		sort.SliceStable(adj[k], func(i, j int) bool { return adj[k][i].Origem.peso() < adj[k][j].Origem.peso() })
	}
	// Busca em largura por nível; em cada nível guarda o melhor custo por nó.
	fronteira := []estado{{unidade: de}}
	visitado := map[string]bool{de: true}
	for nivel := 0; nivel < MaxPassos && len(fronteira) > 0; nivel++ {
		melhores := map[string]estado{}
		for _, e := range fronteira {
			for _, p := range adj[e.unidade] {
				if visitado[p.Para] {
					continue
				}
				passos := append(append([]Passo{}, e.passos...), p)
				n := estado{unidade: p.Para, custo: e.custo + p.Origem.peso(), passos: passos}
				if atual, ok := melhores[p.Para]; !ok || n.custo < atual.custo {
					melhores[p.Para] = n
				}
			}
		}
		if alvo, ok := melhores[para]; ok {
			return montar(de, para, alvo.passos), true
		}
		fronteira = fronteira[:0]
		chaves := make([]string, 0, len(melhores))
		for k := range melhores {
			chaves = append(chaves, k)
		}
		sort.Strings(chaves)
		for _, k := range chaves {
			visitado[k] = true
			fronteira = append(fronteira, melhores[k])
		}
	}
	return Caminho{}, false
}

func montar(de, para string, passos []Passo) Caminho {
	num, den := decimal.NewFromInt(1), decimal.NewFromInt(1)
	for _, p := range passos {
		num = num.Mul(p.Numerador)
		den = den.Mul(p.Denomin)
	}
	return Caminho{De: de, Para: para, Numerador: num, Denominador: den, Passos: passos}
}

// Conflito descreve um cadastro novo que contradiz o que o grafo já sabe.
type Conflito struct {
	Existente Caminho
	FatorNovo decimal.Decimal
}

func (c Conflito) Error() string {
	return fmt.Sprintf("o cadastro já leva %s a %s por outro caminho com fator %s (%s), e a conversão informada diz %s — corrija uma das duas",
		c.Existente.De, c.Existente.Para, Numero(c.Existente.Fator()), c.Existente.Explicacao(), Numero(c.FatorNovo))
}

// ToleranciaConsistencia: diferença relativa aceita entre o fator novo e o
// caminho que já existe (arredondamentos de cadastro, como 25,4 × 39,37).
var ToleranciaConsistencia = decimal.RequireFromString("0.0001")

// VerificarConsistencia: a aresta nova não pode contradizer um caminho que já
// existe por OUTRAS arestas. Cadastrar UN→KG = 23 quando UN→M = 6 e M→KG = 4
// já dizem 24 é um erro que só apareceria no inventário.
//
// Arestas do mesmo par da nova (a versão que ela substitui) são ignoradas, e
// arestas padrão também: elas valem para todos os itens e não "contradizem" um
// fator específico — a conversão do item é justamente a exceção.
func VerificarConsistencia(existentes []Aresta, nova Aresta) *Conflito {
	nova.De, nova.Para = Normalizar(nova.De), Normalizar(nova.Para)
	k := par(nova.De, nova.Para)
	outras := make([]Aresta, 0, len(existentes))
	for _, a := range existentes {
		if par(Normalizar(a.De), Normalizar(a.Para)) == k {
			continue
		}
		outras = append(outras, a)
	}
	c, ok := Resolver(outras, nova.De, nova.Para)
	if !ok || c.OrigemPrincipal() == OrigemPadrao {
		return nil
	}
	fNovo := nova.Fator()
	fExist := c.Fator()
	if fExist.IsZero() {
		return nil
	}
	dif := fNovo.Sub(fExist).Abs().DivRound(fExist, 10)
	if dif.GreaterThan(ToleranciaConsistencia) {
		return &Conflito{Existente: c, FatorNovo: fNovo}
	}
	return nil
}

// PoliticaDeArredondamento devolve a política da primeira conversão do item
// (ou do lote) no caminho; as padrão não têm política.
func (c Caminho) PoliticaDeArredondamento() (arred, tol decimal.Decimal, tipo string) {
	for _, p := range c.Passos {
		if p.Origem != OrigemPadrao {
			tipo = p.TipoTolerancia
			if tipo == "" {
				tipo = "VALUE"
			}
			return p.ArredondamentoPct, p.Tolerancia, tipo
		}
	}
	return decimal.Zero, decimal.Zero, "VALUE"
}
