package planning_uc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Situação do cenário de simulação.
const (
	ScenarioDraft      = "RASCUNHO"  // sem cálculo, ou ajustes mudaram depois do cálculo
	ScenarioCalculated = "CALCULADO" // resultados refletem os ajustes atuais
)

// Tipos de ajuste do cenário.
const (
	ChangeDemand   = "DEMANDA"
	ChangeStock    = "ESTOQUE"
	ChangeLeadTime = "LEAD_TIME"
	ChangeCapacity = "CAPACIDADE"
)

// Referência do ajuste de demanda.
const (
	DemandRefNew         = "NOVA"
	DemandRefIndependent = "DEMANDA_INDEPENDENTE"
	DemandRefSalesOrder  = "PEDIDO_VENDA"
)

// Scenario é um cenário de simulação sobre um plano base. Ele calcula num plano
// oculto (ScenarioPlanCode) clonado do base a cada cálculo.
type Scenario struct {
	ID               int64      `json:"id"`
	Name             string     `json:"name"`
	Description      string     `json:"description,omitempty"`
	BasePlanCode     int64      `json:"base_plan_code"`
	BasePlanName     string     `json:"base_plan_name"`
	ScenarioPlanCode int64      `json:"scenario_plan_code"`
	Status           string     `json:"status"`
	CalculatedAt     *time.Time `json:"calculated_at,omitempty"`
	CalculationNotes string     `json:"calculation_notes,omitempty"`
	PromotedAt       *time.Time `json:"promoted_at,omitempty"`
	PromotedByName   string     `json:"promoted_by_name,omitempty"`
	CreatedByName    string     `json:"created_by_name,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	ChangesCount     int        `json:"changes_count"`
}

// ScenarioChange é um ajuste do cenário (um tipo por linha).
type ScenarioChange struct {
	ID              int64            `json:"id"`
	Kind            string           `json:"kind"`
	ItemCode        *int64           `json:"item_code,omitempty"`
	ItemDescription string           `json:"item_description,omitempty"`
	Mask            string           `json:"mask,omitempty"`
	DemandRefKind   string           `json:"demand_ref_kind,omitempty"`
	DemandRefCode   *int64           `json:"demand_ref_code,omitempty"`
	DemandRefLabel  string           `json:"demand_ref_label,omitempty"`
	Quantity        *decimal.Decimal `json:"quantity,omitempty"`
	NeedDate        *time.Time       `json:"need_date,omitempty"`
	LeadTimeDays    *int             `json:"lead_time_days,omitempty"`
	MachineID       *int64           `json:"machine_id,omitempty"`
	MachineName     string           `json:"machine_name,omitempty"`
	HoursPerDay     *decimal.Decimal `json:"hours_per_day,omitempty"`
	StopFrom        *time.Time       `json:"stop_from,omitempty"`
	StopTo          *time.Time       `json:"stop_to,omitempty"`
	Notes           string           `json:"notes,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
}

// BaseDemand é uma demanda que o último cálculo do plano base considerou —
// o que o planejador pode alterar ou retirar no cenário.
type BaseDemand struct {
	RefKind         string          `json:"ref_kind"`
	RefCode         int64           `json:"ref_code"`
	Label           string          `json:"label"`
	ItemCode        int64           `json:"item_code"`
	ItemDescription string          `json:"item_description"`
	NeedDate        time.Time       `json:"need_date"`
	Quantity        decimal.Decimal `json:"quantity"`
}

// PromoteResult diz o que a promoção aplicou ao plano real.
type PromoteResult struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Removed int `json:"removed"`
	// Skipped: ajustes de pedido de venda — o pedido do cliente não é alterado
	// pelo planejamento.
	Skipped int `json:"skipped"`
}

// ScenarioRepository guarda cenários e ajustes e mantém o plano oculto.
type ScenarioRepository interface {
	Create(ctx context.Context, name, description string, basePlan int64, actor uuid.UUID) (*Scenario, error)
	List(ctx context.Context) ([]Scenario, error)
	Get(ctx context.Context, id int64) (*Scenario, error)
	UpdateMeta(ctx context.Context, id int64, name, description string) error
	Delete(ctx context.Context, id int64) error
	Changes(ctx context.Context, id int64) ([]ScenarioChange, error)
	AddChange(ctx context.Context, id int64, c ScenarioChange, actor uuid.UUID) (*ScenarioChange, error)
	DeleteChange(ctx context.Context, id, changeID int64) error
	// SyncPlan copia para o plano oculto a configuração atual do plano base.
	SyncPlan(ctx context.Context, id int64) error
	MarkCalculated(ctx context.Context, id int64, notes string) error
	BaseDemands(ctx context.Context, basePlan int64) ([]BaseDemand, error)
	// Validação de referências na empresa da sessão.
	ItemExists(ctx context.Context, item int64) (bool, error)
	MachineExists(ctx context.Context, machine int64) (bool, error)
	DemandRefExists(ctx context.Context, kind string, code int64) (bool, error)
	// Promote aplica ao plano real os ajustes de demanda independente e de
	// demanda nova, numa transação, e marca o cenário como promovido.
	Promote(ctx context.Context, id int64, actor uuid.UUID) (*PromoteResult, error)
	WorkCenterNames(ctx context.Context, ids []int64) (map[int64]string, error)
}
