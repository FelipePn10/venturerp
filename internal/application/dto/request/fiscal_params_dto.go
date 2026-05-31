package request

// ─── Legal Devices ─────────────────────────────────────────────────────────────

type CreateLegalDeviceDTO struct {
	Type        string `json:"type"`        // ICMS | IPI | LAUDO | PIS | COFINS
	Description string `json:"description"`
}

type UpdateLegalDeviceDTO struct {
	ID          int64  `json:"id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

// ─── CFOP / Naturezas de Operação ──────────────────────────────────────────────

type CreateCFOPDTO struct {
	Code            int32   `json:"code"`
	Description     string  `json:"description"`
	DescriptionFull *string `json:"description_full,omitempty"`
	Utilization     string  `json:"utilization"` // INDUSTRIALIZACAO_COMERCIO | IMOBILIZADO | USO_CONSUMO
	OrigemClasIPI   *string `json:"origem_clas_ipi,omitempty"` // COMPRA | VENDA
	IndOperacao     string  `json:"ind_operacao"`   // NORMAL | ENERGIA_ELETRICA | TELECOMUNICACAO
	TipoUtilizacao  string  `json:"tipo_utilizacao"` // NORMAL | VENDA_COMERCIAL_EXPORTADORA | COMPRA_FIM_ESPECIFICO_EXPORTACAO | EXPORTACAO
	CodigoAnexoSN   *string `json:"codigo_anexo_sn,omitempty"`
	DIFAL           bool    `json:"difal"`
	Doacao          bool    `json:"doacao"`
}

type UpdateCFOPDTO struct {
	ID              int64   `json:"id"`
	Code            int32   `json:"code"`
	Description     string  `json:"description"`
	DescriptionFull *string `json:"description_full,omitempty"`
	Utilization     string  `json:"utilization"`
	OrigemClasIPI   *string `json:"origem_clas_ipi,omitempty"`
	IndOperacao     string  `json:"ind_operacao"`
	TipoUtilizacao  string  `json:"tipo_utilizacao"`
	CodigoAnexoSN   *string `json:"codigo_anexo_sn,omitempty"`
	DIFAL           bool    `json:"difal"`
	Doacao          bool    `json:"doacao"`
	IsActive        bool    `json:"is_active"`
}

// ─── ICMS/IPI Tax Params ────────────────────────────────────────────────────────

type CreateTaxParamDTO struct {
	// Search keys
	NCMCode               *string `json:"ncm_code,omitempty"`
	ItemCode              *int64  `json:"item_code,omitempty"`
	ItemConfigMask        *string `json:"item_config_mask,omitempty"`
	UF                    string  `json:"uf"`
	OperationType         string  `json:"operation_type"` // AMBAS | ENTRADA | SAIDA | CUSTOS

	// Optional FK filters
	CustomerCode              *int64 `json:"customer_code,omitempty"`
	CustomerEstablishmentCode *int64 `json:"customer_establishment_code,omitempty"`
	MarketSegmentID           *int64 `json:"market_segment_id,omitempty"`
	InvoiceTypeExitID         *int64 `json:"invoice_type_exit_id,omitempty"`
	InvoiceTypeEntryID        *int64 `json:"invoice_type_entry_id,omitempty"`
	TaxTypeID                 *int64 `json:"tax_type_id,omitempty"`

	// Flags
	IsPreferred     bool `json:"is_preferred"`
	IsSimpleOptante bool `json:"is_simples_optante"`

	// ICMS rates
	ICMSPctContrib          float64 `json:"icms_pct_contrib"`
	LegalDeviceICMSContribID *int64  `json:"legal_device_icms_contrib_id,omitempty"`
	ICMSPctNonContrib       float64 `json:"icms_pct_non_contrib"`
	LegalDeviceICMSNonContribID *int64 `json:"legal_device_icms_non_contrib_id,omitempty"`

	// ICMS reduction
	ICMSRedPctContrib           float64 `json:"icms_red_pct_contrib"`
	ICMSRedTargetContrib        *string `json:"icms_red_target_contrib,omitempty"` // BASE | PERCENTUAL
	LegalDeviceICMSRedContribID *int64  `json:"legal_device_icms_red_contrib_id,omitempty"`
	ICMSRedPctNonContrib        float64 `json:"icms_red_pct_non_contrib"`
	ICMSRedTargetNonContrib     *string `json:"icms_red_target_non_contrib,omitempty"`
	LegalDeviceICMSRedNonContribID *int64 `json:"legal_device_icms_red_non_contrib_id,omitempty"`

	// ICMS deferral
	ICMSDeferralPct           float64 `json:"icms_deferral_pct"`
	ICMSDeferralTarget        *string `json:"icms_deferral_target,omitempty"`
	LegalDeviceICMSDeferralID *int64  `json:"legal_device_icms_deferral_id,omitempty"`
	CodBenefRBC               *string `json:"cod_benef_rbc,omitempty"`

	// ICMS substitution
	ICMSSubstPctContrib          float64 `json:"icms_subst_pct_contrib"`
	LegalDeviceICMSSubstContribID *int64  `json:"legal_device_icms_subst_contrib_id,omitempty"`
	ICMSSubstPctNonContrib       float64 `json:"icms_subst_pct_non_contrib"`
	LegalDeviceICMSSubstNonContribID *int64 `json:"legal_device_icms_subst_non_contrib_id,omitempty"`
	ICMSSubstPctContribUC        float64 `json:"icms_subst_pct_contrib_uc"`
	ICMSSubstRedPct              float64 `json:"icms_subst_red_pct"`
	LegalDeviceICMSSubstRedID    *int64  `json:"legal_device_icms_subst_red_id,omitempty"`

	// ICMS ST calculation
	ICMSInternalPct        float64 `json:"icms_internal_pct"`
	BCICMSSTModality       *string `json:"bc_icms_st_modality,omitempty"`
	ICMSPctForSTContrib    float64 `json:"icms_pct_for_st_contrib"`
	ICMSPctForSTNonContrib float64 `json:"icms_pct_for_st_non_contrib"`

	// CST / CSOSN
	CSTSituationB      *string `json:"cst_situation_b,omitempty"`
	CSOSNIGMS          *string `json:"csosn_icms,omitempty"`
	CSTICMSContrib     *string `json:"cst_icms_contrib,omitempty"`
	CSTICMSNonContrib  *string `json:"cst_icms_non_contrib,omitempty"`
	CodBeneficioFiscal *string `json:"cod_beneficio_fiscal,omitempty"`
	CSTICMSContribDev  *string `json:"cst_icms_contrib_dev,omitempty"`
	CSTICMSNonContribDev *string `json:"cst_icms_non_contrib_dev,omitempty"`

	// IPI reduction
	IPIRedPctContrib          float64 `json:"ipi_red_pct_contrib"`
	IPIRedTargetContrib       *string `json:"ipi_red_target_contrib,omitempty"`
	LegalDeviceIPIContribID   *int64  `json:"legal_device_ipi_contrib_id,omitempty"`
	IPIRedPctNonContrib       float64 `json:"ipi_red_pct_non_contrib"`
	IPIRedTargetNonContrib    *string `json:"ipi_red_target_non_contrib,omitempty"`
	LegalDeviceIPINonContribID *int64  `json:"legal_device_ipi_non_contrib_id,omitempty"`
	CSTIPIExit                *string `json:"cst_ipi_exit,omitempty"`
	CSTIPIEntry               *string `json:"cst_ipi_entry,omitempty"`

	// FCI
	ICMSPctOrigins1238      float64 `json:"icms_pct_origins_1238"`
	CalcBaseRedFCI          bool    `json:"calc_base_red_fci"`
	ICMSSubstPctOrigins1238 float64 `json:"icms_subst_pct_origins_1238"`
	CSTICMSFci              *string `json:"cst_icms_fci,omitempty"`
	UsesICMSZonaFranca      bool    `json:"uses_icms_zona_franca"`
	DifAliqSTContribUC      float64 `json:"dif_aliq_st_contrib_uc"`
	CodBenefContrib         *string `json:"cod_benef_contrib,omitempty"`
	CodBenefNonContrib      *string `json:"cod_benef_non_contrib,omitempty"`

	// ICMS additions
	ICMSAcresPctContrib       float64 `json:"icms_acres_pct_contrib"`
	ICMSAcresTypeContrib      *string `json:"icms_acres_type_contrib,omitempty"` // FUNDO_COMBATE_POBREZA | OUTROS
	ICMSAcresSumContrib       bool    `json:"icms_acres_sum_contrib"`
	ICMSAcresPctNonContrib    float64 `json:"icms_acres_pct_non_contrib"`
	ICMSAcresTypeNonContrib   *string `json:"icms_acres_type_non_contrib,omitempty"`
	ICMSAcresSumNonContrib    bool    `json:"icms_acres_sum_non_contrib"`
	ICMSSTAcresPctContrib     float64 `json:"icms_st_acres_pct_contrib"`
	ICMSSTAcresTypeContrib    *string `json:"icms_st_acres_type_contrib,omitempty"`
	ICMSSTAcresSumContrib     bool    `json:"icms_st_acres_sum_contrib"`
	ICMSSTAcresPctNonContrib  float64 `json:"icms_st_acres_pct_non_contrib"`
	ICMSSTAcresTypeNonContrib *string `json:"icms_st_acres_type_non_contrib,omitempty"`
	ICMSSTAcresSumNonContrib  bool    `json:"icms_st_acres_sum_non_contrib"`
	FCPSTPartilhaPct          float64 `json:"fcp_st_partilha_pct"`

	// DIFAL
	ICMSDifalRedPct float64 `json:"icms_difal_red_pct"`
	ICMSDifalType   *string `json:"icms_difal_type,omitempty"` // TRIBUTADO | ISENTO_OUTRAS | NAO_CONSIDERA

	// DIFAL purchases
	DifalPurchaseRedPct    float64 `json:"difal_purchase_red_pct"`
	DifalPurchaseRedTarget *string `json:"difal_purchase_red_target,omitempty"`
}

type UpdateTaxParamDTO struct {
	ID       int64 `json:"id"`
	IsActive bool  `json:"is_active"`
	CreateTaxParamDTO
}

// ─── Item Classification Masks ───────────────────────────────────────────────

type CreateClassificationMaskDTO struct {
	Mask        string `json:"mask"`
	Description string `json:"description"`
}

type UpdateClassificationMaskDTO struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

// ─── Item Classifications ────────────────────────────────────────────────────

type CreateItemClassificationDTO struct {
	Code        string  `json:"code"`
	MaskCode    int64   `json:"mask_code"`
	ParentCode  *string `json:"parent_code,omitempty"`
	Description string  `json:"description"`
}

type UpdateItemClassificationDTO struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

// ─── Countries ────────────────────────────────────────────────────────────────

type CreateCountryDTO struct {
	Sigla     string  `json:"sigla"`
	Name      string  `json:"name"`
	DDI       *string `json:"ddi,omitempty"`
	BacenCode *string `json:"bacen_code,omitempty"`
	SisComex  *string `json:"sis_comex,omitempty"`
}

type UpdateCountryDTO struct {
	ID        int64   `json:"id"`
	Sigla     string  `json:"sigla"`
	Name      string  `json:"name"`
	DDI       *string `json:"ddi,omitempty"`
	BacenCode *string `json:"bacen_code,omitempty"`
	SisComex  *string `json:"sis_comex,omitempty"`
	IsActive  bool    `json:"is_active"`
}

// ─── UFs ──────────────────────────────────────────────────────────────────────

type CreateUFDTO struct {
	Sigla        string  `json:"sigla"`
	Name         string  `json:"name"`
	CountrySigla string  `json:"country_sigla"`
	IBGECode     *string `json:"ibge_code,omitempty"`
}

type UpdateUFDTO struct {
	ID       int64   `json:"id"`
	Sigla    string  `json:"sigla"`
	Name     string  `json:"name"`
	IBGECode *string `json:"ibge_code,omitempty"`
	IsActive bool    `json:"is_active"`
}
