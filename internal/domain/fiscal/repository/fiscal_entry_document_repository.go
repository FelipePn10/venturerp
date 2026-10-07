package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

// FiscalEntryDocumentRepository é a nota de entrada como documento completo:
// cabeçalho, itens conciliados e classificados, parcelas e a distribuição de
// cada parcela por plano de contas. Toda gravação aqui é transacional — nota
// meio gravada (itens sem cabeçalho, parcela sem rateio, título sem estoque)
// é exatamente o que a conferência existe para impedir.
type FiscalEntryDocumentRepository interface {
	// FindEntryByChave devolve a nota ativa (não cancelada) da empresa com a chave, ou nil.
	FindEntryByChave(ctx context.Context, chave string) (*entity.FiscalEntry, error)
	// CreateEntryDocument grava cabeçalho, itens e parcelas numa transação. A
	// mesma chave importada ao mesmo tempo por duas pessoas entra uma vez só.
	CreateEntryDocument(ctx context.Context, e *entity.FiscalEntry) (*entity.FiscalEntry, error)
	// GetEntryXML devolve o XML original guardado (vazio quando a nota foi
	// lançada à mão).
	GetEntryXML(ctx context.Context, id int64) (numero int64, xml string, err error)
	// GetEntryDocument lê a nota com itens (com os nomes do cadastro) e parcelas.
	GetEntryDocument(ctx context.Context, id int64) (*entity.FiscalEntry, error)
	// SaveConciliation grava a conciliação/classificação dos itens, os vínculos
	// produto × fornecedor a memorizar e, quando parcelas != nil, substitui as
	// parcelas e a distribuição. Tudo ou nada.
	SaveConciliation(ctx context.Context, entryID int64, cab HeaderConciliation, itens []ItemConciliation, vinculos []SupplierItemLink, parcelas []*entity.FiscalEntryInstallment, status entity.FiscalEntryStatus) error
	// AprovarEntrada efetiva a nota numa transação só: títulos do contas a
	// pagar (fornecedor e retenções) com rateio, entrada no estoque, baixa e
	// faturamento do pedido de compra, lançamentos contábeis e a situação.
	// Recusa nota que não esteja pendente/conferida (aprovar duas vezes
	// duplicaria tudo).
	AprovarEntrada(ctx context.Context, a AprovacaoEntrada) (*ResultadoAprovacao, error)
	// CancelarEntrada cancela a nota. Aprovada, estorna tudo o que a aprovação
	// fez — desde que nenhum título tenha sido pago e o material ainda esteja
	// no estoque.
	CancelarEntrada(ctx context.Context, c CancelamentoEntrada) (*ResultadoCancelamento, error)

	// Apoio à conciliação.
	FindSupplierByDocument(ctx context.Context, document string) (*SupplierRef, error)
	// SupplierByCode devolve o fornecedor da empresa pelo código (situação e
	// bloqueio), ou nil.
	SupplierByCode(ctx context.Context, code int64) (*SupplierRef, error)
	// LinkSupplierToPendingEntries liga o fornecedor às notas ainda não
	// aprovadas da empresa emitidas pelo CNPJ e sem fornecedor; devolve quantas.
	LinkSupplierToPendingEntries(ctx context.Context, cnpj string, code int64) (int64, error)
	ResolveByEAN(ctx context.Context, supplierCode *int64, ean string) (*ItemMatch, error)
	LastAccountForItem(ctx context.Context, itemCode int64) (*ItemAccount, error)
	PlanoContasByCodigo(ctx context.Context, codigo string) (*int64, error)
	SearchItems(ctx context.Context, termos []string, limite int) ([]ItemCandidate, error)
	ItemsByCode(ctx context.Context, codes []int64) (map[int64]ItemCandidate, error)
	ExistingPlanos(ctx context.Context, ids []int64) (map[int64]bool, error)
	ExistingCentros(ctx context.Context, ids []int64) (map[int64]bool, error)
	ExistingWarehouses(ctx context.Context, ids []int64) (map[int64]string, error)
	SupplierLinksForEntry(ctx context.Context, supplierCode int64) ([]ItemCandidate, error)
	EntryOperations(ctx context.Context, codes []int64) (map[int64]EntryOperation, error)
	// PurchaseOrderLines lista as linhas do pedido de compra do fornecedor:
	// por referência (número/código do pedido que vem no XML) ou pelos códigos
	// das linhas. Linhas de outra empresa nunca voltam.
	PurchaseOrderLines(ctx context.Context, supplierCode int64, refs []string, lineCodes []int64) ([]PurchaseOrderLine, error)
	// OpenPurchaseOrderLines lista as linhas do fornecedor para o item ainda
	// com saldo a faturar (para vincular a nota ao pedido na conferência).
	OpenPurchaseOrderLines(ctx context.Context, supplierCode, itemCode int64) ([]PurchaseOrderLine, error)
	NCMIPIRates(ctx context.Context, ncms []string) (map[string]decimal.Decimal, error)
	// Contabilização.
	AccountingParams(ctx context.Context) (*AccountingPostingParams, error)
	SaveAccountingParams(ctx context.Context, p *AccountingPostingParams) error
	PlanoContasAccounts(ctx context.Context, planoIDs []int64) (map[int64]int64, error)
	ValidAccountingAccounts(ctx context.Context, planID int64, ids []int64) (map[int64]bool, error)
	// VincularPlanoContaContabil liga (ou desliga, nil) o plano de contas
	// gerencial à conta contábil.
	VincularPlanoContaContabil(ctx context.Context, planoID int64, contaID *int64) error
	AccountingPlanExists(ctx context.Context, planID int64) (bool, error)
	// ContaContabilDoBanco: a conta contábil ligada à conta bancária (nil se não ligada).
	ContaContabilDoBanco(ctx context.Context, contaBancariaID int64) (*int64, error)
	// VincularContaBancariaContabil liga (ou desliga, com nil) a conta bancária à conta contábil.
	VincularContaBancariaContabil(ctx context.Context, contaBancariaID int64, contaID *int64) error
	// Contabilizado diz se a origem já tem lançamentos contábeis na empresa.
	// RetencaoTipo devolve o imposto do título de retenção (vazio nos demais).
	RetencaoTipo(ctx context.Context, contaPagarID int64) (string, error)
	Contabilizado(ctx context.Context, origem string, id int64) (bool, error)
}

// HeaderConciliation são as escolhas do cabeçalho na conferência.
type HeaderConciliation struct {
	EntryOperationCode *int64
	PurchaseOrderCode  *int64
}

// ItemConciliation é o que o usuário decidiu para um item da nota.
type ItemConciliation struct {
	ItemID                int64
	ItemCode              *int64
	ItemSupplierID        *int64
	ResolutionStrategy    *string
	PlanoContasID         *int64
	CentroCustoID         *int64
	FatorConversao        *decimal.Decimal
	QuantidadeEstoque     *decimal.Decimal
	CfopEntrada           *string
	EntryOperationCode    *int64
	MovimentaEstoque      bool
	GeraFinanceiro        bool
	WarehouseID           *int64
	PurchaseOrderCode     *int64
	PurchaseOrderItemCode *int64
	GeraCreditoICMS       bool
	GeraCreditoIPI        bool
	GeraCreditoPIS        bool
	GeraCreditoCOFINS     bool
	GeraCreditoIBSCBS     bool
}

// SupplierItemLink é o vínculo produto × fornecedor (o "de/para") a memorizar
// para a próxima nota do mesmo fornecedor vir conciliada sozinha.
type SupplierItemLink struct {
	SupplierCode        int64
	ItemCode            int64
	SupplierItemCode    string
	SupplierDescription string
	XMLUOM              *string
	ConversionFactor    *decimal.Decimal
	Barcode             *string
	CreatedBy           uuid.UUID
}

// TituloAPagar é um título do contas a pagar nascido da nota: uma parcela do
// fornecedor ou o imposto retido a recolher.
type TituloAPagar struct {
	InstallmentID   int64
	TipoDocumento   string // NFE ou RETENCAO
	RetencaoTipo    string // IRRF, PIS, COFINS, CSLL, INSS, ISS (só RETENCAO)
	NumeroDocumento string
	FornecedorCode  *int64
	FornecedorCNPJ  string
	DataEmissao     time.Time
	DataVencimento  time.Time
	Valor           decimal.Decimal
	ParcelaNumero   int
	ParcelaTotal    int
	FormaPagamento  *string
	PlanoContasID   *int64
	CentroCustoID   *int64
	Observacao      *string
	Rateio          []entity.InstallmentAllocation
}

// MovimentoEntrada é o que um item da nota faz no estoque e no pedido.
type MovimentoEntrada struct {
	ItemID           int64
	ItemCode         int64
	WarehouseID      int64
	Quantidade       decimal.Decimal // na unidade de estoque
	CustoTotal       decimal.Decimal // custo de aquisição da linha toda
	MovimentaEstoque bool
	// Pedido de compra (3-way): a parte já recebida fisicamente não entra de
	// novo no estoque; tudo é somado ao faturado da linha.
	PurchaseOrderItemCode *int64
	FatorPedido           decimal.Decimal // unidades de estoque por unidade do pedido
}

// LancamentoContabil é uma partida (débito/crédito) gerada pela nota.
type LancamentoContabil struct {
	PlanID    int64
	DebitoID  int64
	CreditoID int64
	DebitoCC  *int64
	CreditoCC *int64
	Valor     decimal.Decimal
	Historico string
}

type AprovacaoEntrada struct {
	EntryID     int64
	UserID      uuid.UUID
	Titulos     []TituloAPagar
	Movimentos  []MovimentoEntrada
	Lancamentos []LancamentoContabil
	Custos      map[int64]decimal.Decimal // custo de aquisição por item da nota
}

type ResultadoAprovacao struct {
	TitulosGerados     int
	MovimentosGerados  int
	QtdJaRecebida      decimal.Decimal
	LancamentosGerados int
	Movimentados       []MovimentoGerado
}

// MovimentoGerado é a entrada no estoque que a aprovação fez para um item.
type MovimentoGerado struct {
	ItemID                int64
	ItemCode              int64
	WarehouseID           int64
	Quantidade            decimal.Decimal
	PurchaseOrderCode     *int64
	PurchaseOrderItemCode *int64
}

type CancelamentoEntrada struct {
	EntryID int64
	UserID  uuid.UUID
	Motivo  string
}

type ResultadoCancelamento struct {
	EraAprovada        bool
	TitulosCancelados  int
	MovimentosEstorno  int
	LancamentosEstorno int
	DataEntrada        time.Time
}

type SupplierRef struct {
	Code             int64
	Name             string
	IsActive         bool
	Blocked          bool
	FinancialAccount *string
}

type ItemMatch struct {
	ItemCode       int64
	ItemSupplierID *int64
}

type ItemAccount struct {
	PlanoContasID int64
	CentroCustoID *int64
}

// ItemCandidate é um item do cadastro oferecido na conciliação.
type ItemCandidate struct {
	Code               int64
	Name               string
	UOM                string
	NCM                string
	SupplierItemCode   string
	ItemSupplierID     *int64
	IsActive           bool
	Origem             string // origem da mercadoria (0 nacional, 1 importação direta…)
	DefaultWarehouseID *int64
	// Do vínculo produto × fornecedor (quando a busca parte dele).
	SupplierDescription string
	XMLUOM              *string
	ConversionFactor    *decimal.Decimal
	Barcode             string
}

// EntryOperation é o tipo de operação de entrada (o "TES").
type EntryOperation struct {
	Code             int64
	Description      string
	Natureza         string
	MovimentaEstoque bool
	GeraFinanceiro   bool
	CreditaICMS      bool
	CreditaIPI       bool
	CreditaPISCOFINS bool
	IsActive         bool
}

// PurchaseOrderLine é a linha do pedido de compra vista pela nota de entrada.
type PurchaseOrderLine struct {
	Code              int64
	PurchaseOrderCode int64
	OrderNumber       int64
	Sequence          int
	ItemCode          int64
	Status            string
	RequestedQty      decimal.Decimal // unidade do pedido
	ReceivedQty       decimal.Decimal
	InvoicedQty       decimal.Decimal
	CancelledQty      decimal.Decimal
	InternalQty       decimal.Decimal // total na unidade de estoque (0 = mesma unidade)
	UnitPrice         decimal.Decimal // por unidade do pedido
	InternalPrice     decimal.Decimal
	TolerancePct      decimal.Decimal
	WarehouseID       *int64
	AccountingAccount *string
	OperationCode     *int64
	// OrderStatus é a situação da capa: só pedido aprovado recebe material.
	OrderStatus string
}

// FatorEstoque é quantas unidades de estoque há em uma unidade do pedido.
func (l PurchaseOrderLine) FatorEstoque() decimal.Decimal {
	if l.InternalQty.IsPositive() && l.RequestedQty.IsPositive() {
		return l.InternalQty.Div(l.RequestedQty)
	}
	return decimal.NewFromInt(1)
}

// AccountingPostingParams são as contas da contabilização automática.
type AccountingPostingParams struct {
	PlanID                       int64      `json:"plan_id"`
	FornecedoresAccountID        int64      `json:"fornecedores_account_id"`
	ICMSRecuperarAccountID       *int64     `json:"icms_recuperar_account_id,omitempty"`
	IPIRecuperarAccountID        *int64     `json:"ipi_recuperar_account_id,omitempty"`
	PISRecuperarAccountID        *int64     `json:"pis_recuperar_account_id,omitempty"`
	COFINSRecuperarAccountID     *int64     `json:"cofins_recuperar_account_id,omitempty"`
	IBSRecuperarAccountID        *int64     `json:"ibs_recuperar_account_id,omitempty"`
	CBSRecuperarAccountID        *int64     `json:"cbs_recuperar_account_id,omitempty"`
	IRRFRecolherAccountID        *int64     `json:"irrf_recolher_account_id,omitempty"`
	PCCRecolherAccountID         *int64     `json:"pcc_recolher_account_id,omitempty"`
	INSSRecolherAccountID        *int64     `json:"inss_recolher_account_id,omitempty"`
	ISSRecolherAccountID         *int64     `json:"iss_recolher_account_id,omitempty"`
	DespesaPadraoAccountID       *int64     `json:"despesa_padrao_account_id,omitempty"`
	ContabilizarEntrada          bool       `json:"contabilizar_entrada"`
	ContabilizarPagamentos       bool       `json:"contabilizar_pagamentos"`
	ContabilizarRecebimentos     bool       `json:"contabilizar_recebimentos"`
	ContabilizarSaidas           bool       `json:"contabilizar_saidas"`
	BancoPadraoAccountID         *int64     `json:"banco_padrao_account_id,omitempty"`
	JurosPagosAccountID          *int64     `json:"juros_pagos_account_id,omitempty"`
	DescontosObtidosAccountID    *int64     `json:"descontos_obtidos_account_id,omitempty"`
	ClientesAccountID            *int64     `json:"clientes_account_id,omitempty"`
	JurosRecebidosAccountID      *int64     `json:"juros_recebidos_account_id,omitempty"`
	DescontosConcedidosAccountID *int64     `json:"descontos_concedidos_account_id,omitempty"`
	ReceitaVendasAccountID       *int64     `json:"receita_vendas_account_id,omitempty"`
	ICMSVendasAccountID          *int64     `json:"icms_vendas_account_id,omitempty"`
	ICMSRecolherAccountID        *int64     `json:"icms_recolher_account_id,omitempty"`
	ICMSSTRecolherAccountID      *int64     `json:"icms_st_recolher_account_id,omitempty"`
	IPIRecolherAccountID         *int64     `json:"ipi_recolher_account_id,omitempty"`
	PISVendasAccountID           *int64     `json:"pis_vendas_account_id,omitempty"`
	PISRecolherAccountID         *int64     `json:"pis_recolher_account_id,omitempty"`
	COFINSVendasAccountID        *int64     `json:"cofins_vendas_account_id,omitempty"`
	COFINSRecolherAccountID      *int64     `json:"cofins_recolher_account_id,omitempty"`
	CMVAccountID                 *int64     `json:"cmv_account_id,omitempty"`
	EstoqueAccountID             *int64     `json:"estoque_account_id,omitempty"`
	UpdatedBy                    *uuid.UUID `json:"-"`
}
