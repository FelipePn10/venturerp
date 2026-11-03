package request

import "github.com/shopspring/decimal"

// UpdatePurchaseOrderDTO é a capa inteira como a tela a edita. Situação,
// origem e totais vêm no corpo por compatibilidade, mas não são gravados daqui:
// a situação muda por aprovar/autorizar/cancelar e os totais saem das linhas.
type UpdatePurchaseOrderDTO struct {
	Code                int64           `json:"code"`
	Status              string          `json:"status"`
	Origin              string          `json:"origin"`
	EmissionDate        string          `json:"emission_date,omitempty"`
	DeliveryDate        *string         `json:"delivery_date,omitempty"`
	SupplierCode        *int64          `json:"supplier_code,omitempty"`
	PaymentTermCode     *int64          `json:"payment_term_code,omitempty"`
	CurrencyCode        string          `json:"currency_code"`
	ShippingAddressCode *int64          `json:"shipping_address_code,omitempty"`
	Notes               *string         `json:"notes,omitempty"`
	TotalGross          decimal.Decimal `json:"total_gross"`
	TotalNet            decimal.Decimal `json:"total_net"`
	TotalDiscount       decimal.Decimal `json:"total_discount"`
	IsFirm              bool            `json:"is_firm"`

	PriceTableCode         *int64          `json:"price_table_code,omitempty"`
	InvoiceTypeCode        *int64          `json:"invoice_type_code,omitempty"`
	FinancialAccount       *string         `json:"financial_account,omitempty"`
	FreightType            string          `json:"freight_type,omitempty"`
	FreightValueType       *string         `json:"freight_value_type,omitempty"`
	FreightValueMode       *string         `json:"freight_value_mode,omitempty"`
	FreightValue           decimal.Decimal `json:"freight_value,omitempty"`
	CarrierCode            *int64          `json:"carrier_code,omitempty"`
	RedispatchCarrierCode  *int64          `json:"redispatch_carrier_code,omitempty"`
	RedispatchFreightType  *string         `json:"redispatch_freight_type,omitempty"`
	RedispatchFreightValue decimal.Decimal `json:"redispatch_freight_value,omitempty"`
	AdvanceDate            *string         `json:"advance_date,omitempty"`
	AdvanceValue           decimal.Decimal `json:"advance_value,omitempty"`
	IncotermCode           *string         `json:"incoterm_code,omitempty"`
	ShipmentDate           *string         `json:"shipment_date,omitempty"`
	TalaoNumber            *string         `json:"talao_number,omitempty"`
}

// UpdatePurchaseOrderItemDTO altera uma linha de pedido ainda não aprovado.
// Campo ausente mantém o valor atual.
type UpdatePurchaseOrderItemDTO struct {
	PurchaseOrderCode        int64            `json:"-"`
	ItemLineCode             int64            `json:"-"`
	RequestedQty             *decimal.Decimal `json:"requested_qty,omitempty"`
	UnitPrice                *decimal.Decimal `json:"unit_price,omitempty"`
	DiscountPct              *decimal.Decimal `json:"discount_pct,omitempty"`
	IPIPct                   *decimal.Decimal `json:"ipi_pct,omitempty"`
	ICMSPct                  *decimal.Decimal `json:"icms_pct,omitempty"`
	TolerancePct             *decimal.Decimal `json:"tolerance_pct,omitempty"`
	WarehouseID              *int64           `json:"warehouse_id,omitempty"`
	CostCenterCode           *int64           `json:"cost_center_code,omitempty"`
	DeliveryDate             *string          `json:"delivery_date,omitempty"`
	PromisedDate             *string          `json:"promised_date,omitempty"`
	UtilizationType          *string          `json:"utilization_type,omitempty"`
	FiscalClassificationCode *int64           `json:"fiscal_classification_code,omitempty"`
	InvoiceTypeCode          *int64           `json:"invoice_type_code,omitempty"`
	Notes                    *string          `json:"notes,omitempty"`
}

// CancelPurchaseOrderItemDTO remove a linha (pedido não aprovado) ou elimina
// o saldo que falta receber (pedido aprovado).
type CancelPurchaseOrderItemDTO struct {
	PurchaseOrderCode int64  `json:"-"`
	ItemLineCode      int64  `json:"-"`
	Motivo            string `json:"motivo"`
}
