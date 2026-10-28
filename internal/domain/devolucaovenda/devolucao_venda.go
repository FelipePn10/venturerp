// Package devolucaovenda: devolução de venda do cliente (comercial, Fase 5.1).
// O cálculo proporcional dos itens e impostos é o mesmo da devolução de compra
// (fiscal/devolucao); aqui ficam o documento, o CFOP de devolução de venda e
// as regras da efetivação.
package devolucaovenda

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/devolucao"
)

const (
	EmissaoPropria = "EMISSAO_PROPRIA"
	NFCliente      = "NF_CLIENTE"

	Rascunho  = "RASCUNHO"
	Emitida   = "EMITIDA"
	Efetivada = "EFETIVADA"
	Cancelada = "CANCELADA"

	// Tipo de referência dos movimentos de estoque da devolução.
	ReferenciaEstoque = "DEVOLUCAO_VENDA"
)

var Motivos = map[string]string{
	"DEFEITO": "Defeito", "AVARIA_TRANSPORTE": "Avaria no transporte", "ERRO_PEDIDO": "Erro no pedido",
	"DESISTENCIA": "Desistência do cliente", "GARANTIA": "Garantia", "OUTROS": "Outros",
}

// sufixos: CFOP de venda (3 últimos dígitos) → CFOP de devolução de venda.
var sufixos = map[string]string{
	"101": "201", // venda de produção do estabelecimento
	"102": "202", // venda de mercadoria de terceiros
	"103": "203", "104": "204", "105": "205", "106": "206",
	"109": "208", "110": "209", // Zona Franca / ALC
	"401": "410", // venda de produção com ST (substituto)
	"402": "410",
	"403": "411", // venda de terceiros com ST (substituto)
	"405": "411", // venda de terceiros com ST (substituído)
	"122": "201", "123": "202",
}

// CFOP de devolução para o CFOP da venda: 5xxx → 1xxx, 6xxx → 2xxx, 7xxx →
// 3xxx; sem correspondência, x949 (outra entrada).
func CFOP(cfopVenda string) string {
	c := strings.TrimSpace(cfopVenda)
	prefixo := "1"
	if len(c) == 4 {
		switch c[0] {
		case '6':
			prefixo = "2"
		case '7':
			prefixo = "3"
		}
		if s, ok := sufixos[c[1:]]; ok {
			return prefixo + s
		}
	}
	return prefixo + "949"
}

// LinhaDaNota: a linha da NF de saída com o que já foi devolvido.
type LinhaDaNota struct {
	FiscalExitItemID int64           `json:"fiscal_exit_item_id"`
	Sequencia        int             `json:"sequencia"`
	ItemCode         *int64          `json:"item_code,omitempty"`
	Descricao        string          `json:"descricao"`
	CFOP             string          `json:"cfop"`
	Quantidade       decimal.Decimal `json:"quantidade"`
	Devolvida        decimal.Decimal `json:"devolvida"`
	Disponivel       decimal.Decimal `json:"disponivel"`
	ValorUnitario    decimal.Decimal `json:"valor_unitario"`
	ControlaSerie    bool            `json:"controla_serie"`
	// Séries que saíram na nota e ainda não voltaram (id → número).
	Series []SerieDaNota `json:"series"`
	origem devolucao.Origem
}

type SerieDaNota struct {
	ID     int64  `json:"id"`
	Numero string `json:"numero"`
}

func (l *LinhaDaNota) DefinirOrigem(o devolucao.Origem) {
	l.origem = o
	l.Disponivel = o.Disponivel()
}

func (l LinhaDaNota) Origem() devolucao.Origem { return l.origem }

// Pedido de devolução de uma linha.
type Pedido struct {
	FiscalExitItemID int64           `json:"fiscal_exit_item_id"`
	Quantidade       decimal.Decimal `json:"quantidade"`
	Series           []int64         `json:"series,omitempty"`
}

// Item calculado da devolução.
type Item struct {
	ID                        int64            `json:"id"`
	FiscalExitItemID          int64            `json:"fiscal_exit_item_id"`
	ItemCode                  *int64           `json:"item_code,omitempty"`
	Descricao                 string           `json:"descricao,omitempty"`
	CFOP                      string           `json:"cfop,omitempty"`
	Quantidade                decimal.Decimal  `json:"quantidade"`
	ValorProdutos             decimal.Decimal  `json:"valor_produtos"`
	ICMS                      decimal.Decimal  `json:"valor_icms"`
	ST                        decimal.Decimal  `json:"valor_icms_st"`
	IPI                       decimal.Decimal  `json:"valor_ipi"`
	PIS                       decimal.Decimal  `json:"valor_pis"`
	COFINS                    decimal.Decimal  `json:"valor_cofins"`
	Total                     decimal.Decimal  `json:"valor_total"`
	Series                    []int64          `json:"series"`
	CustoUnitario             *decimal.Decimal `json:"custo_unitario,omitempty"`
	BaseICMS, BaseIPI, BaseST decimal.Decimal  `json:"-"`
}

// Calcular os itens pedidos contra as linhas da nota: quantidade disponível,
// proporcional nos valores e impostos, série obrigatória (uma por unidade e
// entre as que saíram na nota) quando o item controla série.
func Calcular(linhas []LinhaDaNota, pedidos []Pedido) ([]Item, decimal.Decimal, error) {
	if len(pedidos) == 0 {
		return nil, decimal.Zero, domainerr.NewValidation("escolha ao menos um item para devolver")
	}
	porID := map[int64]LinhaDaNota{}
	for _, l := range linhas {
		porID[l.FiscalExitItemID] = l
	}
	visto := map[int64]bool{}
	total := decimal.Zero
	out := make([]Item, 0, len(pedidos))
	for _, p := range pedidos {
		l, ok := porID[p.FiscalExitItemID]
		if !ok {
			return nil, decimal.Zero, domainerr.NewValidation(fmt.Sprintf("a linha %d não é da nota devolvida", p.FiscalExitItemID))
		}
		if visto[p.FiscalExitItemID] {
			return nil, decimal.Zero, domainerr.NewValidation(fmt.Sprintf("a linha %d foi informada duas vezes", l.Sequencia))
		}
		visto[p.FiscalExitItemID] = true
		ln, err := devolucao.Montar(l.origem, p.Quantidade)
		if err != nil {
			return nil, decimal.Zero, err
		}
		if l.ControlaSerie {
			if !p.Quantidade.Equal(p.Quantidade.Truncate(0)) || int64(len(p.Series)) != p.Quantidade.IntPart() {
				return nil, decimal.Zero, domainerr.NewValidation(fmt.Sprintf("o item %q controla série: escolha %s série(s) devolvida(s)", l.Descricao, p.Quantidade.String()))
			}
			saiu := map[int64]bool{}
			for _, s := range l.Series {
				saiu[s.ID] = true
			}
			usadas := map[int64]bool{}
			for _, s := range p.Series {
				if !saiu[s] {
					return nil, decimal.Zero, domainerr.NewValidation(fmt.Sprintf("a série %d não saiu nesta nota (ou já voltou)", s))
				}
				if usadas[s] {
					return nil, decimal.Zero, domainerr.NewValidation("série repetida na devolução")
				}
				usadas[s] = true
			}
		}
		it := Item{FiscalExitItemID: l.FiscalExitItemID, ItemCode: l.ItemCode, Descricao: l.Descricao, CFOP: CFOP(l.CFOP), Quantidade: p.Quantidade,
			ValorProdutos: ln.Total, ICMS: ln.ICMS, ST: ln.ST, IPI: ln.IPI, PIS: ln.PIS, COFINS: ln.COFINS,
			Total: ln.ValorContabil().Round(2), Series: append([]int64{}, p.Series...), BaseICMS: ln.BaseICMS, BaseIPI: ln.BaseIPI, BaseST: ln.BaseST}
		total = total.Add(it.Total)
		out = append(out, it)
	}
	return out, total.Round(2), nil
}

// Devolucao: o documento.
type Devolucao struct {
	ID              int64           `json:"id"`
	Numero          int64           `json:"numero"`
	FiscalExitID    int64           `json:"fiscal_exit_id"`
	NumeroNF        int64           `json:"numero_nf"`
	ChaveNF         string          `json:"chave_nf,omitempty"`
	CustomerCode    *int64          `json:"customer_code,omitempty"`
	ClienteNome     string          `json:"cliente_nome,omitempty"`
	Modo            string          `json:"modo"`
	Situacao        string          `json:"situacao"`
	Motivo          string          `json:"motivo"`
	Observacao      *string         `json:"observacao,omitempty"`
	WarehouseID     int64           `json:"warehouse_id"`
	NFClienteChave  *string         `json:"nf_cliente_chave,omitempty"`
	NFClienteNumero *int64          `json:"nf_cliente_numero,omitempty"`
	NFPropriaID     *int64          `json:"nf_propria_id,omitempty"`
	NFPropriaStatus *string         `json:"nf_propria_status,omitempty"`
	ValorTotal      decimal.Decimal `json:"valor_total"`
	Abatido         decimal.Decimal `json:"abatido"`
	CreditoCliente  decimal.Decimal `json:"credito_cliente"`
	AdiantamentoID  *int64          `json:"adiantamento_id,omitempty"`
	EfetivadaEm     *time.Time      `json:"efetivada_em,omitempty"`
	CanceladaEm     *time.Time      `json:"cancelada_em,omitempty"`
	MotivoCancel    *string         `json:"motivo_cancelamento,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	Itens           []Item          `json:"itens"`
}

// ConferirAbertura: motivo conhecido; no modo NF do cliente a chave tem 44 dígitos.
func ConferirAbertura(modo, motivo string, chave *string) error {
	if modo != EmissaoPropria && modo != NFCliente {
		return domainerr.NewValidation("o modo é emissão própria (NF-e de entrada da empresa) ou NF do cliente")
	}
	if _, ok := Motivos[motivo]; !ok {
		return domainerr.NewValidation("motivo da devolução inválido")
	}
	if modo == NFCliente {
		if chave == nil || len(strings.TrimSpace(*chave)) != 44 {
			return domainerr.NewValidation("informe a chave de 44 dígitos da NF de devolução emitida pelo cliente")
		}
		for _, r := range *chave {
			if r < '0' || r > '9' {
				return domainerr.NewValidation("a chave da NF do cliente só tem dígitos")
			}
		}
	}
	return nil
}

// PodeEfetivar: no modo NF do cliente, do rascunho; na emissão própria, só com
// a NF-e de entrada autorizada (situação EMITIDA).
func PodeEfetivar(d *Devolucao) error {
	switch {
	case d.Situacao == Efetivada:
		return nil // idempotente
	case d.Situacao == Cancelada:
		return domainerr.NewValidation("a devolução foi cancelada")
	case d.Modo == NFCliente && d.Situacao == Rascunho:
		return nil
	case d.Modo == EmissaoPropria && d.Situacao == Emitida:
		return nil
	}
	return domainerr.NewValidation("a devolução por emissão própria efetiva quando a NF-e de entrada é autorizada")
}

// Efetivacao: o que a efetivação grava, numa transação.
type Efetivacao struct {
	Devolucao *Devolucao
	Data      time.Time
	Por       uuid.UUID
}

type Resultado struct {
	Abatido        decimal.Decimal
	CreditoCliente decimal.Decimal
	AdiantamentoID *int64
	Entradas       int
	Series         int
	Comissao       decimal.Decimal // estornada (negativa)
}

type Repository interface {
	// LinhasDaNota da NF de saída autorizada da empresa, com o já devolvido.
	LinhasDaNota(ctx context.Context, fiscalExitID int64) (numeroNF int64, cliente *int64, chave string, linhas []LinhaDaNota, err error)
	Criar(ctx context.Context, d *Devolucao, por *uuid.UUID) error
	Obter(ctx context.Context, id int64) (*Devolucao, error)
	Listar(ctx context.Context, fiscalExitID *int64, situacao string) ([]Devolucao, error)
	// VincularNFPropria grava a NF-e de entrada gerada (rascunho) na devolução.
	VincularNFPropria(ctx context.Context, id, exitID int64) error
	// MarcarEmitida: a NF-e própria foi autorizada.
	MarcarEmitida(ctx context.Context, id int64) error
	// Efetivar (idempotente): estoque, financeiro, comissão e séries.
	Efetivar(ctx context.Context, e Efetivacao) (*Resultado, error)
	Cancelar(ctx context.Context, id int64, motivo string, por *uuid.UUID) error
}
