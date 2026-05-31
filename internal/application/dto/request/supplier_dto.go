package request

import "github.com/google/uuid"

// ─── Supplier Types ─────────────────────────────────────────────────────────

type CreateSupplierTypeDTO struct {
	Description string `json:"description"`
	Kind        string `json:"kind"` // NORMAL | TRANSPORTADORA | TRANSP_REDESP | REDESPACHO
}

type UpdateSupplierTypeDTO struct {
	Code        int64  `json:"code"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
	IsActive    bool   `json:"is_active"`
}

// ─── Supplier Contact Types ───────────────────────────────────────────────────

type CreateSupplierContactTypeDTO struct {
	Description string `json:"description"`
}

type UpdateSupplierContactTypeDTO struct {
	Code        int64  `json:"code"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

// ─── Supplier ───────────────────────────────────────────────────────────────

type CreateSupplierDTO struct {
	CorporateCode                   *int64    `json:"corporate_code,omitempty"`
	IsRepresentative                bool      `json:"is_representative"`
	IsCustomer                      bool      `json:"is_customer"`
	Name                            string    `json:"name"`
	TradeName                       *string   `json:"trade_name,omitempty"`
	PersonType                      string    `json:"person_type"`   // JURIDICA | FISICA
	DocumentType                    string    `json:"document_type"` // CNPJ | CPF | ESTRANGEIRO | ISENTO
	DocumentNumber                  string    `json:"document_number"`
	StateRegistration               *string   `json:"state_registration,omitempty"`
	MunicipalRegistration           *string   `json:"municipal_registration,omitempty"`
	SupplierTypeCode                *int64    `json:"supplier_type_code,omitempty"`
	PaymentConditionID              *int64    `json:"payment_condition_id,omitempty"`
	CarrierID                       *int64    `json:"carrier_id,omitempty"`
	RegionID                        *int64    `json:"region_id,omitempty"`
	FreightType                     string    `json:"freight_type,omitempty"`
	ViticolaObligation              string    `json:"viticola_obligation,omitempty"`
	GLNCode                         *string   `json:"gln_code,omitempty"`
	AgricultureMinistryRegistration *string   `json:"agriculture_ministry_registration,omitempty"`
	ICMSContributor                 string    `json:"icms_contributor,omitempty"`
	IsMEI                           bool      `json:"is_mei"`
	TrackingPlatform                string    `json:"tracking_platform,omitempty"`
	Homologated                     bool      `json:"homologated"`
	CreatedBy                       uuid.UUID `json:"created_by"`
}

type UpdateSupplierDTO struct {
	Code                            int64   `json:"code"`
	CorporateCode                   *int64  `json:"corporate_code,omitempty"`
	IsActive                        bool    `json:"is_active"`
	IsRepresentative                bool    `json:"is_representative"`
	IsCustomer                      bool    `json:"is_customer"`
	Name                            string  `json:"name"`
	TradeName                       *string `json:"trade_name,omitempty"`
	PersonType                      string  `json:"person_type"`
	DocumentType                    string  `json:"document_type"`
	DocumentNumber                  string  `json:"document_number"`
	StateRegistration               *string `json:"state_registration,omitempty"`
	MunicipalRegistration           *string `json:"municipal_registration,omitempty"`
	SupplierTypeCode                *int64  `json:"supplier_type_code,omitempty"`
	PaymentConditionID              *int64  `json:"payment_condition_id,omitempty"`
	CarrierID                       *int64  `json:"carrier_id,omitempty"`
	RegionID                        *int64  `json:"region_id,omitempty"`
	FreightType                     string  `json:"freight_type,omitempty"`
	ViticolaObligation              string  `json:"viticola_obligation,omitempty"`
	GLNCode                         *string `json:"gln_code,omitempty"`
	AgricultureMinistryRegistration *string `json:"agriculture_ministry_registration,omitempty"`
	ICMSContributor                 string  `json:"icms_contributor,omitempty"`
	IsMEI                           bool    `json:"is_mei"`
	TrackingPlatform                string  `json:"tracking_platform,omitempty"`
	Homologated                     bool    `json:"homologated"`
}

type BlockSupplierDTO struct {
	Code   int64  `json:"code"`
	Reason string `json:"reason"`
}

// ─── Supplier Address ──────────────────────────────────────────────────────

type AddSupplierAddressDTO struct {
	SupplierCode int64   `json:"supplier_code"`
	AddressType  string  `json:"address_type"`
	ZipCode      *string `json:"zip_code,omitempty"`
	Street       *string `json:"street,omitempty"`
	Number       *string `json:"number,omitempty"`
	Complement   *string `json:"complement,omitempty"`
	Neighborhood *string `json:"neighborhood,omitempty"`
	City         *string `json:"city,omitempty"`
	UF           *string `json:"uf,omitempty"`
	Country      string  `json:"country,omitempty"`
	IsDefault    bool    `json:"is_default"`
}

// ─── Supplier Phone / Email ─────────────────────────────────────────────────

type AddSupplierPhoneDTO struct {
	SupplierCode int64  `json:"supplier_code"`
	Number       string `json:"number"`
	Ranking      int32  `json:"ranking"`
}

type AddSupplierEmailDTO struct {
	SupplierCode int64  `json:"supplier_code"`
	Email        string `json:"email"`
	Ranking      int32  `json:"ranking"`
}

// ─── Supplier Due Date (Vencimento) ─────────────────────────────────────────

type AddSupplierDueDateDTO struct {
	SupplierCode       int64   `json:"supplier_code"`
	Description        string  `json:"description"`
	Ranking            int32   `json:"ranking"`
	BaseDate           string  `json:"base_date,omitempty"`
	PaymentConditionID *int64  `json:"payment_condition_id,omitempty"`
	PaymentType        string  `json:"payment_type,omitempty"`
	SubsequentMonth    bool    `json:"subsequent_month"`
	Rounding           string  `json:"rounding,omitempty"`
	ReceiptStartTime   *string `json:"receipt_start_time,omitempty"`
	ReceiptEndTime     *string `json:"receipt_end_time,omitempty"`
	AvgUnloadMinutes   *int32  `json:"avg_unload_minutes,omitempty"`
}

// ─── Supplier Contact ──────────────────────────────────────────────────────

type AddSupplierContactDTO struct {
	SupplierCode     int64   `json:"supplier_code"`
	ContactTypeID    *int64  `json:"contact_type_id,omitempty"`
	Name             string  `json:"name"`
	Position         *string `json:"position,omitempty"`
	Department       *string `json:"department,omitempty"`
	Ranking          int32   `json:"ranking"`
	Observation      *string `json:"observation,omitempty"`
	PurchaseOrderTag *string `json:"purchase_order_tag,omitempty"`
}

type AddSupplierContactPhoneDTO struct {
	ContactID int64  `json:"contact_id"`
	Value     string `json:"value"`
	Ranking   int32  `json:"ranking"`
}

type AddSupplierContactEmailDTO struct {
	ContactID int64  `json:"contact_id"`
	Value     string `json:"value"`
	Ranking   int32  `json:"ranking"`
}

// ─── Supplier ↔ Enterprise ─────────────────────────────────────────────────

type AddSupplierEnterpriseDTO struct {
	SupplierCode         int64   `json:"supplier_code"`
	EnterpriseCode       int64   `json:"enterprise_code"`
	FinancialAccount     *string `json:"financial_account,omitempty"`
	AppliesIPI           bool    `json:"applies_ipi"`
	DefaultInvoiceTypeID *int64  `json:"default_invoice_type_id,omitempty"`
	PurchasePriceTableID *int64  `json:"purchase_price_table_id,omitempty"`
}

type UpdateSupplierEnterpriseDTO struct {
	ID                   int64   `json:"id"`
	FinancialAccount     *string `json:"financial_account,omitempty"`
	AppliesIPI           bool    `json:"applies_ipi"`
	DefaultInvoiceTypeID *int64  `json:"default_invoice_type_id,omitempty"`
	PurchasePriceTableID *int64  `json:"purchase_price_table_id,omitempty"`
	IsActive             bool    `json:"is_active"`
}

// ─── Supplier Parameters ───────────────────────────────────────────────────

type UpsertSupplierParametersDTO struct {
	EnterpriseCode            int64   `json:"enterprise_code"`
	DefaultFinancialAccount   *string `json:"default_financial_account,omitempty"`
	UniqueItemCodePerSupplier bool    `json:"unique_item_code_per_supplier"`
	RequiresFinancialAccount  bool    `json:"requires_financial_account"`
	PurchaseSupplierTypeID    *int64  `json:"purchase_supplier_type_id,omitempty"`
	CopyObsToPurchaseOrder    bool    `json:"copy_obs_to_purchase_order"`
	CopyObsToEntryInvoice     bool    `json:"copy_obs_to_entry_invoice"`
	HomologationDefault       bool    `json:"homologation_default"`
	UseStockUOM               bool    `json:"use_stock_uom"`
	GenericSupplierCode       *int64  `json:"generic_supplier_code,omitempty"`
	DefaultDueBaseDate        string  `json:"default_due_base_date,omitempty"`
}
