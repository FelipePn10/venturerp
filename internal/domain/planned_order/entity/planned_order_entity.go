package entity

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/google/uuid"
)

type PlannedOrder struct {
	ID                   int64
	Code                 int64
	OrderNumber          int64
	ItemCode             int64
	Mask                 *string
	Quantity             decimal.Decimal
	QuantityLoss         decimal.Decimal
	QuantityCorrected    decimal.Decimal
	OrderType            types.OrderType
	Status               types.OrderStatus
	PlanCode             *int64
	DemandType           types.DemandType
	DemandCode           *int64
	NeedDate             time.Time
	StartDate            *time.Time
	EndDate              *time.Time
	CostCenterCode       *int64
	EmployeeCode         *int64
	MachineCode          *int64
	WarehouseCode        *int64
	InterFactory         bool
	SourceEnterpriseCode *int64
	AutoRelease          bool
	MRPSuggestionCode    *int64
	ProductionTime       float64
	Priority             *string
	LLC                  int
	Notes                *string
	ParentOrderCode      *int64
	SalesOrderCode       *int64
	IsFirm               bool
	IsActive             bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
	CreatedBy            uuid.UUID
}

// AlertaDoMRP é a exceção do último cálculo, ainda não tratada, sobre a ordem
// planejada (ex.: CANCEL — "não cobre nenhuma necessidade").
type AlertaDoMRP struct {
	Tipo         string
	Descricao    string
	DataSugerida *time.Time
}
