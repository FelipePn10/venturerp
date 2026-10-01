package response

import "time"

type WorkCenterOptionResponse struct {
	ID          int64   `json:"id"`
	Code        int64   `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	IsActive    bool    `json:"is_active"`
}

type WorkCenterOptionPageResponse struct {
	Items  []WorkCenterOptionResponse `json:"items"`
	Total  int64                      `json:"total"`
	Limit  int                        `json:"limit"`
	Offset int                        `json:"offset"`
}

type WorkCenterCostResponse struct {
	ID                 int64     `json:"id"`
	WorkCenterID       int64     `json:"work_center_id"`
	CostPerHour        float64   `json:"cost_per_hour"`
	MachineCostPerHour float64   `json:"machine_cost_per_hour"`
	LaborCostPerHour   float64   `json:"labor_cost_per_hour"`
	Currency           string    `json:"currency"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ItemPurchaseCostResponse struct {
	ID        int64     `json:"id"`
	ItemCode  int64     `json:"item_code"`
	UnitCost  float64   `json:"unit_cost"`
	Currency  string    `json:"currency"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CostRollupResponse é o custo-padrão apurado, aberto por componente.
//
// `material_cost`, `labor_cost`, `overhead_cost` e `total_cost` mantêm o nome e o
// sentido de antes — precificação e relatórios já os leem. Os campos novos abrem o
// que estava agregado: preparação, máquina, serviço de terceiro, nível próprio ×
// nível inferior, o rastro dos indiretos e a árvore da estrutura.
type CostRollupResponse struct {
	ItemCode     int64   `json:"item_code"`
	Mask         string  `json:"mask"`
	MaterialCost float64 `json:"material_cost"`
	LaborCost    float64 `json:"labor_cost"`
	OverheadCost float64 `json:"overhead_cost"`
	// Componentes abertos (migração 000373).
	SetupCost       float64 `json:"setup_cost"`
	MachineCost     float64 `json:"machine_cost"`
	SubcontractCost float64 `json:"subcontract_cost"`
	OwnLevelCost    float64 `json:"own_level_cost"`
	LowerLevelCost  float64 `json:"lower_level_cost"`
	// Lote usado na apuração: o setup é diluído por ele, então duas apurações com
	// lotes diferentes não são comparáveis e a tela precisa mostrar qual foi.
	LotSize      float64   `json:"lot_size"`
	TotalCost    float64   `json:"total_cost"`
	Currency     string    `json:"currency"`
	CalculatedAt time.Time `json:"calculated_at"`
	// Rastro de cada regra de rateio aplicada: sem ele o indireto é um número no
	// total que ninguém consegue explicar.
	Overheads []CostOverheadAppliedResponse `json:"overheads,omitempty"`
	// Composição por nível da estrutura, do item para os componentes.
	Tree []CostRollupNodeResponse `json:"tree,omitempty"`
	// O que não deu certo sem invalidar a apuração (log da árvore, histórico).
	Avisos []string `json:"avisos,omitempty"`
}

// CostOverheadAppliedResponse espelha entity.RateioAplicado no contrato HTTP.
type CostOverheadAppliedResponse struct {
	RuleID      int64   `json:"rule_id"`
	Code        string  `json:"code"`
	Description string  `json:"description"`
	Base        string  `json:"base"`
	Method      string  `json:"method"`
	Rate        float64 `json:"rate"`
	BaseValue   float64 `json:"base_value"`
	Applied     float64 `json:"applied"`
}

// CostRollupNodeResponse é um nó da árvore de custo: um item da estrutura com o
// custo dele aberto.
type CostRollupNodeResponse struct {
	ItemCode        int64   `json:"item_code"`
	Mask            string  `json:"mask"`
	Level           int     `json:"level"`
	MaterialCost    float64 `json:"material_cost"`
	SetupCost       float64 `json:"setup_cost"`
	MachineCost     float64 `json:"machine_cost"`
	LaborCost       float64 `json:"labor_cost"`
	SubcontractCost float64 `json:"subcontract_cost"`
	OverheadCost    float64 `json:"overhead_cost"`
	LowerLevelCost  float64 `json:"lower_level_cost"`
	TotalCost       float64 `json:"total_cost"`
}

// CostOverheadRuleResponse é uma regra do esquema de rateio no contrato HTTP.
type CostOverheadRuleResponse struct {
	ID            int64   `json:"id"`
	Code          string  `json:"code"`
	Description   string  `json:"description"`
	Base          string  `json:"base"`
	Method        string  `json:"method"`
	Rate          float64 `json:"rate"`
	WorkCenterID  *int64  `json:"work_center_id,omitempty"`
	ItemCode      *int64  `json:"item_code,omitempty"`
	PlanoContasID *int64  `json:"plano_contas_id,omitempty"`
	CentroCustoID *int64  `json:"centro_custo_id,omitempty"`
	ValidFrom     string  `json:"valid_from"`
	ValidTo       *string `json:"valid_to,omitempty"`
	IsActive      bool    `json:"is_active"`
	Notes         *string `json:"notes,omitempty"`
}

// CostHistoryResponse é uma apuração passada, para comparar componente a
// componente ao longo do tempo.
type CostHistoryResponse struct {
	ID              int64                         `json:"id"`
	ItemCode        int64                         `json:"item_code"`
	Mask            string                        `json:"mask"`
	LotSize         float64                       `json:"lot_size"`
	MaterialCost    float64                       `json:"material_cost"`
	SetupCost       float64                       `json:"setup_cost"`
	MachineCost     float64                       `json:"machine_cost"`
	LaborCost       float64                       `json:"labor_cost"`
	SubcontractCost float64                       `json:"subcontract_cost"`
	OverheadCost    float64                       `json:"overhead_cost"`
	OwnLevelCost    float64                       `json:"own_level_cost"`
	LowerLevelCost  float64                       `json:"lower_level_cost"`
	TotalCost       float64                       `json:"total_cost"`
	Currency        string                        `json:"currency"`
	Overheads       []CostOverheadAppliedResponse `json:"overheads,omitempty"`
	CalculatedAt    time.Time                     `json:"calculated_at"`
}
