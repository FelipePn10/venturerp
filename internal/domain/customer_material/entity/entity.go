// Package entity descreve o material do cliente em poder da empresa
// (beneficiamento / estoque de terceiros).
//
// Não confundir com `third_party_service_*`, que é o sentido OPOSTO: nós mandando
// operação de roteiro para fora. Aqui o material é do cliente, entra por NF-e de
// remessa (CFOP 5901), é processado e volta na mesma nota em que o serviço é
// faturado (CFOP 5124 para o serviço, 5902 para o material).
//
// Quantidades e valores são `decimal`: material de terceiro alimenta documento
// fiscal, e ponto flutuante gera divergência de centavo que a SEFAZ rejeita.
package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// StatusRemessa acompanha o consumo do saldo de uma remessa.
type StatusRemessa string

const (
	// StatusAberta: nada saiu ainda.
	StatusAberta StatusRemessa = "ABERTA"
	// StatusParcial: parte do material já voltou, o saldo segue vinculado.
	StatusParcial StatusRemessa = "PARCIAL"
	// StatusEncerrada: saldo zerado, ou encerramento aprovado com saldo registrado.
	StatusEncerrada StatusRemessa = "ENCERRADA"
	// StatusCancelada: a remessa foi desfeita (nota cancelada pelo cliente).
	StatusCancelada StatusRemessa = "CANCELADA"
)

// TipoMovimento distingue o que entrou do que saiu.
type TipoMovimento string

const (
	// MovimentoRecebimento é a entrada física do material do cliente.
	MovimentoRecebimento TipoMovimento = "RECEIPT"
	// MovimentoRetorno é material processado voltando com o faturamento (CFOP 5902).
	MovimentoRetorno TipoMovimento = "RETURN"
	// MovimentoSobra é material não utilizado voltando (CFOP 5903).
	MovimentoSobra TipoMovimento = "LEFTOVER"
	// MovimentoSucata é sucata gerada no processo, com destinação obrigatória.
	MovimentoSucata TipoMovimento = "SCRAP"
	// MovimentoAjuste corrige divergência apurada, com justificativa obrigatória.
	MovimentoAjuste TipoMovimento = "ADJUSTMENT"
)

// TiposDeSaida lista os movimentos que consomem saldo do cliente.
func TiposDeSaida() []TipoMovimento {
	return []TipoMovimento{MovimentoRetorno, MovimentoSobra, MovimentoSucata, MovimentoAjuste}
}

// ConsomeSaldo informa se o movimento baixa o saldo em poder da empresa.
func (t TipoMovimento) ConsomeSaldo() bool {
	return t != MovimentoRecebimento
}

// DestinoSucata registra o que foi feito com a sucata gerada. As opções vêm das
// regras que o cliente documentou; sucata sem destino não é aceita.
type DestinoSucata string

const (
	DestinoCliente  DestinoSucata = "CLIENTE"
	DestinoDescarte DestinoSucata = "DESCARTE"
	DestinoRetencao DestinoSucata = "RETENCAO"
	DestinoOutra    DestinoSucata = "OUTRA"
)

// Remessa é a NF-e com que o cliente entregou material para beneficiar.
type Remessa struct {
	ID           int64
	EnterpriseID int64
	CustomerCode int64
	NFeNumber    int64
	NFeSeries    string
	NFeKey       *string
	CFOP         string
	IssueDate    time.Time
	ReceivedAt   time.Time
	// Prazo fiscal para o retorno — 30 dias na operação da Usimac. Fica por
	// remessa porque é o que dispara a cobrança de quem está vencendo.
	FiscalReturnDeadline time.Time
	TotalValue           decimal.Decimal
	Status               StatusRemessa
	// Pedido de beneficiamento. Nulo é caso previsto: material chega sem pedido e
	// entra bloqueado até regularizar.
	SalesOrderCode *int64
	Blocked        bool
	BlockReason    *string
	ClosedAt       *time.Time
	ClosedBy       *string
	CloseReason    *string
	Notes          *string
	CreatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Itens          []*ItemRemessa
}

// SaldoTotal soma o saldo em poder da empresa em todas as linhas.
func (r *Remessa) SaldoTotal() decimal.Decimal {
	total := decimal.Zero
	for _, item := range r.Itens {
		total = total.Add(item.BalanceQty)
	}
	return total
}

// StatusCalculado deriva a situação a partir do consumo das linhas. Usado depois
// de cada movimento, para o status nunca contradizer o saldo.
//
// Encerramento manual é decisão de pessoa, não cálculo: uma remessa já encerrada
// ou cancelada permanece como está.
func (r *Remessa) StatusCalculado() StatusRemessa {
	if r.Status == StatusEncerrada || r.Status == StatusCancelada {
		return r.Status
	}
	if len(r.Itens) == 0 {
		return StatusAberta
	}
	algumConsumo := false
	saldoZerado := true
	for _, item := range r.Itens {
		if item.Consumido().IsPositive() {
			algumConsumo = true
		}
		if item.BalanceQty.IsPositive() {
			saldoZerado = false
		}
	}
	switch {
	case saldoZerado:
		return StatusEncerrada
	case algumConsumo:
		return StatusParcial
	default:
		return StatusAberta
	}
}

// ItemRemessa é uma linha da nota do cliente e o saldo dela.
type ItemRemessa struct {
	ID           int64
	EnterpriseID int64
	RemittanceID int64
	LineNumber   int32
	// Código do item COMO VEIO na nota do cliente: é com ele que o retorno fiscal
	// sai, porque é o que o cliente reconhece.
	CustomerItemCode string
	// Item interno correspondente, quando existir. Opcional de propósito: o
	// controle não depende de cadastrar peça de cliente no nosso cadastro.
	ItemCode    *int64
	Description string
	// NCM da nota de ENTRADA; o retorno tem de sair com o mesmo.
	NCM         string
	CST         *string
	UOM         string
	QtyInvoiced decimal.Decimal
	QtyReceived decimal.Decimal
	UnitValue   decimal.Decimal
	QtyReturned decimal.Decimal
	QtyLeftover decimal.Decimal
	QtyScrapped decimal.Decimal
	// BalanceQty e DivergenceQty são colunas geradas no banco: não existe estado
	// aqui para divergir do que os movimentos dizem.
	BalanceQty          decimal.Decimal
	DivergenceQty       decimal.Decimal
	DivergenceReason    *string
	DivergenceSettledBy *string
	DivergenceSettledAt *time.Time
	WarehouseID         *int64
	Address             *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Consumido é o total que já saiu desta linha.
func (i *ItemRemessa) Consumido() decimal.Decimal {
	return i.QtyReturned.Add(i.QtyLeftover).Add(i.QtyScrapped)
}

// TemDivergencia informa se o físico conferido difere da nota.
func (i *ItemRemessa) TemDivergencia() bool {
	return !i.DivergenceQty.IsZero()
}

// Movimento é um lançamento no razão do material de terceiro.
type Movimento struct {
	ID                int64
	EnterpriseID      int64
	RemittanceItemID  int64
	MovementType      TipoMovimento
	Quantity          decimal.Decimal
	UnitValue         decimal.Decimal
	CFOP              *string
	ProductionOrderID *int64
	// Nota de saída que documentou o movimento. Nulo até a nota ser emitida: o
	// movimento físico existe antes do documento fiscal.
	FiscalExitID     *int64
	ScrapDestination *DestinoSucata
	Reason           *string
	// Repetir a requisição com a mesma chave devolve o movimento original, em vez
	// de baixar o saldo duas vezes.
	IdempotencyKey string
	CreatedBy      string
	CreatedAt      time.Time
	// Estorno: preenchido quando a nota que documentou o movimento foi cancelada.
	// O movimento continua no razão — "saiu e voltou" é informação — e a quantidade
	// volta ao saldo do cliente.
	ReversedAt     *time.Time
	ReversedBy     *string
	ReversalReason *string
}

// Estornado diz se o movimento ainda sustenta baixa de saldo.
func (m *Movimento) Estornado() bool { return m.ReversedAt != nil }

// SaldoPorItem agrega, por item e cliente, quanto material de terceiro está em
// poder da empresa. É o que a tela de inventário mostra ao lado do estoque
// próprio — somando cada um no seu lugar, nunca no mesmo total.
type SaldoPorItem struct {
	CustomerCode     int64
	CustomerName     string
	CustomerItemCode string
	ItemCode         *int64
	Description      string
	UOM              string
	Balance          decimal.Decimal
	// Remessas abertas que sustentam este saldo, da mais antiga para a mais nova.
	RemessasAbertas int32
	// Prazo fiscal mais próximo entre as remessas que compõem o saldo.
	PrazoMaisProximo *time.Time
}

// EventoDeAuditoria é uma linha da trilha que os triggers da migração 000371
// gravam a cada insert, alteração ou exclusão nas três tabelas do
// beneficiamento. O levantamento da Usimac (documento 10) pede usuário, data e
// hora, operação, registro afetado, informação anterior, nova informação e
// motivo — esta estrutura é exatamente isso.
//
// Antes e Depois são o registro inteiro em JSON, do jeito que ele estava. Guardar
// o registro completo em vez dos campos alterados é o que permite responder uma
// pergunta que ninguém fez na hora de gravar; Alterados diz onde olhar.
type EventoDeAuditoria struct {
	ID           int64
	EntityType   string
	EntityID     int64
	RemittanceID *int64
	// Acao é INSERT, UPDATE ou DELETE.
	Acao      string
	Alterados []string
	Antes     []byte
	Depois    []byte
	Motivo    *string
	// AtorID pode ser nulo: correção por script ou carga não tem sessão. Perder o
	// autor é ruim, perder o fato seria pior.
	AtorID     *string
	AtorNome   *string
	AtorEmail  *string
	OcorridoEm time.Time
}
