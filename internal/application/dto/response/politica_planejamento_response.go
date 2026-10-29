package response

import "time"

// PoliticaPlanejamentoResponse é uma regra da VPLA0310. Campo ausente herda.
type PoliticaPlanejamentoResponse struct {
	ID int64 `json:"id"`
	// Escopo: ITEM ou CLASSIFICACAO.
	Escopo              string    `json:"escopo"`
	ItemCode            *int64    `json:"item_code,omitempty"`
	ItemBusinessCode    string    `json:"item_business_code,omitempty"`
	ItemName            string    `json:"item_name,omitempty"`
	ClassificationID    *int64    `json:"classification_id,omitempty"`
	ClassificationCode  string    `json:"classification_code,omitempty"`
	ClassificationLabel string    `json:"classification_label,omitempty"`
	Estrategia          *string   `json:"estrategia,omitempty"`
	ConsumoDiasAntes    *int      `json:"consumo_dias_antes,omitempty"`
	ConsumoDiasDepois   *int      `json:"consumo_dias_depois,omitempty"`
	HorizonteFirmeDias  *int      `json:"horizonte_firme_dias,omitempty"`
	TempoSegurancaDias  *int      `json:"tempo_seguranca_dias,omitempty"`
	Observacao          *string   `json:"observacao,omitempty"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// PoliticaEfetivaResponse é a política que vale para o item, com a origem de
// cada valor (do item, da classificação X ou parâmetro da empresa).
type PoliticaEfetivaResponse struct {
	ItemBusinessCode    string                 `json:"item_business_code"`
	ItemName            string                 `json:"item_name"`
	Estrategia          string                 `json:"estrategia"`
	EstrategiaDescricao string                 `json:"estrategia_descricao"`
	ConsumoDiasAntes    int                    `json:"consumo_dias_antes"`
	ConsumoDiasDepois   int                    `json:"consumo_dias_depois"`
	HorizonteFirmeDias  int                    `json:"horizonte_firme_dias"`
	TempoSegurancaDias  int                    `json:"tempo_seguranca_dias"`
	Origem              OrigemPoliticaResponse `json:"origem"`
}

type OrigemPoliticaResponse struct {
	Estrategia     string `json:"estrategia"`
	Janela         string `json:"janela"`
	HorizonteFirme string `json:"horizonte_firme"`
	TempoSeguranca string `json:"tempo_seguranca"`
}
