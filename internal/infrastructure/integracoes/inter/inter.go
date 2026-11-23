// Package inter é o adaptador da porta ports.CobrancaBancaria para a API
// Cobrança v3 do Banco Inter (boleto com Pix). Ver docs/dev/integracoes-provedores.md.
//
// Campos conferidos contra os tipos gerados da OpenAPI do Inter (SDK inter.js):
// a consulta aninha situação/valor em "cobranca", códigos em "boleto" e "pix".
// Homologar no sandbox antes do primeiro uso em produção.
package inter

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

const (
	baseProducao = "https://cdpj.partners.bancointer.com.br"
	baseSandbox  = "https://cdpj-sandbox.partners.uatinter.co"
	escopos      = "boleto-cobranca.read boleto-cobranca.write"
)

// Config vem do .env da instância (INTER_*).
type Config struct {
	ClientID, ClientSecret string
	CertFile, KeyFile      string
	ContaCorrente          string
	Sandbox                bool
}

// Configurado diz se há o mínimo para falar com o Inter.
func (c Config) Configurado() bool {
	return c.ClientID != "" && c.ClientSecret != "" && c.CertFile != "" && c.KeyFile != ""
}

type Client struct {
	cfg     Config
	base    string
	http    *http.Client
	espera  time.Duration // entre consultas enquanto a emissão processa
	mu      sync.Mutex
	token   string
	validoA time.Time
}

// New carrega o certificado mTLS da integração.
func New(cfg Config) (*Client, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("certificado do Inter: %w", err)
	}
	base := baseProducao
	if cfg.Sandbox {
		base = baseSandbox
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}}
	return newClient(cfg, base, &http.Client{Timeout: 30 * time.Second, Transport: tr}), nil
}

func newClient(cfg Config, base string, hc *http.Client) *Client {
	return &Client{cfg: cfg, base: strings.TrimRight(base, "/"), http: hc, espera: time.Second}
}

func (c *Client) obterToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.validoA) {
		return c.token, nil
	}
	form := url.Values{"client_id": {c.cfg.ClientID}, "client_secret": {c.cfg.ClientSecret}, "grant_type": {"client_credentials"}, "scope": {escopos}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/oauth/v2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("token do Inter: %w", err)
	}
	defer resp.Body.Close()
	corpo, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token do Inter: HTTP %d: %s", resp.StatusCode, resumo(corpo))
	}
	var t struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(corpo, &t); err != nil || t.AccessToken == "" {
		return "", fmt.Errorf("token do Inter: resposta inválida")
	}
	if t.ExpiresIn <= 0 {
		t.ExpiresIn = 3600
	}
	// Renova um minuto antes de vencer (o token vale 1 h; limite de 5 pedidos/min).
	c.token, c.validoA = t.AccessToken, time.Now().Add(time.Duration(t.ExpiresIn-60)*time.Second)
	return c.token, nil
}

func (c *Client) chamar(ctx context.Context, metodo, caminho string, corpo any, saida any) error {
	tok, err := c.obterToken(ctx)
	if err != nil {
		return err
	}
	var leitor io.Reader
	if corpo != nil {
		b, err := json.Marshal(corpo)
		if err != nil {
			return err
		}
		leitor = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, metodo, c.base+caminho, leitor)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.ContaCorrente != "" {
		req.Header.Set("x-conta-corrente", c.cfg.ContaCorrente)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("Inter %s %s: %w", metodo, caminho, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("Inter %s %s: HTTP %d: %s", metodo, caminho, resp.StatusCode, resumo(b))
	}
	if saida != nil && len(b) > 0 {
		return json.Unmarshal(b, saida)
	}
	return nil
}

func resumo(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 300 {
		s = s[:300] + "…"
	}
	return s
}

type pagador struct {
	CpfCnpj     string `json:"cpfCnpj"`
	TipoPessoa  string `json:"tipoPessoa"`
	Nome        string `json:"nome"`
	Endereco    string `json:"endereco"`
	Numero      string `json:"numero,omitempty"`
	Complemento string `json:"complemento,omitempty"`
	Bairro      string `json:"bairro,omitempty"`
	Cidade      string `json:"cidade"`
	UF          string `json:"uf"`
	CEP         string `json:"cep"`
	Email       string `json:"email,omitempty"`
}

type emissao struct {
	SeuNumero      string      `json:"seuNumero"`
	ValorNominal   json.Number `json:"valorNominal"` // número no JSON (decimal sairia como texto)
	DataVencimento string      `json:"dataVencimento"`
	NumDiasAgenda  int         `json:"numDiasAgenda"`
	Pagador        pagador     `json:"pagador"`
}

type detalhe struct {
	Cobranca struct {
		CodigoSolicitacao string `json:"codigoSolicitacao"`
		SeuNumero         string `json:"seuNumero"`
		Situacao          string `json:"situacao"`
		DataSituacao      string `json:"dataSituacao"`
		// Número ou texto: o decimal aceita os dois (um string aqui quebrava com número).
		ValorTotalRecebido *decimal.Decimal `json:"valorTotalRecebido"`
		OrigemRecebimento  string           `json:"origemRecebimento"`
	} `json:"cobranca"`
	Boleto *struct {
		NossoNumero    string `json:"nossoNumero"`
		CodigoBarras   string `json:"codigoBarras"`
		LinhaDigitavel string `json:"linhaDigitavel"`
	} `json:"boleto"`
	Pix *struct {
		Txid          string `json:"txid"`
		PixCopiaECola string `json:"pixCopiaECola"`
	} `json:"pix"`
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

func corta(s string, n int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) > n {
		return string([]rune(s)[:n])
	}
	return s
}

// GerarPix emite o boleto com Pix e espera a emissão (assíncrona) por alguns
// segundos; se o Inter ainda processa, devolve Pendente — os códigos chegam pela
// consulta seguinte.
func (c *Client) GerarPix(ctx context.Context, t ports.CobrancaPix) (*ports.PixGerado, error) {
	doc := soDigitos(t.Pagador.Documento)
	tipo := "JURIDICA"
	if len(doc) == 11 {
		tipo = "FISICA"
	}
	if doc == "" || t.Pagador.Nome == "" || t.Pagador.Logradouro == "" || t.Pagador.Cidade == "" || t.Pagador.UF == "" || len(soDigitos(t.Pagador.CEP)) != 8 {
		return nil, fmt.Errorf("o boleto exige do cliente: documento, nome, logradouro, cidade, UF e CEP")
	}
	if t.Valor.LessThan(decimal.RequireFromString("2.50")) {
		return nil, fmt.Errorf("o Inter não emite cobrança abaixo de R$ 2,50")
	}
	venc := t.Vencimento
	if hoje := time.Now(); venc.Before(time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, venc.Location())) {
		venc = hoje // título vencido: nova cobrança vence hoje
	}
	var r struct {
		CodigoSolicitacao string `json:"codigoSolicitacao"`
	}
	if err := c.chamar(ctx, http.MethodPost, "/cobranca/v3/cobrancas", emissao{
		SeuNumero: corta(t.SeuNumero, 15), ValorNominal: json.Number(t.Valor.StringFixed(2)), DataVencimento: venc.Format("2006-01-02"),
		Pagador: pagador{CpfCnpj: doc, TipoPessoa: tipo, Nome: corta(t.Pagador.Nome, 100), Endereco: corta(t.Pagador.Logradouro, 100),
			Numero: corta(t.Pagador.Numero, 10), Complemento: corta(t.Pagador.Complemento, 30), Bairro: corta(t.Pagador.Bairro, 60),
			Cidade: corta(t.Pagador.Cidade, 60), UF: strings.ToUpper(corta(t.Pagador.UF, 2)), CEP: soDigitos(t.Pagador.CEP), Email: corta(t.Pagador.Email, 50)},
	}, &r); err != nil {
		return nil, err
	}
	if r.CodigoSolicitacao == "" {
		return nil, errors.New("o Inter não devolveu o código da solicitação")
	}
	out := &ports.PixGerado{IDExterno: r.CodigoSolicitacao, Pendente: true}
	for tentativa := 0; tentativa < 5; tentativa++ {
		d, err := c.consultar(ctx, r.CodigoSolicitacao)
		if err != nil {
			return out, nil // emitida; os códigos ficam para a consulta seguinte
		}
		if d.Cobranca.Situacao == "FALHA_EMISSAO" {
			return nil, fmt.Errorf("o Inter recusou a emissão da cobrança %s", r.CodigoSolicitacao)
		}
		if codigosDa(d, out); !out.Pendente {
			return out, nil
		}
		select {
		case <-ctx.Done():
			return out, nil
		case <-time.After(c.espera):
		}
	}
	return out, nil
}

func (c *Client) consultar(ctx context.Context, codigo string) (*detalhe, error) {
	var d detalhe
	if err := c.chamar(ctx, http.MethodGet, "/cobranca/v3/cobrancas/"+url.PathEscape(codigo), nil, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// codigosDa preenche os códigos de pagamento a partir da consulta.
func codigosDa(d *detalhe, out *ports.PixGerado) {
	out.Situacao = d.Cobranca.Situacao
	if d.Boleto != nil && d.Boleto.LinhaDigitavel != "" {
		out.Pendente = false
		out.NossoNumero, out.LinhaDigitavel, out.CodigoBarras = d.Boleto.NossoNumero, d.Boleto.LinhaDigitavel, d.Boleto.CodigoBarras
		if d.Pix != nil {
			out.TxID, out.CopiaECola = d.Pix.Txid, d.Pix.PixCopiaECola
		}
	}
}

// ConsultarCobranca relê uma cobrança já emitida (emitir de novo não cria outra).
func (c *Client) ConsultarCobranca(ctx context.Context, codigo string) (*ports.PixGerado, error) {
	d, err := c.consultar(ctx, codigo)
	if err != nil {
		return nil, err
	}
	out := &ports.PixGerado{IDExterno: codigo, Pendente: true}
	codigosDa(d, out)
	return out, nil
}

// CancelarCobranca baixa a cobrança no Inter (título cancelado ou recebido por
// outro meio): o cliente não consegue mais pagar o boleto/PIX dela.
func (c *Client) CancelarCobranca(ctx context.Context, codigo, motivo string) error {
	if motivo = corta(motivo, 50); motivo == "" {
		motivo = "Cancelado no ERP"
	}
	return c.chamar(ctx, http.MethodPost, "/cobranca/v3/cobrancas/"+url.PathEscape(codigo)+"/cancelar",
		map[string]string{"motivoCancelamento": motivo}, nil)
}

// RegistrarWebhook cadastra no Inter a URL que recebe os avisos de pagamento.
func (c *Client) RegistrarWebhook(ctx context.Context, endereco string) error {
	if !strings.HasPrefix(endereco, "https://") {
		return fmt.Errorf("o Inter só aceita webhook em HTTPS")
	}
	return c.chamar(ctx, http.MethodPut, "/cobranca/v3/cobrancas/webhook", map[string]string{"webhookUrl": endereco}, nil)
}

// ConsultarPagamento lê a cobrança no Inter (o aviso do webhook só dispara isto).
func (c *Client) ConsultarPagamento(ctx context.Context, codigo string) (*ports.PagamentoConsultado, error) {
	d, err := c.consultar(ctx, codigo)
	if err != nil {
		return nil, err
	}
	p := &ports.PagamentoConsultado{IDExterno: d.Cobranca.CodigoSolicitacao, SeuNumero: d.Cobranca.SeuNumero, Situacao: d.Cobranca.Situacao,
		Forma: d.Cobranca.OrigemRecebimento, Pago: d.Cobranca.Situacao == "RECEBIDO" || d.Cobranca.Situacao == "MARCADO_RECEBIDO"}
	if d.Cobranca.ValorTotalRecebido != nil {
		p.Valor = *d.Cobranca.ValorTotalRecebido
	}
	if d.Cobranca.DataSituacao != "" {
		if t, err := time.Parse("2006-01-02", d.Cobranca.DataSituacao[:min(10, len(d.Cobranca.DataSituacao))]); err == nil {
			p.Data = t
		}
	}
	if p.Data.IsZero() {
		p.Data = time.Now()
	}
	return p, nil
}

var _ ports.CobrancaBancaria = (*Client)(nil)
