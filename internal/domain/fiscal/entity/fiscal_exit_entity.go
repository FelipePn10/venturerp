package entity

import (
	"time"

	"github.com/google/uuid"
)

type FiscalExitStatus string

const (
	ExitStatusDraft                 FiscalExitStatus = "DRAFT"
	ExitStatusAuthorized            FiscalExitStatus = "AUTHORIZED"
	ExitStatusCancelled             FiscalExitStatus = "CANCELLED"
	ExitStatusRejected              FiscalExitStatus = "REJECTED"
	ExitStatusAwaitingAuthorization FiscalExitStatus = "AGUARDANDO_AUTORIZACAO"
)

type FiscalExit struct {
	ID                      int64
	EnterpriseID            int64
	ChaveAcesso             *string
	NumeroNF                int64
	Serie                   string
	DataEmissao             time.Time
	DataSaida               *time.Time
	CnpjDestinatario        *string
	RazaoSocialDestinatario *string
	IEDestinatario          *string
	UFDestinatario          *string
	// Endereco do destinatario. O layout da NF-e exige logradouro, numero,
	// bairro, municipio, codigo IBGE do municipio e CEP: sem eles a SEFAZ
	// rejeita a autorizacao por campo obrigatorio ausente.
	DestLogradouro      *string
	DestNumero          *string
	DestComplemento     *string
	DestBairro          *string
	DestMunicipio       *string
	DestCodigoMunicipio *string
	DestCEP             *string
	DestEmail           *string
	DestTelefone        *string
	// CustomerCode e o cliente de onde o endereco e a condicao de pagamento
	// foram resolvidos; e por ele que o titulo nasce com dono.
	CustomerCode     *int64
	Cfop             string
	NaturezaOperacao string
	ValorProdutos    float64
	ValorFrete       float64
	ValorSeguro      float64
	ValorDesconto    float64
	ValorIPI         float64
	ValorICMS        float64
	ValorPIS         float64
	ValorCOFINS      float64
	BaseICMSST       float64
	ValorICMSST      float64
	ValorTotal       float64
	SalesOrderCode   *int64
	SourceType       *string
	// CustomerMaterialRemittanceID liga a nota à remessa de beneficiamento que a
	// originou (migração 000374). É o que permite RETOMAR um rascunho quando o
	// faturamento falha no meio, em vez de criar outra nota ao repetir.
	CustomerMaterialRemittanceID *int64
	// Finalidade da NF-e: 1 normal, 2 complementar, 3 ajuste, 4 devolução.
	Finalidade int
	// NFeReferenciada é a chave da nota de origem (devolução de compra).
	NFeReferenciada *string
	// FiscalEntryID: a nota de entrada devolvida (devolução de compra).
	FiscalEntryID *int64
	// TipoDocumento da NF-e: 0 entrada (devolução de venda emitida pela
	// empresa), 1 saída. nil = saída.
	TipoDocumento *int
	// DevolucaoVendaID: a devolução de venda que a NF-e de entrada efetiva.
	DevolucaoVendaID *int64
	// FinalidadeCompra que valeu no cálculo (REVENDA, INDUSTRIALIZACAO,
	// USO_CONSUMO); nil = deduzida do destinatário.
	FinalidadeCompra *string
	// InvoiceTypeID: tipo de nota (VFIS0110) cujos indicadores tributaram a saída.
	InvoiceTypeID *int64
	// InformacoesComplementares (infCpl): o texto do usuário mais o que a lei
	// manda constar (Simples Nacional, base legal dos benefícios).
	InformacoesComplementares *string
	// SupplierCode: o fornecedor destinatário da devolução.
	SupplierCode          *int64
	ShipmentLoadCode      *int64
	ShipmentCode          *int64
	FiscalCouponNumber    *string
	FiscalCouponDate      *time.Time
	FiscalCouponECFSerial *string
	Status                FiscalExitStatus
	Protocolo             *string
	XmlPath               *string
	DanfePath             *string
	FocusRef              *string
	IsActive              bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
	CreatedBy             uuid.UUID
	Itens                 []*FiscalExitItem
}

type FiscalExitItem struct {
	ID                int64
	FiscalExitID      int64
	Sequence          int
	ItemCode          *int64
	Ncm               *string
	Cfop              string
	Quantity          float64
	UnitPrice         float64
	TotalPrice        float64
	BaseICMS          float64
	AliqICMS          float64
	ValorICMS         float64
	ValorICMSDiferido float64
	BaseIPI           float64
	AliqIPI           float64
	ValorIPI          float64
	AliqPIS           float64
	ValorPIS          float64
	AliqCOFINS        float64
	ValorCOFINS       float64
	BaseICMSST        float64
	AliqICMSST        float64
	ValorICMSST       float64
	MVA               float64
	CstICMS           *string
	CstIPI            *string
	CstPIS            *string
	CstCOFINS         *string
	OrigemMercadoria  string
	Description       *string
	// UnidadeComercial é a unidade da linha na NF-e (`uCom`). Nulo cai em "UN" no
	// autorizador, que era o valor FIXO de antes — a nota de beneficiamento da Usimac
	// tem linhas em KG, e sair em UN descreve outra mercadoria.
	UnidadeComercial *string
	// CodigoProduto é o `cProd` da linha. Texto porque pode ser o código DO CLIENTE
	// (material de terceiro, que não existe no nosso cadastro) ou conter letras.
	// Nulo cai em `ItemCode`.
	CodigoProduto *string
	// SalesOrderItemCode é a linha do pedido de venda faturada por esta linha
	// (faturamento a partir do pedido, total ou parcial).
	SalesOrderItemCode *int64
	// FiscalEntryItemID: a linha da nota de entrada devolvida por esta linha.
	FiscalEntryItemID *int64
	// FiscalExitItemOrigemID: a linha da NF de saída devolvida (devolução de venda).
	FiscalExitItemOrigemID *int64
	// CodigoBeneficioFiscal é o cBenef da linha (exigido pela SEFAZ de algumas
	// UFs para CST com benefício: diferimento, isenção, redução…).
	CodigoBeneficioFiscal *string
	// RegraICMSIPIID é a regra do cadastro de parâmetros de ICMS/IPI (VFIS0320)
	// que tributou a linha; nil = regra geral da empresa.
	RegraICMSIPIID *int64
	// ICMSReducaoBasePct (pRedBC) e ICMSDiferimentoPct (pDif) da linha, em %.
	// Zero no diferimento = linha antiga: vale o da configuração fiscal.
	ICMSReducaoBasePct float64
	ICMSDiferimentoPct float64
	// ICMSSTReducaoBasePct: pRedBCST (%) da linha.
	ICMSSTReducaoBasePct float64
	// Simples Nacional: pCredSN (%) e vCredICMSSN.
	AliqCreditoSimples  float64
	ValorCreditoSimples float64
	// Grupo ICMSUFDest (DIFAL a não contribuinte): base, alíquotas em % e valores.
	ICMSUFDestBase        float64
	ICMSUFDestAliqInterna float64
	ICMSAliqInterestadual float64
	ICMSUFDestValor       float64
	FCPUFDestPct          float64
	FCPUFDestValor        float64
	// ClassTribIBSCBS é o cClassTrib próprio da linha (sobrescreve o do CFOP e
	// o da classificação fiscal). ReformaEnviada é o grupo IBS/CBS transmitido.
	ClassTribIBSCBS *string
	ReformaEnviada  *ReformaEnviada
	CreatedAt       time.Time
}

// ReformaEnviada é o retrato do grupo IBS/CBS que a linha levou à SEFAZ.
type ReformaEnviada struct {
	CST, ClassTrib, Origem string
	Base, IBSUF, IBSMun    *float64
	CBS                    *float64
	AliqEfetivaCBS         *float64
}
