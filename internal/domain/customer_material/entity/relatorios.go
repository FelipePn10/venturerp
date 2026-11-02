package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// MovimentoDoRelatorio é uma linha da movimentação de material de terceiro —
// entrada (remessa do cliente, CFOP 5901) e saída (retorno 5902, sobra e sucata
// 5903) — com a nota de origem e a de saída. É o relatório de "movimentação de
// estoque (entrada e saída)" que a Usimac pede para a rastreabilidade fiscal.
type MovimentoDoRelatorio struct {
	Movimento
	RemittanceID     int64
	CustomerCode     int64
	CustomerName     string
	NFeNumber        int64
	NFeSeries        string
	CustomerItemCode string
	Description      string
	NCM              string
	UOM              string
	// Nota de saída que documentou o movimento (nula na entrada e no ajuste).
	NotaSaidaNumero *int64
	NotaSaidaSerie  *string
	NotaSaidaStatus *string
	// Número da OF do movimento, o que está no papel da fábrica.
	ProductionOrderNumber *int64
}

// HorasDeBeneficiamento são as horas trabalhadas numa OF de beneficiamento —
// a OF cujo item leva material do cliente na estrutura. O apontamento dessa OF
// é só de horas (o material volta fiscalmente), e é com isso que a Usimac
// quantifica o trabalho feito para o terceiro.
type HorasDeBeneficiamento struct {
	ProductionOrderID int64
	OrderNumber       int64
	Status            string
	ItemCode          int64
	ItemBusinessCode  string
	ItemDescription   string
	PlannedQty        decimal.Decimal
	ProducedQty       decimal.Decimal
	// Cliente e pedido vêm do pedido de venda que originou a OF (ordem
	// planejada). OF criada à mão, sem pedido, fica sem cliente.
	CustomerCode     *int64
	CustomerName     string
	SalesOrderCode   *int64
	SalesOrderNumber *int64
	// HorasApontadas soma a duração (fim − início) dos apontamentos da OF.
	HorasApontadas decimal.Decimal
	Apontamentos   int32
	// HorasDeOperacao soma o que foi lançado nas operações do roteiro (avanço
	// de operação e coletor). As duas fontes são mostradas separadas, não
	// somadas: a mesma hora pode ter sido lançada nas duas.
	HorasDeOperacao  decimal.Decimal
	PrimeiroRegistro *time.Time
	UltimoRegistro   *time.Time
}
