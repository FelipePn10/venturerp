package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AdiantamentoTipo distinguishes advances paid to suppliers (PAGAR) from
// advances received from customers (RECEBER).
type AdiantamentoTipo string

const (
	AdiantamentoTipoPagar   AdiantamentoTipo = "PAGAR"
	AdiantamentoTipoReceber AdiantamentoTipo = "RECEBER"
)

type AdiantamentoStatus string

const (
	AdiantamentoStatusAberto    AdiantamentoStatus = "ABERTO"
	AdiantamentoStatusParcial   AdiantamentoStatus = "PARCIAL"
	AdiantamentoStatusQuitado   AdiantamentoStatus = "QUITADO"
	AdiantamentoStatusCancelado AdiantamentoStatus = "CANCELADO"
)

// Adiantamento is an advance payment with a balance that can be applied to one
// or more contas a pagar / a receber.
type Adiantamento struct {
	ID               int64
	Tipo             AdiantamentoTipo
	ParceiroID       *int64
	ContaBancariaID  int64
	NumeroDocumento  *string
	DataAdiantamento time.Time
	ValorOriginal    decimal.Decimal
	ValorUtilizado   decimal.Decimal
	Status           AdiantamentoStatus
	Descricao        *string
	IsActive         bool
	CreatedBy        uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Saldo is the remaining (unused) amount of the advance.
func (a *Adiantamento) Saldo() decimal.Decimal {
	return a.ValorOriginal.Sub(a.ValorUtilizado)
}

// AdiantamentoAplicacao records one application of an advance onto a title.
type AdiantamentoAplicacao struct {
	ID             int64
	AdiantamentoID int64
	ContaTipo      string // PAGAR | RECEBER
	ContaID        int64
	ValorAplicado  decimal.Decimal
	DataAplicacao  time.Time
	CreatedBy      uuid.UUID
	CreatedAt      time.Time
}
