package entity

import "time"

// ItemConsumptionAverage is the average monthly consumption of an item, derived
// from outbound stock movements over a trailing window. It feeds the reorder
// point so replenishment no longer depends on a manually maintained figure.
type ItemConsumptionAverage struct {
	ID                    int64
	ItemCode              int64
	AvgMonthlyConsumption float64
	TotalConsumed         float64
	WindowMonths          int
	CalculatedAt          time.Time
}
