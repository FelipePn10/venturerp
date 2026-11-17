package response

import (
	"github.com/shopspring/decimal"

	"encoding/json"
	"time"
)

// ─── Region ──────────────────────────────────────────────────────────────────

type RegionResponse struct {
	ID          int64  `json:"id"`
	Code        int64  `json:"code"`
	Description string `json:"description"`
	UF          string `json:"uf"`
	City        string `json:"city"`
	IsActive    bool   `json:"is_active"`
}

// ─── Market Segment ───────────────────────────────────────────────────────────

type MarketSegmentResponse struct {
	ID                    int64  `json:"id"`
	Code                  int64  `json:"code"`
	Description           string `json:"description"`
	ParentID              *int64 `json:"parent_id,omitempty"`
	HasPISCOFINSRetention bool   `json:"has_pis_cofins_retention"`
	RetentionIndicator    *int16 `json:"retention_indicator,omitempty"`
	IsActive              bool   `json:"is_active"`
}

// ─── Customer Contact Type ────────────────────────────────────────────────────

type ContactTypeResponse struct {
	ID          int64  `json:"id"`
	Code        int64  `json:"code"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

// ─── Customer Type ────────────────────────────────────────────────────────────

type CustomerTypeResponse struct {
	ID           int64  `json:"id"`
	Code         int64  `json:"code"`
	Description  string `json:"description"`
	Category     string `json:"category"`
	DeliveryDays int16  `json:"delivery_days"`
	IsActive     bool   `json:"is_active"`
}

// ─── Carrier Group ────────────────────────────────────────────────────────────

type CarrierGroupResponse struct {
	ID          int64  `json:"id"`
	Code        int64  `json:"code"`
	Description string `json:"description"`
}

// ─── Carrier ─────────────────────────────────────────────────────────────────

type CarrierResponse struct {
	ID                int64  `json:"id"`
	Code              int64  `json:"code"`
	Description       string `json:"description"`
	BillingType       string `json:"billing_type"`
	UsesCreditLimit   bool   `json:"uses_credit_limit"`
	ConsiderAvailable bool   `json:"consider_available"`
	PostponeDueDate   bool   `json:"postpone_due_date"`
	ReceiptDays       int16  `json:"receipt_days"`
	PaymentDays       int16  `json:"payment_days"`
	IsActive          bool   `json:"is_active"`
}

// ─── Payment Condition ────────────────────────────────────────────────────────

type PaymentConditionResponse struct {
	ID           int64                 `json:"id"`
	Code         int64                 `json:"code"`
	Description  string                `json:"description"`
	CarrierID    *int64                `json:"carrier_id,omitempty"`
	AnalysisType string                `json:"analysis_type"`
	ParcelStart  string                `json:"parcel_start"`
	Expenses     float64               `json:"expenses"`
	AverageTerm  int16                 `json:"average_term"`
	IsSpecial    bool                  `json:"is_special"`
	IsRevenue    bool                  `json:"is_revenue"`
	IsAtSight    bool                  `json:"is_at_sight"`
	IsActive     bool                  `json:"is_active"`
	Installments []InstallmentResponse `json:"installments,omitempty"`
}

type InstallmentResponse struct {
	ID                int64   `json:"id"`
	InstallmentNumber int16   `json:"installment_number"`
	DueDays           int16   `json:"due_days"`
	Description       *string `json:"description,omitempty"`
	DocumentType      *string `json:"document_type,omitempty"`
	MovementType      *string `json:"movement_type,omitempty"`
	CarrierID         *int64  `json:"carrier_id,omitempty"`
	// Percentage é quanto do total a parcela leva; nulo em todas = partes iguais.
	Percentage *float64 `json:"percentage,omitempty"`
	// BaseEvent: EMISSAO, ENTRADA, ENTREGA ou FATURAMENTO.
	BaseEvent string `json:"base_event,omitempty"`
	// Aviso é o que falta para a condição fechar 100%. Não é erro: a parcela foi
	// gravada, e quem monta 30/20/25/25 passa por 30%, 50% e 75% antes de fechar.
	Aviso *string `json:"warning,omitempty"`
}

// ParcelaPlanoResponse é uma linha do plano de pagamento já resolvida em
// dinheiro e data — o que o cliente lê na proposta.
type ParcelaPlanoResponse struct {
	Numero       int16   `json:"numero"`
	Percentual   float64 `json:"percentual"`
	Valor        float64 `json:"valor"`
	Vencimento   string  `json:"vencimento"`
	DiasPrazo    int16   `json:"dias_prazo"`
	Evento       string  `json:"evento"`
	EventoRotulo string  `json:"evento_rotulo"`
	Descricao    string  `json:"descricao"`
	// Estimado marca o vencimento contado de um evento cuja data ainda não
	// existe (entrega/faturamento): é projeção pela emissão, não compromisso.
	Estimado     bool    `json:"estimado"`
	DocumentType *string `json:"document_type,omitempty"`
}

// PlanoDePagamentoResponse é o plano inteiro de um documento.
type PlanoDePagamentoResponse struct {
	CondicaoCode      int64                  `json:"condicao_code"`
	CondicaoDescricao string                 `json:"condicao_descricao"`
	Total             float64                `json:"total"`
	Parcelas          []ParcelaPlanoResponse `json:"parcelas"`
	// Aviso explica por que o plano pode estar incompleto (condição sem
	// percentual fechado, data de entrega ausente). Vazio = plano firme.
	Aviso string `json:"aviso,omitempty"`
}

// ─── Sales Table ──────────────────────────────────────────────────────────────

type SalesTableResponse struct {
	ID                         int64      `json:"id"`
	Code                       int64      `json:"code"`
	Description                string     `json:"description"`
	ValidityStart              *time.Time `json:"validity_start,omitempty"`
	ValidityEnd                *time.Time `json:"validity_end,omitempty"`
	ToleranceMinPct            float64    `json:"tolerance_min_pct"`
	ToleranceMaxPct            float64    `json:"tolerance_max_pct"`
	PriceFormation             string     `json:"price_formation"`
	DecimalPlaces              int16      `json:"decimal_places"`
	IsActive                   bool       `json:"is_active"`
	Composition                string     `json:"composition"`
	TableType                  string     `json:"table_type"`
	BaseDate                   string     `json:"base_date"`
	AllowItemsBelowCent        bool       `json:"allow_items_below_cent"`
	ICMSInterestadualPorDentro bool       `json:"icms_interestadual_por_dentro"`
	Observation                *string    `json:"observation,omitempty"`
}

// ─── Invoice Type ─────────────────────────────────────────────────────────────

type InvoiceTypeResponse struct {
	ID                      int64   `json:"id"`
	Code                    int64   `json:"code"`
	Description             string  `json:"description"`
	Type                    string  `json:"type"`
	StockMovement           string  `json:"stock_movement"`
	ICMSType                string  `json:"icms_type"`
	ICMSPct                 float64 `json:"icms_pct"`
	ICMSReductionPct        float64 `json:"icms_reduction_pct"`
	IPIPct                  float64 `json:"ipi_pct"`
	PISPct                  float64 `json:"pis_pct"`
	COFINSPct               float64 `json:"cofins_pct"`
	ISSQNPct                float64 `json:"issqn_pct"`
	IRPct                   float64 `json:"ir_pct"`
	CSLLPct                 float64 `json:"csll_pct"`
	INSSPct                 float64 `json:"inss_pct"`
	GeneratesRevenue        bool    `json:"generates_revenue"`
	UpdatesInventory        bool    `json:"updates_inventory"`
	GeneratesFinancialTitle bool    `json:"generates_financial_title"`
	ConsidersGoals          bool    `json:"considers_goals"`
	CalcSubstitutionTax     bool    `json:"calc_substitution_tax"`
	CalcICMSDeferral        bool    `json:"calc_icms_deferral"`
	CalcPISCOFINS           bool    `json:"calc_pis_cofins"`
	CalcDIFAL               bool    `json:"calc_difal"`
	RequiresSalesOrder      bool    `json:"requires_sales_order"`
	ListsFiscalBooks        bool    `json:"lists_fiscal_books"`
	IsActive                bool    `json:"is_active"`
	// NF-e / FocusNFE fields
	ModelNF            *string `json:"model_nf,omitempty"`
	CSTICMS            *string `json:"cst_icms,omitempty"`
	CSOSNTICMS         *string `json:"csosn_icms,omitempty"`
	CSTIPI             *string `json:"cst_ipi,omitempty"`
	CSTPIS             *string `json:"cst_pis,omitempty"`
	CSTCOFINS          *string `json:"cst_cofins,omitempty"`
	BaixaPedido        bool    `json:"baixa_pedido"`
	GeraTituloDev      bool    `json:"gera_titulo_dev"`
	ExigeSuframa       bool    `json:"exige_suframa"`
	IRPctPresumption   float64 `json:"ir_pct_presumption"`
	CSLLPctPresumption float64 `json:"csll_pct_presumption"`
	// Extended fields (migration 000126)
	DescriptionNF            *string   `json:"description_nf,omitempty"`
	ImpostosNFe              *string   `json:"impostos_nfe,omitempty"`
	CFOPId                   *int64    `json:"cfop_id,omitempty"`
	DispositivoLegalIPIId    *int64    `json:"dispositivo_legal_ipi_id,omitempty"`
	DispositivoLegalICMSId   *int64    `json:"dispositivo_legal_icms_id,omitempty"`
	DispositivoLegalICMSSTId *int64    `json:"dispositivo_legal_icms_st_id,omitempty"`
	DispositivoLegalPISId    *int64    `json:"dispositivo_legal_pis_id,omitempty"`
	DispositivoLegalCOFINSId *int64    `json:"dispositivo_legal_cofins_id,omitempty"`
	HierarchyIPI             *string   `json:"hierarchy_ipi,omitempty"`
	HierarchyICMS            *string   `json:"hierarchy_icms,omitempty"`
	HierarchyICMSST          *string   `json:"hierarchy_icms_st,omitempty"`
	HierarchyPIS             *string   `json:"hierarchy_pis,omitempty"`
	HierarchyCOFINS          *string   `json:"hierarchy_cofins,omitempty"`
	IPITransferSalesTableId  *int64    `json:"ipi_transfer_sales_table_id,omitempty"`
	ListaValorContabil       bool      `json:"lista_valor_contabil"`
	ListaRegistroSaida       bool      `json:"lista_registro_saida"`
	ListaICMSIPI             bool      `json:"lista_icms_ipi"`
	SintegraSpedFiscal       bool      `json:"sintegra_sped_fiscal"`
	CalcFomentar             bool      `json:"calc_fomentar"`
	ExcecaoFomentar          bool      `json:"excecao_fomentar"`
	CompRessRetST            bool      `json:"comp_ress_ret_st"`
	CalcReducao              bool      `json:"calc_reducao"`
	ComplementoItens         bool      `json:"complemento_itens"`
	BuscaTipoNF              bool      `json:"busca_tipo_nf"`
	ICMSSTUltEntrada         bool      `json:"icms_st_ult_entrada"`
	SomenteConsultaLotes     bool      `json:"somente_consulta_lotes"`
	CalcImpIBPT              bool      `json:"calc_imp_ibpt"`
	CredPresumidoICMS        bool      `json:"cred_presumido_icms"`
	CIAP                     bool      `json:"ciap"`
	VlrAgregadoBaseSubst     bool      `json:"vlr_agregado_base_subst"`
	ContratoFacon            bool      `json:"contrato_facon"`
	DescICMSLicitacoes       bool      `json:"desc_icms_licitacoes"`
	Sisdeclara               bool      `json:"sisdeclara"`
	CodClasTrib              *string   `json:"cod_clas_trib,omitempty"`
	CodClasTribTribReg       *string   `json:"cod_clas_trib_trib_reg,omitempty"`
	CodMotivoRestCompICMSST  *string   `json:"cod_motivo_rest_comp_icms_st,omitempty"`
	CodBeneficioFiscal       *string   `json:"cod_beneficio_fiscal,omitempty"`
	CreatedAt                time.Time `json:"created_at"`
}

// ─── Sales Table Price ────────────────────────────────────────────────────────

type SalesTablePriceResponse struct {
	ID            int64     `json:"id"`
	SalesTableID  int64     `json:"sales_table_id"`
	ItemCode      string    `json:"item_code"`
	Price         float64   `json:"price"`
	UME           *string   `json:"ume,omitempty"`
	UMC           *string   `json:"umc,omitempty"`
	PriceConv     float64   `json:"price_conv"`
	Formula       *string   `json:"formula,omitempty"`
	Situation     string    `json:"situation"`
	Blocked       bool      `json:"blocked"`
	Observation   *string   `json:"observation,omitempty"`
	ProductLineID *int64    `json:"product_line_id,omitempty"`
	ItemMask      *string   `json:"item_mask,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type SalesItemPricingResponse struct {
	SalesTableCode int64   `json:"sales_table_code"`
	SalesTableID   int64   `json:"sales_table_id"`
	ItemCode       string  `json:"item_code"`
	UnitPrice      float64 `json:"unit_price"`
	Quantity       float64 `json:"quantity"`
	TotalGross     float64 `json:"total_gross"`
	Source         string  `json:"source"`
	Situation      string  `json:"situation"`
	Blocked        bool    `json:"blocked"`
	UME            *string `json:"ume,omitempty"`
	UMC            *string `json:"umc,omitempty"`
	Formula        *string `json:"formula,omitempty"`
}

type SalesTableCandidateResponse struct {
	SalesTableCode int64   `json:"sales_table_code"`
	Description    string  `json:"description"`
	BasePrice      float64 `json:"base_price"`
	AppliedPrice   float64 `json:"applied_price"`
	Unit           string  `json:"unit,omitempty"`
	Currency       string  `json:"currency"`
	Source         string  `json:"source"`
	Priority       int     `json:"priority"`
}

type SalesTableResolutionResponse struct {
	AutoSelected bool                          `json:"auto_selected"`
	Selected     *SalesTableCandidateResponse  `json:"selected,omitempty"`
	Candidates   []SalesTableCandidateResponse `json:"candidates"`
}

type SalesPriceFormationResponse struct {
	SalesTableCode          int64   `json:"sales_table_code"`
	ItemCode                string  `json:"item_code,omitempty"`
	BaseCost                float64 `json:"base_cost"`
	SuggestedPrice          float64 `json:"suggested_price"`
	MarkupPct               float64 `json:"markup_pct"`
	MarginPct               float64 `json:"margin_pct"`
	ExpensesPct             float64 `json:"expenses_pct"`
	TaxesPct                float64 `json:"taxes_pct"`
	FreightPct              float64 `json:"freight_pct"`
	CommissionPct           float64 `json:"commission_pct"`
	DiscountPct             float64 `json:"discount_pct"`
	ContributionMarginPct   float64 `json:"contribution_margin_pct"`
	ContributionMarginValue float64 `json:"contribution_margin_value"`
	DecimalPlaces           int16   `json:"decimal_places"`
}

type SalesPricePolicyResponse struct {
	ID             int64           `json:"id"`
	Code           int64           `json:"code"`
	Description    string          `json:"description"`
	CostSource     string          `json:"cost_source"`
	Priority       int64           `json:"priority"`
	Sequence       int64           `json:"sequence"`
	PolicyScope    string          `json:"policy_scope"`
	PolicyTypes    string          `json:"policy_types"`
	MarkupPct      float64         `json:"markup_pct"`
	MarginPct      float64         `json:"margin_pct"`
	MaxMarginPct   float64         `json:"max_margin_pct"`
	IdealMarginPct float64         `json:"ideal_margin_pct"`
	MarginStepPct  float64         `json:"margin_step_pct"`
	ExpensesPct    float64         `json:"expenses_pct"`
	TaxesPct       float64         `json:"taxes_pct"`
	FreightPct     float64         `json:"freight_pct"`
	CommissionPct  float64         `json:"commission_pct"`
	DiscountPct    float64         `json:"discount_pct"`
	MinMarginPct   float64         `json:"min_margin_pct"`
	MaxDiscountPct float64         `json:"max_discount_pct"`
	IncidencesJSON json.RawMessage `json:"incidences_json,omitempty"`
	SalesTableID   *int64          `json:"sales_table_id,omitempty"`
	ValidityStart  *time.Time      `json:"validity_start,omitempty"`
	ValidityEnd    *time.Time      `json:"validity_end,omitempty"`
	IsActive       bool            `json:"is_active"`
	Observation    *string         `json:"observation,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type GeneratedSalesTablePriceResponse struct {
	ItemCode   string                   `json:"item_code"`
	BaseCost   float64                  `json:"base_cost"`
	CostSource string                   `json:"cost_source"`
	OldPrice   *float64                 `json:"old_price,omitempty"`
	Price      float64                  `json:"price"`
	HistoryID  int64                    `json:"history_id"`
	PriceRow   *SalesTablePriceResponse `json:"price_row"`
}

type GenerateSalesTablePricesResponse struct {
	SalesTableCode int64                              `json:"sales_table_code"`
	PolicyCode     int64                              `json:"policy_code"`
	Generated      []GeneratedSalesTablePriceResponse `json:"generated"`
	Warnings       []string                           `json:"warnings,omitempty"`
}

type SalesTablePriceHistoryResponse struct {
	ID                int64     `json:"id"`
	SalesTablePriceID *int64    `json:"sales_table_price_id,omitempty"`
	SalesTableID      int64     `json:"sales_table_id"`
	SalesTableCode    int64     `json:"sales_table_code"`
	ItemCode          string    `json:"item_code"`
	OldPrice          *float64  `json:"old_price,omitempty"`
	NewPrice          float64   `json:"new_price"`
	BaseCost          *float64  `json:"base_cost,omitempty"`
	Source            string    `json:"source"`
	PolicyCode        *int64    `json:"policy_code,omitempty"`
	Reason            *string   `json:"reason,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// ─── Commercial Policy ───────────────────────────────────────────────────────

type CommercialPolicyResponse struct {
	ID                     int64                          `json:"id"`
	Code                   int64                          `json:"code"`
	Description            string                         `json:"description"`
	Kind                   string                         `json:"kind"`
	ChoiceType             string                         `json:"choice_type"`
	CalcType               string                         `json:"calc_type"`
	PercentValue           float64                        `json:"percent_value"`
	FixedValue             float64                        `json:"fixed_value"`
	MaxPercent             float64                        `json:"max_percent"`
	MaxValue               float64                        `json:"max_value"`
	MinGrossValue          float64                        `json:"min_gross_value"`
	MaxGrossValue          float64                        `json:"max_gross_value"`
	MinQuantity            float64                        `json:"min_quantity"`
	MaxQuantity            float64                        `json:"max_quantity"`
	Priority               int64                          `json:"priority"`
	Sequence               int64                          `json:"sequence"`
	Stackable              bool                           `json:"stackable"`
	RequiresApproval       bool                           `json:"requires_approval"`
	AppliesOnNetValue      bool                           `json:"applies_on_net_value"`
	AllowManualChange      bool                           `json:"allow_manual_change"`
	AllowHigherValues      bool                           `json:"allow_higher_values"`
	UsedInCommission       bool                           `json:"used_in_commission"`
	AppliesToItems         bool                           `json:"applies_to_items"`
	SubtractCommissionBase bool                           `json:"subtract_commission_base"`
	DataTypesJSON          json.RawMessage                `json:"data_types_json,omitempty"`
	CommissionDiscountMode string                         `json:"commission_discount_mode"`
	CustomerCode           *int64                         `json:"customer_code,omitempty"`
	CustomerTypeID         *int64                         `json:"customer_type_id,omitempty"`
	MarketSegmentID        *int64                         `json:"market_segment_id,omitempty"`
	RegionID               *int64                         `json:"region_id,omitempty"`
	SalesTableID           *int64                         `json:"sales_table_id,omitempty"`
	PaymentConditionID     *int64                         `json:"payment_condition_id,omitempty"`
	CarrierID              *int64                         `json:"carrier_id,omitempty"`
	ItemCode               *string                        `json:"item_code,omitempty"`
	ItemMask               *string                        `json:"item_mask,omitempty"`
	ProductLineID          *int64                         `json:"product_line_id,omitempty"`
	ItemClassification     *string                        `json:"item_classification,omitempty"`
	RuleJSON               json.RawMessage                `json:"rule_json,omitempty"`
	ValidityStart          *time.Time                     `json:"validity_start,omitempty"`
	ValidityEnd            *time.Time                     `json:"validity_end,omitempty"`
	IsActive               bool                           `json:"is_active"`
	Observation            *string                        `json:"observation,omitempty"`
	CreatedAt              time.Time                      `json:"created_at"`
	UpdatedAt              time.Time                      `json:"updated_at"`
	Lines                  []CommercialPolicyLineResponse `json:"lines,omitempty"`
}

type CommercialPolicyLineResponse struct {
	ID             int64           `json:"id"`
	PolicyID       int64           `json:"policy_id"`
	LineNumber     int64           `json:"line_number"`
	SequenceNumber int64           `json:"sequence_number"`
	Description    *string         `json:"description,omitempty"`
	CalcType       string          `json:"calc_type"`
	PercentValue   float64         `json:"percent_value"`
	FixedValue     float64         `json:"fixed_value"`
	MinValue       float64         `json:"min_value"`
	MaxValue       float64         `json:"max_value"`
	VariablesJSON  json.RawMessage `json:"variables_json,omitempty"`
	ValidityStart  *time.Time      `json:"validity_start,omitempty"`
	ValidityEnd    *time.Time      `json:"validity_end,omitempty"`
	IsActive       bool            `json:"is_active"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CommercialPolicySpecificItemResponse struct {
	ID                 int64      `json:"id"`
	PolicyID           int64      `json:"policy_id"`
	ItemCode           *string    `json:"item_code,omitempty"`
	ItemMask           *string    `json:"item_mask,omitempty"`
	ProductLineID      *int64     `json:"product_line_id,omitempty"`
	ItemClassification *string    `json:"item_classification,omitempty"`
	ValidityStart      *time.Time `json:"validity_start,omitempty"`
	ValidityEnd        *time.Time `json:"validity_end,omitempty"`
	BlockDiscount      bool       `json:"block_discount"`
	BlockSurcharge     bool       `json:"block_surcharge"`
	IgnoreItemPolicies bool       `json:"ignore_item_policies"`
	BlockManualChange  bool       `json:"block_manual_change"`
	CreatedAt          time.Time  `json:"created_at"`
}

type CommercialPolicyEffectResponse struct {
	PolicyCode       int64           `json:"policy_code"`
	PolicyID         int64           `json:"policy_id"`
	Description      string          `json:"description"`
	Kind             string          `json:"kind"`
	CalcType         string          `json:"calc_type"`
	PercentValue     decimal.Decimal `json:"percent_value"`
	FixedValue       decimal.Decimal `json:"fixed_value"`
	AppliedValue     decimal.Decimal `json:"applied_value"`
	RequiresApproval bool            `json:"requires_approval"`
	Stackable        bool            `json:"stackable"`
}

type CommercialPolicyEvaluationResponse struct {
	GrossValue       decimal.Decimal                  `json:"gross_value"`
	DiscountValue    decimal.Decimal                  `json:"discount_value"`
	SurchargeValue   decimal.Decimal                  `json:"surcharge_value"`
	FreightValue     decimal.Decimal                  `json:"freight_value"`
	CommissionValue  decimal.Decimal                  `json:"commission_value"`
	NetValue         decimal.Decimal                  `json:"net_value"`
	RequiresApproval bool                             `json:"requires_approval"`
	Effects          []CommercialPolicyEffectResponse `json:"effects"`
}

// ─── Tax Type ─────────────────────────────────────────────────────────────────

type TaxTypeResponse struct {
	ID                            int64  `json:"id"`
	Code                          int64  `json:"code"`
	Description                   string `json:"description"`
	IPIBaseTotalItems             bool   `json:"ipi_base_total_items"`
	IPIBaseSubtractDiscount       bool   `json:"ipi_base_subtract_discount"`
	IPIBaseAddFreight             bool   `json:"ipi_base_add_freight"`
	IPIBaseAddExpenses            bool   `json:"ipi_base_add_expenses"`
	ICMSBaseTotalItems            bool   `json:"icms_base_total_items"`
	ICMSBaseSubtractDiscount      bool   `json:"icms_base_subtract_discount"`
	ICMSBaseAddFreight            bool   `json:"icms_base_add_freight"`
	ICMSBaseAddIPI                bool   `json:"icms_base_add_ipi"`
	ICMSBaseAddExpenses           bool   `json:"icms_base_add_expenses"`
	PISCOFINSBaseTotalItems       bool   `json:"pis_cofins_base_total_items"`
	PISCOFINSBaseSubtractDiscount bool   `json:"pis_cofins_base_subtract_discount"`
	PISCOFINSBaseAddFreight       bool   `json:"pis_cofins_base_add_freight"`
	PISCOFINSBaseAddInsurance     bool   `json:"pis_cofins_base_add_insurance"`
	PISCOFINSBaseAddExpenses      bool   `json:"pis_cofins_base_add_expenses"`
	CSLLBaseTotalItems            bool   `json:"csll_base_total_items"`
	CSLLBaseSubtractDiscount      bool   `json:"csll_base_subtract_discount"`
	CSLLBaseAddFreight            bool   `json:"csll_base_add_freight"`
	IRBaseTotalItems              bool   `json:"ir_base_total_items"`
	IRBaseSubtractDiscount        bool   `json:"ir_base_subtract_discount"`
	IRBaseAddFreight              bool   `json:"ir_base_add_freight"`
	IsConsumer                    bool   `json:"is_consumer"`
	IsActive                      bool   `json:"is_active"`
}

// ─── Customer ─────────────────────────────────────────────────────────────────

type CustomerResponse struct {
	ID                    int64                     `json:"id"`
	Code                  int64                     `json:"code"`
	CorporateCode         *int64                    `json:"corporate_code,omitempty"`
	IsCorporate           bool                      `json:"is_corporate"`
	Name                  string                    `json:"name"`
	TradeName             *string                   `json:"trade_name,omitempty"`
	DocumentType          string                    `json:"document_type"`
	DocumentNumber        string                    `json:"document_number"`
	StateRegistration     *string                   `json:"state_registration,omitempty"`
	MunicipalRegistration *string                   `json:"municipal_registration,omitempty"`
	SuframaCode           *string                   `json:"suframa_code,omitempty"`
	SuframaExpiry         *time.Time                `json:"suframa_expiry,omitempty"`
	RegionID              *int64                    `json:"region_id,omitempty"`
	RegionCode            *int64                    `json:"region_code,omitempty"`
	MarketSegmentID       *int64                    `json:"market_segment_id,omitempty"`
	MarketSegmentCode     *int64                    `json:"market_segment_code,omitempty"`
	CustomerTypeID        *int64                    `json:"customer_type_id,omitempty"`
	CustomerTypeCode      *int64                    `json:"customer_type_code,omitempty"`
	CarrierID             *int64                    `json:"carrier_id,omitempty"`
	CarrierCode           *int64                    `json:"carrier_code,omitempty"`
	TaxTypeID             *int64                    `json:"tax_type_id,omitempty"`
	TaxTypeCode           *int64                    `json:"tax_type_code,omitempty"`
	FinalidadeCompra      *string                   `json:"finalidade_compra,omitempty"`
	CreditLimit           float64                   `json:"credit_limit"`
	Website               *string                   `json:"website,omitempty"`
	IEIndicator           *string                   `json:"ie_indicator,omitempty"`
	TaxRegime             *string                   `json:"tax_regime,omitempty"`
	MainCNAE              *string                   `json:"main_cnae,omitempty"`
	NFeEmail              *string                   `json:"nfe_email,omitempty"`
	BillingEmail          *string                   `json:"billing_email,omitempty"`
	RepresentativeCode    *int64                    `json:"representative_code,omitempty"`
	ShippingCarrierID     *int64                    `json:"shipping_carrier_id,omitempty"`
	FreightType           *string                   `json:"freight_type,omitempty"`
	RequiresCustomerPO    bool                      `json:"requires_customer_po"`
	RequiresQualityCert   bool                      `json:"requires_quality_cert"`
	AllowsPartialBilling  bool                      `json:"allows_partial_billing"`
	ReceivingNotes        *string                   `json:"receiving_notes,omitempty"`
	CommercialNotes       *string                   `json:"commercial_notes,omitempty"`
	FinancialNotes        *string                   `json:"financial_notes,omitempty"`
	RegistrationStatus    *string                   `json:"registration_status,omitempty"`
	RegistrationCheckedAt *time.Time                `json:"registration_checked_at,omitempty"`
	OpeningDate           *time.Time                `json:"opening_date,omitempty"`
	ShareCapital          *float64                  `json:"share_capital,omitempty"`
	CompanySize           *string                   `json:"company_size,omitempty"`
	LegalNature           *string                   `json:"legal_nature,omitempty"`
	CreditScore           *int32                    `json:"credit_score,omitempty"`
	CreditRiskClass       *string                   `json:"credit_risk_class,omitempty"`
	CreditLimitSuggested  *float64                  `json:"credit_limit_suggested,omitempty"`
	CreditLimitExpiresAt  *time.Time                `json:"credit_limit_expires_at,omitempty"`
	CreditLimitExpired    bool                      `json:"credit_limit_expired"`
	CreditReviewedAt      *time.Time                `json:"credit_reviewed_at,omitempty"`
	IsActive              bool                      `json:"is_active"`
	Blocked               bool                      `json:"blocked"`
	BlockReason           *string                   `json:"block_reason,omitempty"`
	CreatedAt             time.Time                 `json:"created_at"`
	UpdatedAt             time.Time                 `json:"updated_at"`
	Addresses             []CustomerAddressResponse `json:"addresses,omitempty"`
	Contacts              []CustomerContactResponse `json:"contacts,omitempty"`
}

type CustomerAddressResponse struct {
	ID           int64   `json:"id"`
	AddressType  string  `json:"address_type"`
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

type CustomerContactResponse struct {
	ID              int64   `json:"id"`
	ContactTypeID   *int64  `json:"contact_type_id,omitempty"`
	ContactTypeCode *int64  `json:"contact_type_code,omitempty"`
	Name            string  `json:"name"`
	Email           *string `json:"email,omitempty"`
	Phone           *string `json:"phone,omitempty"`
	Mobile          *string `json:"mobile,omitempty"`
	Position        *string `json:"position,omitempty"`
	IsPrimary       bool    `json:"is_primary"`
}

// CreditSuggestionResponse é a análise de crédito explicada.
type CreditSuggestionResponse struct {
	CustomerCode    *int64                 `json:"customer_code,omitempty"`
	Score           int                    `json:"score"`
	RiskClass       string                 `json:"risk_class"`
	SuggestedLimit  float64                `json:"suggested_limit"`
	CurrentLimit    *float64               `json:"current_limit,omitempty"`
	ExpiresAt       time.Time              `json:"expires_at"`
	HasHistory      bool                   `json:"has_history"`
	HardStop        bool                   `json:"hard_stop"`
	Deteriorating   bool                   `json:"deteriorating"`
	GroupMembers    []int64                `json:"group_members"`
	RegistrationOK  bool                   `json:"registration_checked"`
	MaxWithoutAdmin float64                `json:"max_without_admin"`
	Factors         []CreditFactorResponse `json:"factors"`
}

type CreditFactorResponse struct {
	Code      string `json:"code"`
	Label     string `json:"label"`
	Points    int    `json:"points"`
	MaxPoints int    `json:"max_points"`
	Detail    string `json:"detail"`
	Blocking  bool   `json:"blocking,omitempty"`
}

type CreditReviewResponse struct {
	ID             int64                  `json:"id"`
	Origin         string                 `json:"origin"`
	SuggestedLimit float64                `json:"suggested_limit"`
	ApprovedLimit  float64                `json:"approved_limit"`
	PreviousLimit  *float64               `json:"previous_limit,omitempty"`
	Score          int                    `json:"score"`
	RiskClass      string                 `json:"risk_class"`
	ExpiresAt      time.Time              `json:"expires_at"`
	Deviates       bool                   `json:"deviates"`
	Reason         *string                `json:"reason,omitempty"`
	Factors        []CreditFactorResponse `json:"factors"`
	CreatedBy      string                 `json:"created_by"`
	CreatedByName  string                 `json:"created_by_name"`
	CreatedAt      time.Time              `json:"created_at"`
}

type CreditPolicyResponse struct {
	MaxLimit              float64 `json:"max_limit"`
	NewCustomerCapitalPct float64 `json:"new_customer_capital_pct"`
	NewCustomerCap        float64 `json:"new_customer_cap"`
	NewCustomerRevenuePct float64 `json:"new_customer_revenue_pct"`
	OverrideTolerancePct  float64 `json:"override_tolerance_pct"`
	OverdueBlockDays      int     `json:"overdue_block_days"`
	HistoryMonths         int     `json:"history_months"`
	ValidityMonthsA       int     `json:"validity_months_a"`
	ValidityMonthsB       int     `json:"validity_months_b"`
	ValidityMonthsC       int     `json:"validity_months_c"`
	ValidityMonthsD       int     `json:"validity_months_d"`
	ValidityMonthsE       int     `json:"validity_months_e"`
	RegistrationCheckDays int     `json:"registration_check_days"`
}

type CustomerUsageResponse struct {
	Code  string `json:"code"`
	Label string `json:"label"`
	Total int64  `json:"total"`
	Open  int64  `json:"open"`
}

// CustomerUsageReportResponse diz o que se pode fazer com o cadastro e por quê.
type CustomerUsageReportResponse struct {
	CanDelete        bool                    `json:"can_delete"`
	CanInactivate    bool                    `json:"can_inactivate"`
	DeleteReason     string                  `json:"delete_reason,omitempty"`
	InactivateReason string                  `json:"inactivate_reason,omitempty"`
	Usages           []CustomerUsageResponse `json:"usages"`
}

type MonthValueResponse struct {
	Month string  `json:"month"`
	Value float64 `json:"value"`
}

type CustomerOverviewResponse struct {
	OpenOrders         int64                `json:"open_orders"`
	OpenOrdersValue    float64              `json:"open_orders_value"`
	ProductionOrders   int64                `json:"production_orders_in_progress"`
	OpenQuotations     int64                `json:"open_quotations"`
	OpenServiceOrders  int64                `json:"open_service_orders"`
	OpenReceivables    float64              `json:"open_receivables"`
	OverdueReceivables float64              `json:"overdue_receivables"`
	MaxOpenOverdueDays int                  `json:"max_open_overdue_days"`
	Billed12m          float64              `json:"billed_12m"`
	LastOrderDate      *time.Time           `json:"last_order_date,omitempty"`
	LastInvoiceDate    *time.Time           `json:"last_invoice_date,omitempty"`
	AvgDelayDays       float64              `json:"avg_delay_days"`
	MaxDelayDays       int                  `json:"max_delay_days"`
	OnTimePct          *float64             `json:"on_time_pct,omitempty"`
	GroupMembers       int                  `json:"group_members"`
	GroupExposure      float64              `json:"group_exposure"`
	ConcentrationPct   float64              `json:"concentration_pct"`
	CreditLimit        float64              `json:"credit_limit"`
	CreditUsed         float64              `json:"credit_used"`
	CreditAvailable    float64              `json:"credit_available"`
	BillingByMonth     []MonthValueResponse `json:"billing_by_month"`
	Alerts             []string             `json:"alerts"`
}
