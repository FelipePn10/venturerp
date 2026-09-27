package request

// SalvarTransportadoraDTO é o cadastro de transportadora vindo da tela. Frota e
// regiões vêm inteiras: a tela manda o que está na grade, e o backend substitui.
type SalvarTransportadoraDTO struct {
	SupplierCode       int64                      `json:"supplier_code"`
	ANTTRNTRC          *string                    `json:"antt_rntrc"`
	ANTTExpiry         *string                    `json:"antt_expiry"`
	ShipperType        *string                    `json:"shipper_type"`
	Modal              string                     `json:"modal"`
	IssuesCTe          *bool                      `json:"issues_cte"`
	DefaultFreightType *string                    `json:"default_freight_type"`
	FreightMinValue    *float64                   `json:"freight_min_value"`
	FreightKgRate      *float64                   `json:"freight_kg_rate"`
	FreightPctValue    *float64                   `json:"freight_pct_value"`
	GrisPct            *float64                   `json:"gris_pct"`
	TollPer100Kg       *float64                   `json:"toll_per_100kg"`
	InsuranceCompany   *string                    `json:"insurance_company"`
	InsurancePolicy    *string                    `json:"insurance_policy"`
	InsuranceExpiry    *string                    `json:"insurance_expiry"`
	InsuranceCoverage  *float64                   `json:"insurance_coverage"`
	AverageLeadDays    *int16                     `json:"average_lead_days"`
	TrackingURL        *string                    `json:"tracking_url"`
	ContactName        *string                    `json:"contact_name"`
	ContactPhone       *string                    `json:"contact_phone"`
	ContactEmail       *string                    `json:"contact_email"`
	Notes              *string                    `json:"notes"`
	IsActive           *bool                      `json:"is_active"`
	Vehicles           []TransportadoraVeiculoDTO `json:"vehicles"`
	ServiceAreas       []TransportadoraRegiaoDTO  `json:"service_areas"`
}

type TransportadoraVeiculoDTO struct {
	Plate          string   `json:"plate"`
	Description    *string  `json:"description"`
	VehicleType    *string  `json:"vehicle_type"`
	Axles          *int16   `json:"axles"`
	CapacityKg     *float64 `json:"capacity_kg"`
	CapacityM3     *float64 `json:"capacity_m3"`
	ANTTOwner      *string  `json:"antt_owner"`
	DriverName     *string  `json:"driver_name"`
	DriverDocument *string  `json:"driver_document"`
	DriverLicense  *string  `json:"driver_license"`
	IsActive       *bool    `json:"is_active"`
}

type TransportadoraRegiaoDTO struct {
	State          *string  `json:"state"`
	City           *string  `json:"city"`
	PostalCodeFrom *string  `json:"postal_code_from"`
	PostalCodeTo   *string  `json:"postal_code_to"`
	LeadDays       *int16   `json:"lead_days"`
	MinValue       *float64 `json:"min_value"`
	KgRate         *float64 `json:"kg_rate"`
	PctValue       *float64 `json:"pct_value"`
	IsActive       *bool    `json:"is_active"`
}

// RegistrarOcorrenciaDTO é a ocorrência de entrega: é ela que transforma
// "essa transportadora atrasa" em número.
type RegistrarOcorrenciaDTO struct {
	OccurrenceDate *string  `json:"occurrence_date"`
	OccurrenceType string   `json:"occurrence_type"`
	SalesOrderCode *int64   `json:"sales_order_code"`
	DelayDays      *int16   `json:"delay_days"`
	CostImpact     *float64 `json:"cost_impact"`
	Description    *string  `json:"description"`
}
