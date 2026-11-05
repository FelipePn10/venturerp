package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// Direção da consignação.
const (
	// ConsignacaoSaida: mercadoria nossa com o cliente; segue no nosso estoque.
	ConsignacaoSaida = "SAIDA"
	// ConsignacaoEntrada: material do fornecedor conosco; ainda não é estoque nosso.
	ConsignacaoEntrada = "ENTRADA"
)

// Como a consignação baixa.
const (
	BaixaConsignacaoVenda     = "VENDA"
	BaixaConsignacaoConsumo   = "CONSUMO"
	BaixaConsignacaoDevolucao = "DEVOLUCAO"
)

// Situações da consignação.
const (
	ConsignacaoAberta    = "ABERTA"
	ConsignacaoEncerrada = "ENCERRADA"
	ConsignacaoCancelada = "CANCELADA"
)

// ReferenceTypeConsignment marca os movimentos da consignação.
const ReferenceTypeConsignment = "CONSIGNACAO"

// ReferenceTypeCNPJTransfer marca os movimentos da transferência entre CNPJs.
const ReferenceTypeCNPJTransfer = "TRANSF_CNPJ"

// Consignacao é uma remessa ou um recebimento em consignação.
type Consignacao struct {
	ID          int64           `json:"id"`
	Direcao     string          `json:"direction"`
	Parceiro    int64           `json:"partner_code"`
	NomeParte   string          `json:"partner_name"`
	ItemCode    int64           `json:"item_code"`
	Codigo      string          `json:"business_code"`
	Nome        string          `json:"item_name"`
	Mask        string          `json:"mask"`
	Lote        *string         `json:"lot,omitempty"`
	AlmoxOrigem *int64          `json:"origin_warehouse_id,omitempty"`
	Almox       *int64          `json:"warehouse_id,omitempty"`
	Quantidade  decimal.Decimal `json:"quantity"`
	Baixada     decimal.Decimal `json:"settled_qty"`
	Devolvida   decimal.Decimal `json:"returned_qty"`
	Valor       decimal.Decimal `json:"unit_value"`
	NotaSaida   *int64          `json:"fiscal_exit_id,omitempty"`
	Documento   *string         `json:"document,omitempty"`
	Emissao     time.Time       `json:"issue_date"`
	Prazo       *time.Time      `json:"due_date,omitempty"`
	Status      string          `json:"status"`
	Notas       *string         `json:"notes,omitempty"`
	CreatedBy   uuid.UUID       `json:"-"`
}

// Saldo é o que ainda está em consignação.
func (c Consignacao) Saldo() decimal.Decimal {
	return c.Quantidade.Sub(c.Baixada).Sub(c.Devolvida)
}

// Conferir valida a baixa antes de mexer no saldo.
func (c Consignacao) Conferir(q decimal.Decimal) error {
	if c.Status != ConsignacaoAberta {
		return domainerr.NewValidation("esta consignação não está aberta")
	}
	if !q.IsPositive() {
		return domainerr.NewValidation("informe a quantidade")
	}
	if q.GreaterThan(c.Saldo()) {
		return domainerr.NewValidation("a quantidade passa do saldo em consignação (" + QuantidadeBR(c.Saldo()) + ")")
	}
	return nil
}

// RemessaConsignada é uma linha da NF-e de remessa em consignação.
type RemessaConsignada struct {
	NotaSaida     int64
	Cliente       int64
	ItemCode      int64
	Mask          string
	Lote          *string
	Endereco      *string
	AlmoxOrigem   int64
	Quantidade    decimal.Decimal
	ValorUnitario decimal.Decimal
	Documento     string
	Emissao       time.Time
	Ator          uuid.UUID
}

// VendaConsignada é uma linha da NF-e que vende o que estava consignado.
type VendaConsignada struct {
	NotaSaida  int64
	Cliente    int64
	ItemCode   int64
	Mask       string
	Quantidade decimal.Decimal
	Documento  string
	Emissao    time.Time
	Ator       uuid.UUID
}

// BaixaConsignacao é a devolução ou o consumo pedido na tela.
type BaixaConsignacao struct {
	ID         int64
	Quantidade decimal.Decimal
	Almox      int64 // consumo: onde entra. Devolução de saída: 0 volta à origem.
	Documento  *string
	Ator       uuid.UUID
}

// LinhaEntreCNPJs é uma linha da NF-e de transferência a outro estabelecimento.
type LinhaEntreCNPJs struct {
	ItemCode   int64
	Mask       string
	Lote       *string
	Endereco   *string
	AlmoxDe    int64
	Quantidade decimal.Decimal
}

// EnvioEntreCNPJs é a NF-e de transferência autorizada.
type EnvioEntreCNPJs struct {
	NotaSaida int64
	Destino   int64
	Chave     string
	Linhas    []LinhaEntreCNPJs
	Ator      uuid.UUID
}

// TransferenciaEntreCNPJs é o envio visto pela origem ou pelo destino.
type TransferenciaEntreCNPJs struct {
	ID         int64      `json:"id"`
	Origem     int64      `json:"origin_enterprise_id"`
	NomeOrigem string     `json:"origin_name"`
	Destino    int64      `json:"destination_enterprise_id"`
	NomeDest   string     `json:"destination_name"`
	NotaSaida  int64      `json:"fiscal_exit_id"`
	Chave      string     `json:"nfe_key"`
	Status     string     `json:"status"`
	Entrada    *int64     `json:"fiscal_entry_id,omitempty"`
	RecebidaEm *time.Time `json:"received_at,omitempty"`
	CriadaEm   time.Time  `json:"created_at"`
}

// Situações da transferência entre CNPJs.
const (
	TransferenciaEmTransito = "EM_TRANSITO"
	TransferenciaRecebida   = "RECEBIDA"
	TransferenciaCancelada  = "CANCELADA"
)

// MesmaRaizDeCNPJ diz se os dois CNPJs são do mesmo titular. Transferência é só entre
// estabelecimentos da mesma pessoa jurídica: entre empresas diferentes é venda.
func MesmaRaizDeCNPJ(a, b string) bool {
	return len(a) == 14 && len(b) == 14 && a != b && a[:8] == b[:8]
}
