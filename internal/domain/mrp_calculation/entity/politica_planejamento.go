package entity

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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

// Período de agrupamento das necessidades (lote por período).
const (
	LotePeriodoNenhum = "NENHUM"
	LotePeriodoDia    = "DIA"
	LotePeriodoSemana = "SEMANA"
	LotePeriodoMes    = "MES"
)

// LotePeriodoValido diz se o período existe.
func LotePeriodoValido(p string) bool {
	switch p {
	case LotePeriodoNenhum, LotePeriodoDia, LotePeriodoSemana, LotePeriodoMes:
		return true
	}
	return false
}

// InicioDoPeriodo é o começo do período da data: o próprio dia, a segunda-feira
// da semana ou o dia 1º do mês. Fora disso (NENHUM), a data.
func InicioDoPeriodo(periodo string, d time.Time) time.Time {
	dia := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	switch periodo {
	case LotePeriodoSemana:
		desdeSegunda := (int(dia.Weekday()) + 6) % 7
		return dia.AddDate(0, 0, -desdeSegunda)
	case LotePeriodoMes:
		return time.Date(dia.Year(), dia.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	return dia
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
	// LotePeriodo junta as necessidades do período numa ordem; LoteMaximo
	// quebra a necessidade em ordens de no máximo esse valor (só do item).
	LotePeriodo *string
	LoteMaximo  *decimal.Decimal
	Observacao  *string
	CreatedAt   time.Time
	CreatedBy   uuid.UUID
	UpdatedAt   time.Time
	UpdatedBy   *uuid.UUID

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
	if p.LotePeriodo != nil && !LotePeriodoValido(*p.LotePeriodo) {
		return "período de lote inválido: use NENHUM, DIA, SEMANA ou MES"
	}
	if p.LoteMaximo != nil {
		if !p.LoteMaximo.IsPositive() {
			return "o lote máximo precisa ser maior que zero"
		}
		if p.ItemCode == nil {
			return "lote máximo é quantidade do item: informe na política do item, não da classificação"
		}
	}
	if p.Estrategia == nil && p.ConsumoDiasAntes == nil && p.ConsumoDiasDepois == nil &&
		p.HorizonteFirmeDias == nil && p.TempoSegurancaDias == nil && p.LotePeriodo == nil && p.LoteMaximo == nil {
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
	LotePeriodo    string
	LoteMaximo     string
}

// PoliticaEfetiva é a política que vale para o item depois da herança.
type PoliticaEfetiva struct {
	Estrategia         string
	ConsumoDiasAntes   int
	ConsumoDiasDepois  int
	HorizonteFirmeDias int
	TempoSegurancaDias int
	LotePeriodo        string
	// LoteMaximo zero = sem lote máximo.
	LoteMaximo decimal.Decimal
	Origem     OrigemPolitica
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
	if r.LotePeriodo != nil {
		ef.LotePeriodo, ef.Origem.LotePeriodo = *r.LotePeriodo, origem
	}
	if r.LoteMaximo != nil {
		ef.LoteMaximo, ef.Origem.LoteMaximo = *r.LoteMaximo, origem
	}
}

// DividirPorLoteMaximo quebra a quantidade em ordens de no máximo o lote
// máximo — ajustado para baixo ao lote múltiplo, para cada ordem continuar
// múltipla dele. A última leva o resto. Sem lote máximo, uma ordem só.
func DividirPorLoteMaximo(qtd, loteMaximo, loteMultiplo decimal.Decimal) []decimal.Decimal {
	if !qtd.IsPositive() {
		return nil
	}
	maximo := loteMaximo
	if maximo.IsPositive() && loteMultiplo.IsPositive() {
		if ajustado := maximo.Div(loteMultiplo).Floor().Mul(loteMultiplo); ajustado.IsPositive() {
			maximo = ajustado
		}
	}
	if !maximo.IsPositive() || qtd.LessThanOrEqual(maximo) {
		return []decimal.Decimal{qtd}
	}
	var out []decimal.Decimal
	for resta := qtd; resta.IsPositive(); resta = resta.Sub(maximo) {
		out = append(out, decimal.Min(resta, maximo))
	}
	return out
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
