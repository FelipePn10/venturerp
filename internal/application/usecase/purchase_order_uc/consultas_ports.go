package purchase_order_uc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
)

// Portas das consultas de compras que cruzam tabelas de outros módulos (notas
// de entrada, fornecedor, transportadora, configuração da empresa). Ficam
// fora do PurchaseOrderRepository para não obrigar todo dublê de teste dele a
// implementá-las.

// FiltroAcompanhamento seleciona as linhas em aberto de pedidos aprovados.
type FiltroAcompanhamento struct {
	SupplierCode *int64
	AteData      *time.Time // previstas até esta data (inclusive)
}

// LinhaEmAberto é uma linha de pedido aprovado que ainda espera material.
type LinhaEmAberto struct {
	PurchaseOrderCode int64      `json:"purchase_order_code"`
	OrderNumber       int64      `json:"order_number"`
	LineCode          int64      `json:"line_code"`
	Sequence          int        `json:"sequence"`
	ItemCode          int64      `json:"item_code"`
	ItemName          string     `json:"item_name"`
	SupplierCode      int64      `json:"supplier_code"`
	SupplierName      string     `json:"supplier_name"`
	RequestedQty      float64    `json:"requested_qty"`
	ReceivedQty       float64    `json:"received_qty"`
	Saldo             float64    `json:"saldo"`
	DeliveryDate      *time.Time `json:"delivery_date,omitempty"`
	PromisedDate      *time.Time `json:"promised_date,omitempty"`
	DataPrevista      *time.Time `json:"data_prevista,omitempty"`
	DiasAtraso        int        `json:"dias_atraso"`
	UltimoContato     *Followup  `json:"ultimo_contato,omitempty"`
}

// Followup é um contato com o fornecedor sobre a entrega de uma linha.
type Followup struct {
	ID                int64      `json:"id"`
	PurchaseOrderCode int64      `json:"purchase_order_code"`
	LineCode          int64      `json:"line_code"`
	DataPrometida     *time.Time `json:"data_prometida,omitempty"`
	Contato           string     `json:"contato,omitempty"`
	Observacao        string     `json:"observacao,omitempty"`
	RegistradoEm      time.Time  `json:"registrado_em"`
	RegistradoPor     string     `json:"registrado_por,omitempty"`
}

// NotaDaLinha é uma nota de entrada que trouxe material de uma linha.
type NotaDaLinha struct {
	LineCode      int64     `json:"line_code"`
	FiscalEntryID int64     `json:"fiscal_entry_id"`
	NumeroNF      int64     `json:"numero_nf"`
	Serie         string    `json:"serie"`
	DataEmissao   time.Time `json:"data_emissao"`
	DataEntrada   time.Time `json:"data_entrada"`
	Status        string    `json:"status"`
	Quantidade    float64   `json:"quantidade"`
	Unidade       string    `json:"unidade,omitempty"`
	ValorUnitario float64   `json:"valor_unitario"`
	ValorTotal    float64   `json:"valor_total"`
	SupplierName  string    `json:"supplier_name,omitempty"`
}

// CompraDoItem é uma compra passada do item (nota de entrada aprovada).
type CompraDoItem struct {
	FiscalEntryID  int64     `json:"fiscal_entry_id"`
	NumeroNF       int64     `json:"numero_nf"`
	DataEntrada    time.Time `json:"data_entrada"`
	SupplierCode   *int64    `json:"supplier_code,omitempty"`
	SupplierName   string    `json:"supplier_name"`
	Quantidade     float64   `json:"quantidade"`
	Unidade        string    `json:"unidade,omitempty"`
	PrecoNota      float64   `json:"preco_nota"`
	CustoEstoque   float64   `json:"custo_estoque"` // custo de aquisição por unidade de estoque
	QtdEstoque     float64   `json:"qtd_estoque"`
	CustoAquisicao float64   `json:"custo_aquisicao"`
}

// UltimoPedidoDoItem é o preço da última linha de pedido do item.
type UltimoPedidoDoItem struct {
	PurchaseOrderCode int64     `json:"purchase_order_code"`
	OrderNumber       int64     `json:"order_number"`
	EmissionDate      time.Time `json:"emission_date"`
	SupplierName      string    `json:"supplier_name"`
	UnitPrice         float64   `json:"unit_price"`
	PurchaseUOM       string    `json:"purchase_uom,omitempty"`
}

// Destinatario é um e-mail do fornecedor para o pedido.
type Destinatario struct {
	Email    string `json:"email"`
	Nome     string `json:"nome,omitempty"`
	Origem   string `json:"origem"` // CONTATO_PEDIDO, CONTATO, FORNECEDOR
	Sugerido bool   `json:"sugerido"`
}

// Envio é o registro de um envio do pedido ao fornecedor.
type Envio struct {
	ID                int64      `json:"id"`
	PurchaseOrderCode int64      `json:"purchase_order_code"`
	EnviadoEm         time.Time  `json:"enviado_em"`
	EnviadoPor        *uuid.UUID `json:"-"`
	EnviadoPorNome    string     `json:"enviado_por,omitempty"`
	Destinatarios     string     `json:"destinatarios"`
	Assunto           string     `json:"assunto"`
	Situacao          string     `json:"situacao"`
	Erro              string     `json:"erro,omitempty"`
}

// Parte é empresa, fornecedor ou transportadora no documento.
type Parte struct {
	Nome     string
	Fantasia string
	CNPJCPF  string
	IE       string
	Endereco string
	Cidade   string
	UF       string
	CEP      string
	Telefone string
	Email    string
}

// LinhaDocumento é uma linha do pedido como sai no PDF.
type LinhaDocumento struct {
	Sequence     int
	ItemCode     string // código comercial
	Descricao    string
	Unidade      string
	Quantidade   float64
	PrecoUnit    float64
	DescontoPct  float64
	IPIPct       float64
	Total        float64
	Entrega      *time.Time
	Almoxarifado string
	Observacao   string
	Situacao     string
	Cancelada    float64
}

// DadosDocumento é o que o PDF precisa além do pedido.
type DadosDocumento struct {
	Empresa        Parte
	Logo           []byte
	CorMarca       string
	Fornecedor     Parte
	Transportadora string
	Condicao       string
	Comprador      string
	Linhas         []LinhaDocumento
}

// ConsultasCompras é implementada na infraestrutura com SQL direto.
type ConsultasCompras interface {
	LinhasEmAberto(ctx context.Context, f FiltroAcompanhamento) ([]LinhaEmAberto, error)
	RegistrarFollowup(ctx context.Context, f Followup, por uuid.UUID) (*Followup, error)
	Followups(ctx context.Context, lineCode int64) ([]Followup, error)
	NotasDoPedido(ctx context.Context, orderCode int64) ([]NotaDaLinha, error)
	ComprasDoItem(ctx context.Context, itemCode int64, desde time.Time, limite int) ([]CompraDoItem, error)
	UltimoPedidoDoItem(ctx context.Context, itemCode int64) (*UltimoPedidoDoItem, error)
	Destinatarios(ctx context.Context, supplierCode int64) ([]Destinatario, error)
	RegistrarEnvio(ctx context.Context, e Envio) (*Envio, error)
	Envios(ctx context.Context, orderCode int64) ([]Envio, error)
	DadosDocumento(ctx context.Context, orderCode int64) (*DadosDocumento, error)
}

// CondicoesDePagamento lê a condição de pagamento (com as parcelas) pelo código.
type CondicoesDePagamento interface {
	GetPaymentConditionByCode(ctx context.Context, code int64) (*customerentity.PaymentCondition, error)
}

// GeradorPDFPedido desenha o documento do pedido.
type GeradorPDFPedido interface {
	PedidoCompra(d DocumentoPedido) ([]byte, error)
}

// DocumentoPedido junta pedido, dados do documento e a previsão de pagamento.
type DocumentoPedido struct {
	Dados      *DadosDocumento
	Numero     int64
	Codigo     int64
	Emissao    time.Time
	Entrega    *time.Time
	Situacao   string
	Rascunho   bool
	Moeda      string
	Frete      string
	ValorFrete float64
	Observacao string
	Bruto      decimal.Decimal
	Desconto   decimal.Decimal
	IPI        decimal.Decimal
	FreteFOB   decimal.Decimal
	Liquido    decimal.Decimal
	Parcelas   []ParcelaPrevista
	Gerado     time.Time
}
