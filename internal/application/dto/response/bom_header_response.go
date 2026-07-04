package response

import "time"

type BomHeaderResponse struct {
	ID        int64      `json:"id"`
	ItemCode  int64      `json:"item_code"`
	Mask      *string    `json:"mask,omitempty"`
	BomType   string     `json:"bom_type"`
	Version   int32      `json:"version"`
	Status    string     `json:"status"`
	ValidFrom *time.Time `json:"valid_from,omitempty"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
}
