package request

import "time"

// RunPlanningPipelineDTO triggers MRP, then CRP, then APS for a single plan in
// one shot, returning a consolidated viability assessment.
type RunPlanningPipelineDTO struct {
	PlanCode    int64     `json:"plan_code"`
	GenerateLLC bool      `json:"generate_llc"`
	StartFrom   time.Time `json:"start_from"`
}
