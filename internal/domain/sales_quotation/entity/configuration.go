package entity

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const MaxAttachmentSize int64 = 10 * 1024 * 1024

type Parameters struct {
	EnterpriseCode              int64           `json:"enterprise_code"`
	PurchaseOrderPrompt         string          `json:"purchase_order_prompt"`
	DeliveryAuthorizationPrompt string          `json:"delivery_authorization_prompt"`
	FinalConsumerCustomerCode   *int64          `json:"final_consumer_customer_code,omitempty"`
	AllowServiceItemsNFCe       bool            `json:"allow_service_items_nfce"`
	DefaultNFCe                 bool            `json:"default_nfce"`
	MinimumCIFFreight           decimal.Decimal `json:"minimum_cif_freight"`
	AddRedeliveryToFreight      bool            `json:"add_redelivery_to_freight"`
}

func DefaultParameters(enterpriseCode int64) *Parameters {
	return &Parameters{
		EnterpriseCode:              enterpriseCode,
		PurchaseOrderPrompt:         "Ordem de Compra",
		DeliveryAuthorizationPrompt: "Autorização de Entr.",
		MinimumCIFFreight:           decimal.Zero,
	}
}

func (p *Parameters) Validate() error {
	if p.EnterpriseCode <= 0 {
		return errors.New("enterprise is required")
	}
	if strings.TrimSpace(p.PurchaseOrderPrompt) == "" || strings.TrimSpace(p.DeliveryAuthorizationPrompt) == "" {
		return errors.New("quotation prompts are required")
	}
	if p.MinimumCIFFreight.IsNegative() {
		return errors.New("minimum CIF freight cannot be negative")
	}
	return nil
}

type CommissionPattern struct {
	ID             int64           `json:"id"`
	EnterpriseCode int64           `json:"enterprise_code"`
	Code           int64           `json:"code"`
	Description    string          `json:"description"`
	CommissionPct  decimal.Decimal `json:"commission_pct"`
	InvoicePct     decimal.Decimal `json:"invoice_pct"`
	PaymentPct     decimal.Decimal `json:"payment_pct"`
	IsActive       bool            `json:"is_active"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (p *CommissionPattern) Validate() error {
	if p.Code <= 0 || strings.TrimSpace(p.Description) == "" {
		return errors.New("commission pattern code and description are required")
	}
	if p.CommissionPct.IsNegative() || p.InvoicePct.IsNegative() || p.PaymentPct.IsNegative() {
		return errors.New("commission percentages cannot be negative")
	}
	if !p.InvoicePct.Add(p.PaymentPct).Equal(p.CommissionPct) {
		return errors.New("invoice_pct plus payment_pct must equal commission_pct")
	}
	return nil
}

type CancellationReason struct {
	ID                int64     `json:"id"`
	EnterpriseCode    int64     `json:"enterprise_code"`
	Code              int64     `json:"code"`
	Description       string    `json:"description"`
	AllowUncancel     bool      `json:"allow_uncancel"`
	RequireComplement bool      `json:"require_complement"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Event struct {
	ID                     int64      `json:"id"`
	SalesQuotationCode     int64      `json:"sales_quotation_code"`
	SalesQuotationItemCode *int64     `json:"sales_quotation_item_code,omitempty"`
	EventType              string     `json:"event_type"`
	Reason                 string     `json:"reason"`
	Complement             *string    `json:"complement,omitempty"`
	EventDate              time.Time  `json:"event_date"`
	CreatedAt              time.Time  `json:"created_at"`
	CreatedBy              *uuid.UUID `json:"created_by,omitempty"`
}

func (r *CancellationReason) Validate() error {
	if r.Code <= 0 || strings.TrimSpace(r.Description) == "" {
		return errors.New("cancellation reason code and description are required")
	}
	return nil
}

type Attachment struct {
	ID                 int64     `json:"id"`
	SalesQuotationCode int64     `json:"sales_quotation_code"`
	FileName           string    `json:"file_name"`
	ContentType        string    `json:"content_type"`
	StorageKey         string    `json:"storage_key"`
	FileSize           int64     `json:"file_size"`
	UploadedAt         time.Time `json:"uploaded_at"`
	UploadedBy         uuid.UUID `json:"uploaded_by"`
	Content            []byte    `json:"-"`
}

func (a *Attachment) Validate() error {
	if a.SalesQuotationCode <= 0 || strings.TrimSpace(a.FileName) == "" {
		return errors.New("quotation and file name are required")
	}
	if a.FileSize < 0 || a.FileSize > MaxAttachmentSize {
		return errors.New("attachment cannot exceed 10 MB")
	}
	if int64(len(a.Content)) != a.FileSize {
		return errors.New("attachment size does not match content")
	}
	return nil
}

func (q *SalesQuotation) DocumentActionAllowed(action string) bool {
	if q.DAVGeneratedAt == nil {
		return true
	}
	return action == "DAV_REPORT"
}
