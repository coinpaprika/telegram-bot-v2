package commands

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func CommandPrice(argument string) (string, error) {
	log.Debugf("processing command /p with argument :%s", argument)

	c, ticker, err := GetTickerByQuery(argument)
	if err != nil {
		return "", errors.Wrap(err, "command /p")
	}

	if ticker == nil {
		return fmt.Sprintf(
			"[%s \\(%s\\)](https://coinpaprika.com/coin/%s) coin is not actively traded and does not have current price \n"+
				"For more details visit [CoinPaprika](https://coinpaprika.com/coin/%s)🌶",
			escapeMarkdownV2(*c.Name), *c.Symbol, *c.ID, *c.ID), nil
	}

	priceUSD := ticker.Quotes["USD"].Price
	priceBTC := ticker.Quotes["BTC"].Price
	if ticker.Name == nil || ticker.ID == nil || priceUSD == nil || priceBTC == nil {
		return fmt.Sprintf(
			"[%s \\(%s\\)](https://coinpaprika.com/coin/%s) coin is not actively traded and does not have current price \n"+
				"For more details visit [CoinPaprika](https://coinpaprika.com/coin/%s)🌶",
			escapeMarkdownV2(*c.Name), *c.Symbol, *c.ID, *c.ID), nil
	}

	return fmt.Sprintf("*%s price:*\n\n▫️`%s` *USD*\n▫️`%s` *BTC*\n\n"+
		"%s on [CoinPaprika](https://coinpaprika.com/coin/%s)🌶/ Use this [Bot](https://github.com/coinpaprika/telegram-bot-v2/blob/main/README.md)",
		*ticker.Name, formatPriceUS(*priceUSD), formatPriceUS(*priceBTC), *ticker.Symbol, *ticker.ID), nil
}
