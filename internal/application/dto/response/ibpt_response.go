package response

import "time"

// IBPTRateResponse is the API representation of an IBPT approximate tax rate.
type IBPTRateResponse struct {
	ID               int64      `json:"id"`
	NCM              string     `json:"ncm"`
	Ex               string     `json:"ex"`
	UF               string     `json:"uf"`
	Tipo             int16      `json:"tipo"`
	Descricao        string     `json:"descricao"`
	NacionalFederal  float64    `json:"nacional_federal"`
	ImportadoFederal float64    `json:"importado_federal"`
	Estadual         float64    `json:"estadual"`
	Municipal        float64    `json:"municipal"`
	VigenciaInicio   *time.Time `json:"vigencia_inicio,omitempty"`
	VigenciaFim      *time.Time `json:"vigencia_fim,omitempty"`
	Chave            *string    `json:"chave,omitempty"`
	Versao           string     `json:"versao"`
	Fonte            string     `json:"fonte"`
	CreatedAt        time.Time  `json:"created_at"`
}
