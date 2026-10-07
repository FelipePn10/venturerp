package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type FiscalEntryStatus string

const (
	EntryStatusPending    FiscalEntryStatus = "PENDING"
	EntryStatusConferred  FiscalEntryStatus = "CONFERRED"
	EntryStatusApproved   FiscalEntryStatus = "APPROVED"
	EntryStatusWrittenOff FiscalEntryStatus = "WRITTEN_OFF"
	EntryStatusCancelled  FiscalEntryStatus = "CANCELLED"
)

type FiscalEntry struct {
	ID                  int64
	EnterpriseID        int64
	ChaveAcesso         *string
	NumeroNF            int64
	Serie               string
	Modelo              string
	DataEmissao         time.Time
	DataEntrada         time.Time
	CnpjEmitente        string
	RazaoSocialEmitente string
	IEEmitente          *string
	UFEmitente          *string
	ValorProdutos       float64
	ValorFrete          float64
	ValorSeguro         float64
	ValorDesconto       float64
	ValorIPI            float64
	ValorICMS           float64
	ValorPIS            float64
	ValorCOFINS         float64
	ValorTotal          float64
	TipoDocumento       string
	PurchaseOrderCode   *int64
	SupplierCode        *int64
	CteCode             *int64
	Status              FiscalEntryStatus
	XmlPath             *string
	Notes               *string
	IsActive            bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
	CreatedBy           uuid.UUID
	Itens               []*FiscalEntryItem
	Warnings            []string

	// Dados completos do XML (migração 000376).
	NaturezaOperacao          *string
	CnpjDestinatario          *string
	Protocolo                 *string
	ValorICMSST               decimal.Decimal
	ValorOutras               decimal.Decimal
	ModalidadeFrete           *string
	InformacoesComplementares *string
	XMLContent                *string
	SemPagamento              bool
	ApprovedAt                *time.Time
	ApprovedBy                *uuid.UUID
	// SupplierName é o nome do fornecedor do cadastro (leitura).
	SupplierName *string
	Parcelas     []*FiscalEntryInstallment

	// Migração 000377.
	EntryOperationCode *int64
	BaseIBSCBS         decimal.Decimal
	ValorIBS           decimal.Decimal
	ValorCBS           decimal.Decimal
	ValorIS            decimal.Decimal
	ValorRetPIS        decimal.Decimal
	ValorRetCOFINS     decimal.Decimal
	ValorRetCSLL       decimal.Decimal
	BaseIRRF           decimal.Decimal
	ValorIRRF          decimal.Decimal
	BaseRetPrev        decimal.Decimal
	ValorRetPrev       decimal.Decimal
	ValorISSRet        decimal.Decimal
	StockStatus        string
	CancelledAt        *time.Time
	CancelReason       *string
}

// Situação do lançamento da nota no estoque.
const (
	StockStatusNaoAplica = "NAO_APLICA"
	StockStatusPendente  = "PENDENTE"
	StockStatusConcluido = "CONCLUIDO"
	StockStatusEstornado = "ESTORNADO" // nota cancelada depois de movimentar
)

// TotalRetencoes é o que a empresa retém do fornecedor e recolhe ao fisco.
func (e *FiscalEntry) TotalRetencoes() decimal.Decimal {
	return e.ValorRetPIS.Add(e.ValorRetCOFINS).Add(e.ValorRetCSLL).Add(e.ValorIRRF).Add(e.ValorRetPrev).Add(e.ValorISSRet)
}

// ValorAPagar é o que vai ao fornecedor: o total da nota menos as retenções.
func (e *FiscalEntry) ValorAPagar() decimal.Decimal {
	return decimal.NewFromFloat(e.ValorTotal).Round(2).Sub(e.TotalRetencoes())
}

type FiscalEntryItem struct {
	ID                 int64
	FiscalEntryID      int64
	Sequence           int
	ItemCode           *int64
	ItemSupplierID     *int64
	SupplierItemCode   *string
	ResolutionStrategy *string
	ResolvedAt         *time.Time
	UOM                *string
	Ncm                *string
	Cfop               string
	Quantity           float64
	UnitPrice          float64
	TotalPrice         float64
	BaseICMS           float64
	AliqICMS           float64
	ValorICMS          float64
	BaseIPI            float64
	AliqIPI            float64
	ValorIPI           float64
	ValorPIS           float64
	ValorCOFINS        float64
	CstICMS            *string
	CstIPI             *string
	CstPIS             *string
	CstCOFINS          *string
	GeraCreditoICMS    bool
	GeraCreditoIPI     bool
	GeraCreditoPIS     bool
	GeraCreditoCOFINS  bool
	Description        *string
	Notes              *string
	CreatedAt          time.Time

	EAN               *string
	CEST              *string
	Origem            *string
	ValorFrete        decimal.Decimal
	ValorSeguro       decimal.Decimal
	ValorDesconto     decimal.Decimal
	ValorOutras       decimal.Decimal
	BaseICMSST        decimal.Decimal
	ValorICMSST       decimal.Decimal
	ValorContabil     decimal.Decimal
	FatorConversao    *decimal.Decimal
	QuantidadeEstoque *decimal.Decimal
	PedidoCompraXML   *string
	ItemPedidoXML     *string
	PlanoContasID     *int64
	CentroCustoID     *int64
	// Migração 000377: operação (TES), pedido de compra, estoque e Reforma.
	CfopEntrada           *string
	EntryOperationCode    *int64
	MovimentaEstoque      bool
	GeraFinanceiro        bool
	WarehouseID           *int64
	PurchaseOrderCode     *int64
	PurchaseOrderItemCode *int64
	QtdRecebidaAntes      decimal.Decimal
	StockMovementID       *int64
	CustoAquisicao        decimal.Decimal
	CSTIBSCBS             *string
	ClassTrib             *string
	BaseIBSCBS            decimal.Decimal
	AliqIBSUF             decimal.Decimal
	ValorIBSUF            decimal.Decimal
	AliqIBSMun            decimal.Decimal
	ValorIBSMun           decimal.Decimal
	ValorIBS              decimal.Decimal
	AliqCBS               decimal.Decimal
	ValorCBS              decimal.Decimal
	ValorIS               decimal.Decimal
	GeraCreditoIBSCBS     bool
	// Leitura: nomes para a tela de conciliação.
	WarehouseName      *string
	EntryOperationName *string
	ItemNCM            *string
	// Número do pedido de compra e sequência da linha (o que o comprador
	// reconhece; purchase_order_code é a chave interna).
	PurchaseOrderNumber   *int64
	PurchaseOrderSequence *int32
	ItemName              *string
	ItemUOM               *string
	PlanoContasCodigo     *string
	PlanoContasNome       *string
	CentroCustoNome       *string
}

// ChaveConta é o destino financeiro do item (nulo quando não classificado).
func (i *FiscalEntryItem) ChaveConta() *ChaveConta {
	if i.PlanoContasID == nil {
		return nil
	}
	k := ChaveConta{PlanoContasID: *i.PlanoContasID}
	if i.CentroCustoID != nil {
		k.CentroCustoID = *i.CentroCustoID
	}
	return &k
}
