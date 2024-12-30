package commands

import (
	"coinpaprika-telegram-bot/lib/translation"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strings"
)

func CommandSupply(argument string) (string, error) {
	log.Debugf("processing command /s with argument :%s", argument)

	c, ticker, err := GetTickerByQuery(strings.TrimSpace(argument))
	if err != nil {
		return "", errors.Wrap(err, "command /s")
	}

	if ticker == nil || ticker.Name == nil || ticker.ID == nil || ticker.CirculatingSupply == nil {
		return translation.Translate(
			"Coin not traded",
			escapeMarkdownV2(*c.Name), *c.Symbol, *c.ID, *c.ID), nil
	}

	return translation.Translate(
		"Coin supply details",
		escapeMarkdownV2(*ticker.Name), formatSupplyUS(*ticker.CirculatingSupply), *ticker.Symbol, *ticker.ID), nil
}
