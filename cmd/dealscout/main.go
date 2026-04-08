package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/trancee/DealScout/internal/config"
	"github.com/trancee/DealScout/internal/storage"
)

func main() {
	configDir := flag.String("config", "./config/", "Path to config directory")
	seed := flag.Bool("seed", false, "Populate DB without sending notifications")
	dryRun := flag.Bool("dry-run", false, "Full pipeline, log deals, don't notify")
	shopFilter := flag.String("shop", "", "Run only the named shop")
	flag.Parse()

	cfg, err := config.Load(*configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}

	initLogger(cfg.Settings.LogLevel, cfg.Settings.LogFormat)

	slog.Info("config loaded",
		"shops", len(cfg.Shops),
		"deal_rules", len(cfg.DealRules),
		"filters", len(cfg.Filters),
		"base_currency", cfg.Settings.BaseCurrency,
	)

	db, err := storage.Open(cfg.Settings.DatabasePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	slog.Info("database opened", "path", cfg.Settings.DatabasePath)

	// Flags are parsed and available for future phases.
	_ = seed
	_ = dryRun
	_ = shopFilter
}

func initLogger(level, format string) {
	var lvl slog.Level
	switch level {
	case "DEBUG":
		lvl = slog.LevelDebug
	case "WARNING":
		lvl = slog.LevelWarn
	case "ERROR":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}
