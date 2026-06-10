package response

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CondicaoPagamentoResponse is the API representation of a payment condition.
type CondicaoPagamentoResponse struct {
	ID        int64           `json:"id"`
	Nome      string          `json:"nome"`
	Parcelas  json.RawMessage `json:"parcelas,omitempty"`
	Ativo     bool            `json:"ativo"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// PlanoContasResponse is the API representation of a chart-of-accounts entry.
type PlanoContasResponse struct {
	ID         int64     `json:"id"`
	Codigo     string    `json:"codigo"`
	Descricao  string    `json:"descricao"`
	Tipo       string    `json:"tipo"`
	Natureza   string    `json:"natureza"`
	ParentCode *string   `json:"parent_code,omitempty"`
	Nivel      int32     `json:"nivel"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// CentroCustoResponse is the API representation of a cost center (financial).
type CentroCustoResponse struct {
	ID        int64     `json:"id"`
	Codigo    string    `json:"codigo"`
	Descricao string    `json:"descricao"`
	Tipo      string    `json:"tipo"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// ContaBancariaResponse is the API representation of a bank account.
type ContaBancariaResponse struct {
	ID           int64           `json:"id"`
	Banco        string          `json:"banco"`
	Agencia      string          `json:"agencia"`
	Conta        string          `json:"conta"`
	Digito       *string         `json:"digito,omitempty"`
	Descricao    string          `json:"descricao"`
	Titular      *string         `json:"titular,omitempty"`
	SaldoInicial decimal.Decimal `json:"saldo_inicial"`
	ChavePix     *string         `json:"chave_pix,omitempty"`
	TipoChavePix *string         `json:"tipo_chave_pix,omitempty"`
	IsActive     bool            `json:"is_active"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	CreatedBy    uuid.UUID       `json:"created_by"`
}

// ContaPagarResponse is the API representation of an account payable.
type ContaPagarResponse struct {
	ID              int64      `json:"id"`
	NumeroDocumento string     `json:"numero_documento"`
	TipoDocumento   string     `json:"tipo_documento"`
	FornecedorID    *int64     `json:"fornecedor_id,omitempty"`
	FiscalEntryID   *int64     `json:"fiscal_entry_id,omitempty"`
	PurchaseOrderID *int64     `json:"purchase_order_id,omitempty"`
	DataLancamento  time.Time  `json:"data_lancamento"`
	DataEmissao     time.Time  `json:"data_emissao"`
	DataVencimento  time.Time  `json:"data_vencimento"`
	DataPagamento   *time.Time `json:"data_pagamento,omitempty"`

	ValorBruto decimal.Decimal `json:"valor_bruto"`
	Desconto   decimal.Decimal `json:"desconto"`
	Juros      decimal.Decimal `json:"juros"`
	Multa      decimal.Decimal `json:"multa"`
	ValorPago  decimal.Decimal `json:"valor_pago"`

	ParcelaNumero int32  `json:"parcela_numero"`
	ParcelaTotal  int32  `json:"parcela_total"`
	ParcelaPaiID  *int64 `json:"parcela_pai_id,omitempty"`

	ContaBancariaID *int64  `json:"conta_bancaria_id,omitempty"`
	FormaPagamento  *string `json:"forma_pagamento,omitempty"`
	PlanoContasID   *int64  `json:"plano_contas_id,omitempty"`
	CentroCustoID   *int64  `json:"centro_custo_id,omitempty"`

	StatusAprovacao string     `json:"status_aprovacao"`
	AprovadoPor     *uuid.UUID `json:"aprovado_por,omitempty"`
	DataAprovacao   *time.Time `json:"data_aprovacao,omitempty"`
	MotivoRejeicao  *string    `json:"motivo_rejeicao,omitempty"`

	Status                   string          `json:"status"`
	AdiantamentoID           *int64          `json:"adiantamento_id,omitempty"`
	ValorAdiantamentoAbatido decimal.Decimal `json:"valor_adiantamento_abatido"`

	ComprovantePath *string `json:"comprovante_path,omitempty"`
	Observacao      *string `json:"observacao,omitempty"`

	IsActive   bool       `json:"is_active"`
	CriadoPor  uuid.UUID  `json:"criado_por"`
	BaixadoPor *uuid.UUID `json:"baixado_por,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// ContaReceberResponse is the API representation of an account receivable.
type ContaReceberResponse struct {
	ID              int64      `json:"id"`
	NumeroDocumento *string    `json:"numero_documento,omitempty"`
	ClienteID       *int64     `json:"cliente_id,omitempty"`
	FiscalExitID    *int64     `json:"fiscal_exit_id,omitempty"`
	SalesOrderID    *int64     `json:"sales_order_id,omitempty"`
	DataLancamento  time.Time  `json:"data_lancamento"`
	DataEmissao     time.Time  `json:"data_emissao"`
	DataVencimento  time.Time  `json:"data_vencimento"`
	DataRecebimento *time.Time `json:"data_recebimento,omitempty"`

	ValorBruto    decimal.Decimal `json:"valor_bruto"`
	Desconto      decimal.Decimal `json:"desconto"`
	Juros         decimal.Decimal `json:"juros"`
	Multa         decimal.Decimal `json:"multa"`
	ValorRecebido decimal.Decimal `json:"valor_recebido"`

	ParcelaNumero int32  `json:"parcela_numero"`
	ParcelaTotal  int32  `json:"parcela_total"`
	ParcelaPaiID  *int64 `json:"parcela_pai_id,omitempty"`

	ContaBancariaID *int64  `json:"conta_bancaria_id,omitempty"`
	FormaPagamento  *string `json:"forma_pagamento,omitempty"`

	NossoNumero    *string `json:"nosso_numero,omitempty"`
	LinhaDigitavel *string `json:"linha_digitavel,omitempty"`
	CodigoBarras   *string `json:"codigo_barras,omitempty"`
	ChavePixGerada *string `json:"chave_pix_gerada,omitempty"`

	PlanoContasID *int64 `json:"plano_contas_id,omitempty"`
	CentroCustoID *int64 `json:"centro_custo_id,omitempty"`

	Status     string `json:"status"`
	EmProtesto bool   `json:"em_protesto"`

	IsActive   bool       `json:"is_active"`
	CriadoPor  uuid.UUID  `json:"criado_por"`
	BaixadoPor *uuid.UUID `json:"baixado_por,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// TaxAssessmentResponse is the API representation of a tax assessment (apuração).
type TaxAssessmentResponse struct {
	ID             int64           `json:"id"`
	Imposto        string          `json:"imposto"`
	Competencia    string          `json:"competencia"`
	Debitos        decimal.Decimal `json:"debitos"`
	Creditos       decimal.Decimal `json:"creditos"`
	SaldoDevedor   decimal.Decimal `json:"saldo_devedor"`
	SaldoCredor    decimal.Decimal `json:"saldo_credor"`
	Status         string          `json:"status"`
	CpID           *int64          `json:"cp_id,omitempty"`
	DataVencimento *time.Time      `json:"data_vencimento,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// FluxoCaixaResponse is the API representation of a cash-flow entry.
type FluxoCaixaResponse struct {
	ID                     int64           `json:"id"`
	Data                   time.Time       `json:"data"`
	Tipo                   string          `json:"tipo"`
	Valor                  decimal.Decimal `json:"valor"`
	ContaBancariaID        *int64          `json:"conta_bancaria_id,omitempty"`
	ContaBancariaDestinoID *int64          `json:"conta_bancaria_destino_id,omitempty"`
	ContasPagarID          *int64          `json:"contas_pagar_id,omitempty"`
	ContasReceberID        *int64          `json:"contas_receber_id,omitempty"`
	Descricao              *string         `json:"descricao,omitempty"`
	Conciliado             bool            `json:"conciliado"`
	ExtratoHash            *string         `json:"extrato_hash,omitempty"`
	CreatedAt              time.Time       `json:"created_at"`
}

// AdiantamentoResponse is the API representation of an advance (adiantamento).
type AdiantamentoResponse struct {
	ID               int64           `json:"id"`
	Tipo             string          `json:"tipo"`
	ParceiroID       *int64          `json:"parceiro_id,omitempty"`
	ContaBancariaID  int64           `json:"conta_bancaria_id"`
	NumeroDocumento  *string         `json:"numero_documento,omitempty"`
	DataAdiantamento time.Time       `json:"data_adiantamento"`
	ValorOriginal    decimal.Decimal `json:"valor_original"`
	ValorUtilizado   decimal.Decimal `json:"valor_utilizado"`
	Saldo            decimal.Decimal `json:"saldo"`
	Status           string          `json:"status"`
	Descricao        *string         `json:"descricao,omitempty"`
	IsActive         bool            `json:"is_active"`
	CreatedBy        uuid.UUID       `json:"created_by"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// AdiantamentoAplicacaoResponse is the API representation of an advance application.
type AdiantamentoAplicacaoResponse struct {
	ID             int64           `json:"id"`
	AdiantamentoID int64           `json:"adiantamento_id"`
	ContaTipo      string          `json:"conta_tipo"`
	ContaID        int64           `json:"conta_id"`
	ValorAplicado  decimal.Decimal `json:"valor_aplicado"`
	DataAplicacao  time.Time       `json:"data_aplicacao"`
	CreatedBy      uuid.UUID       `json:"created_by"`
	CreatedAt      time.Time       `json:"created_at"`
}
