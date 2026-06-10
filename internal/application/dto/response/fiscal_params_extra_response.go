package response

import "time"

// ICMSReductionSubstitutionResponse is the API representation of an ICMS
// reduction/substitution/deferral parameter.
type ICMSReductionSubstitutionResponse struct {
	ID               int64   `json:"id"`
	ItemID           *int64  `json:"item_id,omitempty"`
	ItemMask         *string `json:"item_mask,omitempty"`
	NCMCode          *string `json:"ncm_code,omitempty"`
	UF               string  `json:"uf"`
	OperationType    string  `json:"operation_type"`
	CustomerID       *int64  `json:"customer_id,omitempty"`
	EstablishmentID  *int64  `json:"establishment_id,omitempty"`
	SupplierID       *int64  `json:"supplier_id,omitempty"`
	InvoiceTypeOutID *int64  `json:"invoice_type_out_id,omitempty"`
	InvoiceTypeInID  *int64  `json:"invoice_type_in_id,omitempty"`
	MarketSegmentID  *int64  `json:"market_segment_id,omitempty"`
	IsPreferential   bool    `json:"is_preferential"`

	ICMSPctContrib              *float64 `json:"icms_pct_contrib,omitempty"`
	ICMSPctNonContrib           *float64 `json:"icms_pct_non_contrib,omitempty"`
	LegalDeviceICMSContribID    *int64   `json:"legal_device_icms_contrib_id,omitempty"`
	LegalDeviceICMSNonContribID *int64   `json:"legal_device_icms_non_contrib_id,omitempty"`

	ICMSRedPctContrib              *float64 `json:"icms_red_pct_contrib,omitempty"`
	ICMSRedTargetContrib           string   `json:"icms_red_target_contrib"`
	ICMSRedPctNonContrib           *float64 `json:"icms_red_pct_non_contrib,omitempty"`
	ICMSRedTargetNonContrib        string   `json:"icms_red_target_non_contrib"`
	LegalDeviceICMSRedContribID    *int64   `json:"legal_device_icms_red_contrib_id,omitempty"`
	LegalDeviceICMSRedNonContribID *int64   `json:"legal_device_icms_red_non_contrib_id,omitempty"`

	ICMSDeferralPct           *float64 `json:"icms_deferral_pct,omitempty"`
	ICMSDeferralTarget        string   `json:"icms_deferral_target"`
	LegalDeviceICMSDeferralID *int64   `json:"legal_device_icms_deferral_id,omitempty"`
	ICMSDeferralBenefitCode   *string  `json:"icms_deferral_benefit_code,omitempty"`

	ICMSSubstPctContrib              *float64 `json:"icms_subst_pct_contrib,omitempty"`
	ICMSSubstPctNonContrib           *float64 `json:"icms_subst_pct_non_contrib,omitempty"`
	ICMSSubstPctContribUC            *float64 `json:"icms_subst_pct_contrib_uc,omitempty"`
	ICMSSubstRedPct                  *float64 `json:"icms_subst_red_pct,omitempty"`
	ICMSInternalPct                  *float64 `json:"icms_internal_pct,omitempty"`
	LegalDeviceICMSSubstContribID    *int64   `json:"legal_device_icms_subst_contrib_id,omitempty"`
	LegalDeviceICMSSubstNonContribID *int64   `json:"legal_device_icms_subst_non_contrib_id,omitempty"`
	LegalDeviceICMSSubstRedID        *int64   `json:"legal_device_icms_subst_red_id,omitempty"`
	ModBCICMSST                      *string  `json:"mod_bc_icms_st,omitempty"`
	ICMSPctForSTContrib              *float64 `json:"icms_pct_for_st_contrib,omitempty"`
	ICMSPctForSTNonContrib           *float64 `json:"icms_pct_for_st_non_contrib,omitempty"`

	CSTICMSContrib       *string `json:"cst_icms_contrib,omitempty"`
	CSTICMSNonContrib    *string `json:"cst_icms_non_contrib,omitempty"`
	CSOSNTICMS           *string `json:"csosn_icms,omitempty"`
	CSTICMSContribDev    *string `json:"cst_icms_contrib_dev,omitempty"`
	CSTICMSNonContribDev *string `json:"cst_icms_non_contrib_dev,omitempty"`
	CSTSitTribB          *string `json:"cst_sit_trib_b,omitempty"`

	FiscalBenefitCodeContrib    *string `json:"fiscal_benefit_code_contrib,omitempty"`
	FiscalBenefitCodeNonContrib *string `json:"fiscal_benefit_code_non_contrib,omitempty"`
	FiscalBenefitCode           *string `json:"fiscal_benefit_code,omitempty"`

	IPIRedPctContrib           *float64 `json:"ipi_red_pct_contrib,omitempty"`
	IPIRedTargetContrib        string   `json:"ipi_red_target_contrib"`
	IPIRedPctNonContrib        *float64 `json:"ipi_red_pct_non_contrib,omitempty"`
	IPIRedTargetNonContrib     string   `json:"ipi_red_target_non_contrib"`
	LegalDeviceIPIContribID    *int64   `json:"legal_device_ipi_contrib_id,omitempty"`
	LegalDeviceIPINonContribID *int64   `json:"legal_device_ipi_non_contrib_id,omitempty"`
	CSTIPIOut                  *string  `json:"cst_ipi_out,omitempty"`
	CSTIPIIn                   *string  `json:"cst_ipi_in,omitempty"`

	FCIICMSPct             *float64 `json:"fci_icms_pct,omitempty"`
	FCIReduceBase          bool     `json:"fci_reduce_base"`
	FCIICMSSubstPct        *float64 `json:"fci_icms_subst_pct,omitempty"`
	FCICSTICMs             *string  `json:"fci_cst_icms,omitempty"`
	FCIUseICMSZF           bool     `json:"fci_use_icms_zf"`
	FCIDIFALSTContribUCPct *float64 `json:"fci_difal_st_contrib_uc_pct,omitempty"`

	ICMSAddPctContrib        *float64 `json:"icms_add_pct_contrib,omitempty"`
	ICMSAddTypeContrib       *string  `json:"icms_add_type_contrib,omitempty"`
	ICMSAddSumAliqContrib    bool     `json:"icms_add_sum_aliq_contrib"`
	ICMSAddPctNonContrib     *float64 `json:"icms_add_pct_non_contrib,omitempty"`
	ICMSAddTypeNonContrib    *string  `json:"icms_add_type_non_contrib,omitempty"`
	ICMSAddSumAliqNonContrib bool     `json:"icms_add_sum_aliq_non_contrib"`
	ICMSSTAddPctContrib      *float64 `json:"icms_st_add_pct_contrib,omitempty"`
	ICMSSTAddTypeContrib     *string  `json:"icms_st_add_type_contrib,omitempty"`
	ICMSSTAddPctNonContrib   *float64 `json:"icms_st_add_pct_non_contrib,omitempty"`
	ICMSSTAddTypeNonContrib  *string  `json:"icms_st_add_type_non_contrib,omitempty"`
	FCPPartitionPct          *float64 `json:"fcp_partition_pct,omitempty"`

	DIFALICMSRedPct     *float64 `json:"difal_icms_red_pct,omitempty"`
	DIFALICMSType       *string  `json:"difal_icms_type,omitempty"`
	DIFALPurchaseRedPct *float64 `json:"difal_purchase_red_pct,omitempty"`

	IsSimplesOptante bool      `json:"is_simples_optante"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
}

// ICMSSummaryEntryAdditionalResponse is the API representation of a summary entry additional.
type ICMSSummaryEntryAdditionalResponse struct {
	ID                   int64     `json:"id"`
	SummaryEntryID       int64     `json:"summary_entry_id"`
	Sequence             int       `json:"sequence"`
	ArrecadacaoIndicator string    `json:"arrecadacao_indicator"`
	Processo             *string   `json:"processo,omitempty"`
	Arrecadacao          *string   `json:"arrecadacao,omitempty"`
	Description          *string   `json:"description,omitempty"`
	DIEFTable            *string   `json:"dief_table,omitempty"`
	DIEFCode             *string   `json:"dief_code,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}

// ICMSSTRestitutionResponse is the API representation of an ICMS-ST restitution.
type ICMSSTRestitutionResponse struct {
	ID                      int64      `json:"id"`
	EmpresaID               int        `json:"empresa_id"`
	Period                  string     `json:"period"`
	RestitutionType         string     `json:"restitution_type"`
	UF                      string     `json:"uf"`
	OrigDocModel            *string    `json:"orig_doc_model,omitempty"`
	OrigDocSeries           *string    `json:"orig_doc_series,omitempty"`
	OrigDocNumber           *string    `json:"orig_doc_number,omitempty"`
	OrigDocDate             *time.Time `json:"orig_doc_date,omitempty"`
	OrigEmitterCNPJ         *string    `json:"orig_emitter_cnpj,omitempty"`
	OrigEmitterIE           *string    `json:"orig_emitter_ie,omitempty"`
	ItemID                  *int64     `json:"item_id,omitempty"`
	ItemCode                *string    `json:"item_code,omitempty"`
	CFOP                    *string    `json:"cfop,omitempty"`
	MotivoCode              *string    `json:"motivo_code,omitempty"`
	CSTICMS                 *string    `json:"cst_icms,omitempty"`
	ICMSSTBase              float64    `json:"icms_st_base"`
	ICMSSTAliq              float64    `json:"icms_st_aliq"`
	ICMSSTValue             float64    `json:"icms_st_value"`
	ICMSSTBaseRestitution   float64    `json:"icms_st_base_restitution"`
	ICMSSTValueRestitution  float64    `json:"icms_st_value_restitution"`
	ICMSSTConsolidatedBase  float64    `json:"icms_st_consolidated_base"`
	ICMSSTConsolidatedValue float64    `json:"icms_st_consolidated_value"`
	H030IndEstoque          *string    `json:"h030_ind_estoque,omitempty"`
	SpedBlock               *string    `json:"sped_block,omitempty"`
	IsActive                bool       `json:"is_active"`
	CreatedAt               time.Time  `json:"created_at"`
}

// SpecialAdjustmentNoteResponse is the API representation of a special adjustment note.
type SpecialAdjustmentNoteResponse struct {
	ID                      int64                               `json:"id"`
	EmpresaID               int                                 `json:"empresa_id"`
	Purpose                 string                              `json:"purpose"`
	Status                  string                              `json:"status"`
	Number                  *string                             `json:"number,omitempty"`
	Series                  *string                             `json:"series,omitempty"`
	IssueDate               time.Time                           `json:"issue_date"`
	Period                  string                              `json:"period"`
	InvoiceTypeID           *int64                              `json:"invoice_type_id,omitempty"`
	CFOPID                  *int64                              `json:"cfop_id,omitempty"`
	ICMSApuracaoLineID      *int64                              `json:"icms_apuracao_line_id,omitempty"`
	AdjustmentCodeID        *int64                              `json:"adjustment_code_id,omitempty"`
	AdjustmentDocCodeID     *int64                              `json:"adjustment_doc_code_id,omitempty"`
	History                 *string                             `json:"history,omitempty"`
	AutoGenerateSummary     bool                                `json:"auto_generate_summary"`
	GeneratedSummaryEntryID *int64                              `json:"generated_summary_entry_id,omitempty"`
	TotalValue              float64                             `json:"total_value"`
	TotalICMS               float64                             `json:"total_icms"`
	TotalIPI                float64                             `json:"total_ipi"`
	Observation             *string                             `json:"observation,omitempty"`
	Items                   []SpecialAdjustmentNoteItemResponse `json:"items,omitempty"`
	CreatedAt               time.Time                           `json:"created_at"`
}

// SpecialAdjustmentNoteItemResponse is the API representation of a special adjustment note line.
type SpecialAdjustmentNoteItemResponse struct {
	ID                int64     `json:"id"`
	NoteID            int64     `json:"note_id"`
	Sequence          int       `json:"sequence"`
	ItemID            *int64    `json:"item_id,omitempty"`
	ItemCode          *string   `json:"item_code,omitempty"`
	Description       *string   `json:"description,omitempty"`
	Quantity          float64   `json:"quantity"`
	Unit              *string   `json:"unit,omitempty"`
	UnitValue         float64   `json:"unit_value"`
	TotalValue        float64   `json:"total_value"`
	ICMSBase          float64   `json:"icms_base"`
	ICMSPct           float64   `json:"icms_pct"`
	ICMSDeferralPct   float64   `json:"icms_deferral_pct"`
	ICMSValue         float64   `json:"icms_value"`
	ICMSDeferredValue float64   `json:"icms_deferred_value"`
	IPIBase           float64   `json:"ipi_base"`
	IPIPct            float64   `json:"ipi_pct"`
	IPIValue          float64   `json:"ipi_value"`
	CSTICMS           *string   `json:"cst_icms,omitempty"`
	CSTIPI            *string   `json:"cst_ipi,omitempty"`
	CFOPID            *int64    `json:"cfop_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}
