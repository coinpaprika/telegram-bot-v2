package database

import (
	"database/sql"
	"fmt"
	"log"
)

var DB *sql.DB

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

	createMetricsTable := `
		CREATE TABLE IF NOT EXISTS metrics (
		metric_name TEXT NOT NULL,
		label_key TEXT DEFAULT NULL,
		label_value TEXT DEFAULT NULL,
		metric_value REAL NOT NULL,
		PRIMARY KEY (metric_name, label_key, label_value)
	);`
	_, err = DB.Exec(createMetricsTable)
	if err != nil {
		return fmt.Errorf("failed to create metrics table: %w", err)
	}

	log.Println("Database initialized successfully.")
	return nil
}

func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
