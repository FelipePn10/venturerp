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
	DestLogradouro          *string                  `json:"dest_logradouro,omitempty"`
	DestNumero              *string                  `json:"dest_numero,omitempty"`
	DestComplemento         *string                  `json:"dest_complemento,omitempty"`
	DestBairro              *string                  `json:"dest_bairro,omitempty"`
	DestMunicipio           *string                  `json:"dest_municipio,omitempty"`
	DestCodigoMunicipio     *string                  `json:"dest_codigo_municipio,omitempty"`
	DestCEP                 *string                  `json:"dest_cep,omitempty"`
	DestEmail               *string                  `json:"dest_email,omitempty"`
	DestTelefone            *string                  `json:"dest_telefone,omitempty"`
	CustomerCode            *int64                   `json:"customer_code,omitempty"`
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
	// Avisos do que aconteceu depois da SEFAZ (estoque, contabilidade).
	Warnings []string `json:"warnings,omitempty"`
	// Devolução de compra: finalidade 4, nota de entrada referenciada.
	Finalidade      int     `json:"finalidade"`
	NFeReferenciada *string `json:"nfe_referenciada,omitempty"`
	FiscalEntryID   *int64  `json:"fiscal_entry_id,omitempty"`
	SupplierCode    *int64  `json:"supplier_code,omitempty"`
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
	Warnings            []string                  `json:"warnings,omitempty"`

	// Nota completa (XML, conciliação e financeiro).
	NaturezaOperacao          *string                          `json:"natureza_operacao,omitempty"`
	CnpjDestinatario          *string                          `json:"cnpj_destinatario,omitempty"`
	Protocolo                 *string                          `json:"protocolo,omitempty"`
	ValorICMSST               float64                          `json:"valor_icms_st"`
	ValorOutras               float64                          `json:"valor_outras"`
	ModalidadeFrete           *string                          `json:"modalidade_frete,omitempty"`
	InformacoesComplementares *string                          `json:"informacoes_complementares,omitempty"`
	SemPagamento              bool                             `json:"sem_pagamento"`
	SupplierName              *string                          `json:"supplier_name,omitempty"`
	ApprovedAt                *time.Time                       `json:"approved_at,omitempty"`
	Parcelas                  []FiscalEntryInstallmentResponse `json:"parcelas,omitempty"`
	TotaisPorConta            []FiscalEntryAccountTotal        `json:"totais_por_conta,omitempty"`
	Pendencias                []FiscalEntryPendencia           `json:"pendencias,omitempty"`
	PodeAprovar               bool                             `json:"pode_aprovar"`
	ItensConciliados          int                              `json:"itens_conciliados"`
	ItensClassificados        int                              `json:"itens_classificados"`

	EntryOperationCode *int64                   `json:"entry_operation_code,omitempty"`
	BaseIBSCBS         float64                  `json:"base_ibscbs"`
	ValorIBS           float64                  `json:"valor_ibs"`
	ValorCBS           float64                  `json:"valor_cbs"`
	ValorIS            float64                  `json:"valor_is"`
	ValorRetPIS        float64                  `json:"valor_ret_pis"`
	ValorRetCOFINS     float64                  `json:"valor_ret_cofins"`
	ValorRetCSLL       float64                  `json:"valor_ret_csll"`
	BaseIRRF           float64                  `json:"base_irrf"`
	ValorIRRF          float64                  `json:"valor_irrf"`
	BaseRetPrev        float64                  `json:"base_ret_prev"`
	ValorRetPrev       float64                  `json:"valor_ret_prev"`
	ValorISSRet        float64                  `json:"valor_iss_ret"`
	TotalRetencoes     float64                  `json:"total_retencoes"`
	ValorAPagar        float64                  `json:"valor_a_pagar"`
	StockStatus        string                   `json:"stock_status,omitempty"`
	CancelledAt        *time.Time               `json:"cancelled_at,omitempty"`
	CancelReason       *string                  `json:"cancel_reason,omitempty"`
	Retencoes          []FiscalEntryRetencao    `json:"retencoes,omitempty"`
	Divergencias       []FiscalEntryDivergencia `json:"divergencias,omitempty"`
}

// FiscalEntryRetencao é um imposto retido que vira título a recolher.
type FiscalEntryRetencao struct {
	Tipo       string  `json:"tipo"`
	Descricao  string  `json:"descricao"`
	Valor      float64 `json:"valor"`
	Vencimento string  `json:"vencimento"`
}

// FiscalEntryDivergencia é uma diferença entre a nota e o que o ERP esperava.
type FiscalEntryDivergencia struct {
	Nivel     string `json:"nivel"`
	Item      int    `json:"item"`
	Tipo      string `json:"tipo"`
	Mensagem  string `json:"mensagem"`
	Esperado  string `json:"esperado,omitempty"`
	Informado string `json:"informado,omitempty"`
}

// FiscalEntryInstallmentResponse é uma parcela (duplicata) da nota de entrada.
type FiscalEntryInstallmentResponse struct {
	ID             int64                               `json:"id"`
	Numero         int                                 `json:"numero"`
	Documento      *string                             `json:"documento,omitempty"`
	DataVencimento string                              `json:"data_vencimento"`
	Valor          float64                             `json:"valor"`
	FormaPagamento *string                             `json:"forma_pagamento,omitempty"`
	Origem         string                              `json:"origem"`
	ContaPagarID   *int64                              `json:"conta_pagar_id,omitempty"`
	Distribuicao   []FiscalEntryInstallmentAllocResult `json:"distribuicao"`
}

type FiscalEntryInstallmentAllocResult struct {
	PlanoContasID int64   `json:"plano_contas_id"`
	CentroCustoID *int64  `json:"centro_custo_id,omitempty"`
	Valor         float64 `json:"valor"`
}

// FiscalEntryAccountTotal é quanto da nota vai para cada plano de contas.
type FiscalEntryAccountTotal struct {
	PlanoContasID     int64   `json:"plano_contas_id"`
	PlanoContasCodigo string  `json:"plano_contas_codigo,omitempty"`
	PlanoContasNome   string  `json:"plano_contas_nome,omitempty"`
	CentroCustoID     *int64  `json:"centro_custo_id,omitempty"`
	CentroCustoNome   string  `json:"centro_custo_nome,omitempty"`
	Valor             float64 `json:"valor"`
	Percentual        float64 `json:"percentual"`
}

type FiscalEntryPendencia struct {
	Nivel    string `json:"nivel"`
	Campo    string `json:"campo"`
	Mensagem string `json:"mensagem"`
}

// FiscalEntryItemSuggestion é um item do cadastro sugerido para conciliar uma
// linha da nota, com o porquê da sugestão.
type FiscalEntryItemSuggestion struct {
	ItemCode       int64   `json:"item_code"`
	Name           string  `json:"name"`
	UOM            string  `json:"uom,omitempty"`
	NCM            string  `json:"ncm,omitempty"`
	Score          float64 `json:"score"`
	Motivo         string  `json:"motivo"`
	ItemSupplierID *int64  `json:"item_supplier_id,omitempty"`
	IsActive       bool    `json:"is_active"`
}

// FiscalEntryItemResponse is the API representation of an inbound fiscal document line.
type FiscalEntryItemResponse struct {
	ID                 int64      `json:"id"`
	FiscalEntryID      int64      `json:"fiscal_entry_id"`
	Sequence           int        `json:"sequence"`
	ItemCode           *int64     `json:"item_code,omitempty"`
	ItemSupplierID     *int64     `json:"item_supplier_id,omitempty"`
	SupplierItemCode   *string    `json:"supplier_item_code,omitempty"`
	ResolutionStrategy *string    `json:"resolution_strategy,omitempty"`
	ResolvedAt         *time.Time `json:"resolved_at,omitempty"`
	Ncm                *string    `json:"ncm,omitempty"`
	Cfop               string     `json:"cfop"`
	Quantity           float64    `json:"quantity"`
	UnitPrice          float64    `json:"unit_price"`
	TotalPrice         float64    `json:"total_price"`
	BaseICMS           float64    `json:"base_icms"`
	AliqICMS           float64    `json:"aliq_icms"`
	ValorICMS          float64    `json:"valor_icms"`
	BaseIPI            float64    `json:"base_ipi"`
	AliqIPI            float64    `json:"aliq_ipi"`
	ValorIPI           float64    `json:"valor_ipi"`
	ValorPIS           float64    `json:"valor_pis"`
	ValorCOFINS        float64    `json:"valor_cofins"`
	CstICMS            *string    `json:"cst_icms,omitempty"`
	CstIPI             *string    `json:"cst_ipi,omitempty"`
	CstPIS             *string    `json:"cst_pis,omitempty"`
	CstCOFINS          *string    `json:"cst_cofins,omitempty"`
	GeraCreditoICMS    bool       `json:"gera_credito_icms"`
	GeraCreditoIPI     bool       `json:"gera_credito_ipi"`
	GeraCreditoPIS     bool       `json:"gera_credito_pis"`
	GeraCreditoCOFINS  bool       `json:"gera_credito_cofins"`
	Description        *string    `json:"description,omitempty"`
	Notes              *string    `json:"notes,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`

	UOM               *string  `json:"uom,omitempty"`
	EAN               *string  `json:"ean,omitempty"`
	CEST              *string  `json:"cest,omitempty"`
	Origem            *string  `json:"origem,omitempty"`
	ValorFrete        float64  `json:"valor_frete"`
	ValorSeguro       float64  `json:"valor_seguro"`
	ValorDesconto     float64  `json:"valor_desconto"`
	ValorOutras       float64  `json:"valor_outras"`
	BaseICMSST        float64  `json:"base_icms_st"`
	ValorICMSST       float64  `json:"valor_icms_st"`
	ValorContabil     float64  `json:"valor_contabil"`
	FatorConversao    *float64 `json:"fator_conversao,omitempty"`
	QuantidadeEstoque *float64 `json:"quantidade_estoque,omitempty"`
	PedidoCompraXML   *string  `json:"pedido_compra_xml,omitempty"`
	PlanoContasID     *int64   `json:"plano_contas_id,omitempty"`
	CentroCustoID     *int64   `json:"centro_custo_id,omitempty"`
	ItemName          *string  `json:"item_name,omitempty"`
	ItemUOM           *string  `json:"item_uom,omitempty"`
	PlanoContasCodigo *string  `json:"plano_contas_codigo,omitempty"`
	PlanoContasNome   *string  `json:"plano_contas_nome,omitempty"`
	CentroCustoNome   *string  `json:"centro_custo_nome,omitempty"`

	CfopEntrada           *string `json:"cfop_entrada,omitempty"`
	EntryOperationCode    *int64  `json:"entry_operation_code,omitempty"`
	EntryOperationName    *string `json:"entry_operation_name,omitempty"`
	MovimentaEstoque      bool    `json:"movimenta_estoque"`
	GeraFinanceiro        bool    `json:"gera_financeiro"`
	WarehouseID           *int64  `json:"warehouse_id,omitempty"`
	WarehouseName         *string `json:"warehouse_name,omitempty"`
	PurchaseOrderCode     *int64  `json:"purchase_order_code,omitempty"`
	PurchaseOrderItemCode *int64  `json:"purchase_order_item_code,omitempty"`
	PurchaseOrderNumber   *int64  `json:"purchase_order_number,omitempty"`
	PurchaseOrderSequence *int32  `json:"purchase_order_sequence,omitempty"`
	QtdRecebidaAntes      float64 `json:"qtd_recebida_antes"`
	StockMovementID       *int64  `json:"stock_movement_id,omitempty"`
	CustoAquisicao        float64 `json:"custo_aquisicao"`
	ItemNCM               *string `json:"item_ncm,omitempty"`
	CSTIBSCBS             *string `json:"cst_ibscbs,omitempty"`
	ClassTrib             *string `json:"cclass_trib,omitempty"`
	BaseIBSCBS            float64 `json:"base_ibscbs"`
	ValorIBS              float64 `json:"valor_ibs"`
	ValorCBS              float64 `json:"valor_cbs"`
	ValorIS               float64 `json:"valor_is"`
	GeraCreditoIBSCBS     bool    `json:"gera_credito_ibscbs"`
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
	FocusNfeConfigured        bool      `json:"focus_nfe_configured"`
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
	BrandColor                *string   `json:"brand_color,omitempty"`
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

// PedidoParaFaturarResponse é o pedido de venda visto pelo faturamento: o que
// já foi faturado, o que está em nota ainda não autorizada e o que falta, com
// cliente, representante/comissão e condição de pagamento que a nota herdará.
type PedidoParaFaturarResponse struct {
	SalesOrderCode       int64                   `json:"sales_order_code"`
	OrderNumber          int64                   `json:"order_number"`
	Status               string                  `json:"status"`
	EmissionDate         string                  `json:"emission_date"`
	CustomerCode         *int64                  `json:"customer_code,omitempty"`
	CustomerName         string                  `json:"customer_name"`
	CustomerDocument     string                  `json:"customer_document"`
	CustomerIE           string                  `json:"customer_ie,omitempty"`
	CustomerUF           string                  `json:"customer_uf,omitempty"`
	CustomerCity         string                  `json:"customer_city,omitempty"`
	EnderecoCompleto     bool                    `json:"endereco_completo"`
	RepresentativeCode   *int64                  `json:"representative_code,omitempty"`
	RepresentativeName   string                  `json:"representative_name,omitempty"`
	CommissionPct        float64                 `json:"commission_pct"`
	Comissoes            []ComissaoDoPedido      `json:"comissoes,omitempty"`
	PaymentTermCode      *int64                  `json:"payment_term_code,omitempty"`
	PaymentTermDescricao string                  `json:"payment_term_descricao,omitempty"`
	FreightType          *string                 `json:"freight_type,omitempty"`
	FreightValue         float64                 `json:"freight_value"`
	InsuranceValue       float64                 `json:"insurance_value"`
	DiscountValue        float64                 `json:"discount_value"`
	CfopSugerido         string                  `json:"cfop_sugerido"`
	NaturezaSugerida     string                  `json:"natureza_sugerida"`
	Itens                []PedidoItemParaFaturar `json:"itens"`
	ValorPendente        float64                 `json:"valor_pendente"`
	Impedimentos         []string                `json:"impedimentos,omitempty"`
	PodeFaturar          bool                    `json:"pode_faturar"`
	NotasDoPedido        []NotaDoPedido          `json:"notas_do_pedido,omitempty"`
}

type ComissaoDoPedido struct {
	RepresentativeCode int64   `json:"representative_code"`
	RepresentativeName string  `json:"representative_name"`
	Papel              string  `json:"papel"`
	CommissionPct      float64 `json:"commission_pct"`
}

type PedidoItemParaFaturar struct {
	SalesOrderItemCode  int64   `json:"sales_order_item_code"`
	Sequence            int     `json:"sequence"`
	ItemCode            int64   `json:"item_code"`
	ItemName            string  `json:"item_name"`
	NCM                 string  `json:"ncm,omitempty"`
	UOM                 string  `json:"uom,omitempty"`
	Origem              string  `json:"origem"`
	QuantidadePedida    float64 `json:"quantidade_pedida"`
	QuantidadeCancelada float64 `json:"quantidade_cancelada"`
	QuantidadeFaturada  float64 `json:"quantidade_faturada"`
	QuantidadeEmNota    float64 `json:"quantidade_em_nota"`
	QuantidadePendente  float64 `json:"quantidade_pendente"`
	PrecoUnitario       float64 `json:"preco_unitario"`
	DescontoPct         float64 `json:"desconto_pct"`
	IPIPct              float64 `json:"ipi_pct"`
	ValorPendente       float64 `json:"valor_pendente"`
}

type NotaDoPedido struct {
	ID         int64   `json:"id"`
	NumeroNF   int64   `json:"numero_nf"`
	Status     string  `json:"status"`
	ValorTotal float64 `json:"valor_total"`
}

// NFeRecebidaResponse é uma NF-e emitida contra o CNPJ da empresa.
type NFeRecebidaResponse struct {
	ChaveAcesso   string    `json:"chave_acesso"`
	CNPJEmitente  string    `json:"cnpj_emitente"`
	NomeEmitente  string    `json:"nome_emitente"`
	NumeroNF      *int64    `json:"numero_nf,omitempty"`
	Serie         *string   `json:"serie,omitempty"`
	DataEmissao   *string   `json:"data_emissao,omitempty"`
	ValorTotal    float64   `json:"valor_total"`
	Situacao      string    `json:"situacao"`
	Manifestacao  *string   `json:"manifestacao,omitempty"`
	XMLCompleto   bool      `json:"xml_completo"`
	FiscalEntryID *int64    `json:"fiscal_entry_id,omitempty"`
	SyncedAt      time.Time `json:"synced_at"`
	// Prazo da manifestação conclusiva (emissão + 180 dias) para as notas que
	// ainda não a têm; AlertaPrazo quando faltam 30 dias ou menos (ou venceu).
	PrazoManifestacao *string `json:"prazo_manifestacao,omitempty"`
	DiasParaPrazo     *int    `json:"dias_para_prazo,omitempty"`
	AlertaPrazo       bool    `json:"alerta_prazo"`
}

// DFeStatusResponse é a situação da sincronização das notas recebidas.
type DFeStatusResponse struct {
	Automatico       bool       `json:"automatico"`
	SincronizadoEm   *time.Time `json:"sincronizado_em,omitempty"`
	UltimaTentativa  *time.Time `json:"ultima_tentativa,omitempty"`
	UltimoErro       *string    `json:"ultimo_erro,omitempty"`
	PrazoProximo     int        `json:"prazo_proximo"`
	PrazoVencido     int        `json:"prazo_vencido"`
	IntervaloMinutos int        `json:"intervalo_minutos"`
	AlertaPrazoDias  int        `json:"alerta_prazo_dias"`
}

// LinhaPedidoCompraResponse é uma linha de pedido de compra que pode ser
// vinculada à linha da nota de entrada.
type LinhaPedidoCompraResponse struct {
	Code              int64   `json:"code"`
	PurchaseOrderCode int64   `json:"purchase_order_code"`
	OrderNumber       int64   `json:"order_number"`
	Sequence          int     `json:"sequence"`
	ItemCode          int64   `json:"item_code"`
	Status            string  `json:"status"`
	RequestedQty      float64 `json:"requested_qty"`
	ReceivedQty       float64 `json:"received_qty"`
	InvoicedQty       float64 `json:"invoiced_qty"`
	SaldoAFaturar     float64 `json:"saldo_a_faturar"`
	UnitPrice         float64 `json:"unit_price"`
	FatorEstoque      float64 `json:"fator_estoque"`
	WarehouseID       *int64  `json:"warehouse_id,omitempty"`
}

// FreteResponse é o frete sobre compras (CT-e da transportadora).
type FreteResponse struct {
	ID                 int64                   `json:"id"`
	ChaveCTe           *string                 `json:"chave_cte,omitempty"`
	Numero             int64                   `json:"numero"`
	Serie              string                  `json:"serie"`
	DataEmissao        string                  `json:"data_emissao"`
	CNPJTransportadora string                  `json:"cnpj_transportadora"`
	NomeTransportadora string                  `json:"nome_transportadora"`
	UFTransportadora   *string                 `json:"uf_transportadora,omitempty"`
	SupplierCode       *int64                  `json:"supplier_code,omitempty"`
	SupplierName       *string                 `json:"supplier_name,omitempty"`
	CFOP               *string                 `json:"cfop,omitempty"`
	ValorFrete         float64                 `json:"valor_frete"`
	BaseICMS           float64                 `json:"base_icms"`
	AliqICMS           float64                 `json:"aliq_icms"`
	ValorICMS          float64                 `json:"valor_icms"`
	CreditaICMS        bool                    `json:"credita_icms"`
	CustoFrete         float64                 `json:"custo_frete"`
	TipoRateio         string                  `json:"tipo_rateio"`
	DataVencimento     string                  `json:"data_vencimento"`
	Status             string                  `json:"status"`
	ContaPagarID       *int64                  `json:"conta_pagar_id,omitempty"`
	Observacao         *string                 `json:"observacao,omitempty"`
	CreatedAt          time.Time               `json:"created_at"`
	LancadoEm          *time.Time              `json:"lancado_em,omitempty"`
	CanceladoEm        *time.Time              `json:"cancelado_em,omitempty"`
	CancelReason       *string                 `json:"cancel_reason,omitempty"`
	Notas              []FreteNotaResponse     `json:"notas"`
	Alocacoes          []FreteAlocacaoResponse `json:"alocacoes,omitempty"`
	// Previa é o rateio que o lançamento faria agora (frete pendente).
	Previa []FreteAlocacaoResponse `json:"previa,omitempty"`
	Avisos []string                `json:"avisos,omitempty"`
}

type FreteNotaResponse struct {
	FiscalEntryID int64   `json:"fiscal_entry_id"`
	NumeroNF      int64   `json:"numero_nf"`
	Serie         string  `json:"serie"`
	Emitente      string  `json:"emitente"`
	ValorTotal    float64 `json:"valor_total"`
	Status        string  `json:"status"`
	ChaveAcesso   *string `json:"chave_acesso,omitempty"`
}

type FreteAlocacaoResponse struct {
	FiscalEntryID     int64   `json:"fiscal_entry_id"`
	FiscalEntryItemID int64   `json:"fiscal_entry_item_id"`
	ItemCode          *int64  `json:"item_code,omitempty"`
	Descricao         string  `json:"descricao"`
	Valor             float64 `json:"valor"`
	ValorEstoque      float64 `json:"valor_estoque"`
	ValorDespesa      float64 `json:"valor_despesa"`
}

// DevolucaoPreviaResponse: o que pode ser devolvido de uma nota de entrada.
type DevolucaoPreviaResponse struct {
	FiscalEntryID int64                 `json:"fiscal_entry_id"`
	NumeroNF      int64                 `json:"numero_nf"`
	Serie         string                `json:"serie"`
	Fornecedor    string                `json:"fornecedor"`
	Status        string                `json:"status"`
	ChaveAcesso   *string               `json:"chave_acesso,omitempty"`
	Itens         []DevolucaoItemPrevia `json:"itens"`
	// Endereco do fornecedor lido do XML; Faltando diz o que a tela tem de completar.
	Endereco DevolucaoEndereco `json:"endereco"`
}

type DevolucaoEndereco struct {
	Logradouro      string   `json:"logradouro"`
	Numero          string   `json:"numero"`
	Complemento     string   `json:"complemento"`
	Bairro          string   `json:"bairro"`
	Municipio       string   `json:"municipio"`
	CodigoMunicipio string   `json:"codigo_municipio"`
	CEP             string   `json:"cep"`
	Faltando        []string `json:"faltando"`
}

type DevolucaoItemPrevia struct {
	FiscalEntryItemID int64   `json:"fiscal_entry_item_id"`
	Sequence          int     `json:"sequence"`
	ItemCode          *int64  `json:"item_code,omitempty"`
	Descricao         string  `json:"descricao"`
	UOM               *string `json:"uom,omitempty"`
	Quantidade        float64 `json:"quantidade"`
	Devolvida         float64 `json:"devolvida"`
	Disponivel        float64 `json:"disponivel"`
	ValorUnitario     float64 `json:"valor_unitario"`
	CFOPDevolucao     string  `json:"cfop_devolucao"`
}
