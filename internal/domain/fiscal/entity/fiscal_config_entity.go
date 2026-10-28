package entity

import (
	"time"

	"github.com/google/uuid"
)

type FiscalConfig struct {
	ID                        int64
	EnterpriseID              int64
	CnpjEmpresa               string
	RazaoSocial               string
	TradeName                 string
	Email                     string
	IEEmpresa                 *string
	RegimeTributario          string
	UFEmpresa                 string
	IcmsInternoAliquota       float64
	IcmsDiferimentoPercentual float64
	// ExigirRegraICMSIPI: a linha de nota sem regra no cadastro de parâmetros
	// de ICMS/IPI (VFIS0320) não é transmitida. Desligado vale a regra geral.
	ExigirRegraICMSIPI bool
	// AliquotaCreditoSimples: pCredSN (%) do optante do Simples Nacional.
	AliquotaCreditoSimples float64
	// CBenefDiferimento: cBenef do diferimento da configuração (PR830001 no PR),
	// usado no CST 51 quando a linha e a regra não trazem outro.
	CBenefDiferimento      *string
	FocusNfeToken          *string
	FocusNfeAmbiente       string
	JurosMes               float64
	MultaAtraso            float64
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

// TipoNotaFiscal: os indicadores do tipo de nota (VFIS0110) que mudam a
// tributação da saída.
type TipoNotaFiscal struct {
	ID                 int64
	Code               int64
	Descricao          string
	Tipo               string
	TipoICMS           string // TRIBUTADO, ISENTO, OUTROS
	ReducaoICMSPct     float64
	CalculaST          bool
	CalculaDiferimento bool
	CalculaPISCOFINS   bool
	CalculaDIFAL       bool
}
