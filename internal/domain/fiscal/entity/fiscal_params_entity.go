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
	CfopTipoUtilizacaoNormal                     CfopTipoUtilizacao = "NORMAL"
	CfopTipoUtilizacaoVendaComercialExportadora   CfopTipoUtilizacao = "VENDA_COMERCIAL_EXPORTADORA"
	CfopTipoUtilizacaoCompraFimEspecificoExportacao CfopTipoUtilizacao = "COMPRA_FIM_ESPECIFICO_EXPORTACAO"
	CfopTipoUtilizacaoExportacao                  CfopTipoUtilizacao = "EXPORTACAO"
)

type CFOP struct {
	ID               int64
	Code             int32
	Description      string
	DescriptionFull  *string
	Utilization      CfopUtilization
	OrigemClasIPI    *string
	IndOperacao      CfopIndOperacao
	TipoUtilizacao   CfopTipoUtilizacao
	CodigoAnexoSN    *string
	DIFAL            bool
	Doacao           bool
	IsActive         bool
	CreatedAt        time.Time
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
	NCMCode               *string
	ItemCode              *int64
	ItemConfigMask        *string
	UF                    string
	OperationType         TaxParamOperation

	// ── Optional FK filters ──────────────────────────────────────────────────
	CustomerCode              *int64
	CustomerEstablishmentCode *int64
	MarketSegmentID           *int64
	InvoiceTypeExitID         *int64
	InvoiceTypeEntryID        *int64
	TaxTypeID                 *int64

	// ── Flags ────────────────────────────────────────────────────────────────
	IsPreferred       bool
	IsSimpleOptante   bool

	// ── ICMS rates ───────────────────────────────────────────────────────────
	ICMSPctContrib          float64
	LegalDeviceICMSContribID *int64
	ICMSPctNonContrib       float64
	LegalDeviceICMSNonContribID *int64

	// ── ICMS reduction ───────────────────────────────────────────────────────
	ICMSRedPctContrib           float64
	ICMSRedTargetContrib        *IcmsReductionTarget
	LegalDeviceICMSRedContribID *int64
	ICMSRedPctNonContrib        float64
	ICMSRedTargetNonContrib     *IcmsReductionTarget
	LegalDeviceICMSRedNonContribID *int64

	// ── ICMS deferral ────────────────────────────────────────────────────────
	ICMSDeferralPct            float64
	ICMSDeferralTarget         *IcmsReductionTarget
	LegalDeviceICMSDeferralID  *int64
	CodBenefRBC                *string

	// ── ICMS substitution ────────────────────────────────────────────────────
	ICMSSubstPctContrib          float64
	LegalDeviceICMSSubstContribID *int64
	ICMSSubstPctNonContrib       float64
	LegalDeviceICMSSubstNonContribID *int64
	ICMSSubstPctContribUC        float64
	ICMSSubstRedPct              float64
	LegalDeviceICMSSubstRedID    *int64

	// ── ICMS ST calculation ──────────────────────────────────────────────────
	ICMSInternalPct         float64
	BCICMSSTModality        *string
	ICMSPctForSTContrib     float64
	ICMSPctForSTNonContrib  float64

	// ── CST / CSOSN ──────────────────────────────────────────────────────────
	CSTSituationB       *string
	CSOSNIGMS           *string
	CSTICMSContrib      *string
	CSTICMSNonContrib   *string
	CodBeneficioFiscal  *string
	CSTICMSContribDev   *string
	CSTICMSNonContribDev *string

	// ── IPI reduction ────────────────────────────────────────────────────────
	IPIRedPctContrib          float64
	IPIRedTargetContrib       *IcmsReductionTarget
	LegalDeviceIPIContribID   *int64
	IPIRedPctNonContrib       float64
	IPIRedTargetNonContrib    *IcmsReductionTarget
	LegalDeviceIPINonContribID *int64
	CSTIPIExit                *string
	CSTIPIEntry               *string

	// ── FCI ──────────────────────────────────────────────────────────────────
	ICMSPctOrigins1238       float64
	CalcBaseRedFCI           bool
	ICMSSubstPctOrigins1238  float64
	CSTICMSFci               *string
	UsesICMSZonaFranca       bool
	DifAliqSTContribUC       float64
	CodBenefContrib          *string
	CodBenefNonContrib       *string

	// ── ICMS additions ───────────────────────────────────────────────────────
	ICMSAcresPctContrib        float64
	ICMSAcresTypeContrib       *IcmsAcresType
	ICMSAcresSumContrib        bool
	ICMSAcresPctNonContrib     float64
	ICMSAcresTypeNonContrib    *IcmsAcresType
	ICMSAcresSumNonContrib     bool
	ICMSSTAcresPctContrib      float64
	ICMSSTAcresTypeContrib     *IcmsAcresType
	ICMSSTAcresSumContrib      bool
	ICMSSTAcresPctNonContrib   float64
	ICMSSTAcresTypeNonContrib  *IcmsAcresType
	ICMSSTAcresSumNonContrib   bool
	FCPSTPartilhaPct           float64

	// ── DIFAL (EC/87) ────────────────────────────────────────────────────────
	ICMSDifalRedPct   float64
	ICMSDifalType     *IcmsDifalType

	// ── DIFAL purchases ──────────────────────────────────────────────────────
	DifalPurchaseRedPct    float64
	DifalPurchaseRedTarget *IcmsReductionTarget

	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ─── DAPI Transfer Reason ─────────────────────────────────────────────────────

type DAPITransferReason struct {
	ID          int64
	Code        string
	Reason      string
	Destination *string
	ValidFrom   *time.Time
	ValidTo     *time.Time
	IsActive    bool
	CreatedAt   time.Time
}

// ─── ICMS Apuração Adjustment Code (SPED tabela 5.1.1) ───────────────────────

type ICMSApuracaoAdjustmentCode struct {
	ID          int64
	Code        string
	UF          string
	Description string
	ValidFrom   *time.Time
	ValidTo     *time.Time
	IsActive    bool
	CreatedAt   time.Time
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
	ID          int64
	UF          string
	Code        string
	Description string
	TableRef    ICMSAdjustmentTableRef
	ValidFrom   *time.Time
	ValidTo     *time.Time
	IsActive    bool
	CreatedAt   time.Time
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
	ID                       int64
	Code                     string
	Description              string
	LineType                 ApuracaoLineType
	AcceptsEntries           bool
	Nature                   *string
	ApuracaoAdjCodeID        *int64
	IsActive                 bool
	CreatedAt                time.Time
}

// ─── ICMS Summary Entry ───────────────────────────────────────────────────────

type ICMSSummaryEntry struct {
	ID             int64
	Period         string
	UF             string
	CFOPID         *int64
	ICMSBase       float64
	ICMSValue      float64
	ICMSBaseOther  float64
	ICMSValueOther float64
	IsActive       bool
	CreatedAt      time.Time
}

type ICMSSummaryEntryNote struct {
	ID             int64
	SummaryEntryID int64
	NoteNumber     string
	NoteSeries     *string
	EmitterCNPJ    *string
	IssueDate      time.Time
	ItemValue      float64
	ICMSBase       float64
	ICMSValue      float64
	Observation    *string
	CreatedAt      time.Time
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

type SimplesNacionalApuracao struct {
	ID                  int64
	Period              string
	Annex               SimplesNacionalAnnex
	ReceitaInterna      float64
	ReceitaExterna      float64
	FolhaPagamento      float64
	ReceitaBruta12M     float64
	SimplesRecolhido    float64
	AliquotaNominal     float64
	AliquotaEfetiva     float64
	AliquotaEfetivaICMS float64
	ParcelaDeduzir      float64
	Observation         *string
	IsActive            bool
	CreatedAt           time.Time
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
	ID                              int64
	ItemID                          *int64
	ItemMask                        *string
	NCMCode                         *string
	UF                              string
	OperationType                   ICMSOperationType
	CustomerID                      *int64
	EstablishmentID                 *int64
	SupplierID                      *int64
	InvoiceTypeOutID                *int64
	InvoiceTypeInID                 *int64
	MarketSegmentID                 *int64
	IsPreferential                  bool
	// ICMS alíquotas
	ICMSPctContrib                  *float64
	ICMSPctNonContrib               *float64
	LegalDeviceICMSContribID        *int64
	LegalDeviceICMSNonContribID     *int64
	// Redução ICMS
	ICMSRedPctContrib               *float64
	ICMSRedTargetContrib            ICMSReductionTarget
	ICMSRedPctNonContrib            *float64
	ICMSRedTargetNonContrib         ICMSReductionTarget
	LegalDeviceICMSRedContribID     *int64
	LegalDeviceICMSRedNonContribID  *int64
	// Diferimento
	ICMSDeferralPct                 *float64
	ICMSDeferralTarget              ICMSReductionTarget
	LegalDeviceICMSDeferralID       *int64
	ICMSDeferralBenefitCode         *string
	// Substituição tributária
	ICMSSubstPctContrib             *float64
	ICMSSubstPctNonContrib          *float64
	ICMSSubstPctContribUC           *float64
	ICMSSubstRedPct                 *float64
	ICMSInternalPct                 *float64
	LegalDeviceICMSSubstContribID   *int64
	LegalDeviceICMSSubstNonContribID *int64
	LegalDeviceICMSSubstRedID       *int64
	ModBCICMSST                     *string
	ICMSPctForSTContrib             *float64
	ICMSPctForSTNonContrib          *float64
	// CST / CSOSN
	CSTICMSContrib                  *string
	CSTICMSNonContrib               *string
	CSOSNTICMS                      *string
	CSTICMSContribDev               *string
	CSTICMSNonContribDev            *string
	CSTSitTribB                     *string
	// Benefício fiscal
	FiscalBenefitCodeContrib        *string
	FiscalBenefitCodeNonContrib     *string
	FiscalBenefitCode               *string
	// IPI redução
	IPIRedPctContrib                *float64
	IPIRedTargetContrib             ICMSReductionTarget
	IPIRedPctNonContrib             *float64
	IPIRedTargetNonContrib          ICMSReductionTarget
	LegalDeviceIPIContribID         *int64
	LegalDeviceIPINonContribID      *int64
	CSTIPIOut                       *string
	CSTIPIIn                        *string
	// FCI
	FCIICMSPct                      *float64
	FCIReduceBase                   bool
	FCIICMSSubstPct                 *float64
	FCICSTICMs                      *string
	FCIUseICMSZF                    bool
	FCIDIFALSTContribUCPct          *float64
	// Acréscimos ICMS
	ICMSAddPctContrib               *float64
	ICMSAddTypeContrib              *string
	ICMSAddSumAliqContrib           bool
	ICMSAddPctNonContrib            *float64
	ICMSAddTypeNonContrib           *string
	ICMSAddSumAliqNonContrib        bool
	ICMSSTAddPctContrib             *float64
	ICMSSTAddTypeContrib            *string
	ICMSSTAddPctNonContrib          *float64
	ICMSSTAddTypeNonContrib         *string
	FCPPartitionPct                 *float64
	// DIFAL EC 87/2015
	DIFALICMSRedPct                 *float64
	DIFALICMSType                   *string
	DIFALPurchaseRedPct             *float64
	// Optante Simples
	IsSimplesOptante                bool
	IsActive                        bool
	CreatedAt                       time.Time
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
	ID                  int64
	SummaryEntryID      int64
	Sequence            int
	ArrecadacaoIndicator ArrecadacaoIndicator
	Processo            *string
	Arrecadacao         *string
	Description         *string
	DIEFTable           *string
	DIEFCode            *string
	CreatedAt           time.Time
}

// ─── ICMS ST Restitution / Ressarcimento / Complementação ────────────────────

type ICMSSTRestitutionType string

const (
	ICMSSTRestituicao    ICMSSTRestitutionType = "RESTITUICAO"
	ICMSSTRessarcimento  ICMSSTRestitutionType = "RESSARCIMENTO"
	ICMSSTComplementacao ICMSSTRestitutionType = "COMPLEMENTACAO"
)

type ICMSSTRestitution struct {
	ID                       int64
	EmpresaID                int
	Period                   string
	RestitutionType          ICMSSTRestitutionType
	UF                       string
	OrigDocModel             *string
	OrigDocSeries            *string
	OrigDocNumber            *string
	OrigDocDate              *time.Time
	OrigEmitterCNPJ          *string
	OrigEmitterIE            *string
	ItemID                   *int64
	ItemCode                 *string
	CFOP                     *string
	MotivoCode               *string
	CSTICMS                  *string
	ICMSSTBase               float64
	ICMSSTAliq               float64
	ICMSSTValue              float64
	ICMSSTBaseRestitution    float64
	ICMSSTValueRestitution   float64
	ICMSSTConsolidatedBase   float64
	ICMSSTConsolidatedValue  float64
	H030IndEstoque           *string
	SpedBlock                *string
	IsActive                 bool
	CreatedAt                time.Time
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
	ID                      int64
	EmpresaID               int
	Purpose                 SpecialNotePurpose
	Status                  SpecialNoteStatus
	Number                  *string
	Series                  *string
	IssueDate               time.Time
	Period                  string
	InvoiceTypeID           *int64
	CFOPID                  *int64
	ICMSApuracaoLineID      *int64
	AdjustmentCodeID        *int64
	AdjustmentDocCodeID     *int64
	History                 *string
	AutoGenerateSummary     bool
	GeneratedSummaryEntryID *int64
	TotalValue              float64
	TotalICMS               float64
	TotalIPI                float64
	Observation             *string
	Items                   []SpecialAdjustmentNoteItem
	CreatedAt               time.Time
}

type SpecialAdjustmentNoteItem struct {
	ID               int64
	NoteID           int64
	Sequence         int
	ItemID           *int64
	ItemCode         *string
	Description      *string
	Quantity         float64
	Unit             *string
	UnitValue        float64
	TotalValue       float64
	ICMSBase         float64
	ICMSPct          float64
	ICMSDeferralPct  float64
	ICMSValue        float64
	ICMSDeferredValue float64
	IPIBase          float64
	IPIPct           float64
	IPIValue         float64
	CSTICMS          *string
	CSTIPI           *string
	CFOPID           *int64
	CreatedAt        time.Time
}
