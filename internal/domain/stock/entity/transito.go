package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Situações do envio em trânsito.
const (
	TransitoAberto    = "EM_TRANSITO"
	TransitoEncerrado = "ENCERRADA"
)

// Transito é um envio entre almoxarifados: já saiu da origem e ainda não chegou
// ao destino. Enquanto isso o material fica no almoxarifado de trânsito.
type Transito struct {
	ID            int64           `json:"id"`
	Code          int64           `json:"code"`
	ItemCode      int64           `json:"item_code"`
	Mask          string          `json:"mask"`
	Lot           *string         `json:"lot,omitempty"`
	DeAlmox       int64           `json:"from_warehouse_id"`
	AlmoxTransito int64           `json:"transit_warehouse_id"`
	ParaAlmox     int64           `json:"to_warehouse_id"`
	Quantidade    decimal.Decimal `json:"quantity"`
	Recebido      decimal.Decimal `json:"received_qty"`
	Devolvido     decimal.Decimal `json:"returned_qty"`
	Perdido       decimal.Decimal `json:"lost_qty"`
	Status        string          `json:"status"`
	Previsao      *time.Time      `json:"expected_date,omitempty"`
	RequisicaoID  *int64          `json:"requisition_id,omitempty"`
	Notas         *string         `json:"notes,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	ClosedAt      *time.Time      `json:"closed_at,omitempty"`
}

// Pendente é o que ainda está na estrada.
func (t Transito) Pendente() decimal.Decimal {
	return t.Quantidade.Sub(t.Recebido).Sub(t.Devolvido).Sub(t.Perdido)
}

// EnvioTransito é o pedido de envio. AlmoxTransito zero usa o único almoxarifado
// de trânsito da empresa.
type EnvioTransito struct {
	ItemCode      int64
	Mask          string
	Lote          *string
	EnderecoDe    *string
	DeAlmox       int64
	AlmoxTransito int64
	ParaAlmox     int64
	Quantidade    decimal.Decimal
	Previsao      *time.Time
	Notas         *string
	Ator          uuid.UUID
	RequisicaoID  *int64
}

// Destinos do que está em trânsito.
const (
	TransitoReceber  = "RECEBER"
	TransitoDevolver = "DEVOLVER"
	TransitoPerda    = "PERDA"
)

// BaixaTransito é o que acontece com parte do envio: chegar, voltar ou se perder.
type BaixaTransito struct {
	ID         int64
	Acao       string
	Quantidade decimal.Decimal
	Endereco   *string
	Motivo     string
	Ator       uuid.UUID
}
