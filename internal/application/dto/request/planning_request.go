package request

import (
	"time"

	"github.com/shopspring/decimal"
)

// RunPlanningPipelineDTO triggers MRP, then CRP, then APS for a single plan in
// one shot, returning a consolidated viability assessment.
type RunPlanningPipelineDTO struct {
	PlanCode           int64     `json:"plan_code"`
	InitialOrderNumber int64     `json:"initial_order_number"`
	GenerateLLC        bool      `json:"generate_llc"`
	StartFrom          time.Time `json:"start_from"`
}

// UpdateAutoRunSettingsDTO é a parametrização da execução automática do
// planejamento (janela noturna) da empresa da sessão.
type UpdateAutoRunSettingsDTO struct {
	IsEnabled          bool  `json:"is_enabled"`
	RunHour            int   `json:"run_hour"`
	RunMinute          int   `json:"run_minute"`
	PlanCode           int64 `json:"plan_code"`
	InitialOrderNumber int64 `json:"initial_order_number"`
	GenerateLLC        bool  `json:"generate_llc"`
}

// CreateScenarioDTO cria um cenário de simulação sobre um plano.
type CreateScenarioDTO struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	BasePlanCode int64  `json:"base_plan_code"`
}

// UpdateScenarioDTO renomeia/descreve o cenário.
type UpdateScenarioDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ScenarioChangeDTO é um ajuste do cenário. kind: DEMANDA (demand_ref_kind
// NOVA com item/quantidade/data, ou DEMANDA_INDEPENDENTE/PEDIDO_VENDA com
// demand_ref_code e nova quantidade e/ou data — quantidade 0 retira),
// ESTOQUE (item, quantity + entrada / − baixa), LEAD_TIME (item,
// lead_time_days) ou CAPACIDADE (machine_id, hours_per_day e/ou parada
// stop_from..stop_to). Datas no formato ano-mês-dia.
type ScenarioChangeDTO struct {
	Kind          string           `json:"kind"`
	ItemCode      *int64           `json:"item_code,omitempty"`
	Mask          string           `json:"mask,omitempty"`
	DemandRefKind string           `json:"demand_ref_kind,omitempty"`
	DemandRefCode *int64           `json:"demand_ref_code,omitempty"`
	Quantity      *decimal.Decimal `json:"quantity,omitempty"`
	NeedDate      *string          `json:"need_date,omitempty"`
	LeadTimeDays  *int             `json:"lead_time_days,omitempty"`
	MachineID     *int64           `json:"machine_id,omitempty"`
	HoursPerDay   *decimal.Decimal `json:"hours_per_day,omitempty"`
	StopFrom      *string          `json:"stop_from,omitempty"`
	StopTo        *string          `json:"stop_to,omitempty"`
	Notes         string           `json:"notes,omitempty"`
}
