package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// BotConfig configuration of the bot
type BotConfig struct {
	Token          string
	Debug          bool
	UpdatesTimeout int
}

// Bot telegram interaction client
type Bot struct {
	Bot    *tgbotapi.BotAPI
	Config BotConfig
}

// Message a telegram message struct
type Message struct {
	ChatID    int
	MessageID int
	Text      string
}
