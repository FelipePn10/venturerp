package request

import "github.com/google/uuid"

// CreateEnterpriseDTO: código 0 = próximo livre; nome vazio = razão social.
// created_by é sempre o usuário da sessão (o valor enviado é ignorado).
// Com CNPJ, a empresa nasce com a identidade fiscal (fiscal_configs).
type CreateEnterpriseDTO struct {
	Code      int       `json:"code"`
	Name      string    `json:"name"`
	CreatedBy uuid.UUID `json:"created_by"`

	CNPJ               string `json:"cnpj"`
	RazaoSocial        string `json:"razao_social"`
	NomeFantasia       string `json:"nome_fantasia"`
	InscricaoEstadual  string `json:"inscricao_estadual"`
	InscricaoMunicipal string `json:"inscricao_municipal"`
	RegimeTributario   string `json:"regime_tributario"`
	UF                 string `json:"uf"`
	Municipio          string `json:"municipio"`
	CodigoMunicipio    string `json:"codigo_municipio"`
	CEP                string `json:"cep"`
	Logradouro         string `json:"logradouro"`
	Numero             string `json:"numero"`
	Complemento        string `json:"complemento"`
	Bairro             string `json:"bairro"`
	Telefone           string `json:"telefone"`
	Email              string `json:"email"`
	MatrizCNPJ         string `json:"matriz_cnpj"`
}

// IdentificacaoEmpresaDTO: a identificação da empresa da sessão (VEMP0100).
type IdentificacaoEmpresaDTO struct {
	CNPJ              string `json:"cnpj"`
	RazaoSocial       string `json:"razao_social"`
	NomeFantasia      string `json:"nome_fantasia"`
	InscricaoEstadual string `json:"inscricao_estadual"`
	RegimeTributario  string `json:"regime_tributario"`
	UF                string `json:"uf"`
	Municipio         string `json:"municipio"`
	CodigoMunicipio   string `json:"codigo_municipio"`
	CEP               string `json:"cep"`
	Logradouro        string `json:"logradouro"`
	Numero            string `json:"numero"`
	Complemento       string `json:"complemento"`
	Bairro            string `json:"bairro"`
	Telefone          string `json:"telefone"`
	Email             string `json:"email"`
}
