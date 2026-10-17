package entity

import "time"

type UsageType string

const (
	UsageProducao      UsageType = "PRODUCAO"
	UsageCompras       UsageType = "COMPRAS"
	UsageVendas        UsageType = "VENDAS"
	UsageGeral         UsageType = "GERAL"
	UsageAjuste        UsageType = "AJUSTE"
	UsageTransferencia UsageType = "TRANSFERENCIA"
)

type Direction string

const (
	DirEntrada       Direction = "ENTRADA"
	DirSaida         Direction = "SAIDA"
	DirTransferencia Direction = "TRANSFERENCIA"
	DirAmbos         Direction = "AMBOS"
)

type StockMovementType struct {
	ID                   int64     `json:"id"`
	Sigla                string    `json:"sigla"`
	Description          string    `json:"description"`
	UsageType            UsageType `json:"usage_type"`
	EntryOrder           bool      `json:"entry_order"`
	ExitOrder            bool      `json:"exit_order"`
	ConsidersConsumption bool      `json:"considers_consumption"`
	UpdatesAvgCost       bool      `json:"updates_avg_cost"`
	IsAdjustment         bool      `json:"is_adjustment"`
	UpdatesCycleCount    bool      `json:"updates_cycle_count"`
	ShowsInSummary       bool      `json:"shows_in_summary"`
	EntryExit            Direction `json:"entry_exit"`
	GeneratesFCIMovement bool      `json:"generates_fci_movement"`
	IsActive             bool      `json:"is_active"`
	CreatedAt            time.Time `json:"created_at"`
	// RequiresCostCenter: o lançamento manual com este tipo exige centro de custo.
	RequiresCostCenter bool `json:"requires_cost_center"`
	// AllowsManual: o tipo pode ser usado no lançamento manual (os gerados por
	// processo — nota, produção — não).
	AllowsManual bool `json:"allows_manual"`
}
