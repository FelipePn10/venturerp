package purchase_order_uc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/shopspring/decimal"
)

const (
	PositionAttended  = "ATTENDED"
	PositionPending   = "PENDING"
	PositionCancelled = "CANCELLED"
)

var ErrAttachmentNotFound = errors.New("purchase order attachment not found")

type PurchaseOrderConsultationFilter struct {
	OrderFrom, OrderTo       *int64
	SupplierFrom, SupplierTo *int64
	RequestTypeCode          *int64
	ItemFrom, ItemTo         *int64
	Position                 string
	AllItems                 bool
	Convert                  bool
	TargetCurrency           string
	BaseDate                 *time.Time
	OnlyKanban               bool
	EmissionFrom, EmissionTo *time.Time
	DeliveryFrom, DeliveryTo *time.Time
	BuyerCode                *int64
	OrderType                string
	ImportFrom, ImportTo     *int64
	Limit, Offset            int
	EnterpriseID             int64
}

type PurchaseOrderConsultationItem struct {
	Code         int64            `json:"code"`
	Sequence     int              `json:"sequence"`
	ItemCode     int64            `json:"item_code"`
	Position     string           `json:"position"`
	RequestedQty decimal.Decimal  `json:"requested_qty"`
	ReceivedQty  decimal.Decimal  `json:"received_qty"`
	CancelledQty decimal.Decimal  `json:"cancelled_qty"`
	UnitPrice    decimal.Decimal  `json:"unit_price"`
	Gross        decimal.Decimal  `json:"gross"`
	Discount     decimal.Decimal  `json:"discount"`
	Additions    decimal.Decimal  `json:"additions"`
	Net          decimal.Decimal  `json:"net"`
	Total        decimal.Decimal  `json:"total"`
	IPIBase      *decimal.Decimal `json:"ipi_base,omitempty"`
	IPIRate      *decimal.Decimal `json:"ipi_rate,omitempty"`
	IPIValue     *decimal.Decimal `json:"ipi_value,omitempty"`
	ICMSBase     *decimal.Decimal `json:"icms_base,omitempty"`
	ICMSRate     *decimal.Decimal `json:"icms_rate,omitempty"`
	ICMSValue    *decimal.Decimal `json:"icms_value,omitempty"`
	ICMSSTBase   *decimal.Decimal `json:"icms_st_base,omitempty"`
	ICMSSTRate   *decimal.Decimal `json:"icms_st_rate,omitempty"`
	ICMSSTValue  *decimal.Decimal `json:"icms_st_value,omitempty"`
}

type PurchaseOrderAttachment struct {
	ID          int64     `json:"id"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type"`
	FileSize    int64     `json:"file_size"`
	CreatedAt   time.Time `json:"created_at"`
}

type PurchaseOrderAttachmentFile struct {
	PurchaseOrderAttachment
	Content []byte
}

type PurchaseOrderConsultationResult struct {
	Code                int64                           `json:"code"`
	OrderNumber         int64                           `json:"order_number"`
	SupplierCode        *int64                          `json:"supplier_code,omitempty"`
	RequestTypeCode     *int64                          `json:"request_type_code,omitempty"`
	EmissionDate        time.Time                       `json:"emission_date"`
	DeliveryDate        *time.Time                      `json:"delivery_date,omitempty"`
	BuyerCode           *int64                          `json:"buyer_code,omitempty"`
	OrderType           string                          `json:"order_type"`
	CustomerCode        *int64                          `json:"customer_code,omitempty"`
	KanbanOrigin        bool                            `json:"kanban_origin"`
	ImportProcesses     []int64                         `json:"import_processes,omitempty"`
	CurrencyCode        string                          `json:"currency_code"`
	DisplayCurrency     string                          `json:"display_currency"`
	ConversionRate      decimal.Decimal                 `json:"conversion_rate"`
	ProductsTotal       decimal.Decimal                 `json:"products_total"`
	Freight             decimal.Decimal                 `json:"freight"`
	ProductsWithFreight decimal.Decimal                 `json:"products_with_freight"`
	Discount            decimal.Decimal                 `json:"discount"`
	Additions           decimal.Decimal                 `json:"additions"`
	Net                 decimal.Decimal                 `json:"net"`
	Total               decimal.Decimal                 `json:"total"`
	Items               []PurchaseOrderConsultationItem `json:"items"`
	Attachments         []PurchaseOrderAttachment       `json:"attachments,omitempty"`
}

type PurchaseOrderConsultationReader interface {
	Consult(ctx context.Context, filter PurchaseOrderConsultationFilter) ([]PurchaseOrderConsultationResult, error)
	GetAttachment(ctx context.Context, enterpriseID, orderCode, attachmentID int64) (*PurchaseOrderAttachmentFile, error)
}

type ConsultPurchaseOrdersUseCase struct {
	Reader PurchaseOrderConsultationReader
	Auth   ports.AuthService
}

func (uc *ConsultPurchaseOrdersUseCase) Execute(ctx context.Context, filter PurchaseOrderConsultationFilter) ([]PurchaseOrderConsultationResult, error) {
	if !uc.Auth.CanListPurchaseOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	enterpriseID, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	filter.EnterpriseID = enterpriseID
	if err := validateConsultationFilter(&filter); err != nil {
		return nil, err
	}
	return uc.Reader.Consult(ctx, filter)
}

func (uc *ConsultPurchaseOrdersUseCase) DownloadAttachment(ctx context.Context, orderCode, attachmentID int64) (*PurchaseOrderAttachmentFile, error) {
	if !uc.Auth.CanListPurchaseOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	enterpriseID, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	return uc.Reader.GetAttachment(ctx, enterpriseID, orderCode, attachmentID)
}

func validateConsultationFilter(f *PurchaseOrderConsultationFilter) error {
	if f.Limit == 0 {
		f.Limit = 100
	}
	if f.Limit < 1 || f.Limit > 500 || f.Offset < 0 {
		return fmt.Errorf("invalid pagination")
	}
	f.Position = strings.ToUpper(strings.TrimSpace(f.Position))
	f.OrderType = strings.ToUpper(strings.TrimSpace(f.OrderType))
	f.TargetCurrency = strings.ToUpper(strings.TrimSpace(f.TargetCurrency))
	if f.Position != "" && f.Position != PositionAttended && f.Position != PositionPending && f.Position != PositionCancelled {
		return fmt.Errorf("invalid position")
	}
	if f.OrderType != "" && f.OrderType != "OCL" && f.OrderType != "OSL" && f.OrderType != "ORM" && f.OrderType != "ORD" {
		return fmt.Errorf("invalid order type")
	}
	if f.Convert && (len(f.TargetCurrency) != 3 || f.BaseDate == nil) {
		return fmt.Errorf("target_currency and base_date are required for conversion")
	}
	if !f.Convert && (f.TargetCurrency != "" || f.BaseDate != nil) {
		return fmt.Errorf("conversion parameters require convert=true")
	}
	for _, pair := range [][2]*int64{{f.OrderFrom, f.OrderTo}, {f.SupplierFrom, f.SupplierTo}, {f.ItemFrom, f.ItemTo}, {f.ImportFrom, f.ImportTo}} {
		if pair[0] != nil && pair[1] != nil && *pair[0] > *pair[1] {
			return fmt.Errorf("invalid interval")
		}
	}
	for _, pair := range [][2]*time.Time{{f.EmissionFrom, f.EmissionTo}, {f.DeliveryFrom, f.DeliveryTo}} {
		if pair[0] != nil && pair[1] != nil && pair[0].After(*pair[1]) {
			return fmt.Errorf("invalid date interval")
		}
	}
	return nil
}
