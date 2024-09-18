package commands

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"strings"
)

func formatPriceUS(price float64) string {
	decimals := 6
	if price > 1.2 {
		decimals = 2
	} else if price < 0.00001 {
		decimals = 8
	}
	thousandSeparator := ","

	p := message.NewPrinter(language.English)
	withCommaThousandSep := p.Sprintf("%.*f", decimals, price)
	formatted := strings.ReplaceAll(withCommaThousandSep, ",", thousandSeparator)

	return formatted
}

func formatSupplyUS(supply int64) string {
	p := message.NewPrinter(language.English)
	return p.Sprintf("%d", supply)
}
