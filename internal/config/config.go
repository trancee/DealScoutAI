package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load reads all YAML config files from configDir and returns a validated Config.
func Load(configDir string) (*Config, error) {
	cfg := &Config{}

	if err := loadYAML(filepath.Join(configDir, "settings.yaml"), &cfg.Settings); err != nil {
		return nil, fmt.Errorf("loading settings.yaml: %w", err)
	}

	applyDefaults(&cfg.Settings)

	if err := loadShops(filepath.Join(configDir, "shops.yaml"), cfg); err != nil {
		return nil, fmt.Errorf("loading shops.yaml: %w", err)
	}

	if err := loadYAML(filepath.Join(configDir, "deal_rules.yaml"), &cfg.DealRules); err != nil {
		return nil, fmt.Errorf("loading deal_rules.yaml: %w", err)
	}

	if err := loadYAML(filepath.Join(configDir, "filters.yaml"), &cfg.Filters); err != nil {
		return nil, fmt.Errorf("loading filters.yaml: %w", err)
	}

	if err := loadSecrets(configDir, cfg); err != nil {
		return nil, err
	}

	resolvePaths(configDir, cfg)

	return cfg, nil
}

// shopsFile is the top-level wrapper for shops.yaml which nests shops under a key.
type shopsFile struct {
	Shops []Shop `yaml:"shops"`
}

func loadShops(path string, cfg *Config) error {
	var sf shopsFile
	if err := loadYAML(path, &sf); err != nil {
		return err
	}
	cfg.Shops = sf.Shops
	return nil
}

func loadSecrets(configDir string, cfg *Config) error {
	// Environment variables take priority.
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	channel := os.Getenv("TELEGRAM_CHANNEL")

	// Try loading secrets file (optional if env vars are set).
	secretsPath := filepath.Join(configDir, "secrets.yaml")
	if err := loadYAML(secretsPath, &cfg.Secrets); err != nil && (token == "" || channel == "") {
		return fmt.Errorf("loading secrets.yaml: %w (set TELEGRAM_BOT_TOKEN and TELEGRAM_CHANNEL env vars as alternative)", err)
	}

	if token != "" {
		cfg.Secrets.TelegramBotToken = token
	}
	if channel != "" {
		cfg.Secrets.TelegramChannel = channel
	}

	if cfg.Secrets.TelegramBotToken == "" || cfg.Secrets.TelegramChannel == "" {
		return fmt.Errorf("telegram credentials missing: set TELEGRAM_BOT_TOKEN/TELEGRAM_CHANNEL env vars or provide secrets.yaml")
	}

	return nil
}

func resolvePaths(configDir string, cfg *Config) {
	if cfg.Settings.DatabasePath != "" && !filepath.IsAbs(cfg.Settings.DatabasePath) {
		cfg.Settings.DatabasePath = filepath.Join(configDir, cfg.Settings.DatabasePath)
	}
	if cfg.Settings.DumpDir != "" && !filepath.IsAbs(cfg.Settings.DumpDir) {
		cfg.Settings.DumpDir = filepath.Join(configDir, cfg.Settings.DumpDir)
	}
	for i := range cfg.Shops {
		for j := range cfg.Shops[i].Categories {
			bt := cfg.Shops[i].Categories[j].BodyTemplate
			if bt != "" && !filepath.IsAbs(bt) {
				cfg.Shops[i].Categories[j].BodyTemplate = filepath.Join(configDir, bt)
			}
			if pa := cfg.Shops[i].Categories[j].PriceAPI; pa != nil && pa.BodyTemplate != "" && !filepath.IsAbs(pa.BodyTemplate) {
				cfg.Shops[i].Categories[j].PriceAPI.BodyTemplate = filepath.Join(configDir, pa.BodyTemplate)
			}
		}
	}
}

func loadYAML(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, target)
}

func applyDefaults(s *Settings) {
	if s.DefaultMaxPages == 0 {
		s.DefaultMaxPages = 5
	}
	if s.LogLevel == "" {
		s.LogLevel = "INFO"
	}
	if s.LogFormat == "" {
		s.LogFormat = "text"
	}
	if s.MaxRetries == 0 {
		s.MaxRetries = 3
	}
	if s.MaxConcurrentShops == 0 {
		s.MaxConcurrentShops = 5
	}
	if s.PriceHistoryRetentionDays == 0 {
		s.PriceHistoryRetentionDays = 90
	}
	if s.NotificationCooldownHours == 0 {
		s.NotificationCooldownHours = 24
	}
	if s.FetchDelaySeconds == 0 {
		s.FetchDelaySeconds = 2
	}
	if s.ExchangeRateCacheTTLHours == 0 {
		s.ExchangeRateCacheTTLHours = 24
	}
	if s.ExchangeRateProvider == "" {
		s.ExchangeRateProvider = "https://api.frankfurter.app"
	}
	if s.DatabasePath == "" {
		s.DatabasePath = "data/dealscout.db"
	}
	if s.DumpDir == "" {
		s.DumpDir = "data/dumps"
	}
}
