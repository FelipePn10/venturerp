package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Situações da requisição ao almoxarifado.
const (
	RequisicaoAberta    = "ABERTA"
	RequisicaoParcial   = "PARCIAL"
	RequisicaoAtendida  = "ATENDIDA"
	RequisicaoCancelada = "CANCELADA"
)

// RequisicaoMaterial é o pedido interno de material ao almoxarifado (consumo,
// manutenção, escritório): quem pede, para qual centro de custo e o quê. O
// almoxarife atende total ou parcialmente; cada atendimento é uma saída do
// estoque lançada no centro de custo da requisição.
type RequisicaoMaterial struct {
	ID                    int64                    `json:"id"`
	Code                  int64                    `json:"code"`
	WarehouseID           int64                    `json:"warehouse_id"`
	RequesterName         *string                  `json:"requester_name,omitempty"`
	RequesterEmployeeCode *int64                   `json:"requester_employee_code,omitempty"`
	CostCenterCode        *int32                   `json:"cost_center_code,omitempty"`
	NeededDate            *time.Time               `json:"needed_date,omitempty"`
	Status                string                   `json:"status"`
	Notes                 *string                  `json:"notes,omitempty"`
	CreatedBy             uuid.UUID                `json:"created_by"`
	CreatedAt             time.Time                `json:"created_at"`
	UpdatedAt             time.Time                `json:"updated_at"`
	Itens                 []RequisicaoMaterialItem `json:"items"`
}

type RequisicaoMaterialItem struct {
	ID        int64           `json:"id"`
	Sequence  int             `json:"sequence"`
	ItemCode  int64           `json:"item_code"`
	Mask      string          `json:"mask"`
	Quantity  decimal.Decimal `json:"quantity"`
	IssuedQty decimal.Decimal `json:"issued_qty"`
	Notes     *string         `json:"notes,omitempty"`
}

// Pendente é o que falta atender da linha.
func (i RequisicaoMaterialItem) Pendente() decimal.Decimal {
	p := i.Quantity.Sub(i.IssuedQty)
	if p.IsNegative() {
		return decimal.Zero
	}
	return p
}

// AtendimentoLinha: quanto entregar de uma linha (e de onde: lote/endereço).
type AtendimentoLinha struct {
	ItemID   int64           `json:"item_id"`
	Quantity decimal.Decimal `json:"quantity"`
	Lot      *string         `json:"lot,omitempty"`
	Address  *string         `json:"address,omitempty"`
}
