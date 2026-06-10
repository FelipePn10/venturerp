package response

import (
	"time"

	"github.com/google/uuid"
)

// ComponentResponse is the API representation of a component.
type ComponentResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	GroupCode string    `json:"group_code"`
	Code      string    `json:"code"`
	Warehouse int64     `json:"warehouse"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}
