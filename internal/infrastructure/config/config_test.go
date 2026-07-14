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
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET is required") {
		t.Fatalf("Load() error = %v, want missing JWT_SECRET error", err)
	}
}

func TestLoadAcceptsJWTSecretInProduction(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("ENV", "production")
	t.Setenv("JWT_SECRET", "test-only-production-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.JWTSecret != "test-only-production-secret" {
		t.Fatalf("JWTSecret = %q, want environment value", cfg.JWTSecret)
	}
}
