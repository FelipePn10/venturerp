package response

import (
	"time"

	"github.com/google/uuid"
)

// FiscalExitResponse is the API representation of an outbound fiscal document (NF-e saída).
type FiscalExitResponse struct {
	ID                      int64                    `json:"id"`
	ChaveAcesso             *string                  `json:"chave_acesso,omitempty"`
	NumeroNF                int64                    `json:"numero_nf"`
	Serie                   string                   `json:"serie"`
	DataEmissao             time.Time                `json:"data_emissao"`
	DataSaida               *time.Time               `json:"data_saida,omitempty"`
	CnpjDestinatario        *string                  `json:"cnpj_destinatario,omitempty"`
	RazaoSocialDestinatario *string                  `json:"razao_social_destinatario,omitempty"`
	IEDestinatario          *string                  `json:"ie_destinatario,omitempty"`
	UFDestinatario          *string                  `json:"uf_destinatario,omitempty"`
	Cfop                    string                   `json:"cfop"`
	NaturezaOperacao        string                   `json:"natureza_operacao"`
	ValorProdutos           float64                  `json:"valor_produtos"`
	ValorFrete              float64                  `json:"valor_frete"`
	ValorSeguro             float64                  `json:"valor_seguro"`
	ValorDesconto           float64                  `json:"valor_desconto"`
	ValorIPI                float64                  `json:"valor_ipi"`
	ValorICMS               float64                  `json:"valor_icms"`
	ValorPIS                float64                  `json:"valor_pis"`
	ValorCOFINS             float64                  `json:"valor_cofins"`
	BaseICMSST              float64                  `json:"base_icms_st"`
	ValorICMSST             float64                  `json:"valor_icms_st"`
	ValorTotal              float64                  `json:"valor_total"`
	SalesOrderCode          *int64                   `json:"sales_order_code,omitempty"`
	SourceType              *string                  `json:"source_type,omitempty"`
	ShipmentLoadCode        *int64                   `json:"shipment_load_code,omitempty"`
	ShipmentCode            *int64                   `json:"shipment_code,omitempty"`
	FiscalCouponNumber      *string                  `json:"fiscal_coupon_number,omitempty"`
	FiscalCouponDate        *time.Time               `json:"fiscal_coupon_date,omitempty"`
	FiscalCouponECFSerial   *string                  `json:"fiscal_coupon_ecf_serial,omitempty"`
	Status                  string                   `json:"status"`
	Protocolo               *string                  `json:"protocolo,omitempty"`
	XmlPath                 *string                  `json:"xml_path,omitempty"`
	DanfePath               *string                  `json:"danfe_path,omitempty"`
	FocusRef                *string                  `json:"focus_ref,omitempty"`
	IsActive                bool                     `json:"is_active"`
	CreatedAt               time.Time                `json:"created_at"`
	UpdatedAt               time.Time                `json:"updated_at"`
	CreatedBy               uuid.UUID                `json:"created_by"`
	Itens                   []FiscalExitItemResponse `json:"itens,omitempty"`
}

// FiscalExitItemResponse is the API representation of an outbound fiscal document line.
type FiscalExitItemResponse struct {
	ID                int64     `json:"id"`
	FiscalExitID      int64     `json:"fiscal_exit_id"`
	Sequence          int       `json:"sequence"`
	ItemCode          *int64    `json:"item_code,omitempty"`
	Ncm               *string   `json:"ncm,omitempty"`
	Cfop              string    `json:"cfop"`
	Quantity          float64   `json:"quantity"`
	UnitPrice         float64   `json:"unit_price"`
	TotalPrice        float64   `json:"total_price"`
	BaseICMS          float64   `json:"base_icms"`
	AliqICMS          float64   `json:"aliq_icms"`
	ValorICMS         float64   `json:"valor_icms"`
	ValorICMSDiferido float64   `json:"valor_icms_diferido"`
	BaseIPI           float64   `json:"base_ipi"`
	AliqIPI           float64   `json:"aliq_ipi"`
	ValorIPI          float64   `json:"valor_ipi"`
	AliqPIS           float64   `json:"aliq_pis"`
	ValorPIS          float64   `json:"valor_pis"`
	AliqCOFINS        float64   `json:"aliq_cofins"`
	ValorCOFINS       float64   `json:"valor_cofins"`
	BaseICMSST        float64   `json:"base_icms_st"`
	AliqICMSST        float64   `json:"aliq_icms_st"`
	ValorICMSST       float64   `json:"valor_icms_st"`
	MVA               float64   `json:"mva"`
	CstICMS           *string   `json:"cst_icms,omitempty"`
	CstIPI            *string   `json:"cst_ipi,omitempty"`
	CstPIS            *string   `json:"cst_pis,omitempty"`
	CstCOFINS         *string   `json:"cst_cofins,omitempty"`
	OrigemMercadoria  string    `json:"origem_mercadoria"`
	Description       *string   `json:"description,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// FiscalEntryResponse is the API representation of an inbound fiscal document (NF-e entrada).
type FiscalEntryResponse struct {
	ID                  int64                     `json:"id"`
	ChaveAcesso         *string                   `json:"chave_acesso,omitempty"`
	NumeroNF            int64                     `json:"numero_nf"`
	Serie               string                    `json:"serie"`
	Modelo              string                    `json:"modelo"`
	DataEmissao         time.Time                 `json:"data_emissao"`
	DataEntrada         time.Time                 `json:"data_entrada"`
	CnpjEmitente        string                    `json:"cnpj_emitente"`
	RazaoSocialEmitente string                    `json:"razao_social_emitente"`
	IEEmitente          *string                   `json:"ie_emitente,omitempty"`
	UFEmitente          *string                   `json:"uf_emitente,omitempty"`
	ValorProdutos       float64                   `json:"valor_produtos"`
	ValorFrete          float64                   `json:"valor_frete"`
	ValorSeguro         float64                   `json:"valor_seguro"`
	ValorDesconto       float64                   `json:"valor_desconto"`
	ValorIPI            float64                   `json:"valor_ipi"`
	ValorICMS           float64                   `json:"valor_icms"`
	ValorPIS            float64                   `json:"valor_pis"`
	ValorCOFINS         float64                   `json:"valor_cofins"`
	ValorTotal          float64                   `json:"valor_total"`
	TipoDocumento       string                    `json:"tipo_documento"`
	PurchaseOrderCode   *int64                    `json:"purchase_order_code,omitempty"`
	SupplierCode        *int64                    `json:"supplier_code,omitempty"`
	CteCode             *int64                    `json:"cte_code,omitempty"`
	Status              string                    `json:"status"`
	XmlPath             *string                   `json:"xml_path,omitempty"`
	Notes               *string                   `json:"notes,omitempty"`
	IsActive            bool                      `json:"is_active"`
	CreatedAt           time.Time                 `json:"created_at"`
	UpdatedAt           time.Time                 `json:"updated_at"`
	CreatedBy           uuid.UUID                 `json:"created_by"`
	Itens               []FiscalEntryItemResponse `json:"itens,omitempty"`
}

// FiscalEntryItemResponse is the API representation of an inbound fiscal document line.
type FiscalEntryItemResponse struct {
	ID                int64     `json:"id"`
	FiscalEntryID     int64     `json:"fiscal_entry_id"`
	Sequence          int       `json:"sequence"`
	ItemCode          *int64    `json:"item_code,omitempty"`
	Ncm               *string   `json:"ncm,omitempty"`
	Cfop              string    `json:"cfop"`
	Quantity          float64   `json:"quantity"`
	UnitPrice         float64   `json:"unit_price"`
	TotalPrice        float64   `json:"total_price"`
	BaseICMS          float64   `json:"base_icms"`
	AliqICMS          float64   `json:"aliq_icms"`
	ValorICMS         float64   `json:"valor_icms"`
	BaseIPI           float64   `json:"base_ipi"`
	AliqIPI           float64   `json:"aliq_ipi"`
	ValorIPI          float64   `json:"valor_ipi"`
	ValorPIS          float64   `json:"valor_pis"`
	ValorCOFINS       float64   `json:"valor_cofins"`
	CstICMS           *string   `json:"cst_icms,omitempty"`
	CstIPI            *string   `json:"cst_ipi,omitempty"`
	CstPIS            *string   `json:"cst_pis,omitempty"`
	CstCOFINS         *string   `json:"cst_cofins,omitempty"`
	GeraCreditoICMS   bool      `json:"gera_credito_icms"`
	GeraCreditoIPI    bool      `json:"gera_credito_ipi"`
	GeraCreditoPIS    bool      `json:"gera_credito_pis"`
	GeraCreditoCOFINS bool      `json:"gera_credito_cofins"`
	Description       *string   `json:"description,omitempty"`
	Notes             *string   `json:"notes,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// FiscalConfigResponse is the API representation of the fiscal configuration.
type FiscalConfigResponse struct {
	ID                        int64     `json:"id"`
	CnpjEmpresa               string    `json:"cnpj_empresa"`
	RazaoSocial               string    `json:"razao_social"`
	IEEmpresa                 *string   `json:"ie_empresa,omitempty"`
	RegimeTributario          string    `json:"regime_tributario"`
	UFEmpresa                 string    `json:"uf_empresa"`
	IcmsInternoAliquota       float64   `json:"icms_interno_aliquota"`
	IcmsDiferimentoPercentual float64   `json:"icms_diferimento_percentual"`
	FocusNfeToken             *string   `json:"focus_nfe_token,omitempty"`
	FocusNfeAmbiente          string    `json:"focus_nfe_ambiente"`
	JurosMes                  float64   `json:"juros_mes"`
	MultaAtraso               float64   `json:"multa_atraso"`
	VencimentoIcmsDia         int       `json:"vencimento_icms_dia"`
	VencimentoIPIDia          int       `json:"vencimento_ipi_dia"`
	VencimentoPisCofinsDia    int       `json:"vencimento_pis_cofins_dia"`
	Logradouro                string    `json:"logradouro"`
	Numero                    string    `json:"numero"`
	Complemento               *string   `json:"complemento,omitempty"`
	Bairro                    string    `json:"bairro"`
	Municipio                 string    `json:"municipio"`
	CodigoMunicipio           string    `json:"codigo_municipio"`
	CEP                       string    `json:"cep"`
	Telefone                  *string   `json:"telefone,omitempty"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
	UpdatedBy                 uuid.UUID `json:"updated_by"`
}

// FiscalCTeResponse is the API representation of a CT-e.
type FiscalCTeResponse struct {
	ID                  int64     `json:"id"`
	ChaveAcesso         *string   `json:"chave_acesso,omitempty"`
	NumeroCTe           int64     `json:"numero_cte"`
	Serie               string    `json:"serie"`
	DataEmissao         time.Time `json:"data_emissao"`
	DataEntrada         time.Time `json:"data_entrada"`
	CnpjEmitente        string    `json:"cnpj_emitente"`
	RazaoSocialEmitente string    `json:"razao_social_emitente"`
	IEEmitente          *string   `json:"ie_emitente,omitempty"`
	UFEmitente          *string   `json:"uf_emitente,omitempty"`
	Cfop                string    `json:"cfop"`
	ValorFrete          float64   `json:"valor_frete"`
	ValorSeguro         float64   `json:"valor_seguro"`
	ValorOutros         float64   `json:"valor_outros"`
	ValorTotal          float64   `json:"valor_total"`
	ValorICMS           float64   `json:"valor_icms"`
	BaseICMS            float64   `json:"base_icms"`
	AliqICMS            float64   `json:"aliq_icms"`
	CstICMS             *string   `json:"cst_icms,omitempty"`
	TipoRateio          string    `json:"tipo_rateio"`
	FiscalEntryID       *int64    `json:"fiscal_entry_id,omitempty"`
	Status              string    `json:"status"`
	FocusRef            *string   `json:"focus_ref,omitempty"`
	Protocolo           *string   `json:"protocolo,omitempty"`
	EmissionData        *string   `json:"emission_data,omitempty"`
	XmlPath             *string   `json:"xml_path,omitempty"`
	Notes               *string   `json:"notes,omitempty"`
	IsActive            bool      `json:"is_active"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// CartaCorrecaoResponse is the API representation of a correction letter (CC-e).
type CartaCorrecaoResponse struct {
	ID            int64     `json:"id"`
	FiscalExitID  int64     `json:"fiscal_exit_id"`
	NumeroSeq     int       `json:"numero_seq"`
	TextoCorrecao string    `json:"texto_correcao"`
	FocusRef      *string   `json:"focus_ref,omitempty"`
	Status        string    `json:"status"`
	Protocolo     *string   `json:"protocolo,omitempty"`
	ChaveEvento   *string   `json:"chave_evento,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// NcmTaxTableResponse is the API representation of an NCM tax table entry.
type NcmTaxTableResponse struct {
	ID          int64     `json:"id"`
	Ncm         string    `json:"ncm"`
	AliqIPI     float64   `json:"aliq_ipi"`
	AliqPis     float64   `json:"aliq_pis"`
	AliqCofins  float64   `json:"aliq_cofins"`
	CstPis      string    `json:"cst_pis"`
	CstCofins   string    `json:"cst_cofins"`
	CstIPI      string    `json:"cst_ipi"`
	Description *string   `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}
