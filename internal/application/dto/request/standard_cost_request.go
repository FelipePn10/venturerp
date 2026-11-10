package request

import "github.com/shopspring/decimal"

type UpsertWorkCenterCostDTO struct {
	WorkCenterID int64           `json:"work_center_id"`
	CostPerHour  decimal.Decimal `json:"cost_per_hour"` // blended rate (machine-rate fallback)
	// Optional machine × labor split. When omitted, machine uses cost_per_hour and labor is 0.
	MachineCostPerHour decimal.Decimal `json:"machine_cost_per_hour"`
	LaborCostPerHour   decimal.Decimal `json:"labor_cost_per_hour"`
	Currency           string          `json:"currency"`
	// UpdatedBy vem do JWT; nunca do corpo da requisição — aceitar do cliente
	// deixava qualquer um assinar a alteração de tarifa como outra pessoa.
	UpdatedBy string `json:"-"`
}

type UpsertItemPurchaseCostDTO struct {
	ItemCode int64           `json:"item_code"`
	UnitCost decimal.Decimal `json:"unit_cost"`
	Currency string          `json:"currency"`
	// UpdatedBy vem do JWT (handler), como na tarifa do centro: aceitar do
	// corpo deixava assinar a alteração como outra pessoa — e, sem ele, a tela
	// recebia "não foi possível identificar o usuário".
	UpdatedBy string `json:"-"`
}

type CostRollupDTO struct {
	ItemCode int64  `json:"item_code"`
	Mask     string `json:"mask"`
	// LotSize is the reference production lot used to amortize operation setup over the
	// standard cost (setup/lote ÷ lot). Defaults to 1 (setup fully charged per unit).
	LotSize      decimal.Decimal `json:"lot_size"`
	CalculatedBy string          `json:"calculated_by"`
}

// ─── esquema de rateio de indiretos (migração 000373) ────────────────────────

// CostOverheadRuleDTO é uma linha do esquema de cálculo de indiretos.
//
// `rate` entra como FRAÇÃO no método percentual (0,12 para 12%), igual ao resto do
// domínio (perda de estrutura, alíquota fiscal). O banco recusa acima de 10 para
// pegar quem digitou 12 querendo 12%.
type CostOverheadRuleDTO struct {
	Code          string          `json:"code"`
	Description   string          `json:"description"`
	Base          string          `json:"base"`
	Method        string          `json:"method"`
	Rate          decimal.Decimal `json:"rate"`
	WorkCenterID  *int64          `json:"work_center_id,omitempty"`
	ItemCode      *int64          `json:"item_code,omitempty"`
	PlanoContasID *int64          `json:"plano_contas_id,omitempty"`
	CentroCustoID *int64          `json:"centro_custo_id,omitempty"`
	ValidFrom     string          `json:"valid_from"`
	ValidTo       *string         `json:"valid_to,omitempty"`
	IsActive      *bool           `json:"is_active,omitempty"`
	Notes         *string         `json:"notes,omitempty"`
	// CreatedBy vem do JWT, nunca do corpo: quem cadastrou a taxa que entra no
	// custo de todo produto não pode ser escolhido pelo cliente.
	CreatedBy string `json:"-"`
}
