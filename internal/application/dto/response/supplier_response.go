package response

import (
	"time"

	"github.com/google/uuid"
)

// SupplierTypeResponse is the API representation of a supplier type.
type SupplierTypeResponse struct {
	ID          int64     `json:"id"`
	Code        int64     `json:"code"`
	Description string    `json:"description"`
	Kind        string    `json:"kind"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// SupplierContactTypeResponse is the API representation of a contact type.
type SupplierContactTypeResponse struct {
	ID          int64     `json:"id"`
	Code        int64     `json:"code"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// SupplierResponse is the API representation of a supplier.
type SupplierResponse struct {
	ID                              int64                     `json:"id"`
	Code                            int64                     `json:"code"`
	CorporateCode                   *int64                    `json:"corporate_code,omitempty"`
	IsActive                        bool                      `json:"is_active"`
	IsRepresentative                bool                      `json:"is_representative"`
	IsCustomer                      bool                      `json:"is_customer"`
	Name                            string                    `json:"name"`
	TradeName                       *string                   `json:"trade_name,omitempty"`
	PersonType                      string                    `json:"person_type"`
	DocumentType                    string                    `json:"document_type"`
	DocumentNumber                  string                    `json:"document_number"`
	StateRegistration               *string                   `json:"state_registration,omitempty"`
	MunicipalRegistration           *string                   `json:"municipal_registration,omitempty"`
	SupplierTypeID                  *int64                    `json:"supplier_type_id,omitempty"`
	PaymentConditionID              *int64                    `json:"payment_condition_id,omitempty"`
	CarrierID                       *int64                    `json:"carrier_id,omitempty"`
	RegionID                        *int64                    `json:"region_id,omitempty"`
	FreightType                     string                    `json:"freight_type"`
	RegisterDate                    time.Time                 `json:"register_date"`
	ViticolaObligation              string                    `json:"viticola_obligation"`
	GLNCode                         *string                   `json:"gln_code,omitempty"`
	AgricultureMinistryRegistration *string                   `json:"agriculture_ministry_registration,omitempty"`
	ICMSContributor                 string                    `json:"icms_contributor"`
	IsMEI                           bool                      `json:"is_mei"`
	TrackingPlatform                string                    `json:"tracking_platform"`
	Homologated                     bool                      `json:"homologated"`
	LastSefazQuery                  *time.Time                `json:"last_sefaz_query,omitempty"`
	BillingReceiptStatus            *string                   `json:"billing_receipt_status,omitempty"`
	LastSefazUpdate                 *time.Time                `json:"last_sefaz_update,omitempty"`
	SefazUpdateUser                 *string                   `json:"sefaz_update_user,omitempty"`
	Blocked                         bool                      `json:"blocked"`
	BlockReason                     *string                   `json:"block_reason,omitempty"`
	CreatedAt                       time.Time                 `json:"created_at"`
	CreatedBy                       uuid.UUID                 `json:"created_by"`
	UpdatedAt                       time.Time                 `json:"updated_at"`
	Addresses                       []SupplierAddressResponse `json:"addresses,omitempty"`
	Phones                          []SupplierPhoneResponse   `json:"phones,omitempty"`
	Emails                          []SupplierEmailResponse   `json:"emails,omitempty"`
	DueDates                        []SupplierDueDateResponse `json:"due_dates,omitempty"`
	Contacts                        []SupplierContactResponse `json:"contacts,omitempty"`
}

// SupplierAddressResponse is the API representation of a supplier address.
type SupplierAddressResponse struct {
	ID           int64     `json:"id"`
	SupplierID   int64     `json:"supplier_id"`
	AddressType  string    `json:"address_type"`
	ZipCode      *string   `json:"zip_code,omitempty"`
	Street       *string   `json:"street,omitempty"`
	Number       *string   `json:"number,omitempty"`
	Complement   *string   `json:"complement,omitempty"`
	Neighborhood *string   `json:"neighborhood,omitempty"`
	City         *string   `json:"city,omitempty"`
	UF           *string   `json:"uf,omitempty"`
	Country      string    `json:"country"`
	IsDefault    bool      `json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
}

// SupplierPhoneResponse is the API representation of a supplier phone.
type SupplierPhoneResponse struct {
	ID         int64     `json:"id"`
	SupplierID int64     `json:"supplier_id"`
	Number     string    `json:"number"`
	Ranking    int32     `json:"ranking"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// SupplierEmailResponse is the API representation of a supplier email.
type SupplierEmailResponse struct {
	ID         int64     `json:"id"`
	SupplierID int64     `json:"supplier_id"`
	Email      string    `json:"email"`
	Ranking    int32     `json:"ranking"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// SupplierDueDateResponse is the API representation of a supplier due date config.
type SupplierDueDateResponse struct {
	ID                 int64     `json:"id"`
	SupplierID         int64     `json:"supplier_id"`
	Description        string    `json:"description"`
	Ranking            int32     `json:"ranking"`
	BaseDate           string    `json:"base_date"`
	PaymentConditionID *int64    `json:"payment_condition_id,omitempty"`
	PaymentType        string    `json:"payment_type"`
	SubsequentMonth    bool      `json:"subsequent_month"`
	Rounding           string    `json:"rounding"`
	ReceiptStartTime   *string   `json:"receipt_start_time,omitempty"`
	ReceiptEndTime     *string   `json:"receipt_end_time,omitempty"`
	AvgUnloadMinutes   *int32    `json:"avg_unload_minutes,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// SupplierContactResponse is the API representation of a supplier contact.
type SupplierContactResponse struct {
	ID               int64                          `json:"id"`
	SupplierID       int64                          `json:"supplier_id"`
	ContactTypeID    *int64                         `json:"contact_type_id,omitempty"`
	Name             string                         `json:"name"`
	Position         *string                        `json:"position,omitempty"`
	Department       *string                        `json:"department,omitempty"`
	Ranking          int32                          `json:"ranking"`
	Observation      *string                        `json:"observation,omitempty"`
	PurchaseOrderTag *string                        `json:"purchase_order_tag,omitempty"`
	IsActive         bool                           `json:"is_active"`
	CreatedAt        time.Time                      `json:"created_at"`
	Phones           []SupplierContactPhoneResponse `json:"phones,omitempty"`
	Emails           []SupplierContactEmailResponse `json:"emails,omitempty"`
}

// SupplierContactPhoneResponse is the API representation of a contact phone.
type SupplierContactPhoneResponse struct {
	ID        int64  `json:"id"`
	ContactID int64  `json:"contact_id"`
	Value     string `json:"value"`
	Ranking   int32  `json:"ranking"`
}

// SupplierContactEmailResponse is the API representation of a contact email.
type SupplierContactEmailResponse struct {
	ID        int64  `json:"id"`
	ContactID int64  `json:"contact_id"`
	Value     string `json:"value"`
	Ranking   int32  `json:"ranking"`
}

// SupplierEnterpriseResponse is the API representation of a supplier↔enterprise link.
type SupplierEnterpriseResponse struct {
	ID                   int64     `json:"id"`
	SupplierID           int64     `json:"supplier_id"`
	EnterpriseCode       int64     `json:"enterprise_code"`
	FinancialAccount     *string   `json:"financial_account,omitempty"`
	AppliesIPI           bool      `json:"applies_ipi"`
	DefaultInvoiceTypeID *int64    `json:"default_invoice_type_id,omitempty"`
	PurchasePriceTableID *int64    `json:"purchase_price_table_id,omitempty"`
	IsActive             bool      `json:"is_active"`
	CreatedAt            time.Time `json:"created_at"`
}

// SupplierParametersResponse is the API representation of supplier parameters.
type SupplierParametersResponse struct {
	ID                        int64     `json:"id"`
	EnterpriseCode            int64     `json:"enterprise_code"`
	DefaultFinancialAccount   *string   `json:"default_financial_account,omitempty"`
	UniqueItemCodePerSupplier bool      `json:"unique_item_code_per_supplier"`
	RequiresFinancialAccount  bool      `json:"requires_financial_account"`
	PurchaseSupplierTypeID    *int64    `json:"purchase_supplier_type_id,omitempty"`
	CopyObsToPurchaseOrder    bool      `json:"copy_obs_to_purchase_order"`
	CopyObsToEntryInvoice     bool      `json:"copy_obs_to_entry_invoice"`
	HomologationDefault       bool      `json:"homologation_default"`
	UseStockUOM               bool      `json:"use_stock_uom"`
	GenericSupplierCode       *int64    `json:"generic_supplier_code,omitempty"`
	DefaultDueBaseDate        string    `json:"default_due_base_date"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}
