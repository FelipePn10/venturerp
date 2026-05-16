package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

type TaxAssessmentStatus string

const (
	TaxStatusApurar  TaxAssessmentStatus = "APURAR"
	TaxStatusApurado TaxAssessmentStatus = "APURADO"
	TaxStatusPago    TaxAssessmentStatus = "PAGO"
)

type TaxAssessment struct {
	ID            int64
	Imposto       string
	Competencia   string
	Debitos       decimal.Decimal
	Creditos      decimal.Decimal
	SaldoDevedor  decimal.Decimal
	SaldoCredor   decimal.Decimal
	Status        TaxAssessmentStatus
	CpID          *int64
	DataVencimento *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
