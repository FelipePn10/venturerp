package entity

import "time"

type SequenceStatus string

const (
	StatusScheduled SequenceStatus = "SCHEDULED"
	StatusConfirmed SequenceStatus = "CONFIRMED"
	StatusDone      SequenceStatus = "DONE"
)

type ProductionSequence struct {
	ID                int64
	ProductionOrderID int64
	OperationID       *int64
	WorkCenterID      int64
	SequencePosition  int
	ScheduledStart    time.Time
	ScheduledEnd      time.Time
	Status            SequenceStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// GanttTask is the projection returned by the Gantt endpoint.
type GanttTask struct {
	SequenceID        int64
	ProductionOrderID int64
	WorkCenterID      int64
	SequencePosition  int
	ScheduledStart    time.Time
	ScheduledEnd      time.Time
	Status            SequenceStatus
	DurationHours     float64
}
