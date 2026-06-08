package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Frequency string
type OrderStatus string

const (
	FrequencyDaily      Frequency = "DAILY"
	FrequencyWeekly     Frequency = "WEEKLY"
	FrequencyMonthly    Frequency = "MONTHLY"
	FrequencyCustomDays Frequency = "CUSTOM_DAYS"

	OrderStatusPlanned    OrderStatus = "PLANNED"
	OrderStatusInProgress OrderStatus = "IN_PROGRESS"
	OrderStatusDone       OrderStatus = "DONE"
	OrderStatusCancelled  OrderStatus = "CANCELLED"
)

type MaintenancePlan struct {
	ID              int64
	Code            int64
	MachineID       int64
	WorkCenterID    *int64
	Description     string
	Frequency       Frequency
	FrequencyDays   int
	EstimatedHours  float64
	LastExecutedAt  *time.Time
	NextScheduledAt *time.Time
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CreatedBy       uuid.UUID
}

type MaintenanceOrder struct {
	ID             int64
	PlanID         int64
	MachineID      *int64
	WorkCenterID   *int64
	ScheduledDate  time.Time
	EstimatedHours float64
	ActualHours    *float64
	Status         OrderStatus
	StartedAt      *time.Time
	CompletedAt    *time.Time
	Notes          *string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewMaintenancePlan(
	machineID int64,
	workCenterID *int64,
	description string,
	frequency Frequency,
	frequencyDays int,
	estimatedHours float64,
	createdBy uuid.UUID,
) (*MaintenancePlan, error) {
	if machineID <= 0 {
		return nil, errors.New("machine_id is required")
	}
	if description == "" {
		return nil, errors.New("description is required")
	}
	if frequencyDays <= 0 {
		return nil, errors.New("frequency_days must be positive")
	}
	if estimatedHours <= 0 {
		return nil, errors.New("estimated_hours must be positive")
	}
	nextScheduled := time.Now().AddDate(0, 0, frequencyDays)
	return &MaintenancePlan{
		MachineID:       machineID,
		WorkCenterID:    workCenterID,
		Description:     description,
		Frequency:       frequency,
		FrequencyDays:   frequencyDays,
		EstimatedHours:  estimatedHours,
		NextScheduledAt: &nextScheduled,
		IsActive:        true,
		CreatedBy:       createdBy,
	}, nil
}

func NewMaintenanceOrder(
	planID int64,
	machineID *int64,
	workCenterID *int64,
	scheduledDate time.Time,
	estimatedHours float64,
) (*MaintenanceOrder, error) {
	if planID <= 0 {
		return nil, errors.New("plan_id is required")
	}
	if scheduledDate.IsZero() {
		return nil, errors.New("scheduled_date is required")
	}
	if estimatedHours <= 0 {
		return nil, errors.New("estimated_hours must be positive")
	}
	return &MaintenanceOrder{
		PlanID:         planID,
		MachineID:      machineID,
		WorkCenterID:   workCenterID,
		ScheduledDate:  scheduledDate,
		EstimatedHours: estimatedHours,
		Status:         OrderStatusPlanned,
		IsActive:       true,
	}, nil
}
