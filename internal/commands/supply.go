package commands

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func CommandSupply(argument string) (string, error) {
	log.Debugf("processing command /s with argument :%s", argument)

	ticker, err := GetTickerByQuery(argument)

	if err != nil {
		return "", errors.Wrap(err, "command /s")
	}

	if ticker.Name == nil || ticker.ID == nil || ticker.CirculatingSupply == nil {
		return "", errors.Wrap(errors.New("missing data"), "command /s")
	}

	return fmt.Sprintf(
		"*%s circulating supply:*\n\n▫️`%d`\n\n[See %s on CoinPaprika 🌶](http://coinpaprika.com/coin/%s)",
		*ticker.Name, *ticker.CirculatingSupply, *ticker.Name, *ticker.ID), nil
}
