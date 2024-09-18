package commands

import (
	"time"
)

type CacheItem struct {
	ChartData  []byte
	Caption    string
	Expiration time.Time
}

var chartCache = make(map[string]*CacheItem)

func cacheGet(ticker string) (*CacheItem, bool) {
	if item, found := chartCache[ticker]; found && time.Now().Before(item.Expiration) {
		return item, true
	}
	return nil, false
}

func cacheSet(ticker string, chartData []byte, caption string, duration time.Duration) {
	chartCache[ticker] = &CacheItem{
		ChartData:  chartData,
		Caption:    caption,
		Expiration: time.Now().Add(duration),
	}
}
