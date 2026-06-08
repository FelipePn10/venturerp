package request

type UpdatePurchaseOrderDTO struct {
	Code                int64   `json:"code"`
	Status              string  `json:"status"`
	Origin              string  `json:"origin"`
	DeliveryDate        *string `json:"delivery_date,omitempty"`
	SupplierCode        *int64  `json:"supplier_code,omitempty"`
	PaymentTermCode     *int64  `json:"payment_term_code,omitempty"`
	CurrencyCode        string  `json:"currency_code"`
	ShippingAddressCode *int64  `json:"shipping_address_code,omitempty"`
	Notes               *string `json:"notes,omitempty"`
	TotalGross          float64 `json:"total_gross"`
	TotalNet            float64 `json:"total_net"`
	TotalDiscount       float64 `json:"total_discount"`
	IsFirm              bool    `json:"is_firm"`
}
