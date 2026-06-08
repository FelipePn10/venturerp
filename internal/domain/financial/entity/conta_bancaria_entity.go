package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ContaBancaria struct {
	ID           int64
	Banco        string
	Agencia      string
	Conta        string
	Digito       *string
	Descricao    string
	Titular      *string
	SaldoInicial decimal.Decimal
	ChavePix     *string
	TipoChavePix *string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedBy    uuid.UUID
}
