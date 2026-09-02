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

func TestLoadTrainingRequiresSeparateIdentityDatabase(t *testing.T) {
	for name, identityURL := range map[string]string{
		"missing":       "",
		"same database": "postgres://localhost/training",
		"same database with other credentials and options": "postgresql://other:secret@localhost:5432/training?application_name=identity",
	} {
		t.Run(name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			t.Setenv("ENV", "development")
			t.Setenv("DATA_ENVIRONMENT", "training")
			t.Setenv("DATABASE_URL", "postgres://localhost/training")
			t.Setenv("IDENTITY_DATABASE_URL", identityURL)
			if _, err := Load(); err == nil {
				t.Fatal("Load() expected unsafe training configuration to fail")
			}
		})
	}
}

func TestLoadAcceptsSeparatedTrainingDatabases(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("ENV", "development")
	t.Setenv("DATA_ENVIRONMENT", "training")
	t.Setenv("DATABASE_URL", "postgres://localhost/training")
	t.Setenv("IDENTITY_DATABASE_URL", "postgres://localhost/production")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.IsTraining() {
		t.Fatal("expected training configuration")
	}
}
