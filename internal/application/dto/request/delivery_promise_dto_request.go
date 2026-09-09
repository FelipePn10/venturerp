package request

type DeliveryPromiseLineDTO struct {
	ItemCode  int64   `json:"item_code"`
	Mask      string  `json:"mask"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

// DeliveryPromiseOccupationDTO é montado a partir da query string, não de um
// corpo JSON: os tanques chegam como `tank_code` repetido na URL. As tags ficam
// como "-" para não anunciar um contrato de corpo que o handler nunca lê.
type DeliveryPromiseOccupationDTO struct {
	FromDate      string  `json:"-"`
	ToDate        string  `json:"-"`
	DailyCapacity float64 `json:"-"`
	TankCodes     []int64 `json:"-"`
}

type DeliveryTankReservationDTO struct {
	CustomerCode          *int64                   `json:"customer_code,omitempty"`
	RequestedDeliveryDate string                   `json:"requested_delivery_date"`
	FirmDays              int                      `json:"firm_days"`
	DailyCapacity         float64                  `json:"daily_capacity"`
	VerifyStock           bool                     `json:"verify_stock"`
	Commit                bool                     `json:"commit"`
	Notes                 *string                  `json:"notes,omitempty"`
	Lines                 []DeliveryPromiseLineDTO `json:"lines"`
}

type DeliveryRescheduleBatchDTO struct {
	DeliveryFrom       string  `json:"delivery_from"`
	DeliveryTo         string  `json:"delivery_to"`
	NewDate            string  `json:"new_date"`
	CustomerCode       *int64  `json:"customer_code,omitempty"`
	RepresentativeCode *int64  `json:"representative_code,omitempty"`
	SalesOrderCodes    []int64 `json:"sales_order_codes,omitempty"`
	ItemCodes          []int64 `json:"item_codes,omitempty"`
	Reason             *string `json:"reason,omitempty"`
}
