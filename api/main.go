package main

import (
	"context"
	"os"
	"time"

	applogger "github.com/FelipePn10/panossoerp/internal/infrastructure/logger"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/observability"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/audit"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/config"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database"
	httpmw "github.com/FelipePn10/panossoerp/internal/interfaces/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		applogger.New("info").Fatal("failed to load config", "error", err)
		os.Exit(1)
	}

	log := applogger.New(cfg.LogLevel)

	db, err := database.NewDB(cfg)
	if err != nil {
		log.Fatal("failed to connect to database", "error", err)
	}
	defer db.Close()
	log.Info("database connected")

	shutdownTracing, err := observability.InitTracing(context.Background(), cfg.OTELServiceName, cfg.OTELNamespace, cfg.Env)
	if err != nil {
		log.Fatal("failed to initialize tracing", "error", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracing(ctx); err != nil {
			log.Error("failed to shutdown tracing", "error", err)
		}
	}()

	auditSink := audit.NewPgSink(db.Pool, log)
	defer auditSink.Close()

	api := application{
		config:    cfg,
		logger:    log,
		db:        db,
		metrics:   httpmw.NewMetrics(),
		auditSink: auditSink,
	}

	log.Info("starting server", "port", cfg.ServerPort, "env", cfg.Env, "log_level", cfg.LogLevel)

	if err := api.run(api.mount()); err != nil {
		log.Fatal("application error", "error", err)
	}
}
