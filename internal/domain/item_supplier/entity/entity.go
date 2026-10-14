package entity

import (
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
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
		return nil, domainerr.NewValidation("informe a empresa, o item e o fornecedor")
	}
	if ranking <= 0 {
		ranking = 1
	}
	now := time.Now()
	return &ItemPreferredSupplier{EnterpriseID: enterpriseID, ItemCode: itemCode, SupplierCode: supplierCode, Mask: strings.TrimSpace(mask), Ranking: ranking, PackageQuantity: decimal.Zero, IsActive: true, CreatedAt: now, CreatedBy: createdBy, UpdatedAt: now}, nil
}
func (s *ItemPreferredSupplier) Validate() error {
	if s.PackageQuantity.IsNegative() {
		return domainerr.NewValidation("a quantidade por embalagem não pode ser negativa")
	}
	if s.LeadTimeDays < 0 {
		return domainerr.NewValidation("o lead time em dias não pode ser negativo")
	}
	if s.IsPreferred {
		s.Ranking = 1
	}
	if s.DirectBilling && !s.IsPreferred {
		return domainerr.NewValidation("faturamento direto exige fornecedor preferencial")
	}
	if s.ThirdPartyOrder && !s.DirectBilling {
		return domainerr.NewValidation("ordem de terceiro exige faturamento direto")
	}
	if s.XMLUOM != nil && s.UOM == nil {
		return domainerr.NewValidation("a unidade do XML exige a unidade do fornecedor")
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
	if e <= 0 || link <= 0 || on.IsZero() || (status != "PENDENTE" && status != "APROVADO" && status != "REJEITADO" && status != "EXPIRADO") {
		return nil, domainerr.NewValidation("laudo de qualidade inválido")
	}
	return &QualityReport{EnterpriseID: e, ItemSupplierID: link, RegisteredOn: on, Status: status, CreatedBy: by}, nil
}
