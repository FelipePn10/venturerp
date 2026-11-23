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
	MachineID                          *int64
	GoodQuantity, ScrapQuantity, Hours decimal.Decimal
	ScrapReason                        *string
	CompleteOperation                  bool
}

type ScanResult struct {
	NextAction        string  `json:"next_action,omitempty"`
	Message           string  `json:"message,omitempty"`
	ProductionOrderID int64   `json:"production_order_id"`
	OperationID       *int64  `json:"operation_id,omitempty"`
	OrderNumber       int64   `json:"order_number"`
	Status            string  `json:"status"`
	OperationStatus   *string `json:"operation_status,omitempty"`
	Replayed          bool    `json:"replayed"`
	// Baixas: o material que a leitura baixou do estoque (baixa automática),
	// para o operador ver na hora.
	Baixas []BaixaDaLeitura `json:"auto_issues,omitempty"`
}

// BaixaDaLeitura é um material baixado pela leitura que apontou peças.
type BaixaDaLeitura struct {
	ItemCode    int64           `json:"item_code"`
	Quantidade  decimal.Decimal `json:"quantity"`
	WarehouseID int64           `json:"warehouse_id"`
}
