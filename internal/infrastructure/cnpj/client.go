// Package cnpj contains HTTP adapters that resolve company data from public
// CNPJ registries, implementing domain/cnpj/service.Provider.
//
// Two sources are supported:
//   - BrasilAPI (brasilapi.com.br): reliable, returns the full address, CNAE
//     and Simples Nacional flags — but no Inscrição Estadual.
//   - CNPJá Open (open.cnpja.com): returns the Inscrições Estaduais (IE) plus
//     address, at the cost of a tight free-tier rate limit.
//
// The default "auto" provider queries CNPJá first (so the IE is filled in) and
// falls back to BrasilAPI when CNPJá is unavailable or rate-limited, so the user
// always gets at least the address and razão social.
package cnpj

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/cnpj/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cnpj/service"
)

var nonDigit = regexp.MustCompile(`\D`)

func onlyDigits(s string) string { return nonDigit.ReplaceAllString(s, "") }

// Config controls which adapter(s) back the lookup.
type Config struct {
	Provider     string        // "auto" (default), "brasilapi" or "cnpja"
	BrasilAPIURL string        // base URL, no trailing slash
	CNPJaURL     string        // base URL, no trailing slash
	Timeout      time.Duration // per-request timeout
}

// withDefaults fills blank fields with sensible production defaults.
func (c Config) withDefaults() Config {
	if c.Provider == "" {
		c.Provider = "auto"
	}
	if c.BrasilAPIURL == "" {
		c.BrasilAPIURL = "https://brasilapi.com.br/api/cnpj/v1"
	}
	if c.CNPJaURL == "" {
		c.CNPJaURL = "https://open.cnpja.com"
	}
	if c.Timeout <= 0 {
		c.Timeout = 8 * time.Second
	}
	return c
}

// New builds a Provider from configuration.
func New(cfg Config) service.Provider {
	cfg = cfg.withDefaults()
	httpc := &http.Client{Timeout: cfg.Timeout}

	brasil := &brasilAPIProvider{base: cfg.BrasilAPIURL, http: httpc}
	cnpja := &cnpjaProvider{base: cfg.CNPJaURL, http: httpc}

	switch strings.ToLower(cfg.Provider) {
	case "brasilapi":
		return brasil
	case "cnpja":
		return cnpja
	default: // "auto"
		return &chainProvider{primary: cnpja, fallback: brasil}
	}
}

// chainProvider tries the IE-capable source first and degrades to the fallback
// when it errors (rate limit, timeout, outage). A genuine "not found" from the
// primary is propagated as-is — falling back would not help.
type chainProvider struct {
	primary  service.Provider
	fallback service.Provider
}

func (c *chainProvider) Lookup(ctx context.Context, cnpj string) (*entity.Company, error) {
	comp, err := c.primary.Lookup(ctx, cnpj)
	if err == nil {
		return comp, nil
	}
	if err == service.ErrNotFound {
		return nil, err
	}
	return c.fallback.Lookup(ctx, cnpj)
}

// doGET performs a GET and maps transport/status errors onto the domain errors.
func doGET(ctx context.Context, httpc *http.Client, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return service.ErrUnavailable
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "panossoerp/1.0")

	resp, err := httpc.Do(req)
	if err != nil {
		return service.ErrUnavailable
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return service.ErrNotFound
	case resp.StatusCode == http.StatusTooManyRequests,
		resp.StatusCode >= 500:
		return service.ErrUnavailable
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("cnpj provider returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return service.ErrUnavailable
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("cnpj provider: decode response: %w", err)
	}
	return nil
}
