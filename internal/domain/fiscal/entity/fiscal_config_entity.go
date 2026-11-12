package entity

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/google/uuid"
)

type FiscalConfig struct {
	ID           int64
	EnterpriseID int64
	CnpjEmpresa  string
	RazaoSocial  string
	TradeName    string
	Email        string
	IEEmpresa    *string
	// InscricaoMunicipal: do prestador na NFS-e. Mantida na identificação da
	// empresa (VEMP0100); a gravação da configuração fiscal não mexe nela.
	InscricaoMunicipal        *string
	RegimeTributario          string
	UFEmpresa                 string
	IcmsInternoAliquota       decimal.Decimal
	IcmsDiferimentoPercentual decimal.Decimal
	// ExigirRegraICMSIPI: a linha de nota sem regra no cadastro de parâmetros
	// de ICMS/IPI (VFIS0320) não é transmitida. Desligado vale a regra geral.
	ExigirRegraICMSIPI bool
	// AliquotaCreditoSimples: pCredSN (%) do optante do Simples Nacional.
	AliquotaCreditoSimples decimal.Decimal
	// CBenefDiferimento: cBenef do diferimento da configuração (PR830001 no PR),
	// usado no CST 51 quando a linha e a regra não trazem outro.
	CBenefDiferimento *string
	// ObsRetornoIndustrializacao: texto legal da nota de retorno do beneficiamento.
	// Vazio em SP usa a Portaria CAT 22/2007; nas demais UFs a nota é recusada.
	ObsRetornoIndustrializacao *string
	// CBenefSuspensao: cBenef do material que volta (CST 50), na UF que o exige.
	CBenefSuspensao        *string
	FocusNfeToken          *string
	FocusNfeAmbiente       string
	JurosMes               decimal.Decimal
	MultaAtraso            decimal.Decimal
	VencimentoIcmsDia      int
	VencimentoIPIDia       int
	VencimentoPisCofinsDia int
	// Endereço do emitente (obrigatório para NF-e em produção)
	Logradouro      string
	Numero          string
	Complemento     *string
	Bairro          string
	Municipio       string
	CodigoMunicipio string
	CEP             string
	Telefone        *string
	// Branding para o cabeçalho dos relatórios (letterhead profissional)
	Logo       []byte  // imagem PNG/JPEG do logo, opcional
	LogoMime   *string // mime type do logo (image/png, image/jpeg)
	BrandColor *string // cor da marca em #RRGGBB
	CreatedAt  time.Time
	UpdatedAt  time.Time
	UpdatedBy  uuid.UUID
}

// TipoNotaFiscal: os indicadores do tipo de nota (VCLI0530) que mudam a
// tributação da saída.
type TipoNotaFiscal struct {
	ID                 int64
	Code               int64
	Descricao          string
	Tipo               string
	TipoICMS           string // TRIBUTADO, ISENTO, OUTROS
	ReducaoICMSPct     decimal.Decimal
	CalculaST          bool
	CalculaDiferimento bool
	CalculaPISCOFINS   bool
	CalculaDIFAL       bool
}

// EfeitosDoTipoNota: o que o tipo de nota diz que a autorização faz. Bonificação,
// remessa e transferência não cobram; o cadastro é quem manda.
type EfeitosDoTipoNota struct {
	GeraTitulo       bool
	MovimentaEstoque bool
	BaixaPedido      bool
	GeraReceita      bool
	// Tipo é a natureza do cadastro: remessa e venda em consignação e transferência
	// não baixam o estoque como a venda comum.
	Tipo string
}

// Tipos de nota que mudam o caminho do estoque.
const (
	TipoNotaRemessaConsignacao = "REMESSA_CONSIGNACAO"
	TipoNotaVendaConsignacao   = "VENDA_CONSIGNACAO"
	TipoNotaTransferencia      = "TRANSFERENCIA"
)

// EfeitosPadraoDaNota: sem tipo de nota, a nota é uma venda comum.
func EfeitosPadraoDaNota() EfeitosDoTipoNota {
	return EfeitosDoTipoNota{GeraTitulo: true, MovimentaEstoque: true, BaixaPedido: true, GeraReceita: true, Tipo: "VENDA"}
}
