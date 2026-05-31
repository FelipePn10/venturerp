package request

import "github.com/google/uuid"

type CreateStateGroupDTO struct {
	Description string    `json:"description"`
	UFs         []string  `json:"ufs,omitempty"`
	CreatedBy   uuid.UUID `json:"created_by"`
}

type AddStateGroupUFDTO struct {
	StateGroupCode int64  `json:"-"`
	UF             string `json:"uf"`
}

type CreateEntryOperationDTO struct {
	Description        string    `json:"description"`
	NatureOperation    string    `json:"nature_operation"`
	InvoiceTypeCode    *int64    `json:"invoice_type_code,omitempty"`
	ClassificationType *string   `json:"classification_type,omitempty"`
	ClassificationCode *string   `json:"classification_code,omitempty"`
	StateGroupCode     *int64    `json:"state_group_code,omitempty"`
	SupplierTypeCode   *int64    `json:"supplier_type_code,omitempty"`
	CreatedBy          uuid.UUID `json:"created_by"`
}

type UpdateEntryOperationDTO struct {
	Code               int64   `json:"code"`
	Description        string  `json:"description"`
	NatureOperation    string  `json:"nature_operation"`
	InvoiceTypeCode    *int64  `json:"invoice_type_code,omitempty"`
	ClassificationType *string `json:"classification_type,omitempty"`
	ClassificationCode *string `json:"classification_code,omitempty"`
	StateGroupCode     *int64  `json:"state_group_code,omitempty"`
	SupplierTypeCode   *int64  `json:"supplier_type_code,omitempty"`
	IsActive           bool    `json:"is_active"`
}
