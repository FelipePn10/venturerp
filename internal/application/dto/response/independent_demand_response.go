package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// IndependentDemandResponse é a demanda independente com a posição de hoje.
type IndependentDemandResponse struct {
	CodeDemand       int64           `json:"code_demand"`
	ItemCode         int64           `json:"item_code"`
	ItemBusinessCode string          `json:"item_business_code,omitempty"`
	ItemName         string          `json:"item_name,omitempty"`
	Mask             *string         `json:"mask,omitempty"`
	CostCenterCode   *int64          `json:"cost_center_code,omitempty"`
	Quantity         decimal.Decimal `json:"quantity"`
	DemandDate       time.Time       `json:"demand_date"`
	IsActive         bool            `json:"is_active"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	CreatedBy        uuid.UUID       `json:"created_by"`

	Motivo          string `json:"motivo"`
	MotivoDescricao string `json:"motivo_descricao"`
	// Familia: VENDA (a venda consome) ou FINALIDADE (atendida pela saída).
	Familia            string     `json:"familia"`
	WarehouseID        *int64     `json:"warehouse_id,omitempty"`
	WarehouseNome      string     `json:"warehouse_nome,omitempty"`
	CustomerCode       *int64     `json:"customer_code,omitempty"`
	CustomerNome       string     `json:"customer_nome,omitempty"`
	Projeto            *string    `json:"projeto,omitempty"`
	Prioridade         int        `json:"prioridade"`
	Observacao         *string    `json:"observacao,omitempty"`
	Programacao        *uuid.UUID `json:"programacao,omitempty"`
	Situacao           string     `json:"situacao"`
	EncerradaEm        *time.Time `json:"encerrada_em,omitempty"`
	MotivoEncerramento *string    `json:"motivo_encerramento,omitempty"`

	// Posição calculada: o que a venda consumiu, o que já saiu para a
	// finalidade e o que o MRP vai planejar.
	Posicao   string          `json:"posicao"`
	Consumida decimal.Decimal `json:"consumida"`
	Atendida  decimal.Decimal `json:"atendida"`
	APlanejar decimal.Decimal `json:"a_planejar"`
	VenceEm   *time.Time      `json:"vence_em,omitempty"`
	// Coberta: o que as ordens ligadas (não canceladas) cobrem. SemOrdem: o
	// que o MRP vai planejar e nenhuma ordem ligada cobre ainda — o que
	// "Gerar OF agora" oferece.
	Coberta  decimal.Decimal `json:"coberta"`
	SemOrdem decimal.Decimal `json:"sem_ordem"`

	Ordens []OrdemDaDemandaResponse `json:"ordens,omitempty"`
	// Consumo: as linhas de pedido que consumiram a demanda (rastreio).
	Consumo []ConsumoDaVendaResponse `json:"consumo,omitempty"`
}

// ConsumoDaVendaResponse é uma linha de pedido que consumiu parte de uma
// demanda ou previsão.
type ConsumoDaVendaResponse struct {
	Pedido     int64           `json:"pedido"`
	Sequencia  int             `json:"sequencia"`
	Cliente    string          `json:"cliente,omitempty"`
	Data       time.Time       `json:"data"`
	Quantidade decimal.Decimal `json:"quantidade"`
}

// ConsumoDoItemResponse é a visão do consumo de um item (SAP MD73): a
// previsão por semana, as demandas de expectativa de venda e as vendas, com
// quem consumiu o quê e a venda que passou da expectativa.
type ConsumoDoItemResponse struct {
	ItemBusinessCode  string `json:"item_business_code"`
	ItemName          string `json:"item_name"`
	Estrategia        string `json:"estrategia"`
	ConsumoDiasAntes  int    `json:"consumo_dias_antes"`
	ConsumoDiasDepois int    `json:"consumo_dias_depois"`
	SobEncomenda      bool   `json:"sob_encomenda"`

	Previsoes []PrevisaoConsumidaResponse `json:"previsoes"`
	Demandas  []DemandaConsumidaResponse  `json:"demandas"`
	Vendas    []VendaDoConsumoResponse    `json:"vendas"`
	// Totais: expectativa (previsão + demanda), consumido e o que o MRP ainda
	// planeja da expectativa; vendido e a venda acima da expectativa.
	TotalExpectativa decimal.Decimal `json:"total_expectativa"`
	TotalConsumido   decimal.Decimal `json:"total_consumido"`
	TotalAPlanejar   decimal.Decimal `json:"total_a_planejar"`
	TotalVendido     decimal.Decimal `json:"total_vendido"`
	TotalAcima       decimal.Decimal `json:"total_acima"`
}

type PrevisaoConsumidaResponse struct {
	Mask       string                   `json:"mask,omitempty"`
	Semana     time.Time                `json:"semana"`
	Quantidade decimal.Decimal          `json:"quantidade"`
	Consumida  decimal.Decimal          `json:"consumida"`
	Resto      decimal.Decimal          `json:"resto"`
	Consumo    []ConsumoDaVendaResponse `json:"consumo,omitempty"`
}

type DemandaConsumidaResponse struct {
	CodeDemand int64                    `json:"code_demand"`
	Mask       *string                  `json:"mask,omitempty"`
	Motivo     string                   `json:"motivo"`
	Data       time.Time                `json:"data"`
	Quantidade decimal.Decimal          `json:"quantidade"`
	Consumida  decimal.Decimal          `json:"consumida"`
	APlanejar  decimal.Decimal          `json:"a_planejar"`
	Posicao    string                   `json:"posicao"`
	Consumo    []ConsumoDaVendaResponse `json:"consumo,omitempty"`
}

type VendaDoConsumoResponse struct {
	Pedido     int64           `json:"pedido"`
	Sequencia  int             `json:"sequencia"`
	Cliente    string          `json:"cliente,omitempty"`
	Mask       string          `json:"mask,omitempty"`
	Data       time.Time       `json:"data"`
	Quantidade decimal.Decimal `json:"quantidade"`
	// Consumiu: quanto da expectativa a venda tomou; Acima: o que ela vendeu
	// além da expectativa (o MRP planeja pelo pedido).
	Consumiu decimal.Decimal `json:"consumiu"`
	Acima    decimal.Decimal `json:"acima"`
}

type OrdemDaDemandaResponse struct {
	PlannedOrderCode  *int64          `json:"planned_order_code,omitempty"`
	ProductionOrderID *int64          `json:"production_order_id,omitempty"`
	OrderNumber       *int64          `json:"order_number,omitempty"`
	Status            string          `json:"status"`
	Quantidade        decimal.Decimal `json:"quantidade"`
	Planejada         decimal.Decimal `json:"planejada"`
	Produzida         decimal.Decimal `json:"produzida"`
}
