# DealScout — Design Document

> **Version:** 3.0
> **Date:** 2026-04-08
> **Status:** Approved

---

## 1. Overview

DealScout is an autonomous price-monitoring agent that scrapes product listings from multiple online shops, tracks prices over time, detects deals based on category-specific rules, and posts notifications to a Telegram channel organized by category topics.

It is a **pure data pipeline** — no runtime LLM. Shop data is extracted via declarative CSS selectors (HTML shops) or JSON field path mappings (API shops), with optional per-shop and per-category cleaning functions for product name normalization.

### 1.1 Key Design Principles

| Principle | Description |
|---|---|
| **No runtime LLM** | LLM is a dev-time tool only (generating parser code). Runtime is pure parsing. |
| **Declarative parsers** | Shop extraction defined in YAML; custom code only when needed. |
| **Two-stage cleaning** | Shop cleaner → category cleaner for product name normalization. |
| **Cross-shop tracking** | One product identity across shops; price history per shop. |
| **Deduplication** | Never re-post a deal for the same product+shop within a cooldown window. |
| **Run once, exit** | Designed for system cron. No long-running daemon. |
| **Minimal dependencies** | Three external Go dependencies. Everything else is stdlib. |
| **Concurrent by default** | Shops are fetched in parallel; pages within a shop are sequential. |

---

## 2. Architecture

### 2.1 High-Level Data Flow

```
┌───────────────┐     ┌──────────────────────┐     ┌───────────────────┐
│  System Cron  │────▶│  Fetcher             │────▶│  Parser           │
│  (hourly)     │     │  (rate-limited HTTP) │     │  (HTML or JSON)   │
└───────────────┘     │  concurrent per shop │     └────────┬──────────┘
                      └──────────────────────┘              │ []RawProduct
                                                            ▼
                                                   ┌───────────────────┐
                                                   │  Cleaning         │
                                                   │  Pipeline         │
                                                   │  shop → category  │
                                                   └────────┬──────────┘
                                                            │ []CleanProduct
                                                            ▼
┌───────────────┐     ┌──────────────────────┐     ┌───────────────────┐
│  Telegram     │◀────│  Deal Evaluator      │◀────│  Storage (SQLite) │
│  (sendPhoto)  │     │  (rules + dedup)     │     │  (price history)  │
│  Forum Topics │     └──────────────────────┘     └───────────────────┘
└───────────────┘
```

### 2.2 Processing Pipeline (per cron run)

1. **Load config** — read `settings.yaml`, `shops.yaml`, `deal_rules.yaml`, `filters.yaml`.
2. **Open DB** — connect to SQLite (WAL mode for concurrent writes). Create tables if missing.
3. **Refresh exchange rates** — check SQLite cache, fetch from API if stale (>24h).
4. **For each shop** (concurrently, up to `max_concurrent_shops`):
   - **For each category** within the shop:
     - **For each page** up to `max_pages` (sequential, respecting `fetch_delay_seconds`):
       - **Fetch** the listing page URL with retries and User-Agent rotation.
       - **Parse** using the shop's `source_type` (`html` or `json`) and declared field mappings / CSS selectors.
     - **Clean** each product through the two-stage pipeline:
       - Shop cleaner strips shop-specific junk.
       - Category cleaner normalizes brand/model names, applies skip lists from `filters.yaml`.
     - **Parse prices** using shared price parser utility.
     - **Resolve URLs** against base URL (auto-derived or configured).
     - **Normalize currency** to CHF using cached exchange rates.
     - **Sanity check** — reject prices outside `[min_price * 0.1, max_price * 10]`.
     - **Store** product and append price history record (per shop).
     - **Evaluate deal rules**:
       - First-seen product + price in range → deal.
       - Returning product + price in range + drop ≥ `min_discount_pct` from last DB price → deal.
     - **Deduplication check** — skip if same product+shop was notified within cooldown.
     - **Collect deals** for notification.
5. **Notify** — send all deals via Telegram `sendPhoto` (or `sendMessage` fallback) to the correct Forum topic per category. 1-second delay between messages.
6. **Prune** — delete price history older than `price_history_retention_days`.
7. **Log run summary** — products checked, deals found, notifications sent, errors, duration.
8. **Exit.**

---

## 3. Module Design

### 3.1 Project Structure

```
DealScout/
├── cmd/
│   └── dealscout/
│       └── main.go                    # CLI entry point (run once, exit)
├── internal/
│   ├── fetcher/
│   │   └── fetcher.go                 # Rate-limited HTTP with retries & UA rotation
│   ├── parser/
│   │   ├── html.go                    # Generic HTML parser (CSS selectors via goquery)
│   │   ├── json.go                    # Generic JSON parser (dot-notation field paths)
│   │   ├── price.go                   # Shared price string parser (all formats)
│   │   ├── registry.go                # Maps shop config → parser + cleaners
│   │   └── cleaners/
│   │       ├── shops/                 # Per-shop cleaning
│   │       │   ├── amazon.go
│   │       │   ├── galaxus.go
│   │       │   └── ...
│   │       └── categories/            # Per-category normalization
│   │           ├── smartphone.go
│   │           ├── laptop.go
│   │           └── headphones.go
│   ├── deal/
│   │   └── evaluator.go               # Deal rule engine + sanity bounds
│   ├── storage/
│   │   ├── database.go                # SQLite connection (WAL mode) + migrations
│   │   └── models.go                  # Product, PriceHistory, DealNotification, ExchangeRate
│   ├── notifier/
│   │   └── telegram.go                # sendPhoto + sendMessage fallback (raw net/http)
│   ├── currency/
│   │   └── converter.go               # Exchange rate API + SQLite-backed 24h cache
│   └── config/
│       └── config.go                  # YAML config loader + types
├── config/
│   ├── shops.yaml                     # Shop definitions + field mappings (nested by category)
│   ├── deal_rules.yaml                # Per-category deal rules
│   ├── filters.yaml                   # Per-category skip lists & exclusion regexes
│   ├── settings.yaml                  # Global settings (no secrets)
│   ├── secrets.yaml                   # Telegram credentials (gitignored)
│   ├── secrets.yaml.example           # Reference template (committed)
│   └── templates/                     # POST body templates for GraphQL/API shops
│       ├── galaxus_smartphone.json
│       ├── galaxus_laptop.json
│       └── ...
├── go.mod
├── go.sum
└── README.md
```

### 3.2 Module Responsibilities

#### `cmd/dealscout/main.go`

Entry point. Parses CLI flags, loads config, opens DB, launches concurrent shop workers, collects deals, sends notifications, prunes old data, logs summary, exits.

**CLI Flags:**

| Flag | Behavior |
|---|---|
| `--config "/path/to/config/"` | Config directory (default: `./config/`) |
| `--seed` | Populate DB, send zero notifications |
| `--dry-run` | Full pipeline, log deals, don't post to Telegram |
| `--shop "Galaxus"` | Run only the named shop |

All file paths in YAML configs are resolved relative to the `--config` directory.

Fatal errors (config parse failure, DB can't open, missing Telegram credentials) cause immediate exit. Shop-level failures are logged and skipped.

#### `internal/fetcher/`

| File | Responsibility |
|---|---|
| `fetcher.go` | Downloads a URL with configurable delay between requests, User-Agent rotation, optional proxy support, and exponential-backoff retries on HTTP 429/5xx. Supports GET and POST (with body from template files, placeholders replaced for pagination). Applies per-shop custom HTTP headers (e.g., CSRF tokens). Logs request/response details at DEBUG level. |

#### `internal/parser/`

| File | Responsibility |
|---|---|
| `html.go` | Generic HTML parser. Uses CSS selectors (via goquery) declared in `shops.yaml` to find product cards and extract fields (title, price, old_price, URL, image). |
| `json.go` | Generic JSON parser. Uses dot-notation field paths declared in `shops.yaml` to walk response JSON and extract product arrays and fields. |
| `price.go` | Shared price string parser. Handles all European formats: `CHF 119.–`, `€ 99,90`, `1'299.00`, `1.299,00`, etc. Strips currency symbols, normalizes separators, returns `float64`. |
| `registry.go` | Resolves a shop config to the correct parser type (HTML, JSON, or embedded JSON) + optional shop cleaner + category cleaner. |
| `cleaners/shops/*.go` | Per-shop cleaning functions. Strip shop-specific artifacts from product names. |
| `cleaners/categories/*.go` | Per-category normalization. Brand/model canonicalization, skip list filtering (reads from `filters.yaml`). |

#### `internal/deal/`

| File | Responsibility |
|---|---|
| `evaluator.go` | Applies deal rules from `deal_rules.yaml`. Checks sanity bounds `[min_price * 0.1, max_price * 10]`, price range, discount % from last DB price, first-seen logic, and notification cooldown. |

#### `internal/storage/`

| File | Responsibility |
|---|---|
| `database.go` | SQLite connection (WAL mode for concurrent goroutines), auto-creates parent directories for the database path, table creation, migrations, retention pruning. |
| `models.go` | Data structures and DB queries for Product, PriceHistory, DealNotification, ExchangeRate. |

#### `internal/notifier/`

| File | Responsibility |
|---|---|
| `telegram.go` | Sends notifications via raw `net/http` POST to Telegram Bot API. `sendPhoto` with MarkdownV2 caption to the correct Forum topic (`message_thread_id`) per category. Falls back to `sendMessage` if no image URL or if photo send fails. 1-second delay between messages. |

#### `internal/currency/`

| File | Responsibility |
|---|---|
| `converter.go` | Fetches exchange rates from configurable API (default: `frankfurter.app`). Caches in SQLite `exchange_rates` table for 24 hours. Converts any currency to CHF. Logs warning and skips product on conversion failure. |

#### `internal/config/`

| File | Responsibility |
|---|---|
| `config.go` | Loads and validates all YAML config files. Defines Go structs for settings, shops (nested by category), deal rules, and filters. Resolves all file paths relative to the config directory. |

---

## 4. Data Model

### 4.1 Entity-Relationship Diagram

```
┌─────────────────┐       ┌──────────────────────┐       ┌─────────────────────┐
│    Product      │       │   PriceHistory       │       │  DealNotification   │
├─────────────────┤       ├──────────────────────┤       ├─────────────────────┤
│ id (PK)         │──1:N─▶│ id (PK)              │       │ id (PK)             │
│ name (cleaned)  │       │ product_id (FK)      │       │ product_id (FK)     │
│ category        │       │ shop                 │       │ shop                │
│ first_seen      │       │ price                │       │ price               │
│                 │       │ currency             │       │ notified_at         │
│ UNIQUE(name,    │──1:N─▶│ old_price (optional) │       └─────────────────────┘
│   category)     │       │ url                  │               ▲
└─────────────────┘       │ timestamp            │               │
                          └──────────────────────┘      ────1:N──┘

┌─────────────────────┐
│  ExchangeRate       │
├─────────────────────┤
│ currency (PK)       │
│ rate                │
│ fetched_at          │
└─────────────────────┘
```

### 4.2 Table Definitions

#### `products`

| Column | Type | Constraints |
|---|---|---|
| `id` | INTEGER | PK, auto-increment |
| `name` | TEXT | NOT NULL |
| `category` | TEXT | NOT NULL |
| `first_seen` | DATETIME | NOT NULL, default=now |
| | | UNIQUE(`name`, `category`) |

#### `price_history`

| Column | Type | Constraints |
|---|---|---|
| `id` | INTEGER | PK, auto-increment |
| `product_id` | INTEGER | FK → products.id |
| `shop` | TEXT | NOT NULL |
| `price` | REAL | NOT NULL (in original currency) |
| `currency` | TEXT | NOT NULL |
| `old_price` | REAL | NULL (shop's advertised "was" price) |
| `url` | TEXT | NOT NULL |
| `timestamp` | DATETIME | NOT NULL, default=now |

#### `deal_notifications`

| Column | Type | Constraints |
|---|---|---|
| `id` | INTEGER | PK, auto-increment |
| `product_id` | INTEGER | FK → products.id |
| `shop` | TEXT | NOT NULL |
| `price` | REAL | NOT NULL |
| `notified_at` | DATETIME | NOT NULL, default=now |

#### `exchange_rates`

| Column | Type | Constraints |
|---|---|---|
| `currency` | TEXT | PK |
| `rate` | REAL | NOT NULL (rate to base currency CHF) |
| `fetched_at` | DATETIME | NOT NULL |

---

## 5. Deal Evaluation Logic

### 5.1 Decision Flowchart

```
              Parse product from shop listing
                       │
                       ▼
            Two-stage clean (shop → category)
                       │
                       ▼
            Parse price string → float64
                       │
                       ▼
            Normalize price to CHF
                       │
                       ▼
          ┌─ Category rule exists? ──No──▶ SKIP (log warning)
          │            │
          │           Yes
          │            ▼
          │   Sanity check: price in
          │   [min * 0.1, max * 10]? ──No──▶ REJECT (log warning)
          │            │
          │           Yes
          │            ▼
          │   Store product + price history
          │            │
          │            ▼
          │   Price in [min_price, max_price]? ──No──▶ No deal
          │            │
          │           Yes
          │            ▼
          │   First time seeing product+shop? ──Yes──▶ DEAL ✓
          │            │
          │            No
          │            ▼
          │   Price drop ≥ min_discount_pct
          │   from last DB price? ──No──▶ No deal
          │            │
          │           Yes
          │            ▼
          │   Notified within cooldown? ──Yes──▶ SKIP (dedup)
          │            │
          │            No
          │            ▼
          └────────── DEAL ✓ → Post to Telegram
```

### 5.2 Discount Calculation

```
discount_pct = ((last_db_price - current_price) / last_db_price) * 100
```

- `last_db_price` = most recent `price_history` entry for this product+shop.
- Only positive discounts (price drops) qualify.
- The shop's own `old_price` is stored and displayed in notifications but does **not** drive the deal decision.

### 5.3 Sanity Bounds

Any extracted price outside `[min_price * 0.1, max_price * 10]` for its category is rejected as a likely parse error. Logged as a warning, product skipped for this run.

---

## 6. Configuration

### 6.1 `config/settings.yaml`

```yaml
# Base currency for price normalization
base_currency: "CHF"

# Notification cooldown — don't re-post same product+shop within this window
notification_cooldown_hours: 24

# Fetch settings
fetch_delay_seconds: 2
max_retries: 3

# Concurrency
max_concurrent_shops: 5

# Exchange rate
exchange_rate_provider: "https://api.frankfurter.app"
exchange_rate_cache_ttl_hours: 24

# Logging
log_level: "INFO"       # DEBUG, INFO, WARNING, ERROR
log_format: "text"      # text or json

# Database
database_path: "data/dealscout.db"

# Data retention
price_history_retention_days: 90

# Pagination default (can be overridden per shop category)
default_max_pages: 5

# Telegram topics (secrets in secrets.yaml or env vars)
telegram_topics:
  smartphone: 0          # message_thread_id (0 = General topic)
  laptop: 0
  headphones: 0
```

### 6.2 `config/shops.yaml`

```yaml
shops:
  - name: "Galaxus"
    source_type: "json"
    method: "POST"
    cleaner: "galaxus"
    headers:                             # optional, per-shop custom HTTP headers
      content-type: "application/json"
      apollo-require-preflight: "true"
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

      - category: "laptop"
        url: "https://www.galaxus.ch/api/graphql/product-type-filter-products"
        body_template: "templates/galaxus_laptop.json"
        max_pages: 3
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

  - name: "Amazon"
    source_type: "html"
    cleaner: "amazon"
    base_url: "https://www.amazon.de"    # optional, auto-derived from URL if omitted
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
```

### 6.3 `config/deal_rules.yaml`

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

### 6.4 `config/filters.yaml`

```yaml
smartphone:
  skip_brands:
    - Doro
    - Emporia
    - Crosscall
    - Beafon
    - Brondi
    - Swisstone
    # ... (full list ported from previous project)
  exclusion_regex: "(?i)Cover|Hülle|Armband|Ladegerät|Schutzfolie|SmartTag|Smartwatch"

laptop:
  skip_brands: []
  exclusion_regex: "(?i)Tasche|Sleeve|Adapter"

headphones:
  skip_brands: []
  exclusion_regex: "(?i)Ersatzpolster|Kabel|Etui"
```

### 6.5 `config/secrets.yaml`

Gitignored. Contains sensitive credentials. Environment variables take priority.

```yaml
telegram_bot_token: "123456:ABC-DEF..."
telegram_channel: "-1001234567890"
```

**Priority chain:** `TELEGRAM_BOT_TOKEN` env var → `secrets.yaml` → fatal error if neither set.

A `config/secrets.yaml.example` is committed for reference:

```yaml
telegram_bot_token: ""
telegram_channel: ""
```

### 6.6 POST Body Templates

Template files live in `config/templates/`. They contain the raw POST body with placeholders for pagination:

**`config/templates/galaxus_smartphone.json`:**
```json
[{
  "operationName": "PRODUCT_TYPE_FILTER_PRODUCTS",
  "variables": {
    "productTypeId": 24,
    "offset": {offset},
    "limit": 24,
    "filters": []
  },
  "query": "query PRODUCT_TYPE_FILTER_PRODUCTS(...) { ... }"
}]
```

Placeholders (`{offset}`, `{page}`) are replaced by the fetcher based on the pagination config.

---

## 7. Telegram Notification Format

### 7.1 Channel Structure

A Telegram **supergroup with Forum Topics enabled**. Each product category maps to a topic via `message_thread_id` in `settings.yaml`.

| Topic | Category |
|---|---|
| 📱 Smartphones | `smartphone` |
| 💻 Laptops | `laptop` |
| 🎧 Headphones | `headphones` |

### 7.2 Message Format

Notifications are sent via `sendPhoto` with a MarkdownV2 caption to the category's topic. Falls back to `sendMessage` if no image URL is available or image fetch fails.

```
┌─────────────────────────┐
│  [Product Image]        │
│                         │
│  🔥 Samsung Galaxy A15  │
│  💰 CHF 119.00          │
│  📉 -14% (was CHF 139)  │
│  🏷️ Shop says: CHF 159  │
│  🏪 Galaxus             │
│  🔗 View Product        │
└─────────────────────────┘
```

- **Price**: current price in base currency (CHF).
- **Tracked drop**: percentage and absolute from last DB price.
- **Shop says**: the shop's advertised "was" price (display only, if available).
- **1 second delay** between messages to avoid Telegram rate limits.
- Image URLs are passed as-is to Telegram (best effort). If Telegram can't fetch the image, fall back to `sendMessage`.

---

## 8. External Dependencies

### 8.1 Go Modules

| Dependency | Purpose |
|---|---|
| `modernc.org/sqlite` | Pure Go SQLite driver (no CGO) |
| `gopkg.in/yaml.v3` | YAML config parsing |
| `github.com/PuerkitoBio/goquery` | HTML parsing with CSS selectors |

All other functionality uses Go stdlib: `net/http`, `encoding/json`, `database/sql`, `log/slog`, `time`, `os`, `regexp`, `sync`, `flag`.

Zero CGO. Fully static binary. Cross-compiles to any OS.

### 8.2 External Services

| Service | Purpose |
|---|---|
| Shop websites / APIs | Product listing data |
| `frankfurter.app` (configurable) | Exchange rate API (free, no key) |
| Telegram Bot API | Deal notifications via Forum Topics |

---

## 9. Resilience & Error Handling

| Category | Behavior |
|---|---|
| **Fatal errors** | Config parse failure, DB can't open, Telegram credentials missing → **exit immediately** |
| **Shop fetch failure** | Log warning, **skip shop**, continue with others |
| **Parse error** | Log warning, **skip product**, continue |
| **Price parse failure** | Log warning, **skip product** |
| **Sanity check failure** | Price outside `[min * 0.1, max * 10]` → log warning, **skip product** |
| **Currency conversion failure** | Log warning, **skip product** (never post unconverted prices) |
| **Telegram send failure** | Log error, **skip notification**, continue with next deal |
| **Telegram image failure** | Fall back to `sendMessage` without image |
| **HTTP 429 / 5xx** | Exponential backoff retries up to `max_retries` |

---

## 10. Concurrency Model

Shops are fetched concurrently using a worker pool:

- `max_concurrent_shops` goroutines (configurable, default 5).
- Each goroutine processes one shop at a time (all categories, all pages).
- Pages within a shop are fetched **sequentially** with `fetch_delay_seconds` between requests.
- SQLite is opened in **WAL mode** to support concurrent writes from multiple goroutines.
- Deals are collected into a shared slice (mutex-protected) and sent to Telegram **sequentially** after all shops complete.

---

## 11. Deployment

### 11.1 Build

```bash
go build -o dealscout ./cmd/dealscout/
```

Produces a single static binary. No CGO, no runtime dependencies.

### 11.2 Run

```bash
# Full run
./dealscout --config /path/to/config/

# Initial DB population (no notifications)
./dealscout --config /path/to/config/ --seed

# Test run (log deals, don't notify)
./dealscout --config /path/to/config/ --dry-run

# Single shop
./dealscout --config /path/to/config/ --shop "Galaxus"
```

### 11.3 Cron Setup

```cron
0 * * * * /opt/dealscout/dealscout --config /opt/dealscout/config/ >> /var/log/dealscout.log 2>&1
```

Runs every hour on the hour.

### 11.4 Cross-Compilation

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o dealscout-linux ./cmd/dealscout/

# Linux ARM64 (Raspberry Pi, etc.)
GOOS=linux GOARCH=arm64 go build -o dealscout-arm64 ./cmd/dealscout/
```

---

## 12. Testing Strategy

| Test Area | Approach |
|---|---|
| Deal evaluation | Rule logic, sanity bounds, first-seen, discount calculation, edge cases |
| HTML parser | CSS selector extraction with fixture HTML files |
| JSON parser | Field path extraction with fixture JSON files |
| Price parser | All European formats: `CHF 119.–`, `€ 99,90`, `1'299.00`, etc. |
| Cleaners | Shop + category cleaners with known input/output pairs |
| Currency | Conversion accuracy, cache hit/miss, API failure handling |
| Telegram | Message formatting, MarkdownV2 escaping |
| Config | YAML loading, validation, defaults, path resolution |
| URL resolution | Relative URLs, base URL override |

All tests use Go's built-in `testing` package. Fixture HTML/JSON files live alongside `_test.go` files. External services (HTTP, Telegram) are mocked via `httptest`.

Run: `go test ./...`

---

## 13. Future Considerations

- **Cross-shop price comparison** — notify when a product is cheapest across all tracked shops. DB schema already supports this.
- **Headless browser** — Playwright/Rod for JavaScript-heavy shops that don't work with HTTP fetch.
- **Multi-channel notifications** — Discord, Slack, email via additional notifier modules.
- **Web dashboard** — lightweight UI to browse tracked products and price history.
- **Price trend charts** — generate sparkline images for Telegram messages.

---

## 14. Decision Log

| # | Topic | Decision |
|---|---|---|
| 1 | Runtime LLM | None — LLM is dev-time only |
| 2 | Language | Go (from scratch, previous repo as reference) |
| 3 | Hardware requirement | Any (no GPU needed) |
| 4 | Parser architecture | Declarative YAML + optional cleaners |
| 5 | Cleaning pipeline | Two-stage: shop cleaner → category cleaner |
| 6 | Skip lists | `config/filters.yaml`, per category |
| 7 | Data sources | HTML (CSS selectors) + JSON (field paths) |
| 8 | Listing pages | Multiple products per URL, paginated |
| 9 | Pagination | `max_pages` per shop category, global default |
| 10 | Price filtering | Server-side via pre-configured URL params |
| 11 | Product identity | Cleaned name + category (shop-agnostic) |
| 12 | Cross-shop tracking | One Product, PriceHistory per shop |
| 13 | Deal trigger | Per-shop price drop from DB history |
| 14 | First-seen products | Notify if in price range |
| 15 | Discount source | DB history drives logic; shop's "was" price displayed only |
| 16 | Semantic validation | Sanity bounds `[min_price * 0.1, max_price * 10]` |
| 17 | Base currency | CHF |
| 18 | Exchange rates | Live API, SQLite-backed 24h cache |
| 19 | Telegram method | `sendPhoto` + `sendMessage` fallback |
| 20 | Telegram rate limit | 1s delay between messages |
| 21 | Telegram organization | Forum Topics per category |
| 22 | Database | SQLite (pure Go, no CGO), configurable path, WAL mode |
| 23 | Scheduler | System cron, run once and exit |
| 24 | Logging | `log/slog` (stdlib), structured, with run summary |
| 25 | Docker | Not needed — single static binary |
| 26 | Credentials | `secrets.yaml` (gitignored) + env var override |
| 27 | Dependencies | 3: modernc.org/sqlite, yaml.v3, goquery |
| 28 | Error handling | Fatal → exit; shop failures → log + skip |
| 29 | Testing | `_test.go` + fixture files |
| 30 | CSS selectors | goquery (replaces raw x/net/html) |
| 31 | SQLite driver | modernc.org/sqlite (pure Go, no CGO) |
| 32 | GraphQL POST bodies | Template files with placeholders |
| 33 | Exchange rate cache | SQLite table, not in-memory |
| 34 | Shop config structure | Nested: shop → categories |
| 35 | Price parsing | Shared utility for all European formats |
| 36 | URL resolution | Auto-derive from listing URL + optional `base_url` override |
| 37 | DB retention | 90 days default, configurable, pruned each run |
| 38 | First-run flood | `--seed` flag for initial DB population without notifications |
| 39 | CLI flags | `--config`, `--seed`, `--dry-run`, `--shop` |
| 40 | Concurrency | Worker pool per shop, configurable `max_concurrent_shops` |
| 41 | Image handling | Best effort URL pass-through, fallback to sendMessage |
| 42 | Custom HTTP headers | Per-shop `headers` map in `shops.yaml` for CSRF tokens, API keys, etc. |
| 43 | Database directory | Auto-created by `storage.Open` if it does not exist |
| 44 | Debug logging | Fetcher logs request/response at DEBUG level (method, URL, headers, body preview) |
| 45 | Embedded JSON | `json_selector` extracts JSON from HTML `<script>` tags before parsing with field paths |
| 46 | Price divisor | `price_divisor` on shop category divides parsed prices (for APIs returning cents) |
| 47 | Composite title | `title_prefix` field path prepended to `title` (e.g., brand + model) |
| 48 | Map products | JSON parser handles `map[string]object` in addition to arrays at the products path |
| 49 | Two-step price API | `price_api` config enriches products with prices from a secondary API call |
| 50 | Optional price field | JSON parser treats price as optional (defaults to 0) for search-only APIs |
| 51 | POST body detection | `fetchPage` sends POST when `body_template` is set, regardless of pagination type |
| 52 | Foletti shop | HTML parsing with `page_param` pagination, spec-stripping cleaner |
| 53 | Array index in field paths | JSON `walkPath` supports `prices.0.finalPrice` for array element access |
| 54 | GET request headers | `Fetcher.Get` accepts optional headers; `fetchPage` passes shop headers for all request types |
| 55 | Mediamarkt shop | HTML parsing with `data-test` attribute selectors, paginated listing |
| 56 | JSONP callback stripping | `jsonp_callback` config strips JSONP wrapper (e.g., `X({...});` → `{...}`) before JSON parsing |
| 57 | Orderflow shop | HTML parsing (same platform as Foletti), expanded accessory exclusion filter |
| 58 | Response dump | Each shop fetch is saved to `dump_dir/<shop>/<category>.<page>.txt` with curl command header; cleared per shop on each run |
| 59 | Response cache | File-based TTL cache (`cache_dir`, default 25 min) skips HTTP fetches when fresh cached response exists |
| 60 | Dynamic price placeholders | `{min_price}`, `{max_price}` in URLs and templates resolved from deal rules; `{base64_start}...{base64_end}` for encoded filters |
| 61 | Data directory at root | `data/` lives at project root (CWD), not under config dir |
| 62 | URL resolution after enrichment | Product URLs resolved after price enrichment to avoid corrupting IDs used by secondary APIs |
| 63 | URL template | `url_template` with `{id}` placeholder for shops with custom product URL patterns (e.g., Conrad, Interdiscount, Alltron) |
| 64 | GET price enrichment | `price_api` supports GET with `{ids}` placeholder for comma-joined product IDs (e.g., Alltron tiles API) |
| 65 | Price API title/image | `title_path` and `image_path` on `price_api` enrich products with title and image from secondary API |
| 66 | Optional title in JSON | JSON parser allows empty titles for SKU-only search APIs enriched by a secondary API |
| 67 | Multi-URL categories | `urls` list on ShopCategory merges products from multiple URL sources into one category |
