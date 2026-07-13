package request

import "time"

type SequenceOrdersDTO struct {
	StartFrom     time.Time `json:"start_from"`
	OrderIDs      []int64   `json:"order_ids,omitempty"`
	MachineIDs    []int64   `json:"machine_ids,omitempty"`
	WorkCenterIDs []int64   `json:"work_center_ids,omitempty"`
	OperationIDs  []int64   `json:"operation_ids,omitempty"`
}

type SequencingViewDTO struct {
	From            time.Time `json:"from"`
	To              time.Time `json:"to"`
	ResourceGroupID int64     `json:"resource_group_id"`
	FromOrder       *int64    `json:"from_order,omitempty"`
	ToOrder         *int64    `json:"to_order,omitempty"`
	FromMachine     *int64    `json:"from_machine,omitempty"`
	ToMachine       *int64    `json:"to_machine,omitempty"`
	FromWorkCenter  *int64    `json:"from_work_center,omitempty"`
	ToWorkCenter    *int64    `json:"to_work_center,omitempty"`
	FromPlanner     *int64    `json:"from_planner,omitempty"`
	ToPlanner       *int64    `json:"to_planner,omitempty"`
	TimeUnit        string    `json:"time_unit,omitempty"`
	RefreshValue    int       `json:"refresh_value,omitempty"`
}

type ResourceGroupDTO struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}
type MachineCalendarIntervalDTO struct {
	Weekday int    `json:"weekday"`
	Start   string `json:"start"`
	End     string `json:"end"`
}
type MachineCalendarDTO struct {
	Code        int64                        `json:"code"`
	Description string                       `json:"description"`
	Intervals   []MachineCalendarIntervalDTO `json:"intervals"`
}
type SequencingSettingsDTO struct {
	ListOnlyActiveResources bool `json:"list_only_active_resources"`
}
type WorkCenterSequencingDTO struct {
	MachineCostCenterID *int64 `json:"machine_cost_center_id"`
	LaborCostCenterID   *int64 `json:"labor_cost_center_id"`
	CapacityHours       string `json:"capacity_hours"`
}
type ResourceSequencingDTO struct {
	ResourceGroupID *int64 `json:"resource_group_id"`
	CalendarID      *int64 `json:"calendar_id"`
	Location        string `json:"location"`
	IsCritical      bool   `json:"is_critical"`
	IsActive        bool   `json:"is_active"`
}
type MachineDowntimeDTO struct {
	MachineID          int64     `json:"machine_id"`
	StartsAt           time.Time `json:"starts_at"`
	EndsAt             time.Time `json:"ends_at"`
	DowntimeType       string    `json:"downtime_type"`
	Reason             string    `json:"reason"`
	MaintenanceOrderID *int64    `json:"maintenance_order_id,omitempty"`
}
type EmployeeContactDTO struct {
	ContactType string `json:"contact_type"`
	Value       string `json:"value"`
	IsPrimary   bool   `json:"is_primary"`
}
type EmployeeFunctionDTO struct {
	FunctionName string `json:"function_name"`
	CostCenterID *int64 `json:"cost_center_id,omitempty"`
	IsSupervisor bool   `json:"is_supervisor"`
	IsManager    bool   `json:"is_manager"`
}
type EmployeeSequencingProfileDTO struct {
	Contacts    []EmployeeContactDTO  `json:"contacts"`
	Functions   []EmployeeFunctionDTO `json:"functions"`
	CreditLimit string                `json:"credit_limit"`
	ValidUntil  *time.Time            `json:"valid_until,omitempty"`
}
type ServiceItemDTO struct {
	ItemCode int64  `json:"item_code"`
	Quantity string `json:"quantity"`
	Notes    string `json:"notes,omitempty"`
}
type MachineServiceDTO struct {
	ServiceCode            string           `json:"service_code"`
	Description            string           `json:"description"`
	ServiceType            string           `json:"service_type"`
	FrequencyValue         int              `json:"frequency_value"`
	FrequencyUnit          string           `json:"frequency_unit"`
	MaxTolerance           int              `json:"max_tolerance"`
	SupplierCode           *int64           `json:"supplier_code,omitempty"`
	ImplementedOn          time.Time        `json:"implemented_on"`
	LastExecutedOn         *time.Time       `json:"last_executed_on,omitempty"`
	Notes                  string           `json:"notes,omitempty"`
	Items                  []ServiceItemDTO `json:"items"`
	ResponsibleEmployeeIDs []int64          `json:"responsible_employee_ids"`
}
type SpecialValueDTO struct {
	Name         string `json:"name"`
	ValueType    string `json:"value_type"`
	TextValue    string `json:"text_value,omitempty"`
	NumericValue string `json:"numeric_value,omitempty"`
	MaxLength    *int   `json:"max_length,omitempty"`
}
type MachineIndustrialProfileDTO struct {
	UsageDescription                 string              `json:"usage_description"`
	AcquiredOn                       *time.Time          `json:"acquired_on,omitempty"`
	PreparationTime                  string              `json:"preparation_time"`
	PreparationTimeUnit              string              `json:"preparation_time_unit"`
	SupplierCode                     *int64              `json:"supplier_code,omitempty"`
	Brand                            string              `json:"brand"`
	IsPreferred                      bool                `json:"is_preferred"`
	MaintenanceResponsibleEmployeeID *int64              `json:"maintenance_responsible_employee_id,omitempty"`
	Services                         []MachineServiceDTO `json:"services"`
	SpecialValues                    []SpecialValueDTO   `json:"special_values"`
}

type GanttByWorkCenterDTO struct {
	WorkCenterID int64     `json:"work_center_id"`
	From         time.Time `json:"from"`
	To           time.Time `json:"to"`
}

// RescheduleSequenceDTO is a planner's manual move of one scheduled operation on
// the board ("drag-drop"): a new start, optionally a different work center, and
// whether dependent operations should be pushed to keep the finish-start chain
// valid. When NewWorkCenterID is nil the bar stays on its current resource; when
// Cascade is nil it defaults to true.
type RescheduleSequenceDTO struct {
	SequenceID      int64     `json:"sequence_id"`
	NewStart        time.Time `json:"new_start"`
	NewWorkCenterID *int64    `json:"new_work_center_id,omitempty"`
	NewMachineID    *int64    `json:"new_machine_id,omitempty"`
	Cascade         *bool     `json:"cascade,omitempty"`
}
