package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ItemPreferredSupplier links an item to a supplier with a preference ranking,
// and carries the item's code/description/UOM at that supplier.
type ItemPreferredSupplier struct {
	ID                  int64
	ItemCode            int64
	SupplierCode        int64
	Ranking             int32
	SupplierItemCode    *string
	SupplierDescription *string
	UOM                 *string
	LeadTimeDays        int32
	IsActive            bool
	CreatedAt           time.Time
	CreatedBy           uuid.UUID
}

func NewItemPreferredSupplier(itemCode, supplierCode int64, ranking int32, createdBy uuid.UUID) (*ItemPreferredSupplier, error) {
	if itemCode == 0 || supplierCode == 0 {
		return nil, fmt.Errorf("item_code and supplier_code are required")
	}
	if ranking <= 0 {
		ranking = 1
	}
	return &ItemPreferredSupplier{
		ItemCode:     itemCode,
		SupplierCode: supplierCode,
		Ranking:      ranking,
		IsActive:     true,
		CreatedAt:    time.Now(),
		CreatedBy:    createdBy,
	}, nil
}
