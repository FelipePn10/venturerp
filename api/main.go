package main

import (
	"os"

	applogger "github.com/FelipePn10/panossoerp/internal/infrastructure/logger"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/config"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		// Logger not yet ready — use a temporary one just for this fatal.
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

	api := application{
		config: cfg,
		logger: log,
		db:     db,
	}

	log.Info("starting server", "port", cfg.ServerPort, "env", cfg.Env, "log_level", cfg.LogLevel)

	if err := api.run(api.mount()); err != nil {
		log.Fatal("application error", "error", err)
	}
}
