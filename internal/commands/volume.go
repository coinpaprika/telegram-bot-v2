package commands

import (
	"coinpaprika-telegram-bot/lib/translation"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strings"
)

func CommandVolume(argument string) (string, error) {
	log.Debugf("processing command /v with argument :%s", argument)

	c, ticker, err := GetTickerByQuery(strings.TrimSpace(argument))
	if err != nil {
		return "", errors.Wrap(err, "command /v")
	}

	if ticker == nil {
		return translation.Translate(
			"Coin not traded",
			escapeMarkdownV2(*c.Name), *c.Symbol, *c.ID, *c.ID), nil
	}

	volumeUSD := ticker.Quotes["USD"].Volume24h
	if ticker.Name == nil || ticker.ID == nil || volumeUSD == nil {
		return translation.Translate(
			"Coin not traded",
			escapeMarkdownV2(*c.Name), *c.Symbol, *c.ID, *c.ID), nil
	}

	return translation.Translate(
		"Coin volume details",
		*ticker.Name, formatPriceUS(*volumeUSD, true), *ticker.Symbol, *ticker.ID), nil
}
