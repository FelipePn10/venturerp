package response

import (
	"time"

	"github.com/google/uuid"
)

// ProductResponse is the API representation of a product.
type ProductResponse struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	GroupCode string    `json:"group_code"`
	Name      string    `json:"name"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
