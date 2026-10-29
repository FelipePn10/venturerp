package entity

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Política de planejamento por item ou por classificação (VPLA0310). Ver
// .ai/tasks/politica-planejamento-enterprise.md.

// Estratégia de planejamento do item.
const (
	// EstrategiaConsumo: a venda consome a previsão e a demanda de
	// expectativa de venda (SAP, estratégia 40). É o padrão.
	EstrategiaConsumo = "CONSUMO"
	// EstrategiaSobEncomenda: só o pedido de venda gera necessidade; a
	// previsão e a demanda de expectativa de venda do item não entram no MRP
	// (SAP, estratégia 20). Demanda de finalidade própria continua valendo.
	EstrategiaSobEncomenda = "SOB_ENCOMENDA"
)

// EstrategiaValida diz se a estratégia existe.
func EstrategiaValida(e string) bool {
	return e == EstrategiaConsumo || e == EstrategiaSobEncomenda
}

// Limite de dias dos campos da política (janela, horizonte e segurança).
const DiasMaximosPolitica = 365

// PoliticaPlanejamento é uma regra: do item (ItemCode) ou da classificação
// (ClassificationID), nunca dos dois. Campo nil herda do nível de cima.
type PoliticaPlanejamento struct {
	ID                 int64
	ItemCode           *int64
	ClassificationID   *int64
	Estrategia         *string
	ConsumoDiasAntes   *int
	ConsumoDiasDepois  *int
	HorizonteFirmeDias *int
	TempoSegurancaDias *int
	Observacao         *string
	CreatedAt          time.Time
	CreatedBy          uuid.UUID
	UpdatedAt          time.Time
	UpdatedBy          *uuid.UUID

	// Rótulos para a tela (preenchidos pela leitura).
	ItemBusinessCode    string
	ItemName            string
	ClassificationCode  string
	ClassificationLabel string
	// Nível da classificação na árvore (1 = raiz): a mais profunda vence.
	ClassificationLevel int
}

// Validar confere o que não depende do banco.
func (p *PoliticaPlanejamento) Validar() string {
	if (p.ItemCode == nil) == (p.ClassificationID == nil) {
		return "a política é de um item ou de uma classificação (um dos dois)"
	}
	if p.Estrategia != nil && !EstrategiaValida(*p.Estrategia) {
		return "estratégia inválida: use CONSUMO ou SOB_ENCOMENDA"
	}
	for nome, v := range map[string]*int{
		"dias antes da janela de consumo":  p.ConsumoDiasAntes,
		"dias depois da janela de consumo": p.ConsumoDiasDepois,
		"horizonte firme":                  p.HorizonteFirmeDias,
		"tempo de segurança":               p.TempoSegurancaDias,
	} {
		if v != nil && (*v < 0 || *v > DiasMaximosPolitica) {
			return fmt.Sprintf("%s vai de 0 a %d dias", nome, DiasMaximosPolitica)
		}
	}
	if p.Estrategia == nil && p.ConsumoDiasAntes == nil && p.ConsumoDiasDepois == nil &&
		p.HorizonteFirmeDias == nil && p.TempoSegurancaDias == nil {
		return "informe ao menos um campo da política (os vazios seguem o nível de cima)"
	}
	return ""
}

// De onde veio cada valor da política efetiva.
const (
	OrigemItem    = "ITEM"
	OrigemEmpresa = "EMPRESA"
	// Classificação: "CLASSIFICACAO:<código>".
	origemClassificacao = "CLASSIFICACAO:"
)

// OrigemPolitica diz de onde veio cada campo da política efetiva.
type OrigemPolitica struct {
	Estrategia     string
	Janela         string
	HorizonteFirme string
	TempoSeguranca string
}

// PoliticaEfetiva é a política que vale para o item depois da herança.
type PoliticaEfetiva struct {
	Estrategia         string
	ConsumoDiasAntes   int
	ConsumoDiasDepois  int
	HorizonteFirmeDias int
	TempoSegurancaDias int
	Origem             OrigemPolitica
}

// SobEncomenda diz se o item é planejado só pelo pedido.
func (p PoliticaEfetiva) SobEncomenda() bool { return p.Estrategia == EstrategiaSobEncomenda }

// ResolverPoliticas monta a política efetiva de cada item que tem regra: a do
// item, campo a campo, sobre a da classificação mais profunda (e, no mesmo
// nível, a de menor id), sobre a da empresa. porClassificacao são as regras
// de classificação que alcançam cada item (atribuída a ele ou ancestral).
func ResolverPoliticas(empresa PoliticaEfetiva, doItem map[int64]*PoliticaPlanejamento, porClassificacao map[int64][]*PoliticaPlanejamento) map[int64]PoliticaEfetiva {
	out := make(map[int64]PoliticaEfetiva, len(doItem)+len(porClassificacao))
	itens := map[int64]bool{}
	for item := range doItem {
		itens[item] = true
	}
	for item := range porClassificacao {
		itens[item] = true
	}
	for item := range itens {
		regras := append([]*PoliticaPlanejamento(nil), porClassificacao[item]...)
		// Da mais rasa para a mais profunda: a aplicada por último vence.
		sort.SliceStable(regras, func(a, b int) bool {
			if regras[a].ClassificationLevel != regras[b].ClassificationLevel {
				return regras[a].ClassificationLevel < regras[b].ClassificationLevel
			}
			return regras[a].ID > regras[b].ID
		})
		ef := empresa
		for _, r := range regras {
			aplicar(&ef, r, origemClassificacao+r.ClassificationCode)
		}
		if r := doItem[item]; r != nil {
			aplicar(&ef, r, OrigemItem)
		}
		out[item] = ef
	}
	return out
}

func aplicar(ef *PoliticaEfetiva, r *PoliticaPlanejamento, origem string) {
	if r.Estrategia != nil {
		ef.Estrategia, ef.Origem.Estrategia = *r.Estrategia, origem
	}
	if r.ConsumoDiasAntes != nil {
		ef.ConsumoDiasAntes, ef.Origem.Janela = *r.ConsumoDiasAntes, origem
	}
	if r.ConsumoDiasDepois != nil {
		ef.ConsumoDiasDepois, ef.Origem.Janela = *r.ConsumoDiasDepois, origem
	}
	if r.HorizonteFirmeDias != nil {
		ef.HorizonteFirmeDias, ef.Origem.HorizonteFirme = *r.HorizonteFirmeDias, origem
	}
	if r.TempoSegurancaDias != nil {
		ef.TempoSegurancaDias, ef.Origem.TempoSeguranca = *r.TempoSegurancaDias, origem
	}
}

// DescreverOrigem é a origem como o usuário lê.
func DescreverOrigem(origem string) string {
	switch {
	case origem == OrigemItem:
		return "do item"
	case origem == OrigemEmpresa:
		return "parâmetro da empresa"
	case strings.HasPrefix(origem, origemClassificacao):
		return "da classificação " + strings.TrimPrefix(origem, origemClassificacao)
	}
	return origem
}

// FiltroPolitica da lista da VPLA0310.
type FiltroPolitica struct {
	Escopo string // ITEM, CLASSIFICACAO ou vazio
	Busca  string
}
