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
- **Embedded JSON extraction** — parse product data from `<script>` tags in HTML (e.g., JSON-LD, server-side state)
- **Two-stage product name cleaning** — shop cleaners + category cleaners for cross-shop product matching
- **Cross-shop price tracking** — one product identity across shops, price history per shop
- **Smart deal detection** — based on tracked price history, not shop-advertised discounts
- **Notification deduplication** — configurable cooldown window per product+shop
- **Concurrent fetching** — shops processed in parallel with a configurable worker pool
- **Response dumps** — every HTTP response saved to disk with a reproducible `curl` command for debugging
- **Zero CGO** — pure Go, single static binary, cross-compiles to any OS/arch
- **Minimal dependencies** — 3 direct dependencies: `goquery`, `yaml.v3`, `modernc.org/sqlite`

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

### 7. Set up Telegram notifications

DealScout sends notifications to a Telegram supergroup with Forum Topics. Each product category (smartphone, laptop, etc.) gets its own topic thread.

#### Step 1: Create a Telegram Bot

1. Open Telegram and search for **@BotFather**
2. Send `/newbot` and follow the prompts to name your bot
3. BotFather will give you a **bot token** like `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`
4. Save this token — you'll need it for `secrets.yaml`

#### Step 2: Create a supergroup with Forum Topics

1. In Telegram, create a new **Group** (not a channel)
2. Go to **Group Settings** → **Group type** → set to **Public** (temporarily, to get the chat ID)
3. Enable **Topics**: Group Settings → **Topics** → toggle on
4. Create a topic for each product category you want to monitor:
   - 📱 Smartphones
   - 💻 Laptops
   - 🎧 Headphones

#### Step 3: Add the bot to the group

1. Open the group and tap **Add Members**
2. Search for your bot by username and add it
3. Go to **Group Settings** → **Administrators** → **Add Admin** → select your bot
4. Grant these permissions: **Post Messages**, **Edit Messages**

#### Step 4: Get the group chat ID

Send a message in the group, then open this URL in your browser (replace `BOT_TOKEN` with your actual token):

```
https://api.telegram.org/bot<BOT_TOKEN>/getUpdates
```

Look for `"chat":{"id":-100xxxxxxxxxx}` in the response. The chat ID is the negative number starting with `-100`.

Alternatively, add **@RawDataBot** to the group temporarily — it will reply with the chat ID.

#### Step 5: Get the topic thread IDs

Each Forum Topic has a `message_thread_id`. To find them:

1. Send a message in each topic
2. Call `getUpdates` again (see Step 4)
3. Look for `"message_thread_id":NNN` in each message — that's the topic ID

The **General** topic always has `message_thread_id: 0` (or is omitted).

#### Step 6: Configure DealScout

Add the topic IDs to `config/settings.yaml`:

```yaml
telegram_topics:
  smartphone: 42       # message_thread_id for the 📱 Smartphones topic
  laptop: 43           # message_thread_id for the 💻 Laptops topic
  headphones: 44       # message_thread_id for the 🎧 Headphones topic
```

Create `config/secrets.yaml` (gitignored):

```yaml
telegram_bot_token: "123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
telegram_channel: "-1001234567890"
```

Or use environment variables (takes priority over the file):

```bash
export TELEGRAM_BOT_TOKEN="123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
export TELEGRAM_CHANNEL="-1001234567890"
```

#### Step 7: Test the connection

```bash
# Dry run — finds deals but doesn't send notifications
./dealscout --config ./config/ --dry-run

# Seed the database first (no notifications)
./dealscout --config ./config/ --seed

# Full run — sends notifications to Telegram
./dealscout --config ./config/
```

> **Tip:** Always run `--seed` first to populate the database without flooding the channel. After seeding, only new products and price drops will trigger notifications.

#### Troubleshooting

| Problem | Solution |
|---------|----------|
| `telegram credentials missing` | Check `secrets.yaml` exists or env vars are set |
| `sendPhoto returned 403` | Bot is not an admin in the group |
| `sendPhoto returned 400` | Invalid `message_thread_id` — verify topic IDs |
| Notifications go to General instead of a topic | The topic ID is `0` or missing in `telegram_topics` |
| Image not showing | Telegram couldn't fetch the image URL — falls back to text-only `sendMessage` |
| Duplicate notifications | Check `notification_cooldown_hours` in `settings.yaml` (default: 24h) |

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
GOOS=linux GOARCH=amd64 make build

# Linux ARM64 (Raspberry Pi)
GOOS=linux GOARCH=arm64 make build
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
| `make clean` | Remove the compiled binary |

### Running tests directly

```bash
go test ./...           # All tests
go test -race ./...     # With race detector
go test -v ./...        # Verbose output
```

The test suite covers 53 test functions (83 including sub-cases) across all modules, using in-memory SQLite databases and `httptest` servers — no external services required.

### CI

GitHub Actions runs on every push to `main` and on pull requests. The pipeline runs lint, test (with race detector and coverage), and build. See [`.github/workflows/ci.yml`](.github/workflows/ci.yml).

## Dependencies

| Module | Purpose |
|--------|---------|
| [`github.com/PuerkitoBio/goquery`](https://github.com/PuerkitoBio/goquery) | HTML parsing with CSS selectors |
| [`gopkg.in/yaml.v3`](https://github.com/go-yaml/yaml) | YAML configuration parsing |
| [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) | Pure Go SQLite driver (no CGO) |

Everything else uses Go's standard library.

## License

Public Domain. See [LICENSE](LICENSE).
