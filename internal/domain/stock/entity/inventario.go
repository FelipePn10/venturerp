package entity

import "github.com/shopspring/decimal"

// Situações do inventário físico.
const (
	InventarioAberto              = "OPEN"
	InventarioEmConferencia       = "EM_CONFERENCIA"
	InventarioAguardandoAprovacao = "AGUARDANDO_APROVACAO"
	InventarioFechado             = "CLOSED"
)

// Quando o ajuste do inventário passa pela aprovação.
const (
	AprovacaoNunca          = "NUNCA"
	AprovacaoForaTolerancia = "FORA_TOLERANCIA"
	AprovacaoSempre         = "SEMPRE"
)

// InventarioEmAndamento diz se o inventário ainda não terminou.
func InventarioEmAndamento(status string) bool {
	return status == InventarioAberto || status == InventarioEmConferencia || status == InventarioAguardandoAprovacao
}

// ForaDaTolerancia diz se a diferença de uma linha passa do percentual sobre o
// saldo da foto ou do valor. Tolerância não informada não barra. Sem saldo na
// foto o percentual não existe: qualquer diferença passa dele.
func ForaDaTolerancia(sistema, diferenca, custo decimal.Decimal, pct, valor *decimal.Decimal) bool {
	if diferenca.IsZero() {
		return false
	}
	if pct != nil {
		if sistema.IsZero() {
			return true
		}
		percentual := diferenca.Abs().Div(sistema.Abs()).Mul(decimal.NewFromInt(100))
		if percentual.GreaterThan(*pct) {
			return true
		}
	}
	return valor != nil && diferenca.Abs().Mul(custo).GreaterThan(*valor)
}

// ResumoFechamento é o que o inventário ajustaria se fechasse agora.
type ResumoFechamento struct {
	NaoContados    int
	ComDiferenca   int
	ForaTolerancia int
	ValorAbsoluto  decimal.Decimal
}

// ExigeAprovacao diz se o fechamento passa pela alçada.
func (i *PhysicalInventory) ExigeAprovacao(r ResumoFechamento) bool {
	switch i.ApprovalMode {
	case AprovacaoSempre:
		return r.ComDiferenca > 0
	case AprovacaoForaTolerancia:
		return r.ForaTolerancia > 0
	default:
		return false
	}
}

// InconsistenciaEstoque é uma divergência entre o saldo gravado e o que os
// movimentos ou as reservas dizem.
type InconsistenciaEstoque struct {
	Tipo         string          `json:"kind"`
	ItemCode     int64           `json:"item_code"`
	Codigo       string          `json:"business_code"`
	Nome         string          `json:"name"`
	Mask         string          `json:"mask"`
	Almoxarifado int64           `json:"warehouse_id"`
	Gravado      decimal.Decimal `json:"recorded_qty"`
	Esperado     decimal.Decimal `json:"expected_qty"`
	Diferenca    decimal.Decimal `json:"difference_qty"`
}

// Tipos de inconsistência.
const (
	InconsistenciaSaldo     = "SALDO_X_MOVIMENTOS"
	InconsistenciaLote      = "LOTE_X_SALDO"
	InconsistenciaReservado = "RESERVADO_X_RESERVAS"
)
