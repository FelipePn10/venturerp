package entity

import "time"

type TaxScenario struct {
	ID              int64
	ScenarioName    string
	DestinationUF   *string
	DestinationType *string
	AliqICMS        float64
	DifICMSPct      float64
	CstICMS         string
	CalcDifal       bool
	AliqFCP         float64
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
