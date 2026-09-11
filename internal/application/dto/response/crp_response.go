package response

import "time"

type CRPSummaryResponse struct {
	PlanCode      int64 `json:"plan_code"`
	TotalEntries  int   `json:"total_entries"`
	OverloadCount int   `json:"overload_count"`
	// WorkCentersNoCapacity lista os centros de trabalho sem capacidade
	// cadastrada. O cálculo assume 8 h para não travar, mas o número de
	// sobrecarga desses centros não significa nada até alguém cadastrar a
	// capacidade real — e o usuário precisa saber disso.
	WorkCentersNoCapacity []int64 `json:"work_centers_without_capacity,omitempty"`
}

// CRPPlanResponse é uma linha do catálogo de planos do modal de CRP (VPRO0200):
// o usuário escolhe daqui em vez de digitar o código do plano de cabeça.
type CRPPlanResponse struct {
	PlanCode int64  `json:"plan_code"`
	Name     string `json:"name"`
	// TotalEntries é quanto de carga já foi calculada; zero significa que o
	// plano ainda precisa ser calculado antes de exportar.
	TotalEntries   int64   `json:"total_entries"`
	OverloadCount  int64   `json:"overload_count"`
	MaxLoadPct     float64 `json:"max_load_pct"`
	Calculated     bool    `json:"calculated"`
	LastCalculated *string `json:"last_calculated_at,omitempty"`
}

type CRPEntryResponse struct {
	ID             int64     `json:"id"`
	PlanCode       int64     `json:"plan_code"`
	WorkCenterID   int64     `json:"work_center_id"`
	ReqDate        time.Time `json:"req_date"`
	RequiredHours  float64   `json:"required_hours"`
	AvailableHours float64   `json:"available_hours"`
	LoadPct        float64   `json:"load_pct"`
	IsOverloaded   bool      `json:"is_overloaded"`
}
