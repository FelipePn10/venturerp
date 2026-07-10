package request

import "github.com/google/uuid"

// ─── serials (physical instances of a tool master) ───────────────────────────

type CreateToolSerialDTO struct {
	ToolID       int64     `json:"tool_id"`
	SerialNumber string    `json:"serial_number"`
	Status       string    `json:"status"` // ATIVA | MANUTENCAO | INATIVA | BAIXADA (default ATIVA)
	Location     string    `json:"location"`
	Notes        string    `json:"notes"`
	CreatedBy    uuid.UUID `json:"created_by"`
}

type UpdateToolSerialDTO struct {
	ID           int64  `json:"id"`
	SerialNumber string `json:"serial_number"`
	Status       string `json:"status"`
	Location     string `json:"location"`
	Notes        string `json:"notes"`
}

// ─── tool production sheet ────────────────────────────────────────────────────

// AssignToolSerialDTO binds a serial to a production-order operation/tool.
type AssignToolSerialDTO struct {
	OperationID int64     `json:"operation_id"` // production_order_operations.id
	ToolID      int64     `json:"tool_id"`
	SerialID    int64     `json:"serial_id"`
	AssignedBy  uuid.UUID `json:"-"` // filled from the authenticated user
}

// SubstituteToolSerialDTO replaces the serial already assigned to an
// operation/tool, recording the reason in the audit trail.
type SubstituteToolSerialDTO struct {
	OperationID   int64     `json:"operation_id"`
	ToolID        int64     `json:"tool_id"`
	NewSerialID   int64     `json:"new_serial_id"`
	Reason        string    `json:"reason"`
	SubstitutedBy uuid.UUID `json:"-"` // filled from the authenticated user
}
