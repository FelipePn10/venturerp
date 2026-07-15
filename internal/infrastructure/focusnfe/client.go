package focusnfe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
}

func NewClient(token, ambiente string) *Client {
	base := baseURLHomologacao
	if strings.ToLower(ambiente) == "producao" {
		base = baseURLProducao
	}
	return &Client{
		token:   token,
		baseURL: base,
		httpCli: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) WithLogger(fn func(endpoint, method, reqBody, respBody string, statusCode, durationMs int)) *Client {
	c.onRequest = fn
	return c
}

func (c *Client) do(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
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
}

type NFEFormaPagamento struct {
	FormaPagamento string  `json:"forma_pagamento"`
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
}

type NFEResponse struct {
	Status        string `json:"status"`
	Ref           string `json:"ref"`
	ChaveNFe      string `json:"chave_nfe,omitempty"`
	Protocolo     string `json:"protocolo,omitempty"`
	PathXML       string `json:"path_xml_nota_fiscal,omitempty"`
	PathDANFE     string `json:"path_danfe,omitempty"`
	MensagemSEFAZ string `json:"mensagem_sefaz,omitempty"`
	CodigoSEFAZ   string `json:"codigo_sefaz,omitempty"`
	Erros         []struct {
		Code    string `json:"codigo"`
		Message string `json:"mensagem"`
	} `json:"erros,omitempty"`
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
		if resp.Status == "autorizada" || resp.Status == "denegada" || resp.Status == "erro_autorizacao" || resp.Status == "cancelada" {
			break
		}
		time.Sleep(2 * time.Second)
		pollBody, _, pollErr := c.do(ctx, http.MethodGet, fmt.Sprintf("/nfe/%s", ref), nil)
		if pollErr == nil {
			_ = json.Unmarshal(pollBody, &resp)
		}
	}

	if resp.Status != "autorizada" {
		msg := resp.MensagemSEFAZ
		if msg == "" && len(resp.Erros) > 0 {
			msg = resp.Erros[0].Message
		}
		return &resp, fmt.Errorf("NF-e não autorizada: status=%s, msg=%s", resp.Status, msg)
	}

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
		return nil, fmt.Errorf("NF-e com chave %s não encontrada", chaveAcesso)
	}
	var resp NFeEntradaResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling NF-e entrada response (status %d): %w", statusCode, err)
	}
	return &resp, nil
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
		return nil, fmt.Errorf("documento %s não encontrado na SEFAZ/Receita", documento)
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

// ManifestarDestinatario sends POST /nfe/manifesto to register the recipient's
// manifestation about an incoming NF-e.
func (c *Client) ManifestarDestinatario(ctx context.Context, p ManifestacaoPayload) (map[string]interface{}, error) {
	body, statusCode, err := c.do(ctx, http.MethodPost, "/nfe/manifesto", p)
	if err != nil {
		return nil, fmt.Errorf("manifestação destinatário: %w", err)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling manifestação response (status %d): %w", statusCode, err)
	}
	return resp, nil
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
	payload := map[string]string{"descricao_correcao": textoCorrecao}
	body, _, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/nfe/%s/carta_correcao", ref), payload)
	if err != nil {
		return nil, err
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling CCe response: %w", err)
	}
	return resp, nil
}
