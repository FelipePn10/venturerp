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
	ShipmentLoadCode             *int64
	ShipmentCode                 *int64
	FiscalCouponNumber           *string
	FiscalCouponDate             *time.Time
	FiscalCouponECFSerial        *string
	Status                       FiscalExitStatus
	Protocolo                    *string
	XmlPath                      *string
	DanfePath                    *string
	FocusRef                     *string
	IsActive                     bool
	CreatedAt                    time.Time
	UpdatedAt                    time.Time
	CreatedBy                    uuid.UUID
	Itens                        []*FiscalExitItem
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
	CreatedAt     time.Time
}
