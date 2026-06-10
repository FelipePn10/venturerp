package response

import (
	"time"

	"github.com/google/uuid"
)

// WarehouseResponse is the API representation of a warehouse.
type WarehouseResponse struct {
	ID                  int32     `json:"id"`
	Code                int       `json:"code"`
	Description         string    `json:"description"`
	Location            string    `json:"location"`
	Type                string    `json:"type"`
	Disposition         bool      `json:"disposition"`
	ReservationsAllowed bool      `json:"reservations_allowed"`
	CreatedBy           uuid.UUID `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
}
