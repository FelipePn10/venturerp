package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type OperationOrigin string
type OperationSituation string
type RouteSituation string
type RouteOpSituation string

const (
	OriginInternal  OperationOrigin = "INTERNA"
	OriginExternal  OperationOrigin = "EXTERNA"
	OriginThirdPart OperationOrigin = "TERCEIROS"

	SituationApproved OperationSituation = "APROVADA"
	SituationInactive OperationSituation = "INATIVA"

	RouteSituationApproved RouteSituation = "APROVADA"
	RouteSituationInactive RouteSituation = "INATIVA"

	RouteOpApproved RouteOpSituation = "APROVADA"
	RouteOpInactive RouteOpSituation = "INATIVA"
	RouteOpGhost    RouteOpSituation = "FANTASMA"
)

// Operation is a catalog-level manufacturing step.
type Operation struct {
	ID                  int64
	Code                int64
	Name                string
	Description         *string
	Origin              OperationOrigin
	Situation           OperationSituation
	DefaultWorkCenterID *int64  // FK → machine_types.id
	StandardTime        float64 // hours
	SetupTime           float64 // hours
	IsActive            bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
	CreatedBy           uuid.UUID
}

func NewOperation(
	code int64,
	name string,
	description *string,
	origin OperationOrigin,
	defaultWorkCenterID *int64,
	standardTime, setupTime float64,
	createdBy uuid.UUID,
) (*Operation, error) {
	if code <= 0 {
		return nil, errors.New("operation code must be positive")
	}
	if name == "" {
		return nil, errors.New("operation name is required")
	}
	if standardTime < 0 {
		return nil, errors.New("standard_time cannot be negative")
	}
	return &Operation{
		Code:                code,
		Name:                name,
		Description:         description,
		Origin:              origin,
		Situation:           SituationApproved,
		DefaultWorkCenterID: defaultWorkCenterID,
		StandardTime:        standardTime,
		SetupTime:           setupTime,
		IsActive:            true,
		CreatedBy:           createdBy,
	}, nil
}

// ManufacturingRoute is the route header linked to an item (and optional mask).
type ManufacturingRoute struct {
	ID          int64
	Code        int64
	ItemCode    int64
	Mask        *string
	Alternative int16
	Description *string
	Situation   RouteSituation
	IsStandard  bool
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   uuid.UUID
}

func NewManufacturingRoute(
	code, itemCode int64,
	mask *string,
	alternative int16,
	description *string,
	isStandard bool,
	createdBy uuid.UUID,
) (*ManufacturingRoute, error) {
	if code <= 0 {
		return nil, errors.New("route code must be positive")
	}
	if itemCode <= 0 {
		return nil, errors.New("item_code must be positive")
	}
	if alternative <= 0 {
		alternative = 1
	}
	return &ManufacturingRoute{
		Code:        code,
		ItemCode:    itemCode,
		Mask:        mask,
		Alternative: alternative,
		Description: description,
		Situation:   RouteSituationApproved,
		IsStandard:  isStandard,
		IsActive:    true,
		CreatedBy:   createdBy,
	}, nil
}

// RouteOperation is one step inside a manufacturing route.
type RouteOperation struct {
	ID              int64
	RouteID         int64
	Sequence        int16
	OperationID     int64
	WorkCenterID    *int64   // overrides operation default when set
	StandardTime    *float64 // nil = inherit from operation
	SetupTime       *float64 // nil = inherit from operation
	Situation       RouteOpSituation
	Notes           *string
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
	OperationOrigin OperationOrigin // INTERNA / EXTERNA / TERCEIROS

	// denormalized for reads
	OperationName    string
	WorkCenterName   string
	EffectiveStdTime float64
	EffectiveSetup   float64
	RequiresOperator bool // herdado de machine_types; quando true o CPM ignora overlap_pct
}

// ExternalOp is a value object representing an external/third-party operation
// from the standard route of an item — used by MRP to generate SERVICO orders.
type ExternalOp struct {
	RouteOpID      int64
	OperationID    int64
	OperationName  string
	WorkCenterID   *int64
	EffectiveHours float64
	Origin         OperationOrigin
}

func NewRouteOperation(
	routeID int64,
	sequence int16,
	operationID int64,
	workCenterID *int64,
	standardTime, setupTime *float64,
	notes *string,
) (*RouteOperation, error) {
	if routeID <= 0 {
		return nil, errors.New("route_id must be positive")
	}
	if sequence <= 0 {
		return nil, errors.New("sequence must be positive")
	}
	if operationID <= 0 {
		return nil, errors.New("operation_id must be positive")
	}
	return &RouteOperation{
		RouteID:      routeID,
		Sequence:     sequence,
		OperationID:  operationID,
		WorkCenterID: workCenterID,
		StandardTime: standardTime,
		SetupTime:    setupTime,
		Situation:    RouteOpApproved,
		Notes:        notes,
		IsActive:     true,
	}, nil
}

// NetworkEdge represents a predecessor → successor dependency in a route.
type NetworkEdge struct {
	ID            int64
	PredecessorID int64
	SuccessorID   int64
	OverlapPct    float64 // 0..100: how much of successor can start before predecessor ends
	CreatedAt     time.Time
}

// LeadTimeResult holds the critical-path lead time for a route.
type LeadTimeResult struct {
	RouteID      int64
	CriticalPath []int64 // route_operation IDs in order
	TotalHours   float64
}
