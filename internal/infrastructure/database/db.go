package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/config"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

type DB struct {
	Pool *pgxpool.Pool
}

// PoolLimits descreve o dimensionamento do pool. Zero em qualquer campo mantém o
// padrão do pgxpool para aquele campo.
type PoolLimits struct {
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

func NewDB(cfg *config.Config) (*DB, error) {
	return NewDBURLWithLimits(cfg.DatabaseURL, PoolLimits{
		MaxConns:        int32(cfg.DBMaxConns),
		MinConns:        int32(cfg.DBMinConns),
		MaxConnLifetime: time.Duration(cfg.DBMaxConnLifetime) * time.Minute,
		MaxConnIdleTime: time.Duration(cfg.DBMaxConnIdleTime) * time.Minute,
	})
}

// NewDBURL abre o pool com o dimensionamento padrão do driver. Mantido para os
// usos que só têm a URL em mãos (ferramentas e testes).
func NewDBURL(databaseURL string) (*DB, error) {
	return NewDBURLWithLimits(databaseURL, PoolLimits{})
}

// buildPoolConfig aplica os limites sobre a configuração derivada da URL. Separado
// de NewDBURLWithLimits para poder ser conferido sem um PostgreSQL de pé: é aqui
// que mora a decisão de dimensionamento, e ela precisa ser testável.
func buildPoolConfig(databaseURL string, limits PoolLimits) (*pgxpool.Config, error) {
	poolCfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DB config: %w", err)
	}

	// A URL manda: um `pool_max_conns` escrito nela é escolha explícita de quem
	// configurou aquele ambiente e não deve ser sobrescrita pelo padrão global.
	if limits.MaxConns > 0 && !strings.Contains(databaseURL, "pool_max_conns") {
		poolCfg.MaxConns = limits.MaxConns
	}
	if limits.MinConns > 0 && !strings.Contains(databaseURL, "pool_min_conns") {
		poolCfg.MinConns = limits.MinConns
	}
	if limits.MaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = limits.MaxConnLifetime
	}
	if limits.MaxConnIdleTime > 0 {
		poolCfg.MaxConnIdleTime = limits.MaxConnIdleTime
	}
	// MinConns acima do teto faria o pgxpool recusar a abertura do pool; um ajuste
	// coerente aqui é melhor que a API não subir por erro de driver.
	if poolCfg.MinConns > poolCfg.MaxConns {
		poolCfg.MinConns = poolCfg.MaxConns
	}
	return poolCfg, nil
}

func NewDBURLWithLimits(databaseURL string, limits PoolLimits) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolCfg, err := buildPoolConfig(databaseURL, limits)
	if err != nil {
		return nil, err
	}

	poolCfg.ConnConfig.Tracer = otelpgx.NewTracer()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping DB: %w", err)
	}

	return &DB{
		Pool: pool,
	}, nil
}

func (db *DB) Queries() *sqlc.Queries {
	return sqlc.New(db.Pool)
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}
