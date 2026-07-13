package entity

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ItemPreferredSupplier struct {
	ID                    int64
	EnterpriseID          int64
	ItemCode              int64
	SupplierCode          int64
	Mask                  string
	Ranking               int32
	SupplierItemCode      *string
	SupplierDescription   *string
	UOM                   *string
	XMLUOM                *string
	ConversionFactor      *decimal.Decimal
	PackageQuantity       decimal.Decimal
	IsPreferred           bool
	SupplierUF            *string
	ClassificationID      *int64
	ClassificationDate    *time.Time
	ClassificationGrade   *decimal.Decimal
	DirectBilling         bool
	ThirdPartyOrder       bool
	IgnoreAvgCostAddition bool
	Ecommerce             bool
	Barcode               *string
	Notes                 *string
	ValidUntil            *time.Time
	LeadTimeDays          int32
	IsActive              bool
	CreatedAt             time.Time
	CreatedBy             uuid.UUID
	UpdatedAt             time.Time
}

func NewItemPreferredSupplier(enterpriseID, itemCode, supplierCode int64, mask string, ranking int32, createdBy uuid.UUID) (*ItemPreferredSupplier, error) {
	if enterpriseID <= 0 || itemCode <= 0 || supplierCode <= 0 {
		return nil, fmt.Errorf("enterprise, item_code and supplier_code are required")
	}
	if ranking <= 0 {
		ranking = 1
	}
	now := time.Now()
	return &ItemPreferredSupplier{EnterpriseID: enterpriseID, ItemCode: itemCode, SupplierCode: supplierCode, Mask: strings.TrimSpace(mask), Ranking: ranking, PackageQuantity: decimal.Zero, IsActive: true, CreatedAt: now, CreatedBy: createdBy, UpdatedAt: now}, nil
}
func (s *ItemPreferredSupplier) Validate() error {
	if s.PackageQuantity.IsNegative() {
		return fmt.Errorf("package_quantity must not be negative")
	}
	if s.LeadTimeDays < 0 {
		return fmt.Errorf("lead_time_days must not be negative")
	}
	if s.IsPreferred {
		s.Ranking = 1
	}
	if s.DirectBilling && !s.IsPreferred {
		return fmt.Errorf("direct_billing requires preferred supplier")
	}
	if s.ThirdPartyOrder && !s.DirectBilling {
		return fmt.Errorf("third_party_order requires direct_billing")
	}
	if s.XMLUOM != nil && s.UOM == nil {
		return fmt.Errorf("xml_uom requires supplier uom")
	}
	return nil
}

type QualityReport struct {
	ID             int64
	EnterpriseID   int64
	ItemSupplierID int64
	RegisteredOn   time.Time
	Status         string
	FileName       *string
	ContentType    *string
	Content        []byte
	Notes          *string
	CreatedAt      time.Time
	CreatedBy      uuid.UUID
}

func NewQualityReport(e, link int64, on time.Time, status string, by uuid.UUID) (*QualityReport, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if e <= 0 || link <= 0 || on.IsZero() || (status != "PENDING" && status != "APPROVED" && status != "REJECTED" && status != "EXPIRED") {
		return nil, fmt.Errorf("invalid quality report")
	}
	return &QualityReport{EnterpriseID: e, ItemSupplierID: link, RegisteredOn: on, Status: status, CreatedBy: by}, nil
}
