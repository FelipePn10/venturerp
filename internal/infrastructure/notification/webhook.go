package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	neturl "net/url"
	"strings"
	"time"
)

type WebhookClient struct {
	httpCli *http.Client
}

func NewWebhookClient() *WebhookClient {
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Do not inherit HTTP(S)_PROXY: a proxy would resolve the destination itself
	// and bypass the address checks performed by DialContext.
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("invalid webhook address: %w", err)
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("resolving webhook host: %w", err)
		}
		for _, resolved := range ips {
			if !isPublicWebhookIP(resolved.IP) {
				return nil, fmt.Errorf("webhook destination resolves to a non-public address")
			}
		}
		if len(ips) == 0 {
			return nil, errors.New("webhook host did not resolve to an address")
		}
		for _, resolved := range ips {
			conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(resolved.IP.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			err = dialErr
		}
		return nil, fmt.Errorf("connecting to webhook host: %w", err)
	}
	return &WebhookClient{
		httpCli: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errors.New("too many webhook redirects")
				}
				return validateWebhookURL(req.URL)
			},
		},
	}
}

func isPublicWebhookIP(ip net.IP) bool {
	return ip != nil && ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() &&
		!ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsUnspecified()
}

func validateWebhookURL(target *neturl.URL) error {
	if target == nil || !strings.EqualFold(target.Scheme, "https") || target.Hostname() == "" || target.User != nil {
		return errors.New("webhook URL must be an HTTPS URL without credentials")
	}
	if ip := net.ParseIP(target.Hostname()); ip != nil && !isPublicWebhookIP(ip) {
		return errors.New("webhook destination must use a public address")
	}
	return nil
}

// Send posts payload as JSON to url. Returns nil on 2xx.
func (c *WebhookClient) Send(ctx context.Context, url string, payload any) error {
	target, err := neturl.Parse(url)
	if err != nil {
		return fmt.Errorf("parsing webhook URL: %w", err)
	}
	if err := validateWebhookURL(target); err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpCli.Do(req)
	if err != nil {
		return fmt.Errorf("sending webhook: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
