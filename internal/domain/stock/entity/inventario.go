package entity

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

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
	MaiorLinha     decimal.Decimal
}

// LimiteDeAjuste é até quanto o usuário fecha um inventário sem alçada. Nil naquele
// campo não limita.
type LimiteDeAjuste struct {
	ValorDocumento *decimal.Decimal
	ValorLinha     *decimal.Decimal
}

// Excede diz se o ajuste passa do limite do usuário.
func (l LimiteDeAjuste) Excede(r ResumoFechamento) bool {
	return (l.ValorDocumento != nil && r.ValorAbsoluto.GreaterThan(*l.ValorDocumento)) ||
		(l.ValorLinha != nil && r.MaiorLinha.GreaterThan(*l.ValorLinha))
}

// ToleranciaDoPerfil é o limite de ajuste de um perfil de acesso.
type ToleranciaDoPerfil struct {
	PerfilID       int64            `json:"profile_id"`
	Nome           string           `json:"profile_name"`
	ValorDocumento *decimal.Decimal `json:"max_document_value,omitempty"`
	ValorLinha     *decimal.Decimal `json:"max_line_value,omitempty"`
}

// ExigeAprovacao diz se o fechamento passa pela alçada. O limite do perfil de quem
// fecha vale sobre a regra do inventário: acima dele vai à alçada mesmo no modo
// NUNCA, e dentro dele dispensa o FORA_TOLERANCIA. SEMPRE não abre exceção.
func (i *PhysicalInventory) ExigeAprovacao(r ResumoFechamento, limite *LimiteDeAjuste) bool {
	if i.ApprovalMode == AprovacaoSempre {
		return r.ComDiferenca > 0
	}
	if limite != nil {
		return r.ComDiferenca > 0 && limite.Excede(r)
	}
	switch i.ApprovalMode {
	case AprovacaoForaTolerancia:
		return r.ForaTolerancia > 0
	default:
		return false
	}
}

// CadernoInventario é um lote de linhas impresso para a contagem no papel. O avanço
// vem das próprias linhas: digitada a contagem, o caderno está pronto.
type CadernoInventario struct {
	Numero       int        `json:"number"`
	Almoxarifado int64      `json:"warehouse_id"`
	Linhas       int        `json:"lines"`
	Contadas     int        `json:"counted_lines"`
	Impressoes   int        `json:"print_count"`
	ImpressoEm   *time.Time `json:"printed_at,omitempty"`
}

// Situacao do caderno, na ordem em que ele avança.
func (c CadernoInventario) Situacao() string {
	switch {
	case c.Linhas > 0 && c.Contadas >= c.Linhas:
		return "DIGITADO"
	case c.Contadas > 0:
		return "EM_DIGITACAO"
	case c.Impressoes > 0:
		return "IMPRESSO"
	}
	return "GERADO"
}

// MarshalJSON leva a situação junto.
func (c CadernoInventario) MarshalJSON() ([]byte, error) {
	type caderno CadernoInventario
	return json.Marshal(struct {
		caderno
		Situacao string `json:"status"`
	}{caderno(c), c.Situacao()})
}

// DividirEmCadernos reparte as linhas, já ordenadas, respeitando o limite por caderno
// e sem misturar almoxarifados: cada caderno vai para uma equipe num lugar só.
func DividirEmCadernos(almoxarifados []int64, linhasPorCaderno int) []int {
	out := make([]int, len(almoxarifados))
	numero, naPagina := 0, 0
	for i, a := range almoxarifados {
		if i == 0 || naPagina >= linhasPorCaderno || a != almoxarifados[i-1] {
			numero++
			naPagina = 0
		}
		naPagina++
		out[i] = numero
	}
	return out
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
