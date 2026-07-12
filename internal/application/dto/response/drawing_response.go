package response

import "time"

type DrawingResponse struct {
	ID           int64                     `json:"id"`
	Code         string                    `json:"code"`
	Digit        string                    `json:"digit"`
	Format       string                    `json:"format"`
	Model        string                    `json:"model,omitempty"`
	ItemCode     *int64                    `json:"item_code,omitempty"`
	Description  string                    `json:"description,omitempty"`
	UOM          string                    `json:"uom,omitempty"`
	Weight       *float64                  `json:"weight,omitempty"`
	MaterialSpec string                    `json:"material_spec,omitempty"`
	CreationDate *time.Time                `json:"creation_date,omitempty"`
	IsActive     bool                      `json:"is_active"`
	Revisions    []DrawingRevisionResponse `json:"revisions,omitempty"`
}

type DrawingRevisionResponse struct {
	ID            int64                         `json:"id"`
	DrawingID     int64                         `json:"drawing_id"`
	Revision      string                        `json:"revision"`
	CompositeCode string                        `json:"composite_code"` // Desenho(20)+Dígito+Formato+Revisão
	StartDate     *time.Time                    `json:"start_date,omitempty"`
	EndDate       *time.Time                    `json:"end_date,omitempty"`
	MaterialSpec  string                        `json:"material_spec,omitempty"`
	Reason        string                        `json:"reason,omitempty"`
	ApprovedBy    string                        `json:"approved_by,omitempty"`
	ApprovalDate  *time.Time                    `json:"approval_date,omitempty"`
	IsCurrent     bool                          `json:"is_current"`
	Distributions []DrawingDistributionResponse `json:"distributions,omitempty"`
}

type DrawingDistributionResponse struct {
	ID            int64      `json:"id"`
	RevisionID    int64      `json:"revision_id"`
	Recipient     string     `json:"recipient"`
	DistributedAt *time.Time `json:"distributed_at,omitempty"`
	Notes         string     `json:"notes,omitempty"`
}

type DrawingCharacteristicResponse struct {
	ID               int64  `json:"id"`
	DrawingID        int64  `json:"drawing_id"`
	CharacteristicID int64  `json:"characteristic_id"`
	Operator         string `json:"operator"`
	VariableID       *int64 `json:"variable_id,omitempty"`
}

type ItemEngineeringDrawingResponse struct {
	ItemCode    int64  `json:"item_code"`
	Mask        string `json:"mask,omitempty"`
	DrawingCode string `json:"drawing_code"`
}

type DrawingManufacturingParametersResponse struct {
	Parameter8ReplicateDrawingRevision bool `json:"parameter_8_replicate_drawing_revision"`
}
