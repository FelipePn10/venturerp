package focusnfe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	baseURLProducao    = "https://api.focusnfe.com.br/v2"
	baseURLHomologacao = "https://homologacao.focusnfe.com.br/v2"
)

type Client struct {
	token     string
	baseURL   string
	httpCli   *http.Client
	onRequest func(endpoint, method, reqBody, respBody string, statusCode, durationMs int)
	configErr error
}

func NewClient(token, ambiente string) *Client {
	base := baseURLHomologacao
	providerEnvironment := strings.ToLower(strings.TrimSpace(ambiente))
	if providerEnvironment == "producao" {
		base = baseURLProducao
	}
	client := &Client{
		token:   token,
		baseURL: base,
		httpCli: &http.Client{Timeout: 30 * time.Second},
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("DATA_ENVIRONMENT")), "training") && providerEnvironment == "producao" {
		client.configErr = fmt.Errorf("Focus NF-e production environment is forbidden while DATA_ENVIRONMENT=training")
	}
	return client
}

func (c *Client) WithLogger(fn func(endpoint, method, reqBody, respBody string, statusCode, durationMs int)) *Client {
	c.onRequest = fn
	return c
}

func (c *Client) do(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	if c.configErr != nil {
		return nil, 0, c.configErr
	}
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling request: %w", err)
		}
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}
	req.SetBasicAuth(c.token, "")
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := c.httpCli.Do(req)
	durationMs := int(time.Since(start).Milliseconds())
	if err != nil {
		if c.onRequest != nil {
			c.onRequest(path, method, string(reqBody), err.Error(), 0, durationMs)
		}
		return nil, 0, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	if c.onRequest != nil {
		c.onRequest(path, method, string(reqBody), string(respBytes), resp.StatusCode, durationMs)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var apiErr struct {
			Codigo   string `json:"codigo"`
			Mensagem string `json:"mensagem"`
		}
		if json.Unmarshal(respBytes, &apiErr) == nil && apiErr.Mensagem != "" {
			if apiErr.Codigo != "" {
				return respBytes, resp.StatusCode, fmt.Errorf("Focus NF-e HTTP %d (%s): %s", resp.StatusCode, apiErr.Codigo, apiErr.Mensagem)
			}
			return respBytes, resp.StatusCode, fmt.Errorf("Focus NF-e HTTP %d: %s", resp.StatusCode, apiErr.Mensagem)
		}

		return respBytes, resp.StatusCode, fmt.Errorf("Focus NF-e HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
	}

	return respBytes, resp.StatusCode, nil
}

// NFe payload structures (subset needed for emission)

type NFEEmitente struct {
	CNPJ             string `json:"cnpj"`
	Nome             string `json:"nome"`
	Logradouro       string `json:"logradouro"`
	Numero           string `json:"numero"`
	Bairro           string `json:"bairro"`
	Municipio        string `json:"municipio"`
	UF               string `json:"uf"`
	CEP              string `json:"cep"`
	Telefone         string `json:"telefone,omitempty"`
	RegimeTributario int    `json:"regime_tributario"`
}

type NFEDestinatario struct {
	CNPJCPF     string  `json:"cnpj_cpf"`
	Nome        string  `json:"nome"`
	Logradouro  string  `json:"logradouro,omitempty"`
	Numero      string  `json:"numero,omitempty"`
	Bairro      string  `json:"bairro,omitempty"`
	Municipio   string  `json:"municipio,omitempty"`
	UF          string  `json:"uf,omitempty"`
	CEP         string  `json:"cep,omitempty"`
	Email       string  `json:"email,omitempty"`
	IndicadorIE int     `json:"indicador_ie"`
	IE          *string `json:"ie,omitempty"`
}

type NFEItem struct {
	NumeroItem                     int      `json:"numero_item"`
	CodigoProduto                  string   `json:"codigo_produto"`
	Descricao                      string   `json:"descricao"`
	CodigoNCM                      string   `json:"codigo_ncm"`
	CFOP                           string   `json:"cfop"`
	UnidadeComercial               string   `json:"unidade_comercial"`
	QuantidadeComercial            float64  `json:"quantidade_comercial"`
	ValorUnitarioComercial         float64  `json:"valor_unitario_comercial"`
	ValorBruto                     float64  `json:"valor_bruto"`
	CodigoSituacaoTributariaICMS   string   `json:"codigo_situacao_tributaria_icms"`
	ModalidadeBaseCalculoICMS      int      `json:"modalidade_determinacao_bc_icms"`
	ValorBaseCalculoICMS           float64  `json:"valor_base_calculo_icms"`
	AliquotaICMS                   float64  `json:"aliquota_icms"`
	ValorICMS                      float64  `json:"valor_icms"`
	PercentualDiferimento          *float64 `json:"percentual_diferimento,omitempty"`
	ValorICMSDiferido              *float64 `json:"valor_icms_diferido,omitempty"`
	ModalidadeBaseCalculoICMSST    *int     `json:"modalidade_determinacao_bc_icms_st,omitempty"`
	BaseCalculoICMSST              *float64 `json:"valor_base_calculo_icms_st,omitempty"`
	AliquotaICMSST                 *float64 `json:"aliquota_icms_st,omitempty"`
	ValorICMSST                    *float64 `json:"valor_icms_st,omitempty"`
	PercentualMVAICMSST            *float64 `json:"percentual_margem_valor_adicionado_icms_st,omitempty"`
	CodigoSituacaoTributariaIPI    string   `json:"codigo_situacao_tributaria_ipi"`
	AliquotaIPI                    float64  `json:"aliquota_ipi"`
	ValorIPI                       float64  `json:"valor_ipi"`
	CodigoSituacaoTributariaPIS    string   `json:"codigo_situacao_tributaria_pis"`
	AliquotaPIS                    float64  `json:"aliquota_pis"`
	ValorPIS                       float64  `json:"valor_pis"`
	CodigoSituacaoTributariaCOFINS string   `json:"codigo_situacao_tributaria_cofins"`
	AliquotaCOFINS                 float64  `json:"aliquota_cofins"`
	ValorCOFINS                    float64  `json:"valor_cofins"`
	OrigemMercadoria               int      `json:"origem_mercadoria"`
	CEST                           string   `json:"cest,omitempty"`
	// Rateio do frete, seguro e desconto da nota (a SEFAZ soma os itens).
	ValorFrete    float64 `json:"valor_frete,omitempty"`
	ValorSeguro   float64 `json:"valor_seguro,omitempty"`
	ValorDesconto float64 `json:"valor_desconto,omitempty"`
	BaseIPI       float64 `json:"ipi_base_calculo,omitempty"`
	// Reforma tributária (grupo IBSCBS, obrigatório desde 2026).
	IBSCBS *NFEItemIBSCBS `json:"-"`
	// DFeReferenciado (por item): na devolução, a chave da nota de origem e o
	// número do item nela — a SEFAZ rejeita a devolução sem ele (321).
	DFeRefChave string `json:"-"`
	DFeRefItem  int    `json:"-"`
}

// NFEItemIBSCBS é o grupo IBS/CBS do item (NT 2025.002).
type NFEItemIBSCBS struct {
	CST, ClassTrib                           string
	Base                                     float64
	AliqIBSUF, ValorIBSUF, AliqIBSMun        float64
	ValorIBSMun, ValorIBS, AliqCBS, ValorCBS float64
}

type NFEFormaPagamento struct {
	FormaPagamento string  `json:"forma_pagamento"`
	Valor          float64 `json:"valor"`
}

// NFEDuplicata e a parcela da fatura que viaja na NF-e (grupo cobr/dup).
//
// Uma venda a prazo sem duplicata sai da SEFAZ como se fosse a vista: o cliente
// recebe a nota sem saber quanto paga em cada vencimento, e o boleto emitido
// depois nao casa com o documento fiscal.
type NFEDuplicata struct {
	Numero         string  `json:"numero"`
	DataVencimento string  `json:"data_vencimento"`
	Valor          float64 `json:"valor"`
}

type NFEPayload struct {
	NaturezaOperacao  string              `json:"natureza_operacao"`
	DataEmissao       string              `json:"data_emissao"`
	TipoDocumento     int                 `json:"tipo_documento"`
	LocalDestino      int                 `json:"local_destino"`
	FinalidadeEmissao int                 `json:"finalidade_emissao"`
	ConsumidorFinal   int                 `json:"consumidor_final"`
	PresencaComprador int                 `json:"presenca_comprador"`
	Emitente          NFEEmitente         `json:"emitente"`
	Destinatario      NFEDestinatario     `json:"destinatario"`
	Items             []NFEItem           `json:"items"`
	FormaPagamento    []NFEFormaPagamento `json:"forma_pagamento"`
	Duplicatas        []NFEDuplicata      `json:"duplicatas,omitempty"`
	// NotasReferenciadas: a nota de origem (devolução, complementar, ajuste).
	NotasReferenciadas []NFERef `json:"notas_referenciadas,omitempty"`
	// Totais e frete da nota.
	ModalidadeFrete                    int     `json:"modalidade_frete"`
	ValorProdutos, ValorTotal          float64 `json:"-"`
	ValorFrete, ValorSeguro, ValorDesc float64 `json:"-"`
	InformacoesAdicionais              string  `json:"-"`
}

// MarshalJSON escreve a nota no formato da API v2 da Focus: campos planos
// (cnpj_emitente, nome_destinatario, formas_pagamento...), não objetos
// aninhados. O emitente vai só pelo CNPJ — nome, endereço, IE e regime vêm do
// cadastro da empresa no painel da Focus, que é o mesmo da SEFAZ.
func (p NFEPayload) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"natureza_operacao":  p.NaturezaOperacao,
		"data_emissao":       p.DataEmissao,
		"tipo_documento":     p.TipoDocumento,
		"local_destino":      p.LocalDestino,
		"finalidade_emissao": p.FinalidadeEmissao,
		"consumidor_final":   p.ConsumidorFinal,
		"presenca_comprador": p.PresencaComprador,
		"modalidade_frete":   p.ModalidadeFrete,
		"cnpj_emitente":      soDigitos(p.Emitente.CNPJ),
	}
	if p.ModalidadeFrete == 0 && p.ValorFrete == 0 {
		m["modalidade_frete"] = 9 // sem ocorrência de transporte
	}
	d := p.Destinatario
	doc := soDigitos(d.CNPJCPF)
	if len(doc) == 11 {
		m["cpf_destinatario"] = doc
	} else {
		m["cnpj_destinatario"] = doc
	}
	m["nome_destinatario"] = d.Nome
	m["indicador_inscricao_estadual_destinatario"] = d.IndicadorIE
	if d.IndicadorIE == 1 && d.IE != nil {
		m["inscricao_estadual_destinatario"] = soDigitos(*d.IE)
	}
	for k, v := range map[string]string{"logradouro_destinatario": d.Logradouro, "numero_destinatario": d.Numero,
		"bairro_destinatario": d.Bairro, "municipio_destinatario": d.Municipio, "uf_destinatario": d.UF,
		"cep_destinatario": soDigitos(d.CEP), "email_destinatario": d.Email} {
		if strings.TrimSpace(v) != "" {
			m[k] = strings.TrimSpace(v)
		}
	}
	if p.ValorProdutos > 0 {
		m["valor_produtos"] = p.ValorProdutos
	}
	if p.ValorTotal > 0 {
		m["valor_total"] = p.ValorTotal
	}
	if p.ValorFrete > 0 {
		m["valor_frete"] = p.ValorFrete
	}
	if p.ValorSeguro > 0 {
		m["valor_seguro"] = p.ValorSeguro
	}
	if p.ValorDesc > 0 {
		m["valor_desconto"] = p.ValorDesc
	}
	if strings.TrimSpace(p.InformacoesAdicionais) != "" {
		m["informacoes_adicionais_contribuinte"] = p.InformacoesAdicionais
	}
	itens := make([]map[string]any, 0, len(p.Items))
	for _, it := range p.Items {
		itens = append(itens, it.campos())
	}
	m["items"] = itens
	formas := make([]map[string]any, 0, len(p.FormaPagamento))
	for _, f := range p.FormaPagamento {
		formas = append(formas, map[string]any{"forma_pagamento": f.FormaPagamento, "valor_pagamento": f.Valor})
	}
	m["formas_pagamento"] = formas
	if len(p.Duplicatas) > 0 {
		m["duplicatas"] = p.Duplicatas
	}
	// A referência vai num nível só (rejeição 1010): com o DFeReferenciado nos
	// itens (devolução, desde a reforma), o grupo da nota não vai.
	refPorItem := false
	for _, it := range p.Items {
		if it.DFeRefChave != "" {
			refPorItem = true
		}
	}
	if len(p.NotasReferenciadas) > 0 && !refPorItem {
		m["notas_referenciadas"] = p.NotasReferenciadas
	}
	return json.Marshal(m)
}

// cstComICMS: CSTs (e CSOSN) que destacam base, alíquota e valor do ICMS.
var cstComICMS = map[string]bool{"00": true, "10": true, "20": true, "51": true, "70": true, "90": true, "900": true}

func (it NFEItem) campos() map[string]any {
	m := map[string]any{
		"numero_item": it.NumeroItem, "codigo_produto": it.CodigoProduto, "descricao": it.Descricao, "cfop": it.CFOP,
		"codigo_ncm": it.CodigoNCM, "unidade_comercial": it.UnidadeComercial, "quantidade_comercial": it.QuantidadeComercial,
		"valor_unitario_comercial": it.ValorUnitarioComercial, "valor_bruto": it.ValorBruto,
		"unidade_tributavel": it.UnidadeComercial, "quantidade_tributavel": it.QuantidadeComercial,
		"valor_unitario_tributavel": it.ValorUnitarioComercial, "inclui_no_total": 1,
		"icms_origem": it.OrigemMercadoria, "icms_situacao_tributaria": it.CodigoSituacaoTributariaICMS,
		"pis_situacao_tributaria": it.CodigoSituacaoTributariaPIS, "cofins_situacao_tributaria": it.CodigoSituacaoTributariaCOFINS,
	}
	if it.CEST != "" {
		m["cest"] = it.CEST
	}
	for k, v := range map[string]float64{"valor_frete": it.ValorFrete, "valor_seguro": it.ValorSeguro, "valor_desconto": it.ValorDesconto} {
		if v > 0 {
			m[k] = v
		}
	}
	if cstComICMS[it.CodigoSituacaoTributariaICMS] {
		m["icms_modalidade_base_calculo"] = it.ModalidadeBaseCalculoICMS
		m["icms_base_calculo"] = it.ValorBaseCalculoICMS
		m["icms_aliquota"] = it.AliquotaICMS
		m["icms_valor"] = it.ValorICMS
	}
	if it.PercentualDiferimento != nil {
		m["icms_percentual_diferimento"] = *it.PercentualDiferimento
	}
	if it.ValorICMSDiferido != nil {
		m["icms_valor_diferido"] = *it.ValorICMSDiferido
	}
	if it.ValorICMSST != nil {
		if it.ModalidadeBaseCalculoICMSST != nil {
			m["icms_modalidade_base_calculo_st"] = *it.ModalidadeBaseCalculoICMSST
		}
		if it.PercentualMVAICMSST != nil {
			m["icms_margem_valor_adicionado_st"] = *it.PercentualMVAICMSST
		}
		if it.BaseCalculoICMSST != nil {
			m["icms_base_calculo_st"] = *it.BaseCalculoICMSST
		}
		if it.AliquotaICMSST != nil {
			m["icms_aliquota_st"] = *it.AliquotaICMSST
		}
		m["icms_valor_st"] = *it.ValorICMSST
	}
	if it.CodigoSituacaoTributariaIPI != "" {
		m["ipi_situacao_tributaria"] = it.CodigoSituacaoTributariaIPI
		m["ipi_codigo_enquadramento_legal"] = "999"
		if it.ValorIPI > 0 {
			base := it.BaseIPI
			if base == 0 {
				base = it.ValorBruto
			}
			m["ipi_base_calculo"] = base
			m["ipi_aliquota"] = it.AliquotaIPI
			m["ipi_valor"] = it.ValorIPI
		}
	}
	baseContrib := it.ValorBruto - it.ValorDesconto
	if it.ValorPIS > 0 {
		m["pis_base_calculo"] = baseContrib
		m["pis_aliquota_porcentual"] = it.AliquotaPIS
		m["pis_valor"] = it.ValorPIS
	}
	if it.ValorCOFINS > 0 {
		m["cofins_base_calculo"] = baseContrib
		m["cofins_aliquota_porcentual"] = it.AliquotaCOFINS
		m["cofins_valor"] = it.ValorCOFINS
	}
	if it.DFeRefChave != "" && it.DFeRefItem > 0 {
		m["chave_acesso_dfe_referenciado"] = it.DFeRefChave
		m["numero_item_dfe_referenciado"] = fmt.Sprint(it.DFeRefItem)
	}
	if g := it.IBSCBS; g != nil {
		m["ibs_cbs_situacao_tributaria"] = g.CST
		m["ibs_cbs_classificacao_tributaria"] = g.ClassTrib
		m["ibs_cbs_base_calculo"] = g.Base
		m["ibs_uf_aliquota"] = g.AliqIBSUF
		m["ibs_uf_valor"] = g.ValorIBSUF
		m["ibs_mun_aliquota"] = g.AliqIBSMun
		m["ibs_mun_valor"] = g.ValorIBSMun
		m["ibs_valor_total"] = g.ValorIBS
		m["cbs_aliquota"] = g.AliqCBS
		m["cbs_valor"] = g.ValorCBS
	}
	return m
}

func soDigitos(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// NFERef referencia uma NF-e pela chave (grupo NFref do XML).
type NFERef struct {
	ChaveNFe string `json:"chave_nfe"`
}

// NFEResponse é a resposta da Focus (emissão, consulta e cancelamento). A API
// v2 responde "autorizado", "erro_autorizacao", "denegado", "cancelado",
// "erro_cancelamento" e "processando_autorizacao"; a chave vem com o prefixo
// "NFe" (use Chave()).
type NFEResponse struct {
	Status        string `json:"status"`
	Ref           string `json:"ref"`
	ChaveNFe      string `json:"chave_nfe,omitempty"`
	Numero        string `json:"numero,omitempty"`
	Serie         string `json:"serie,omitempty"`
	Protocolo     string `json:"protocolo,omitempty"`
	PathXML       string `json:"caminho_xml_nota_fiscal,omitempty"`
	PathDANFE     string `json:"caminho_danfe,omitempty"`
	PathXMLCancel string `json:"caminho_xml_cancelamento,omitempty"`
	MensagemSEFAZ string `json:"mensagem_sefaz,omitempty"`
	CodigoSEFAZ   string `json:"status_sefaz,omitempty"`
	Mensagem      string `json:"mensagem,omitempty"`
	Erros         []struct {
		Code    string `json:"codigo"`
		Message string `json:"mensagem"`
	} `json:"erros,omitempty"`
}

// Chave devolve a chave de acesso só com os 44 dígitos.
func (r *NFEResponse) Chave() string { return strings.TrimPrefix(strings.TrimSpace(r.ChaveNFe), "NFe") }

// Autorizada diz se a SEFAZ autorizou o uso (a API v2 diz "autorizado").
func (r *NFEResponse) Autorizada() bool { return r.Status == "autorizado" || r.Status == "autorizada" }

func (r *NFEResponse) motivo() string {
	msg := r.MensagemSEFAZ
	if msg == "" {
		msg = r.Mensagem
	}
	if msg == "" && len(r.Erros) > 0 {
		msg = r.Erros[0].Message
	}
	if r.CodigoSEFAZ != "" {
		msg = r.CodigoSEFAZ + " — " + msg
	}
	return msg
}

// EmitirNFe sends POST /nfe?ref={ref} and polls until authorized or error.
func (c *Client) EmitirNFe(ctx context.Context, ref string, payload NFEPayload) (*NFEResponse, error) {
	path := fmt.Sprintf("/nfe?ref=%s", ref)
	body, statusCode, err := c.do(ctx, http.MethodPost, path, payload)
	if err != nil {
		return nil, err
	}

	var resp NFEResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling response (status %d): %w", statusCode, err)
	}

	// Poll until terminal state
	for i := 0; i < 30; i++ {
		if resp.Autorizada() || strings.HasPrefix(resp.Status, "erro") || resp.Status == "denegado" || resp.Status == "denegada" ||
			resp.Status == "cancelado" || resp.Status == "cancelada" {
			break
		}
		time.Sleep(2 * time.Second)
		pollBody, _, pollErr := c.do(ctx, http.MethodGet, fmt.Sprintf("/nfe/%s", ref), nil)
		if pollErr == nil {
			_ = json.Unmarshal(pollBody, &resp)
		}
	}

	if !resp.Autorizada() {
		return &resp, fmt.Errorf("NF-e não autorizada (%s): %s", resp.Status, resp.motivo())
	}
	resp.ChaveNFe = resp.Chave()
	return &resp, nil
}

// DocumentURL builds the absolute URL for a path returned by Focus NF-e
// (path_danfe / path_xml_nota_fiscal). Focus returns these relative to the API
// domain root, so the "/v2" suffix is stripped from the base URL. An already
// absolute URL is returned unchanged.
func (c *Client) DocumentURL(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	origin := strings.TrimSuffix(c.baseURL, "/v2")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return origin + path
}

// ConsultarNFe returns current status of a NF-e.
func (c *Client) ConsultarNFe(ctx context.Context, ref string) (*NFEResponse, error) {
	body, _, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/nfe/%s", ref), nil)
	if err != nil {
		return nil, err
	}
	var resp NFEResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling consult response: %w", err)
	}
	resp.ChaveNFe = resp.Chave()
	return &resp, nil
}

// CancelarNFe sends DELETE /nfe/{ref} with justificativa.
func (c *Client) CancelarNFe(ctx context.Context, ref, justificativa string) (*NFEResponse, error) {
	payload := map[string]string{"justificativa": justificativa}
	body, _, err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/nfe/%s", ref), payload)
	if err != nil {
		return nil, err
	}
	var resp NFEResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling cancel response: %w", err)
	}
	// A Focus responde 200 também quando a SEFAZ recusa o cancelamento
	// ("erro_cancelamento", por exemplo fora do prazo): só "cancelado" vale.
	if resp.Status != "cancelado" && resp.Status != "cancelada" {
		return &resp, fmt.Errorf("cancelamento não aceito pela SEFAZ (%s): %s", resp.Status, resp.motivo())
	}
	return &resp, nil
}

// ─── NF-e de Entrada (Compra) ─────────────────────────────────────────────────

// NFeEntradaItem represents one line in an incoming NF-e (purchase/entry).
type NFeEntradaItem struct {
	NumeroItem          int     `json:"numero_item"`
	CodigoProduto       string  `json:"codigo_produto"`
	Descricao           string  `json:"descricao"`
	CFOP                string  `json:"cfop"`
	UnidadeComercial    string  `json:"unidade_comercial"`
	QuantidadeComercial float64 `json:"quantidade_comercial"`
	ValorUnitario       float64 `json:"valor_unitario_comercial"`
	ValorTotal          float64 `json:"valor_total"`
}

// NFeEntradaResponse is the Focus response for a consulted purchase NF-e.
type NFeEntradaResponse struct {
	Status       string           `json:"status"`
	ChaveNFe     string           `json:"chave_nfe"`
	NumeroNF     string           `json:"numero"`
	Serie        string           `json:"serie"`
	DataEmissao  string           `json:"data_emissao"`
	CnpjEmitente string           `json:"cnpj_emitente"`
	NomeEmitente string           `json:"nome_emitente"`
	ValorTotal   float64          `json:"valor_total"`
	Items        []NFeEntradaItem `json:"items"`
}

// ConsultarNFePorChave fetches an incoming NF-e by its 44-digit access key (chave de acesso).
// Uses Focus NF-e endpoint GET /v2/nfe_entrada/{chave}.
// Returns a parsed structure with line items for stock entry automation.
func (c *Client) ConsultarNFePorChave(ctx context.Context, chaveAcesso string) (*NFeEntradaResponse, error) {
	path := fmt.Sprintf("/nfe_entrada/%s", chaveAcesso)
	body, statusCode, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("consulting NF-e entrada: %w", err)
	}
	if statusCode == 404 {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("NF-e com chave %s não encontrada", chaveAcesso))
	}
	var resp NFeEntradaResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling NF-e entrada response (status %d): %w", statusCode, err)
	}
	return &resp, nil
}

// BaixarXMLNFeRecebida baixa o XML de uma NF-e emitida CONTRA a empresa
// (GET /nfes_recebidas/{chave}.xml). É o mesmo arquivo que o fornecedor
// mandaria: a importação por chave segue o mesmo caminho da importação do
// arquivo. Requer a manifestação do destinatário habilitada na Focus.
func (c *Client) BaixarXMLNFeRecebida(ctx context.Context, chaveAcesso string) ([]byte, error) {
	chave := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, chaveAcesso)
	body, statusCode, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/nfes_recebidas/%s.xml", chave), nil)
	if statusCode == http.StatusNotFound {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf(
			"a NF-e de chave %s não foi encontrada entre as notas recebidas na Focus NF-e: confira a chave ou importe o arquivo XML", chave))
	}
	if err != nil {
		return nil, fmt.Errorf("baixando XML da NF-e recebida: %w", err)
	}
	return body, nil
}

// CadastroResponse holds the relevant fields of Focus's registration query.
type CadastroResponse struct {
	CNPJ              string `json:"cnpj"`
	Nome              string `json:"nome"`
	UF                string `json:"uf"`
	SituacaoCadastral string `json:"situacao_cadastral"`
	Habilitado        bool   `json:"habilitado"`
}

// ConsultarCadastro queries the registration data for a CNPJ/CPF on SEFAZ/Receita
// via Focus (GET /cnpjs/{cnpj}).
func (c *Client) ConsultarCadastro(ctx context.Context, documento string) (*CadastroResponse, error) {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, documento)
	path := fmt.Sprintf("/cnpjs/%s", digits)
	body, statusCode, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("consulting cadastro: %w", err)
	}
	if statusCode == 404 {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("documento %s não encontrado na SEFAZ/Receita", documento))
	}
	var resp CadastroResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling cadastro response (status %d): %w", statusCode, err)
	}
	return &resp, nil
}

// ManifestacaoPayload is the body for a recipient manifestation (manifestação
// do destinatário). Tipo is one of: ciencia, confirmacao, desconhecimento,
// nao_realizada (a justificativa is required for the last two).
type ManifestacaoPayload struct {
	CNPJ          string `json:"cnpj"`
	ChaveNFe      string `json:"chave_nfe"`
	Tipo          string `json:"tipo"`
	Justificativa string `json:"justificativa,omitempty"`
}

// ManifestarDestinatario registra a manifestação do destinatário sobre uma
// NF-e recebida: POST /nfes_recebidas/{chave}/manifesto com o tipo
// (ciencia, confirmacao, desconhecimento, nao_realizada) e, nos dois
// últimos, a justificativa. (Antes chamava /nfe/manifesto, que não existe
// na API da Focus — a manifestação nunca chegava à SEFAZ.)
func (c *Client) ManifestarDestinatario(ctx context.Context, p ManifestacaoPayload) (map[string]interface{}, error) {
	chave := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, p.ChaveNFe)
	corpo := map[string]string{"tipo": p.Tipo}
	if strings.TrimSpace(p.Justificativa) != "" {
		corpo["justificativa"] = p.Justificativa
	}
	body, statusCode, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/nfes_recebidas/%s/manifesto", chave), corpo)
	if err != nil {
		return nil, fmt.Errorf("manifestação destinatário: %w", err)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling manifestação response (status %d): %w", statusCode, err)
	}
	return resp, nil
}

// NFeRecebida é uma NF-e emitida contra o CNPJ da empresa, como a Focus
// devolve na distribuição DF-e.
type NFeRecebida struct {
	ChaveNFe                 string      `json:"chave_nfe"`
	NomeEmitente             string      `json:"nome_emitente"`
	DocumentoEmitente        string      `json:"documento_emitente"`
	ValorTotal               json.Number `json:"valor_total"`
	DataEmissao              string      `json:"data_emissao"`
	Situacao                 string      `json:"situacao"`
	ManifestacaoDestinatario *string     `json:"manifestacao_destinatario"`
	NFeCompleta              bool        `json:"nfe_completa"`
	Versao                   json.Number `json:"versao"`
}

// ListarNFesRecebidas devolve as NF-e recebidas com versão maior que a
// informada (até 100 por chamada) e a maior versão devolvida (cabeçalho
// X-Max-Version), para a próxima busca continuar de onde parou.
func (c *Client) ListarNFesRecebidas(ctx context.Context, cnpj string, versao int64) ([]NFeRecebida, int64, error) {
	if c.configErr != nil {
		return nil, 0, c.configErr
	}
	digitos := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, cnpj)
	path := fmt.Sprintf("/nfes_recebidas?cnpj=%s&versao=%d", digitos, versao)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, 0, err
	}
	req.SetBasicAuth(c.token, "")
	start := time.Now()
	resp, err := c.httpCli.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("consultando NF-e recebidas: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	if c.onRequest != nil {
		c.onRequest(path, http.MethodGet, "", string(body), resp.StatusCode, int(time.Since(start).Milliseconds()))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr struct {
			Mensagem string `json:"mensagem"`
		}
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Mensagem != "" {
			return nil, 0, fmt.Errorf("Focus NF-e HTTP %d: %s", resp.StatusCode, apiErr.Mensagem)
		}
		return nil, 0, fmt.Errorf("Focus NF-e HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var lista []NFeRecebida
	if err := json.Unmarshal(body, &lista); err != nil {
		return nil, 0, fmt.Errorf("lendo NF-e recebidas: %w", err)
	}
	maxVersao := versao
	if v := resp.Header.Get("X-Max-Version"); v != "" {
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil && n > maxVersao {
			maxVersao = n
		}
	}
	for _, n := range lista {
		if v, err := n.Versao.Int64(); err == nil && v > maxVersao {
			maxVersao = v
		}
	}
	return lista, maxVersao, nil
}

// InutilizacaoPayload is the body for an NF-e numbering inutilization.
type InutilizacaoPayload struct {
	CNPJ          string `json:"cnpj"`
	Serie         int    `json:"serie"`
	NumeroInicial int    `json:"numero_inicial"`
	NumeroFinal   int    `json:"numero_final"`
	Justificativa string `json:"justificativa"`
}

// InutilizarNumeracao sends POST /nfe/inutilizacao to invalidate a range of
// unused NF-e numbers at SEFAZ.
func (c *Client) InutilizarNumeracao(ctx context.Context, p InutilizacaoPayload) (map[string]interface{}, error) {
	body, statusCode, err := c.do(ctx, http.MethodPost, "/nfe/inutilizacao", p)
	if err != nil {
		return nil, fmt.Errorf("inutilização de numeração: %w", err)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling inutilização response (status %d): %w", statusCode, err)
	}
	return resp, nil
}

// ─── NFS-e (Nota Fiscal de Serviços eletrônica) ───────────────────────────────

// NFSePrestador identifies the service provider (the company).
type NFSePrestador struct {
	CNPJ               string `json:"cnpj"`
	InscricaoMunicipal string `json:"inscricao_municipal,omitempty"`
	CodigoMunicipio    string `json:"codigo_municipio"`
}

// NFSeTomador identifies the service taker (customer).
type NFSeTomador struct {
	CNPJ            string `json:"cnpj,omitempty"`
	CPF             string `json:"cpf,omitempty"`
	RazaoSocial     string `json:"razao_social,omitempty"`
	Email           string `json:"email,omitempty"`
	Logradouro      string `json:"logradouro,omitempty"`
	Numero          string `json:"numero,omitempty"`
	Complemento     string `json:"complemento,omitempty"`
	Bairro          string `json:"bairro,omitempty"`
	CodigoMunicipio string `json:"codigo_municipio,omitempty"`
	UF              string `json:"uf,omitempty"`
	CEP             string `json:"cep,omitempty"`
}

// NFSePayload is the Focus NFS-e (ABRASF) emission payload.
type NFSePayload struct {
	DataEmissao               string        `json:"data_emissao"`
	NaturezaOperacao          int           `json:"natureza_operacao"`
	OptanteSimplesNacional    bool          `json:"optante_simples_nacional"`
	IncentivadorCultural      bool          `json:"incentivador_cultural"`
	Prestador                 NFSePrestador `json:"prestador"`
	Tomador                   NFSeTomador   `json:"tomador"`
	ItemListaServico          string        `json:"item_lista_servico"`
	CodigoTributarioMunicipio string        `json:"codigo_tributario_municipio,omitempty"`
	Discriminacao             string        `json:"discriminacao"`
	CodigoMunicipio           string        `json:"codigo_municipio"`
	ValorServicos             float64       `json:"valor_servicos"`
	ValorDeducoes             float64       `json:"valor_deducoes,omitempty"`
	AliquotaISS               float64       `json:"aliquota"`
	IssRetido                 bool          `json:"iss_retido"`
	ValorIss                  float64       `json:"valor_iss,omitempty"`
}

// NFSeResponse is the Focus response for an NFS-e emission/consult.
type NFSeResponse struct {
	Status            string `json:"status"`
	Ref               string `json:"ref"`
	NumeroNFSe        string `json:"numero,omitempty"`
	CodigoVerificacao string `json:"codigo_verificacao,omitempty"`
	URL               string `json:"url,omitempty"`
	CaminhoXML        string `json:"caminho_xml_nota_fiscal,omitempty"`
	MensagemSEFAZ     string `json:"mensagem_sefaz,omitempty"`
	Erros             []struct {
		Code    string `json:"codigo"`
		Message string `json:"mensagem"`
	} `json:"erros,omitempty"`
}

// EmitirNFSe sends POST /nfse?ref={ref} and polls until a terminal state.
func (c *Client) EmitirNFSe(ctx context.Context, ref string, payload NFSePayload) (*NFSeResponse, error) {
	body, statusCode, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/nfse?ref=%s", ref), payload)
	if err != nil {
		return nil, err
	}
	var resp NFSeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling NFS-e response (status %d): %w", statusCode, err)
	}

	for i := 0; i < 30; i++ {
		if resp.Status == "autorizado" || resp.Status == "erro_autorizacao" || resp.Status == "cancelado" {
			break
		}
		time.Sleep(2 * time.Second)
		pollBody, _, pollErr := c.do(ctx, http.MethodGet, fmt.Sprintf("/nfse/%s", ref), nil)
		if pollErr == nil {
			_ = json.Unmarshal(pollBody, &resp)
		}
	}

	if resp.Status != "autorizado" {
		msg := resp.MensagemSEFAZ
		if msg == "" && len(resp.Erros) > 0 {
			msg = resp.Erros[0].Message
		}
		return &resp, fmt.Errorf("NFS-e não autorizada: status=%s, msg=%s", resp.Status, msg)
	}
	return &resp, nil
}

// ConsultarNFSe returns the current status of an NFS-e by ref.
func (c *Client) ConsultarNFSe(ctx context.Context, ref string) (*NFSeResponse, error) {
	body, _, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/nfse/%s", ref), nil)
	if err != nil {
		return nil, err
	}
	var resp NFSeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling NFS-e consult response: %w", err)
	}
	return &resp, nil
}

// CancelarNFSe sends DELETE /nfse/{ref} with a justificativa.
func (c *Client) CancelarNFSe(ctx context.Context, ref, justificativa string) (*NFSeResponse, error) {
	payload := map[string]string{"justificativa": justificativa}
	body, _, err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/nfse/%s", ref), payload)
	if err != nil {
		return nil, err
	}
	var resp NFSeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling NFS-e cancel response: %w", err)
	}
	return &resp, nil
}

// ─── CT-e (Conhecimento de Transporte Eletrônico) ─────────────────────────────

// CTePayload is the subset of the Focus CT-e v2 layout needed to authorize a
// standard rodoviário CT-e. The caller supplies the structured emission detail;
// the emitente is filled from the fiscal config.
type CTePayload struct {
	NaturezaOperacao    string   `json:"natureza_operacao"`
	TipoCte             int      `json:"tipo_cte"`        // 0 normal, 1 complemento, 2 anulação, 3 substituto
	TipoServico         int      `json:"tipo_servico"`    // 0 normal
	Modal               string   `json:"modal,omitempty"` // "01" rodoviário
	DataEmissao         string   `json:"data_emissao"`
	MunicipioInicioUF   string   `json:"uf_inicio"`
	MunicipioInicio     string   `json:"municipio_inicio"`
	MunicipioFimUF      string   `json:"uf_fim"`
	MunicipioFim        string   `json:"municipio_fim"`
	TomadorServico      *int     `json:"tomador_servico,omitempty"` // 0 rem,1 exp,2 receb,3 dest,4 outros
	Emitente            CTeParte `json:"emitente"`
	Remetente           CTeParte `json:"remetente"`
	Destinatario        CTeParte `json:"destinatario"`
	ProdutoPredominante string   `json:"produto_predominante,omitempty"`
	ValorCarga          float64  `json:"valor_carga,omitempty"`
	ValorTotalPrestacao float64  `json:"valor_total_prestacao"`
	ValorReceber        float64  `json:"valor_a_receber"`
	ICMS                CTeICMS  `json:"icms"`
	RNTRC               string   `json:"rntrc,omitempty"`
}

// CTeParte is a generic party (emitente/remetente/destinatário) for the CT-e.
type CTeParte struct {
	CNPJ            string `json:"cnpj,omitempty"`
	CPF             string `json:"cpf,omitempty"`
	IE              string `json:"inscricao_estadual,omitempty"`
	Nome            string `json:"nome,omitempty"`
	Fantasia        string `json:"nome_fantasia,omitempty"`
	Logradouro      string `json:"logradouro,omitempty"`
	Numero          string `json:"numero,omitempty"`
	Bairro          string `json:"bairro,omitempty"`
	Municipio       string `json:"municipio,omitempty"`
	CodigoMunicipio string `json:"codigo_municipio,omitempty"`
	UF              string `json:"uf,omitempty"`
	CEP             string `json:"cep,omitempty"`
	Telefone        string `json:"telefone,omitempty"`
}

// CTeICMS carries the CT-e ICMS taxation (situação tributária + base/alíquota).
type CTeICMS struct {
	SituacaoTributaria string  `json:"situacao_tributaria"` // e.g. "00", "90"
	BaseCalculo        float64 `json:"base_calculo,omitempty"`
	Aliquota           float64 `json:"aliquota,omitempty"`
	Valor              float64 `json:"valor,omitempty"`
}

// CTeResponse is the Focus response for a CT-e emission/consult.
type CTeResponse struct {
	Status        string `json:"status"`
	Ref           string `json:"ref"`
	ChaveCTe      string `json:"chave_cte,omitempty"`
	Protocolo     string `json:"protocolo,omitempty"`
	PathXML       string `json:"caminho_xml,omitempty"`
	PathDACTE     string `json:"caminho_dacte,omitempty"`
	MensagemSEFAZ string `json:"mensagem_sefaz,omitempty"`
	Erros         []struct {
		Code    string `json:"codigo"`
		Message string `json:"mensagem"`
	} `json:"erros,omitempty"`
}

// AutorizarCTe sends POST /cte?ref={ref} and polls until a terminal state.
func (c *Client) AutorizarCTe(ctx context.Context, ref string, payload CTePayload) (*CTeResponse, error) {
	body, statusCode, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/cte?ref=%s", ref), payload)
	if err != nil {
		return nil, err
	}
	var resp CTeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling CT-e response (status %d): %w", statusCode, err)
	}

	for i := 0; i < 30; i++ {
		if resp.Status == "autorizado" || resp.Status == "erro_autorizacao" || resp.Status == "cancelado" {
			break
		}
		time.Sleep(2 * time.Second)
		pollBody, _, pollErr := c.do(ctx, http.MethodGet, fmt.Sprintf("/cte/%s", ref), nil)
		if pollErr == nil {
			_ = json.Unmarshal(pollBody, &resp)
		}
	}

	if resp.Status != "autorizado" {
		msg := resp.MensagemSEFAZ
		if msg == "" && len(resp.Erros) > 0 {
			msg = resp.Erros[0].Message
		}
		return &resp, fmt.Errorf("CT-e não autorizado: status=%s, msg=%s", resp.Status, msg)
	}
	return &resp, nil
}

// ConsultarCTe returns the current status of a CT-e by ref.
func (c *Client) ConsultarCTe(ctx context.Context, ref string) (*CTeResponse, error) {
	body, _, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/cte/%s", ref), nil)
	if err != nil {
		return nil, err
	}
	var resp CTeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling CT-e consult response: %w", err)
	}
	return &resp, nil
}

// EmitirCCe sends POST /nfe/{ref}/carta_correcao.
func (c *Client) EmitirCCe(ctx context.Context, ref, textoCorrecao string) (map[string]interface{}, error) {
	payload := map[string]string{"correcao": textoCorrecao}
	body, _, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/nfe/%s/carta_correcao", ref), payload)
	if err != nil {
		return nil, err
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling CCe response: %w", err)
	}
	// A SEFAZ pode recusar a carta com HTTP 200: só "autorizado" vale.
	if st, _ := resp["status"].(string); st != "" && st != "autorizado" {
		msg, _ := resp["mensagem_sefaz"].(string)
		return resp, fmt.Errorf("carta de correção não aceita pela SEFAZ (%s): %s", st, msg)
	}
	return resp, nil
}
