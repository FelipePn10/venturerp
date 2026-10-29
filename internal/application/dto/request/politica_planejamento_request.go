package request

// PoliticaPlanejamentoDTO cria ou altera a política de planejamento de um
// item (ItemCode, código de negócio) ou de uma classificação
// (ClassificationID) — um dos dois, só na criação. Campo ausente herda do
// nível de cima (classificação e, por fim, os parâmetros 46 a 49).
type PoliticaPlanejamentoDTO struct {
	ItemCode           TextCode `json:"item_code,omitempty"`
	ClassificationID   *int64   `json:"classification_id,omitempty"`
	Estrategia         *string  `json:"estrategia,omitempty"`
	ConsumoDiasAntes   *int     `json:"consumo_dias_antes,omitempty"`
	ConsumoDiasDepois  *int     `json:"consumo_dias_depois,omitempty"`
	HorizonteFirmeDias *int     `json:"horizonte_firme_dias,omitempty"`
	TempoSegurancaDias *int     `json:"tempo_seguranca_dias,omitempty"`
	Observacao         *string  `json:"observacao,omitempty"`
}
