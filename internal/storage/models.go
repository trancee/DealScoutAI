package storage

import "fmt"

// UpsertProduct inserts a product or returns the existing one.
// Returns (id, isNew, error).
func (d *Database) UpsertProduct(name, category string) (int64, bool, error) {
	var id int64
	err := d.db.QueryRow(
		"SELECT id FROM products WHERE name = ? AND category = ?",
		name, category,
	).Scan(&id)

	if err == nil {
		return id, false, nil
	}

	result, err := d.db.Exec(
		"INSERT INTO products (name, category) VALUES (?, ?)",
		name, category,
	)
	if err != nil {
		return 0, false, fmt.Errorf("inserting product: %w", err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		return 0, false, fmt.Errorf("getting last insert id: %w", err)
	}

	return id, true, nil
}

// AppendPriceHistory adds a price record for a product at a shop.
func (d *Database) AppendPriceHistory(productID int64, shop string, price float64, currency string, oldPrice *float64, url string) error {
	_, err := d.db.Exec(
		"INSERT INTO price_history (product_id, shop, price, currency, old_price, url) VALUES (?, ?, ?, ?, ?, ?)",
		productID, shop, price, currency, oldPrice, url,
	)
	if err != nil {
		return fmt.Errorf("appending price history: %w", err)
	}
	return nil
}

// LastPrice returns the most recent price for a product+shop.
// Returns (price, found, error).
func (d *Database) LastPrice(productID int64, shop string) (float64, bool, error) {
	var price float64
	err := d.db.QueryRow(
		"SELECT price FROM price_history WHERE product_id = ? AND shop = ? ORDER BY id DESC LIMIT 1",
		productID, shop,
	).Scan(&price)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("querying last price: %w", err)
	}
	return price, true, nil
}

// RecordNotification records that a deal notification was sent.
func (d *Database) RecordNotification(productID int64, shop string, price float64) error {
	_, err := d.db.Exec(
		"INSERT INTO deal_notifications (product_id, shop, price) VALUES (?, ?, ?)",
		productID, shop, price,
	)
	if err != nil {
		return fmt.Errorf("recording notification: %w", err)
	}
	return nil
}

// WasNotifiedWithinCooldown checks if a notification was sent for product+shop within the cooldown window.
func (d *Database) WasNotifiedWithinCooldown(productID int64, shop string, cooldownHours int) (bool, error) {
	var count int
	err := d.db.QueryRow(
		"SELECT COUNT(*) FROM deal_notifications WHERE product_id = ? AND shop = ? AND notified_at > datetime('now', ? || ' hours')",
		productID, shop, -cooldownHours,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking cooldown: %w", err)
	}
	return count > 0, nil
}

// UpsertExchangeRate stores or updates an exchange rate.
func (d *Database) UpsertExchangeRate(currency string, rate float64) error {
	_, err := d.db.Exec(
		`INSERT INTO exchange_rates (currency, rate, fetched_at) VALUES (?, ?, datetime('now'))
		 ON CONFLICT(currency) DO UPDATE SET rate = excluded.rate, fetched_at = excluded.fetched_at`,
		currency, rate,
	)
	if err != nil {
		return fmt.Errorf("upserting exchange rate: %w", err)
	}
	return nil
}

// ExchangeRate returns the cached rate for a currency if it's within maxAgeHours.
// Returns (rate, fresh, error).
func (d *Database) ExchangeRate(currency string, maxAgeHours int) (float64, bool, error) {
	var rate float64
	err := d.db.QueryRow(
		"SELECT rate FROM exchange_rates WHERE currency = ? AND fetched_at > datetime('now', ? || ' hours')",
		currency, -maxAgeHours,
	).Scan(&rate)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("querying exchange rate: %w", err)
	}
	return rate, true, nil
}

// PruneOldPriceHistory deletes price history older than retentionDays.
// Returns the number of rows deleted.
func (d *Database) PruneOldPriceHistory(retentionDays int) (int64, error) {
	result, err := d.db.Exec(
		"DELETE FROM price_history WHERE timestamp < datetime('now', ? || ' days')",
		-retentionDays,
	)
	if err != nil {
		return 0, fmt.Errorf("pruning price history: %w", err)
	}
	return result.RowsAffected()
}
