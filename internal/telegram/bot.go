package telegram

import (
	"coinpaprika-telegram-bot/internal/commands"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"regexp"
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

func (b *Bot) HandleUpdate(u tgbotapi.Update) string {
	text := `Please use one of the commands:

			/start or /help 	show this message
			/p <symbol> 		check the coin price
			/s <symbol> 		check the circulating supply
			/v <symbol> 		check the 24h volume
			/c <symbol> 		get the price chart

			/source 			show source code of this bot
			`
	log.Debugf("received command: %s", u.Message.Command())

	tickers := extractTickers(u.Message.Text)
	if len(tickers) > 0 {
		ticker := tickers[0]
		text = fmt.Sprintf("You mentioned the ticker: %s", ticker)

		var err error
		chartData, caption, err := commands.CommandChartWithTicker(ticker)
		if err != nil {
			text = "invalid coin name|ticker|symbol, please try again"
			log.Error(err)
		} else {
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

			if err != nil {
				log.Error("error deleting chart file:", err)
			}

			return ""
		}
	}

	var err error = nil
	switch u.Message.Command() {
	case "source":
		text = "https://github.com/coinpaprika/telegram-bot"
	case "p":
		if text, err = commands.CommandPrice(u.Message.CommandArguments()); err != nil {
			text = "invalid coin name|ticker|symbol, please try again"
			log.Error(err)
		}
	case "s":
		if text, err = commands.CommandSupply(u.Message.CommandArguments()); err != nil {
			text = "invalid coin name|ticker|symbol, please try again"
			log.Error(err)
		}
	case "v":
		if text, err = commands.CommandVolume(u.Message.CommandArguments()); err != nil {
			text = "invalid coin name|ticker|symbol, please try again"
			log.Error(err)
		}
	case "c":
		chartData, caption, err := commands.CommandChart(u.Message.CommandArguments())
		if err != nil {
			text = "invalid coin name|ticker|symbol, please try again"
			log.Error(err)
		} else {
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

			if err != nil {
				log.Error("error deleting chart file:", err)
			}

			return ""
		}
	}

	return text
}

func extractTickers(message string) []string {
	re := regexp.MustCompile(`\$(\w+)`)
	matches := re.FindAllStringSubmatch(message, -1)

	var tickers []string
	for _, match := range matches {
		if len(match) > 1 {
			tickers = append(tickers, match[1])
		}
	}
	return tickers
}
