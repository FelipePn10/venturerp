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
			return nil, errors.New("endereço de notificação inválido")
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("não foi possível localizar o endereço de notificação %s", host)
		}
		for _, resolved := range ips {
			if !isPublicWebhookIP(resolved.IP) {
				return nil, errors.New("o endereço de notificação aponta para uma rede interna e não é permitido")
			}
		}
		if len(ips) == 0 {
			return nil, errors.New("não foi possível resolver o endereço de notificação")
		}
		for _, resolved := range ips {
			conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(resolved.IP.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			err = dialErr
		}
		return nil, fmt.Errorf("não foi possível conectar ao endereço de notificação: %w", err)
	}
	return &WebhookClient{
		httpCli: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errors.New("o endereço de notificação redirecionou vezes demais")
				}
				return validateWebhookURL(req.URL)
			},
		},
	}
}

// faixasNaoPublicas são endereços "globais" para a biblioteca padrão que não
// são internet pública: CGNAT (redes internas de provedores e VPNs como
// Tailscale), redes reservadas/de teste e NAT64, que embute um IPv4 interno.
var faixasNaoPublicas = func() []*net.IPNet {
	out := []*net.IPNet{}
	for _, c := range []string{
		"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15",
		"198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "255.255.255.255/32",
		"64:ff9b::/96", "64:ff9b:1::/48", "2001:db8::/32", "100::/64",
	} {
		_, n, err := net.ParseCIDR(c)
		if err == nil {
			out = append(out, n)
		}
	}
	return out
}()

func isPublicWebhookIP(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return false
	}
	for _, n := range faixasNaoPublicas {
		if n.Contains(ip) {
			return false
		}
	}
	return true
}

func validateWebhookURL(target *neturl.URL) error {
	if target == nil || !strings.EqualFold(target.Scheme, "https") || target.Hostname() == "" || target.User != nil {
		return errors.New("o endereço de notificação deve ser HTTPS e sem usuário e senha na URL")
	}
	if ip := net.ParseIP(target.Hostname()); ip != nil && !isPublicWebhookIP(ip) {
		return errors.New("o endereço de notificação precisa ser público")
	}
	return nil
}

// ValidateURL confere o endereço antes de enviar: HTTPS, sem credenciais na
// URL e, se for IP, público. O nome é conferido de novo na conexão (DNS).
func ValidateWebhookURL(raw string) error {
	target, err := neturl.Parse(strings.TrimSpace(raw))
	if err != nil {
		return errors.New("o endereço de notificação não é uma URL válida")
	}
	return validateWebhookURL(target)
}

// Send posts payload as JSON to url. Returns nil on 2xx.
func (c *WebhookClient) Send(ctx context.Context, url string, payload any) error {
	target, err := neturl.Parse(strings.TrimSpace(url))
	if err != nil {
		return errors.New("o endereço de notificação não é uma URL válida")
	}
	if err := validateWebhookURL(target); err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("montando a notificação: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(body))
	if err != nil {
		return errors.New("o endereço de notificação não é uma URL válida")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpCli.Do(req)
	if err != nil {
		// O erro do cliente HTTP repete a URL e detalhes de rede em inglês;
		// a causa que importa ao usuário vem das checagens acima, em português.
		var urlErr *neturl.Error
		if errors.As(err, &urlErr) && urlErr.Err != nil {
			return fmt.Errorf("não foi possível enviar a notificação: %s", urlErr.Err.Error())
		}
		return errors.New("não foi possível enviar a notificação")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("o endereço de notificação respondeu com erro (HTTP %d)", resp.StatusCode)
	}
	return nil
}
