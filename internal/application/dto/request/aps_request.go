package request

import "time"

type SequenceOrdersDTO struct {
	StartFrom time.Time `json:"start_from"`
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
	Cascade         *bool     `json:"cascade,omitempty"`
}
