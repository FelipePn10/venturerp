package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
)

type APSRepository interface {
	UpsertSequence(ctx context.Context, seq *entity.ProductionSequence) (*entity.ProductionSequence, error)
	GetSequence(ctx context.Context, id int64) (*entity.ProductionSequence, error)
	UpdateSequence(ctx context.Context, seq *entity.ProductionSequence) (*entity.ProductionSequence, error)
	ListByOrder(ctx context.Context, orderID int64) ([]*entity.ProductionSequence, error)
	ListByWorkCenter(ctx context.Context, workCenterID int64, from, to time.Time) ([]*entity.ProductionSequence, error)
	DeleteByOrder(ctx context.Context, orderID int64) error

	// Data needed by the sequencing algorithm
	GetOpenProductionOrders(ctx context.Context) ([]OrderRow, error)
	GetOrderOperations(ctx context.Context, orderID int64) ([]OpRow, error)
	GetWorkCenterCapacity(ctx context.Context, workCenterID int64) (float64, error)

	// Data feeding the monthly schedule board (Gantt). [from, to) is a half-open
	// window; bars come back with their labels already joined. Bars carry raw
	// quantities/hours so the use case can derive completion and lateness.
	ListScheduledBars(ctx context.Context, from, to time.Time) ([]*entity.GanttBar, error)
	ListFallbackBars(ctx context.Context, from, to time.Time) ([]*entity.GanttBar, error)
	ListResourceLoad(ctx context.Context, from, to time.Time) ([]*entity.GanttResourceLoad, error)

	// Finish-start dependencies between scheduled bars, derived from
	// route_operation_network. ListDependencies is window-scoped (board view);
	// ListOrderDependencies returns one order's edges (cascade reschedule).
	ListDependencies(ctx context.Context, from, to time.Time) ([]*entity.GanttDependency, error)
	ListOrderDependencies(ctx context.Context, orderID int64) ([]*entity.GanttDependency, error)
}

type OrderRow struct {
	ID          int64
	Priority    int
	PlannedDate time.Time
}

type OpRow struct {
	ID           int64
	Sequence     int
	WorkCenterID *int64
	PlannedHours float64
	SetupHours   float64
}

type SequenceFilter struct {
	OrderIDs      []int64
	MachineIDs    []int64
	WorkCenterIDs []int64
	OperationIDs  []int64
}

type SequencingEventRow struct {
	EventType         string
	ProductionOrderID int64
	OrderNumber       int64
	MachineID         *int64
	WorkCenterID      *int64
	OperationID       *int64
	EventAt           time.Time
	Quantity          string
	Reason            string
}

type SequencingResourceRow struct {
	ID              int64
	Code            int64
	Name            string
	WorkCenterID    int64
	ResourceGroupID *int64
	IsActive        bool
}

// SelectionRepository is the tenant-aware extension used by product sequencing.
// It is separate from APSRepository to keep legacy board implementations compatible.
type SelectionRepository interface {
	GetSelectedProductionOrders(ctx context.Context, filter SequenceFilter) ([]OrderRow, error)
	GetSelectedOrderOperations(ctx context.Context, orderID int64, filter SequenceFilter) ([]OpRow, error)
	ListSequencingEvents(ctx context.Context, filter SequenceFilter) ([]SequencingEventRow, error)
	ListSequencingResources(ctx context.Context) ([]SequencingResourceRow, error)
	ListSequencingView(ctx context.Context, filter SequencingViewFilter) ([]*entity.ProductionSequence, error)
	ListAvailabilityWindows(ctx context.Context, workCenterID int64, machineIDs []int64, from, to time.Time) ([]AvailabilityWindow, error)
	ListCandidateMachines(ctx context.Context, workCenterID int64, machineIDs []int64) ([]MachineCandidate, error)
	ListMachineDowntimeWindows(ctx context.Context, machineID int64, from, to time.Time) ([]AvailabilityWindow, error)
}
type AvailabilityWindow struct{ Start, End time.Time }
type MachineCandidate struct {
	ID            int64
	CapacityHours float64
}

type SequencingViewFilter struct {
	From, To                                                                                         time.Time
	ResourceGroupID                                                                                  int64
	FromOrder, ToOrder, FromMachine, ToMachine, FromWorkCenter, ToWorkCenter, FromPlanner, ToPlanner *int64
}

type ResourceGroup struct {
	ID                int64
	Code, Description string
}
type MachineCalendarInterval struct {
	Weekday    int
	Start, End string
}
type MachineCalendar struct {
	ID, Code    int64
	Description string
	Intervals   []MachineCalendarInterval
}
type MachineDowntime struct {
	ID, MachineID        int64
	StartsAt, EndsAt     time.Time
	DowntimeType, Reason string
	MaintenanceOrderID   *int64
}
type EmployeeContact struct {
	ID                 int64
	ContactType, Value string
	IsPrimary          bool
}
type EmployeeFunction struct {
	ID                      int64
	FunctionName            string
	CostCenterID            *int64
	IsSupervisor, IsManager bool
}
type EmployeeSequencingProfile struct {
	Contacts    []EmployeeContact
	Functions   []EmployeeFunction
	CreditLimit string
	ValidUntil  *time.Time
}
type ServiceItem struct {
	ID              int64
	ItemCode        int64
	Quantity, Notes string
}
type MachineService struct {
	ID                                    int64
	ServiceCode, Description, ServiceType string
	FrequencyValue                        int
	FrequencyUnit                         string
	MaxTolerance                          int
	SupplierCode                          *int64
	ImplementedOn                         time.Time
	LastExecutedOn                        *time.Time
	Notes                                 string
	Items                                 []ServiceItem
	ResponsibleEmployeeIDs                []int64
}
type SpecialValue struct {
	FieldID                                  int64
	Name, ValueType, TextValue, NumericValue string
	MaxLength                                *int
}
type MachineIndustrialProfile struct {
	UsageDescription                     string
	AcquiredOn                           *time.Time
	PreparationTime, PreparationTimeUnit string
	SupplierCode                         *int64
	Brand                                string
	IsPreferred                          bool
	MaintenanceResponsibleEmployeeID     *int64
	Services                             []MachineService
	SpecialValues                        []SpecialValue
}

type ConfigurationRepository interface {
	UpsertResourceGroup(context.Context, string, string) (ResourceGroup, error)
	ListResourceGroups(context.Context) ([]ResourceGroup, error)
	UpsertMachineCalendar(context.Context, int64, string, []MachineCalendarInterval) (MachineCalendar, error)
	ListMachineCalendars(context.Context) ([]MachineCalendar, error)
	UpdateSequencingSettings(context.Context, bool) error
	UpdateWorkCenterSequencing(context.Context, int64, *int64, *int64, string) error
	UpdateResourceSequencing(context.Context, int64, *int64, *int64, string, bool, bool) error
	DeleteResourceGroup(context.Context, int64) error
	DeleteMachineCalendar(context.Context, int64) error
	CreateMachineDowntime(context.Context, MachineDowntime) (MachineDowntime, error)
	ListMachineDowntimes(context.Context, int64, time.Time, time.Time) ([]MachineDowntime, error)
	DeleteMachineDowntime(context.Context, int64) error
	UpsertEmployeeSequencingProfile(context.Context, int64, EmployeeSequencingProfile) error
	UpsertMachineIndustrialProfile(context.Context, int64, MachineIndustrialProfile) error
	GetEmployeeSequencingProfile(context.Context, int64) (EmployeeSequencingProfile, error)
	GetMachineIndustrialProfile(context.Context, int64) (MachineIndustrialProfile, error)
	UpdateEmployeeContact(context.Context, int64, int64, EmployeeContact) error
	DeleteEmployeeContact(context.Context, int64, int64) error
	UpdateEmployeeFunction(context.Context, int64, int64, EmployeeFunction) error
	DeleteEmployeeFunction(context.Context, int64, int64) error
	UpdateMachineService(context.Context, int64, int64, MachineService) error
	DeleteMachineService(context.Context, int64, int64) error
	UpdateMachineServiceItem(context.Context, int64, int64, int64, ServiceItem) error
	DeleteMachineServiceItem(context.Context, int64, int64, int64) error
	UpdateMachineSpecialValue(context.Context, int64, int64, SpecialValue) error
	DeleteMachineSpecialValue(context.Context, int64, int64) error
}
