package config

import (
	"github.com/spf13/viper"
	"sync"
)

var once sync.Once

func InitConfig() {
	once.Do(func() {
		viper.AutomaticEnv()

		viper.BindEnv("metrics_port", "METRICS_PORT")
		viper.BindEnv("telegram_bot_token", "TELEGRAM_BOT_TOKEN")
		viper.BindEnv("api_pro_key", "API_PRO_KEY")
		viper.BindEnv("debug", "DEBUG")
		viper.BindEnv("lang", "LANG")

		viper.SetDefault("metrics_port", 9090)
		viper.SetDefault("debug", false)
		viper.SetDefault("lang", "en")
	})
}

func GetString(key string) string {
	InitConfig()
	return viper.GetString(key)
}

func GetInt(key string) int {
	InitConfig()
	return viper.GetInt(key)
}

func GetBool(key string) bool {
	InitConfig()
	return viper.GetBool(key)
}
