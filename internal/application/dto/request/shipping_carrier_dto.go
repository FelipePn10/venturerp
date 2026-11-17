package request

import "github.com/shopspring/decimal"

// SalvarTransportadoraDTO é o cadastro de transportadora vindo da tela. Frota e
// regiões vêm inteiras: a tela manda o que está na grade, e o backend substitui.
type SalvarTransportadoraDTO struct {
	SupplierCode          int64                      `json:"supplier_code"`
	ANTTRNTRC             *string                    `json:"antt_rntrc"`
	ANTTExpiry            *string                    `json:"antt_expiry"`
	ShipperType           *string                    `json:"shipper_type"`
	Modal                 string                     `json:"modal"`
	IssuesCTe             *bool                      `json:"issues_cte"`
	DefaultFreightType    *string                    `json:"default_freight_type"`
	FreightMinValue       *decimal.Decimal           `json:"freight_min_value"`
	FreightKgRate         *decimal.Decimal           `json:"freight_kg_rate"`
	FreightPctValue       *decimal.Decimal           `json:"freight_pct_value"`
	GrisPct               *decimal.Decimal           `json:"gris_pct"`
	TollPer100Kg          *decimal.Decimal           `json:"toll_per_100kg"`
	InsuranceCompany      *string                    `json:"insurance_company"`
	InsurancePolicy       *string                    `json:"insurance_policy"`
	InsuranceExpiry       *string                    `json:"insurance_expiry"`
	InsuranceCoverage     *decimal.Decimal           `json:"insurance_coverage"`
	AverageLeadDays       *int16                     `json:"average_lead_days"`
	TrackingURL           *string                    `json:"tracking_url"`
	ContactName           *string                    `json:"contact_name"`
	ContactPhone          *string                    `json:"contact_phone"`
	ContactEmail          *string                    `json:"contact_email"`
	Notes                 *string                    `json:"notes"`
	CubageFactor          *decimal.Decimal           `json:"cubage_factor"`
	DispatchFee           *decimal.Decimal           `json:"dispatch_fee"`
	AcceptsDangerousGoods *bool                      `json:"accepts_dangerous_goods"`
	IsActive              *bool                      `json:"is_active"`
	Vehicles              []TransportadoraVeiculoDTO `json:"vehicles"`
	ServiceAreas          []TransportadoraRegiaoDTO  `json:"service_areas"`
}

type TransportadoraVeiculoDTO struct {
	Plate             string           `json:"plate"`
	Description       *string          `json:"description"`
	VehicleType       *string          `json:"vehicle_type"`
	Axles             *int16           `json:"axles"`
	CapacityKg        *decimal.Decimal `json:"capacity_kg"`
	CapacityM3        *decimal.Decimal `json:"capacity_m3"`
	ANTTOwner         *string          `json:"antt_owner"`
	DriverName        *string          `json:"driver_name"`
	DriverDocument    *string          `json:"driver_document"`
	DriverLicense     *string          `json:"driver_license"`
	VehicleTypeID     *int64           `json:"vehicle_type_id"`
	PlateState        *string          `json:"plate_state"`
	Renavam           *string          `json:"renavam"`
	TareKg            *decimal.Decimal `json:"tare_kg"`
	WheelType         *string          `json:"wheel_type"`
	BodyType          *string          `json:"body_type"`
	OwnerType         *string          `json:"owner_type"`
	DriverCNHCategory *string          `json:"driver_cnh_category"`
	DriverCNHExpiry   *string          `json:"driver_cnh_expiry"`
	IsActive          *bool            `json:"is_active"`
}

type TransportadoraRegiaoDTO struct {
	State          *string          `json:"state"`
	City           *string          `json:"city"`
	PostalCodeFrom *string          `json:"postal_code_from"`
	PostalCodeTo   *string          `json:"postal_code_to"`
	LeadDays       *int16           `json:"lead_days"`
	MinValue       *decimal.Decimal `json:"min_value"`
	KgRate         *decimal.Decimal `json:"kg_rate"`
	PctValue       *decimal.Decimal `json:"pct_value"`
	IsActive       *bool            `json:"is_active"`
}

// RegistrarOcorrenciaDTO é a ocorrência de entrega: é ela que transforma
// "essa transportadora atrasa" em número.
type RegistrarOcorrenciaDTO struct {
	OccurrenceDate *string          `json:"occurrence_date"`
	OccurrenceType string           `json:"occurrence_type"`
	SalesOrderCode *int64           `json:"sales_order_code"`
	ShipmentCode   *int64           `json:"shipment_code"` // código do romaneio, o que a tela mostra
	DelayDays      *int16           `json:"delay_days"`
	CostImpact     *decimal.Decimal `json:"cost_impact"`
	Description    *string          `json:"description"`
}

// SalvarTipoVeiculoDTO é o cadastro de apoio de tipo de veículo.
type SalvarTipoVeiculoDTO struct {
	Code        string           `json:"code"`
	Description string           `json:"description"`
	WheelType   *string          `json:"wheel_type"`
	BodyType    *string          `json:"body_type"`
	Axles       *int16           `json:"axles"`
	CapacityKg  *decimal.Decimal `json:"capacity_kg"`
	CapacityM3  *decimal.Decimal `json:"capacity_m3"`
	IsActive    *bool            `json:"is_active"`
}

// SalvarTipoOcorrenciaDTO é o cadastro de apoio de tipo de ocorrência.
type SalvarTipoOcorrenciaDTO struct {
	Code            string `json:"code"`
	Description     string `json:"description"`
	CountsAsFailure *bool  `json:"counts_as_failure"`
	IsActive        *bool  `json:"is_active"`
}
