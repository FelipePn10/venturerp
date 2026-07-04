package request

import (
	"time"

	"github.com/google/uuid"
)

type CreateBomHeaderDTO struct {
	ItemCode  int64      `json:"item_code"`
	Mask      *string    `json:"mask,omitempty"`
	BomType   string     `json:"bom_type"` // EBOM | MBOM (default MBOM)
	ValidFrom *time.Time `json:"valid_from,omitempty"`
	CreatedBy uuid.UUID  `json:"created_by"`
}

type UpdateBomHeaderStatusDTO struct {
	ID     int64  `json:"id"`
	Status string `json:"status"` // DRAFT | APPROVED | OBSOLETE
}
