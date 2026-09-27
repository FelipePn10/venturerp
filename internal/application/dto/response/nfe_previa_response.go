package response

// PreviaNFeResponse é a nota fiscal montada e conferida ANTES de ir para a
// SEFAZ.
//
// Emitir é irreversível: nota autorizada só sai do ar por cancelamento, que tem
// prazo, justificativa e rastro. O time precisava de um lugar para ler a nota
// inteira — emitente, destinatário, cada item com NCM/CFOP/CST, cada imposto,
// os totais e as parcelas — e uma lista do que ainda impede a emissão, em vez de
// descobrir o problema na recusa da SEFAZ.
type PreviaNFeResponse struct {
	FiscalExitID     int64  `json:"fiscal_exit_id"`
	NumeroNF         int64  `json:"numero_nf"`
	Serie            string `json:"serie"`
	Status           string `json:"status"`
	DataEmissao      string `json:"data_emissao"`
	DataSaida        string `json:"data_saida,omitempty"`
	NaturezaOperacao string `json:"natureza_operacao"`
	Cfop             string `json:"cfop"`
	// Ambiente é "producao" ou "homologacao": em homologação a nota é um teste
	// sem valor fiscal, e quem confere precisa saber disso antes de assinar.
	Ambiente         string `json:"ambiente"`
	TipoOperacao     string `json:"tipo_operacao"`
	ConsumidorFinal  bool   `json:"consumidor_final"`
	SalesOrderCode   *int64 `json:"sales_order_code,omitempty"`
	ShipmentLoadCode *int64 `json:"shipment_load_code,omitempty"`

	Emitente     PreviaParteNFe     `json:"emitente"`
	Destinatario PreviaParteNFe     `json:"destinatario"`
	Itens        []PreviaItemNFe    `json:"itens"`
	Totais       PreviaTotaisNFe    `json:"totais"`
	Pagamento    PreviaPagamentoNFe `json:"pagamento"`

	// Pendencias lista o que está errado ou ausente. Nível "IMPEDE" trava a
	// emissão; "ATENCAO" deixa emitir, mas tem consequência.
	Pendencias    []PreviaPendenciaNFe `json:"pendencias"`
	PodeAutorizar bool                 `json:"pode_autorizar"`
	// PayloadEnviado é o JSON exato que será transmitido, para conferência
	// técnica e para o suporte não precisar adivinhar o que foi enviado.
	PayloadEnviado string `json:"payload_enviado"`
}

type PreviaParteNFe struct {
	Documento       string `json:"documento"`
	Nome            string `json:"nome"`
	IE              string `json:"ie,omitempty"`
	Logradouro      string `json:"logradouro,omitempty"`
	Numero          string `json:"numero,omitempty"`
	Complemento     string `json:"complemento,omitempty"`
	Bairro          string `json:"bairro,omitempty"`
	Municipio       string `json:"municipio,omitempty"`
	CodigoMunicipio string `json:"codigo_municipio,omitempty"`
	UF              string `json:"uf,omitempty"`
	CEP             string `json:"cep,omitempty"`
	Email           string `json:"email,omitempty"`
	Telefone        string `json:"telefone,omitempty"`
}

type PreviaItemNFe struct {
	Sequence    int     `json:"sequence"`
	ItemCode    *int64  `json:"item_code,omitempty"`
	Descricao   string  `json:"descricao"`
	Ncm         string  `json:"ncm"`
	Cfop        string  `json:"cfop"`
	UM          string  `json:"um"`
	Quantidade  float64 `json:"quantidade"`
	ValorUnit   float64 `json:"valor_unitario"`
	ValorTotal  float64 `json:"valor_total"`
	Origem      string  `json:"origem_mercadoria"`
	CstICMS     string  `json:"cst_icms"`
	BaseICMS    float64 `json:"base_icms"`
	AliqICMS    float64 `json:"aliq_icms"`
	ValorICMS   float64 `json:"valor_icms"`
	CstIPI      string  `json:"cst_ipi"`
	AliqIPI     float64 `json:"aliq_ipi"`
	ValorIPI    float64 `json:"valor_ipi"`
	CstPIS      string  `json:"cst_pis"`
	ValorPIS    float64 `json:"valor_pis"`
	CstCOFINS   string  `json:"cst_cofins"`
	ValorCOFINS float64 `json:"valor_cofins"`
	BaseICMSST  float64 `json:"base_icms_st"`
	ValorICMSST float64 `json:"valor_icms_st"`
	MVA         float64 `json:"mva"`
}

type PreviaTotaisNFe struct {
	ValorProdutos float64 `json:"valor_produtos"`
	ValorFrete    float64 `json:"valor_frete"`
	ValorSeguro   float64 `json:"valor_seguro"`
	ValorDesconto float64 `json:"valor_desconto"`
	ValorIPI      float64 `json:"valor_ipi"`
	ValorICMS     float64 `json:"valor_icms"`
	BaseICMSST    float64 `json:"base_icms_st"`
	ValorICMSST   float64 `json:"valor_icms_st"`
	ValorPIS      float64 `json:"valor_pis"`
	ValorCOFINS   float64 `json:"valor_cofins"`
	ValorTotalNF  float64 `json:"valor_total_nf"`
	// Conferencia repete a conta do total em texto, porque "por que deu esse
	// valor?" é a pergunta que mais aparece na conferência de uma nota.
	Conferencia string `json:"conferencia"`
}

type PreviaPagamentoNFe struct {
	CondicaoCode      *int64             `json:"condicao_code,omitempty"`
	CondicaoDescricao string             `json:"condicao_descricao,omitempty"`
	Origem            string             `json:"origem"`
	Parcelas          []PreviaParcelaNFe `json:"parcelas"`
	Aviso             string             `json:"aviso,omitempty"`
}

type PreviaParcelaNFe struct {
	Numero           int16   `json:"numero"`
	Percentual       float64 `json:"percentual"`
	Valor            float64 `json:"valor"`
	Vencimento       string  `json:"vencimento"`
	Descricao        string  `json:"descricao"`
	FormaPagamentoNF string  `json:"forma_pagamento_nf"`
	Estimado         bool    `json:"estimado"`
}

type PreviaPendenciaNFe struct {
	Nivel        string `json:"nivel"`
	Campo        string `json:"campo"`
	Mensagem     string `json:"mensagem"`
	ComoResolver string `json:"como_resolver"`
}
