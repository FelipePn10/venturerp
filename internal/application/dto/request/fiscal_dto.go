package request

import "time"

type CreateFiscalEntryDTO struct {
	ChaveAcesso         *string                   `json:"chave_acesso,omitempty"`
	NumeroNF            int64                     `json:"numero_nf"`
	Serie               string                    `json:"serie"`
	Modelo              string                    `json:"modelo"`
	DataEmissao         string                    `json:"data_emissao"`
	DataEntrada         string                    `json:"data_entrada"`
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
	CteCode             *int64                    `json:"cte_code,omitempty"`
	Notes               *string                   `json:"notes,omitempty"`
	Itens               []CreateFiscalEntryItemDTO `json:"itens"`
}

type CreateFiscalEntryItemDTO struct {
	Sequence          int     `json:"sequence"`
	ItemCode          *int64  `json:"item_code,omitempty"`
	Ncm               *string `json:"ncm,omitempty"`
	Cfop              string  `json:"cfop"`
	Quantity          float64 `json:"quantity"`
	UnitPrice         float64 `json:"unit_price"`
	TotalPrice        float64 `json:"total_price"`
	BaseICMS          float64 `json:"base_icms"`
	AliqICMS          float64 `json:"aliq_icms"`
	ValorICMS         float64 `json:"valor_icms"`
	BaseIPI           float64 `json:"base_ipi"`
	AliqIPI           float64 `json:"aliq_ipi"`
	ValorIPI          float64 `json:"valor_ipi"`
	ValorPIS          float64 `json:"valor_pis"`
	ValorCOFINS       float64 `json:"valor_cofins"`
	CstICMS           *string `json:"cst_icms,omitempty"`
	CstIPI            *string `json:"cst_ipi,omitempty"`
	CstPIS            *string `json:"cst_pis,omitempty"`
	CstCOFINS         *string `json:"cst_cofins,omitempty"`
	GeraCreditoICMS   bool    `json:"gera_credito_icms"`
	GeraCreditoIPI    bool    `json:"gera_credito_ipi"`
	GeraCreditoPIS    bool    `json:"gera_credito_pis"`
	GeraCreditoCOFINS bool    `json:"gera_credito_cofins"`
	Description       *string `json:"description,omitempty"`
	Notes             *string `json:"notes,omitempty"`
}

type UpdateFiscalEntryDTO struct {
	ID          int64   `json:"id"`
	Notes       *string `json:"notes,omitempty"`
	XmlPath     *string `json:"xml_path,omitempty"`
}

type ApproveFiscalEntryDTO struct {
	ID int64 `json:"id"`
}

type UploadNFEDTO struct {
	XmlContent string `json:"xml_content"`
}

type CreateFiscalExitDTO struct {
	NumeroNF                int64                     `json:"numero_nf"`
	Serie                   string                    `json:"serie"`
	DataEmissao             string                    `json:"data_emissao"`
	DataSaida               *string                   `json:"data_saida,omitempty"`
	CnpjDestinatario        *string                   `json:"cnpj_destinatario,omitempty"`
	RazaoSocialDestinatario *string                   `json:"razao_social_destinatario,omitempty"`
	IEDestinatario          *string                   `json:"ie_destinatario,omitempty"`
	UFDestinatario          *string                   `json:"uf_destinatario,omitempty"`
	TipoPessoa              *string                   `json:"tipo_pessoa,omitempty"`
	Cfop                    string                    `json:"cfop"`
	NaturezaOperacao        string                    `json:"natureza_operacao"`
	ValorProdutos           float64                   `json:"valor_produtos"`
	ValorFrete              float64                   `json:"valor_frete"`
	ValorSeguro             float64                   `json:"valor_seguro"`
	ValorDesconto           float64                   `json:"valor_desconto"`
	SalesOrderCode          *int64                    `json:"sales_order_code,omitempty"`
	Itens                   []CreateFiscalExitItemDTO `json:"itens"`
}

type CreateFiscalExitItemDTO struct {
	Sequence         int     `json:"sequence"`
	ItemCode         *int64  `json:"item_code,omitempty"`
	Ncm              *string `json:"ncm,omitempty"`
	Cfop             string  `json:"cfop"`
	Quantity         float64 `json:"quantity"`
	UnitPrice        float64 `json:"unit_price"`
	TotalPrice       float64 `json:"total_price"`
	OrigemMercadoria string  `json:"origem_mercadoria"`
	Description      *string `json:"description,omitempty"`
}

type UpdateFiscalConfigDTO struct {
	CnpjEmpresa               string  `json:"cnpj_empresa"`
	RazaoSocial               string  `json:"razao_social"`
	IEEmpresa                 *string `json:"ie_empresa,omitempty"`
	RegimeTributario          string  `json:"regime_tributario"`
	UFEmpresa                 string  `json:"uf_empresa"`
	IcmsInternoAliquota       float64 `json:"icms_interno_aliquota"`
	IcmsDiferimentoPercentual float64 `json:"icms_diferimento_percentual"`
	FocusNfeToken             *string `json:"focus_nfe_token,omitempty"`
	FocusNfeAmbiente          string  `json:"focus_nfe_ambiente"`
	JurosMes                  float64 `json:"juros_mes"`
	MultaAtraso               float64 `json:"multa_atraso"`
	VencimentoIcmsDia         int     `json:"vencimento_icms_dia"`
	VencimentoIPIDia          int     `json:"vencimento_ipi_dia"`
	VencimentoPisCofinsDia    int     `json:"vencimento_pis_cofins_dia"`
	// Endereço do emitente
	Logradouro      string  `json:"logradouro"`
	Numero          string  `json:"numero"`
	Complemento     *string `json:"complemento,omitempty"`
	Bairro          string  `json:"bairro"`
	Municipio       string  `json:"municipio"`
	CodigoMunicipio string  `json:"codigo_municipio"`
	CEP             string  `json:"cep"`
	Telefone        *string `json:"telefone,omitempty"`
}

type UpsertNcmTaxDTO struct {
	Ncm        string  `json:"ncm"`
	AliqIPI    float64 `json:"aliq_ipi"`
	AliqPis    float64 `json:"aliq_pis"`
	AliqCofins float64 `json:"aliq_cofins"`
	CstPis     string  `json:"cst_pis"`
	CstCofins  string  `json:"cst_cofins"`
	CstIPI     string  `json:"cst_ipi"`
	Description *string `json:"description,omitempty"`
}

type UpsertICMSInterstateDTO struct {
	OriginUF      string  `json:"origin_uf"`
	DestinationUF string  `json:"destination_uf"`
	AliqICMS      float64 `json:"aliq_icms"`
}

type UpsertICMSInternalDTO struct {
	UF       string  `json:"uf"`
	AliqICMS float64 `json:"aliq_icms"`
	AliqFCP  float64 `json:"aliq_fcp"`
}

type CreateCTeDTO struct {
	NumeroCTe           int64    `json:"numero_cte"`
	Serie               string   `json:"serie"`
	DataEmissao         string   `json:"data_emissao"`
	DataEntrada         string   `json:"data_entrada"`
	CnpjEmitente        string   `json:"cnpj_emitente"`
	RazaoSocialEmitente string   `json:"razao_social_emitente"`
	IEEmitente          *string  `json:"ie_emitente,omitempty"`
	UFEmitente          *string  `json:"uf_emitente,omitempty"`
	Cfop                string   `json:"cfop"`
	ValorFrete          float64  `json:"valor_frete"`
	ValorSeguro         float64  `json:"valor_seguro"`
	ValorOutros         float64  `json:"valor_outros"`
	ValorTotal          float64  `json:"valor_total"`
	ValorICMS           float64  `json:"valor_icms"`
	BaseICMS            float64  `json:"base_icms"`
	AliqICMS            float64  `json:"aliq_icms"`
	CstICMS             *string  `json:"cst_icms,omitempty"`
	TipoRateio          string   `json:"tipo_rateio"`
	FiscalEntryID       *int64   `json:"fiscal_entry_id,omitempty"`
	Notes               *string  `json:"notes,omitempty"`
}

var _ = time.Now
