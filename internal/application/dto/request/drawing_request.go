package request

import "github.com/google/uuid"

type DrawingDTO struct {
	ID           int64     `json:"id"`
	Code         string    `json:"code"`
	Digit        string    `json:"digit"`
	Format       string    `json:"format"`
	Model        string    `json:"model"`
	ItemCode     *int64    `json:"item_code"`
	Description  string    `json:"description"`
	UOM          string    `json:"uom"`
	Weight       *float64  `json:"weight"`
	MaterialSpec string    `json:"material_spec"`
	CreationDate string    `json:"creation_date"` // YYYY-MM-DD
	CreatedBy    uuid.UUID `json:"-"`
}

type DrawingRevisionDTO struct {
	ID           int64     `json:"id"`
	Revision     string    `json:"revision"`
	StartDate    string    `json:"start_date"`
	EndDate      string    `json:"end_date"`
	MaterialSpec string    `json:"material_spec"`
	Reason       string    `json:"reason"`
	ApprovedBy   string    `json:"approved_by"`
	ApprovalDate string    `json:"approval_date"`
	IsCurrent    bool      `json:"is_current"`
	UpdatedBy    uuid.UUID `json:"-"`
}

type DrawingDistributionDTO struct {
	Recipient     string `json:"recipient"`
	DistributedAt string `json:"distributed_at"`
	Notes         string `json:"notes"`
}

type DrawingCharacteristicDTO struct {
	CharacteristicID int64  `json:"characteristic_id"`
	Operator         string `json:"operator"`
	VariableID       *int64 `json:"variable_id"`
}

type MaintainItemDrawingCodeDTO struct {
	ItemCode    int64     `json:"item_code"`
	Mask        string    `json:"mask,omitempty"`
	DrawingCode string    `json:"drawing_code"`
	UpdatedBy   uuid.UUID `json:"-"`
}

type DrawingManufacturingParametersDTO struct {
	ReplicateDrawingRevision bool      `json:"replicate_drawing_revision"`
	UpdatedBy                uuid.UUID `json:"-"`
}
