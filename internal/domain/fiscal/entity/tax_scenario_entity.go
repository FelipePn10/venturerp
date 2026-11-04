package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

type TaxScenario struct {
	ID              int64
	ScenarioName    string
	DestinationUF   *string
	DestinationType *string
	AliqICMS        decimal.Decimal
	DifICMSPct      decimal.Decimal
	CstICMS         string
	CalcDifal       bool
	AliqFCP         decimal.Decimal
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
