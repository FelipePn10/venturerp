package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

type FluxoCaixaTipo string

const (
	FluxoCaixaTipoEntrada      FluxoCaixaTipo = "ENTRADA"
	FluxoCaixaTipoSaida        FluxoCaixaTipo = "SAIDA"
	FluxoCaixaTipoTransferencia FluxoCaixaTipo = "TRANSFERENCIA"
)

type FluxoCaixa struct {
	ID                    int64
	Data                  time.Time
	Tipo                  FluxoCaixaTipo
	Valor                 decimal.Decimal
	ContaBancariaID       *int64
	ContaBancariaDestinoID *int64
	ContasPagarID         *int64
	ContasReceberID       *int64
	Descricao             *string
	Conciliado            bool
	ExtratoHash           *string
	CreatedAt             time.Time
}
