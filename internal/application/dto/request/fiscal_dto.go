package request

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

type CreateFiscalEntryDTO struct {
	ChaveAcesso         *string                    `json:"chave_acesso,omitempty"`
	NumeroNF            int64                      `json:"numero_nf"`
	Serie               string                     `json:"serie"`
	Modelo              string                     `json:"modelo"`
	DataEmissao         string                     `json:"data_emissao"`
	DataEntrada         string                     `json:"data_entrada"`
	CnpjEmitente        string                     `json:"cnpj_emitente"`
	RazaoSocialEmitente string                     `json:"razao_social_emitente"`
	IEEmitente          *string                    `json:"ie_emitente,omitempty"`
	UFEmitente          *string                    `json:"uf_emitente,omitempty"`
	ValorProdutos       float64                    `json:"valor_produtos"`
	ValorFrete          float64                    `json:"valor_frete"`
	ValorSeguro         float64                    `json:"valor_seguro"`
	ValorDesconto       float64                    `json:"valor_desconto"`
	ValorIPI            float64                    `json:"valor_ipi"`
	ValorICMS           float64                    `json:"valor_icms"`
	ValorPIS            float64                    `json:"valor_pis"`
	ValorCOFINS         float64                    `json:"valor_cofins"`
	ValorTotal          float64                    `json:"valor_total"`
	TipoDocumento       string                     `json:"tipo_documento"`
	PurchaseOrderCode   *int64                     `json:"purchase_order_code,omitempty"`
	CteCode             *int64                     `json:"cte_code,omitempty"`
	Notes               *string                    `json:"notes,omitempty"`
	Itens               []CreateFiscalEntryItemDTO `json:"itens"`
	// SupplierCode é o fornecedor do cadastro. Vazio, o sistema procura pelo
	// CNPJ do emitente.
	SupplierCode *int64 `json:"supplier_code,omitempty"`
	// Parcelas da nota. Vazias, a nota nasce com uma parcela única para 30 dias.
	Parcelas []FiscalEntryInstallmentDTO `json:"parcelas,omitempty"`
	// Tipo de operação de entrada da nota (padrão dos itens).
	EntryOperationCode *int64 `json:"entry_operation_code,omitempty"`
	// Retenções na fonte (nota de serviço / prestação com retenção).
	ValorRetPIS    decimal.Decimal `json:"valor_ret_pis"`
	ValorRetCOFINS decimal.Decimal `json:"valor_ret_cofins"`
	ValorRetCSLL   decimal.Decimal `json:"valor_ret_csll"`
	ValorIRRF      decimal.Decimal `json:"valor_irrf"`
	ValorRetPrev   decimal.Decimal `json:"valor_ret_prev"`
	ValorISSRet    decimal.Decimal `json:"valor_iss_ret"`
}

type CreateFiscalEntryItemDTO struct {
	Sequence              int     `json:"sequence"`
	ItemCode              *int64  `json:"item_code,omitempty"`
	SupplierItemCode      *string `json:"supplier_item_code,omitempty"`
	UOM                   *string `json:"uom,omitempty"`
	Ncm                   *string `json:"ncm,omitempty"`
	Cfop                  string  `json:"cfop"`
	Quantity              float64 `json:"quantity"`
	UnitPrice             float64 `json:"unit_price"`
	TotalPrice            float64 `json:"total_price"`
	BaseICMS              float64 `json:"base_icms"`
	AliqICMS              float64 `json:"aliq_icms"`
	ValorICMS             float64 `json:"valor_icms"`
	BaseIPI               float64 `json:"base_ipi"`
	AliqIPI               float64 `json:"aliq_ipi"`
	ValorIPI              float64 `json:"valor_ipi"`
	ValorPIS              float64 `json:"valor_pis"`
	ValorCOFINS           float64 `json:"valor_cofins"`
	CstICMS               *string `json:"cst_icms,omitempty"`
	CstIPI                *string `json:"cst_ipi,omitempty"`
	CstPIS                *string `json:"cst_pis,omitempty"`
	CstCOFINS             *string `json:"cst_cofins,omitempty"`
	GeraCreditoICMS       bool    `json:"gera_credito_icms"`
	GeraCreditoIPI        bool    `json:"gera_credito_ipi"`
	GeraCreditoPIS        bool    `json:"gera_credito_pis"`
	GeraCreditoCOFINS     bool    `json:"gera_credito_cofins"`
	Description           *string `json:"description,omitempty"`
	Notes                 *string `json:"notes,omitempty"`
	PlanoContasID         *int64  `json:"plano_contas_id,omitempty"`
	CentroCustoID         *int64  `json:"centro_custo_id,omitempty"`
	EntryOperationCode    *int64  `json:"entry_operation_code,omitempty"`
	WarehouseID           *int64  `json:"warehouse_id,omitempty"`
	PurchaseOrderItemCode *int64  `json:"purchase_order_item_code,omitempty"`
}

type ApproveFiscalEntryDTO struct {
	ID int64 `json:"id"`
}

type UploadNFEDTO struct {
	XmlContent        string `json:"xml_content"`
	PurchaseOrderCode *int64 `json:"purchase_order_code,omitempty"`
	// DataEntrada é a data em que a mercadoria entrou (AAAA-MM-DD). Vazia, vale
	// a data de saída/entrada do XML ou, na falta dela, a emissão.
	DataEntrada *string `json:"data_entrada,omitempty"`
}

// SaveFiscalEntryConciliationDTO grava a conciliação dos itens da nota com o
// cadastro, o plano de contas de cada item e, opcionalmente, as parcelas com a
// distribuição por plano de contas.
type SaveFiscalEntryConciliationDTO struct {
	Itens []FiscalEntryItemConciliationDTO `json:"itens"`
	// Tipo de operação de entrada da nota (padrão dos itens sem operação
	// própria) e pedido de compra da nota.
	EntryOperationCode *int64 `json:"entry_operation_code,omitempty"`
	PurchaseOrderCode  *int64 `json:"purchase_order_code,omitempty"`
	// Parcelas, quando enviadas, substituem as parcelas da nota. Ausentes, as
	// parcelas são mantidas e só a distribuição é refeita quando
	// RecalcularDistribuicao for verdadeiro (ou quando a classificação mudou).
	Parcelas               []FiscalEntryInstallmentDTO `json:"parcelas,omitempty"`
	RecalcularDistribuicao bool                        `json:"recalcular_distribuicao,omitempty"`
}

type FiscalEntryItemConciliationDTO struct {
	ID            int64  `json:"id"`
	ItemCode      *int64 `json:"item_code,omitempty"`
	PlanoContasID *int64 `json:"plano_contas_id,omitempty"`
	CentroCustoID *int64 `json:"centro_custo_id,omitempty"`
	// LembrarVinculo memoriza o "de/para" código do fornecedor → item do
	// cadastro: a próxima nota do mesmo fornecedor já vem conciliada.
	LembrarVinculo bool `json:"lembrar_vinculo,omitempty"`
	// FatorConversao converte a unidade da nota para a do cadastro (ex.: nota
	// em CX com 12 unidades → 12). Vazio = 1.
	FatorConversao *decimal.Decimal `json:"fator_conversao,omitempty"`
	// Tipo de operação de entrada do item (o "TES"): CFOP de entrada, estoque,
	// financeiro e créditos. Vazio = o da nota ou o do pedido.
	EntryOperationCode *int64 `json:"entry_operation_code,omitempty"`
	WarehouseID        *int64 `json:"warehouse_id,omitempty"`
	// Linha do pedido de compra (3-way). Zero desfaz o vínculo.
	PurchaseOrderItemCode *int64 `json:"purchase_order_item_code,omitempty"`
	// CFOP de entrada informado à mão (prevalece sobre o calculado).
	CfopEntrada *string `json:"cfop_entrada,omitempty"`
}

// CancelFiscalEntryDTO cancela a nota (aprovada: estorna tudo).
type CancelFiscalEntryDTO struct {
	Motivo string `json:"motivo"`
}

type FiscalEntryInstallmentDTO struct {
	Numero         int                        `json:"numero"`
	Documento      *string                    `json:"documento,omitempty"`
	DataVencimento string                     `json:"data_vencimento"`
	Valor          decimal.Decimal            `json:"valor"`
	FormaPagamento *string                    `json:"forma_pagamento,omitempty"`
	Distribuicao   []FiscalEntryAllocationDTO `json:"distribuicao,omitempty"`
}

type FiscalEntryAllocationDTO struct {
	PlanoContasID int64           `json:"plano_contas_id"`
	CentroCustoID *int64          `json:"centro_custo_id,omitempty"`
	Valor         decimal.Decimal `json:"valor"`
}

type CreateFiscalExitDTO struct {
	NumeroNF                int64   `json:"numero_nf"`
	Serie                   string  `json:"serie"`
	DataEmissao             string  `json:"data_emissao"`
	DataSaida               *string `json:"data_saida,omitempty"`
	CnpjDestinatario        *string `json:"cnpj_destinatario,omitempty"`
	RazaoSocialDestinatario *string `json:"razao_social_destinatario,omitempty"`
	IEDestinatario          *string `json:"ie_destinatario,omitempty"`
	UFDestinatario          *string `json:"uf_destinatario,omitempty"`
	TipoPessoa              *string `json:"tipo_pessoa,omitempty"`
	// Endereço do destinatário. Quando a nota nasce de um pedido de venda ou de
	// uma carga, o que vier vazio aqui é preenchido pelo endereço de entrega do
	// cliente (ou pelo de cobrança, quando não há um de entrega).
	DestLogradouro        *string                   `json:"dest_logradouro,omitempty"`
	DestNumero            *string                   `json:"dest_numero,omitempty"`
	DestComplemento       *string                   `json:"dest_complemento,omitempty"`
	DestBairro            *string                   `json:"dest_bairro,omitempty"`
	DestMunicipio         *string                   `json:"dest_municipio,omitempty"`
	DestCodigoMunicipio   *string                   `json:"dest_codigo_municipio,omitempty"`
	DestCEP               *string                   `json:"dest_cep,omitempty"`
	DestEmail             *string                   `json:"dest_email,omitempty"`
	DestTelefone          *string                   `json:"dest_telefone,omitempty"`
	CustomerCode          *int64                    `json:"customer_code,omitempty"`
	Cfop                  string                    `json:"cfop"`
	NaturezaOperacao      string                    `json:"natureza_operacao"`
	ValorProdutos         float64                   `json:"valor_produtos"`
	ValorFrete            float64                   `json:"valor_frete"`
	ValorSeguro           float64                   `json:"valor_seguro"`
	ValorDesconto         float64                   `json:"valor_desconto"`
	SalesOrderCode        *int64                    `json:"sales_order_code,omitempty"`
	SourceType            *string                   `json:"source_type,omitempty"`
	ShipmentLoadCode      *int64                    `json:"shipment_load_code,omitempty"`
	ShipmentCode          *int64                    `json:"shipment_code,omitempty"`
	FiscalCouponNumber    *string                   `json:"fiscal_coupon_number,omitempty"`
	FiscalCouponDate      *string                   `json:"fiscal_coupon_date,omitempty"`
	FiscalCouponECFSerial *string                   `json:"fiscal_coupon_ecf_serial,omitempty"`
	Itens                 []CreateFiscalExitItemDTO `json:"itens"`
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
	// ICMS-ST: when MvaPct > 0 the engine computes Substituição Tributária for
	// this item. MvaPct is a ratio (0.40 = 40%); it may already be the adjusted
	// MVA for interstate operations. AliqInternaDestinoST overrides the
	// destination internal rate, and RedBaseSTPct reduces the ST base (ratio).
	MvaPct               float64 `json:"mva_pct,omitempty"`
	AliqInternaDestinoST float64 `json:"aliq_interna_destino_st,omitempty"`
	RedBaseSTPct         float64 `json:"red_base_st_pct,omitempty"`
	// Linha do pedido de venda faturada, unidade comercial e código do produto
	// impressos na nota (preenchidos quando a nota nasce do pedido).
	SalesOrderItemCode *int64  `json:"sales_order_item_code,omitempty"`
	UnidadeComercial   *string `json:"unidade_comercial,omitempty"`
	CodigoProduto      *string `json:"codigo_produto,omitempty"`
}

type CreateFiscalExitFromLoadDTO struct {
	LoadCode                int64                        `json:"load_code"`
	Serie                   string                       `json:"serie"`
	DataEmissao             string                       `json:"data_emissao"`
	DataSaida               *string                      `json:"data_saida,omitempty"`
	CnpjDestinatario        *string                      `json:"cnpj_destinatario,omitempty"`
	RazaoSocialDestinatario *string                      `json:"razao_social_destinatario,omitempty"`
	IEDestinatario          *string                      `json:"ie_destinatario,omitempty"`
	UFDestinatario          *string                      `json:"uf_destinatario,omitempty"`
	TipoPessoa              *string                      `json:"tipo_pessoa,omitempty"`
	Cfop                    string                       `json:"cfop"`
	NaturezaOperacao        string                       `json:"natureza_operacao"`
	ValorFrete              float64                      `json:"valor_frete"`
	ValorSeguro             float64                      `json:"valor_seguro"`
	ValorDesconto           float64                      `json:"valor_desconto"`
	OrigemMercadoria        string                       `json:"origem_mercadoria,omitempty"`
	ItemOverrides           []FiscalExitLoadItemOverride `json:"item_overrides,omitempty"`
}

type FiscalExitLoadItemOverride struct {
	ShipmentCode         *int64   `json:"shipment_code,omitempty"`
	ItemCode             int64    `json:"item_code"`
	UnitPrice            *float64 `json:"unit_price,omitempty"`
	Ncm                  *string  `json:"ncm,omitempty"`
	Cfop                 *string  `json:"cfop,omitempty"`
	OrigemMercadoria     *string  `json:"origem_mercadoria,omitempty"`
	Description          *string  `json:"description,omitempty"`
	MvaPct               float64  `json:"mva_pct,omitempty"`
	AliqInternaDestinoST float64  `json:"aliq_interna_destino_st,omitempty"`
	RedBaseSTPct         float64  `json:"red_base_st_pct,omitempty"`
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
	Ncm         string  `json:"ncm"`
	AliqIPI     float64 `json:"aliq_ipi"`
	AliqPis     float64 `json:"aliq_pis"`
	AliqCofins  float64 `json:"aliq_cofins"`
	CstPis      string  `json:"cst_pis"`
	CstCofins   string  `json:"cst_cofins"`
	CstIPI      string  `json:"cst_ipi"`
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
	NumeroCTe           int64   `json:"numero_cte"`
	Serie               string  `json:"serie"`
	DataEmissao         string  `json:"data_emissao"`
	DataEntrada         string  `json:"data_entrada"`
	CnpjEmitente        string  `json:"cnpj_emitente"`
	RazaoSocialEmitente string  `json:"razao_social_emitente"`
	IEEmitente          *string `json:"ie_emitente,omitempty"`
	UFEmitente          *string `json:"uf_emitente,omitempty"`
	Cfop                string  `json:"cfop"`
	ValorFrete          float64 `json:"valor_frete"`
	ValorSeguro         float64 `json:"valor_seguro"`
	ValorOutros         float64 `json:"valor_outros"`
	ValorTotal          float64 `json:"valor_total"`
	ValorICMS           float64 `json:"valor_icms"`
	BaseICMS            float64 `json:"base_icms"`
	AliqICMS            float64 `json:"aliq_icms"`
	CstICMS             *string `json:"cst_icms,omitempty"`
	TipoRateio          string  `json:"tipo_rateio"`
	FiscalEntryID       *int64  `json:"fiscal_entry_id,omitempty"`
	Notes               *string `json:"notes,omitempty"`
	// EmissionData carries the full CT-e emission detail (natureza, tipo_cte,
	// tipo_servico, modal, remetente, destinatário, tomador, municípios,
	// produto predominante, valores). Required only when the CT-e will be
	// authorized at SEFAZ; stored as-is and used by the authorize use case.
	EmissionData json.RawMessage `json:"emission_data,omitempty"`
}

var _ = time.Now
