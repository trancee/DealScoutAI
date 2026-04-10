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
	Proxy                     string         `yaml:"proxy"`
	DatabasePath              string         `yaml:"database_path"`
	DumpDir                   string         `yaml:"dump_dir"`
	CacheDir                  string         `yaml:"cache_dir"`
	CacheTTLMinutes           int            `yaml:"cache_ttl_minutes"`
	PriceHistoryRetentionDays int            `yaml:"price_history_retention_days"`
	DefaultMaxPages           int            `yaml:"default_max_pages"`
	TelegramTopics            map[string]int `yaml:"telegram_topics"`
}

// Shop represents a single shop with its categories.
type Shop struct {
	Name          string            `yaml:"name"`
	SourceType    string            `yaml:"source_type"`
	Method        string            `yaml:"method"`
	Cleaner       string            `yaml:"cleaner"`
	BaseURL       string            `yaml:"base_url"`
	Headers       map[string]string `yaml:"headers"`
	TokenEndpoint *TokenEndpoint    `yaml:"token_endpoint"`
	PriceBuckets  *PriceBuckets     `yaml:"price_buckets"`
	Categories    []ShopCategory    `yaml:"categories"`
}

// ShopCategory represents one category within a shop.
// Fields are grouped logically via inline sub-structs but remain flat in YAML.
type ShopCategory struct {
	Category string `yaml:"category"`

	// Fetching — how to retrieve listing pages.
	Fetching `yaml:",inline"`

	// Parsing — how to extract products from the response.
	Parsing `yaml:",inline"`

	// Pricing — price adjustment, secondary API, and currency.
	Pricing `yaml:",inline"`

	// Output — how to construct product URLs.
	Output `yaml:",inline"`
}

// Fetching controls how listing pages are retrieved.
type Fetching struct {
	URL          string     `yaml:"url"`
	URLs         []string   `yaml:"urls"`
	BodyTemplate string     `yaml:"body_template"`
	MaxPages     int        `yaml:"max_pages"`
	Pagination   Pagination `yaml:"pagination"`
}

// Parsing controls how products are extracted from the response.
type Parsing struct {
	Selectors     map[string]string `yaml:"selectors"`
	Fields        map[string]string `yaml:"fields"`
	JSONSelector  string            `yaml:"json_selector"`
	JSONPCallback string            `yaml:"jsonp_callback"`
}

// Pricing controls price adjustment, enrichment, and currency.
type Pricing struct {
	PriceDivisor float64   `yaml:"price_divisor"`
	PriceAPI     *PriceAPI `yaml:"price_api"`
	Currency     string    `yaml:"currency"`
}

// Output controls how product URLs are constructed.
type Output struct {
	URLTemplate string `yaml:"url_template"`
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
	TitlePath    string            `yaml:"title_path"`
	ImagePath    string            `yaml:"image_path"`
	ProductsPath string            `yaml:"products_path"`
}

// Pagination defines how to navigate through listing pages.
type Pagination struct {
	Type    string `yaml:"type"`
	Param   string `yaml:"param"`
	PerPage int    `yaml:"per_page"`
	Start   int    `yaml:"start"`
}

// TokenEndpoint defines an HTTP endpoint that returns a bearer token.
// The token is fetched once per shop run and injected as an Authorization header.
type TokenEndpoint struct {
	URL       string `yaml:"url"`
	TokenPath string `yaml:"token_path"`
}

// PriceBuckets defines predefined price ranges for APIs that only support bucket-based filtering.
// At runtime, buckets overlapping the deal rule's [min_price, max_price] are selected
// and formatted into the {price_buckets} URL placeholder.
type PriceBuckets struct {
	Format string       `yaml:"format"`
	Ranges []PriceRange `yaml:"ranges"`
}

// PriceRange defines a single price bucket with start and end values.
type PriceRange struct {
	Start float64 `yaml:"start"`
	End   float64 `yaml:"end"`
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
