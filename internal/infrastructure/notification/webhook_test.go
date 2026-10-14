package notification

import (
	"context"
	neturl "net/url"
	"strings"
	"testing"
)

func TestWebhookRejectsUnsafeDestinations(t *testing.T) {
	client := NewWebhookClient()
	tests := []string{
		"http://example.com/hook",
		"https://127.0.0.1/hook",
		"https://[::1]/hook",
		"https://169.254.169.254/latest/meta-data",
		"https://10.0.0.1/hook",
		"https://user:pass@example.com/hook",
		// Faixas "globais" que não são internet pública.
		"https://100.64.0.10/hook",
		"https://198.18.0.1/hook",
		"https://[64:ff9b::a00:1]/hook",
		"https://[::ffff:127.0.0.1]/hook",
		"https://0.0.0.0/hook",
	}
	for _, target := range tests {
		t.Run(target, func(t *testing.T) {
			err := client.Send(context.Background(), target, map[string]string{"test": "value"})
			if err == nil {
				t.Fatal("unsafe webhook destination was accepted")
			}
		})
	}
}

func TestValidateWebhookURLAllowsPublicHTTPS(t *testing.T) {
	target, err := neturl.Parse("https://example.com/hooks/mrp")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateWebhookURL(target); err != nil {
		t.Fatalf("public HTTPS URL rejected: %v", err)
	}
}

func TestWebhookLocalhostResolutionIsBlocked(t *testing.T) {
	err := NewWebhookClient().Send(context.Background(), "https://localhost/hook", struct{}{})
	if err == nil || !strings.Contains(err.Error(), "rede interna") {
		t.Fatalf("localhost error = %v, want non-public rejection", err)
	}
}

// As mensagens chegam ao usuário: português e sem o texto do cliente HTTP.
func TestWebhookMensagensEmPortugues(t *testing.T) {
	for _, alvo := range []string{"ht!tp://x", "http://example.com", "https://10.0.0.1/x", "https://localhost/x"} {
		err := NewWebhookClient().Send(context.Background(), alvo, struct{}{})
		if err == nil {
			t.Fatalf("%s aceito", alvo)
		}
		for _, ingles := range []string{"webhook", "dial", "Post \"", "lookup", "non-public"} {
			if strings.Contains(err.Error(), ingles) {
				t.Fatalf("%s: mensagem técnica: %v", alvo, err)
			}
		}
	}
	if err := ValidateWebhookURL("https://hooks.exemplo.com.br/mrp"); err != nil {
		t.Fatalf("URL pública recusada: %v", err)
	}
}
