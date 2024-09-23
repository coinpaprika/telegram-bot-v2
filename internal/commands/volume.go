package commands

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func CommandVolume(argument string) (string, error) {
	log.Debugf("processing command /v with argument :%s", argument)

	ticker, err := GetTickerByQuery(argument)
	if err != nil {
		return "", errors.Wrap(err, "command /v")
	}

	volumeUSD := ticker.Quotes["USD"].Volume24h
	if ticker.Name == nil || ticker.ID == nil || volumeUSD == nil {
		return fmt.Sprintf("This coin is not actively traded and doesn't have current price \n"+
			"For more details visit [coinpaprika.com]https://coinpaprika.com/coin/%s", *ticker.ID), nil
	}

	return fmt.Sprintf(
		"*%s 24h volume:*\n\n▫️`%s` *USD*\n\n"+
			"%s on [CoinPaprika](https://coinpaprika.com/coin/%s)🌶/ Use this [Bot](https://github.com/coinpaprika/telegram-bot-v2/blob/main/README.md)",
		*ticker.Name, formatPriceUS(*volumeUSD), *ticker.Symbol, *ticker.ID), nil
}
