package entity

import "time"

type FiscalCTe struct {
	ID                  int64       `json:"id"`
	ChaveAcesso         *string     `json:"chave_acesso,omitempty"`
	NumeroCTe           int64       `json:"numero_cte"`
	Serie               string      `json:"serie"`
	DataEmissao         time.Time   `json:"data_emissao"`
	DataEntrada         time.Time   `json:"data_entrada"`
	CnpjEmitente        string      `json:"cnpj_emitente"`
	RazaoSocialEmitente string      `json:"razao_social_emitente"`
	IEEmitente          *string     `json:"ie_emitente,omitempty"`
	UFEmitente          *string     `json:"uf_emitente,omitempty"`
	Cfop                string      `json:"cfop"`
	ValorFrete          float64     `json:"valor_frete"`
	ValorSeguro         float64     `json:"valor_seguro"`
	ValorOutros         float64     `json:"valor_outros"`
	ValorTotal          float64     `json:"valor_total"`
	ValorICMS           float64     `json:"valor_icms"`
	BaseICMS            float64     `json:"base_icms"`
	AliqICMS            float64     `json:"aliq_icms"`
	CstICMS             *string     `json:"cst_icms,omitempty"`
	TipoRateio          string      `json:"tipo_rateio"` // VALOR ou PESO
	FiscalEntryID       *int64      `json:"fiscal_entry_id,omitempty"`
	Status              string      `json:"status"`
	FocusRef            *string     `json:"focus_ref,omitempty"`
	Protocolo           *string     `json:"protocolo,omitempty"`
	EmissionData        *string     `json:"emission_data,omitempty"` // JSON com o detalhe de emissão (partes, modal, municípios)
	XmlPath             *string     `json:"xml_path,omitempty"`
	Notes               *string     `json:"notes,omitempty"`
	IsActive            bool        `json:"is_active"`
	CreatedAt           time.Time   `json:"created_at"`
	UpdatedAt           time.Time   `json:"updated_at"`
	CreatedBy           interface{} `json:"created_by"`
}

type CartaCorrecao struct {
	ID            int64       `json:"id"`
	FiscalExitID  int64       `json:"fiscal_exit_id"`
	NumeroSeq     int         `json:"numero_seq"`
	TextoCorrecao string      `json:"texto_correcao"`
	FocusRef      *string     `json:"focus_ref,omitempty"`
	Status        string      `json:"status"`
	Protocolo     *string     `json:"protocolo,omitempty"`
	ChaveEvento   *string     `json:"chave_evento,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	CreatedBy     interface{} `json:"created_by"`
}
