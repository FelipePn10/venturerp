package response

import (
	"time"

	"github.com/google/uuid"
)

// ItemPreferredSupplierResponse is the API representation of an item↔supplier link.
type ItemPreferredSupplierResponse struct {
	ID                  int64     `json:"id"`
	ItemCode            int64     `json:"item_code"`
	SupplierCode        int64     `json:"supplier_code"`
	Ranking             int32     `json:"ranking"`
	SupplierItemCode    *string   `json:"supplier_item_code,omitempty"`
	SupplierDescription *string   `json:"supplier_description,omitempty"`
	UOM                 *string   `json:"uom,omitempty"`
	LeadTimeDays        int32     `json:"lead_time_days"`
	IsActive            bool      `json:"is_active"`
	CreatedAt           time.Time `json:"created_at"`
	CreatedBy           uuid.UUID `json:"created_by"`
}
