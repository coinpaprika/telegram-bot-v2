package alert

import (
	"coinpaprika-telegram-bot/internal/database"
	"coinpaprika-telegram-bot/internal/price"
	"coinpaprika-telegram-bot/internal/telegram"
	"coinpaprika-telegram-bot/lib/helpers"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// alertProcessingMutex ensures only one alert processor runs at a time
var alertProcessingMutex sync.Mutex

// CheckAlerts compares alerts with live prices and sends notifications
func CheckAlerts(bot *telegram.Bot) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🔥 Panic recovered in alert checker: %v. Restarting alert checker in 10 seconds...\n", r)
			time.Sleep(10 * time.Second)
			go StartAlertService(bot)
		}
	}()

	log.Println("🔄 Checking alerts...")

	alerts, err := database.GetAllAlerts()
	if err != nil {
		log.Printf("❌ Failed to fetch alerts from the database: %v\n", err)
		return
	}

	for _, alert := range alerts {
		priceInfo, exists := price.GetPrice(alert.Ticker)
		if !exists {
			log.Printf("⚠️ No price data found for ticker: %s\n", alert.Ticker)
			continue
		}

		referencePrice := alert.CurrentPrice
		if err != nil {
			log.Printf("❌ Failed to parse current price for alert ID: %d | Error: %v\n", alert.ID, err)
			continue
		}

		if alert.AlertType == "price" {
			log.Printf("🔍 Checking price alert ID: %d | Ticker: %s | Target: %.2f | Current: %.2f\n",
				alert.ID, alert.Ticker, alert.Target, priceInfo.PriceUSD)

			if priceInfo.PriceUSD >= alert.Target {
				message := fmt.Sprintf(
					"🚨 *Price Alert Triggered*\n\n*%s \\(%s\\)* has reached the target price of *$%s*\nCurrent Price: *$%s*",
					helpers.EscapeMarkdownV2(priceInfo.Name),
					helpers.EscapeMarkdownV2(priceInfo.Symbol),
					helpers.FormatPriceRoundedUS(math.Round(alert.Target)),
					helpers.FormatPriceRoundedUS(math.Round(priceInfo.PriceUSD)),
				)

				err := bot.SendMessage(telegram.Message{
					ChatID: int(alert.ChatID),
					Text:   message,
				})
				if err != nil {
					log.Printf("❌ Failed to send price alert notification: %v\n", err)
				} else {
					log.Printf("✅ Price alert notification sent to Chat ID: %d\n", alert.ChatID)
				}

				_ = database.DeleteAlert(alert.ID)
			}
		} else if alert.AlertType == "percent" {
			percentageChange := ((priceInfo.PriceUSD - referencePrice) / referencePrice) * 100
			targetPercent := alert.Target

			log.Printf("🔍 Checking percent alert ID: %d | Ticker: %s | Target: %.2f%% | Current Change: %.2f%%\n",
				alert.ID, alert.Ticker, targetPercent, percentageChange)

			if (targetPercent > 0 && percentageChange >= targetPercent) || (targetPercent < 0 && percentageChange <= targetPercent) {
				message := fmt.Sprintf(
					"🚨 *Percent Alert Triggered*\n\n*%s \\(%s\\)* has reached the target change of *%s%%*\nCurrent Change: *%s%%*",
					helpers.EscapeMarkdownV2(priceInfo.Name),
					helpers.EscapeMarkdownV2(priceInfo.Symbol),
					helpers.EscapeMarkdownV2(fmt.Sprintf("%.2f", targetPercent)),
					helpers.EscapeMarkdownV2(fmt.Sprintf("%.2f", percentageChange)),
				)

				err := bot.SendMessage(telegram.Message{
					ChatID: int(alert.ChatID),
					Text:   message,
				})
				if err != nil {
					log.Printf("❌ Failed to send percent alert notification: %v\n", err)
				} else {
					log.Printf("✅ Percent alert notification sent to Chat ID: %d\n", alert.ChatID)
				}

				_ = database.DeleteAlert(alert.ID)
			}
		}
	}

	log.Println("✅ Alert check completed.")
}

// StartAlertService starts a background service to check alerts every minute
func StartAlertService(bot *telegram.Bot) {
	go func() {
		for {
			alertProcessingMutex.Lock()
			CheckAlerts(bot)
			alertProcessingMutex.Unlock()
			time.Sleep(1 * time.Minute) // Check alerts every 1 minute
		}
	}()
	log.Println("🚀 Alert service started.")
}
