package entity

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Situações da requisição ao almoxarifado.
const (
	RequisicaoPendente  = "PENDENTE_APROVACAO"
	RequisicaoReprovada = "REPROVADA"
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
	Kind                  string                   `json:"kind"`
	DestinoID             *int64                   `json:"destination_warehouse_id,omitempty"`
	Status                string                   `json:"status"`
	Notes                 *string                  `json:"notes,omitempty"`
	CreatedBy             uuid.UUID                `json:"created_by"`
	CreatedAt             time.Time                `json:"created_at"`
	UpdatedAt             time.Time                `json:"updated_at"`
	Itens                 []RequisicaoMaterialItem `json:"items"`
	// Valor é o custo do que foi pedido, para a alçada. Só na criação.
	Valor decimal.Decimal `json:"-"`
	// AguardaAprovacao: nasce pendente e só reserva depois de aprovada.
	AguardaAprovacao bool `json:"-"`
}

// Tipos de requisição: a de consumo baixa no centro de custo; a de transferência
// leva o material a outro almoxarifado e é a ordem de reposição.
const (
	RequisicaoConsumo       = "CONSUMO"
	RequisicaoTransferencia = "TRANSFERENCIA"
)

// Atendivel diz se a requisição pode receber entrega.
func (r RequisicaoMaterial) Atendivel() bool {
	return r.Status == RequisicaoAberta || r.Status == RequisicaoParcial
}

type RequisicaoMaterialItem struct {
	ID        int64           `json:"id"`
	Sequence  int             `json:"sequence"`
	ItemCode  int64           `json:"item_code"`
	Mask      string          `json:"mask"`
	Quantity  decimal.Decimal `json:"quantity"`
	IssuedQty decimal.Decimal `json:"issued_qty"`
	// ReservedQty é o que ainda está reservado para a linha.
	ReservedQty decimal.Decimal `json:"reserved_qty"`
	Notes       *string         `json:"notes,omitempty"`
}

// QuantidadeBR escreve a quantidade como o usuário lê: 12,5 e não 12.5.
func QuantidadeBR(q decimal.Decimal) string {
	return strings.Replace(q.String(), ".", ",", 1)
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
	// DestinationAddress: endereço de chegada, na requisição de transferência.
	DestinationAddress *string `json:"destination_address,omitempty"`
	// ViaTransito manda a entrega para o trânsito, em vez de transferir direto.
	ViaTransito bool `json:"via_transit,omitempty"`
	// AlmoxTransito: qual trânsito usar, quando a empresa tem mais de um.
	AlmoxTransito int64 `json:"transit_warehouse_id,omitempty"`
}
