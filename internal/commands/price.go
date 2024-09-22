package commands

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func CommandPrice(argument string) (string, error) {
	log.Debugf("processing command /p with argument :%s", argument)

	ticker, err := GetTickerByQuery(argument)
	if err != nil {
		return "", errors.Wrap(err, "command /p")
	}

	priceUSD := ticker.Quotes["USD"].Price
	priceBTC := ticker.Quotes["BTC"].Price
	if ticker.Name == nil || ticker.ID == nil || priceUSD == nil || priceBTC == nil {
		return fmt.Sprintf("This coin is not actively traded and doesn't have current price \n"+
			"For more details visit [coinpaprika.com]https://coinpaprika.com/coin/%s", *ticker.ID), nil
	}

	return fmt.Sprintf("*%s price:*\n\n▫️`%.8f` *USD*\n▫️`%.8f` *BTC*\n\n[See %s on CoinPaprika 🌶](https://coinpaprika.com/coin/%s)", *ticker.Name, *priceUSD, *priceBTC, *ticker.Name, *ticker.ID), nil
}
