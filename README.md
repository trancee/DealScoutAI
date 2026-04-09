# DealScout

An autonomous price-monitoring agent that scrapes product listings from multiple Swiss online shops, tracks prices over time, detects deals based on category-specific rules, and posts notifications to a Telegram channel organized by Forum Topics.

## How it works

```
System Cron → Fetch (concurrent) → Parse (HTML/JSON) → Clean → Normalize → Convert → Evaluate → Dedup → Notify
```

1. **Fetch** product listings from configured shops (HTML pages or JSON APIs) with rate limiting, retries, response caching, and debug dumps.
2. **Parse** products using declarative CSS selectors (HTML), dot-notation field paths (JSON), or embedded JSON in HTML `<script>` tags.
3. **Clean** product names through a two-stage pipeline: shop-specific artifact removal → category-level normalization. Skip-listed brands and exclusion regex patterns filter out accessories.
4. **Normalize** product names to consistent title case with brand-specific corrections (e.g., `APPLE IPHONE 8` → `Apple iPhone 8`).
5. **Convert** prices to CHF using cached exchange rates (24h TTL).
6. **Evaluate** deal rules: first-seen products in the price range trigger a deal; returning products trigger a deal when the price drops by at least `min_discount_pct` from the last tracked price. Sanity bounds reject likely parse errors. A cooldown window prevents duplicate notifications.
7. **Dedup** across shops: if the same product appears from multiple shops, only the cheapest is notified.
8. **Notify** via Telegram `sendMessage` with MarkdownV2 to category-specific Forum Topics.
9. **Prune** old price history, log a run summary with all products found, and exit.

DealScout is designed to run via system cron. It is not a long-running daemon.

## Supported shops

| Shop | Type | Method |
|------|------|--------|
| Ackermann | HTML | CSS selectors with brand prefix |
| Alltron | JSON | Two-step API (search + tiles) |
| Amazon | HTML | CSS selectors |
| Brack | Embedded JSON | `<script>` tag extraction with price divisor |
| Conrad | JSON | Two-step API (search + price enrichment) |
| Foletti | HTML | CSS selectors with pagination |
| HopCash | JSON | Multi-URL categories (iPhone, Samsung, Others) |
| Interdiscount | JSON | REST API with array index field paths |
| Mediamarkt | HTML | `data-test` attribute selectors |
| mobilezone | JSON | JSONP callback stripping |
| Orderflow | HTML | Same platform as Foletti |

Galaxus is configured but disabled (blocked by Akamai Bot Manager — requires headless browser).

## Features

- **Declarative shop configuration** — add new shops via YAML, no code changes needed
- **Multiple parser types** — HTML (CSS selectors), JSON (dot-notation), embedded JSON in `<script>` tags
- **Two-step API enrichment** — search API for product list, secondary API for prices (Conrad, Alltron)
- **Multi-URL categories** — merge products from multiple URL sources into one category (HopCash)
- **Dynamic price placeholders** — `{min_price}`, `{max_price}` in URLs resolved from deal rules
- **Base64-encoded filters** — `{base64_start}...{base64_end}` for shops with encoded URL parameters (Ackermann)
- **JSONP callback stripping** — `jsonp_callback` config strips wrapper before parsing (mobilezone)
- **Product name normalization** — title case, brand corrections, model number cleanup across 20+ brands
- **Cross-shop deduplication** — only notify the cheapest deal per product across all shops
- **Response caching** — file-based TTL cache (default 25 min) avoids redundant fetches during development
- **Response dumps** — every HTTP response saved with a reproducible `curl` command for debugging
- **Telegram proxy support** — HTTP/HTTPS/SOCKS5 proxy for Telegram API requests
- **Zero CGO** — pure Go, single static binary, cross-compiles to any OS/arch

## Requirements

- Go 1.22 or later
- A Telegram Bot token and a supergroup channel with Forum Topics enabled
- (Optional) `golangci-lint` for linting, `make` for build automation

## Setup

### 1. Clone and build

```bash
git clone https://github.com/trancee/DealScout.git
cd DealScout
make build
```

Or without Make:

```bash
go build -trimpath -ldflags '-s -w' -o dealscout ./cmd/dealscout/
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
dump_dir: "data/dumps"
cache_dir: "data/cache"
cache_ttl_minutes: 25
price_history_retention_days: 90
default_max_pages: 5
telegram_topics:
  smartphones: 42            # message_thread_id per category
  laptops: 43
  headphones: 44
```

### 4. Configure shops

See `config/shops.yaml` for all supported shops. Each shop defines:

- **HTML shops**: CSS selectors for product card, title, price, old price, URL, image
- **JSON shops**: dot-notation field paths for products array and fields
- **Embedded JSON**: CSS selector for `<script>` tag + JSON field paths
- **Two-step APIs**: primary search + `price_api` for enrichment

Price placeholders `{min_price}` and `{max_price}` are resolved from deal rules.

### 5. Configure deal rules

Create `config/deal_rules.yaml`:

```yaml
smartphones:
  min_price: 50
  max_price: 350
  min_discount_pct: 10

laptops:
  min_price: 400
  max_price: 1200
  min_discount_pct: 15
```

### 6. Configure filters (optional)

Create `config/filters.yaml`:

```yaml
smartphones:
  skip_brands:
    - Doro
    - Emporia
  exclusion_regex: '(?i)Adapter|AirTag|Case|Charger|Cover\b|Hülle|Kopfhörer|Ladegerät|Schutzfolie'
```

### 7. Set up Telegram notifications

DealScout sends notifications to a Telegram supergroup with Forum Topics. Each product category gets its own topic thread.

#### Step 1: Create a Telegram Bot

1. Open Telegram and search for **@BotFather**
2. Send `/newbot` and follow the prompts
3. Save the **bot token** (e.g., `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`)

#### Step 2: Create a supergroup with Forum Topics

1. Create a new **Group** → enable **Topics** in settings
2. Create a topic for each category (📱 Smartphones, 💻 Laptops, etc.)

#### Step 3: Add the bot as admin

Add the bot to the group and promote to admin with **Post Messages** permission.

#### Step 4: Get the chat ID and topic IDs

```
https://api.telegram.org/bot<BOT_TOKEN>/getUpdates
```

Look for `"chat":{"id":-100xxxxxxxxxx}` and `"message_thread_id":NNN` for each topic.

#### Step 5: Configure credentials

Create `config/secrets.yaml` (gitignored):

```yaml
telegram_bot_token: "123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
telegram_channel: "-1001234567890"
```

Or use environment variables (takes priority):

```bash
export TELEGRAM_BOT_TOKEN="123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
export TELEGRAM_CHANNEL="-1001234567890"
```

#### Step 6: Test the connection

```bash
./dealscout --config ./config/ --test-telegram
```

#### Using a proxy

```yaml
# In settings.yaml — HTTP, HTTPS, or SOCKS5
proxy: "socks5://127.0.0.1:1080"
```

#### Troubleshooting

| Problem | Solution |
|---------|----------|
| `telegram credentials missing` | Check `secrets.yaml` or env vars |
| `sendMessage 403: ...` | Bot is not an admin — check error description |
| `sendMessage 400: ...` | Invalid `message_thread_id` — verify topic IDs |
| Messages go to General | Topic ID is `0` or missing in `telegram_topics` |
| Proxy blocking API | Set `proxy` in `settings.yaml` |

## Usage

```bash
# Full run
./dealscout --config ./config/

# Seed database (no notifications)
./dealscout --config ./config/ --seed

# Dry run (find deals, don't notify)
./dealscout --config ./config/ --dry-run

# Single shop
./dealscout --config ./config/ --shop "Brack"

# Test Telegram
./dealscout --config ./config/ --test-telegram
```

> **Tip:** Always run `--seed` first to populate the database. After seeding, only new products and price drops trigger notifications.

### Cron setup

```cron
0 * * * * /opt/dealscout/dealscout --config /opt/dealscout/config/ >> /var/log/dealscout.log 2>&1
```

### Cross-compilation

```bash
GOOS=linux GOARCH=amd64 make build    # Linux AMD64
GOOS=linux GOARCH=arm64 make build    # Raspberry Pi
```

## Notification format

```
🔥 Samsung Galaxy A15
💰 CHF 119.00 ~CHF 139.00~ (-14%)
🏪 Brack
🔗 View Product
```

Price line shows: current price, ~~strikethrough old price~~, discount percentage.

## Run output

At the end of each run, a product table shows all evaluated products:

```
DEAL PRODUCT                             SHOP             PRICE  OLD PRICE     DROP  REASON              URL
---- -------                             ----             -----  ---------     ----  ------              ---
 🔥  Samsung Galaxy A16                  Brack            99.00                -21%                      https://brack.ch/...
 💲  Samsung Galaxy A16                  Conrad          117.95     141.95           insufficient_disc.  https://conrad.ch/...
     Apple iPhone 17 Pro                 Brack          1049.00                      out_of_range        https://brack.ch/...
```

- 🔥 = deal, cheapest across shops (will be notified)
- 💲 = deal, but cheaper exists at another shop
- (blank) = not a deal (reason shown)

## Project structure

```
DealScout/
├── cmd/dealscout/main.go              # CLI entry point
├── internal/
│   ├── config/                        # YAML config loading + validation
│   ├── jsonpath/                      # Shared JSON dot-notation path walker
│   ├── storage/                       # SQLite (WAL mode) + CRUD operations
│   ├── fetcher/                       # Rate-limited HTTP with retries + UA rotation
│   ├── parser/                        # HTML, JSON, embedded JSON parsers
│   │   └── cleaners/                  # Shop cleaners, filters, name normalization
│   ├── currency/                      # Exchange rate API + SQLite cache
│   ├── deal/                          # Deal rule engine + deduplication
│   ├── notifier/                      # Telegram sendMessage with MarkdownV2
│   └── pipeline/                      # Orchestrator: stages, cache, dump, dedup
├── config/                            # YAML configuration files
│   ├── settings.yaml
│   ├── shops.yaml
│   ├── deal_rules.yaml
│   ├── filters.yaml
│   ├── secrets.yaml                   # Gitignored
│   └── templates/                     # POST body templates for API shops
├── data/                              # Runtime data (gitignored)
│   ├── dealscout.db                   # SQLite database
│   ├── cache/                         # Response cache (TTL-based)
│   └── dumps/                         # Response dumps with curl commands
└── plans/                             # Implementation plans
```

## Development

### Make targets

| Target | Description |
|--------|-------------|
| `make` | Run all checks and build (default) |
| `make build` | Compile optimized binary (~10MB, stripped) |
| `make test` | Run all tests with race detector |
| `make lint` | Run `golangci-lint` |
| `make vet` | Run `go vet` |
| `make fmt` | Format all Go source files |
| `make check` | vet + lint + test |
| `make clean` | Remove binary, cache, and dumps |

### CI

GitHub Actions runs on every push to `main` and on pull requests. See [`.github/workflows/ci.yml`](.github/workflows/ci.yml).

## Dependencies

| Module | Purpose |
|--------|---------|
| [`github.com/PuerkitoBio/goquery`](https://github.com/PuerkitoBio/goquery) | HTML parsing with CSS selectors |
| [`gopkg.in/yaml.v3`](https://github.com/go-yaml/yaml) | YAML configuration parsing |
| [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) | Pure Go SQLite driver (no CGO) |
| [`golang.org/x/text`](https://pkg.go.dev/golang.org/x/text) | Unicode-aware title casing for name normalization |

Everything else uses Go's standard library.

## License

Public Domain. See [LICENSE](LICENSE).
