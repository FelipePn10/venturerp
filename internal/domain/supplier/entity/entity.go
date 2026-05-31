package entity

import (
	"fmt"
	"regexp"
	"time"

	"github.com/FelipePn10/panossoerp/internal/pkg/validation"
	"github.com/google/uuid"
)

// ─── Enums ───────────────────────────────────────────────────────────────────

type SupplierKind string

const (
	KindNormal         SupplierKind = "NORMAL"
	KindTransportadora SupplierKind = "TRANSPORTADORA"
	KindTranspRedesp   SupplierKind = "TRANSP_REDESP"
	KindRedespacho     SupplierKind = "REDESPACHO"
)

// RequiresStateRegistration reports whether a supplier of this kind must
// inform the Inscrição Estadual. Carriers/redispatch are exempt.
func (k SupplierKind) RequiresStateRegistration() bool {
	switch k {
	case KindTransportadora, KindTranspRedesp, KindRedespacho:
		return false
	default:
		return true
	}
}

type PersonType string

const (
	PersonJuridica PersonType = "JURIDICA"
	PersonFisica   PersonType = "FISICA"
)

type DocumentType string

const (
	DocumentCNPJ        DocumentType = "CNPJ"
	DocumentCPF         DocumentType = "CPF"
	DocumentEstrangeiro DocumentType = "ESTRANGEIRO"
	DocumentIsento      DocumentType = "ISENTO"
)

type FreightType string

const (
	FreightCIF       FreightType = "CIF"
	FreightDAF       FreightType = "DAF"
	FreightFOB       FreightType = "FOB"
	FreightSemFrete  FreightType = "SEM_FRETE"
	FreightConvenio  FreightType = "CONVENIO"
	FreightRetira    FreightType = "RETIRA"
	FreightCortesia  FreightType = "CORTESIA"
	FreightTerceiros FreightType = "TERCEIROS"
)

type ViticolaObligation string

const (
	ViticolaNunca   ViticolaObligation = "NUNCA"
	ViticolaAsVezes ViticolaObligation = "AS_VEZES"
	ViticolaSempre  ViticolaObligation = "SEMPRE"
)

type ICMSContributor string

const (
	ICMSContribuinte    ICMSContributor = "CONTRIBUINTE"
	ICMSNaoContribuinte ICMSContributor = "NAO_CONTRIBUINTE"
	ICMSIsento          ICMSContributor = "ISENTO"
)

type TrackingPlatform string

const (
	TrackingSSW      TrackingPlatform = "SSW"
	TrackingFreteWeb TrackingPlatform = "FRETEWEB"
	TrackingEngloba  TrackingPlatform = "ENGLOBA_SISTEMAS"
	TrackingNenhum   TrackingPlatform = "NENHUM"
)

type AddressType string

const (
	AddressCobranca  AddressType = "COBRANCA"
	AddressEntrega   AddressType = "ENTREGA"
	AddressComercial AddressType = "COMERCIAL"
	AddressOutro     AddressType = "OUTRO"
)

type BaseDate string

const (
	BaseDateEmissao   BaseDate = "EMISSAO"
	BaseDateEntrada   BaseDate = "ENTRADA"
	BaseDateDigitacao BaseDate = "DIGITACAO"
)

type DuePaymentType string

const (
	DuePaymentSemanal      DuePaymentType = "SEMANAL"
	DuePaymentMensal       DuePaymentType = "MENSAL"
	DuePaymentNaoInformado DuePaymentType = "NAO_INFORMADO"
)

type DueRounding string

const (
	RoundingPosterga DueRounding = "POSTERGA"
	RoundingAntecipa DueRounding = "ANTECIPA"
	RoundingUtil     DueRounding = "UTIL"
	RoundingFixo     DueRounding = "FIXO"
)

// agriRegRe validates the Registro M.A. format AA-99999-9.
var agriRegRe = regexp.MustCompile(`^[A-Z]{2}-\d{5}-\d$`)

// ─── Supplier Type ─────────────────────────────────────────────────────────────

type SupplierType struct {
	ID          int64
	Code        int64
	Description string
	Kind        SupplierKind
	IsActive    bool
	CreatedAt   time.Time
}

func NewSupplierType(code int64, description string, kind SupplierKind) (*SupplierType, error) {
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if kind == "" {
		kind = KindNormal
	}
	return &SupplierType{
		Code:        code,
		Description: description,
		Kind:        kind,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}, nil
}

// ─── Supplier Contact Type ──────────────────────────────────────────────────────

type SupplierContactType struct {
	ID          int64
	Code        int64
	Description string
	IsActive    bool
	CreatedAt   time.Time
}

func NewSupplierContactType(code int64, description string) (*SupplierContactType, error) {
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	return &SupplierContactType{
		Code:        code,
		Description: description,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}, nil
}

// ─── Supplier ───────────────────────────────────────────────────────────────────

type Supplier struct {
	ID                              int64
	Code                            int64
	CorporateCode                   *int64
	IsActive                        bool
	IsRepresentative                bool
	IsCustomer                      bool
	Name                            string
	TradeName                       *string
	PersonType                      PersonType
	DocumentType                    DocumentType
	DocumentNumber                  string
	StateRegistration               *string
	MunicipalRegistration           *string
	SupplierTypeID                  *int64
	PaymentConditionID              *int64
	CarrierID                       *int64
	RegionID                        *int64
	FreightType                     FreightType
	RegisterDate                    time.Time
	ViticolaObligation              ViticolaObligation
	GLNCode                         *string
	AgricultureMinistryRegistration *string
	ICMSContributor                 ICMSContributor
	IsMEI                           bool
	TrackingPlatform                TrackingPlatform
	Homologated                     bool
	LastSefazQuery                  *time.Time
	BillingReceiptStatus            *string
	LastSefazUpdate                 *time.Time
	SefazUpdateUser                 *string
	Blocked                         bool
	BlockReason                     *string
	CreatedAt                       time.Time
	CreatedBy                       uuid.UUID
	UpdatedAt                       time.Time

	Addresses []*SupplierAddress
	Phones    []*SupplierPhone
	Emails    []*SupplierEmail
	DueDates  []*SupplierDueDate
	Contacts  []*SupplierContact
}

// SupplierInput carries the optional/typed fields needed to build a Supplier
// alongside the kind of its supplier type (for the IE rule).
type SupplierInput struct {
	Name           string
	TradeName      *string
	PersonType     PersonType
	DocumentType   DocumentType
	DocumentNumber string
	// TypeKind is the kind of the referenced supplier_type, used to decide
	// whether the state registration is mandatory. Empty means NORMAL.
	TypeKind                        SupplierKind
	StateRegistration               *string
	IsMEI                           bool
	AgricultureMinistryRegistration *string
}

func NewSupplier(code int64, in SupplierInput, createdBy uuid.UUID) (*Supplier, error) {
	if code == 0 {
		return nil, fmt.Errorf("code is required")
	}
	if in.Name == "" {
		return nil, fmt.Errorf("name (razão social) is required")
	}
	if in.PersonType == "" {
		in.PersonType = PersonJuridica
	}
	if in.DocumentType == "" {
		if in.PersonType == PersonFisica {
			in.DocumentType = DocumentCPF
		} else {
			in.DocumentType = DocumentCNPJ
		}
	}
	if in.DocumentNumber == "" {
		return nil, fmt.Errorf("document_number (CNPJ/CPF) is required")
	}
	// Validate the check digits for Brazilian documents (ESTRANGEIRO/ISENTO skip).
	switch in.DocumentType {
	case DocumentCNPJ:
		if !validation.ValidateCNPJ(in.DocumentNumber) {
			return nil, fmt.Errorf("CNPJ inválido")
		}
	case DocumentCPF:
		if !validation.ValidateCPF(in.DocumentNumber) {
			return nil, fmt.Errorf("CPF inválido")
		}
	}
	// MEI cannot be set for individuals (Pessoa Física).
	if in.IsMEI && in.PersonType == PersonFisica {
		return nil, fmt.Errorf("microempreendedor individual não pode ser marcado para pessoa física")
	}
	// State registration is mandatory unless the supplier is a carrier/redispatch.
	if in.TypeKind.RequiresStateRegistration() {
		if in.StateRegistration == nil || *in.StateRegistration == "" {
			return nil, fmt.Errorf("inscrição estadual é obrigatória para este tipo de fornecedor")
		}
	}
	if in.AgricultureMinistryRegistration != nil && *in.AgricultureMinistryRegistration != "" {
		if !agriRegRe.MatchString(*in.AgricultureMinistryRegistration) {
			return nil, fmt.Errorf("registro M.A. deve obedecer o formato AA-99999-9")
		}
	}

	now := time.Now()
	return &Supplier{
		Code:                            code,
		IsActive:                        true,
		Name:                            in.Name,
		TradeName:                       in.TradeName,
		PersonType:                      in.PersonType,
		DocumentType:                    in.DocumentType,
		DocumentNumber:                  in.DocumentNumber,
		StateRegistration:               in.StateRegistration,
		FreightType:                     FreightSemFrete,
		RegisterDate:                    now,
		ViticolaObligation:              ViticolaNunca,
		AgricultureMinistryRegistration: in.AgricultureMinistryRegistration,
		ICMSContributor:                 ICMSContribuinte,
		IsMEI:                           in.IsMEI,
		TrackingPlatform:                TrackingNenhum,
		Blocked:                         false,
		CreatedAt:                       now,
		CreatedBy:                       createdBy,
		UpdatedAt:                       now,
	}, nil
}

// ─── Supplier Address ────────────────────────────────────────────────────────

type SupplierAddress struct {
	ID           int64
	SupplierID   int64
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

// ─── Supplier Phone ──────────────────────────────────────────────────────────

type SupplierPhone struct {
	ID         int64
	SupplierID int64
	Number     string
	Ranking    int32
	IsActive   bool
	CreatedAt  time.Time
}

// ─── Supplier Email ──────────────────────────────────────────────────────────

type SupplierEmail struct {
	ID         int64
	SupplierID int64
	Email      string
	Ranking    int32
	IsActive   bool
	CreatedAt  time.Time
}

// ─── Supplier Due Date (Vencimento) ────────────────────────────────────────────

type SupplierDueDate struct {
	ID                 int64
	SupplierID         int64
	Description        string
	Ranking            int32
	BaseDate           BaseDate
	PaymentConditionID *int64
	PaymentType        DuePaymentType
	SubsequentMonth    bool
	Rounding           DueRounding
	ReceiptStartTime   *string
	ReceiptEndTime     *string
	AvgUnloadMinutes   *int32
	CreatedAt          time.Time
}

// ─── Supplier Contact ──────────────────────────────────────────────────────────

type SupplierContact struct {
	ID               int64
	SupplierID       int64
	ContactTypeID    *int64
	Name             string
	Position         *string
	Department       *string
	Ranking          int32
	Observation      *string
	PurchaseOrderTag *string
	IsActive         bool
	CreatedAt        time.Time
	Phones           []*SupplierContactPhone
	Emails           []*SupplierContactEmail
}

type SupplierContactPhone struct {
	ID        int64
	ContactID int64
	Value     string
	Ranking   int32
}

type SupplierContactEmail struct {
	ID        int64
	ContactID int64
	Value     string
	Ranking   int32
}

// ─── Supplier ↔ Enterprise (Pasta Empresas) ────────────────────────────────────

type SupplierEnterprise struct {
	ID                   int64
	SupplierID           int64
	EnterpriseCode       int64
	FinancialAccount     *string
	AppliesIPI           bool
	DefaultInvoiceTypeID *int64
	PurchasePriceTableID *int64
	IsActive             bool
	CreatedAt            time.Time
}

// ─── Supplier Parameters ───────────────────────────────────────────────────────

type SupplierParameters struct {
	ID                        int64
	EnterpriseCode            int64
	DefaultFinancialAccount   *string
	UniqueItemCodePerSupplier bool
	RequiresFinancialAccount  bool
	PurchaseSupplierTypeID    *int64
	CopyObsToPurchaseOrder    bool
	CopyObsToEntryInvoice     bool
	HomologationDefault       bool
	UseStockUOM               bool
	GenericSupplierCode       *int64
	DefaultDueBaseDate        BaseDate
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}
