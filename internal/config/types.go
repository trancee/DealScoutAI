package config

// Config holds the complete application configuration loaded from all YAML files.
type Config struct {
	Settings  Settings
	Shops     []Shop
	DealRules map[string]DealRule
	Filters   map[string]Filter
	Secrets   Secrets
}

// Settings maps to config/settings.yaml.
type Settings struct {
	BaseCurrency              string         `yaml:"base_currency"`
	NotificationCooldownHours int            `yaml:"notification_cooldown_hours"`
	FetchDelaySeconds         int            `yaml:"fetch_delay_seconds"`
	MaxRetries                int            `yaml:"max_retries"`
	MaxConcurrentShops        int            `yaml:"max_concurrent_shops"`
	ExchangeRateProvider      string         `yaml:"exchange_rate_provider"`
	ExchangeRateCacheTTLHours int            `yaml:"exchange_rate_cache_ttl_hours"`
	LogLevel                  string         `yaml:"log_level"`
	LogFormat                 string         `yaml:"log_format"`
	DatabasePath              string         `yaml:"database_path"`
	PriceHistoryRetentionDays int            `yaml:"price_history_retention_days"`
	DefaultMaxPages           int            `yaml:"default_max_pages"`
	TelegramTopics            map[string]int `yaml:"telegram_topics"`
}

// Shop represents a single shop with its categories.
type Shop struct {
	Name       string            `yaml:"name"`
	SourceType string            `yaml:"source_type"`
	Method     string            `yaml:"method"`
	Cleaner    string            `yaml:"cleaner"`
	BaseURL    string            `yaml:"base_url"`
	Headers    map[string]string `yaml:"headers"`
	Categories []ShopCategory    `yaml:"categories"`
}

// ShopCategory represents one category within a shop.
type ShopCategory struct {
	Category      string            `yaml:"category"`
	URL           string            `yaml:"url"`
	BodyTemplate  string            `yaml:"body_template"`
	MaxPages      int               `yaml:"max_pages"`
	Pagination    Pagination        `yaml:"pagination"`
	Selectors     map[string]string `yaml:"selectors"`
	Fields        map[string]string `yaml:"fields"`
	JSONSelector  string            `yaml:"json_selector"`
	PriceDivisor  float64           `yaml:"price_divisor"`
	PriceAPI      *PriceAPI         `yaml:"price_api"`
	JSONPCallback string            `yaml:"jsonp_callback"`
	Currency      string            `yaml:"currency"`
}

// PriceAPI defines a secondary API call to fetch prices for products.
type PriceAPI struct {
	URL          string            `yaml:"url"`
	BodyTemplate string            `yaml:"body_template"`
	Headers      map[string]string `yaml:"headers"`
	IDField      string            `yaml:"id_field"`
	PricePath    string            `yaml:"price_path"`
	OldPricePath string            `yaml:"old_price_path"`
	IDPath       string            `yaml:"id_path"`
	ProductsPath string            `yaml:"products_path"`
}

// Pagination defines how to navigate through listing pages.
type Pagination struct {
	Type    string `yaml:"type"`
	Param   string `yaml:"param"`
	PerPage int    `yaml:"per_page"`
	Start   int    `yaml:"start"`
}

// DealRule defines the deal detection parameters for a category.
type DealRule struct {
	MinPrice       float64 `yaml:"min_price"`
	MaxPrice       float64 `yaml:"max_price"`
	MinDiscountPct float64 `yaml:"min_discount_pct"`
}

// Filter defines skip lists and exclusion patterns for a category.
type Filter struct {
	SkipBrands     []string `yaml:"skip_brands"`
	ExclusionRegex string   `yaml:"exclusion_regex"`
}

// Secrets holds sensitive credentials.
type Secrets struct {
	TelegramBotToken string `yaml:"telegram_bot_token"`
	TelegramChannel  string `yaml:"telegram_channel"`
}
