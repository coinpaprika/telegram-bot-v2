package commands

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"strings"
)

func escapeMarkdownV2(text string) string {
	charactersToEscape := []string{".", "-", "_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "=", "|", "{", "}", "!"}

	for _, char := range charactersToEscape {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}
	return text
}

func formatPriceUS(price float64, escapeMarkdown bool) string {
	decimals := 6

	if price >= 1000 {
		decimals = 0
	} else if price > 1.2 {
		decimals = 2
	} else if price < 0.00001 {
		decimals = 8
	}

	thousandSeparator := ","

	p := message.NewPrinter(language.English)
	withCommaThousandSep := p.Sprintf("%.*f", decimals, price)
	formatted := strings.ReplaceAll(withCommaThousandSep, ",", thousandSeparator)

	if escapeMarkdown {
		return escapeMarkdownV2(formatted)
	}
	return formatted
}

func formatPriceRoundedUS(price float64) string {
	roundedPrice := int(price + 0.5)

	thousandSeparator := ","

	p := message.NewPrinter(language.English)
	withCommaThousandSep := p.Sprintf("%d", roundedPrice)
	formatted := strings.ReplaceAll(withCommaThousandSep, ",", thousandSeparator)

	return escapeMarkdownV2(formatted)
}

func formatSupplyUS(supply int64) string {
	p := message.NewPrinter(language.English)
	return escapeMarkdownV2(p.Sprintf("%d", supply))
}
