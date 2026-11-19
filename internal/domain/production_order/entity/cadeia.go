package entity

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Conferência do início da OF (parâmetro de planejamento 52): o que fazer
// quando a OF vai começar sem o material todo — inclusive a peça que outra OF
// (a filha) ainda está fabricando. O SAP faz a mesma conferência na liberação
// da ordem, com o mesmo trio de respostas.
const (
	ParametroConferenciaInicio = 52

	ConferenciaInicioNao      = "NAO"
	ConferenciaInicioAvisar   = "AVISAR"
	ConferenciaInicioBloquear = "BLOQUEAR"
)

// ModoConferenciaValido diz se o valor é um dos modos do parâmetro 52.
func ModoConferenciaValido(v string) bool {
	switch v {
	case ConferenciaInicioNao, ConferenciaInicioAvisar, ConferenciaInicioBloquear:
		return true
	}
	return false
}

// ModoConferenciaDoParametro lê o valor gravado; vazio ou desconhecido é
// AVISAR, o padrão da migração.
func ModoConferenciaDoParametro(v string) string {
	v = strings.ToUpper(strings.TrimSpace(v))
	if ModoConferenciaValido(v) {
		return v
	}
	return ConferenciaInicioAvisar
}

// OrdemFilhaPendente é a OF (ou ordem planejada ainda sem OF) do componente
// fabricado que abastece a OF que vai começar e ainda não terminou.
type OrdemFilhaPendente struct {
	PlannedOrderCode int64
	// ProductionOrderID/OrderNumber: zero quando a ordem planejada ainda não
	// gerou OF.
	ProductionOrderID int64
	OrderNumber       int64
	ItemCode          int64
	BusinessCode      string
	Description       string
	Status            string
	Planejada         decimal.Decimal
	Produzida         decimal.Decimal
}

// PendenciasDoInicio é o que impede (ou merece aviso) o início da OF.
type PendenciasDoInicio struct {
	Filhas []OrdemFilhaPendente
	// Faltas: linhas de material com falta (Falta() > 0).
	Faltas []MaterialAvailabilityLine
}

// Vazia diz se não há nada a avisar.
func (p PendenciasDoInicio) Vazia() bool { return len(p.Filhas) == 0 && len(p.Faltas) == 0 }

// Mensagens descreve cada pendência para quem está iniciando a OF.
func (p PendenciasDoInicio) Mensagens() []string {
	out := make([]string, 0, len(p.Filhas)+len(p.Faltas))
	for _, f := range p.Filhas {
		item := rotuloDoItem(f.ItemCode, f.BusinessCode, f.Description)
		if f.ProductionOrderID == 0 {
			out = append(out, fmt.Sprintf("o componente %s ainda não tem OF: a ordem planejada %d não foi liberada", item, f.PlannedOrderCode))
			continue
		}
		out = append(out, fmt.Sprintf("a OF %d do componente %s ainda não foi concluída (%s de %s entregues)",
			f.ProductionOrderID, item, QuantidadeTexto(f.Produzida), QuantidadeTexto(f.Planejada)))
	}
	for _, l := range p.Faltas {
		out = append(out, fmt.Sprintf("falta %s %s de %s para baixar nesta OF (disponível %s)",
			QuantidadeTexto(l.Falta()), strings.TrimSpace(l.UOM), rotuloDoItem(l.ItemCode, l.BusinessCode, l.Description),
			QuantidadeTexto(l.Disponivel())))
	}
	return out
}

// AvaliarInicio aplica o modo às pendências: no BLOQUEAR com pendência devolve
// a recusa; no AVISAR, os avisos; no NAO, nada.
func AvaliarInicio(modo string, p PendenciasDoInicio) (recusa string, avisos []string) {
	if modo == ConferenciaInicioNao || p.Vazia() {
		return "", nil
	}
	msgs := p.Mensagens()
	if modo == ConferenciaInicioBloquear {
		return "a OF não pode começar: " + strings.Join(msgs, "; ") +
			" (parâmetro de planejamento 52 está em BLOQUEAR)", nil
	}
	return "", msgs
}

func rotuloDoItem(code int64, business, description string) string {
	rotulo := strings.TrimSpace(business)
	if rotulo == "" {
		rotulo = fmt.Sprintf("%d", code)
	}
	if d := strings.TrimSpace(description); d != "" {
		rotulo += " (" + d + ")"
	}
	return rotulo
}

// QuantidadeTexto escreve a quantidade com vírgula e sem zeros à direita.
func QuantidadeTexto(q decimal.Decimal) string {
	return strings.Replace(q.Round(6).String(), ".", ",", 1)
}

// CadeiaDaOF é a posição da OF na cadeia de fabricação: os pedidos de venda que
// ela atende (direto ou herdados do pai), as ordens pai que ela abastece e as
// ordens filhas dos componentes fabricados para ela.
type CadeiaDaOF struct {
	Pedidos []PedidoDaCadeia
	// Demandas: as demandas independentes (produção para estoque, amostra…)
	// que a OF atende, direto ou por uma OF acima dela na cadeia. Sem pedido de
	// venda, é o que diz para que a peça está sendo feita.
	Demandas []DemandaDaCadeia
	Pais     []OrdemDaCadeia
	Filhas   []OrdemDaCadeia
}

// DemandaDaCadeia é a demanda independente ligada à OF ou a uma ancestral dela.
type DemandaDaCadeia struct {
	Code         int64
	Motivo       string
	ItemCode     int64
	BusinessCode string
	Description  string
	Quantity     decimal.Decimal
	DemandDate   time.Time
	Situacao     string
}

// PedidoDaCadeia é a linha do pedido de venda ligada à OF.
type PedidoDaCadeia struct {
	SalesOrderCode     int64
	SalesOrderNumber   int64
	SalesOrderItemCode *int64
	Sequence           *int32
	Customer           string
}

// OrdemDaCadeia é a ordem (OF ou planejada ainda sem OF) de um elo da cadeia.
type OrdemDaCadeia struct {
	PlannedOrderCode  *int64
	ProductionOrderID *int64
	OrderNumber       *int64
	ItemCode          int64
	BusinessCode      string
	Description       string
	Status            string
	Quantity          decimal.Decimal
	Produced          decimal.Decimal
	// Reserved: na filha, quanto da peça dela está reservado para esta OF.
	Reserved decimal.Decimal
}
