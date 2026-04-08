You are an expert Python software engineer and AI developer.  
Your task is to build a **full Python project** called DealScoutAI: an autonomous AI agent that monitors product prices across multiple online shops and posts **category-specific deals** to a Telegram channel. The project must be **ready-to-run**, modular, and production-ready.

> **Key design principles:** deduplication of notifications, structured logging, LLM response validation, rate-limited fetching, and currency normalization.

---

### Project Requirements

1. **Architecture**
   - Fully autonomous AI agent approach:
     - Fetch product pages from shops (HTTP requests, with rate limiting)
     - Pre-process HTML (strip scripts, nav, footer) to reduce token usage
     - Send cleaned HTML to LLM for structured extraction
     - Validate LLM response against a Pydantic schema
     - Normalize currency to a single base currency (configurable)
     - Compare prices in database
     - Detect **deals using category-specific rules** (price range + price-drop %)
     - Deduplicate: skip deals already posted within a cooldown period
     - Generate a formatted message for Telegram
   - Scheduler to run every configurable interval
   - Configurable shop list in YAML or JSON (with schema example)
   - Modular code: agents/, fetcher/, notifiers/, storage/, scheduler/, config/, utils/

2. **Category-Specific Deal Rules**
   - Each product category can define what counts as a deal.
   - A deal requires **both** a valid price range **and** a minimum price-drop percentage from the last known price.
   - Example configuration in `config/deal_rules.yaml`:
     ```yaml
     smartphone:
       min_price: 50
       max_price: 150
       min_discount_pct: 10   # only post if ≥10% drop from last known price
     laptop:
       min_price: 400
       max_price: 1200
       min_discount_pct: 15
     headphones:
       min_price: 20
       max_price: 200
       min_discount_pct: 5
     ```
   - The AI agent must check the extracted product price against the category rule (range + discount %) before posting to Telegram.
   - If a product is seen for the first time (no price history), treat it as a deal if it falls within the price range.

3. **Python Modules**
   - `agents/deal_agent.py`: core LLM logic, input HTML → output JSON (with category detection)
   - `agents/html_preprocessor.py`: strip scripts, styles, nav, footer from raw HTML to reduce token usage
   - `agents/schemas.py`: Pydantic models for validating LLM extraction output
   - `fetcher/fetch.py`: lightweight URL fetcher (with headers, retries, rate limiting, User-Agent rotation)
   - `notifiers/telegram.py`: send messages to Telegram channel via `sendMessage`
   - `storage/database.py`: database connection + ORM (SQLite by default)
   - `storage/models.py`: Product, PriceHistory & DealNotification models
   - `scheduler/run_checks.py`: orchestrates fetching, AI extraction, DB, category-rule check, deduplication, Telegram
   - `config/shops.yaml`: list of shops + product categories (see schema below)
   - `config/deal_rules.yaml`: category-specific deal rules
   - `config/settings.yaml`: global settings (base currency, cooldown, log level)
   - `utils/logger.py`: structured logging setup (stdlib `logging` or `loguru`)
   - `utils/currency.py`: currency normalization to a configurable base currency
   - `main.py`: CLI entry point to start the agent
   - `requirements.txt`: all dependencies

4. **Telegram Integration**
   - Uses Bot API (`sendMessage` with MarkdownV2 formatting)
   - Reads BOT_TOKEN and CHANNEL from `.env`
   - Posts clean, Markdown-formatted messages with product title, price, original price, discount %, product link, and shop name
   - Images are included as inline links (not `sendPhoto`)
   - Handles Telegram API rate limits and errors gracefully

5. **Database**
   - SQLite (default) or PostgreSQL (optional)
   - Tables:
     - Products: id, title, url, shop, category, current_price, currency, last_checked
     - PriceHistory: id, product_id, price, currency, timestamp
     - DealNotifications: id, product_id, price, notified_at (for deduplication)
   - Deduplication: do not re-post a deal for the same product within a configurable cooldown period (default: 24 hours)

6. **Scheduler**
   - Use APScheduler or cron
   - Configurable interval

7. **Currency Normalization**
   - All prices are normalized to a configurable **base currency** (e.g., `EUR`) before deal evaluation
   - `utils/currency.py` handles conversion using a free exchange rate API (e.g., `exchangerate.host` or `frankfurter.app`)
   - Exchange rates are cached for a configurable TTL (default: 1 hour) to avoid excessive API calls
   - If conversion fails, log a warning and skip the product (do not post unconverted prices)

8. **Fetcher Resilience**
   - Configurable delay between requests (default: 2 seconds)
   - User-Agent rotation from a predefined list
   - Optional proxy support (list of proxies in `config/settings.yaml`)
   - Respect `robots.txt` (optional, configurable)
   - Retry with exponential backoff on transient failures (HTTP 429, 5xx)

9. **Logging & Monitoring**
   - Structured logging via `utils/logger.py` (JSON or human-readable, configurable)
   - Log levels: DEBUG, INFO, WARNING, ERROR
   - Key events logged: fetch success/failure, LLM extraction result, deal detected, Telegram post sent/failed
   - Basic stats printed on each scheduler run (products checked, deals found, notifications sent)

10. **Docker**
    - Dockerfile and docker-compose.yml to run the app easily
    - Volume mount for SQLite database persistence

11. **Project Structure**
    ```
    DealScoutAI/
    ├── agents/
    │   ├── __init__.py
    │   ├── deal_agent.py
    │   ├── html_preprocessor.py
    │   └── schemas.py
    ├── fetcher/
    │   ├── __init__.py
    │   └── fetch.py
    ├── notifiers/
    │   ├── __init__.py
    │   └── telegram.py
    ├── storage/
    │   ├── __init__.py
    │   ├── database.py
    │   └── models.py
    ├── scheduler/
    │   ├── __init__.py
    │   └── run_checks.py
    ├── config/
    │   ├── shops.yaml
    │   ├── deal_rules.yaml
    │   └── settings.yaml
    ├── utils/
    │   ├── __init__.py
    │   ├── logger.py
    │   └── currency.py
    ├── tests/
    │   ├── test_deal_rules.py
    │   ├── test_schemas.py
    │   ├── test_currency.py
    │   └── test_telegram_format.py
    ├── main.py
    ├── README.md
    ├── requirements.txt
    ├── docker-compose.yml
    └── .env.example
    ```

12. **Shops Configuration Schema**
    Example `config/shops.yaml`:
    ```yaml
    shops:
      - name: "Amazon"
        base_url: "https://amazon.com"
        product_urls:
          - url: "https://amazon.com/dp/B0..."
            category: "smartphone"
          - url: "https://amazon.com/dp/B0..."
            category: "laptop"
        fetch_interval_minutes: 60

      - name: "MediaMarkt"
        base_url: "https://mediamarkt.com"
        product_urls:
          - url: "https://mediamarkt.com/product/123"
            category: "headphones"
        fetch_interval_minutes: 120
    ```

13. **Global Settings Schema**
    Example `config/settings.yaml`:
    ```yaml
    base_currency: "EUR"
    notification_cooldown_hours: 24
    fetch_delay_seconds: 2
    log_level: "INFO"
    log_format: "human"  # or "json"
    exchange_rate_cache_ttl_minutes: 60
    proxies: []  # optional list of proxy URLs
    ```

14. **Testing**
    - Use `pytest` as the test framework
    - Include unit tests for:
      - Deal rule evaluation (price range + discount % logic)
      - LLM response validation (Pydantic schema parsing, malformed input handling)
      - Currency normalization (conversion, caching, failure handling)
      - Telegram message formatting
    - Mock all external dependencies (HTTP, LLM API, Telegram API) in tests
    - Tests must be runnable with `pytest tests/`

15. **Features**
    - Autonomous price monitoring using LLM
    - Category-specific deal detection (price range + discount %)
    - Price-drop tracking with historical comparison
    - Currency normalization across shops
    - Deduplication of Telegram notifications
    - Telegram notifications via `sendMessage`
    - Configurable shops, categories, and global settings
    - Structured logging
    - Rate-limited, resilient fetching
    - Modular, maintainable, and production-ready

16. **Output Requirements**
    - Provide the **full code** for each module
    - Include `README.md` with installation and usage instructions
    - Include `requirements.txt` and `.env.example`
    - Include `config/shops.yaml`, `config/deal_rules.yaml`, and `config/settings.yaml` templates
    - Include `main.py` as the CLI entry point (e.g., `python main.py` to start)
    - Include Dockerfile and docker-compose.yml
    - Include `tests/` with pytest unit tests

17. **Constraints**
    - Use Python 3.10+
    - LLM interaction should be modular (so the model can be swapped)
    - All URLs and shop pages should be fetched programmatically, no manual input
    - HTML must be pre-processed before sending to LLM (strip unnecessary elements to reduce tokens)
    - LLM output must be validated against a Pydantic schema; retry or skip on malformed responses
    - The AI agent should extract product name, price, currency, availability, image URL, and product URL
    - The AI agent must also identify the **product category**
    - All prices must be normalized to the configured base currency before deal evaluation
    - Only post to Telegram if **the price falls within the category-specific deal rules AND meets the discount % threshold**
    - Do not re-post a deal for the same product within the notification cooldown period
    - All Telegram messages should be MarkdownV2 formatted, sent via `sendMessage` (images as inline links)
    - The code should include inline comments explaining each step
    - All modules must use structured logging via `utils/logger.py`
    - Include `__init__.py` in every package directory

---

Generate the full DealScoutAI project code **module by module**, starting from a ready-to-run project scaffold, including all files mentioned above. Make sure the code is well-structured, functional, and production-ready, and that it enforces **category-specific deal rules** (price range + discount %) when posting to Telegram, with proper deduplication, currency normalization, structured logging, and resilient fetching.