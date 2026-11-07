// Package pdm é o padrão descritivo de materiais: o modelo de atributos de cada
// par grupo+modificador e a descrição técnica gerada a partir dele. Com o padrão,
// dois cadastradores escrevem o mesmo item do mesmo jeito, e o duplicado aparece
// pela assinatura em vez de se esconder atrás da grafia.
package pdm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

// Tipos de valor de um atributo.
const (
	TipoTexto  = "TEXTO"
	TipoNumero = "NUMERO"
	TipoLista  = "LISTA"
)

// TamanhoDescricaoNota é o limite de xProd na NF-e: descrição maior que isso
// precisa da descrição comercial para ir à nota.
const TamanhoDescricaoNota = 120

// Valor é uma opção de um atributo do tipo lista.
type Valor struct {
	ID         int64   `json:"id,omitempty"`
	Valor      string  `json:"value"`
	Abreviacao *string `json:"abbreviation,omitempty"`
	Ordem      int     `json:"position"`
	Ativo      bool    `json:"is_active"`
}

// Atributo é uma característica do modelo: o que o cadastrador preenche.
type Atributo struct {
	ID           int64   `json:"id,omitempty"`
	GroupCode    int32   `json:"group_code"`
	ModifierID   int32   `json:"modifier_id"`
	Nome         string  `json:"name"`
	Ordem        int     `json:"position"`
	Tipo         string  `json:"value_type"`
	Unidade      *string `json:"unit,omitempty"`
	Prefixo      *string `json:"prefix,omitempty"`
	Casas        int     `json:"decimal_places"`
	Obrigatorio  bool    `json:"is_required"`
	NaDescricao  bool    `json:"in_description"`
	NaAssinatura bool    `json:"in_signature"`
	Ativo        bool    `json:"is_active"`
	Valores      []Valor `json:"values"`
}

// Conferir valida o atributo antes de gravar.
func (a *Atributo) Conferir() error {
	a.Nome = strings.ToUpper(strings.TrimSpace(a.Nome))
	if a.Nome == "" {
		return domainerr.NewValidation("informe o nome do atributo")
	}
	if a.GroupCode == 0 || a.ModifierID == 0 {
		return domainerr.NewValidation("o atributo pertence a um grupo e a um modificador")
	}
	if a.Tipo == "" {
		a.Tipo = TipoTexto
	}
	if a.Tipo != TipoTexto && a.Tipo != TipoNumero && a.Tipo != TipoLista {
		return domainerr.NewValidation("tipo de atributo inválido: use texto, número ou lista")
	}
	if a.Casas < 0 || a.Casas > 6 {
		return domainerr.NewValidation("as casas decimais vão de 0 a 6")
	}
	if a.Tipo == TipoLista {
		ativos := 0
		vistos := map[string]bool{}
		for i := range a.Valores {
			v := &a.Valores[i]
			v.Valor = strings.ToUpper(strings.TrimSpace(v.Valor))
			if v.Valor == "" {
				return domainerr.NewValidation("a lista tem um valor em branco")
			}
			if vistos[v.Valor] {
				return domainerr.NewValidation(fmt.Sprintf("o valor %s está repetido na lista", v.Valor))
			}
			vistos[v.Valor] = true
			if v.Ativo {
				ativos++
			}
		}
		if ativos == 0 {
			return domainerr.NewValidation("atributo do tipo lista precisa de ao menos um valor ativo")
		}
	}
	return nil
}

// Modelo é o padrão de um par grupo+modificador.
type Modelo struct {
	GroupCode            int32      `json:"group_code"`
	GroupDescription     string     `json:"group_description"`
	ModifierID           int32      `json:"modifier_id"`
	ModifierDescription  string     `json:"modifier_description"`
	ModifierAbbreviation *string    `json:"modifier_abbreviation,omitempty"`
	Atributos            []Atributo `json:"attributes"`
}

// TemPadrao diz se o par tem atributos ativos; sem eles a descrição segue livre.
func (m Modelo) TemPadrao() bool {
	for _, a := range m.Atributos {
		if a.Ativo {
			return true
		}
	}
	return false
}

// Resultado é a aplicação do modelo aos valores informados.
type Resultado struct {
	Descricao   string                  `json:"description"`
	Assinatura  string                  `json:"signature"`
	Atributos   []valueobject.Attribute `json:"attributes"`
	Tamanho     int                     `json:"length"`
	PassaDaNota bool                    `json:"exceeds_invoice_limit"`
	Erros       []string                `json:"errors"`
}

var espacos = regexp.MustCompile(`\s+`)

// Aplicar normaliza os valores, confere o preenchimento e monta a descrição:
// modificador seguido dos atributos, na ordem do modelo.
func (m Modelo) Aplicar(informados []valueobject.Attribute) Resultado {
	return m.aplicar(informados, false)
}

// AplicarNaAlteracao aceita o atributo e o valor que foram inativados depois do
// cadastro: sem isso, o item antigo só salvaria com a descrição refeita.
func (m Modelo) AplicarNaAlteracao(informados []valueobject.Attribute) Resultado {
	return m.aplicar(informados, true)
}

func (m Modelo) aplicar(informados []valueobject.Attribute, legado bool) Resultado {
	r := Resultado{Atributos: []valueobject.Attribute{}, Erros: []string{}}
	porNome := map[string]string{}
	for _, v := range informados {
		porNome[strings.ToUpper(strings.TrimSpace(v.Name))] = strings.TrimSpace(v.Value)
	}
	nome := strings.ToUpper(strings.TrimSpace(m.ModifierDescription))
	if m.ModifierAbbreviation != nil && strings.TrimSpace(*m.ModifierAbbreviation) != "" {
		nome = strings.ToUpper(strings.TrimSpace(*m.ModifierAbbreviation))
	}
	partes := []string{nome}
	assinatura := []string{fmt.Sprintf("%d|%d", m.GroupCode, m.ModifierID)}
	for _, a := range m.Atributos {
		if !a.Ativo && !(legado && porNome[a.Nome] != "") {
			continue
		}
		bruto := porNome[a.Nome]
		delete(porNome, a.Nome)
		if bruto == "" {
			if a.Obrigatorio {
				r.Erros = append(r.Erros, fmt.Sprintf("informe %s", a.Nome))
			}
			continue
		}
		valor, texto, err := a.normalizar(bruto, legado)
		if err != nil {
			r.Erros = append(r.Erros, err.Error())
			continue
		}
		r.Atributos = append(r.Atributos, valueobject.Attribute{Name: a.Nome, Value: valor})
		if a.NaDescricao {
			partes = append(partes, texto)
		}
		if a.NaAssinatura {
			assinatura = append(assinatura, a.Nome+"="+valor)
		}
	}
	for n := range porNome {
		if n != "" {
			r.Erros = append(r.Erros, fmt.Sprintf("o atributo %s não faz parte do padrão deste modificador", n))
		}
	}
	r.Descricao = espacos.ReplaceAllString(strings.TrimSpace(strings.Join(partes, " ")), " ")
	r.Tamanho = utf8.RuneCountInString(r.Descricao)
	r.PassaDaNota = r.Tamanho > TamanhoDescricaoNota
	soma := sha256.Sum256([]byte(strings.Join(assinatura, ";")))
	r.Assinatura = hex.EncodeToString(soma[:])
	return r
}

// normalizar devolve o valor gravado e o texto que entra na descrição.
func (a Atributo) normalizar(bruto string, legado bool) (valor, texto string, err error) {
	prefixo, unidade := "", ""
	if a.Prefixo != nil {
		prefixo = strings.ToUpper(strings.TrimSpace(*a.Prefixo))
	}
	if a.Unidade != nil {
		unidade = strings.ToUpper(strings.TrimSpace(*a.Unidade))
	}
	switch a.Tipo {
	case TipoNumero:
		n, e := decimal.NewFromString(strings.ReplaceAll(bruto, ",", "."))
		if e != nil {
			return "", "", fmt.Errorf("%s precisa ser um número", a.Nome)
		}
		valor = n.Round(int32(a.Casas)).StringFixed(int32(a.Casas))
		// Na descrição o zero à direita só atrapalha: 6,00 vira 6.
		exibido := strings.ReplaceAll(n.Round(int32(a.Casas)).String(), ".", ",")
		return valor, prefixo + exibido + unidade, nil
	case TipoLista:
		alvo := strings.ToUpper(bruto)
		for _, v := range a.Valores {
			if (!v.Ativo && !legado) || v.Valor != alvo {
				continue
			}
			exibido := v.Valor
			if v.Abreviacao != nil && strings.TrimSpace(*v.Abreviacao) != "" {
				exibido = strings.ToUpper(strings.TrimSpace(*v.Abreviacao))
			}
			return v.Valor, prefixo + exibido + unidade, nil
		}
		return "", "", fmt.Errorf("%s não aceita o valor %s", a.Nome, bruto)
	}
	valor = strings.ToUpper(bruto)
	return valor, prefixo + valor + unidade, nil
}

// Uso: quantos itens usam cada grupo e modificador, para decidir antes de inativar.
type Uso struct {
	Grupos        map[int32]int `json:"groups"`
	Modificadores map[int32]int `json:"modifiers"`
}

// Repository grava o padrão, sempre da empresa da sessão.
type Repository interface {
	Modelo(ctx context.Context, group, modifier int32) (*Modelo, error)
	ModificadoresDoGrupo(ctx context.Context, group int32) ([]int32, error)
	SalvarModificadoresDoGrupo(ctx context.Context, group int32, modificadores []int32) error
	SalvarAbreviacao(ctx context.Context, modifier int32, abreviacao *string) error
	SalvarAtributo(ctx context.Context, a *Atributo, por string) (*Atributo, error)
	InativarAtributo(ctx context.Context, id int64) error
	ItensComAssinatura(ctx context.Context, assinatura string, excluir string) ([]string, error)
	Uso(ctx context.Context) (*Uso, error)
}
