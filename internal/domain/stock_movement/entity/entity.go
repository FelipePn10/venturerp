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
	ID                   int64
	Sigla                string
	Description          string
	UsageType            UsageType
	EntryOrder           bool
	ExitOrder            bool
	ConsidersConsumption bool
	UpdatesAvgCost       bool
	IsAdjustment         bool
	UpdatesCycleCount    bool
	ShowsInSummary       bool
	EntryExit            Direction
	GeneratesFCIMovement bool
	IsActive             bool
	CreatedAt            time.Time
	// RequiresCostCenter: o lançamento manual com este tipo exige centro de custo.
	RequiresCostCenter bool
	// AllowsManual: o tipo pode ser usado no lançamento manual (os gerados por
	// processo — nota, produção — não).
	AllowsManual bool
}
