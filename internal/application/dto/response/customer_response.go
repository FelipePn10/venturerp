package response

import "time"

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
	MarketSegmentID       *int64                    `json:"market_segment_id,omitempty"`
	CustomerTypeID        *int64                    `json:"customer_type_id,omitempty"`
	PaymentConditionID    *int64                    `json:"payment_condition_id,omitempty"`
	SalesTableID          *int64                    `json:"sales_table_id,omitempty"`
	CarrierID             *int64                    `json:"carrier_id,omitempty"`
	CarrierGroupID        *int64                    `json:"carrier_group_id,omitempty"`
	InvoiceTypeID         *int64                    `json:"invoice_type_id,omitempty"`
	TaxTypeID             *int64                    `json:"tax_type_id,omitempty"`
	PaymentCondVisibility string                    `json:"payment_cond_visibility"`
	CreditLimit           float64                   `json:"credit_limit"`
	Website               *string                   `json:"website,omitempty"`
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
	ID            int64   `json:"id"`
	ContactTypeID *int64  `json:"contact_type_id,omitempty"`
	Name          string  `json:"name"`
	Email         *string `json:"email,omitempty"`
	Phone         *string `json:"phone,omitempty"`
	Mobile        *string `json:"mobile,omitempty"`
	Position      *string `json:"position,omitempty"`
	IsPrimary     bool    `json:"is_primary"`
}
