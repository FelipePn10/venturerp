package response

import "time"

// PlanningPipelineResponse consolidates the result of running MRP → CRP → APS
// for a plan, with an overall viability verdict.
type PlanningPipelineResponse struct {
	PlanCode int64 `json:"plan_code"`

	MRPItems  int    `json:"mrp_items"`
	MRPOrders int    `json:"mrp_orders"`
	MRPStatus string `json:"mrp_status"`

	CRPEntries  int `json:"crp_entries"`
	CRPOverload int `json:"crp_overload"`

	APSOrders     int `json:"aps_orders"`
	APSOperations int `json:"aps_operations"`

	// Viable is true when MRP completed and CRP found no capacity overload.
	Viable bool     `json:"viable"`
	Notes  []string `json:"notes,omitempty"`
}

// AutoRunSettingsResponse é a parametrização da execução automática e a
// situação da última rodada.
type AutoRunSettingsResponse struct {
	IsEnabled          bool       `json:"is_enabled"`
	RunHour            int        `json:"run_hour"`
	RunMinute          int        `json:"run_minute"`
	PlanCode           *int64     `json:"plan_code"`
	InitialOrderNumber int64      `json:"initial_order_number"`
	GenerateLLC        bool       `json:"generate_llc"`
	LastSnapshotAt     *time.Time `json:"last_snapshot_at"`
	LastRunAt          *time.Time `json:"last_run_at"`
	LastRunStatus      string     `json:"last_run_status"`
}

// PlanningRunResponse é uma linha do histórico de execuções do planejamento.
type PlanningRunResponse struct {
	ID              int64      `json:"id"`
	PlanCode        int64      `json:"plan_code"`
	Trigger         string     `json:"trigger"`
	TriggeredBy     *string    `json:"triggered_by"`
	TriggeredByName string     `json:"triggered_by_name"`
	SnapshotAt      time.Time  `json:"snapshot_at"`
	StartedAt       time.Time  `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at"`
	Status          string     `json:"status"`
	ChangedItems    int        `json:"changed_items"`
	MRPOrders       int        `json:"mrp_orders"`
	CRPOverload     int        `json:"crp_overload"`
	Viable          *bool      `json:"viable"`
	Notes           string     `json:"notes"`
}
