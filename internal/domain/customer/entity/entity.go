package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ─── Enums ───────────────────────────────────────────────────────────────────

type CustomerCategory string

const (
	CategoryNormal    CustomerCategory = "NORMAL"
	CategoryConsumidor CustomerCategory = "CONSUMIDOR"
)

type CarrierBillingType string

const (
	BillingCarteira          CarrierBillingType = "CARTEIRA"
	BillingCobrancaEscritural CarrierBillingType = "COBRANCA_ESCRITURAL"
	BillingBoleto            CarrierBillingType = "BOLETO"
)

type PaymentAnalysis string

const (
	AnalysisSempreAnalisa    PaymentAnalysis = "SEMPRE_ANALISA"
	AnalysisBloqueiaSempre   PaymentAnalysis = "BLOQUEIA_SEMPRE"
	AnalysisLiberaSemAnalise PaymentAnalysis = "LIBERA_SEM_ANALISE"
)

type PaymentParcelStart string

const (
	ParcelStartEmissao          PaymentParcelStart = "EMISSAO"
	ParcelStartProximoMes       PaymentParcelStart = "PROXIMO_MES"
	ParcelStartProximaQuinzena  PaymentParcelStart = "PROXIMA_QUINZENA"
)

type PriceFormation string

const (
	PriceInformado            PriceFormation = "INFORMADO"
	PriceCustoMedio           PriceFormation = "CUSTO_MEDIO"
	PriceCustoStandardTotal   PriceFormation = "CUSTO_STANDARD_TOTAL"
	PriceCustoStandardMaterial PriceFormation = "CUSTO_STANDARD_MATERIAL"
	PriceInformadoSemICMS     PriceFormation = "INFORMADO_SEM_ICMS"
	PriceMatOper              PriceFormation = "MAT_OPER"
	PriceTabelaCusto          PriceFormation = "TABELA_CUSTO"
	PriceTransferenciaIPI     PriceFormation = "TRANSFERENCIA_IPI"
	PriceTransferenciaUF      PriceFormation = "TRANSFERENCIA_UF"
)

type TableComposition string

const (
	CompositionExwork TableComposition = "EXWORK"
	CompositionCIF    TableComposition = "CIF"
	CompositionFOB    TableComposition = "FOB"
)

type TableType string

const (
	TableTypeNormal      TableType = "NORMAL"
	TableTypePromocional TableType = "PROMOCIONAL"
)

type BaseDate string

const (
	BaseDatePedido    BaseDate = "PEDIDO"
	BaseDateDataAtual BaseDate = "DATA_ATUAL"
)

type InvoiceTypeKind string

const (
	InvoiceVenda               InvoiceTypeKind = "VENDA"
	InvoiceDevolucao           InvoiceTypeKind = "DEVOLUCAO"
	InvoiceRemessa             InvoiceTypeKind = "REMESSA"
	InvoiceRemessaConsignacao  InvoiceTypeKind = "REMESSA_CONSIGNACAO"
	InvoiceRemessaArmazenagem  InvoiceTypeKind = "REMESSA_ARMAZENAGEM"
	InvoiceRemessaBeneficiamento InvoiceTypeKind = "REMESSA_BENEFICIAMENTO"
	InvoiceRetornoBeneficiamento InvoiceTypeKind = "RETORNO_BENEFICIAMENTO"
	InvoiceSimplesRemessa      InvoiceTypeKind = "SIMPLES_REMESSA"
	InvoiceTransferencia       InvoiceTypeKind = "TRANSFERENCIA"
	InvoiceVendaConsignacao    InvoiceTypeKind = "VENDA_CONSIGNACAO"
	InvoiceComplementarICM     InvoiceTypeKind = "COMPLEMENTAR_ICM"
	InvoiceComplementarIPI     InvoiceTypeKind = "COMPLEMENTAR_IPI"
	InvoiceDemonstracao        InvoiceTypeKind = "DEMONSTRACAO"
	InvoiceEmprestimo          InvoiceTypeKind = "EMPRESTIMO"
	InvoiceFaturamentoAntecipado InvoiceTypeKind = "FATURAMENTO_ANTECIPADO"
	InvoicePrestacaoServicos   InvoiceTypeKind = "PRESTACAO_SERVICOS"
	InvoiceOutros              InvoiceTypeKind = "OUTROS"
)

type InvoiceStock string

const (
	StockAtualiza             InvoiceStock = "ATUALIZA"
	StockNaoAtualiza          InvoiceStock = "NAO_ATUALIZA"
	StockTransferenciaExterna InvoiceStock = "TRANSFERENCIA_EXTERNA"
)

type ImpostosNFe string

const (
	ImpostosICMS       ImpostosNFe = "ICMS"
	ImpostosIPI        ImpostosNFe = "IPI"
	ImpostosPIS        ImpostosNFe = "PIS"
	ImpostosCOFINS     ImpostosNFe = "COFINS"
	ImpostosICMSIPI    ImpostosNFe = "ICMS_IPI"
	ImpostosTodos      ImpostosNFe = "TODOS"
)

type PriceSituation string

const (
	PriceSituationAtivo      PriceSituation = "ATIVO"
	PriceSituationInativo    PriceSituation = "INATIVO"
	PriceSituationPromocional PriceSituation = "PROMOCIONAL"
)


type InvoiceICMSType string

const (
	ICMSTributado InvoiceICMSType = "TRIBUTADO"
	ICMSIsento    InvoiceICMSType = "ISENTO"
	ICMSOutros    InvoiceICMSType = "OUTROS"
)

type DocumentType string

const (
	DocumentCNPJ       DocumentType = "CNPJ"
	DocumentCPF        DocumentType = "CPF"
	DocumentEstrangeiro DocumentType = "ESTRANGEIRO"
	DocumentIsento     DocumentType = "ISENTO"
)

type AddressType string

const (
	AddressCobranca  AddressType = "COBRANCA"
	AddressEntrega   AddressType = "ENTREGA"
	AddressComercial AddressType = "COMERCIAL"
	AddressOutro     AddressType = "OUTRO"
)

type PaymentCondVisibility string

const (
	VisibilitySomenteVinculados PaymentCondVisibility = "SOMENTE_VINCULADOS"
	VisibilityVinculadosENenhum PaymentCondVisibility = "VINCULADOS_E_NENHUM"
	VisibilityTodos             PaymentCondVisibility = "TODOS"
)

// ─── Region ──────────────────────────────────────────────────────────────────

type Region struct {
	ID          int64
	Code        int64
	Description string
	UF          string
	City        string
	IsActive    bool
	CreatedAt   time.Time
	CreatedBy   uuid.UUID
}

func NewRegion(code int64, description string, uf, city string, createdBy uuid.UUID) (*Region, error) {
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if uf == "" {
		return nil, fmt.Errorf("uf is required")
	}
	if city == "" {
		return nil, fmt.Errorf("city is required")
	}
	return &Region{
		Code:        code,
		Description: description,
		UF:          uf,
		City:        city,
		IsActive:    true,
		CreatedAt:   time.Now(),
		CreatedBy:   createdBy,
	}, nil
}

// ─── Market Segment ───────────────────────────────────────────────────────────

type MarketSegment struct {
	ID                     int64
	Code                   int64
	Description            string
	ParentID               *int64
	HasPISCOFINSRetention  bool
	RetentionIndicator     *int16
	IsActive               bool
	CreatedAt              time.Time
}

func NewMarketSegment(code int64, description string, parentID *int64, hasRetention bool, retentionIndicator *int16) (*MarketSegment, error) {
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if hasRetention && retentionIndicator == nil {
		return nil, fmt.Errorf("retention_indicator is required when has_pis_cofins_retention is true")
	}
	return &MarketSegment{
		Code:                  code,
		Description:           description,
		ParentID:              parentID,
		HasPISCOFINSRetention: hasRetention,
		RetentionIndicator:    retentionIndicator,
		IsActive:              true,
		CreatedAt:             time.Now(),
	}, nil
}

// ─── Customer Contact Type ────────────────────────────────────────────────────

type CustomerContactType struct {
	ID          int64
	Code        int64
	Description string
	IsActive    bool
	CreatedAt   time.Time
}

func NewContactType(code int64, description string) (*CustomerContactType, error) {
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	return &CustomerContactType{
		Code:        code,
		Description: description,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}, nil
}

// ─── Customer Type ────────────────────────────────────────────────────────────

type CustomerType struct {
	ID           int64
	Code         int64
	Description  string
	Category     CustomerCategory
	DeliveryDays int16
	IsActive     bool
	CreatedAt    time.Time
}

func NewCustomerType(code int64, description string, category CustomerCategory, deliveryDays int16) (*CustomerType, error) {
	if code == 0 {
		return nil, fmt.Errorf("code is required")
	}
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if category == "" {
		category = CategoryNormal
	}
	return &CustomerType{
		Code:         code,
		Description:  description,
		Category:     category,
		DeliveryDays: deliveryDays,
		IsActive:     true,
		CreatedAt:    time.Now(),
	}, nil
}

// ─── Carrier Group ────────────────────────────────────────────────────────────

type CarrierGroup struct {
	ID          int64
	Code        int64
	Description string
	CreatedAt   time.Time
}

// ─── Carrier ─────────────────────────────────────────────────────────────────

type Carrier struct {
	ID                 int64
	Code               int64
	Description        string
	BillingType        CarrierBillingType
	UsesCreditLimit    bool
	ConsiderAvailable  bool
	PostponeDueDate    bool
	ReceiptDays        int16
	PaymentDays        int16
	IsActive           bool
	CreatedAt          time.Time
}

func NewCarrier(code int64, description string, billingType CarrierBillingType) (*Carrier, error) {
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if billingType == "" {
		billingType = BillingCarteira
	}
	return &Carrier{
		Code:        code,
		Description: description,
		BillingType: billingType,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}, nil
}

// ─── Payment Condition ────────────────────────────────────────────────────────

type PaymentCondition struct {
	ID           int64
	Code         int64
	Description  string
	CarrierID    *int64
	AnalysisType PaymentAnalysis
	ParcelStart  PaymentParcelStart
	Expenses     float64
	AverageTerm  int16
	IsSpecial    bool
	IsRevenue    bool
	IsAtSight    bool
	IsActive     bool
	CreatedAt    time.Time
	Installments []*PaymentInstallment
}

type PaymentInstallment struct {
	ID                  int64
	PaymentConditionID  int64
	InstallmentNumber   int16
	DueDays             int16
	Description         *string
	DocumentType        *string
	MovementType        *string
	CarrierID           *int64
	IsActive            bool
}

func NewPaymentCondition(code int64, description string, analysisType PaymentAnalysis) (*PaymentCondition, error) {
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if analysisType == "" {
		analysisType = AnalysisLiberaSemAnalise
	}
	return &PaymentCondition{
		Code:         code,
		Description:  description,
		AnalysisType: analysisType,
		ParcelStart:  ParcelStartEmissao,
		IsActive:     true,
		CreatedAt:    time.Now(),
	}, nil
}

// ─── Sales Table ──────────────────────────────────────────────────────────────

type SalesTable struct {
	ID                        int64
	Code                      int64
	Description               string
	ValidityStart             *time.Time
	ValidityEnd               *time.Time
	ToleranceMinPct           float64
	ToleranceMaxPct           float64
	PriceFormation            PriceFormation
	DecimalPlaces             int16
	IsActive                  bool
	Composition               TableComposition
	TableType                 TableType
	BaseDate                  BaseDate
	AllowItemsBelowCent       bool
	ICMSInterestadualPorDentro bool
	Observation               *string
	CreatedAt                 time.Time
}

func NewSalesTable(code int64, description string, formation PriceFormation) (*SalesTable, error) {
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if formation == "" {
		formation = PriceInformado
	}
	return &SalesTable{
		Code:           code,
		Description:    description,
		PriceFormation: formation,
		DecimalPlaces:  2,
		IsActive:       true,
		Composition:    CompositionFOB,
		TableType:      TableTypeNormal,
		BaseDate:       BaseDatePedido,
		CreatedAt:      time.Now(),
	}, nil
}

// ─── Invoice Type ─────────────────────────────────────────────────────────────

type InvoiceType struct {
	ID                       int64
	Code                     int64
	Description              string
	Type                     InvoiceTypeKind
	StockMovement            InvoiceStock
	ICMSType                 InvoiceICMSType
	ICMSPct                  float64
	ICMSReductionPct         float64
	IPIPct                   float64
	PISPct                   float64
	COFINSPct                float64
	ISSQNPct                 float64
	IRPct                    float64
	CSLLPct                  float64
	INSSPct                  float64
	GeneratesRevenue         bool
	UpdatesInventory         bool
	GeneratesFinancialTitle  bool
	ConsidersGoals           bool
	CalcSubstitutionTax      bool
	CalcICMSDeferral         bool
	CalcPISCOFINS            bool
	CalcDIFAL                bool
	RequiresSalesOrder       bool
	ListsFiscalBooks         bool
	IsActive                 bool
	// NF-e / FocusNFE fields
	ModelNF             *string // "55" = NF-e, "65" = NFC-e
	DescriptionNF       *string
	ImpostosNFe         *ImpostosNFe
	CSTICMS             *string
	CSOSNTICMS          *string
	CSTIPI              *string
	CSTPIS              *string
	CSTCOFINS           *string
	BaixaPedido         bool
	GeraTituloDev       bool
	ExigeSuframa        bool
	IRPctPresumption    float64
	CSLLPctPresumption  float64
	// FKs
	CFOPId                      *int64
	DispositivoLegalIPIId       *int64
	DispositivoLegalICMSId      *int64
	DispositivoLegalICMSSTId    *int64
	DispositivoLegalPISId       *int64
	DispositivoLegalCOFINSId    *int64
	HierarchyIPI                *string
	HierarchyICMS               *string
	HierarchyICMSST             *string
	HierarchyPIS                *string
	HierarchyCOFINS             *string
	IPITransferSalesTableId     *int64
	// SPED/SINTEGRA flags
	ListaValorContabil          bool
	ListaRegistroSaida          bool
	ListaICMSIPI                bool
	SintegraSpedFiscal          bool
	// Calculation/behavior flags
	CalcFomentar                bool
	ExcecaoFomentar             bool
	CompRessRetST               bool
	CalcReducao                 bool
	ComplementoItens            bool
	BuscaTipoNF                 bool
	ICMSSTUltEntrada            bool
	SomenteConsultaLotes        bool
	CalcImpIBPT                 bool
	CredPresumidoICMS           bool
	CIAP                        bool
	VlrAgregadoBaseSubst        bool
	ContratoFacon               bool
	DescICMSLicitacoes          bool
	Sisdeclara                  bool
	// Classification codes
	CodClasTrib                 *string
	CodClasTribTribReg          *string
	CodMotivoRestCompICMSST     *string
	CodBeneficioFiscal          *string
	CreatedAt                   time.Time
}

func NewInvoiceType(code int64, description string, kind InvoiceTypeKind) (*InvoiceType, error) {
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if kind == "" {
		kind = InvoiceVenda
	}
	return &InvoiceType{
		Code:                    code,
		Description:             description,
		Type:                    kind,
		StockMovement:           StockAtualiza,
		ICMSType:                ICMSTributado,
		GeneratesRevenue:        true,
		UpdatesInventory:        true,
		GeneratesFinancialTitle: true,
		CalcDIFAL:               true,
		ListsFiscalBooks:        true,
		BaixaPedido:             true,
		IsActive:                true,
		CreatedAt:               time.Now(),
	}, nil
}

// ─── Tax Type ─────────────────────────────────────────────────────────────────

type TaxType struct {
	ID                           int64
	Code                         int64
	Description                  string
	IPIBaseTotalItems            bool
	IPIBaseSubtractDiscount      bool
	IPIBaseAddFreight            bool
	IPIBaseAddExpenses           bool
	ICMSBaseTotalItems           bool
	ICMSBaseSubtractDiscount     bool
	ICMSBaseAddFreight           bool
	ICMSBaseAddIPI               bool
	ICMSBaseAddExpenses          bool
	PISCOFINSBaseTotalItems      bool
	PISCOFINSBaseSubtractDiscount bool
	PISCOFINSBaseAddFreight      bool
	PISCOFINSBaseAddInsurance    bool
	PISCOFINSBaseAddExpenses     bool
	CSLLBaseTotalItems           bool
	CSLLBaseSubtractDiscount     bool
	CSLLBaseAddFreight           bool
	IRBaseTotalItems             bool
	IRBaseSubtractDiscount       bool
	IRBaseAddFreight             bool
	IsConsumer                   bool
	IsActive                     bool
	CreatedAt                    time.Time
}

func NewTaxType(code int64, description string) (*TaxType, error) {
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	return &TaxType{
		Code:                         code,
		Description:                  description,
		IPIBaseTotalItems:            true,
		ICMSBaseTotalItems:           true,
		ICMSBaseSubtractDiscount:     true,
		ICMSBaseAddFreight:           true,
		PISCOFINSBaseTotalItems:      true,
		PISCOFINSBaseSubtractDiscount: true,
		CSLLBaseTotalItems:           true,
		CSLLBaseSubtractDiscount:     true,
		IRBaseTotalItems:             true,
		IRBaseSubtractDiscount:       true,
		IsActive:                     true,
		CreatedAt:                    time.Now(),
	}, nil
}

// ─── Customer ─────────────────────────────────────────────────────────────────

type Customer struct {
	ID                     int64
	Code                   int64
	CorporateCode          *int64
	IsCorporate            bool
	Name                   string
	TradeName              *string
	DocumentType           DocumentType
	DocumentNumber         string
	StateRegistration      *string
	MunicipalRegistration  *string
	SuframaCode            *string
	SuframaExpiry          *time.Time
	RegionID               *int64
	MarketSegmentID        *int64
	CustomerTypeID         *int64
	PaymentConditionID     *int64
	SalesTableID           *int64
	CarrierID              *int64
	CarrierGroupID         *int64
	InvoiceTypeID          *int64
	TaxTypeID              *int64
	PaymentCondVisibility  PaymentCondVisibility
	CreditLimit            float64
	Website                *string
	IsActive               bool
	Blocked                bool
	BlockReason            *string
	CreatedAt              time.Time
	CreatedBy              uuid.UUID
	UpdatedAt              time.Time
	Addresses              []*CustomerAddress
	Contacts               []*CustomerContact
}

func NewCustomer(code int64, name string, docType DocumentType, docNumber string, createdBy uuid.UUID) (*Customer, error) {
	if code == 0 {
		return nil, fmt.Errorf("code is required")
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if docNumber == "" {
		return nil, fmt.Errorf("document_number is required")
	}
	if docType == "" {
		docType = DocumentCNPJ
	}
	now := time.Now()
	return &Customer{
		Code:                  code,
		Name:                  name,
		DocumentType:          docType,
		DocumentNumber:        docNumber,
		PaymentCondVisibility: VisibilityTodos,
		IsActive:              true,
		Blocked:               false,
		CreatedAt:             now,
		CreatedBy:             createdBy,
		UpdatedAt:             now,
	}, nil
}

// ─── Customer Address ─────────────────────────────────────────────────────────

type CustomerAddress struct {
	ID           int64
	CustomerID   int64
	AddressType  AddressType
	ZipCode      *string
	Street       *string
	Number       *string
	Complement   *string
	Neighborhood *string
	City         *string
	UF           *string
	Country      string
	IsDefault    bool
	CreatedAt    time.Time
}

// ─── Customer Contact ─────────────────────────────────────────────────────────

type CustomerContact struct {
	ID              int64
	CustomerID      int64
	ContactTypeID   *int64
	Name            string
	Email           *string
	Phone           *string
	Mobile          *string
	Position        *string
	IsPrimary       bool
	IsActive        bool
	CreatedAt       time.Time
}

// ─── Sales Table Price ────────────────────────────────────────────────────────

type SalesTablePrice struct {
	ID            int64
	SalesTableID  int64
	ItemCode      string
	Price         float64
	UME           *string
	UMC           *string
	PriceConv     float64
	Formula       *string
	Situation     PriceSituation
	Blocked       bool
	Observation   *string
	ProductLineID *int64
	ItemMask      *string
	CreatedAt     time.Time
}

