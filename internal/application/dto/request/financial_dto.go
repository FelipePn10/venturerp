package request

type CreateContaBancariaDTO struct {
	Banco        string  `json:"banco"`
	Agencia      string  `json:"agencia"`
	Conta        string  `json:"conta"`
	Digito       *string `json:"digito,omitempty"`
	Descricao    string  `json:"descricao"`
	Titular      *string `json:"titular,omitempty"`
	SaldoInicial float64 `json:"saldo_inicial"`
	ChavePix     *string `json:"chave_pix,omitempty"`
	TipoChavePix *string `json:"tipo_chave_pix,omitempty"`
}

type CreateCondicaoPagamentoDTO struct {
	Nome     string `json:"nome"`
	Parcelas string `json:"parcelas"`
}

type CreatePlanoContasDTO struct {
	Codigo     string  `json:"codigo"`
	Descricao  string  `json:"descricao"`
	Tipo       string  `json:"tipo"`
	Natureza   string  `json:"natureza"`
	ParentCode *string `json:"parent_code,omitempty"`
	Nivel      int32   `json:"nivel"`
}

type CreateCentroCustoDTO struct {
	Codigo    string `json:"codigo"`
	Descricao string `json:"descricao"`
	Tipo      string `json:"tipo"`
}

type CreateContaPagarDTO struct {
	NumeroDocumento string  `json:"numero_documento"`
	TipoDocumento   string  `json:"tipo_documento"`
	FornecedorID    *int64  `json:"fornecedor_id,omitempty"`
	FiscalEntryID   *int64  `json:"fiscal_entry_id,omitempty"`
	PurchaseOrderID *int64  `json:"purchase_order_id,omitempty"`
	DataEmissao     string  `json:"data_emissao"`
	DataVencimento  string  `json:"data_vencimento"`
	ValorBruto      float64 `json:"valor_bruto"`
	Desconto        float64 `json:"desconto"`
	ParcelaNumero   int32   `json:"parcela_numero"`
	ParcelaTotal    int32   `json:"parcela_total"`
	FormaPagamento  *string `json:"forma_pagamento,omitempty"`
	PlanoContasID   *int64  `json:"plano_contas_id,omitempty"`
	CentroCustoID   *int64  `json:"centro_custo_id,omitempty"`
	Observacao      *string `json:"observacao,omitempty"`
}

type ListContasPagarFilter struct {
	Status       *string `json:"status,omitempty"`
	FornecedorID *int64  `json:"fornecedor_id,omitempty"`
	StartDate    *string `json:"start_date,omitempty"`
	EndDate      *string `json:"end_date,omitempty"`
}

type ApproveContaPagarDTO struct {
	MotivoRejeicao *string `json:"motivo_rejeicao,omitempty"`
}

type BaixarContaPagarDTO struct {
	ContaBancariaID int64   `json:"conta_bancaria_id"`
	ValorPago       float64 `json:"valor_pago"`
	DataPagamento   string  `json:"data_pagamento"`
	Observacao      *string `json:"observacao,omitempty"`
}

type CreateContaReceberDTO struct {
	NumeroDocumento *string `json:"numero_documento,omitempty"`
	ClienteID       *int64  `json:"cliente_id,omitempty"`
	FiscalExitID    *int64  `json:"fiscal_exit_id,omitempty"`
	SalesOrderID    *int64  `json:"sales_order_id,omitempty"`
	DataEmissao     string  `json:"data_emissao"`
	DataVencimento  string  `json:"data_vencimento"`
	ValorBruto      float64 `json:"valor_bruto"`
	Desconto        float64 `json:"desconto"`
	ParcelaNumero   int32   `json:"parcela_numero"`
	ParcelaTotal    int32   `json:"parcela_total"`
	FormaPagamento  *string `json:"forma_pagamento,omitempty"`
	PlanoContasID   *int64  `json:"plano_contas_id,omitempty"`
	CentroCustoID   *int64  `json:"centro_custo_id,omitempty"`
	Observacao      *string `json:"observacao,omitempty"`
}

type ListContasReceberFilter struct {
	Status    *string `json:"status,omitempty"`
	ClienteID *int64  `json:"cliente_id,omitempty"`
	StartDate *string `json:"start_date,omitempty"`
	EndDate   *string `json:"end_date,omitempty"`
}

type BaixarContaReceberDTO struct {
	ContaBancariaID int64   `json:"conta_bancaria_id"`
	ValorRecebido   float64 `json:"valor_recebido"`
	DataRecebimento string  `json:"data_recebimento"`
	Observacao      *string `json:"observacao,omitempty"`
}

type GetFluxoCaixaDTO struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type GetFluxoProjetadoDTO struct {
	StartDate string `json:"start_date"`
}

type ApurarImpostosDTO struct {
	Competencia string `json:"competencia"`
}

// CreateAdiantamentoDTO registers an advance payment (to a supplier) or advance
// receipt (from a customer). The cash movement happens immediately.
type CreateAdiantamentoDTO struct {
	Tipo             string  `json:"tipo"` // PAGAR | RECEBER
	ParceiroID       *int64  `json:"parceiro_id,omitempty"`
	ContaBancariaID  int64   `json:"conta_bancaria_id"`
	NumeroDocumento  *string `json:"numero_documento,omitempty"`
	DataAdiantamento string  `json:"data_adiantamento"`
	ValorOriginal    float64 `json:"valor_original"`
	Descricao        *string `json:"descricao,omitempty"`
}

// AplicarAdiantamentoDTO applies an advance balance onto a conta a pagar / receber.
type AplicarAdiantamentoDTO struct {
	ContaTipo     string  `json:"conta_tipo"` // PAGAR | RECEBER
	ContaID       int64   `json:"conta_id"`
	Valor         float64 `json:"valor"`
	DataAplicacao *string `json:"data_aplicacao,omitempty"`
}
