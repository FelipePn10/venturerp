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
	// LogLevel controls verbosity: debug | info | warn | error (default: info)
	LogLevel string `mapstructure:"LOG_LEVEL"`
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
