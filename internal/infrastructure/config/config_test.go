package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadRequiresJWTSecretInProduction(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("ENV", "production")
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "at least 32") {
		t.Fatalf("Load() error = %v, want missing JWT_SECRET error", err)
	}
}

func TestLoadAcceptsJWTSecretInProduction(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("ENV", "production")
	t.Setenv("JWT_SECRET", "test-only-production-secret-32-chars")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.JWTSecret != "test-only-production-secret-32-chars" {
		t.Fatalf("JWTSecret = %q, want environment value", cfg.JWTSecret)
	}
}

func TestLoadRejectsInvalidTrustedProxyCIDR(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("ENV", "development")
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8,not-a-network")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "TRUSTED_PROXY_CIDRS") {
		t.Fatalf("Load() error = %v, want trusted proxy validation error", err)
	}
}
