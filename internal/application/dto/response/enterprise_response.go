package response

import "github.com/google/uuid"

// EnterpriseResponse is the API representation of an enterprise.
type EnterpriseResponse struct {
	ID        int       `json:"id"`
	Code      int       `json:"code"`
	Name      string    `json:"name"`
	CreatedBy uuid.UUID `json:"created_by"`
}

// IdentificacaoEmpresaResponse: a identificação da empresa (VEMP0100).
type IdentificacaoEmpresaResponse struct {
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
