package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ContaReceberStatus string

const (
	ContaReceberStatusPendente     ContaReceberStatus = "PENDENTE"
	ContaReceberStatusAprovado     ContaReceberStatus = "APROVADO"
	ContaReceberStatusRecebido     ContaReceberStatus = "RECEBIDO"
	ContaReceberStatusVencido      ContaReceberStatus = "VENCIDO"
	ContaReceberStatusBaixadoPerda ContaReceberStatus = "BAIXADO_PERDA"
	ContaReceberStatusCancelado    ContaReceberStatus = "CANCELADO"
	ContaReceberStatusRenegociado  ContaReceberStatus = "RENEGOCIADO"
)

type ContaReceber struct {
	ID              int64
	NumeroDocumento *string
	ClienteID       *int64
	FiscalExitID    *int64
	SalesOrderID    *int64

	DataLancamento  time.Time
	DataEmissao     time.Time
	DataVencimento  time.Time
	DataRecebimento *time.Time

	ValorBruto    decimal.Decimal
	Desconto      decimal.Decimal
	Juros         decimal.Decimal
	Multa         decimal.Decimal
	ValorRecebido decimal.Decimal

	ParcelaNumero int32
	ParcelaTotal  int32
	ParcelaPaiID  *int64

	ContaBancariaID *int64
	FormaPagamento  *string

	NossoNumero    *string
	LinhaDigitavel *string
	CodigoBarras   *string
	ChavePixGerada *string

	PlanoContasID *int64
	CentroCustoID *int64

	Status     ContaReceberStatus
	EmProtesto bool

	IsActive    bool
	CriadoPor   uuid.UUID
	BaixadoPor  *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
