package types

type Alert struct {
	ID           int64   `json:"id"`
	ChatID       int64   `json:"chat_id"`
	Ticker       string  `json:"ticker"`
	Target       float64 `json:"target"`
	CurrentPrice float64 `json:"current_price"`
	AlertType    string  `json:"alert_type"` // e.g., "price, percent_change"
	CreatedAt    string  `json:"created_at"`
}
