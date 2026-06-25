package entity

import (
	"time"

	"github.com/google/uuid"
)

type FiscalConfig struct {
	ID                        int64
	CnpjEmpresa               string
	RazaoSocial               string
	IEEmpresa                 *string
	RegimeTributario          string
	UFEmpresa                 string
	IcmsInternoAliquota       float64
	IcmsDiferimentoPercentual float64
	FocusNfeToken             *string
	FocusNfeAmbiente          string
	JurosMes                  float64
	MultaAtraso               float64
	VencimentoIcmsDia         int
	VencimentoIPIDia          int
	VencimentoPisCofinsDia    int
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
