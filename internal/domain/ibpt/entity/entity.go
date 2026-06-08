package entity

import "time"

// IBPTRate is one row of the IBPT/SCI approximate tax burden table (Lei da
// Transparência), keyed by NCM/EX/UF/versão.
type IBPTRate struct {
	ID               int64
	NCM              string
	Ex               string
	UF               string
	Tipo             int16
	Descricao        string
	NacionalFederal  float64
	ImportadoFederal float64
	Estadual         float64
	Municipal        float64
	VigenciaInicio   *time.Time
	VigenciaFim      *time.Time
	Chave            *string
	Versao           string
	Fonte            string
	CreatedAt        time.Time
}
