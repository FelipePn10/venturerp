package entity

import "time"

// ItemConsumptionAverage is the average monthly consumption of an item, derived
// from outbound stock movements over a trailing window. It feeds the reorder
// point so replenishment no longer depends on a manually maintained figure.
// As tags são o contrato da API: esta entidade é devolvida direto pelo handler
// e, sem elas, os campos saíam em PascalCase — renomear um campo em Go quebrava
// o cliente sem aviso.
type ItemConsumptionAverage struct {
	ID                    int64     `json:"id"`
	ItemCode              int64     `json:"item_code"`
	AvgMonthlyConsumption float64   `json:"avg_monthly_consumption"`
	TotalConsumed         float64   `json:"total_consumed"`
	WindowMonths          int       `json:"window_months"`
	CalculatedAt          time.Time `json:"calculated_at"`
}
