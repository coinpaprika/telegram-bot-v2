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
		"*%s circulating supply:*\n\n▫️`%s`\n\n"+
			"%s on [CoinPaprika](https://coinpaprika.com/coin/%s)🌶/ Use this [Bot](https://github.com/coinpaprika/telegram-bot-v2/blob/main/README.md)",
		*ticker.Name, formatSupplyUS(*ticker.CirculatingSupply), *ticker.Symbol, *ticker.ID), nil
}
