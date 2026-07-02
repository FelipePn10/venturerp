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
	StandardTime        float64 // legacy flat time (kept in sync with RunTime)
	SetupTime           float64 // setup, per lot, in TimeUnit

	// Rich time model (defaults; a route operation may override each component).
	RunTime    float64 // machine/processing time per RunBaseQty, in TimeUnit
	LaborTime  float64 // direct-labor time per RunBaseQty (0 ⇒ equals RunTime)
	RunBaseQty float64 // pieces covered by one run cycle (>=1)
	QueueTime  float64 // fixed per lot, in TimeUnit
	WaitTime   float64 // fixed per lot, in TimeUnit
	MoveTime   float64 // fixed per lot, in TimeUnit
	CrewSize   float64 // simultaneous operators (>=1)
	TimeUnit   string  // MIN | HORA | DIA

	// Subcontracting defaults (EXTERNA / TERCEIROS). Nil for internal operations.
	SupplierID      *int64
	ServiceItemCode *int64
	CostPerUnit     *float64
	LeadTimeDays    *int32

	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy uuid.UUID
}

// DefaultTime returns the operation's default time components (in its own TimeUnit).
func (o *Operation) DefaultTime() TimeComponents {
	return TimeComponents{
		Setup:      o.SetupTime,
		Run:        o.RunTime,
		Labor:      o.LaborTime,
		RunBaseQty: o.RunBaseQty,
		Queue:      o.QueueTime,
		Wait:       o.WaitTime,
		Move:       o.MoveTime,
		CrewSize:   o.CrewSize,
		Unit:       o.TimeUnit,
	}
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
		// Rich-time defaults keep the entity valid-by-construction (DB CHECKs):
		// run mirrors the legacy standard time, base/crew ≥ 1, unit = HORA.
		RunTime:    standardTime,
		RunBaseQty: 1,
		CrewSize:   1,
		TimeUnit:   TimeUnitHour,
		IsActive:   true,
		CreatedBy:  createdBy,
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
	ValidFrom   *time.Time // nil = valid from the beginning
	ValidTo     *time.Time // nil = open-ended
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
	validFrom, validTo *time.Time,
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
	if validFrom != nil && validTo != nil && validTo.Before(*validFrom) {
		return nil, errors.New("valid_to cannot be before valid_from")
	}
	return &ManufacturingRoute{
		Code:        code,
		ItemCode:    itemCode,
		Mask:        mask,
		Alternative: alternative,
		Description: description,
		Situation:   RouteSituationApproved,
		IsStandard:  isStandard,
		ValidFrom:   validFrom,
		ValidTo:     validTo,
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

	// Rich time-model overrides (nil ⇒ inherit from the operation default).
	RunTime    *float64
	LaborTime  *float64
	RunBaseQty *float64
	QueueTime  *float64
	WaitTime   *float64
	MoveTime   *float64
	CrewSize   *float64
	TimeUnit   *string

	// Subcontracting overrides (nil ⇒ inherit from the operation).
	SupplierID      *int64
	ServiceItemCode *int64
	CostPerUnit     *float64
	LeadTimeDays    *int32

	// denormalized for reads
	OperationName         string
	WorkCenterName        string
	EffectiveWorkCenterID *int64        // COALESCE(route-op WC, operation default WC) — the CT that actually runs it
	EffectiveStdTime      float64       // = EffTime.Run (hours); kept for backward-compat
	EffectiveSetup        float64       // = EffTime.Setup (hours); kept for backward-compat
	EffTime               OperationTime // resolved, quantity-aware time model (hours)
	RequiresOperator      bool          // herdado de machine_types; quando true o CPM ignora overlap_pct
}

// Overrides returns the route-operation's time overrides for resolution.
// SetupTime (legacy column) doubles as the setup override.
func (ro *RouteOperation) Overrides() TimeOverrides {
	return TimeOverrides{
		Setup:      ro.SetupTime,
		Run:        ro.RunTime,
		Labor:      ro.LaborTime,
		RunBaseQty: ro.RunBaseQty,
		Queue:      ro.QueueTime,
		Wait:       ro.WaitTime,
		Move:       ro.MoveTime,
		CrewSize:   ro.CrewSize,
		Unit:       ro.TimeUnit,
	}
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

	// Subcontracting (effective: route-op override ∘ operation default).
	SupplierID      *int64
	ServiceItemCode *int64
	CostPerUnit     float64
	LeadTimeDays    int32
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

// RouteOpResource is an alternative work center that can run a route operation.
// The primary resource (IsPrimary) mirrors route_operations.work_center_id; the
// others are options the APS/CRP may pick when the primary is overloaded.
type RouteOpResource struct {
	ID               int64
	RouteOperationID int64
	WorkCenterID     int64
	Priority         int16   // 1 = most preferred
	TimeFactor       float64 // scales op time on this resource (1.0 = base, 1.2 = 20% slower)
	IsPrimary        bool
	CreatedAt        time.Time
	UpdatedAt        time.Time

	// denormalized for reads
	WorkCenterName string
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
