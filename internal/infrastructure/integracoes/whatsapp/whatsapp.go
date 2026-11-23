// Package whatsapp é o adaptador de ports.Mensageiro para a WhatsApp Cloud API
// da Meta (direto, sem intermediário — ver docs/dev/integracoes-provedores.md).
// Mensagem iniciada pela empresa só sai por modelo aprovado: o modelo tem UMA
// variável no corpo, que recebe o texto do aviso.
package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

const versaoGraph = "v21.0"

type Config struct {
	Token, PhoneNumberID, Template, Idioma string
}

func (c Config) Configurado() bool { return c.Token != "" && c.PhoneNumberID != "" && c.Template != "" }

type Client struct {
	cfg  Config
	base string
	http *http.Client
}

func New(cfg Config) *Client {
	if cfg.Idioma == "" {
		cfg.Idioma = "pt_BR"
	}
	return &Client{cfg: cfg, base: "https://graph.facebook.com/" + versaoGraph, http: &http.Client{Timeout: 20 * time.Second}}
}

var espacos = regexp.MustCompile(`\s{2,}`)

// ParametroDoModelo: a variável do modelo não aceita quebra de linha, tabulação
// nem mais de quatro espaços seguidos; vai no máximo 1024 caracteres.
func ParametroDoModelo(texto string) string {
	linhas := strings.FieldsFunc(texto, func(r rune) bool { return r == '\n' || r == '\r' })
	for i := range linhas {
		linhas[i] = strings.TrimSpace(linhas[i])
	}
	s := espacos.ReplaceAllString(strings.ReplaceAll(strings.Join(linhas, " · "), "\t", " "), " ")
	if r := []rune(s); len(r) > 1024 {
		s = string(r[:1024])
	}
	return s
}

// NumeroInternacional: a Meta exige o número com o DDI; o cadastro brasileiro
// guarda DDD + número (10 ou 11 dígitos), que ganha o 55 na frente.
func NumeroInternacional(fone string) string {
	var b strings.Builder
	for _, r := range fone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := strings.TrimLeft(b.String(), "0")
	if len(d) == 10 || len(d) == 11 {
		return "55" + d
	}
	return d
}

func (c *Client) Enviar(ctx context.Context, m ports.MensagemCurta) error {
	corpo := map[string]any{
		"messaging_product": "whatsapp", "to": NumeroInternacional(m.Para), "type": "template",
		"template": map[string]any{
			"name": c.cfg.Template, "language": map[string]string{"code": c.cfg.Idioma},
			"components": []any{map[string]any{"type": "body", "parameters": []any{map[string]string{"type": "text", "text": ParametroDoModelo(m.Texto)}}}},
		},
	}
	b, _ := json.Marshal(corpo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/"+c.cfg.PhoneNumberID+"/messages", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("WhatsApp: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		r, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("WhatsApp: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(r)))
	}
	return nil
}

var _ ports.Mensageiro = (*Client)(nil)
