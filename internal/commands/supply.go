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
		return fmt.Sprintf("This coin is not actively traded and doesn't have current price \n"+
			"For more details visit [coinpaprika.com]https://coinpaprika.com/coin/%s", *ticker.ID), nil
	}

	return fmt.Sprintf(
		"*%s circulating supply:*\n\n▫️`%d`\n\n[See %s on CoinPaprika 🌶](http://coinpaprika.com/coin/%s)",
		*ticker.Name, *ticker.CirculatingSupply, *ticker.Name, *ticker.ID), nil
}
