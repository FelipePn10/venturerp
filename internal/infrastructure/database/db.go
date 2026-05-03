package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/config"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

type DB struct {
	Pool *pgxpool.Pool
}

func NewDB(cfg *config.Config) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping DB: %w", err)
	}

	log.Println("Connected to the database successfully")

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
