package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ContaPagarStatus string

const (
	ContaPagarStatusPendente  ContaPagarStatus = "PENDENTE"
	ContaPagarStatusAprovado  ContaPagarStatus = "APROVADO"
	ContaPagarStatusPago      ContaPagarStatus = "PAGO"
	ContaPagarStatusVencido   ContaPagarStatus = "VENCIDO"
	ContaPagarStatusCancelado ContaPagarStatus = "CANCELADO"
)

type ContaPagarStatusAprovacao string

const (
	AprovacaoPendente  ContaPagarStatusAprovacao = "PENDENTE"
	AprovacaoAprovado  ContaPagarStatusAprovacao = "APROVADO"
	AprovacaoRejeitado ContaPagarStatusAprovacao = "REJEITADO"
)

type ContaPagar struct {
	ID              int64
	NumeroDocumento string
	TipoDocumento   string
	FornecedorID    *int64
	FiscalEntryID   *int64
	PurchaseOrderID *int64

	DataLancamento time.Time
	DataEmissao    time.Time
	DataVencimento time.Time
	DataPagamento  *time.Time

	ValorBruto decimal.Decimal
	Desconto   decimal.Decimal
	Juros      decimal.Decimal
	Multa      decimal.Decimal
	ValorPago  decimal.Decimal

	ParcelaNumero int32
	ParcelaTotal  int32
	ParcelaPaiID  *int64

	ContaBancariaID *int64
	FormaPagamento  *string

	PlanoContasID *int64
	CentroCustoID *int64

	StatusAprovacao ContaPagarStatusAprovacao
	AprovadoPor     *uuid.UUID
	DataAprovacao   *time.Time
	MotivoRejeicao  *string

	Status                   ContaPagarStatus
	AdiantamentoID           *int64
	ValorAdiantamentoAbatido decimal.Decimal

	ComprovantePath *string
	Observacao      *string

	IsActive   bool
	CriadoPor  uuid.UUID
	BaixadoPor *uuid.UUID
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
