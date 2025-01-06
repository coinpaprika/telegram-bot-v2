package telegram

import (
	"coinpaprika-telegram-bot/internal/commands"
	"coinpaprika-telegram-bot/internal/database"
	"coinpaprika-telegram-bot/internal/price"
	"coinpaprika-telegram-bot/lib/helpers"
	"coinpaprika-telegram-bot/lib/translation"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strconv"
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
	re := regexp.MustCompile(`^(\S+)\s*(.+)?$`)
	matches := re.FindStringSubmatch(args)

	if len(matches) >= 2 {
		ticker := matches[1]
		target := ""
		if len(matches) == 3 {
			target = matches[2]
		}
		return ticker, target
	}
	return "", ""
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
	case "o":
		coin, timeRange := ParseArguments(u.Message.CommandArguments())
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
	case "alert":
		return b.HandleAlertCommand(u)
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

func (b *Bot) HandleCallbackQuery(callbackQuery *tgbotapi.CallbackQuery) {
	data := callbackQuery.Data
	chatID := callbackQuery.Message.Chat.ID
	messageID := callbackQuery.Message.MessageID // Get the MessageID for deletion

	switch {
	case strings.HasPrefix(data, "alert_select"):
		parts := strings.Split(data, "|")
		if len(parts) < 3 {
			b.Bot.Send(tgbotapi.NewCallback(callbackQuery.ID, "❌ Invalid alert data."))
			return
		}

		ticker := parts[1]
		target := parts[2]

		successMsg, err := b.InsertAlert(chatID, ticker, target)
		if err != nil {
			b.Bot.Send(tgbotapi.NewCallback(callbackQuery.ID, "❌ Failed to save alert. Please try again later."))
			return
		}

		// Delete the options message
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
		_, err = b.Bot.Request(deleteMsg)
		if err != nil {
			log.Error("Failed to delete options message: ", err)
		}

		// Send success message
		msg := tgbotapi.NewMessage(chatID, successMsg)
		msg.ParseMode = "MarkdownV2"
		b.Bot.Send(msg)
		b.Bot.Send(tgbotapi.NewCallback(callbackQuery.ID, "✅ Alert saved successfully."))

	case strings.HasPrefix(data, "alert_cancel"):
		parts := strings.Split(data, "|")
		if len(parts) < 3 {
			b.Bot.Send(tgbotapi.NewCallback(callbackQuery.ID, "❌ Invalid alert data."))
			return
		}

		target := parts[2]

		// Delete the options message
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
		_, err := b.Bot.Request(deleteMsg)
		if err != nil {
			log.Error("Failed to delete options message: ", err)
		}

		msg := tgbotapi.NewMessage(chatID, helpers.EscapeMarkdownV2(fmt.Sprintf(
			"❗ Please send the full link of the coin with target price *%s*.",
			target,
		)))
		msg.ParseMode = "MarkdownV2"
		msg.ReplyMarkup = tgbotapi.ForceReply{
			ForceReply:            true,
			Selective:             true,
			InputFieldPlaceholder: "e.g., coinpaprika.com/coin/btc-bitcoin/",
		}

		_, err = b.Bot.Send(msg)
		if err != nil {
			log.Error("Failed to prompt for full link: ", err)
			b.Bot.Send(tgbotapi.NewCallback(callbackQuery.ID, "❌ Failed to prompt for a reply. Please try again."))
			return
		}

		b.Bot.Send(tgbotapi.NewCallback(callbackQuery.ID, "ℹ️ Please reply with the full coin link."))

	default:
		b.Bot.Send(tgbotapi.NewCallback(callbackQuery.ID, "❌ Unknown action. Please try again."))
	}
}

func (b *Bot) HandleReply(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	replyText := strings.TrimSpace(message.Text)

	if strings.Contains(message.ReplyToMessage.Text, "Please send the full link of the coin") {
		reTarget := regexp.MustCompile(`target price \*(.+?)\*`)
		targetMatches := reTarget.FindStringSubmatch(message.ReplyToMessage.Text)
		if len(targetMatches) < 2 {
			b.SendMessage(Message{
				ChatID: int(chatID),
				Text:   "❌ Could not extract the target price. Please start again.",
			})
			return
		}

		target := targetMatches[1]

		reLink := regexp.MustCompile(`https://coinpaprika\.com/coin/([\w-]+)/?`)
		linkMatches := reLink.FindStringSubmatch(replyText)
		if len(linkMatches) < 2 {
			b.SendMessage(Message{
				ChatID: int(chatID),
				Text:   "❌ Invalid link format. Please provide a valid coin link, such as `https://coinpaprika.com/coin/sol-solana/`.",
			})
			return
		}

		ticker := linkMatches[1]

		successMsg, err := b.InsertAlert(chatID, ticker, target)
		if err != nil {
			b.SendMessage(Message{
				ChatID: int(chatID),
				Text:   "❌ Failed to save alert with the provided link. Please try again.",
			})
			return
		}

		b.SendMessage(Message{
			ChatID: int(chatID),
			Text:   successMsg,
		})
	}
}

// HandleAlertCommand handles the /alert command logic
func (b *Bot) HandleAlertCommand(u tgbotapi.Update) string {
	args := u.Message.CommandArguments()
	ticker, target := ParseArguments(args)

	if ticker == "" || target == "" {
		return helpers.EscapeMarkdownV2("Usage: /alert {ticker} {target} (e.g., /alert btc 98000$ or /alert btc 10%)")
	}

	coins, err := commands.SearchCoins(ticker)
	if err != nil {
		log.Error(err)
		return translation.Translate("❌ Failed to search for the coin.")
	}

	msg := tgbotapi.NewMessage(u.Message.Chat.ID, fmt.Sprintf(
		helpers.EscapeMarkdownV2("🔍 Found %d result(s) for %s. Please select one or choose to send a manual link:"),
		len(coins), ticker,
	))
	msg.ParseMode = "MarkdownV2"

	var buttons [][]tgbotapi.InlineKeyboardButton

	for i, coin := range coins {
		if i >= 4 {
			break
		}
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s (%s)", *coin.Name, *coin.Symbol),
				fmt.Sprintf("alert_select|%s|%s", *coin.ID, target),
			),
		))
	}

	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			"❌ None of these, I'll send the link",
			fmt.Sprintf("alert_cancel|%s|%s", ticker, target),
		),
	))

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)

	_, err = b.Bot.Send(msg)
	if err != nil {
		log.Error("Failed to send coin selection buttons: ", err)
	}
	return ""
}

// InsertAlert handles alert insertion logic
func (b *Bot) InsertAlert(chatID int64, ticker string, target string) (string, error) {
	var alertType string
	var alertTypeSymbol string
	if strings.Contains(target, "%") || strings.HasPrefix(target, "-") {
		alertType = "percent"
		alertTypeSymbol = "%"
		target = strings.ReplaceAll(target, "%", "")
	} else {
		alertType = "price"
		alertTypeSymbol = "$"
	}

	cp, exists := price.GetPrice(ticker)
	if !exists {
		return "", errors.New("failed to get current price")
	}

	err := database.InsertAlert(chatID, ticker, target, alertType, strconv.FormatFloat(cp.PriceUSD, 'f', -1, 64))
	if err != nil {
		log.Error("Failed to save alert: ", err)
		return "", errors.Wrap(err, "failed to insert alert into database")
	}

	successMsg := fmt.Sprintf(
		"✅ Alert set for *%s* at *%s%s*",
		helpers.EscapeMarkdownV2(ticker),
		helpers.EscapeMarkdownV2(target),
		alertTypeSymbol,
	)
	return successMsg, nil
}
