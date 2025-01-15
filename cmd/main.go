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
	dto "github.com/prometheus/client_model/go"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
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
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDB()

	LoadMetricsFromDB()

	price.StartPriceUpdater()
	bot, err := telegram.NewBot(telegram.BotConfig{
		Token:          config.GetString("telegram_bot_token"),
		Debug:          config.GetBool("debug"),
		UpdatesTimeout: 60,
	})

	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	alert.StartAlertService(bot)

	updates, err := bot.GetUpdatesChannel()
	if err != nil {
		log.Fatalf("Failed to get updates channel: %v", err)
	}

	go handleUpdates(bot, updates)

	go func() {
		for {
			time.Sleep(5 * time.Minute)
			SaveMetricsToDB()
		}
	}()

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		SaveMetricsToDB()
		log.Println("Metrics saved, shutting down...")
		os.Exit(0)
	}()

	if err := launchMetricsAndHealthServer(config.GetInt("metrics_port")); err != nil {
		log.Fatalf("Failed to start metrics and health server: %v", err)
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

func LoadMetricsFromDB() {
	metrics.Mutex.Lock()
	defer metrics.Mutex.Unlock()

	// Load non-labeled metrics
	commandsProcessed, _ := database.GetMetric("commands_processed")
	messagesHandled, _ := database.GetMetric("messages_handled")
	channelsCount, _ := database.GetMetric("channels_count")

	metrics.CommandsProcessed.Add(commandsProcessed)
	metrics.MessagesHandled.Add(messagesHandled)
	metrics.ChannelsCount.Set(channelsCount)

	// Load labeled metrics
	loadLabeledMetrics("channel_names", func(chatIDStr, chatName string, _ float64) {
		chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
		if err != nil {
			log.Printf("Failed to parse chatID %s: %v", chatIDStr, err)
			return
		}
		metrics.ChannelNames.WithLabelValues(chatIDStr, chatName).Add(1)
		metrics.ChannelsSet[chatID] = chatName
	})

	loadLabeledMetrics("messages_per_channel", func(chatID, chatName string, value float64) {
		metrics.MessagesPerChannel.WithLabelValues(chatID, chatName).Add(value)
	})

	log.Println("Metrics loaded from database.")
}

func loadLabeledMetrics(metricName string, callback func(labelKey, labelValue string, value float64)) {
	metricsWithLabels, _ := database.GetMetricsWithLabels(metricName)
	for labelKey, labelValues := range metricsWithLabels {
		for labelValue, value := range labelValues {
			callback(labelKey, labelValue, value)
		}
	}
}

func SaveMetricsToDB() {
	metrics.Mutex.Lock()
	defer metrics.Mutex.Unlock()

	// Save non-labeled metrics
	database.SaveMetric("commands_processed", "", "", GetMetricValue(metrics.CommandsProcessed))
	database.SaveMetric("messages_handled", "", "", GetMetricValue(metrics.MessagesHandled))
	database.SaveMetric("channels_count", "", "", float64(len(metrics.ChannelsSet)))

	// Save labeled metrics: channel_names
	for chatID, chatName := range metrics.ChannelsSet {
		database.SaveMetricWithLabels("channel_names", fmt.Sprintf("%d", chatID), chatName, float64(chatID))
	}

	// Save labeled metrics: messages_per_channel
	metricChan := make(chan prometheus.Metric, 1)
	go func() {
		metrics.MessagesPerChannel.Collect(metricChan)
		close(metricChan)
	}()

	for metric := range metricChan {
		metricProto := &dto.Metric{}
		if err := metric.Write(metricProto); err != nil {
			log.Printf("Failed to read MessagesPerChannel metric: %v", err)
			continue
		}
		var chatID, chatName string
		for _, label := range metricProto.Label {
			if label.GetName() == "chat_id" {
				chatID = label.GetValue()
			}
			if label.GetName() == "chat_name" {
				chatName = label.GetValue()
			}
		}
		database.SaveMetricWithLabels("messages_per_channel", chatID, chatName, metricProto.Counter.GetValue())
	}

	log.Println("Metrics saved to database.")
}

func GetMetricValue(metric prometheus.Collector) float64 {
	var metricValue float64
	metricChan := make(chan prometheus.Metric, 1)
	metric.Collect(metricChan)
	close(metricChan)

	metricProto := &dto.Metric{}
	if err := (<-metricChan).Write(metricProto); err != nil {
		log.Printf("Failed to read metric value: %v", err)
		return 0
	}

	if metricProto.Counter != nil {
		metricValue = metricProto.Counter.GetValue()
	} else if metricProto.Gauge != nil {
		metricValue = metricProto.Gauge.GetValue()
	}
	return metricValue
}
