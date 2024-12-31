package telegram

import (
	"coinpaprika-telegram-bot/internal/commands"
	"coinpaprika-telegram-bot/lib/translation"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strings"
)

// NewBot creates new telegram bot
func NewBot(c BotConfig) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(c.Token)
	if err != nil {
		return nil, errors.Wrap(err, "could not create telegram bot")
	}

	bot.Debug = c.Debug

	return &Bot{
		Bot:    bot,
		Config: c,
	}, nil
}

// GetUpdatesChannel gets new updates updates
func (b *Bot) GetUpdatesChannel() (tgbotapi.UpdatesChannel, error) {
	updatesConfig := tgbotapi.NewUpdate(0)
	if b.Config.UpdatesTimeout > 0 {
		updatesConfig.Timeout = b.Config.UpdatesTimeout
	}
	return b.Bot.GetUpdatesChan(updatesConfig), nil
}

// SendMessage sends a telegram message
func (b *Bot) SendMessage(m Message) error {
	msg := tgbotapi.NewMessage(int64(m.ChatID), m.Text)
	msg.ReplyToMessageID = m.MessageID
	msg.DisableWebPagePreview = true
	msg.ParseMode = "MarkdownV2"
	_, err := b.Bot.Send(msg)
	return errors.Wrapf(err, "could not send message: %v", m)
}

func ParseArguments(args string) (string, string) {
	// Use regex to separate coin symbol and time (e.g., "btc 4h")
	re := regexp.MustCompile(`^(\S+)\s*(\d+[hHdD]*)?$`)
	matches := re.FindStringSubmatch(args)

	if len(matches) >= 2 {
		coin := matches[1]
		time := ""
		if len(matches) == 3 {
			time = matches[2]
		}
		return coin, time
	}
	return args, ""
}

// HandleUpdate processes Telegram updates
func (b *Bot) HandleUpdate(u tgbotapi.Update) string {
	text := translation.Translate("Command help message")
	log.Debugf("received command: %s", u.Message.Command())

	var err error = nil

	// Handle commands starting with /
	switch u.Message.Command() {
	case "source":
		text = "https://github\\.com/coinpaprika/telegram\\-bot\\-v2"
	case "p":
		if text, err = commands.CommandPrice(u.Message.CommandArguments()); err != nil {
			text = translation.Translate("Coin not found")
			log.Error(err)
		}
	case "s":
		if text, err = commands.CommandSupply(u.Message.CommandArguments()); err != nil {
			text = translation.Translate("Coin not found")
			log.Error(err)
		}
	case "v":
		if text, err = commands.CommandVolume(u.Message.CommandArguments()); err != nil {
			text = translation.Translate("Coin not found")
			log.Error(err)
		}
	case "c":
		coin, timeRange := ParseArguments(u.Message.CommandArguments())
		chartData, caption, err := commands.CommandChart(coin, timeRange)
		if err != nil {
			text = translation.Translate("Coin not found")
			log.Error(err)
		} else {
			if chartData != nil {
				photo := tgbotapi.NewPhoto(u.Message.Chat.ID, tgbotapi.FileBytes{
					Name:  "chart.png",
					Bytes: chartData,
				})
				photo.Caption = caption
				photo.ParseMode = "MarkdownV2"
				photo.ReplyToMessageID = u.Message.MessageID
				_, err = b.Bot.Send(photo)
				if err != nil {
					log.Error("error sending chart:", err)
				}
				return ""
			} else {
				text = caption
			}
		}
	}

	// Handle $ commands
	if u.Message.Text != "" && u.Message.Text[0] == '$' {
		rawArgs := strings.TrimSpace(u.Message.Text[1:])
		coin, timeRange := ParseArguments(rawArgs)

		chartData, caption, err := commands.CommandChartWithTicker(coin, timeRange)
		if err != nil {
			text = translation.Translate("Coin not found")
			log.Error(err)
		} else {
			if chartData != nil {
				photo := tgbotapi.NewPhoto(u.Message.Chat.ID, tgbotapi.FileBytes{
					Name:  "chart.png",
					Bytes: chartData,
				})
				photo.Caption = caption
				photo.ParseMode = "MarkdownV2"
				photo.ReplyToMessageID = u.Message.MessageID
				_, err = b.Bot.Send(photo)
				if err != nil {
					log.Error("error sending chart:", err)
				}
				return ""
			} else {
				text = caption
			}
		}
	}

	return text
}
