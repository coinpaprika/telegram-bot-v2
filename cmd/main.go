package main

import (
	"bytes"
	"coinpaprika-telegram-bot/config"
	"coinpaprika-telegram-bot/internal/telegram"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
	"runtime"
	"sync"
)

type BotMetrics struct {
	CommandsProcessed prometheus.Counter
	MessagesHandled   prometheus.Counter
	ChannelsCount     prometheus.Gauge
	ChannelsSet       map[int64]struct{}
	Mutex             sync.Mutex
}

var (
	metrics = NewBotMetrics()
)

func init() {
	config.InitConfig()
	setupLogging()
}

func NewBotMetrics() *BotMetrics {
	metrics := &BotMetrics{
		CommandsProcessed: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "coinpaprika",
			Subsystem: "telegram_bot",
			Name:      "commands_processed",
			Help:      "The total number of processed commands",
		}),
		MessagesHandled: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "coinpaprika",
			Subsystem: "telegram_bot",
			Name:      "messages_handled",
			Help:      "The total number of handled messages",
		}),
		ChannelsCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "coinpaprika",
			Subsystem: "telegram_bot",
			Name:      "channels_count",
			Help:      "The current number of unique channels the bot is operating in",
		}),
		ChannelsSet: make(map[int64]struct{}),
	}

	prometheus.MustRegister(metrics.CommandsProcessed)
	prometheus.MustRegister(metrics.MessagesHandled)
	prometheus.MustRegister(metrics.ChannelsCount)

	return metrics
}

func main() {
	bot, err := telegram.NewBot(telegram.BotConfig{
		Token:          config.GetString("telegram_bot_token"),
		Debug:          config.GetBool("debug"),
		UpdatesTimeout: 60,
	})

	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	updates, err := bot.GetUpdatesChannel()
	if err != nil {
		log.Fatalf("failed to get updates channel: %v", err)
	}

	go handleUpdates(bot, updates)

	if err := launchMetricsAndHealthServer(config.GetInt("metrics_port")); err != nil {
		log.Fatalf("failed to start metrics and health server: %v", err)
	}
}

func setupLogging() {
	log.SetLevel(log.ErrorLevel)
	if config.GetBool("debug") {
		log.SetLevel(log.DebugLevel)
	}
	log.Debug("Starting telegram bot...")
}

func handleUpdates(bot *telegram.Bot, updates tgbotapi.UpdatesChannel) {
	for update := range updates {

		if update.Message == nil {
			log.Debug("Received non-message or non-command")
			continue
		}

		if update.Message.IsCommand() == false && (len(update.Message.Text) == 0 || update.Message.Text[0] != '$') {
			continue
		}

		metrics.MessagesHandled.Inc()

		updateChannelsSet(update.Message.Chat.ID)

		handleCommand(bot, update)
	}
}

func handleCommand(bot *telegram.Bot, update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			stackBuf := make([]byte, 1024)
			stackSize := runtime.Stack(stackBuf, false)
			stackTrace := bytes.TrimRight(stackBuf[:stackSize], "\x00")
			log.Errorf("Recovered from panic: %v\nStack trace: %s", r, stackTrace)
		}
	}()

	err := bot.SendMessage(telegram.Message{
		ChatID:    int(update.Message.Chat.ID),
		Text:      bot.HandleUpdate(update),
		MessageID: update.Message.MessageID,
	})

	if err != nil {
		log.Errorf("Failed to send message: %v", err)
	} else {
		metrics.CommandsProcessed.Inc()
	}
}

func updateChannelsSet(chatID int64) {
	metrics.Mutex.Lock()
	defer metrics.Mutex.Unlock()

	if _, exists := metrics.ChannelsSet[chatID]; !exists {
		metrics.ChannelsSet[chatID] = struct{}{}
		metrics.ChannelsCount.Set(float64(len(metrics.ChannelsSet)))
	}
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func launchMetricsAndHealthServer(port int) error {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", healthCheckHandler)

	log.Infof("Launching metrics and health endpoint on :%d", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), http.DefaultServeMux)
}
