package telegram

import (
	"coinpaprika-telegram-bot/internal/commands"
	"coinpaprika-telegram-bot/internal/database"
	"coinpaprika-telegram-bot/internal/price"
	"coinpaprika-telegram-bot/lib/helpers"
	"coinpaprika-telegram-bot/lib/translation"
	"fmt"
	"github.com/coinpaprika/coinpaprika-api-go-client/v2/coinpaprika"
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
		Bot:              bot,
		Config:           c,
		messageTargetMap: make(map[int]string),
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
		args := u.Message.CommandArguments()
		if strings.TrimSpace(args) == "list" {
			text = b.HandleAlertListCommand(u.Message.Chat.ID)
		} else {
			return b.HandleAlertCommand(u)
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

func (b *Bot) HandleCallbackQuery(callbackQuery *tgbotapi.CallbackQuery) {
	data := callbackQuery.Data
	chatID := callbackQuery.Message.Chat.ID
	messageID := callbackQuery.Message.MessageID // Get the MessageID for deletion

	switch {
	case strings.HasPrefix(data, "alert_select"):
		parts := strings.Split(data, "|")
		if len(parts) < 3 {
			b.Bot.Send(tgbotapi.NewCallback(callbackQuery.ID, translation.Translate("Invalid alert data.")))
			return
		}
		target := parts[2]
		ticker, found := price.GetTickerByID(parts[1])
		if !found {
			b.Bot.Send(tgbotapi.NewCallback(callbackQuery.ID, translation.Translate("Invalid alert data.")))
			return
		}

		coin, err := commands.GetCoinByID(ticker)
		if err != nil {
			b.Bot.Send(tgbotapi.NewCallback(callbackQuery.ID, translation.Translate("Invalid alert data.")))
			return
		}

		successMsg, err := b.InsertAlert(chatID, coin, target)
		if err != nil {
			b.Bot.Send(tgbotapi.NewCallback(callbackQuery.ID, translation.Translate("Failed to save alert. Please try again later.")))
			msg := tgbotapi.NewMessage(chatID, err.Error())
			msg.ParseMode = "MarkdownV2"
			b.Bot.Send(msg)
			// Delete the options message
			deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
			_, err = b.Bot.Request(deleteMsg)
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
		b.Bot.Send(tgbotapi.NewCallback(callbackQuery.ID, translation.Translate("Alert saved successfully.")))

	case strings.HasPrefix(data, "alert_cancel"):
		parts := strings.Split(data, "|")
		if len(parts) < 2 {
			b.Bot.Send(tgbotapi.NewCallback(callbackQuery.ID, translation.Translate("Invalid alert data.")))
			return
		}

		target := parts[1]

		// Delete the options message
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
		_, err := b.Bot.Request(deleteMsg)
		if err != nil {
			log.Error("Failed to delete options message: ", err)
		}

		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
			translation.Translate("Please send the full link of the coin with target price %s"),
			helpers.EscapeMarkdownV2(target),
		))
		msg.ParseMode = "MarkdownV2"
		msg.ReplyMarkup = tgbotapi.ForceReply{
			ForceReply:            true,
			Selective:             true,
			InputFieldPlaceholder: translation.Translate("e.g., coinpaprika.com/coin/btc-bitcoin/"),
		}

		m, err := b.Bot.Send(msg)
		if err != nil {
			log.Error("Failed to prompt for full link: ", err)
			b.Bot.Send(tgbotapi.NewCallback(callbackQuery.ID, translation.Translate("Failed to prompt for a reply. Please try again.")))
			return
		}

		b.Bot.Send(tgbotapi.NewCallback(callbackQuery.ID, translation.Translate("Please reply with the full coin link.")))
		b.messageTargetMap[m.MessageID] = target
	default:
		b.Bot.Send(tgbotapi.NewCallback(callbackQuery.ID, translation.Translate("Unknown action. Please try again.")))
	}
}

func (b *Bot) HandleReply(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	replyText := strings.TrimSpace(message.Text)

	// Check if the referenced message exists in the mapping
	if message.ReplyToMessage != nil {
		target, exists := b.messageTargetMap[message.ReplyToMessage.MessageID]
		if !exists {
			b.SendMessage(Message{
				ChatID: int(chatID),
				Text:   translation.Translate("no_context_for_reply"),
			})
			return
		}

		// Validate the link
		reLink := regexp.MustCompile(`https://coinpaprika.com/(coin|waluta|valjuta)/([\w-]+)/?`)
		linkMatches := reLink.FindStringSubmatch(replyText)
		if len(linkMatches) < 3 {
			b.SendMessage(Message{
				ChatID: int(chatID),
				Text:   translation.Translate("invalid_link_format"),
			})
			return
		}

		ticker := linkMatches[2]
		coin, err := commands.GetCoinByID(ticker)
		if err != nil {
			log.Error(err)
			b.SendMessage(Message{
				ChatID: int(chatID),
				Text:   translation.Translate("coin_search_failed"),
			})
			return
		}

		successMsg, err := b.InsertAlert(chatID, coin, target)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, err.Error())
			msg.ParseMode = "MarkdownV2"
			b.Bot.Send(msg)

			return
		}

		b.SendMessage(Message{
			ChatID: int(chatID),
			Text:   successMsg,
		})

		// Remove the mapping after processing the reply
		delete(b.messageTargetMap, message.ReplyToMessage.MessageID)
	} else {
		b.SendMessage(Message{
			ChatID: int(chatID),
			Text:   translation.Translate("no_reply_message"),
		})
	}
}

// HandleAlertCommand handles the /alert command logic
func (b *Bot) HandleAlertCommand(u tgbotapi.Update) string {
	args := u.Message.CommandArguments()
	ticker, target := ParseArguments(args)
	if ticker == "list" {
		b.HandleAlertListCommand(u.Message.Chat.ID)
		return ""
	}

	if ticker == "" || target == "" {
		return helpers.EscapeMarkdownV2(translation.Translate("alert_command_usage"))
	}

	coins, err := commands.SearchCoins(ticker)
	if err != nil {
		log.Error(err)
		return translation.Translate("coin_search_failed")
	}

	var buttons [][]tgbotapi.InlineKeyboardButton

	var counter int
	for i, coin := range coins {
		if i >= 4 {
			break
		}
		p, found := price.GetPrice(*coin.ID)

		if !found {
			continue
		}
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf(translation.Translate("coin_display_format"), *coin.Name, *coin.Symbol),
				fmt.Sprintf("alert_select|%d|%s", p.ID, target),
			),
		))
		counter++
	}

	msg := tgbotapi.NewMessage(u.Message.Chat.ID, fmt.Sprintf(
		helpers.EscapeMarkdownV2(translation.Translate("coin_search_results")),
		counter, ticker,
	))
	msg.ParseMode = "MarkdownV2"

	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			translation.Translate("alert_manual_link_option"),
			fmt.Sprintf("alert_cancel|%s", target),
		),
	))

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)

	_, err = b.Bot.Send(msg)
	if err != nil {
		log.Error(translation.Translate("coin_selection_buttons_failed"), err)
	}
	return ""
}

// InsertAlert handles alert insertion logic
func (b *Bot) InsertAlert(chatID int64, coin *coinpaprika.Coin, target string) (string, error) {
	var alertType string
	var formattedTarget string

	cp, exists := price.GetPrice(*coin.ID)
	if !exists {
		return "", errors.New(translation.Translate("current_price_not_found"))
	}

	if strings.Contains(target, "%") || strings.HasPrefix(target, "-") {
		alertType = "percent"
		target = strings.ReplaceAll(target, "%", "")

		targetValue, err := strconv.ParseFloat(target, 64)
		if err != nil {
			return "", errors.New(fmt.Sprintf(
				translation.Translate("invalid_percent_target"),
				helpers.EscapeMarkdownV2(fmt.Sprintf("%s (%s)", *coin.Name, *coin.Symbol)),
				*coin.ID,
				helpers.FormatPriceUS(cp.PriceUSD, true),
			))
		}
		formattedTarget = helpers.EscapeMarkdownV2(fmt.Sprintf("%.1f%%", targetValue))
	} else {
		targetValue, err := strconv.ParseFloat(target, 64)
		if err != nil {
			return "", errors.New(fmt.Sprintf(
				translation.Translate("invalid_price_target"),
				helpers.EscapeMarkdownV2(fmt.Sprintf("%s (%s)", *coin.Name, *coin.Symbol)),
				*coin.ID,
				helpers.FormatPriceUS(cp.PriceUSD, true),
			))
		}
		alertType = "price"
		formattedTarget = "$" + helpers.FormatPriceUS(targetValue, true)
	}

	err := database.InsertAlert(chatID, *coin.ID, target, alertType, strconv.FormatFloat(cp.PriceUSD, 'f', -1, 64))
	if err != nil {
		log.Error(translation.Translate("alert_save_failed"), err)
		return "", errors.Wrap(err, translation.Translate("database_insert_failed"))
	}

	successMsg := fmt.Sprintf(
		translation.Translate("alert_set_success"),
		helpers.EscapeMarkdownV2(fmt.Sprintf("%s (%s)", *coin.Name, *coin.Symbol)),
		*coin.ID,
		formattedTarget,
	)
	return successMsg, nil
}

func (b *Bot) HandleAlertListCommand(chatID int64) string {
	alerts, err := database.GetAlertsByChatID(chatID)
	if err != nil {
		log.Error(translation.Translate("error_fetching_alerts"), err)
		return translation.Translate("fetch_alerts_failed")
	}

	if len(alerts) == 0 {
		return translation.Translate("no_active_alerts")
	}

	var alertList strings.Builder
	alertList.WriteString(translation.Translate("active_alerts_list_header"))
	for _, alert := range alerts {
		c, err := commands.GetCoinByID(alert.Ticker)
		if err != nil {
			continue
		}

		var targetString string
		if alert.AlertType == "percent" {
			targetString = fmt.Sprintf(translation.Translate("alert_target_percent"), helpers.FormatPercentage(alert.Target))
		} else if alert.AlertType == "price" {
			targetString = fmt.Sprintf(translation.Translate("alert_target_price"), helpers.FormatPriceUS(alert.Target, true))
		} else {
			targetString = fmt.Sprintf(translation.Translate("alert_target_generic"), helpers.FormatPriceUS(alert.Target, true))
		}

		formattedDate := helpers.EscapeMarkdownV2(helpers.FormatDate(alert.CreatedAt))

		alertList.WriteString(fmt.Sprintf(
			translation.Translate("alert_list_item_format"),
			helpers.EscapeMarkdownV2(*c.Name),
			helpers.EscapeMarkdownV2(*c.Symbol),
			targetString,
			formattedDate,
		))
	}

	return alertList.String()
}
