// Package service: a promessa de entrega "capaz de prometer" (melhoria 9).
//
// A data sai, nesta ordem, do que existe: o estoque livre (ATP), o que já está
// a caminho (OF e pedido de compra abertos) menos o que outros pedidos já
// pediram antes, e por fim o que ainda precisa ser feito — produção (prazo do
// item + fila de capacidade dos centros do roteiro + a data dos componentes que
// faltam) ou compra (prazo do fornecedor). O cenário ("e se entrar um pedido de
// 50?") é a mesma conta sem gravar nada.
package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Origens da promessa.
const (
	OrigemEstoque      = "ESTOQUE"
	OrigemEmAndamento  = "ENTRADA_PREVISTA"
	OrigemNovaProducao = "NOVA_PRODUCAO"
	OrigemNovaCompra   = "NOVA_COMPRA"
)

// Movimento previsto: entrada (OF/compra) ou compromisso (pedido de venda).
type Movimento struct {
	Data       time.Time
	Quantidade decimal.Decimal
	Documento  string
}

// Componente da estrutura ativa, na unidade de estoque.
type Componente struct {
	ItemCode int64
	QtdPor   decimal.Decimal
}

// SituacaoDoItem é o que o banco sabe do item.
type SituacaoDoItem struct {
	ItemCode                             int64
	Fabricado                            bool
	Livre                                decimal.Decimal // em estoque, prometível e não reservado
	Entradas                             []Movimento     // OFs e compras abertas (o que falta chegar)
	Compromissos                         []Movimento     // pedidos abertos ainda não reservados
	PrazoFabricacaoDias, PrazoCompraDias int
	// FilaDias: carga aberta dos centros do roteiro ÷ capacidade diária deles.
	FilaDias int
}

// Promessa é a resposta para uma quantidade.
type Promessa struct {
	ItemCode   int64           `json:"item_code"`
	Quantidade decimal.Decimal `json:"quantity"`
	Data       time.Time       `json:"promised_date"`
	Origem     string          `json:"source"`
	Explicacao []string        `json:"explanation"`
	// Componentes: a promessa de cada componente que precisou de reposição.
	Componentes []Promessa `json:"components,omitempty"`
}

// diasUteis soma n dias úteis (seg–sex) a partir de d.
func DiasUteis(d time.Time, n int) time.Time {
	for n > 0 {
		d = d.AddDate(0, 0, 1)
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			n--
		}
	}
	return d
}

// Atender tenta cobrir qtd pelo que existe ou está a caminho. Devolve a data,
// a origem e se conseguiu. consumido é o que outras linhas do mesmo cenário já
// tomaram deste item (é descontado antes).
func Atender(hoje time.Time, qtd, consumido decimal.Decimal, s SituacaoDoItem) (time.Time, string, []string, bool) {
	livre := s.Livre.Sub(consumido)
	if livre.GreaterThanOrEqual(qtd) {
		return hoje, OrigemEstoque, []string{fmt.Sprintf("%s livre em estoque", strings.Replace(livre.Round(3).String(), ".", ",", 1))}, true
	}
	// Linha do tempo: entradas somam, compromissos de outros pedidos subtraem.
	type evento struct {
		data time.Time
		q    decimal.Decimal
		doc  string
	}
	var ev []evento
	for _, e := range s.Entradas {
		ev = append(ev, evento{e.Data, e.Quantidade, e.Documento})
	}
	for _, c := range s.Compromissos {
		ev = append(ev, evento{c.Data, c.Quantidade.Neg(), c.Documento})
	}
	sort.SliceStable(ev, func(i, j int) bool { return ev[i].data.Before(ev[j].data) })
	saldo := livre
	data, doc := time.Time{}, ""
	for i, e := range ev {
		saldo = saldo.Add(e.q)
		if e.q.IsPositive() && saldo.GreaterThanOrEqual(qtd) && data.IsZero() {
			// Só vale se nenhum compromisso posterior derrubar o saldo abaixo de qtd.
			ok := true
			resto := saldo
			for _, d := range ev[i+1:] {
				resto = resto.Add(d.q)
				if resto.LessThan(qtd) {
					ok = false
					break
				}
			}
			if ok {
				data, doc = e.data, e.doc
			}
		}
	}
	if !data.IsZero() {
		if data.Before(hoje) {
			data = hoje
		}
		return data, OrigemEmAndamento, []string{fmt.Sprintf("coberto pela entrada prevista de %s (%s)", data.Format("02/01/2006"), doc)}, true
	}
	return time.Time{}, "", nil, false
}

// NovaReposicao: a data de quando a reposição feita AGORA chega. componentes é
// a data mais tarde dos componentes que precisaram de reposição (zero = todos à
// mão).
func NovaReposicao(hoje time.Time, s SituacaoDoItem, componentes time.Time) (time.Time, string, []string) {
	if !s.Fabricado {
		prazo := s.PrazoCompraDias
		return DiasUteis(hoje, prazo), OrigemNovaCompra, []string{fmt.Sprintf("compra nova: prazo de %d dia(s) útil(eis) do fornecedor", prazo)}
	}
	inicio := hoje
	exp := []string{}
	if componentes.After(inicio) {
		inicio = componentes
		exp = append(exp, fmt.Sprintf("componentes disponíveis em %s", componentes.Format("02/01/2006")))
	}
	if s.FilaDias > 0 {
		inicio = DiasUteis(inicio, s.FilaDias)
		exp = append(exp, fmt.Sprintf("fila de %d dia(s) útil(eis) nos centros do roteiro", s.FilaDias))
	}
	fim := DiasUteis(inicio, s.PrazoFabricacaoDias)
	exp = append(exp, fmt.Sprintf("fabricação: %d dia(s) útil(eis)", s.PrazoFabricacaoDias))
	return fim, OrigemNovaProducao, exp
}
