package types

type DemandType string

const (
	DemandSalesOrder    DemandType = "SALES_ORDER"
	DemandForecast      DemandType = "FORECAST"
	DemandIndependent   DemandType = "INDEPENDENT"
	DemandSafetyStock   DemandType = "SAFETY_STOCK"
	DemandReplenishment DemandType = "REPLENISHMENT"
)
