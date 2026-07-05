package response

import "time"

type DeliveryPromiseAllocationResponse struct {
	TankCode       int64     `json:"tank_code"`
	ItemCode       int64     `json:"item_code"`
	Mask           string    `json:"mask"`
	AllocationDate time.Time `json:"allocation_date"`
	Quantity       float64   `json:"quantity"`
	UnitPrice      float64   `json:"unit_price"`
	Source         string    `json:"source"`
	ReferenceCode  *int64    `json:"reference_code,omitempty"`
}

type DeliveryPromiseOccupationDayResponse struct {
	TankCode        int64                               `json:"tank_code"`
	Date            time.Time                           `json:"date"`
	Capacity        float64                             `json:"capacity"`
	Allocated       float64                             `json:"allocated"`
	Free            float64                             `json:"free"`
	OccupationPct   float64                             `json:"occupation_pct"`
	Quantity        float64                             `json:"quantity"`
	ForecastRevenue float64                             `json:"forecast_revenue"`
	Allocations     []DeliveryPromiseAllocationResponse `json:"allocations"`
	Warnings        []string                            `json:"warnings,omitempty"`
}

type DeliveryTankReservationResponse struct {
	RequestedDeliveryDate time.Time                           `json:"requested_delivery_date"`
	ExpiresAt             time.Time                           `json:"expires_at"`
	Committed             bool                                `json:"committed"`
	Allocations           []DeliveryPromiseAllocationResponse `json:"allocations"`
	Warnings              []string                            `json:"warnings,omitempty"`
}

type DeliveryRescheduleBatchResponse struct {
	UpdatedOrders int64    `json:"updated_orders"`
	UpdatedItems  int64    `json:"updated_items"`
	Skipped       []string `json:"skipped,omitempty"`
}
