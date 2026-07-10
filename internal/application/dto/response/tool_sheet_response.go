package response

import "time"

// ─── serials (physical instances of a tool master) ───────────────────────────

type ToolSerialResponse struct {
	ID           int64     `json:"id"`
	ToolID       int64     `json:"tool_id"`
	SerialNumber string    `json:"serial_number"`
	Status       string    `json:"status"` // ATIVA | MANUTENCAO | INATIVA | BAIXADA
	LifeUsed     float64   `json:"life_used"`
	Location     string    `json:"location,omitempty"`
	Notes        string    `json:"notes,omitempty"`
	IsActive     bool      `json:"is_active"`
	Available    bool      `json:"available"` // can be selected to run an operation
	CreatedAt    time.Time `json:"created_at"`
	// denormalized (populated on sheet reads)
	ToolCode int64  `json:"tool_code,omitempty"`
	ToolName string `json:"tool_name,omitempty"`
}

// ─── tool production sheet ────────────────────────────────────────────────────

// ToolProductionSheetOrderResponse is the production-order header shown on the
// sheet and used as a list-of-values row when picking the order.
type ToolProductionSheetOrderResponse struct {
	EnterpriseID   int64      `json:"enterprise_id,omitempty"`
	EnterpriseName string     `json:"enterprise_name,omitempty"`
	OrderID        int64      `json:"order_id"`
	OrderNumber    int64      `json:"order_number"`
	Type           string     `json:"type"`               // OF | OFC | OUTROS (OFC excluded from LOV)
	TypeRaw        string     `json:"type_raw,omitempty"` // underlying planned-order type
	StartDate      *time.Time `json:"start_date,omitempty"`
	EndDate        *time.Time `json:"end_date,omitempty"`
	Quantity       float64    `json:"quantity"`
	ItemCode       int64      `json:"item_code"`
	ItemName       string     `json:"item_name,omitempty"`
	Configured     string     `json:"configured,omitempty"` // item configuration (mask)
	Status         string     `json:"status"`
}

// ToolProductionSheetResponse is the full sheet: order header + operations block.
type ToolProductionSheetResponse struct {
	Header     ToolProductionSheetOrderResponse `json:"header"`
	Operations []SheetOperationResponse         `json:"operations"`
}

// SheetOperationResponse is one operation of the order with its tools.
type SheetOperationResponse struct {
	OperationID   int64                        `json:"operation_id"`
	Sequence      int                          `json:"sequence"`
	OperationCode *int64                       `json:"operation_code,omitempty"`
	OperationName string                       `json:"operation_name"`
	OperationDesc string                       `json:"operation_description,omitempty"`
	ResourceCode  *int64                       `json:"resource_code,omitempty"`
	ResourceName  string                       `json:"resource_name,omitempty"`
	Status        string                       `json:"status"`
	Tools         []SheetOperationToolResponse `json:"tools"`
}

// SheetOperationToolResponse is one tool required by an operation, with the
// serial currently assigned (if any) and the serials available for selection.
type SheetOperationToolResponse struct {
	ToolID      int64   `json:"tool_id"`
	ToolCode    int64   `json:"tool_code"`
	ToolName    string  `json:"tool_name"`
	QtyRequired float64 `json:"qty_required"`

	AssignedSerialID     *int64 `json:"assigned_serial_id,omitempty"`
	AssignedSerialNumber string `json:"assigned_serial_number,omitempty"`
	AssignedSerialStatus string `json:"assigned_serial_status,omitempty"`

	// CanSubstitute is true when a serial is already assigned — the "Substituir"
	// button is only functional in that case.
	CanSubstitute    bool                 `json:"can_substitute"`
	AvailableSerials []ToolSerialResponse `json:"available_serials"`
}

// ToolSerialSubstitutionResponse is one substitution history entry.
type ToolSerialSubstitutionResponse struct {
	ID              int64     `json:"id"`
	OperationID     int64     `json:"operation_id"`
	ToolID          int64     `json:"tool_id"`
	ToolCode        int64     `json:"tool_code"`
	ToolName        string    `json:"tool_name"`
	OldSerialID     *int64    `json:"old_serial_id,omitempty"`
	OldSerialNumber string    `json:"old_serial_number,omitempty"`
	NewSerialID     int64     `json:"new_serial_id"`
	NewSerialNumber string    `json:"new_serial_number"`
	Reason          string    `json:"reason,omitempty"`
	SubstitutedAt   time.Time `json:"substituted_at"`
}
