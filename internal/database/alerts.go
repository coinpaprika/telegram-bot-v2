package database

import (
	"coinpaprika-telegram-bot/internal/types"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

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

// GetAlertsByChatID fetches all alerts for a specific chat ID
func GetAlertsByChatID(chatID int64) ([]types.Alert, error) {
	query := `SELECT id, ticker, value, alert_type, current_price, created_at FROM alerts WHERE chat_id = ?;`

	rows, err := DB.Query(query, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts for chat ID %d: %w", chatID, err)
	}
	defer rows.Close()

	var alerts []types.Alert
	for rows.Next() {
		var alert types.Alert
		if err := rows.Scan(&alert.ID, &alert.Ticker, &alert.Target, &alert.AlertType, &alert.CurrentPrice, &alert.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}
