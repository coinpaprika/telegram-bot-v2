package price

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// PriceInfo represents the pricing details of a cryptocurrency
type PriceInfo struct {
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	Symbol         string  `json:"symbol"`
	PriceUSD       float64 `json:"price_usd"`
	MarketCap      float64 `json:"market_cap"`
	PriceChange24h float64 `json:"percent_change_24h"`
	LastUpdated    string  `json:"last_updated"`
}

// priceStorage holds the cryptocurrency prices in memory
var (
	cryptoPrices      = make(map[string]PriceInfo)
	idMapping         = make(map[string]string)
	cryptoPricesMutex = sync.RWMutex{}
	idMutex           = sync.RWMutex{}
)

// fetchCryptoPrices fetches prices from CoinPaprika API
func fetchCryptoPrices() {
	apiURL := "https://api.coinpaprika.com/v1/tickers"

	defer func() {
		if r := recover(); r != nil {
			log.Printf("🔥 Panic recovered in price fetcher: %v. Restarting fetcher in 10 seconds...\n", r)
			time.Sleep(10 * time.Second)
			go fetchCryptoPrices()
		}
	}()

	for {
		resp, err := http.Get(apiURL)
		if err != nil {
			log.Printf("❌ Failed to fetch cryptocurrency prices: %v\n", err)
			time.Sleep(30 * time.Second) // Retry after 30 seconds
			continue
		}

		defer resp.Body.Close()

		var tickers []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Symbol      string `json:"symbol"`
			LastUpdated string `json:"last_updated"`
			Quotes      struct {
				USD struct {
					Price          float64 `json:"price"`
					MarketCap      float64 `json:"market_cap"`
					PriceChange24h float64 `json:"percent_change_24h"`
				} `json:"USD"`
			} `json:"quotes"`
		}

		err = json.NewDecoder(resp.Body).Decode(&tickers)
		if err != nil {
			log.Printf("❌ Failed to parse cryptocurrency prices: %v\n", err)
			time.Sleep(30 * time.Second)
			continue
		}

		cryptoPricesMutex.Lock()
		for i, ticker := range tickers {
			cryptoPrices[ticker.ID] = PriceInfo{
				ID:             i + 1,
				Name:           ticker.Name,
				Symbol:         ticker.Symbol,
				PriceUSD:       ticker.Quotes.USD.Price,
				MarketCap:      ticker.Quotes.USD.MarketCap,
				PriceChange24h: ticker.Quotes.USD.PriceChange24h,
				LastUpdated:    ticker.LastUpdated,
			}

			idMapping[strconv.Itoa(i+1)] = ticker.ID
		}
		cryptoPricesMutex.Unlock()

		log.Println("✅ Cryptocurrency prices updated successfully.")

		time.Sleep(30 * time.Second)
	}
}

// StartPriceUpdater initializes the price fetcher goroutine
func StartPriceUpdater() {
	go fetchCryptoPrices()
	log.Println("🚀 Price updater started.")
}

// GetPrice retrieves the price information for a given cryptocurrency ID
func GetPrice(tickerID string) (PriceInfo, bool) {
	cryptoPricesMutex.RLock()
	defer cryptoPricesMutex.RUnlock()

	price, exists := cryptoPrices[tickerID]
	return price, exists
}

func GetTickerByID(ID string) (string, bool) {
	idMutex.RLock()
	defer idMutex.RUnlock()

	ticker, exists := idMapping[ID]
	return ticker, exists
}

// GetAllPrices returns all stored cryptocurrency prices
func GetAllPrices() map[string]PriceInfo {
	cryptoPricesMutex.RLock()
	defer cryptoPricesMutex.RUnlock()

	// Return a copy to prevent external modification
	copy := make(map[string]PriceInfo)
	for k, v := range cryptoPrices {
		copy[k] = v
	}
	return copy
}
