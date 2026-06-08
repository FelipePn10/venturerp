package response

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
