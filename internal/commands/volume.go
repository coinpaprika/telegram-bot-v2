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
		return "", errors.Wrap(errors.New("missing data"), "command /v")
	}

	return fmt.Sprintf(
		"*%s 24h volume:*\n\n▫️`%d`\n\n[See %s on CoinPaprika 🌶](http://coinpaprika.com/coin/%s)",
		*ticker.Name, volumeUSD, *ticker.Name, *ticker.ID), nil
}
