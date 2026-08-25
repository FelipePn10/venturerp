// Package cnpj resolves company registration data for cadastro auto-fill.
package cnpj

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/cnpj/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cnpj/service"
)

const (
	defaultCnpjwsBaseURL = "https://publica.cnpj.ws"
	defaultCnpjaBaseURL  = "https://open.cnpja.com"
)

var nonDigit = regexp.MustCompile(`\D`)

func onlyDigits(s string) string { return nonDigit.ReplaceAllString(s, "") }

func formatCNAE(code int64) string { return strconv.FormatInt(code, 10) }

// Config permits custom endpoints only for isolated tests. Production uses the
// fixed registry endpoints and does not expose provider selection in .env.
type Config struct {
	// BaseURL overrides the primary provider (CNPJ.ws) endpoint.
	BaseURL string
	// FallbackBaseURL overrides the secondary provider (CNPJá) endpoint.
	FallbackBaseURL string
	Timeout         time.Duration
}

// New builds the "auto" provider: CNPJ.ws first (it carries the Inscrições
// Estaduais), falling back to CNPJá when the primary is unavailable.
func New(cfg Config) service.Provider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultCnpjwsBaseURL
	}
	if cfg.FallbackBaseURL == "" {
		cfg.FallbackBaseURL = defaultCnpjaBaseURL
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 8 * time.Second
	}
	return &fallbackProvider{
		primary: &cnpjwsProvider{
			base: cfg.BaseURL,
			http: &http.Client{Timeout: cfg.Timeout},
		},
		secondary: &cnpjaProvider{
			base: cfg.FallbackBaseURL,
			http: &http.Client{Timeout: cfg.Timeout},
		},
	}
}

// fallbackProvider tries providers in order, only moving on to the next when a
// provider is unavailable (rate-limited, timed out, 5xx). A definitive answer
// (ErrNotFound or a data error) is returned as-is.
type fallbackProvider struct {
	primary   service.Provider
	secondary service.Provider
}

func (f *fallbackProvider) Lookup(ctx context.Context, cnpj string) (*entity.Company, error) {
	company, err := f.primary.Lookup(ctx, cnpj)
	if err == nil {
		return company, nil
	}
	if errors.Is(err, service.ErrUnavailable) {
		return f.secondary.Lookup(ctx, cnpj)
	}
	return nil, err
}

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
	case resp.StatusCode == http.StatusTooManyRequests, resp.StatusCode >= 500:
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
