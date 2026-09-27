package response

import (
	"time"

	"github.com/shopspring/decimal"
)

// TransportadoraResponse leva o cadastro já interpretado: rótulo do modal e do
// tipo de transportador, e os alertas do que impede transportar (habilitação ou
// seguro vencido, frota vazia, tabela em branco).
type TransportadoraResponse struct {
	ID                 int64                             `json:"id"`
	SupplierCode       int64                             `json:"supplier_code"`
	SupplierName       string                            `json:"supplier_name"`
	SupplierDocument   string                            `json:"supplier_document"`
	ANTTRNTRC          *string                           `json:"antt_rntrc,omitempty"`
	ANTTExpiry         *string                           `json:"antt_expiry,omitempty"`
	ShipperType        *string                           `json:"shipper_type,omitempty"`
	ShipperTypeLabel   *string                           `json:"shipper_type_label,omitempty"`
	Modal              string                            `json:"modal"`
	ModalLabel         string                            `json:"modal_label"`
	IssuesCTe          bool                              `json:"issues_cte"`
	DefaultFreightType *string                           `json:"default_freight_type,omitempty"`
	FreightMinValue    decimal.Decimal                   `json:"freight_min_value"`
	FreightKgRate      decimal.Decimal                   `json:"freight_kg_rate"`
	FreightPctValue    decimal.Decimal                   `json:"freight_pct_value"`
	GrisPct            decimal.Decimal                   `json:"gris_pct"`
	TollPer100Kg       decimal.Decimal                   `json:"toll_per_100kg"`
	InsuranceCompany   *string                           `json:"insurance_company,omitempty"`
	InsurancePolicy    *string                           `json:"insurance_policy,omitempty"`
	InsuranceExpiry    *string                           `json:"insurance_expiry,omitempty"`
	InsuranceCoverage  decimal.Decimal                   `json:"insurance_coverage"`
	AverageLeadDays    *int16                            `json:"average_lead_days,omitempty"`
	TrackingURL        *string                           `json:"tracking_url,omitempty"`
	ContactName        *string                           `json:"contact_name,omitempty"`
	ContactPhone       *string                           `json:"contact_phone,omitempty"`
	ContactEmail       *string                           `json:"contact_email,omitempty"`
	Notes              *string                           `json:"notes,omitempty"`
	IsActive           bool                              `json:"is_active"`
	Alertas            []string                          `json:"alerts"`
	Vehicles           []TransportadoraVeiculoResponse   `json:"vehicles"`
	ServiceAreas       []TransportadoraRegiaoResponse    `json:"service_areas"`
	Desempenho         *TransportadoraDesempenhoResponse `json:"performance,omitempty"`
	CreatedAt          time.Time                         `json:"created_at"`
	UpdatedAt          time.Time                         `json:"updated_at"`
}

type TransportadoraVeiculoResponse struct {
	ID             int64           `json:"id"`
	Plate          string          `json:"plate"`
	Description    *string         `json:"description,omitempty"`
	VehicleType    *string         `json:"vehicle_type,omitempty"`
	Axles          *int16          `json:"axles,omitempty"`
	CapacityKg     decimal.Decimal `json:"capacity_kg"`
	CapacityM3     decimal.Decimal `json:"capacity_m3"`
	ANTTOwner      *string         `json:"antt_owner,omitempty"`
	DriverName     *string         `json:"driver_name,omitempty"`
	DriverDocument *string         `json:"driver_document,omitempty"`
	DriverLicense  *string         `json:"driver_license,omitempty"`
	IsActive       bool            `json:"is_active"`
}

type TransportadoraRegiaoResponse struct {
	ID             int64           `json:"id"`
	State          *string         `json:"state,omitempty"`
	City           *string         `json:"city,omitempty"`
	PostalCodeFrom *string         `json:"postal_code_from,omitempty"`
	PostalCodeTo   *string         `json:"postal_code_to,omitempty"`
	LeadDays       int16           `json:"lead_days"`
	MinValue       decimal.Decimal `json:"min_value"`
	KgRate         decimal.Decimal `json:"kg_rate"`
	PctValue       decimal.Decimal `json:"pct_value"`
	IsActive       bool            `json:"is_active"`
}

type TransportadoraDesempenhoResponse struct {
	Ocorrencias int             `json:"occurrences"`
	AtrasoMedio decimal.Decimal `json:"average_delay_days"`
	CustoTotal  decimal.Decimal `json:"total_cost_impact"`
	UltimaData  *string         `json:"last_occurrence_date,omitempty"`
	Desde       string          `json:"since"`
}

type TransportadoraOcorrenciaResponse struct {
	ID             int64           `json:"id"`
	CarrierID      int64           `json:"carrier_id"`
	OccurrenceDate string          `json:"occurrence_date"`
	OccurrenceType string          `json:"occurrence_type"`
	SalesOrderCode *int64          `json:"sales_order_code,omitempty"`
	DelayDays      int16           `json:"delay_days"`
	CostImpact     decimal.Decimal `json:"cost_impact"`
	Description    *string         `json:"description,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// CotacaoFreteResponse é o frete aberto em componentes: mostrar só o total
// esconde justamente o que o usuário questiona.
type CotacaoFreteResponse struct {
	CarrierID       int64           `json:"carrier_id"`
	SupplierCode    int64           `json:"supplier_code"`
	SupplierName    string          `json:"supplier_name"`
	Modal           string          `json:"modal"`
	ModalLabel      string          `json:"modal_label"`
	RegiaoID        *int64          `json:"service_area_id,omitempty"`
	RegiaoDescricao string          `json:"service_area"`
	ValorPorPeso    decimal.Decimal `json:"weight_value"`
	ValorAdValorem  decimal.Decimal `json:"ad_valorem_value"`
	ValorGris       decimal.Decimal `json:"gris_value"`
	ValorPedagio    decimal.Decimal `json:"toll_value"`
	PisoAplicado    bool            `json:"minimum_applied"`
	Total           decimal.Decimal `json:"total"`
	PrazoDias       int16           `json:"lead_days"`
	PrevisaoEntrega string          `json:"estimated_delivery"`
	Alertas         []string        `json:"alerts"`
}

// ComparativoFreteResponse é a lista de cotações ordenada: a mais barata
// primeiro, e a mais rápida marcada — é a decisão que a expedição toma.
type ComparativoFreteResponse struct {
	Destino    string                 `json:"destination"`
	PesoKg     decimal.Decimal        `json:"weight_kg"`
	ValorCarga decimal.Decimal        `json:"cargo_value"`
	MaisBarata *int64                 `json:"cheapest_carrier_id,omitempty"`
	MaisRapida *int64                 `json:"fastest_carrier_id,omitempty"`
	Cotacoes   []CotacaoFreteResponse `json:"quotes"`
	NaoAtendem []string               `json:"not_served_by"`
}
