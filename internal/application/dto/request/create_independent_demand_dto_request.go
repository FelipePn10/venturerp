package request

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreateIndependentDemandDTO cadastra a demanda (ou as entregas de uma
// programação). ItemCode é o código de negócio do item; CodeDemand 0 numera
// sozinho.
type CreateIndependentDemandDTO struct {
	CodeDemand     int64           `json:"code_demand"`
	ItemCode       TextCode        `json:"item_code"`
	Mask           *string         `json:"mask,omitempty"`
	CostCenterCode *int64          `json:"cost_center_code,omitempty"`
	Quantity       decimal.Decimal `json:"quantity"`
	DemandDate     string          `json:"demand_date"`
	CreatedBy      uuid.UUID       `json:"-"`
	Motivo         string          `json:"motivo"`
	WarehouseID    *int64          `json:"warehouse_id,omitempty"`
	CustomerCode   *int64          `json:"customer_code,omitempty"`
	Projeto        *string         `json:"projeto,omitempty"`
	Prioridade     int             `json:"prioridade,omitempty"`
	Observacao     *string         `json:"observacao,omitempty"`
	// Programação: Entregas > 1 divide a quantidade em entregas a cada
	// IntervaloDias a partir de DemandDate.
	Entregas      int `json:"entregas,omitempty"`
	IntervaloDias int `json:"intervalo_dias,omitempty"`
}

type UpdateIndependentDemandDTO struct {
	CodeDemand     int64           `json:"code_demand"`
	ItemCode       TextCode        `json:"item_code"`
	Mask           *string         `json:"mask,omitempty"`
	CostCenterCode *int64          `json:"cost_center_code,omitempty"`
	Quantity       decimal.Decimal `json:"quantity"`
	DemandDate     string          `json:"demand_date"`
	Motivo         string          `json:"motivo"`
	WarehouseID    *int64          `json:"warehouse_id,omitempty"`
	CustomerCode   *int64          `json:"customer_code,omitempty"`
	Projeto        *string         `json:"projeto,omitempty"`
	Prioridade     int             `json:"prioridade,omitempty"`
	Observacao     *string         `json:"observacao,omitempty"`
}

// EncerrarDemandaDTO cancela a demanda ou encerra o saldo dela.
type EncerrarDemandaDTO struct {
	Motivo string `json:"motivo"`
}

// AtenderDemandaDTO lança a saída de estoque para a finalidade da demanda.
type AtenderDemandaDTO struct {
	Quantidade       decimal.Decimal `json:"quantidade"`
	WarehouseID      int64           `json:"warehouse_id"`
	MovementTypeCode string          `json:"movement_type_code"`
	Observacao       *string         `json:"observacao,omitempty"`
}

// GerarOFDaDemandaDTO abre a OF já vinculada à demanda.
type GerarOFDaDemandaDTO struct {
	Quantidade decimal.Decimal `json:"quantidade"`
	StartDate  *string         `json:"start_date,omitempty"`
	EndDate    *string         `json:"end_date,omitempty"`
	Priority   *string         `json:"priority,omitempty"`
	Notes      *string         `json:"notes,omitempty"`
}
