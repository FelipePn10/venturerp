package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	IndependentDemandsNo       = "NO"
	IndependentDemandsFromDate = "FROM_DATE"
	IndependentDemandsAll      = "ALL"
)

type ProductionPlan struct {
	ID                  int64
	Code                int64
	Name                string
	IndependentDemands  string // NO, FROM_DATE, ALL
	GroupSameDateOrders bool
	PlanningTypes       []string // MRP, MIN_MAX, REORDER_POINT, MPS, KANBAN
	Classification      *string
	ClassItemCodes      *string
	OrderItemCode       *int64
	LastCalculatedAt    *time.Time
	Parameters          map[string]interface{}
	IsActive            bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
	CreatedBy           uuid.UUID
}

type InterFactoryEnterprise struct {
	EnterpriseCode int64
	EnterpriseName string
	AutoRelease    bool
}
