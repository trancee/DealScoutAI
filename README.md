# DealScout

An autonomous price-monitoring agent that scrapes product listings from multiple online shops, tracks prices over time, detects deals based on category-specific rules, and posts notifications to a Telegram channel organized by Forum Topics.

## How it works

```
System Cron → Fetch (concurrent) → Parse (HTML/JSON) → Clean → Convert → Evaluate → Notify
```

1. **Fetch** product listings from configured shops (HTML pages or JSON APIs) with rate limiting and retries.
2. **Parse** products using declarative CSS selectors (HTML) or dot-notation field paths (JSON).
3. **Clean** product names through a two-stage pipeline: shop-specific artifact removal → category-level normalization. Skip-listed brands and exclusion regex patterns are filtered out.
4. **Convert** prices to CHF using cached exchange rates (24h TTL).
5. **Evaluate** deal rules: first-seen products in the price range trigger a deal; returning products trigger a deal when the price drops by at least `min_discount_pct` from the last tracked price. Sanity bounds reject likely parse errors. A cooldown window prevents duplicate notifications.
6. **Notify** via Telegram `sendPhoto` (with `sendMessage` fallback) to category-specific Forum Topics.
7. **Prune** old price history, log a run summary, and exit.

DealScout is designed to run via system cron. It is not a long-running daemon.

## Features

- **Declarative shop configuration** — add new shops via YAML, no code changes needed for standard HTML/JSON shops
- **Two-stage product name cleaning** — shop cleaners + category cleaners for cross-shop product matching
- **Cross-shop price tracking** — one product identity across shops, price history per shop
- **Smart deal detection** — based on tracked price history, not shop-advertised discounts
- **Notification deduplication** — configurable cooldown window per product+shop
- **Concurrent fetching** — shops processed in parallel with a configurable worker pool
- **Zero CGO** — pure Go, single static binary, cross-compiles to any OS/arch
- **Minimal dependencies** — 3 direct dependencies: `goquery`, `yaml.v3`, `modernc.org/sqlite`

## Requirements

- Go 1.22 or later
- A Telegram Bot token and a supergroup channel with Forum Topics enabled

## Setup

### 1. Clone and build

```bash
git clone https://github.com/trancee/DealScout.git
cd DealScout
go build -o dealscout ./cmd/dealscout/
```

### 2. Create config directory

```bash
mkdir -p config/templates
```

### 3. Configure settings

Create `config/settings.yaml`:

```yaml
base_currency: "CHF"
notification_cooldown_hours: 24
fetch_delay_seconds: 2
max_retries: 3
max_concurrent_shops: 5
exchange_rate_provider: "https://api.frankfurter.app"
exchange_rate_cache_ttl_hours: 24
log_level: "INFO"            # DEBUG, INFO, WARNING, ERROR
log_format: "text"           # text or json
database_path: "data/dealscout.db"
price_history_retention_days: 90
default_max_pages: 5
telegram_topics:
  smartphone: 0              # message_thread_id (0 = General topic)
  laptop: 0
  headphones: 0
```

### 4. Configure shops

Create `config/shops.yaml`:

```yaml
shops:
  - name: "Amazon"
    source_type: "html"
    cleaner: "amazon"
    base_url: "https://www.amazon.de"
    categories:
      - category: "smartphone"
        url: "https://www.amazon.de/s?k=smartphone&page={page}"
        max_pages: 3
        pagination:
          type: "page_param"
          param: "page"
          start: 1
        selectors:
          product_card: "div[data-component-type='s-search-result']"
          title: "h2 a span"
          price: "span.a-price span.a-offscreen"
          old_price: "span.a-price[data-a-strike] span.a-offscreen"
          url: "h2 a[href]"
          image: "img.s-image[src]"
        currency: "EUR"

  - name: "Galaxus"
    source_type: "json"
    method: "POST"
    cleaner: "galaxus"
    categories:
      - category: "smartphone"
        url: "https://www.galaxus.ch/api/graphql/product-type-filter-products"
        body_template: "templates/galaxus_smartphone.json"
        max_pages: 5
        pagination:
          type: "offset"
          param: "offset"
          per_page: 24
        fields:
          products: "data.productType.filterProducts.products.results"
          title: "product.name"
          price: "offer.price.amountInclusive"
          old_price: "offer.insteadOfPrice.price.amountInclusive"
          url: "product.productId"
          image: "product.imageUrl"
        currency: "CHF"
```

### 5. Configure deal rules

Create `config/deal_rules.yaml`:

```yaml
smartphone:
  min_price: 50
  max_price: 350
  min_discount_pct: 10

laptop:
  min_price: 400
  max_price: 1200
  min_discount_pct: 15

headphones:
  min_price: 20
  max_price: 200
  min_discount_pct: 5
```

### 6. Configure filters (optional)

Create `config/filters.yaml`:

```yaml
smartphone:
  skip_brands:
    - Doro
    - Emporia
  exclusion_regex: "(?i)Cover|Hülle|Schutzfolie"

laptop:
  skip_brands: []
  exclusion_regex: "(?i)Tasche|Sleeve"
```

### 7. Add Telegram credentials

Create `config/secrets.yaml` (gitignored):

```yaml
telegram_bot_token: "123456:ABC-DEF..."
telegram_channel: "-1001234567890"
```

Or use environment variables (takes priority over the file):

```bash
export TELEGRAM_BOT_TOKEN="123456:ABC-DEF..."
export TELEGRAM_CHANNEL="-1001234567890"
```

## Usage

```bash
# Full run
./dealscout --config ./config/

# Initial DB population — stores products, sends zero notifications
./dealscout --config ./config/ --seed

# Test run — full pipeline, logs deals, doesn't post to Telegram
./dealscout --config ./config/ --dry-run

# Single shop only
./dealscout --config ./config/ --shop "Galaxus"
```

### Cron setup

```cron
0 * * * * /opt/dealscout/dealscout --config /opt/dealscout/config/ >> /var/log/dealscout.log 2>&1
```

### Cross-compilation

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o dealscout-linux ./cmd/dealscout/

# Linux ARM64 (Raspberry Pi)
GOOS=linux GOARCH=arm64 go build -o dealscout-arm64 ./cmd/dealscout/
```

## Telegram notification format

Notifications are sent as photos with a MarkdownV2 caption to category-specific Forum Topics:

```
🔥 Samsung Galaxy A15
💰 CHF 119.00
📉 -14% (was CHF 139)
🏷️ Shop says: CHF 159
🏪 Galaxus
🔗 View Product
```

## Project structure

```
DealScout/
├── cmd/dealscout/main.go              # CLI entry point
├── internal/
│   ├── config/                        # YAML config loading + validation
│   ├── storage/                       # SQLite (WAL mode) + CRUD operations
│   ├── fetcher/                       # Rate-limited HTTP with retries + UA rotation
│   ├── parser/                        # HTML (goquery) + JSON (dot-notation) parsers
│   │   └── cleaners/                  # Shop + category product name cleaning
│   ├── currency/                      # Exchange rate API + SQLite cache
│   ├── deal/                          # Deal rule engine + deduplication
│   ├── notifier/                      # Telegram sendPhoto + sendMessage fallback
│   └── pipeline/                      # Orchestrator with concurrent worker pool
├── config/                            # YAML configuration files
│   ├── settings.yaml
│   ├── shops.yaml
│   ├── deal_rules.yaml
│   ├── filters.yaml
│   ├── secrets.yaml                   # Gitignored
│   └── templates/                     # POST body templates for API shops
└── plans/                             # Implementation plans
```

## Testing

```bash
# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Verbose output
go test -v ./...
```

The test suite covers 53 test functions (83 including sub-cases) across all modules, using in-memory SQLite databases and `httptest` servers — no external services required.

## Dependencies

| Module | Purpose |
|--------|---------|
| [`github.com/PuerkitoBio/goquery`](https://github.com/PuerkitoBio/goquery) | HTML parsing with CSS selectors |
| [`gopkg.in/yaml.v3`](https://github.com/go-yaml/yaml) | YAML configuration parsing |
| [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) | Pure Go SQLite driver (no CGO) |

Everything else uses Go's standard library.

## License

Public Domain. See [LICENSE](LICENSE).
