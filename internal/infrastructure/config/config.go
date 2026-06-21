package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort  string `mapstructure:"SERVER_ADDR"`
	DatabaseURL string `mapstructure:"DATABASE_URL"`
	JWTSecret   string `mapstructure:"JWT_SECRET"`
	Env         string `mapstructure:"ENV"`
	LogLevel    string `mapstructure:"LOG_LEVEL"`

	// SMTP — e-mail alerts (optional; leave blank to disable)
	SMTPHost     string `mapstructure:"SMTP_HOST"`
	SMTPPort     string `mapstructure:"SMTP_PORT"`
	SMTPUser     string `mapstructure:"SMTP_USER"`
	SMTPPassword string `mapstructure:"SMTP_PASSWORD"`
	SMTPFrom     string `mapstructure:"SMTP_FROM"`

	// HTTP hardening (server-side cross-cutting concerns)
	CORSAllowedOrigins string `mapstructure:"CORS_ALLOWED_ORIGINS"`  // comma-separated; empty in development reflects any origin
	RateLimitRPS       int    `mapstructure:"RATE_LIMIT_RPS"`        // sustained requests/sec per IP (<=0 disables)
	RateLimitBurst     int    `mapstructure:"RATE_LIMIT_BURST"`      // burst capacity per IP
	AuthRateLimitRPM   int    `mapstructure:"AUTH_RATE_LIMIT_RPM"`   // login/register attempts per minute per IP
	AuthRateLimitBurst int    `mapstructure:"AUTH_RATE_LIMIT_BURST"` // burst capacity for auth endpoints
	MaxBodyBytes       int64  `mapstructure:"MAX_BODY_BYTES"`        // request body cap in bytes (0 disables)
	MetricsEnabled     bool   `mapstructure:"METRICS_ENABLED"`       // expose /metrics
	MetricsToken       string `mapstructure:"METRICS_TOKEN"`         // optional bearer token guarding /metrics
	ShutdownTimeoutSec int    `mapstructure:"SHUTDOWN_TIMEOUT_SEC"`  // graceful drain budget in seconds

	// CNPJ auto-lookup — pre-fills cadastro forms (razão social, IE, endereço)
	// from public registries. Disabled gracefully when the provider is offline.
	CNPJProvider     string `mapstructure:"CNPJ_PROVIDER"`      // "auto" (default), "brasilapi" or "cnpja"
	CNPJBrasilAPIURL string `mapstructure:"CNPJ_BRASILAPI_URL"` // base URL (no trailing slash)
	CNPJaURL         string `mapstructure:"CNPJ_CNPJA_URL"`     // base URL (no trailing slash)
	CNPJTimeoutSec   int    `mapstructure:"CNPJ_TIMEOUT_SEC"`   // per-request timeout in seconds
}

// IsDevelopment reports whether the process is NOT running in production. Used
// to relax defaults (e.g. permissive CORS) outside production.
func (c *Config) IsDevelopment() bool {
	return c.Env != "production" && c.Env != "prod"
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	// Defaults (fallback)
	viper.SetDefault("SERVER_ADDR", "5070")
	viper.SetDefault(
		"DATABASE_URL",
		"postgres://panossoerp:panossoerp_10203040@localhost:5432/panossoerpdatabase?sslmode=disable",
	)
	viper.SetDefault("ENV", "development")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("SMTP_HOST", "")
	viper.SetDefault("SMTP_PORT", "587")
	viper.SetDefault("SMTP_USER", "")
	viper.SetDefault("SMTP_PASSWORD", "")
	viper.SetDefault("SMTP_FROM", "")
	viper.SetDefault("CORS_ALLOWED_ORIGINS", "")
	viper.SetDefault("RATE_LIMIT_RPS", 50)
	viper.SetDefault("RATE_LIMIT_BURST", 100)
	viper.SetDefault("AUTH_RATE_LIMIT_RPM", 20)
	viper.SetDefault("AUTH_RATE_LIMIT_BURST", 10)
	viper.SetDefault("MAX_BODY_BYTES", 10485760) // 10 MiB (large enough for NF-e XML import)
	viper.SetDefault("METRICS_ENABLED", true)
	viper.SetDefault("METRICS_TOKEN", "")
	viper.SetDefault("SHUTDOWN_TIMEOUT_SEC", 15)
	viper.SetDefault("CNPJ_PROVIDER", "auto")
	viper.SetDefault("CNPJ_BRASILAPI_URL", "https://brasilapi.com.br/api/cnpj/v1")
	viper.SetDefault("CNPJ_CNPJA_URL", "https://open.cnpja.com")
	viper.SetDefault("CNPJ_TIMEOUT_SEC", 8)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error: %w", err)
		}
	}
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("erro parse config: %w", err)
	}
	return &cfg, nil
}
