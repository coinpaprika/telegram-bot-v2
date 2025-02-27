package helpers

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"strings"
	"time"
)

func EscapeMarkdownV2(text string) string {
	charactersToEscape := []string{".", "-", "_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "=", "|", "{", "}", "!"}

	for _, char := range charactersToEscape {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}
	return text
}

func FormatPriceUS(price float64, escapeMarkdown bool) string {
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
		return EscapeMarkdownV2(formatted)
	}
	return formatted
}

func FormatPriceRoundedUS(price float64) string {
	roundedPrice := int(price + 0.5)

	thousandSeparator := ","

	p := message.NewPrinter(language.English)
	withCommaThousandSep := p.Sprintf("%d", roundedPrice)
	formatted := strings.ReplaceAll(withCommaThousandSep, ",", thousandSeparator)

	return EscapeMarkdownV2(formatted)
}

func FormatSupplyUS(supply int64) string {
	p := message.NewPrinter(language.English)
	return EscapeMarkdownV2(p.Sprintf("%d", supply))
}

func FormatDate(dateString string) string {
	// Example format: "2024-12-31T12:34:56Z"
	t, err := time.Parse(time.RFC3339, dateString)
	if err != nil {
		log.Error("Error parsing date: ", err)
		return "unknown date"
	}
	return t.Format("Jan 2, 2006")
}

func FormatPercentage(value float64) string {
	return fmt.Sprintf("%.1f", value)
}
