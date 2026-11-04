package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

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
	Oficial         CFOPOficial
}

// CFOPOficial: o que vem da tabela oficial do Portal Nacional da NF-e (IT
// 2023.002) — vigência e os indicadores que a SEFAZ usa nas validações.
type CFOPOficial struct {
	Oficial         bool // o código está na tabela oficial
	InicioVigencia  *time.Time
	FimVigencia     *time.Time
	NFe             bool // pode ser usado em NF-e (indNFe)
	Comunicacao     bool // indComunica
	Transporte      bool // indTransp
	Devolucao       bool // indDevol: exige o documento referenciado
	Retorno         bool // indRetor
	Anulacao        bool // indAnula
	Remessa         bool // indRemes
	Combustivel     bool // indComb
	ExclusivoIBSCBS bool // indExcIBSCBS
}

// VigenteEm: dentro da vigência oficial no dia (código fora da tabela não tem
// vigência a conferir).
func (c *CFOP) VigenteEm(dia time.Time) bool {
	o := c.Oficial
	if !o.Oficial {
		return true
	}
	d := time.Date(dia.Year(), dia.Month(), dia.Day(), 0, 0, 0, 0, time.UTC)
	if o.InicioVigencia != nil && d.Before(*o.InicioVigencia) {
		return false
	}
	return o.FimVigencia == nil || !d.After(*o.FimVigencia)
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
	ICMSPctContrib              decimal.Decimal
	LegalDeviceICMSContribID    *int64
	ICMSPctNonContrib           decimal.Decimal
	LegalDeviceICMSNonContribID *int64

	// ── ICMS reduction ───────────────────────────────────────────────────────
	ICMSRedPctContrib              decimal.Decimal
	ICMSRedTargetContrib           *IcmsReductionTarget
	LegalDeviceICMSRedContribID    *int64
	ICMSRedPctNonContrib           decimal.Decimal
	ICMSRedTargetNonContrib        *IcmsReductionTarget
	LegalDeviceICMSRedNonContribID *int64

	// ── ICMS deferral ────────────────────────────────────────────────────────
	ICMSDeferralPct           decimal.Decimal
	ICMSDeferralTarget        *IcmsReductionTarget
	LegalDeviceICMSDeferralID *int64
	CodBenefRBC               *string

	// ── ICMS substitution ────────────────────────────────────────────────────
	ICMSSubstPctContrib              decimal.Decimal
	LegalDeviceICMSSubstContribID    *int64
	ICMSSubstPctNonContrib           decimal.Decimal
	LegalDeviceICMSSubstNonContribID *int64
	ICMSSubstPctContribUC            decimal.Decimal
	ICMSSubstRedPct                  decimal.Decimal
	LegalDeviceICMSSubstRedID        *int64

	// ── ICMS ST calculation ──────────────────────────────────────────────────
	ICMSInternalPct        decimal.Decimal
	BCICMSSTModality       *string
	ICMSPctForSTContrib    decimal.Decimal
	ICMSPctForSTNonContrib decimal.Decimal

	// ── CST / CSOSN ──────────────────────────────────────────────────────────
	CSTSituationB        *string
	CSOSNIGMS            *string
	CSTICMSContrib       *string
	CSTICMSNonContrib    *string
	CodBeneficioFiscal   *string
	CSTICMSContribDev    *string
	CSTICMSNonContribDev *string

	// ── IPI reduction ────────────────────────────────────────────────────────
	IPIRedPctContrib           decimal.Decimal
	IPIRedTargetContrib        *IcmsReductionTarget
	LegalDeviceIPIContribID    *int64
	IPIRedPctNonContrib        decimal.Decimal
	IPIRedTargetNonContrib     *IcmsReductionTarget
	LegalDeviceIPINonContribID *int64
	CSTIPIExit                 *string
	CSTIPIEntry                *string

	// ── FCI ──────────────────────────────────────────────────────────────────
	ICMSPctOrigins1238      decimal.Decimal
	CalcBaseRedFCI          bool
	ICMSSubstPctOrigins1238 decimal.Decimal
	CSTICMSFci              *string
	UsesICMSZonaFranca      bool
	DifAliqSTContribUC      decimal.Decimal
	CodBenefContrib         *string
	CodBenefNonContrib      *string

	// ── ICMS additions ───────────────────────────────────────────────────────
	ICMSAcresPctContrib       decimal.Decimal
	ICMSAcresTypeContrib      *IcmsAcresType
	ICMSAcresSumContrib       bool
	ICMSAcresPctNonContrib    decimal.Decimal
	ICMSAcresTypeNonContrib   *IcmsAcresType
	ICMSAcresSumNonContrib    bool
	ICMSSTAcresPctContrib     decimal.Decimal
	ICMSSTAcresTypeContrib    *IcmsAcresType
	ICMSSTAcresSumContrib     bool
	ICMSSTAcresPctNonContrib  decimal.Decimal
	ICMSSTAcresTypeNonContrib *IcmsAcresType
	ICMSSTAcresSumNonContrib  bool
	FCPSTPartilhaPct          decimal.Decimal

	// ── DIFAL (EC/87) ────────────────────────────────────────────────────────
	ICMSDifalRedPct decimal.Decimal
	ICMSDifalType   *IcmsDifalType

	// ── DIFAL purchases ──────────────────────────────────────────────────────
	DifalPurchaseRedPct    decimal.Decimal
	DifalPurchaseRedTarget *IcmsReductionTarget

	// ── Cenário ─────────────────────────────────────────────────────────────────
	// FinalidadeCompra restringe a regra a REVENDA, INDUSTRIALIZACAO ou
	// USO_CONSUMO (nil = qualquer). MvaPct é a MVA original da ST (40 = 40%) e
	// MvaAjustarInterestadual liga o ajuste da MVA na venda interestadual.
	FinalidadeCompra        *string
	MvaPct                  decimal.Decimal
	MvaAjustarInterestadual bool

	// SupplierCode restringe a regra de entrada a um fornecedor (suppliers.code);
	// regra com fornecedor não vale na saída. OrigemReducaoID: regra copiada do
	// antigo cadastro de redução/substituição (VFIS0330, migração 000433).
	SupplierCode    *int64
	OrigemReducaoID *int64

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
	ID             int64           `json:"id"`
	Period         string          `json:"period"`
	UF             string          `json:"uf"`
	CFOPID         *int64          `json:"cfop_id"`
	ICMSBase       decimal.Decimal `json:"icms_base"`
	ICMSValue      decimal.Decimal `json:"icms_value"`
	ICMSBaseOther  decimal.Decimal `json:"icms_base_other"`
	ICMSValueOther decimal.Decimal `json:"icms_value_other"`
	IsActive       bool            `json:"is_active"`
	CreatedAt      time.Time       `json:"created_at"`
}

type ICMSSummaryEntryNote struct {
	ID             int64           `json:"id"`
	SummaryEntryID int64           `json:"summary_entry_id"`
	NoteNumber     string          `json:"note_number"`
	NoteSeries     *string         `json:"note_series"`
	EmitterCNPJ    *string         `json:"emitter_cnpj"`
	IssueDate      time.Time       `json:"issue_date"`
	ItemValue      decimal.Decimal `json:"item_value"`
	ICMSBase       decimal.Decimal `json:"icms_base"`
	ICMSValue      decimal.Decimal `json:"icms_value"`
	Observation    *string         `json:"observation"`
	CreatedAt      time.Time       `json:"created_at"`
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
	ReceitaInterna      decimal.Decimal      `json:"receita_interna"`
	ReceitaExterna      decimal.Decimal      `json:"receita_externa"`
	FolhaPagamento      decimal.Decimal      `json:"folha_pagamento"`
	ReceitaBruta12M     decimal.Decimal      `json:"receita_bruta_12m"`
	SimplesRecolhido    decimal.Decimal      `json:"simples_recolhido"`
	AliquotaNominal     decimal.Decimal      `json:"aliquota_nominal"`
	AliquotaEfetiva     decimal.Decimal      `json:"aliquota_efetiva"`
	AliquotaEfetivaICMS decimal.Decimal      `json:"aliquota_efetiva_icms"`
	ParcelaDeduzir      decimal.Decimal      `json:"parcela_deduzir"`
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
	ICMSPctContrib              *decimal.Decimal `json:"icms_pct_contrib"`
	ICMSPctNonContrib           *decimal.Decimal `json:"icms_pct_non_contrib"`
	LegalDeviceICMSContribID    *int64           `json:"legal_device_icms_contrib_id"`
	LegalDeviceICMSNonContribID *int64           `json:"legal_device_icms_non_contrib_id"`
	// Redução ICMS
	ICMSRedPctContrib              *decimal.Decimal    `json:"icms_red_pct_contrib"`
	ICMSRedTargetContrib           ICMSReductionTarget `json:"icms_red_target_contrib"`
	ICMSRedPctNonContrib           *decimal.Decimal    `json:"icms_red_pct_non_contrib"`
	ICMSRedTargetNonContrib        ICMSReductionTarget `json:"icms_red_target_non_contrib"`
	LegalDeviceICMSRedContribID    *int64              `json:"legal_device_icms_red_contrib_id"`
	LegalDeviceICMSRedNonContribID *int64              `json:"legal_device_icms_red_non_contrib_id"`
	// Diferimento
	ICMSDeferralPct           *decimal.Decimal    `json:"icms_deferral_pct"`
	ICMSDeferralTarget        ICMSReductionTarget `json:"icms_deferral_target"`
	LegalDeviceICMSDeferralID *int64              `json:"legal_device_icms_deferral_id"`
	ICMSDeferralBenefitCode   *string             `json:"icms_deferral_benefit_code"`
	// Substituição tributária
	ICMSSubstPctContrib              *decimal.Decimal `json:"icms_subst_pct_contrib"`
	ICMSSubstPctNonContrib           *decimal.Decimal `json:"icms_subst_pct_non_contrib"`
	ICMSSubstPctContribUC            *decimal.Decimal `json:"icms_subst_pct_contrib_uc"`
	ICMSSubstRedPct                  *decimal.Decimal `json:"icms_subst_red_pct"`
	ICMSInternalPct                  *decimal.Decimal `json:"icms_internal_pct"`
	LegalDeviceICMSSubstContribID    *int64           `json:"legal_device_icms_subst_contrib_id"`
	LegalDeviceICMSSubstNonContribID *int64           `json:"legal_device_icms_subst_non_contrib_id"`
	LegalDeviceICMSSubstRedID        *int64           `json:"legal_device_icms_subst_red_id"`
	ModBCICMSST                      *string          `json:"mod_bc_icms_st"`
	ICMSPctForSTContrib              *decimal.Decimal `json:"icms_pct_for_st_contrib"`
	ICMSPctForSTNonContrib           *decimal.Decimal `json:"icms_pct_for_st_non_contrib"`
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
	IPIRedPctContrib           *decimal.Decimal    `json:"ipi_red_pct_contrib"`
	IPIRedTargetContrib        ICMSReductionTarget `json:"ipi_red_target_contrib"`
	IPIRedPctNonContrib        *decimal.Decimal    `json:"ipi_red_pct_non_contrib"`
	IPIRedTargetNonContrib     ICMSReductionTarget `json:"ipi_red_target_non_contrib"`
	LegalDeviceIPIContribID    *int64              `json:"legal_device_ipi_contrib_id"`
	LegalDeviceIPINonContribID *int64              `json:"legal_device_ipi_non_contrib_id"`
	CSTIPIOut                  *string             `json:"cst_ipi_out"`
	CSTIPIIn                   *string             `json:"cst_ipi_in"`
	// FCI
	FCIICMSPct             *decimal.Decimal `json:"fci_icms_pct"`
	FCIReduceBase          bool             `json:"fci_reduce_base"`
	FCIICMSSubstPct        *decimal.Decimal `json:"fci_icms_subst_pct"`
	FCICSTICMs             *string          `json:"fci_cst_icms"`
	FCIUseICMSZF           bool             `json:"fci_use_icms_zf"`
	FCIDIFALSTContribUCPct *decimal.Decimal `json:"fci_difal_st_contrib_uc_pct"`
	// Acréscimos ICMS
	ICMSAddPctContrib        *decimal.Decimal `json:"icms_add_pct_contrib"`
	ICMSAddTypeContrib       *string          `json:"icms_add_type_contrib"`
	ICMSAddSumAliqContrib    bool             `json:"icms_add_sum_aliq_contrib"`
	ICMSAddPctNonContrib     *decimal.Decimal `json:"icms_add_pct_non_contrib"`
	ICMSAddTypeNonContrib    *string          `json:"icms_add_type_non_contrib"`
	ICMSAddSumAliqNonContrib bool             `json:"icms_add_sum_aliq_non_contrib"`
	ICMSSTAddPctContrib      *decimal.Decimal `json:"icms_st_add_pct_contrib"`
	ICMSSTAddTypeContrib     *string          `json:"icms_st_add_type_contrib"`
	ICMSSTAddPctNonContrib   *decimal.Decimal `json:"icms_st_add_pct_non_contrib"`
	ICMSSTAddTypeNonContrib  *string          `json:"icms_st_add_type_non_contrib"`
	FCPPartitionPct          *decimal.Decimal `json:"fcp_partition_pct"`
	// DIFAL EC 87/2015
	DIFALICMSRedPct     *decimal.Decimal `json:"difal_icms_red_pct"`
	DIFALICMSType       *string          `json:"difal_icms_type"`
	DIFALPurchaseRedPct *decimal.Decimal `json:"difal_purchase_red_pct"`
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
	ICMSSTBase              decimal.Decimal       `json:"icms_st_base"`
	ICMSSTAliq              decimal.Decimal       `json:"icms_st_aliq"`
	ICMSSTValue             decimal.Decimal       `json:"icms_st_value"`
	ICMSSTBaseRestitution   decimal.Decimal       `json:"icms_st_base_restitution"`
	ICMSSTValueRestitution  decimal.Decimal       `json:"icms_st_value_restitution"`
	ICMSSTConsolidatedBase  decimal.Decimal       `json:"icms_st_consolidated_base"`
	ICMSSTConsolidatedValue decimal.Decimal       `json:"icms_st_consolidated_value"`
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
	TotalValue              decimal.Decimal             `json:"total_value"`
	TotalICMS               decimal.Decimal             `json:"total_icms"`
	TotalIPI                decimal.Decimal             `json:"total_ipi"`
	Observation             *string                     `json:"observation"`
	Items                   []SpecialAdjustmentNoteItem `json:"items"`
	CreatedAt               time.Time                   `json:"created_at"`
}

type SpecialAdjustmentNoteItem struct {
	ID                int64           `json:"id"`
	NoteID            int64           `json:"note_id"`
	Sequence          int             `json:"sequence"`
	ItemID            *int64          `json:"item_id"`
	ItemCode          *string         `json:"item_code"`
	Description       *string         `json:"description"`
	Quantity          decimal.Decimal `json:"quantity"`
	Unit              *string         `json:"unit"`
	UnitValue         decimal.Decimal `json:"unit_value"`
	TotalValue        decimal.Decimal `json:"total_value"`
	ICMSBase          decimal.Decimal `json:"icms_base"`
	ICMSPct           decimal.Decimal `json:"icms_pct"`
	ICMSDeferralPct   decimal.Decimal `json:"icms_deferral_pct"`
	ICMSValue         decimal.Decimal `json:"icms_value"`
	ICMSDeferredValue decimal.Decimal `json:"icms_deferred_value"`
	IPIBase           decimal.Decimal `json:"ipi_base"`
	IPIPct            decimal.Decimal `json:"ipi_pct"`
	IPIValue          decimal.Decimal `json:"ipi_value"`
	CSTICMS           *string         `json:"cst_icms"`
	CSTIPI            *string         `json:"cst_ipi"`
	CFOPID            *int64          `json:"cfop_id"`
	CreatedAt         time.Time       `json:"created_at"`
}
