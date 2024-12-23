package commands

import (
	"coinpaprika-telegram-bot/lib/translation"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strings"
)

func CommandPrice(argument string) (string, error) {
	log.Debugf("processing command /p with argument :%s", argument)

	c, ticker, err := GetTickerByQuery(strings.TrimSpace(argument))
	if err != nil {
		return "", errors.Wrap(err, "command /p")
	}

	if ticker == nil {
		return translation.Translate(
			"Coin not traded",
			escapeMarkdownV2(*c.Name), *c.Symbol, *c.ID, *c.ID), nil
	}

	priceUSD := ticker.Quotes["USD"].Price
	priceBTC := ticker.Quotes["BTC"].Price
	if ticker.Name == nil || ticker.ID == nil || priceUSD == nil || priceBTC == nil {
		return translation.Translate(
			"Coin not traded",
			escapeMarkdownV2(*c.Name), *c.Symbol, *c.ID, *c.ID), nil
	}

	return translation.Translate(
		"Coin price details",
		*ticker.Name, formatPriceUS(*priceUSD, true), formatPriceUS(*priceBTC, true), *ticker.Symbol, *ticker.ID), nil
}
