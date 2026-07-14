// Package cnpj resolves company registration data for cadastro auto-fill.
package cnpj

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/cnpj/service"
)

const defaultBaseURL = "https://open.cnpja.com"

var nonDigit = regexp.MustCompile(`\D`)

func onlyDigits(s string) string { return nonDigit.ReplaceAllString(s, "") }

func formatCNAE(code int64) string { return strconv.FormatInt(code, 10) }

// Config permits a custom endpoint only for isolated tests. Production uses
// the fixed registry endpoint and does not expose provider selection in .env.
type Config struct {
	BaseURL string
	Timeout time.Duration
}

func New(cfg Config) service.Provider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 8 * time.Second
	}
	return &cnpjaProvider{base: cfg.BaseURL, http: &http.Client{Timeout: cfg.Timeout}}
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
