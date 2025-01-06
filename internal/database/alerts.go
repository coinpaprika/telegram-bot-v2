package database

import (
	"coinpaprika-telegram-bot/internal/types"
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

// InitDB initializes the SQLite database
func InitDB(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS alerts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER NOT NULL,
		ticker TEXT NOT NULL,
		value TEXT NOT NULL,
		alert_type TEXT NOT NULL,
		current_price TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = DB.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create alerts table: %w", err)
	}

	log.Println("Database initialized successfully.")
	return nil
}

// InsertAlert saves an alert to the database
func InsertAlert(chatID int64, ticker, value, alertType, currentPrice string) error {
	query := `
	INSERT INTO alerts (chat_id, ticker, value, alert_type, current_price)
	VALUES (?, ?, ?, ?, ?);`

	_, err := DB.Exec(query, chatID, ticker, value, alertType, currentPrice)
	if err != nil {
		return fmt.Errorf("failed to insert alert: %w", err)
	}

	log.Printf("Alert inserted successfully: ChatID: %d, Ticker: %s, Value: %s, Type: %s, CurrentPrice: %s", chatID, ticker, value, alertType, currentPrice)
	return nil
}

// GetAllAlerts fetches all alerts from the database
func GetAllAlerts() ([]types.Alert, error) {
	query := `SELECT id, chat_id, ticker, value, alert_type, current_price, created_at FROM alerts;`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts: %w", err)
	}
	defer rows.Close()

	var alerts []types.Alert
	for rows.Next() {
		var alert types.Alert
		if err := rows.Scan(&alert.ID, &alert.ChatID, &alert.Ticker, &alert.Target, &alert.AlertType, &alert.CurrentPrice, &alert.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// DeleteAlert removes a triggered alert from the database
func DeleteAlert(alertID int64) error {
	query := `DELETE FROM alerts WHERE id = ?;`
	_, err := DB.Exec(query, alertID)
	if err != nil {
		return fmt.Errorf("failed to delete alert: %w", err)
	}
	return nil
}
