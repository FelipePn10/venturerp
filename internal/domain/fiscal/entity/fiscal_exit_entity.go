package entity

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/google/uuid"
)

// Transportador é o grupo transporta da NF-e.
type Transportador struct {
	CNPJCPF   string `json:"cnpj_cpf"`
	Nome      string `json:"nome"`
	IE        string `json:"ie,omitempty"`
	Endereco  string `json:"endereco,omitempty"`
	Municipio string `json:"municipio,omitempty"`
	UF        string `json:"uf,omitempty"`
}

// LocalEntrega é o grupo entrega da NF-e: outro endereço do mesmo destinatário.
type LocalEntrega struct {
	CNPJCPF     string `json:"cnpj_cpf"`
	Nome        string `json:"nome,omitempty"`
	Logradouro  string `json:"logradouro"`
	Numero      string `json:"numero"`
	Complemento string `json:"complemento,omitempty"`
	Bairro      string `json:"bairro"`
	Municipio   string `json:"municipio"`
	UF          string `json:"uf"`
	CEP         string `json:"cep,omitempty"`
}

// TipoFreteDaNota traduz o tipo de frete do orçamento/pedido/cliente para um dos
// quatro da nota (CIF, FOB, TERCEIROS, SEM_FRETE). O orçamento usa a lista
// comercial (CIF/FOB contratado ou próprio, DAF, cortesia, retira…) e ela
// chega ao pedido sem conversão. ok=false: sem equivalente (CONVENIO) ou vazio.
//
// shortcut: CIF/FOB próprio viram 0/1, não os modFrete 3/4 (transporte próprio);
// separar quando algum cliente exigir a distinção na NF-e.
func TipoFreteDaNota(tipo string) (string, bool) {
	t := strings.ReplaceAll(strings.ToUpper(strings.Join(strings.Fields(tipo), " ")), "Ó", "O")
	switch t {
	case "CIF", "CIF CONTRAT.", "CIF PROPRIO", "DAF", "CORTESIA":
		return "CIF", true
	case "FOB", "FOB CONTRAT.", "FOB PROPRIO", "RETIRA":
		return "FOB", true
	case "TERCEIROS":
		return "TERCEIROS", true
	case "SEM_FRETE", "SEM FRETE":
		return "SEM_FRETE", true
	}
	return "", false
}

// ModalidadeFreteNFe é o modFrete: 0 por conta do remetente (CIF), 1 do
// destinatário (FOB), 2 de terceiros, 9 sem transporte. Sem tipo informado,
// frete na nota é do remetente e sem frete é sem transporte.
func ModalidadeFreteNFe(tipo *string, valorFrete decimal.Decimal) int {
	if tipo != nil {
		switch *tipo {
		case "CIF":
			return 0
		case "FOB":
			return 1
		case "TERCEIROS":
			return 2
		case "SEM_FRETE":
			return 9
		}
	}
	if valorFrete.IsPositive() {
		return 0
	}
	return 9
}

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
	ValorProdutos    decimal.Decimal
	ValorFrete       decimal.Decimal
	ValorSeguro      decimal.Decimal
	ValorDesconto    decimal.Decimal
	ValorIPI         decimal.Decimal
	ValorICMS        decimal.Decimal
	ValorPIS         decimal.Decimal
	ValorCOFINS      decimal.Decimal
	BaseICMSST       decimal.Decimal
	ValorICMSST      decimal.Decimal
	ValorTotal       decimal.Decimal
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
	// InvoiceTypeID: tipo de nota (VCLI0530) cujos indicadores tributaram a saída.
	InvoiceTypeID *int64
	// InformacoesComplementares (infCpl): o texto do usuário mais o que a lei
	// manda constar (Simples Nacional, base legal dos benefícios).
	InformacoesComplementares *string
	// TipoFrete: CIF, FOB, TERCEIROS ou SEM_FRETE (modFrete da NF-e); nil =
	// deduzido do valor do frete.
	TipoFrete *string
	// ShippingCarrierID e Transportador: a transportadora e a foto dos dados
	// dela no momento da nota (grupo transporta da NF-e).
	ShippingCarrierID *int64
	Transportador     *Transportador
	// LocalEntrega: endereço de entrega quando é outro que o do destinatário
	// (grupo entrega da NF-e).
	LocalEntrega *LocalEntrega
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
	// Mask, Lot e Address dizem de onde sai o estoque. Sem lote ou endereço a
	// baixa é automática, pela regra do item.
	Mask    string
	Lot     *string
	Address *string

	ID                int64
	FiscalExitID      int64
	Sequence          int
	ItemCode          *int64
	Ncm               *string
	Cfop              string
	Quantity          decimal.Decimal
	UnitPrice         decimal.Decimal
	TotalPrice        decimal.Decimal
	BaseICMS          decimal.Decimal
	AliqICMS          decimal.Decimal
	ValorICMS         decimal.Decimal
	ValorICMSDiferido decimal.Decimal
	BaseIPI           decimal.Decimal
	AliqIPI           decimal.Decimal
	ValorIPI          decimal.Decimal
	AliqPIS           decimal.Decimal
	ValorPIS          decimal.Decimal
	AliqCOFINS        decimal.Decimal
	ValorCOFINS       decimal.Decimal
	BaseICMSST        decimal.Decimal
	AliqICMSST        decimal.Decimal
	ValorICMSST       decimal.Decimal
	MVA               decimal.Decimal
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
	ICMSReducaoBasePct decimal.Decimal
	ICMSDiferimentoPct decimal.Decimal
	// ICMSSTReducaoBasePct: pRedBCST (%) da linha.
	ICMSSTReducaoBasePct decimal.Decimal
	// Simples Nacional: pCredSN (%) e vCredICMSSN.
	AliqCreditoSimples  decimal.Decimal
	ValorCreditoSimples decimal.Decimal
	// Grupo ICMSUFDest (DIFAL a não contribuinte): base, alíquotas em % e valores.
	ICMSUFDestBase        decimal.Decimal
	ICMSUFDestAliqInterna decimal.Decimal
	ICMSAliqInterestadual decimal.Decimal
	ICMSUFDestValor       decimal.Decimal
	FCPUFDestPct          decimal.Decimal
	FCPUFDestValor        decimal.Decimal
	// ClassTribIBSCBS é o cClassTrib próprio da linha (sobrescreve o do CFOP e
	// o da classificação fiscal). ReformaEnviada é o grupo IBS/CBS transmitido.
	ClassTribIBSCBS *string
	ReformaEnviada  *ReformaEnviada
	CreatedAt       time.Time
}

// ReformaEnviada é o retrato do grupo IBS/CBS que a linha levou à SEFAZ.
type ReformaEnviada struct {
	CST, ClassTrib, Origem string
	Base, IBSUF, IBSMun    *decimal.Decimal
	CBS                    *decimal.Decimal
	AliqEfetivaCBS         *decimal.Decimal
}
