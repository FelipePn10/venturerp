package response

import "time"

// ─── Legal Device ─────────────────────────────────────────────────────────────

type LegalDeviceResponse struct {
	ID          int64     `json:"id"`
	Code        int64     `json:"code"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// ─── CFOP ─────────────────────────────────────────────────────────────────────

type CFOPResponse struct {
	ID              int64     `json:"id"`
	Code            int32     `json:"code"`
	Description     string    `json:"description"`
	DescriptionFull *string   `json:"description_full,omitempty"`
	Utilization     string    `json:"utilization"`
	OrigemClasIPI   *string   `json:"origem_clas_ipi,omitempty"`
	IndOperacao     string    `json:"ind_operacao"`
	TipoUtilizacao  string    `json:"tipo_utilizacao"`
	CodigoAnexoSN   *string   `json:"codigo_anexo_sn,omitempty"`
	DIFAL           bool      `json:"difal"`
	Doacao          bool      `json:"doacao"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
}

// ─── ICMS/IPI Tax Param ────────────────────────────────────────────────────────

type TaxParamResponse struct {
	ID                              int64    `json:"id"`
	NCMCode                         *string  `json:"ncm_code,omitempty"`
	ItemCode                        *int64   `json:"item_code,omitempty"`
	ItemConfigMask                  *string  `json:"item_config_mask,omitempty"`
	UF                              string   `json:"uf"`
	OperationType                   string   `json:"operation_type"`
	CustomerCode                    *int64   `json:"customer_code,omitempty"`
	CustomerEstablishmentCode       *int64   `json:"customer_establishment_code,omitempty"`
	MarketSegmentID                 *int64   `json:"market_segment_id,omitempty"`
	InvoiceTypeExitID               *int64   `json:"invoice_type_exit_id,omitempty"`
	InvoiceTypeEntryID              *int64   `json:"invoice_type_entry_id,omitempty"`
	TaxTypeID                       *int64   `json:"tax_type_id,omitempty"`
	IsPreferred                     bool     `json:"is_preferred"`
	IsSimpleOptante                 bool     `json:"is_simples_optante"`
	IsActive                        bool     `json:"is_active"`
	ICMSPctContrib                  float64  `json:"icms_pct_contrib"`
	LegalDeviceICMSContribID        *int64   `json:"legal_device_icms_contrib_id,omitempty"`
	ICMSPctNonContrib               float64  `json:"icms_pct_non_contrib"`
	LegalDeviceICMSNonContribID     *int64   `json:"legal_device_icms_non_contrib_id,omitempty"`
	ICMSRedPctContrib               float64  `json:"icms_red_pct_contrib"`
	ICMSRedTargetContrib            *string  `json:"icms_red_target_contrib,omitempty"`
	LegalDeviceICMSRedContribID     *int64   `json:"legal_device_icms_red_contrib_id,omitempty"`
	ICMSRedPctNonContrib            float64  `json:"icms_red_pct_non_contrib"`
	ICMSRedTargetNonContrib         *string  `json:"icms_red_target_non_contrib,omitempty"`
	LegalDeviceICMSRedNonContribID  *int64   `json:"legal_device_icms_red_non_contrib_id,omitempty"`
	ICMSDeferralPct                 float64  `json:"icms_deferral_pct"`
	ICMSDeferralTarget              *string  `json:"icms_deferral_target,omitempty"`
	LegalDeviceICMSDeferralID       *int64   `json:"legal_device_icms_deferral_id,omitempty"`
	CodBenefRBC                     *string  `json:"cod_benef_rbc,omitempty"`
	ICMSSubstPctContrib             float64  `json:"icms_subst_pct_contrib"`
	LegalDeviceICMSSubstContribID   *int64   `json:"legal_device_icms_subst_contrib_id,omitempty"`
	ICMSSubstPctNonContrib          float64  `json:"icms_subst_pct_non_contrib"`
	LegalDeviceICMSSubstNonContribID *int64  `json:"legal_device_icms_subst_non_contrib_id,omitempty"`
	ICMSSubstPctContribUC           float64  `json:"icms_subst_pct_contrib_uc"`
	ICMSSubstRedPct                 float64  `json:"icms_subst_red_pct"`
	LegalDeviceICMSSubstRedID       *int64   `json:"legal_device_icms_subst_red_id,omitempty"`
	ICMSInternalPct                 float64  `json:"icms_internal_pct"`
	BCICMSSTModality                *string  `json:"bc_icms_st_modality,omitempty"`
	ICMSPctForSTContrib             float64  `json:"icms_pct_for_st_contrib"`
	ICMSPctForSTNonContrib          float64  `json:"icms_pct_for_st_non_contrib"`
	CSTSituationB                   *string  `json:"cst_situation_b,omitempty"`
	CSOSNIGMS                       *string  `json:"csosn_icms,omitempty"`
	CSTICMSContrib                  *string  `json:"cst_icms_contrib,omitempty"`
	CSTICMSNonContrib               *string  `json:"cst_icms_non_contrib,omitempty"`
	CodBeneficioFiscal              *string  `json:"cod_beneficio_fiscal,omitempty"`
	CSTICMSContribDev               *string  `json:"cst_icms_contrib_dev,omitempty"`
	CSTICMSNonContribDev            *string  `json:"cst_icms_non_contrib_dev,omitempty"`
	IPIRedPctContrib                float64  `json:"ipi_red_pct_contrib"`
	IPIRedTargetContrib             *string  `json:"ipi_red_target_contrib,omitempty"`
	LegalDeviceIPIContribID         *int64   `json:"legal_device_ipi_contrib_id,omitempty"`
	IPIRedPctNonContrib             float64  `json:"ipi_red_pct_non_contrib"`
	IPIRedTargetNonContrib          *string  `json:"ipi_red_target_non_contrib,omitempty"`
	LegalDeviceIPINonContribID      *int64   `json:"legal_device_ipi_non_contrib_id,omitempty"`
	CSTIPIExit                      *string  `json:"cst_ipi_exit,omitempty"`
	CSTIPIEntry                     *string  `json:"cst_ipi_entry,omitempty"`
	ICMSPctOrigins1238              float64  `json:"icms_pct_origins_1238"`
	CalcBaseRedFCI                  bool     `json:"calc_base_red_fci"`
	ICMSSubstPctOrigins1238         float64  `json:"icms_subst_pct_origins_1238"`
	CSTICMSFci                      *string  `json:"cst_icms_fci,omitempty"`
	UsesICMSZonaFranca              bool     `json:"uses_icms_zona_franca"`
	DifAliqSTContribUC              float64  `json:"dif_aliq_st_contrib_uc"`
	CodBenefContrib                 *string  `json:"cod_benef_contrib,omitempty"`
	CodBenefNonContrib              *string  `json:"cod_benef_non_contrib,omitempty"`
	ICMSAcresPctContrib             float64  `json:"icms_acres_pct_contrib"`
	ICMSAcresTypeContrib            *string  `json:"icms_acres_type_contrib,omitempty"`
	ICMSAcresSumContrib             bool     `json:"icms_acres_sum_contrib"`
	ICMSAcresPctNonContrib          float64  `json:"icms_acres_pct_non_contrib"`
	ICMSAcresTypeNonContrib         *string  `json:"icms_acres_type_non_contrib,omitempty"`
	ICMSAcresSumNonContrib          bool     `json:"icms_acres_sum_non_contrib"`
	ICMSSTAcresPctContrib           float64  `json:"icms_st_acres_pct_contrib"`
	ICMSSTAcresTypeContrib          *string  `json:"icms_st_acres_type_contrib,omitempty"`
	ICMSSTAcresSumContrib           bool     `json:"icms_st_acres_sum_contrib"`
	ICMSSTAcresPctNonContrib        float64  `json:"icms_st_acres_pct_non_contrib"`
	ICMSSTAcresTypeNonContrib       *string  `json:"icms_st_acres_type_non_contrib,omitempty"`
	ICMSSTAcresSumNonContrib        bool     `json:"icms_st_acres_sum_non_contrib"`
	FCPSTPartilhaPct                float64  `json:"fcp_st_partilha_pct"`
	ICMSDifalRedPct                 float64  `json:"icms_difal_red_pct"`
	ICMSDifalType                   *string  `json:"icms_difal_type,omitempty"`
	DifalPurchaseRedPct             float64  `json:"difal_purchase_red_pct"`
	DifalPurchaseRedTarget          *string  `json:"difal_purchase_red_target,omitempty"`
	CreatedAt                       time.Time `json:"created_at"`
}

// ─── Item Classification Mask ─────────────────────────────────────────────────

type ItemClassificationMaskResponse struct {
	ID          int64     `json:"id"`
	Code        int64     `json:"code"`
	Mask        string    `json:"mask"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// ─── Item Classification ──────────────────────────────────────────────────────

type ItemClassificationResponse struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	MaskID      int64     `json:"mask_id"`
	ParentID    *int64    `json:"parent_id,omitempty"`
	Level       int       `json:"level"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// ─── Country ──────────────────────────────────────────────────────────────────

type CountryResponse struct {
	ID        int64     `json:"id"`
	Sigla     string    `json:"sigla"`
	Name      string    `json:"name"`
	DDI       *string   `json:"ddi,omitempty"`
	BacenCode *string   `json:"bacen_code,omitempty"`
	SisComex  *string   `json:"sis_comex,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── UF ───────────────────────────────────────────────────────────────────────

type UFResponse struct {
	ID        int64     `json:"id"`
	Sigla     string    `json:"sigla"`
	Name      string    `json:"name"`
	CountryID int64     `json:"country_id"`
	IBGECode  *string   `json:"ibge_code,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── DAPI Transfer Reason ─────────────────────────────────────────────────────

type DAPITransferReasonResponse struct {
	ID          int64      `json:"id"`
	Code        string     `json:"code"`
	Reason      string     `json:"reason"`
	Destination *string    `json:"destination,omitempty"`
	ValidFrom   *time.Time `json:"valid_from,omitempty"`
	ValidTo     *time.Time `json:"valid_to,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ─── ICMS Apuração Adjustment Code ───────────────────────────────────────────

type ICMSApuracaoAdjCodeResponse struct {
	ID          int64      `json:"id"`
	Code        string     `json:"code"`
	UF          string     `json:"uf"`
	Description string     `json:"description"`
	ValidFrom   *time.Time `json:"valid_from,omitempty"`
	ValidTo     *time.Time `json:"valid_to,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ─── ICMS Adjustment Code ─────────────────────────────────────────────────────

type ICMSAdjustmentCodeResponse struct {
	ID          int64      `json:"id"`
	UF          string     `json:"uf"`
	Code        string     `json:"code"`
	Description string     `json:"description"`
	TableRef    string     `json:"table_ref"`
	ValidFrom   *time.Time `json:"valid_from,omitempty"`
	ValidTo     *time.Time `json:"valid_to,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ─── ICMS Apuração Line ───────────────────────────────────────────────────────

type ICMSApuracaoLineResponse struct {
	ID                    int64   `json:"id"`
	Code                  string  `json:"code"`
	Description           string  `json:"description"`
	LineType              string  `json:"line_type"`
	AcceptsEntries        bool    `json:"accepts_entries"`
	Nature                *string `json:"nature,omitempty"`
	ApuracaoAdjCodeID     *int64  `json:"apuracao_adj_code_id,omitempty"`
	IsActive              bool    `json:"is_active"`
	CreatedAt             time.Time `json:"created_at"`
}

// ─── ICMS Summary Entry ───────────────────────────────────────────────────────

type ICMSSummaryEntryResponse struct {
	ID             int64     `json:"id"`
	Period         string    `json:"period"`
	UF             string    `json:"uf"`
	CFOPID         *int64    `json:"cfop_id,omitempty"`
	ICMSBase       float64   `json:"icms_base"`
	ICMSValue      float64   `json:"icms_value"`
	ICMSBaseOther  float64   `json:"icms_base_other"`
	ICMSValueOther float64   `json:"icms_value_other"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
}

type ICMSSummaryEntryNoteResponse struct {
	ID             int64     `json:"id"`
	SummaryEntryID int64     `json:"summary_entry_id"`
	NoteNumber     string    `json:"note_number"`
	NoteSeries     *string   `json:"note_series,omitempty"`
	EmitterCNPJ    *string   `json:"emitter_cnpj,omitempty"`
	IssueDate      time.Time `json:"issue_date"`
	ItemValue      float64   `json:"item_value"`
	ICMSBase       float64   `json:"icms_base"`
	ICMSValue      float64   `json:"icms_value"`
	Observation    *string   `json:"observation,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// ─── Simples Nacional Apuração ────────────────────────────────────────────────

type SimplesNacionalApuracaoResponse struct {
	ID                  int64     `json:"id"`
	Period              string    `json:"period"`
	Annex               string    `json:"annex"`
	ReceitaInterna      float64   `json:"receita_interna"`
	ReceitaExterna      float64   `json:"receita_externa"`
	FolhaPagamento      float64   `json:"folha_pagamento"`
	ReceitaBruta12M     float64   `json:"receita_bruta_12m"`
	SimplesRecolhido    float64   `json:"simples_recolhido"`
	AliquotaNominal     float64   `json:"aliquota_nominal"`
	AliquotaEfetiva     float64   `json:"aliquota_efetiva"`
	AliquotaEfetivaICMS float64   `json:"aliquota_efetiva_icms"`
	ParcelaDeduzir      float64   `json:"parcela_deduzir"`
	Observation         *string   `json:"observation,omitempty"`
	IsActive            bool      `json:"is_active"`
	CreatedAt           time.Time `json:"created_at"`
}

// ─── Stock Movement Type ──────────────────────────────────────────────────────

type StockMovementTypeResponse struct {
	ID                   int64     `json:"id"`
	Sigla                string    `json:"sigla"`
	Description          string    `json:"description"`
	UsageType            string    `json:"usage_type"`
	EntryOrder           bool      `json:"entry_order"`
	ExitOrder            bool      `json:"exit_order"`
	ConsidersConsumption bool      `json:"considers_consumption"`
	UpdatesAvgCost       bool      `json:"updates_avg_cost"`
	IsAdjustment         bool      `json:"is_adjustment"`
	UpdatesCycleCount    bool      `json:"updates_cycle_count"`
	ShowsInSummary       bool      `json:"shows_in_summary"`
	EntryExit            string    `json:"entry_exit"`
	GeneratesFCIMovement bool      `json:"generates_fci_movement"`
	IsActive             bool      `json:"is_active"`
	CreatedAt            time.Time `json:"created_at"`
}
