// Package freterapido é o adaptador das portas de cotação e rastreio de frete
// para a Frete Rápido (hub B2B de carga fracionada — docs/dev/integracoes-provedores.md).
// Campos conferidos em dev.freterapido.com.br (cotação v3, ocorrências por NF).
package freterapido

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

type Config struct {
	Token, PlatformCode string
	// Categoria do volume (tabela "Tipos de Volumes" da Frete Rápido); padrão 999.
	Categoria string
}

func (c Config) Configurado() bool { return c.Token != "" }

type Client struct {
	cfg                   Config
	baseCotacao, baseRast string
	http                  *http.Client
}

func New(cfg Config) *Client {
	if cfg.Categoria == "" {
		cfg.Categoria = "999"
	}
	return &Client{cfg: cfg, baseCotacao: "https://sp.freterapido.com", baseRast: "https://freterapido.com", http: &http.Client{Timeout: 30 * time.Second}}
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

func cep(s string) (int, error) {
	n, err := strconv.Atoi(soDigitos(s))
	if err != nil || len(soDigitos(s)) != 8 {
		return 0, fmt.Errorf("CEP inválido: %q", s)
	}
	return n, nil
}

func num(d decimal.Decimal) json.Number { return json.Number(d.String()) }

// Cotar: POST /api/v3/quote/simulate (fracionada), token no corpo.
func (c *Client) Cotar(ctx context.Context, p ports.PedidoDeCotacao) ([]ports.OfertaDeFrete, error) {
	origem, err := cep(p.CEPOrigem)
	if err != nil {
		return nil, fmt.Errorf("origem: %w", err)
	}
	destino, err := cep(p.CEPDestino)
	if err != nil {
		return nil, fmt.Errorf("destino: %w", err)
	}
	if len(p.Volumes) == 0 {
		return nil, fmt.Errorf("informe os volumes da carga (medidas e peso)")
	}
	tipo := 0
	if p.DestinatarioPJ {
		tipo = 1
	}
	vols := make([]map[string]any, 0, len(p.Volumes))
	for _, v := range p.Volumes {
		vols = append(vols, map[string]any{"amount": v.Quantidade, "category": c.cfg.Categoria, "height": num(v.Altura), "width": num(v.Largura),
			"length": num(v.Comp), "unitary_price": num(v.ValorUnitario), "unitary_weight": num(v.PesoKg)})
	}
	corpo := map[string]any{
		"shipper":         map[string]string{"registered_number": soDigitos(p.CNPJEmbarcador), "token": c.cfg.Token, "platform_code": c.cfg.PlatformCode},
		"recipient":       map[string]any{"type": tipo, "registered_number": soDigitos(p.DocDestinatario), "country": "BRA", "zipcode": destino},
		"dispatchers":     []any{map[string]any{"registered_number": soDigitos(p.CNPJEmbarcador), "zipcode": origem, "total_price": num(p.ValorNota), "volumes": vols}},
		"simulation_type": []int{0},
	}
	b, _ := json.Marshal(corpo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseCotacao+"/api/v3/quote/simulate", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	var r struct {
		Dispatchers []struct {
			Offers []struct {
				Carrier struct {
					Name             string `json:"name"`
					RegisteredNumber string `json:"registered_number"`
				} `json:"carrier"`
				Service      string `json:"service"`
				DeliveryTime struct {
					Days          int    `json:"days"`
					EstimatedDate string `json:"estimated_date"`
				} `json:"delivery_time"`
				Expiration *time.Time      `json:"expiration"`
				FinalPrice decimal.Decimal `json:"final_price"`
			} `json:"offers"`
		} `json:"dispatchers"`
	}
	if err := c.fazer(req, &r); err != nil {
		return nil, err
	}
	var out []ports.OfertaDeFrete
	for _, d := range r.Dispatchers {
		for _, o := range d.Offers {
			out = append(out, ports.OfertaDeFrete{Transportadora: o.Carrier.Name, CNPJ: o.Carrier.RegisteredNumber, Servico: o.Service,
				PrazoDias: o.DeliveryTime.Days, PrevisaoEntrega: o.DeliveryTime.EstimatedDate, Preco: o.FinalPrice, Validade: o.Expiration})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Preco.LessThan(out[j].Preco) })
	return out, nil
}

// Rastrear: ocorrências pela chave da NF-e (documento), token na query.
func (c *Client) Rastrear(ctx context.Context, _ string, chave string) ([]ports.EventoDeRastreio, error) {
	chave = soDigitos(chave)
	if len(chave) != 44 {
		return nil, fmt.Errorf("o rastreio usa a chave da NF-e (44 dígitos)")
	}
	nf, _ := json.Marshal([]map[string]string{{"key": chave}})
	q := url.Values{"token": {c.cfg.Token}, "invoices": {string(nf)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseRast+"/api/external/embarcador/v1/quotes/occurrences/invoices?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	// Resposta: lista de objetos indexados pelo id do frete.
	var r []map[string]struct {
		Occurrences []struct {
			Name           string `json:"name"`
			Message        string `json:"message"`
			OccurrenceDate string `json:"occurrence_date"`
		} `json:"occurrences"`
	}
	if err := c.fazer(req, &r); err != nil {
		return nil, err
	}
	var out []ports.EventoDeRastreio
	for _, porFrete := range r {
		for _, f := range porFrete {
			for _, o := range f.Occurrences {
				t, _ := time.ParseInLocation("2006-01-02 15:04:05", o.OccurrenceDate, time.Local)
				sit := o.Name
				if o.Message != "" && o.Message != o.Name {
					sit += " — " + o.Message
				}
				out = append(out, ports.EventoDeRastreio{Quando: t, Situacao: sit})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Quando.Before(out[j].Quando) })
	return out, nil
}

func (c *Client) fazer(req *http.Request, saida any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("Frete Rápido: %w", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("Frete Rápido: token inválido")
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		s := strings.TrimSpace(string(b))
		if len(s) > 300 {
			s = s[:300]
		}
		return fmt.Errorf("Frete Rápido: HTTP %d: %s", resp.StatusCode, s)
	}
	return json.Unmarshal(b, saida)
}

var (
	_ ports.CotacaoDeFrete  = (*Client)(nil)
	_ ports.RastreioDeCarga = (*Client)(nil)
)
