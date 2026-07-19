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
	if err == nil || !strings.Contains(err.Error(), "non-public") {
		t.Fatalf("localhost error = %v, want non-public rejection", err)
	}
}
