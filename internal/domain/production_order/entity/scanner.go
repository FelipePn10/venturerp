package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ScanAction string

const (
	ScanResolve  ScanAction = "RESOLVER"
	ScanStart    ScanAction = "INICIAR"
	ScanAppoint  ScanAction = "APONTAR"
	ScanComplete ScanAction = "CONCLUIR"
)

type ScanToken struct {
	ID, EnterpriseID, ProductionOrderID int64
	OperationID                         *int64
	TokenHash                           []byte
	ValidUntil                          *time.Time
	CreatedBy                           uuid.UUID
}

type ScanCommand struct {
	EnterpriseID                       int64
	UserID                             uuid.UUID
	TokenHash                          []byte
	Action                             ScanAction
	IdempotencyKey, DeviceID           string
	Fingerprint                        []byte
	EmployeeID                         *int64
	GoodQuantity, ScrapQuantity, Hours decimal.Decimal
	ScrapReason                        *string
	CompleteOperation                  bool
}

type ScanResult struct {
	ProductionOrderID int64   `json:"production_order_id"`
	OperationID       *int64  `json:"operation_id,omitempty"`
	OrderNumber       int64   `json:"order_number"`
	Status            string  `json:"status"`
	OperationStatus   *string `json:"operation_status,omitempty"`
	Replayed          bool    `json:"replayed"`
}
