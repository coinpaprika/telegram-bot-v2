package telegram

import (
	"coinpaprika-telegram-bot/internal/commands"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
	text := fmt.Sprintf("Please use one of the commands:\n\n" +
		"/start or /help \tshow this message\n" +
		"/o <symbol> \t\tcheck the coin overview\n" +
		"/p <symbol> \t\tcheck the coin price\n" +
		"/s <symbol> \t\tcheck the circulating supply\n" +
		"/v <symbol> \t\tcheck the 24h volume\n" +
		"/c <symbol> \t\tget the price chart\n\n" +
		"/source \t\tshow source code of this bot\n")

	log.Debugf("received command: %s", u.Message.Command())

	var err error = nil
	switch u.Message.Command() {
	case "source":
		text = "https://github.com/coinpaprika/telegram-bot-v2"
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
	case "o":
		chartData, caption, err := commands.CommandChartWithTicker(u.Message.CommandArguments())
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
