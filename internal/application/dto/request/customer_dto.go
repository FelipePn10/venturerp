package request

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ─── Regions ─────────────────────────────────────────────────────────────────

type CreateRegionDTO struct {
	Description string    `json:"description"`
	UF          string    `json:"uf"`
	City        string    `json:"city"`
	CreatedBy   uuid.UUID `json:"created_by"`
}

type UpdateRegionDTO struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
	UF          string `json:"uf"`
	City        string `json:"city"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

// ─── Market Segments ──────────────────────────────────────────────────────────

type CreateMarketSegmentDTO struct {
	Description           string `json:"description"`
	ParentCode            *int64 `json:"parent_code,omitempty"`
	HasPISCOFINSRetention bool   `json:"has_pis_cofins_retention"`
	RetentionIndicator    *int16 `json:"retention_indicator,omitempty"`
}

type UpdateMarketSegmentDTO struct {
	ID                    int64  `json:"id"`
	Description           string `json:"description"`
	ParentID              *int64 `json:"parent_id,omitempty"`
	HasPISCOFINSRetention bool   `json:"has_pis_cofins_retention"`
	RetentionIndicator    *int16 `json:"retention_indicator,omitempty"`
	IsActive              *bool  `json:"is_active,omitempty"`
}

// ─── Customer Contact Types ───────────────────────────────────────────────────

type CreateContactTypeDTO struct {
	Description string `json:"description"`
}

type UpdateContactTypeDTO struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

// ─── Customer Types ───────────────────────────────────────────────────────────

type CreateCustomerTypeDTO struct {
	Code         int64  `json:"code"`
	Description  string `json:"description"`
	Category     string `json:"category"` // NORMAL | CONSUMIDOR
	DeliveryDays int16  `json:"delivery_days"`
}

type UpdateCustomerTypeDTO struct {
	ID           int64  `json:"id"`
	Description  string `json:"description"`
	Category     string `json:"category"`
	DeliveryDays int16  `json:"delivery_days"`
	IsActive     *bool  `json:"is_active,omitempty"`
}

// ─── Carriers ─────────────────────────────────────────────────────────────────

type CreateCarrierDTO struct {
	Description string `json:"description"`
}

type UpdateCarrierDTO struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

// ─── Carrier Groups ───────────────────────────────────────────────────────────

type CreateCarrierGroupDTO struct {
	Description string `json:"description"`
}

type UpdateCarrierGroupDTO struct {
	Description string `json:"description"`
}

type CarrierGroupMemberDTO struct {
	CarrierGroupCode int64 `json:"carrier_group_code"`
	CarrierCode      int64 `json:"carrier_code"`
}

// ─── Payment Conditions ───────────────────────────────────────────────────────

type CreatePaymentConditionDTO struct {
	Description  string          `json:"description"`
	CarrierCode  *int64          `json:"carrier_code,omitempty"`
	AnalysisType string          `json:"analysis_type"` // SEMPRE_ANALISA | BLOQUEIA_SEMPRE | LIBERA_SEM_ANALISE
	ParcelStart  string          `json:"parcel_start"`  // EMISSAO | PROXIMO_MES | PROXIMA_QUINZENA
	Expenses     decimal.Decimal `json:"expenses"`
	AverageTerm  int16           `json:"average_term"`
	IsSpecial    bool            `json:"is_special"`
	IsRevenue    bool            `json:"is_revenue"`
	IsAtSight    bool            `json:"is_at_sight"`
}

type UpdatePaymentConditionDTO struct {
	Description  string          `json:"description"`
	CarrierCode  *int64          `json:"carrier_code,omitempty"`
	AnalysisType string          `json:"analysis_type"`
	ParcelStart  string          `json:"parcel_start"`
	Expenses     decimal.Decimal `json:"expenses"`
	AverageTerm  int16           `json:"average_term"`
	IsSpecial    bool            `json:"is_special"`
	IsRevenue    bool            `json:"is_revenue"`
	IsAtSight    bool            `json:"is_at_sight"`
	IsActive     *bool           `json:"is_active,omitempty"`
}

type AddInstallmentDTO struct {
	PaymentConditionCode int64   `json:"payment_condition_code"`
	InstallmentNumber    int16   `json:"installment_number"`
	DueDays              int16   `json:"due_days"`
	Description          *string `json:"description,omitempty"`
	DocumentType         *string `json:"document_type,omitempty"`
	MovementType         *string `json:"movement_type,omitempty"`
	CarrierCode          *int64  `json:"carrier_code,omitempty"`
	// Percentage é quanto do total esta parcela leva. Nulo em todas = divisão em
	// partes iguais. Informado em parte delas, a condição é recusada: não há como
	// adivinhar quanto vai no resto.
	Percentage *decimal.Decimal `json:"percentage,omitempty"`
	// BaseEvent: EMISSAO (padrão), ENTRADA (no ato), ENTREGA ou FATURAMENTO.
	BaseEvent *string `json:"base_event,omitempty"`
}

// ─── Sales Tables ─────────────────────────────────────────────────────────────

type CreateSalesTableDTO struct {
	Description                string          `json:"description"`
	ValidityStart              *time.Time      `json:"validity_start,omitempty"`
	ValidityEnd                *time.Time      `json:"validity_end,omitempty"`
	ToleranceMinPct            decimal.Decimal `json:"tolerance_min_pct"`
	ToleranceMaxPct            decimal.Decimal `json:"tolerance_max_pct"`
	PriceFormation             string          `json:"price_formation"`
	DecimalPlaces              int16           `json:"decimal_places"`
	Composition                string          `json:"composition"` // EXWORK | CIF | FOB
	TableType                  string          `json:"table_type"`  // NORMAL | PROMOCIONAL
	BaseDate                   string          `json:"base_date"`   // PEDIDO | DATA_ATUAL
	AllowItemsBelowCent        bool            `json:"allow_items_below_cent"`
	ICMSInterestadualPorDentro bool            `json:"icms_interestadual_por_dentro"`
	Observation                *string         `json:"observation,omitempty"`
}

type UpdateSalesTableDTO struct {
	Code                       int64           `json:"code"`
	Description                string          `json:"description"`
	ValidityStart              *time.Time      `json:"validity_start,omitempty"`
	ValidityEnd                *time.Time      `json:"validity_end,omitempty"`
	ToleranceMinPct            decimal.Decimal `json:"tolerance_min_pct"`
	ToleranceMaxPct            decimal.Decimal `json:"tolerance_max_pct"`
	PriceFormation             string          `json:"price_formation"`
	DecimalPlaces              int16           `json:"decimal_places"`
	IsActive                   *bool           `json:"is_active,omitempty"`
	Composition                string          `json:"composition"`
	TableType                  string          `json:"table_type"`
	BaseDate                   string          `json:"base_date"`
	AllowItemsBelowCent        bool            `json:"allow_items_below_cent"`
	ICMSInterestadualPorDentro bool            `json:"icms_interestadual_por_dentro"`
	Observation                *string         `json:"observation,omitempty"`
}

// ─── Invoice Types ────────────────────────────────────────────────────────────

type CreateInvoiceTypeDTO struct {
	Description             string          `json:"description"`
	Type                    string          `json:"type"`
	StockMovement           string          `json:"stock_movement"`
	ICMSType                string          `json:"icms_type"`
	ICMSReductionPct        decimal.Decimal `json:"icms_reduction_pct"`
	GeneratesRevenue        bool            `json:"generates_revenue"`
	UpdatesInventory        bool            `json:"updates_inventory"`
	GeneratesFinancialTitle bool            `json:"generates_financial_title"`
	CalcSubstitutionTax     bool            `json:"calc_substitution_tax"`
	CalcICMSDeferral        bool            `json:"calc_icms_deferral"`
	CalcPISCOFINS           bool            `json:"calc_pis_cofins"`
	CalcDIFAL               bool            `json:"calc_difal"`
	RequiresSalesOrder      bool            `json:"requires_sales_order"`
	// NF-e / FocusNFE fields
	BaixaPedido  bool `json:"baixa_pedido"`
	ExigeSuframa bool `json:"exige_suframa"`
	// Extended fields (migration 000126)
}

type UpdateInvoiceTypeDTO struct {
	Code                    int64           `json:"code"`
	Description             string          `json:"description"`
	Type                    string          `json:"type"`
	StockMovement           string          `json:"stock_movement"`
	ICMSType                string          `json:"icms_type"`
	ICMSReductionPct        decimal.Decimal `json:"icms_reduction_pct"`
	GeneratesRevenue        bool            `json:"generates_revenue"`
	UpdatesInventory        bool            `json:"updates_inventory"`
	GeneratesFinancialTitle bool            `json:"generates_financial_title"`
	CalcSubstitutionTax     bool            `json:"calc_substitution_tax"`
	CalcICMSDeferral        bool            `json:"calc_icms_deferral"`
	CalcPISCOFINS           bool            `json:"calc_pis_cofins"`
	CalcDIFAL               bool            `json:"calc_difal"`
	RequiresSalesOrder      bool            `json:"requires_sales_order"`
	IsActive                *bool           `json:"is_active,omitempty"`
	BaixaPedido             bool            `json:"baixa_pedido"`
	ExigeSuframa            bool            `json:"exige_suframa"`
}

// ─── Sales Table Prices ────────────────────────────────────────────────────────

type CreateSalesTablePriceDTO struct {
	SalesTableID   int64           `json:"sales_table_id"`
	SalesTableCode int64           `json:"sales_table_code"`
	ItemCode       TextCode        `json:"item_code"`
	Price          decimal.Decimal `json:"price"`
	UME            *string         `json:"ume,omitempty"`
	UMC            *string         `json:"umc,omitempty"`
	PriceConv      decimal.Decimal `json:"price_conv"`
	Formula        *string         `json:"formula,omitempty"`
	Situation      string          `json:"situation"`
	Blocked        bool            `json:"blocked"`
	Observation    *string         `json:"observation,omitempty"`
	ProductLineID  *int64          `json:"product_line_id,omitempty"`
	ItemMask       *string         `json:"item_mask,omitempty"`
}

type UpdateSalesTablePriceDTO struct {
	ID            int64           `json:"id"`
	Price         decimal.Decimal `json:"price"`
	UME           *string         `json:"ume,omitempty"`
	UMC           *string         `json:"umc,omitempty"`
	PriceConv     decimal.Decimal `json:"price_conv"`
	Formula       *string         `json:"formula,omitempty"`
	Situation     string          `json:"situation"`
	Blocked       bool            `json:"blocked"`
	Observation   *string         `json:"observation,omitempty"`
	ProductLineID *int64          `json:"product_line_id,omitempty"`
	ItemMask      *string         `json:"item_mask,omitempty"`
}

type PriceSalesItemDTO struct {
	SalesTableCode int64           `json:"sales_table_code"`
	ItemCode       TextCode        `json:"item_code"`
	Quantity       decimal.Decimal `json:"quantity"`
}

type ResolveSalesTablesForItemDTO struct {
	ItemCode     string
	CustomerCode *int64
	Quantity     decimal.Decimal
	Unit         string
	Currency     string
	ReferenceAt  time.Time
}

type FormSalesPriceDTO struct {
	SalesTableCode int64           `json:"sales_table_code"`
	PolicyCode     *int64          `json:"policy_code,omitempty"`
	ItemCode       TextCode        `json:"item_code,omitempty"`
	BaseCost       decimal.Decimal `json:"base_cost"`
	MarkupPct      decimal.Decimal `json:"markup_pct"`
	MarginPct      decimal.Decimal `json:"margin_pct"`
	ExpensesPct    decimal.Decimal `json:"expenses_pct"`
	TaxesPct       decimal.Decimal `json:"taxes_pct"`
	FreightPct     decimal.Decimal `json:"freight_pct"`
	CommissionPct  decimal.Decimal `json:"commission_pct"`
	DiscountPct    decimal.Decimal `json:"discount_pct"`
}

type CreateSalesPricePolicyDTO struct {
	Description    string          `json:"description"`
	CostSource     string          `json:"cost_source"`
	Priority       int64           `json:"priority"`
	Sequence       int64           `json:"sequence"`
	PolicyScope    string          `json:"policy_scope"`
	PolicyTypes    string          `json:"policy_types"`
	MarkupPct      decimal.Decimal `json:"markup_pct"`
	MarginPct      decimal.Decimal `json:"margin_pct"`
	MaxMarginPct   decimal.Decimal `json:"max_margin_pct"`
	IdealMarginPct decimal.Decimal `json:"ideal_margin_pct"`
	MarginStepPct  decimal.Decimal `json:"margin_step_pct"`
	ExpensesPct    decimal.Decimal `json:"expenses_pct"`
	TaxesPct       decimal.Decimal `json:"taxes_pct"`
	FreightPct     decimal.Decimal `json:"freight_pct"`
	CommissionPct  decimal.Decimal `json:"commission_pct"`
	DiscountPct    decimal.Decimal `json:"discount_pct"`
	MinMarginPct   decimal.Decimal `json:"min_margin_pct"`
	MaxDiscountPct decimal.Decimal `json:"max_discount_pct"`
	IncidencesJSON json.RawMessage `json:"incidences_json"`
	SalesTableCode *int64          `json:"sales_table_code,omitempty"`
	ValidityStart  *time.Time      `json:"validity_start,omitempty"`
	ValidityEnd    *time.Time      `json:"validity_end,omitempty"`
	Observation    *string         `json:"observation,omitempty"`
}

type UpdateSalesPricePolicyDTO struct {
	Code           int64           `json:"code"`
	Description    string          `json:"description"`
	CostSource     string          `json:"cost_source"`
	Priority       int64           `json:"priority"`
	Sequence       int64           `json:"sequence"`
	PolicyScope    string          `json:"policy_scope"`
	PolicyTypes    string          `json:"policy_types"`
	MarkupPct      decimal.Decimal `json:"markup_pct"`
	MarginPct      decimal.Decimal `json:"margin_pct"`
	MaxMarginPct   decimal.Decimal `json:"max_margin_pct"`
	IdealMarginPct decimal.Decimal `json:"ideal_margin_pct"`
	MarginStepPct  decimal.Decimal `json:"margin_step_pct"`
	ExpensesPct    decimal.Decimal `json:"expenses_pct"`
	TaxesPct       decimal.Decimal `json:"taxes_pct"`
	FreightPct     decimal.Decimal `json:"freight_pct"`
	CommissionPct  decimal.Decimal `json:"commission_pct"`
	DiscountPct    decimal.Decimal `json:"discount_pct"`
	MinMarginPct   decimal.Decimal `json:"min_margin_pct"`
	MaxDiscountPct decimal.Decimal `json:"max_discount_pct"`
	IncidencesJSON json.RawMessage `json:"incidences_json"`
	SalesTableCode *int64          `json:"sales_table_code,omitempty"`
	ValidityStart  *time.Time      `json:"validity_start,omitempty"`
	ValidityEnd    *time.Time      `json:"validity_end,omitempty"`
	IsActive       *bool           `json:"is_active,omitempty"`
	Observation    *string         `json:"observation,omitempty"`
}

type GenerateSalesTablePricesDTO struct {
	SalesTableCode int64      `json:"sales_table_code"`
	PolicyCode     int64      `json:"policy_code"`
	ItemCodes      []TextCode `json:"item_codes"`
	WarehouseID    *int64     `json:"warehouse_id,omitempty"`
	Reason         *string    `json:"reason,omitempty"`
}

// ─── Commercial Policies ─────────────────────────────────────────────────────

type CreateCommercialPolicyDTO struct {
	Description            string          `json:"description"`
	Kind                   string          `json:"kind"`
	ChoiceType             string          `json:"choice_type"`
	CalcType               string          `json:"calc_type"`
	PercentValue           decimal.Decimal `json:"percent_value"`
	FixedValue             decimal.Decimal `json:"fixed_value"`
	MaxPercent             decimal.Decimal `json:"max_percent"`
	MaxValue               decimal.Decimal `json:"max_value"`
	MinGrossValue          decimal.Decimal `json:"min_gross_value"`
	MaxGrossValue          decimal.Decimal `json:"max_gross_value"`
	MinQuantity            decimal.Decimal `json:"min_quantity"`
	MaxQuantity            decimal.Decimal `json:"max_quantity"`
	Priority               int64           `json:"priority"`
	Sequence               int64           `json:"sequence"`
	Stackable              bool            `json:"stackable"`
	RequiresApproval       bool            `json:"requires_approval"`
	AppliesOnNetValue      bool            `json:"applies_on_net_value"`
	AllowManualChange      bool            `json:"allow_manual_change"`
	AllowHigherValues      bool            `json:"allow_higher_values"`
	UsedInCommission       bool            `json:"used_in_commission"`
	AppliesToItems         bool            `json:"applies_to_items"`
	SubtractCommissionBase bool            `json:"subtract_commission_base"`
	DataTypesJSON          json.RawMessage `json:"data_types_json"`
	CommissionDiscountMode string          `json:"commission_discount_mode"`
	CustomerCode           *int64          `json:"customer_code,omitempty"`
	CustomerTypeID         *int64          `json:"customer_type_id,omitempty"`
	MarketSegmentID        *int64          `json:"market_segment_id,omitempty"`
	RegionID               *int64          `json:"region_id,omitempty"`
	SalesTableID           *int64          `json:"sales_table_id,omitempty"`
	PaymentConditionID     *int64          `json:"payment_condition_id,omitempty"`
	CarrierID              *int64          `json:"carrier_id,omitempty"`
	ItemCode               *string         `json:"item_code,omitempty"`
	ItemMask               *string         `json:"item_mask,omitempty"`
	ProductLineID          *int64          `json:"product_line_id,omitempty"`
	ItemClassification     *string         `json:"item_classification,omitempty"`
	RuleJSON               json.RawMessage `json:"rule_json"`
	ValidityStart          *time.Time      `json:"validity_start,omitempty"`
	ValidityEnd            *time.Time      `json:"validity_end,omitempty"`
	Observation            *string         `json:"observation,omitempty"`
}

type UpdateCommercialPolicyDTO struct {
	Code                   int64           `json:"code"`
	Description            string          `json:"description"`
	Kind                   string          `json:"kind"`
	ChoiceType             string          `json:"choice_type"`
	CalcType               string          `json:"calc_type"`
	PercentValue           decimal.Decimal `json:"percent_value"`
	FixedValue             decimal.Decimal `json:"fixed_value"`
	MaxPercent             decimal.Decimal `json:"max_percent"`
	MaxValue               decimal.Decimal `json:"max_value"`
	MinGrossValue          decimal.Decimal `json:"min_gross_value"`
	MaxGrossValue          decimal.Decimal `json:"max_gross_value"`
	MinQuantity            decimal.Decimal `json:"min_quantity"`
	MaxQuantity            decimal.Decimal `json:"max_quantity"`
	Priority               int64           `json:"priority"`
	Sequence               int64           `json:"sequence"`
	Stackable              bool            `json:"stackable"`
	RequiresApproval       bool            `json:"requires_approval"`
	AppliesOnNetValue      bool            `json:"applies_on_net_value"`
	AllowManualChange      bool            `json:"allow_manual_change"`
	AllowHigherValues      bool            `json:"allow_higher_values"`
	UsedInCommission       bool            `json:"used_in_commission"`
	AppliesToItems         bool            `json:"applies_to_items"`
	SubtractCommissionBase bool            `json:"subtract_commission_base"`
	DataTypesJSON          json.RawMessage `json:"data_types_json"`
	CommissionDiscountMode string          `json:"commission_discount_mode"`
	CustomerCode           *int64          `json:"customer_code,omitempty"`
	CustomerTypeID         *int64          `json:"customer_type_id,omitempty"`
	MarketSegmentID        *int64          `json:"market_segment_id,omitempty"`
	RegionID               *int64          `json:"region_id,omitempty"`
	SalesTableID           *int64          `json:"sales_table_id,omitempty"`
	PaymentConditionID     *int64          `json:"payment_condition_id,omitempty"`
	CarrierID              *int64          `json:"carrier_id,omitempty"`
	ItemCode               *string         `json:"item_code,omitempty"`
	ItemMask               *string         `json:"item_mask,omitempty"`
	ProductLineID          *int64          `json:"product_line_id,omitempty"`
	ItemClassification     *string         `json:"item_classification,omitempty"`
	RuleJSON               json.RawMessage `json:"rule_json"`
	ValidityStart          *time.Time      `json:"validity_start,omitempty"`
	ValidityEnd            *time.Time      `json:"validity_end,omitempty"`
	IsActive               *bool           `json:"is_active,omitempty"`
	Observation            *string         `json:"observation,omitempty"`
}

type CommercialPolicySpecificItemDTO struct {
	PolicyCode         int64      `json:"policy_code"`
	ItemCode           *TextCode  `json:"item_code,omitempty"`
	ItemMask           *string    `json:"item_mask,omitempty"`
	ProductLineID      *int64     `json:"product_line_id,omitempty"`
	ItemClassification *string    `json:"item_classification,omitempty"`
	ValidityStart      *time.Time `json:"validity_start,omitempty"`
	ValidityEnd        *time.Time `json:"validity_end,omitempty"`
	BlockDiscount      bool       `json:"block_discount"`
	BlockSurcharge     bool       `json:"block_surcharge"`
	IgnoreItemPolicies bool       `json:"ignore_item_policies"`
	BlockManualChange  bool       `json:"block_manual_change"`
}

type CommercialPolicyLineDTO struct {
	PolicyCode     int64           `json:"policy_code"`
	LineNumber     int64           `json:"line_number"`
	SequenceNumber int64           `json:"sequence_number"`
	Description    *string         `json:"description,omitempty"`
	CalcType       string          `json:"calc_type"`
	PercentValue   decimal.Decimal `json:"percent_value"`
	FixedValue     decimal.Decimal `json:"fixed_value"`
	MinValue       decimal.Decimal `json:"min_value"`
	MaxValue       decimal.Decimal `json:"max_value"`
	VariablesJSON  json.RawMessage `json:"variables_json"`
	ValidityStart  *time.Time      `json:"validity_start,omitempty"`
	ValidityEnd    *time.Time      `json:"validity_end,omitempty"`
}

type EvaluateCommercialPoliciesDTO struct {
	GrossValue         decimal.Decimal `json:"gross_value"`
	Quantity           decimal.Decimal `json:"quantity"`
	CustomerCode       *int64          `json:"customer_code,omitempty"`
	CustomerTypeID     *int64          `json:"customer_type_id,omitempty"`
	MarketSegmentID    *int64          `json:"market_segment_id,omitempty"`
	RegionID           *int64          `json:"region_id,omitempty"`
	SalesTableID       *int64          `json:"sales_table_id,omitempty"`
	PaymentConditionID *int64          `json:"payment_condition_id,omitempty"`
	CarrierID          *int64          `json:"carrier_id,omitempty"`
	ItemCode           *TextCode       `json:"item_code,omitempty"`
	ItemMask           *string         `json:"item_mask,omitempty"`
	ProductLineID      *int64          `json:"product_line_id,omitempty"`
	ItemClassification *string         `json:"item_classification,omitempty"`
}

// ─── Tax Types ────────────────────────────────────────────────────────────────

type CreateTaxTypeDTO struct {
	Description string `json:"description"`
	IsConsumer  bool   `json:"is_consumer"`
}

// ─── Customers ────────────────────────────────────────────────────────────────

type CreateCustomerDTO struct {
	Code                  int64      `json:"code"`
	CorporateCode         *int64     `json:"corporate_code,omitempty"`
	IsCorporate           bool       `json:"is_corporate"`
	Name                  string     `json:"name"`
	TradeName             *string    `json:"trade_name,omitempty"`
	DocumentType          string     `json:"document_type"` // CNPJ | CPF | ESTRANGEIRO | ISENTO
	DocumentNumber        string     `json:"document_number"`
	StateRegistration     *string    `json:"state_registration,omitempty"`
	MunicipalRegistration *string    `json:"municipal_registration,omitempty"`
	SuframaCode           *string    `json:"suframa_code,omitempty"`
	SuframaExpiry         *time.Time `json:"suframa_expiry,omitempty"`
	RegionCode            *int64     `json:"region_code,omitempty"`
	MarketSegmentCode     *int64     `json:"market_segment_code,omitempty"`
	CustomerTypeCode      *int64     `json:"customer_type_code,omitempty"`
	CarrierCode           *int64     `json:"carrier_code,omitempty"` // portador de cobrança
	TaxTypeCode           *int64     `json:"tax_type_code,omitempty"`
	// FinalidadeCompra (REVENDA, INDUSTRIALIZACAO, USO_CONSUMO). Na alteração,
	// ausente mantém a atual e vazia ("") limpa.
	FinalidadeCompra *string `json:"finalidade_compra,omitempty"`
	Website          *string `json:"website,omitempty"`

	IEIndicator          *string `json:"ie_indicator,omitempty"` // CONTRIBUINTE | ISENTO | NAO_CONTRIBUINTE
	TaxRegime            *string `json:"tax_regime,omitempty"`   // SIMPLES | MEI | NORMAL
	MainCNAE             *string `json:"main_cnae,omitempty"`
	NFeEmail             *string `json:"nfe_email,omitempty"`
	BillingEmail         *string `json:"billing_email,omitempty"`
	RepresentativeCode   *int64  `json:"representative_code,omitempty"`
	ShippingCarrierID    *int64  `json:"shipping_carrier_id,omitempty"`
	FreightType          *string `json:"freight_type,omitempty"` // CIF | FOB | TERCEIROS | SEM_FRETE
	RequiresCustomerPO   bool    `json:"requires_customer_po"`
	RequiresQualityCert  bool    `json:"requires_quality_cert"`
	AllowsPartialBilling *bool   `json:"allows_partial_billing,omitempty"` // ausente = permite
	ReceivingNotes       *string `json:"receiving_notes,omitempty"`
	CommercialNotes      *string `json:"commercial_notes,omitempty"`
	FinancialNotes       *string `json:"financial_notes,omitempty"`

	// Só na inclusão: o limite passa pela análise de crédito; diferente do
	// sugerido exige CreditLimitReason. Na alteração, o limite muda pela
	// revisão de crédito (PUT /customers/{code}/credit-limit).
	// FormVersion: a VCLI0500 revista manda 2. Sem ele (desktop antigo), a
	// alteração preserva os campos que a tela antiga não conhece.
	FormVersion int `json:"form_version,omitempty"`

	CreditLimit       *decimal.Decimal `json:"credit_limit,omitempty"`
	CreditLimitReason string           `json:"credit_limit_reason,omitempty"`
	CreatedBy         uuid.UUID        `json:"created_by"`
}

// SetCreditLimitDTO grava um limite revisado.
type SetCreditLimitDTO struct {
	CreditLimit decimal.Decimal `json:"credit_limit"`
	Reason      string          `json:"reason"`
}

// CreditPolicyDTO é a política de crédito da empresa.
type CreditPolicyDTO struct {
	MaxLimit              decimal.Decimal `json:"max_limit"`
	NewCustomerCapitalPct decimal.Decimal `json:"new_customer_capital_pct"`
	NewCustomerCap        decimal.Decimal `json:"new_customer_cap"`
	// NewCustomerRevenuePct: ausente (desktop antigo) mantém o valor gravado.
	NewCustomerRevenuePct *decimal.Decimal `json:"new_customer_revenue_pct,omitempty"`
	OverrideTolerancePct  decimal.Decimal  `json:"override_tolerance_pct"`
	OverdueBlockDays      int              `json:"overdue_block_days"`
	HistoryMonths         int              `json:"history_months"`
	ValidityMonthsA       int              `json:"validity_months_a"`
	ValidityMonthsB       int              `json:"validity_months_b"`
	ValidityMonthsC       int              `json:"validity_months_c"`
	ValidityMonthsD       int              `json:"validity_months_d"`
	ValidityMonthsE       int              `json:"validity_months_e"`
	RegistrationCheckDays int              `json:"registration_check_days"`
	// Ausentes (desktop antigo) mantêm o valor gravado.
	CustomerRevenuePct     *decimal.Decimal `json:"customer_revenue_pct,omitempty"`
	DeclaredMonthlyRevenue *decimal.Decimal `json:"declared_monthly_revenue,omitempty"`
	AutoReviewMonths       *int             `json:"auto_review_months,omitempty"`
	AutoIncreaseMaxPct     *decimal.Decimal `json:"auto_increase_max_pct,omitempty"`
}

// FinancialStatementDTO: faturamento anual declarado pelo cliente.
type FinancialStatementDTO struct {
	AnnualRevenue decimal.Decimal `json:"annual_revenue"`
	ReferenceDate string          `json:"reference_date"`
	Source        string          `json:"source"`
	Notes         string          `json:"notes"`
}

// UpdateAddressDTO / UpdateContactDTO: edição na VCLI0500.
type UpdateAddressDTO struct {
	AddAddressDTO
}

type UpdateContactDTO struct {
	AddContactDTO
}

type BlockCustomerDTO struct {
	CustomerCode int64  `json:"customer_code"`
	Reason       string `json:"reason"`
}

// ─── Customer Addresses ───────────────────────────────────────────────────────

type AddAddressDTO struct {
	CustomerCode int64   `json:"customer_code"`
	AddressType  string  `json:"address_type"` // COBRANCA | ENTREGA | COMERCIAL | OUTRO
	ZipCode      *string `json:"zip_code,omitempty"`
	Street       *string `json:"street,omitempty"`
	Number       *string `json:"number,omitempty"`
	Complement   *string `json:"complement,omitempty"`
	Neighborhood *string `json:"neighborhood,omitempty"`
	City         *string `json:"city,omitempty"`
	UF           *string `json:"uf,omitempty"`
	Country      string  `json:"country"`
	IsDefault    bool    `json:"is_default"`
}

// ─── Customer Contacts ────────────────────────────────────────────────────────

type AddContactDTO struct {
	CustomerCode    int64   `json:"customer_code"`
	ContactTypeCode *int64  `json:"contact_type_code,omitempty"`
	Name            string  `json:"name"`
	Email           *string `json:"email,omitempty"`
	Phone           *string `json:"phone,omitempty"`
	Mobile          *string `json:"mobile,omitempty"`
	Position        *string `json:"position,omitempty"`
	IsPrimary       bool    `json:"is_primary"`
}
