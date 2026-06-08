package entity

import (
	"time"

	"github.com/google/uuid"
)

type NFSeStatus string

const (
	NFSeStatusRascunho    NFSeStatus = "RASCUNHO"
	NFSeStatusAutorizada  NFSeStatus = "AUTORIZADA"
	NFSeStatusCancelada   NFSeStatus = "CANCELADA"
	NFSeStatusRejeitada   NFSeStatus = "REJEITADA"
	NFSeStatusProcessando NFSeStatus = "PROCESSANDO"
)

// NFSe is a service invoice (Nota Fiscal de Serviços eletrônica), ABRASF model.
type NFSe struct {
	ID                   int64      `json:"id"`
	NumeroRPS            *int64     `json:"numero_rps,omitempty"`
	SerieRPS             *string    `json:"serie_rps,omitempty"`
	TipoRPS              int        `json:"tipo_rps"`
	DataEmissao          time.Time  `json:"data_emissao"`
	Status               NFSeStatus `json:"status"`
	NaturezaOperacao     int        `json:"natureza_operacao"`
	OptanteSimples       bool       `json:"optante_simples"`
	IncentivadorCultural bool       `json:"incentivador_cultural"`

	// Tomador
	TomadorCnpjCpf         *string `json:"tomador_cnpj_cpf,omitempty"`
	TomadorRazaoSocial     *string `json:"tomador_razao_social,omitempty"`
	TomadorEmail           *string `json:"tomador_email,omitempty"`
	TomadorLogradouro      *string `json:"tomador_logradouro,omitempty"`
	TomadorNumero          *string `json:"tomador_numero,omitempty"`
	TomadorComplemento     *string `json:"tomador_complemento,omitempty"`
	TomadorBairro          *string `json:"tomador_bairro,omitempty"`
	TomadorCodigoMunicipio *string `json:"tomador_codigo_municipio,omitempty"`
	TomadorUF              *string `json:"tomador_uf,omitempty"`
	TomadorCEP             *string `json:"tomador_cep,omitempty"`

	// Serviço
	ItemListaServico          string  `json:"item_lista_servico"`
	CodigoTributarioMunicipio *string `json:"codigo_tributario_municipio,omitempty"`
	Discriminacao             string  `json:"discriminacao"`
	CodigoMunicipio           string  `json:"codigo_municipio"`
	ValorServicos             float64 `json:"valor_servicos"`
	ValorDeducoes             float64 `json:"valor_deducoes"`
	AliquotaISS               float64 `json:"aliquota_iss"`
	IssRetido                 bool    `json:"iss_retido"`
	ValorISS                  float64 `json:"valor_iss"`
	ValorLiquido              float64 `json:"valor_liquido"`

	// Emissão / prefeitura
	FocusRef          *string `json:"focus_ref,omitempty"`
	NumeroNFSe        *string `json:"numero_nfse,omitempty"`
	CodigoVerificacao *string `json:"codigo_verificacao,omitempty"`
	URL               *string `json:"url,omitempty"`
	XMLPath           *string `json:"xml_path,omitempty"`

	SalesOrderCode *int64    `json:"sales_order_code,omitempty"`
	Notes          *string   `json:"notes,omitempty"`
	IsActive       bool      `json:"is_active"`
	CreatedBy      uuid.UUID `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
