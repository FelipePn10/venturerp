package entity

import "time"

// ─── Legal Device ─────────────────────────────────────────────────────────────

type LegalDeviceType string

const (
	LegalDeviceICMS   LegalDeviceType = "ICMS"
	LegalDeviceIPI    LegalDeviceType = "IPI"
	LegalDeviceLAUDO  LegalDeviceType = "LAUDO"
	LegalDevicePIS    LegalDeviceType = "PIS"
	LegalDeviceCOFINS LegalDeviceType = "COFINS"
)

type LegalDevice struct {
	ID          int64
	Code        int64
	Type        LegalDeviceType
	Description string
	IsActive    bool
	CreatedAt   time.Time
}

// ─── CFOP / Natureza de Operação ──────────────────────────────────────────────

type CfopUtilization string

const (
	CfopUtilizationIndustrializacaoComercio CfopUtilization = "INDUSTRIALIZACAO_COMERCIO"
	CfopUtilizationImobilizado              CfopUtilization = "IMOBILIZADO"
	CfopUtilizationUsoConsumo               CfopUtilization = "USO_CONSUMO"
)

type CfopIndOperacao string

const (
	CfopIndOperacaoNormal          CfopIndOperacao = "NORMAL"
	CfopIndOperacaoEnergiaEletrica CfopIndOperacao = "ENERGIA_ELETRICA"
	CfopIndOperacaoTelecomunicacao CfopIndOperacao = "TELECOMUNICACAO"
)

type CfopTipoUtilizacao string

const (
	CfopTipoUtilizacaoNormal                        CfopTipoUtilizacao = "NORMAL"
	CfopTipoUtilizacaoVendaComercialExportadora     CfopTipoUtilizacao = "VENDA_COMERCIAL_EXPORTADORA"
	CfopTipoUtilizacaoCompraFimEspecificoExportacao CfopTipoUtilizacao = "COMPRA_FIM_ESPECIFICO_EXPORTACAO"
	CfopTipoUtilizacaoExportacao                    CfopTipoUtilizacao = "EXPORTACAO"
)

type CFOP struct {
	ID              int64
	Code            int32
	Description     string
	DescriptionFull *string
	Utilization     CfopUtilization
	OrigemClasIPI   *string
	IndOperacao     CfopIndOperacao
	TipoUtilizacao  CfopTipoUtilizacao
	CodigoAnexoSN   *string
	DIFAL           bool
	Doacao          bool
	IsActive        bool
	CreatedAt       time.Time
}

// ─── ICMS/IPI Tax Params (Redução, Substituição, Diferimento) ─────────────────

type TaxParamOperation string

const (
	TaxParamOperationAmbas   TaxParamOperation = "AMBAS"
	TaxParamOperationEntrada TaxParamOperation = "ENTRADA"
	TaxParamOperationSaida   TaxParamOperation = "SAIDA"
	TaxParamOperationCustos  TaxParamOperation = "CUSTOS"
)

type IcmsReductionTarget string

const (
	IcmsReductionTargetBase       IcmsReductionTarget = "BASE"
	IcmsReductionTargetPercentual IcmsReductionTarget = "PERCENTUAL"
)

type IcmsDifalType string

const (
	IcmsDifalTypeTributado    IcmsDifalType = "TRIBUTADO"
	IcmsDifalTypeIsentoOutras IcmsDifalType = "ISENTO_OUTRAS"
	IcmsDifalTypeNaoConsidera IcmsDifalType = "NAO_CONSIDERA"
)

type IcmsAcresType string

const (
	IcmsAcresTypeFundoCombatePobreza IcmsAcresType = "FUNDO_COMBATE_POBREZA"
	IcmsAcresTypeOutros              IcmsAcresType = "OUTROS"
)

type ICMSIPITaxParam struct {
	ID int64

	// ── Search keys ─────────────────────────────────────────────────────────
	NCMCode        *string
	ItemCode       *int64
	ItemConfigMask *string
	UF             string
	OperationType  TaxParamOperation

	// ── Optional FK filters ──────────────────────────────────────────────────
	CustomerCode              *int64
	CustomerEstablishmentCode *int64
	MarketSegmentID           *int64
	InvoiceTypeExitID         *int64
	InvoiceTypeEntryID        *int64
	TaxTypeID                 *int64

	// ── Flags ────────────────────────────────────────────────────────────────
	IsPreferred     bool
	IsSimpleOptante bool

	// ── ICMS rates ───────────────────────────────────────────────────────────
	ICMSPctContrib              float64
	LegalDeviceICMSContribID    *int64
	ICMSPctNonContrib           float64
	LegalDeviceICMSNonContribID *int64

	// ── ICMS reduction ───────────────────────────────────────────────────────
	ICMSRedPctContrib              float64
	ICMSRedTargetContrib           *IcmsReductionTarget
	LegalDeviceICMSRedContribID    *int64
	ICMSRedPctNonContrib           float64
	ICMSRedTargetNonContrib        *IcmsReductionTarget
	LegalDeviceICMSRedNonContribID *int64

	// ── ICMS deferral ────────────────────────────────────────────────────────
	ICMSDeferralPct           float64
	ICMSDeferralTarget        *IcmsReductionTarget
	LegalDeviceICMSDeferralID *int64
	CodBenefRBC               *string

	// ── ICMS substitution ────────────────────────────────────────────────────
	ICMSSubstPctContrib              float64
	LegalDeviceICMSSubstContribID    *int64
	ICMSSubstPctNonContrib           float64
	LegalDeviceICMSSubstNonContribID *int64
	ICMSSubstPctContribUC            float64
	ICMSSubstRedPct                  float64
	LegalDeviceICMSSubstRedID        *int64

	// ── ICMS ST calculation ──────────────────────────────────────────────────
	ICMSInternalPct        float64
	BCICMSSTModality       *string
	ICMSPctForSTContrib    float64
	ICMSPctForSTNonContrib float64

	// ── CST / CSOSN ──────────────────────────────────────────────────────────
	CSTSituationB        *string
	CSOSNIGMS            *string
	CSTICMSContrib       *string
	CSTICMSNonContrib    *string
	CodBeneficioFiscal   *string
	CSTICMSContribDev    *string
	CSTICMSNonContribDev *string

	// ── IPI reduction ────────────────────────────────────────────────────────
	IPIRedPctContrib           float64
	IPIRedTargetContrib        *IcmsReductionTarget
	LegalDeviceIPIContribID    *int64
	IPIRedPctNonContrib        float64
	IPIRedTargetNonContrib     *IcmsReductionTarget
	LegalDeviceIPINonContribID *int64
	CSTIPIExit                 *string
	CSTIPIEntry                *string

	// ── FCI ──────────────────────────────────────────────────────────────────
	ICMSPctOrigins1238      float64
	CalcBaseRedFCI          bool
	ICMSSubstPctOrigins1238 float64
	CSTICMSFci              *string
	UsesICMSZonaFranca      bool
	DifAliqSTContribUC      float64
	CodBenefContrib         *string
	CodBenefNonContrib      *string

	// ── ICMS additions ───────────────────────────────────────────────────────
	ICMSAcresPctContrib       float64
	ICMSAcresTypeContrib      *IcmsAcresType
	ICMSAcresSumContrib       bool
	ICMSAcresPctNonContrib    float64
	ICMSAcresTypeNonContrib   *IcmsAcresType
	ICMSAcresSumNonContrib    bool
	ICMSSTAcresPctContrib     float64
	ICMSSTAcresTypeContrib    *IcmsAcresType
	ICMSSTAcresSumContrib     bool
	ICMSSTAcresPctNonContrib  float64
	ICMSSTAcresTypeNonContrib *IcmsAcresType
	ICMSSTAcresSumNonContrib  bool
	FCPSTPartilhaPct          float64

	// ── DIFAL (EC/87) ────────────────────────────────────────────────────────
	ICMSDifalRedPct float64
	ICMSDifalType   *IcmsDifalType

	// ── DIFAL purchases ──────────────────────────────────────────────────────
	DifalPurchaseRedPct    float64
	DifalPurchaseRedTarget *IcmsReductionTarget

	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ─── DAPI Transfer Reason ─────────────────────────────────────────────────────

type DAPITransferReason struct {
	ID          int64      `json:"id"`
	Code        string     `json:"code"`
	Reason      string     `json:"reason"`
	Destination *string    `json:"destination"`
	ValidFrom   *time.Time `json:"valid_from"`
	ValidTo     *time.Time `json:"valid_to"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ─── ICMS Apuração Adjustment Code (SPED tabela 5.1.1) ───────────────────────

type ICMSApuracaoAdjustmentCode struct {
	ID          int64      `json:"id"`
	Code        string     `json:"code"`
	UF          string     `json:"uf"`
	Description string     `json:"description"`
	ValidFrom   *time.Time `json:"valid_from"`
	ValidTo     *time.Time `json:"valid_to"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ─── ICMS Adjustment Code (SPED tabelas 5.2/5.3/5.6/5.7) ────────────────────

type ICMSAdjustmentTableRef string

const (
	TableRef52 ICMSAdjustmentTableRef = "5.2"
	TableRef53 ICMSAdjustmentTableRef = "5.3"
	TableRef56 ICMSAdjustmentTableRef = "5.6"
	TableRef57 ICMSAdjustmentTableRef = "5.7"
)

type ICMSAdjustmentCode struct {
	ID          int64                  `json:"id"`
	UF          string                 `json:"uf"`
	Code        string                 `json:"code"`
	Description string                 `json:"description"`
	TableRef    ICMSAdjustmentTableRef `json:"table_ref"`
	ValidFrom   *time.Time             `json:"valid_from"`
	ValidTo     *time.Time             `json:"valid_to"`
	IsActive    bool                   `json:"is_active"`
	CreatedAt   time.Time              `json:"created_at"`
}

// ─── ICMS Apuração Line ───────────────────────────────────────────────────────

type ApuracaoLineType string

const (
	LineTypeDebito  ApuracaoLineType = "DEBITO"
	LineTypeCredito ApuracaoLineType = "CREDITO"
	LineTypeSaldo   ApuracaoLineType = "SALDO"
	LineTypeDeducao ApuracaoLineType = "DEDUCAO"
	LineTypeOutros  ApuracaoLineType = "OUTROS"
)

type ICMSApuracaoLine struct {
	ID                int64            `json:"id"`
	Code              string           `json:"code"`
	Description       string           `json:"description"`
	LineType          ApuracaoLineType `json:"line_type"`
	AcceptsEntries    bool             `json:"accepts_entries"`
	Nature            *string          `json:"nature"`
	ApuracaoAdjCodeID *int64           `json:"apuracao_adj_code_id"`
	IsActive          bool             `json:"is_active"`
	CreatedAt         time.Time        `json:"created_at"`
}

// ─── ICMS Summary Entry ───────────────────────────────────────────────────────

type ICMSSummaryEntry struct {
	ID             int64     `json:"id"`
	Period         string    `json:"period"`
	UF             string    `json:"uf"`
	CFOPID         *int64    `json:"cfop_id"`
	ICMSBase       float64   `json:"icms_base"`
	ICMSValue      float64   `json:"icms_value"`
	ICMSBaseOther  float64   `json:"icms_base_other"`
	ICMSValueOther float64   `json:"icms_value_other"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
}

type ICMSSummaryEntryNote struct {
	ID             int64     `json:"id"`
	SummaryEntryID int64     `json:"summary_entry_id"`
	NoteNumber     string    `json:"note_number"`
	NoteSeries     *string   `json:"note_series"`
	EmitterCNPJ    *string   `json:"emitter_cnpj"`
	IssueDate      time.Time `json:"issue_date"`
	ItemValue      float64   `json:"item_value"`
	ICMSBase       float64   `json:"icms_base"`
	ICMSValue      float64   `json:"icms_value"`
	Observation    *string   `json:"observation"`
	CreatedAt      time.Time `json:"created_at"`
}

// ─── Simples Nacional Apuração ────────────────────────────────────────────────

type SimplesNacionalAnnex string

const (
	SimplesAnexoI   SimplesNacionalAnnex = "I"
	SimplesAnexoII  SimplesNacionalAnnex = "II"
	SimplesAnexoIII SimplesNacionalAnnex = "III"
	SimplesAnexoIV  SimplesNacionalAnnex = "IV"
	SimplesAnexoV   SimplesNacionalAnnex = "V"
	SimplesAnexoVI  SimplesNacionalAnnex = "VI"
)

// IsValidSimplesNacionalAnnex diz se o texto é um dos anexos previstos em lei.
// O banco usa um enum, então um valor fora da lista precisa ser recusado antes
// da consulta — senão o Postgres devolve um erro cru que não ajuda o usuário.
func IsValidSimplesNacionalAnnex(value string) bool {
	switch SimplesNacionalAnnex(value) {
	case SimplesAnexoI, SimplesAnexoII, SimplesAnexoIII, SimplesAnexoIV, SimplesAnexoV, SimplesAnexoVI:
		return true
	default:
		return false
	}
}

type SimplesNacionalApuracao struct {
	ID                  int64                `json:"id"`
	Period              string               `json:"period"`
	Annex               SimplesNacionalAnnex `json:"annex"`
	ReceitaInterna      float64              `json:"receita_interna"`
	ReceitaExterna      float64              `json:"receita_externa"`
	FolhaPagamento      float64              `json:"folha_pagamento"`
	ReceitaBruta12M     float64              `json:"receita_bruta_12m"`
	SimplesRecolhido    float64              `json:"simples_recolhido"`
	AliquotaNominal     float64              `json:"aliquota_nominal"`
	AliquotaEfetiva     float64              `json:"aliquota_efetiva"`
	AliquotaEfetivaICMS float64              `json:"aliquota_efetiva_icms"`
	ParcelaDeduzir      float64              `json:"parcela_deduzir"`
	Observation         *string              `json:"observation"`
	IsActive            bool                 `json:"is_active"`
	CreatedAt           time.Time            `json:"created_at"`
}

// ─── ICMS Reduction/Substitution/Deferral ────────────────────────────────────

type ICMSOperationType string

const (
	ICMSOpEntrada ICMSOperationType = "ENTRADA"
	ICMSOpSaida   ICMSOperationType = "SAIDA"
	ICMSOpAmbas   ICMSOperationType = "AMBAS"
	ICMSOpCustos  ICMSOperationType = "CUSTOS"
)

type ICMSReductionTarget string

const (
	ICMSRedBase       ICMSReductionTarget = "BASE"
	ICMSRedPercentual ICMSReductionTarget = "PERCENTUAL"
)

type ICMSReductionSubstitution struct {
	ID               int64             `json:"id"`
	ItemID           *int64            `json:"item_id"`
	ItemMask         *string           `json:"item_mask"`
	NCMCode          *string           `json:"ncm_code"`
	UF               string            `json:"uf"`
	OperationType    ICMSOperationType `json:"operation_type"`
	CustomerID       *int64            `json:"customer_id"`
	EstablishmentID  *int64            `json:"establishment_id"`
	SupplierID       *int64            `json:"supplier_id"`
	InvoiceTypeOutID *int64            `json:"invoice_type_out_id"`
	InvoiceTypeInID  *int64            `json:"invoice_type_in_id"`
	MarketSegmentID  *int64            `json:"market_segment_id"`
	IsPreferential   bool              `json:"is_preferential"`
	// ICMS alíquotas
	ICMSPctContrib              *float64 `json:"icms_pct_contrib"`
	ICMSPctNonContrib           *float64 `json:"icms_pct_non_contrib"`
	LegalDeviceICMSContribID    *int64   `json:"legal_device_icms_contrib_id"`
	LegalDeviceICMSNonContribID *int64   `json:"legal_device_icms_non_contrib_id"`
	// Redução ICMS
	ICMSRedPctContrib              *float64            `json:"icms_red_pct_contrib"`
	ICMSRedTargetContrib           ICMSReductionTarget `json:"icms_red_target_contrib"`
	ICMSRedPctNonContrib           *float64            `json:"icms_red_pct_non_contrib"`
	ICMSRedTargetNonContrib        ICMSReductionTarget `json:"icms_red_target_non_contrib"`
	LegalDeviceICMSRedContribID    *int64              `json:"legal_device_icms_red_contrib_id"`
	LegalDeviceICMSRedNonContribID *int64              `json:"legal_device_icms_red_non_contrib_id"`
	// Diferimento
	ICMSDeferralPct           *float64            `json:"icms_deferral_pct"`
	ICMSDeferralTarget        ICMSReductionTarget `json:"icms_deferral_target"`
	LegalDeviceICMSDeferralID *int64              `json:"legal_device_icms_deferral_id"`
	ICMSDeferralBenefitCode   *string             `json:"icms_deferral_benefit_code"`
	// Substituição tributária
	ICMSSubstPctContrib              *float64 `json:"icms_subst_pct_contrib"`
	ICMSSubstPctNonContrib           *float64 `json:"icms_subst_pct_non_contrib"`
	ICMSSubstPctContribUC            *float64 `json:"icms_subst_pct_contrib_uc"`
	ICMSSubstRedPct                  *float64 `json:"icms_subst_red_pct"`
	ICMSInternalPct                  *float64 `json:"icms_internal_pct"`
	LegalDeviceICMSSubstContribID    *int64   `json:"legal_device_icms_subst_contrib_id"`
	LegalDeviceICMSSubstNonContribID *int64   `json:"legal_device_icms_subst_non_contrib_id"`
	LegalDeviceICMSSubstRedID        *int64   `json:"legal_device_icms_subst_red_id"`
	ModBCICMSST                      *string  `json:"mod_bc_icms_st"`
	ICMSPctForSTContrib              *float64 `json:"icms_pct_for_st_contrib"`
	ICMSPctForSTNonContrib           *float64 `json:"icms_pct_for_st_non_contrib"`
	// CST / CSOSN
	CSTICMSContrib       *string `json:"cst_icms_contrib"`
	CSTICMSNonContrib    *string `json:"cst_icms_non_contrib"`
	CSOSNTICMS           *string `json:"csosn_icms"`
	CSTICMSContribDev    *string `json:"cst_icms_contrib_dev"`
	CSTICMSNonContribDev *string `json:"cst_icms_non_contrib_dev"`
	CSTSitTribB          *string `json:"cst_sit_trib_b"`
	// Benefício fiscal
	FiscalBenefitCodeContrib    *string `json:"fiscal_benefit_code_contrib"`
	FiscalBenefitCodeNonContrib *string `json:"fiscal_benefit_code_non_contrib"`
	FiscalBenefitCode           *string `json:"fiscal_benefit_code"`
	// IPI redução
	IPIRedPctContrib           *float64            `json:"ipi_red_pct_contrib"`
	IPIRedTargetContrib        ICMSReductionTarget `json:"ipi_red_target_contrib"`
	IPIRedPctNonContrib        *float64            `json:"ipi_red_pct_non_contrib"`
	IPIRedTargetNonContrib     ICMSReductionTarget `json:"ipi_red_target_non_contrib"`
	LegalDeviceIPIContribID    *int64              `json:"legal_device_ipi_contrib_id"`
	LegalDeviceIPINonContribID *int64              `json:"legal_device_ipi_non_contrib_id"`
	CSTIPIOut                  *string             `json:"cst_ipi_out"`
	CSTIPIIn                   *string             `json:"cst_ipi_in"`
	// FCI
	FCIICMSPct             *float64 `json:"fci_icms_pct"`
	FCIReduceBase          bool     `json:"fci_reduce_base"`
	FCIICMSSubstPct        *float64 `json:"fci_icms_subst_pct"`
	FCICSTICMs             *string  `json:"fci_cst_icms"`
	FCIUseICMSZF           bool     `json:"fci_use_icms_zf"`
	FCIDIFALSTContribUCPct *float64 `json:"fci_difal_st_contrib_uc_pct"`
	// Acréscimos ICMS
	ICMSAddPctContrib        *float64 `json:"icms_add_pct_contrib"`
	ICMSAddTypeContrib       *string  `json:"icms_add_type_contrib"`
	ICMSAddSumAliqContrib    bool     `json:"icms_add_sum_aliq_contrib"`
	ICMSAddPctNonContrib     *float64 `json:"icms_add_pct_non_contrib"`
	ICMSAddTypeNonContrib    *string  `json:"icms_add_type_non_contrib"`
	ICMSAddSumAliqNonContrib bool     `json:"icms_add_sum_aliq_non_contrib"`
	ICMSSTAddPctContrib      *float64 `json:"icms_st_add_pct_contrib"`
	ICMSSTAddTypeContrib     *string  `json:"icms_st_add_type_contrib"`
	ICMSSTAddPctNonContrib   *float64 `json:"icms_st_add_pct_non_contrib"`
	ICMSSTAddTypeNonContrib  *string  `json:"icms_st_add_type_non_contrib"`
	FCPPartitionPct          *float64 `json:"fcp_partition_pct"`
	// DIFAL EC 87/2015
	DIFALICMSRedPct     *float64 `json:"difal_icms_red_pct"`
	DIFALICMSType       *string  `json:"difal_icms_type"`
	DIFALPurchaseRedPct *float64 `json:"difal_purchase_red_pct"`
	// Optante Simples
	IsSimplesOptante bool      `json:"is_simples_optante"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
}

// ─── ICMS Summary Entry Additional (Aba Adicionais) ──────────────────────────

type ArrecadacaoIndicator string

const (
	ArrecadacaoSEFAZ           ArrecadacaoIndicator = "SEFAZ"
	ArrecadacaoJusticaFederal  ArrecadacaoIndicator = "JUSTICA_FEDERAL"
	ArrecadacaoJusticaEstadual ArrecadacaoIndicator = "JUSTICA_ESTADUAL"
	ArrecadacaoOutros          ArrecadacaoIndicator = "OUTROS"
)

type ICMSSummaryEntryAdditional struct {
	ID                   int64                `json:"id"`
	SummaryEntryID       int64                `json:"summary_entry_id"`
	Sequence             int                  `json:"sequence"`
	ArrecadacaoIndicator ArrecadacaoIndicator `json:"arrecadacao_indicator"`
	Processo             *string              `json:"processo"`
	Arrecadacao          *string              `json:"arrecadacao"`
	Description          *string              `json:"description"`
	DIEFTable            *string              `json:"dief_table"`
	DIEFCode             *string              `json:"dief_code"`
	CreatedAt            time.Time            `json:"created_at"`
}

// ─── ICMS ST Restitution / Ressarcimento / Complementação ────────────────────

type ICMSSTRestitutionType string

const (
	ICMSSTRestituicao    ICMSSTRestitutionType = "RESTITUICAO"
	ICMSSTRessarcimento  ICMSSTRestitutionType = "RESSARCIMENTO"
	ICMSSTComplementacao ICMSSTRestitutionType = "COMPLEMENTACAO"
)

type ICMSSTRestitution struct {
	ID                      int64                 `json:"id"`
	EmpresaID               int                   `json:"empresa_id"`
	Period                  string                `json:"period"`
	RestitutionType         ICMSSTRestitutionType `json:"restitution_type"`
	UF                      string                `json:"uf"`
	OrigDocModel            *string               `json:"orig_doc_model"`
	OrigDocSeries           *string               `json:"orig_doc_series"`
	OrigDocNumber           *string               `json:"orig_doc_number"`
	OrigDocDate             *time.Time            `json:"orig_doc_date"`
	OrigEmitterCNPJ         *string               `json:"orig_emitter_cnpj"`
	OrigEmitterIE           *string               `json:"orig_emitter_ie"`
	ItemID                  *int64                `json:"item_id"`
	ItemCode                *string               `json:"item_code"`
	CFOP                    *string               `json:"cfop"`
	MotivoCode              *string               `json:"motivo_code"`
	CSTICMS                 *string               `json:"cst_icms"`
	ICMSSTBase              float64               `json:"icms_st_base"`
	ICMSSTAliq              float64               `json:"icms_st_aliq"`
	ICMSSTValue             float64               `json:"icms_st_value"`
	ICMSSTBaseRestitution   float64               `json:"icms_st_base_restitution"`
	ICMSSTValueRestitution  float64               `json:"icms_st_value_restitution"`
	ICMSSTConsolidatedBase  float64               `json:"icms_st_consolidated_base"`
	ICMSSTConsolidatedValue float64               `json:"icms_st_consolidated_value"`
	H030IndEstoque          *string               `json:"h030_ind_estoque"`
	SpedBlock               *string               `json:"sped_block"`
	IsActive                bool                  `json:"is_active"`
	CreatedAt               time.Time             `json:"created_at"`
}

// ─── Nota Especial de Ajuste ──────────────────────────────────────────────────

type SpecialNotePurpose string

const (
	SpecialNoteComplementar SpecialNotePurpose = "COMPLEMENTAR"
	SpecialNoteAjuste       SpecialNotePurpose = "AJUSTE"
)

type SpecialNoteStatus string

const (
	SpecialNoteRascunho  SpecialNoteStatus = "RASCUNHO"
	SpecialNoteEmitida   SpecialNoteStatus = "EMITIDA"
	SpecialNoteCancelada SpecialNoteStatus = "CANCELADA"
)

type SpecialAdjustmentNote struct {
	ID                      int64                       `json:"id"`
	EmpresaID               int                         `json:"empresa_id"`
	Purpose                 SpecialNotePurpose          `json:"purpose"`
	Status                  SpecialNoteStatus           `json:"status"`
	Number                  *string                     `json:"number"`
	Series                  *string                     `json:"series"`
	IssueDate               time.Time                   `json:"issue_date"`
	Period                  string                      `json:"period"`
	InvoiceTypeID           *int64                      `json:"invoice_type_id"`
	CFOPID                  *int64                      `json:"cfop_id"`
	ICMSApuracaoLineID      *int64                      `json:"icms_apuracao_line_id"`
	AdjustmentCodeID        *int64                      `json:"adjustment_code_id"`
	AdjustmentDocCodeID     *int64                      `json:"adjustment_doc_code_id"`
	History                 *string                     `json:"history"`
	AutoGenerateSummary     bool                        `json:"auto_generate_summary"`
	GeneratedSummaryEntryID *int64                      `json:"generated_summary_entry_id"`
	TotalValue              float64                     `json:"total_value"`
	TotalICMS               float64                     `json:"total_icms"`
	TotalIPI                float64                     `json:"total_ipi"`
	Observation             *string                     `json:"observation"`
	Items                   []SpecialAdjustmentNoteItem `json:"items"`
	CreatedAt               time.Time                   `json:"created_at"`
}

type SpecialAdjustmentNoteItem struct {
	ID                int64     `json:"id"`
	NoteID            int64     `json:"note_id"`
	Sequence          int       `json:"sequence"`
	ItemID            *int64    `json:"item_id"`
	ItemCode          *string   `json:"item_code"`
	Description       *string   `json:"description"`
	Quantity          float64   `json:"quantity"`
	Unit              *string   `json:"unit"`
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
	CSTICMS           *string   `json:"cst_icms"`
	CSTIPI            *string   `json:"cst_ipi"`
	CFOPID            *int64    `json:"cfop_id"`
	CreatedAt         time.Time `json:"created_at"`
}
