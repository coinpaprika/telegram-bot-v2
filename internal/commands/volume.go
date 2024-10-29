package commands

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func CommandVolume(argument string) (string, error) {
	log.Debugf("processing command /v with argument :%s", argument)

	c, ticker, err := GetTickerByQuery(argument)
	if err != nil {
		return "", errors.Wrap(err, "command /v")
	}

	if ticker == nil {
		return fmt.Sprintf(
			"[%s \\(%s\\)](https://coinpaprika.com/coin/%s) coin is not actively traded and does not have current price \n"+
				"For more details visit [CoinPaprika](https://coinpaprika.com/coin/%s)🌶",
			escapeMarkdownV2(*c.Name), *c.Symbol, *c.ID, *c.ID), nil
	}

	volumeUSD := ticker.Quotes["USD"].Volume24h
	if ticker.Name == nil || ticker.ID == nil || volumeUSD == nil {
		return fmt.Sprintf(
			"[%s \\(%s\\)](https://coinpaprika.com/coin/%s) coin is not actively traded and does not have current price \n"+
				"For more details visit [CoinPaprika](https://coinpaprika.com/coin/%s)🌶",
			escapeMarkdownV2(*c.Name), *c.Symbol, *c.ID, *c.ID), nil
	}

	return fmt.Sprintf(
		"*%s 24h volume:*\n\n▫️`%s` *USD*\n\n"+
			"%s on [CoinPaprika](https://coinpaprika.com/coin/%s)🌶/ Use this [Bot](https://github.com/coinpaprika/telegram-bot-v2/blob/main/README.md)",
		*ticker.Name, formatPriceUS(*volumeUSD), *ticker.Symbol, *ticker.ID), nil
}
