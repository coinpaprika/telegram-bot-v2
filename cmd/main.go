package main

import (
	"bytes"
	"coinpaprika-telegram-bot/config"
	"coinpaprika-telegram-bot/internal/alert"
	"coinpaprika-telegram-bot/internal/database"
	"coinpaprika-telegram-bot/internal/price"
	"coinpaprika-telegram-bot/internal/telegram"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/leonelquinteros/gotext"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
	"runtime"
	"strings"
	"sync"
)

type BotMetrics struct {
	CommandsProcessed  prometheus.Counter
	MessagesHandled    prometheus.Counter
	ChannelsCount      prometheus.Gauge
	ChannelNames       *prometheus.CounterVec
	ChannelsSet        map[int64]string
	MessagesPerChannel *prometheus.CounterVec
	Mutex              sync.Mutex
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
		ChannelNames: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "coinpaprika",
				Subsystem: "telegram_bot",
				Name:      "channel_names",
				Help:      "Tracks channels the bot has interacted with",
			},
			[]string{"chat_id", "chat_name"},
		),
		MessagesPerChannel: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "coinpaprika",
				Subsystem: "telegram_bot",
				Name:      "messages_per_channel",
				Help:      "The total number of messages handled per channel",
			},
			[]string{"chat_id", "chat_name"},
		),
		ChannelsSet: make(map[int64]string),
	}

	prometheus.MustRegister(metrics.CommandsProcessed)
	prometheus.MustRegister(metrics.MessagesHandled)
	prometheus.MustRegister(metrics.ChannelsCount)
	prometheus.MustRegister(metrics.ChannelNames)
	prometheus.MustRegister(metrics.MessagesPerChannel)

	return metrics
}

func main() {
	gotext.Configure("locales", strings.ToLower(config.GetString("lang")), "default")
	err := database.InitDB("/app/data/bot.db")
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	price.StartPriceUpdater()
	bot, err := telegram.NewBot(telegram.BotConfig{
		Token:          config.GetString("telegram_bot_token"),
		Debug:          config.GetBool("debug"),
		UpdatesTimeout: 60,
	})

	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	alert.StartAlertService(bot)

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
		if update.CallbackQuery != nil {
			bot.HandleCallbackQuery(update.CallbackQuery)
			continue
		}

		if update.Message == nil {
			log.Debug("Received non-message or non-command")
			continue
		}

		if update.Message.ReplyToMessage != nil {
			bot.HandleReply(update.Message)
			continue
		}

		if update.Message.IsCommand() == false && (len(update.Message.Text) == 0 || update.Message.Text[0] != '$') {
			continue
		}

		metrics.MessagesHandled.Inc()

		chatID := update.Message.Chat.ID
		chatName := update.Message.Chat.Title
		if chatName == "" {
			chatName = fmt.Sprintf("%s-%d", "PrivateChat", chatID)
		}

		updateChannelsSet(chatID, chatName)

		metrics.MessagesPerChannel.WithLabelValues(
			fmt.Sprintf("%d", chatID), chatName,
		).Inc()

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

func updateChannelsSet(chatID int64, chatName string) {
	metrics.Mutex.Lock()
	defer metrics.Mutex.Unlock()

	if _, exists := metrics.ChannelsSet[chatID]; !exists {
		metrics.ChannelsSet[chatID] = chatName
		metrics.ChannelsCount.Set(float64(len(metrics.ChannelsSet)))

		metrics.ChannelNames.WithLabelValues(fmt.Sprintf("%d", chatID), chatName).Inc()
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
