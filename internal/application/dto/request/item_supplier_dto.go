package request

import "github.com/google/uuid"

type UpsertItemPreferredSupplierDTO struct {
	ItemCode            int64     `json:"item_code"`
	SupplierCode        int64     `json:"supplier_code"`
	Ranking             int32     `json:"ranking"`
	SupplierItemCode    *string   `json:"supplier_item_code,omitempty"`
	SupplierDescription *string   `json:"supplier_description,omitempty"`
	UOM                 *string   `json:"uom,omitempty"`
	LeadTimeDays        int32     `json:"lead_time_days,omitempty"`
	CreatedBy           uuid.UUID `json:"created_by"`
}
