package entity

import (
	"sort"

	"github.com/shopspring/decimal"
)

// ToleranciaAcuracidade é a diferença aceita em cada classe ABC, em percentual.
// Item sem classe usa a da classe C.
type ToleranciaAcuracidade struct {
	A, B, C decimal.Decimal
}

// ContagemAcuracidade é uma contagem encerrada, de inventário ou contagem cíclica.
type ContagemAcuracidade struct {
	Mes         string
	ItemCode    int64
	Codigo      string
	Nome        string
	Classe      string
	WarehouseID int64
	Endereco    string
	Esperado    decimal.Decimal
	Contado     decimal.Decimal
	Custo       decimal.Decimal
}

// Acertou diz se a contagem ficou dentro da tolerância da classe.
func (c ContagemAcuracidade) Acertou(t ToleranciaAcuracidade) bool {
	pct := t.C
	switch c.Classe {
	case "A":
		pct = t.A
	case "B":
		pct = t.B
	}
	limite := c.Esperado.Abs().Mul(pct).Div(decimal.NewFromInt(100))
	return c.Contado.Sub(c.Esperado).Abs().LessThanOrEqual(limite)
}

// GrupoAcuracidade é o resultado de um recorte: mês ou classe.
type GrupoAcuracidade struct {
	Chave      string           `json:"key"`
	Contagens  int              `json:"counts"`
	Acertos    int              `json:"hits"`
	PctLinhas  decimal.Decimal  `json:"line_accuracy_pct"`
	PctValor   *decimal.Decimal `json:"value_accuracy_pct,omitempty"`
	Divergente decimal.Decimal  `json:"divergent_value"`
}

// DivergenciaRecorrente é o item ou endereço que erra mais de uma vez.
type DivergenciaRecorrente struct {
	ItemCode    *int64          `json:"item_code,omitempty"`
	Codigo      string          `json:"business_code,omitempty"`
	Nome        string          `json:"name,omitempty"`
	WarehouseID *int64          `json:"warehouse_id,omitempty"`
	Endereco    string          `json:"address,omitempty"`
	Contagens   int             `json:"counts"`
	Erros       int             `json:"misses"`
	Divergente  decimal.Decimal `json:"divergent_value"`
}

// ResultadoAcuracidade é o painel de acuracidade.
type ResultadoAcuracidade struct {
	Total     GrupoAcuracidade        `json:"total"`
	PorMes    []GrupoAcuracidade      `json:"by_month"`
	PorClasse []GrupoAcuracidade      `json:"by_class"`
	Itens     []DivergenciaRecorrente `json:"recurring_items"`
	Enderecos []DivergenciaRecorrente `json:"recurring_addresses"`
}

const limiteRecorrentes = 20

var cem = decimal.NewFromInt(100)

type somaAcuracidade struct {
	contagens, acertos int
	esperado, erro     decimal.Decimal
}

func (s *somaAcuracidade) somar(c ContagemAcuracidade, t ToleranciaAcuracidade) {
	s.contagens++
	if c.Acertou(t) {
		s.acertos++
	}
	s.esperado = s.esperado.Add(c.Esperado.Abs().Mul(c.Custo))
	s.erro = s.erro.Add(c.Contado.Sub(c.Esperado).Abs().Mul(c.Custo))
}

func (s somaAcuracidade) grupo(chave string) GrupoAcuracidade {
	g := GrupoAcuracidade{Chave: chave, Contagens: s.contagens, Acertos: s.acertos, Divergente: s.erro.Round(2)}
	if s.contagens > 0 {
		g.PctLinhas = decimal.NewFromInt(int64(s.acertos)).Mul(cem).Div(decimal.NewFromInt(int64(s.contagens))).Round(2)
	}
	// Sem valor esperado não há base: item novo achado na contagem não vira -∞%.
	if s.esperado.IsPositive() {
		pct := decimal.Max(decimal.NewFromInt(1).Sub(s.erro.Div(s.esperado)).Mul(cem), decimal.Zero).Round(2)
		g.PctValor = &pct
	}
	return g
}

// ApurarAcuracidade resume as contagens: linhas dentro da tolerância, valor
// acertado, evolução por mês e o que erra de forma recorrente. Recorrente é quem
// errou duas vezes ou mais.
func ApurarAcuracidade(contagens []ContagemAcuracidade, t ToleranciaAcuracidade) ResultadoAcuracidade {
	var total somaAcuracidade
	porMes := map[string]*somaAcuracidade{}
	porClasse := map[string]*somaAcuracidade{}
	type chaveEndereco struct {
		almox    int64
		endereco string
	}
	itens := map[int64]*DivergenciaRecorrente{}
	enderecos := map[chaveEndereco]*DivergenciaRecorrente{}
	acumular := func(d *DivergenciaRecorrente, c ContagemAcuracidade) {
		d.Contagens++
		if !c.Acertou(t) {
			d.Erros++
			d.Divergente = d.Divergente.Add(c.Contado.Sub(c.Esperado).Abs().Mul(c.Custo))
		}
	}
	for _, c := range contagens {
		total.somar(c, t)
		classe := c.Classe
		if classe == "" {
			classe = "Sem classe"
		}
		somarEm(porMes, c.Mes, c, t)
		somarEm(porClasse, classe, c, t)
		it := itens[c.ItemCode]
		if it == nil {
			code := c.ItemCode
			it = &DivergenciaRecorrente{ItemCode: &code, Codigo: c.Codigo, Nome: c.Nome}
			itens[c.ItemCode] = it
		}
		acumular(it, c)
		if c.Endereco != "" {
			k := chaveEndereco{c.WarehouseID, c.Endereco}
			en := enderecos[k]
			if en == nil {
				almox := c.WarehouseID
				en = &DivergenciaRecorrente{WarehouseID: &almox, Endereco: c.Endereco}
				enderecos[k] = en
			}
			acumular(en, c)
		}
	}
	out := ResultadoAcuracidade{
		Total: total.grupo("Total"), PorMes: grupos(porMes), PorClasse: grupos(porClasse),
		Itens: []DivergenciaRecorrente{}, Enderecos: []DivergenciaRecorrente{},
	}
	for _, d := range itens {
		if d.Erros >= 2 {
			out.Itens = append(out.Itens, *d)
		}
	}
	for _, d := range enderecos {
		if d.Erros >= 2 {
			out.Enderecos = append(out.Enderecos, *d)
		}
	}
	out.Itens, out.Enderecos = recorrentes(out.Itens), recorrentes(out.Enderecos)
	return out
}

func somarEm(m map[string]*somaAcuracidade, chave string, c ContagemAcuracidade, t ToleranciaAcuracidade) {
	if m[chave] == nil {
		m[chave] = &somaAcuracidade{}
	}
	m[chave].somar(c, t)
}

func grupos(m map[string]*somaAcuracidade) []GrupoAcuracidade {
	out := make([]GrupoAcuracidade, 0, len(m))
	for k, s := range m {
		out = append(out, s.grupo(k))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Chave < out[j].Chave })
	return out
}

func recorrentes(l []DivergenciaRecorrente) []DivergenciaRecorrente {
	sort.Slice(l, func(i, j int) bool {
		if l[i].Erros != l[j].Erros {
			return l[i].Erros > l[j].Erros
		}
		if !l[i].Divergente.Equal(l[j].Divergente) {
			return l[i].Divergente.GreaterThan(l[j].Divergente)
		}
		return l[i].Codigo+l[i].Endereco < l[j].Codigo+l[j].Endereco
	})
	for i := range l {
		l[i].Divergente = l[i].Divergente.Round(2)
	}
	if len(l) > limiteRecorrentes {
		return l[:limiteRecorrentes]
	}
	return l
}

// FaixaDeIdade é quanto do estoque está parado há tanto tempo.
type FaixaDeIdade struct {
	Faixa string          `json:"range"`
	Itens int             `json:"items"`
	Valor decimal.Decimal `json:"value"`
	Pct   decimal.Decimal `json:"value_pct"`
}

// FaixasDeIdade agrupa o saldo pelo tempo desde o último movimento real.
func FaixasDeIdade(itens []AnaliseItem) []FaixaDeIdade {
	faixas := []FaixaDeIdade{{Faixa: "Até 90 dias"}, {Faixa: "91 a 180 dias"}, {Faixa: "181 a 365 dias"}, {Faixa: "Mais de 1 ano"}, {Faixa: "Sem movimento registrado"}}
	total := decimal.Zero
	for _, a := range itens {
		if !a.Saldo.IsPositive() {
			continue
		}
		i := 4
		if a.DiasSemMovimento != nil {
			switch d := *a.DiasSemMovimento; {
			case d <= 90:
				i = 0
			case d <= 180:
				i = 1
			case d <= 365:
				i = 2
			default:
				i = 3
			}
		}
		faixas[i].Itens++
		faixas[i].Valor = faixas[i].Valor.Add(a.Valor)
		total = total.Add(a.Valor)
	}
	if total.IsPositive() {
		for i := range faixas {
			faixas[i].Pct = faixas[i].Valor.Mul(cem).Div(total).Round(2)
		}
	}
	return faixas
}
