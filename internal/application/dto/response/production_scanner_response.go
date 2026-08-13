package response

import "time"

type ProductionScanTokenResponse struct {
	Token             string     `json:"token"`
	BarcodeValue      string     `json:"barcode_value"`
	ProductionOrderID int64      `json:"production_order_id"`
	OperationID       *int64     `json:"operation_id,omitempty"`
	ValidUntil        *time.Time `json:"valid_until,omitempty"`
}
