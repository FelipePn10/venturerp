package request

// CreateNFSeDTO registers a service invoice (NFS-e) in draft, ready to be
// authorized at the city hall via Focus.
type CreateNFSeDTO struct {
	NumeroRPS            *int64  `json:"numero_rps,omitempty"`
	SerieRPS             *string `json:"serie_rps,omitempty"`
	TipoRPS              int     `json:"tipo_rps"`
	DataEmissao          string  `json:"data_emissao"`
	NaturezaOperacao     int     `json:"natureza_operacao"`
	OptanteSimples       bool    `json:"optante_simples"`
	IncentivadorCultural bool    `json:"incentivador_cultural"`

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

	SalesOrderCode *int64  `json:"sales_order_code,omitempty"`
	Notes          *string `json:"notes,omitempty"`
}

// CancelNFSeDTO carries the cancellation justification.
type CancelNFSeDTO struct {
	Justificativa string `json:"justificativa"`
}
