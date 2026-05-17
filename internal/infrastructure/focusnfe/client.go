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
	CNPJCPF      string  `json:"cnpj_cpf"`
	Nome         string  `json:"nome"`
	Logradouro   string  `json:"logradouro,omitempty"`
	Numero       string  `json:"numero,omitempty"`
	Bairro       string  `json:"bairro,omitempty"`
	Municipio    string  `json:"municipio,omitempty"`
	UF           string  `json:"uf,omitempty"`
	CEP          string  `json:"cep,omitempty"`
	Email        string  `json:"email,omitempty"`
	IndicadorIE  int     `json:"indicador_ie"`
	IE           *string `json:"ie,omitempty"`
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
	Status          string `json:"status"`
	Ref             string `json:"ref"`
	ChaveNFe        string `json:"chave_nfe,omitempty"`
	Protocolo       string `json:"protocolo,omitempty"`
	PathXML         string `json:"path_xml_nota_fiscal,omitempty"`
	PathDANFE       string `json:"path_danfe,omitempty"`
	MensagemSEFAZ   string `json:"mensagem_sefaz,omitempty"`
	CodigoSEFAZ     string `json:"codigo_sefaz,omitempty"`
	Erros           []struct {
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
