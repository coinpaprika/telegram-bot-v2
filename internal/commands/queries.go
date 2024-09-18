package commands

import (
	"coinpaprika-telegram-bot/config"
	"github.com/coinpaprika/coinpaprika-api-go-client/v2/coinpaprika"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"time"
)

var paprikaClient *coinpaprika.Client

func init() {
	paprikaClient = getClient()
}

// GetTickerByQuery retrieves the ticker for the given query (symbol, name, etc.)
func GetTickerByQuery(query string) (*coinpaprika.Ticker, error) {
	currency, err := searchCoin(query)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find coin by query")
	}

	log.Debugf("Best match for query '%s' is: %s", query, *currency.ID)
	return GetTicker(currency)
}

// GetTicker fetches the current ticker for the given coin.
func GetTicker(currency *coinpaprika.Coin) (*coinpaprika.Ticker, error) {
	tickerOpts := &coinpaprika.TickersOptions{Quotes: "USD,BTC,ETH"}
	ticker, err := paprikaClient.Tickers.GetByID(*currency.ID, tickerOpts)
	if err != nil {
		return nil, errors.Wrap(err, "error fetching ticker for: "+*currency.Name)
	}
	return ticker, nil
}

// GetHistoricalTickersByQuery fetches historical tickers for the given query.
func GetHistoricalTickersByQuery(query string) (*coinpaprika.Coin, []*coinpaprika.TickerHistorical, error) {
	currency, err := searchCoin(query)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to find coin by query")
	}

	log.Debugf("Best match for query '%s' is: %s", query, *currency.ID)
	return GetHistoricalTickers(currency)
}

// GetHistoricalTickers fetches historical tickers for the given coin.
func GetHistoricalTickers(currency *coinpaprika.Coin) (*coinpaprika.Coin, []*coinpaprika.TickerHistorical, error) {
	tickerOpts := &coinpaprika.TickersHistoricalOptions{
		Quote:    "USD",
		Limit:    100,
		Interval: "1d",
		Start:    time.Now().Add(-24 * 7 * time.Hour).UTC(),
	}
	tickers, err := paprikaClient.Tickers.GetHistoricalTickersByID(*currency.ID, tickerOpts)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error fetching historical data for: "+*currency.Name)
	}
	return currency, tickers, nil
}

// searchCoin searches for a coin based on the provided query.
func searchCoin(query string) (*coinpaprika.Coin, error) {
	searchOpts := &coinpaprika.SearchOptions{
		Query:      query,
		Categories: "currencies",
		Modifier:   "symbol_search",
	}
	result, err := paprikaClient.Search.Search(searchOpts)
	if err != nil || len(result.Currencies) == 0 {
		log.Debugf("No results for symbol search, trying name search for '%s'", query)
		searchOpts = &coinpaprika.SearchOptions{Query: query, Categories: "currencies"}
		result, err = paprikaClient.Search.Search(searchOpts)
		if err != nil || len(result.Currencies) == 0 {
			return nil, errors.Errorf("invalid coin name, ticker, or symbol: %s", query)
		}
	}

	return result.Currencies[0], nil
}

func getClient() *coinpaprika.Client {
	apiProKey := config.GetString("api_pro_key")
	if apiProKey != "" {
		return coinpaprika.NewClient(nil, coinpaprika.WithAPIKey(apiProKey))
	}
	return coinpaprika.NewClient(nil)
}
