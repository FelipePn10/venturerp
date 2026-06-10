package response

import (
	"time"

	"github.com/google/uuid"
)

// StateGroupResponse is the API representation of a state group (grupo de estado).
type StateGroupResponse struct {
	ID          int64     `json:"id"`
	Code        int64     `json:"code"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   uuid.UUID `json:"created_by"`
	UFs         []string  `json:"ufs,omitempty"`
}

// EntryOperationTypeResponse is the API representation of an entry operation type.
type EntryOperationTypeResponse struct {
	ID                 int64     `json:"id"`
	Code               int64     `json:"code"`
	Description        string    `json:"description"`
	InvoiceTypeCode    *int64    `json:"invoice_type_code,omitempty"`
	NatureOperation    string    `json:"nature_operation"`
	ClassificationType *string   `json:"classification_type,omitempty"`
	ClassificationCode *string   `json:"classification_code,omitempty"`
	StateGroupCode     *int64    `json:"state_group_code,omitempty"`
	SupplierTypeCode   *int64    `json:"supplier_type_code,omitempty"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
	CreatedBy          uuid.UUID `json:"created_by"`
}
