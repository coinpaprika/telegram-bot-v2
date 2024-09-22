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
		"*%s 24h volume:*\n\n▫️`%d`\n\n[See %s on CoinPaprika 🌶](http://coinpaprika.com/coin/%s)",
		*ticker.Name, volumeUSD, *ticker.Name, *ticker.ID), nil
}
