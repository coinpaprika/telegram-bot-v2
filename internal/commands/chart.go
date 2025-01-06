package commands

import (
	"coinpaprika-telegram-bot/internal/chart"
	"coinpaprika-telegram-bot/lib/helpers"
	"coinpaprika-telegram-bot/lib/translation"
	"fmt"
	"github.com/coinpaprika/coinpaprika-api-go-client/v2/coinpaprika"
	"github.com/pkg/errors"
	"github.com/wcharczuk/go-chart/v2/drawing"
	"log"
	"math"
	"time"
)

var ValidTimeRanges = map[string]time.Time{
	"4h":  time.Now().Add(-4 * time.Hour).UTC(),
	"12h": time.Now().Add(-12 * time.Hour).UTC(),
	"24h": time.Now().Add(-24 * time.Hour).UTC(),
	"7d":  time.Now().Add(-24 * 7 * time.Hour).UTC(),
}

func init() {
	darkGrayBlueSeriesColors := []drawing.Color{
		{R: 0, G: 122, B: 255, A: 255},
		{R: 0, G: 122, B: 255, A: 25},
	}

	chart.AddTheme(
		"darkgrayblue",
		chart.ThemeOption{
			IsDarkMode: true,
			AxisSplitLineColor: chart.Color{
				R: 100, G: 100, B: 100, A: 128,
			},
			BackgroundColor: chart.Color{
				R: 55, G: 55, B: 55, A: 255,
			},
			TextColor: chart.Color{
				R: 200, G: 200, B: 200, A: 255,
			},
			SeriesColors: darkGrayBlueSeriesColors,
		},
	)
}

// CommandChart generates the chart and returns the file path.
func CommandChart(argument, timeRange string) ([]byte, string, error) {
	log.Printf("processing command /c with argument :%s", argument)
	t := getTimeRange(timeRange)
	i := getInterval(timeRange)
	if cachedItem, found := cacheGet(fmt.Sprintf("%s-%s", argument, t.String())); found {
		log.Printf("returning cached result for %s", argument)
		return cachedItem.ChartData, cachedItem.Caption, nil
	}

	c, tickers, _ := GetHistoricalTickersByQuery(argument, t, i)

	if len(tickers) <= 0 {
		return nil, translation.Translate(
			"Coin not traded",
			helpers.EscapeMarkdownV2(*c.Name), *c.Symbol, *c.ID, *c.ID), nil
	}

	chartData, err := renderChart(c, tickers, timeRange)
	if err != nil {
		return nil, "", err
	}

	cacheSet(argument, chartData, *c.Name, 5*time.Minute)

	return chartData, translation.Translate(
		"Coin chart details",
		*c.Symbol, *c.ID), nil
}

func CommandChartWithTicker(argument, timeRange string) ([]byte, string, error) {
	log.Printf("processing command ticker with argument :%s", argument)
	t := getTimeRange(timeRange)
	i := getInterval(timeRange)
	cacheKey := fmt.Sprintf("%s-%s-%s", argument, "ticker", t.String())
	if cachedItem, found := cacheGet(cacheKey); found {
		log.Printf("returning cached result for %s", argument)
		return cachedItem.ChartData, cachedItem.Caption, nil
	}

	c, tickers, err := GetHistoricalTickersByQuery(argument, t, i)
	if err != nil {
		return nil, "", err
	}

	_, details, err := GetTicker(c)
	if err != nil {
		return nil, "", err
	}

	if details == nil || details.Quotes == nil {
		return nil, translation.Translate(
			"Coin not traded",
			helpers.EscapeMarkdownV2(*c.Name), *c.Symbol, *c.ID, *c.ID), nil
	}

	usdQuote := details.Quotes["USD"]

	caption := translation.Translate(
		"Ticker details",
		helpers.EscapeMarkdownV2(*details.Name),
		*details.ID,
		*details.Symbol,
		helpers.FormatPriceUS(*usdQuote.Price, true),
		helpers.EscapeMarkdownV2(fmt.Sprintf("%.2f", *usdQuote.PercentChange1h)),
		helpers.EscapeMarkdownV2(fmt.Sprintf("%.2f", *usdQuote.PercentChange24h)),
		helpers.EscapeMarkdownV2(fmt.Sprintf("%.2f", *usdQuote.PercentChange7d)),
		helpers.FormatPriceRoundedUS(math.Round(*usdQuote.Volume24h)),
		helpers.FormatPriceRoundedUS(math.Round(*usdQuote.MarketCap)),
		func() string {
			if details.CirculatingSupply != nil {
				return helpers.FormatSupplyUS(*details.CirculatingSupply)
			}
			return "N/A"
		}(),
		*details.Symbol,
		helpers.FormatSupplyUS(*details.TotalSupply),
		*details.Symbol,
		helpers.EscapeMarkdownV2(*details.Name),
		*details.ID,
	)

	chartData, err := renderChart(c, tickers, timeRange)
	if err != nil {
		return nil, "", err
	}

	cacheSet(cacheKey, chartData, caption, 5*time.Minute)

	return chartData, caption, nil
}

func renderChart(c *coinpaprika.Coin, tickers []*coinpaprika.TickerHistorical, timeRange string) ([]byte, error) {
	if len(tickers) == 0 {
		return nil, errors.New("no tickers available for rendering")
	}

	var times []*time.Time
	var prices []*float64

	for _, t := range tickers {
		if t.Timestamp == nil || t.Price == nil {
			continue
		}
		times = append(times, t.Timestamp)
		prices = append(prices, t.Price)
	}

	if len(times) == 0 || len(prices) == 0 {
		return nil, errors.New("insufficient valid data for rendering chart")
	}

	if len(prices) < 2 {
		return nil, errors.New("not enough data points for rendering chart")
	}

	priceValues := [][]float64{{}}
	for _, price := range prices {
		priceValues[0] = append(priceValues[0], *price)
	}

	// Adjust X-axis labels based on timeRange
	xLabels := []string{}
	var lastLabel string
	for _, t := range times {
		var currentLabel string
		if timeRange == "4h" || timeRange == "12h" || timeRange == "24h" {
			currentLabel = (*t).Format("15:04") // Show time for shorter ranges
		} else {
			currentLabel = (*t).Format("02-Jan") // Show date for weekly range
		}

		if currentLabel != lastLabel {
			xLabels = append(xLabels, currentLabel)
			lastLabel = currentLabel
		} else {
			xLabels = append(xLabels, "-")
		}
	}

	// Ensure alignment between xLabels and priceValues
	if len(xLabels) != len(priceValues[0]) {
		return nil, errors.New("mismatch between number of labels and data points")
	}

	minPrice, maxPrice := getMinMax(prices)
	if minPrice == maxPrice {
		maxPrice += 1 // Prevent division by zero
	}

	padding := (maxPrice - minPrice) * 0.1
	minValue := minPrice - padding
	maxValue := maxPrice + padding

	titleKey := ""
	switch timeRange {
	case "4h":
		titleKey = "price chart 4h"
	case "12h":
		titleKey = "price chart 12h"
	case "24h":
		titleKey = "price chart 24h"
	default:
		titleKey = "price chart 7d"
	}

	p, err := chart.LineRender(
		priceValues,
		chart.TitleTextOptionFunc("CoinPaprika"),
		chart.ThemeOptionFunc("darkgrayblue"),
		chart.WidthOptionFunc(1200),
		chart.LegendLabelsOptionFunc([]string{""}),
		func(opt *chart.ChartOption) {
			opt.BackgroundColor = chart.Color{R: 55, G: 55, B: 55, A: 255}
			opt.FillArea = true
			opt.SymbolShow = BoolPtr(true)
			opt.Opacity = 35
			opt.Title = chart.TitleOption{
				Text: translation.Translate(titleKey, *c.Name, *c.Symbol),
				Left: "center",
				Top:  "20px",
			}
			opt.ValueFormatter = func(v float64) string {
				return helpers.FormatPriceUS(v, false)
			}
			opt.XAxis = chart.XAxisOption{
				Data:        xLabels,
				BoundaryGap: BoolPtr(false),
				FontSize:    12,
				FontColor:   chart.Color{R: 200, G: 200, B: 200, A: 255},
				Show:        BoolPtr(true),
			}
			opt.YAxisOptions = []chart.YAxisOption{
				{
					Min:           &minValue,
					Max:           &maxValue,
					FontSize:      12,
					FontColor:     chart.Color{R: 200, G: 200, B: 200, A: 255},
					Position:      "left",
					SplitLineShow: BoolPtr(true),
					Show:          BoolPtr(true),
				},
			}
		},
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to render chart")
	}

	buf, err := p.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate chart bytes")
	}

	return buf, nil
}

func getMinMax(prices []*float64) (min, max float64) {
	if len(prices) == 0 {
		return 0, 1
	}

	min, max = *prices[0], *prices[0]
	for _, price := range prices {
		if price == nil {
			continue
		}
		if *price < min {
			min = *price
		}
		if *price > max {
			max = *price
		}
	}

	// Prevent division by zero in range calculations
	if min == max {
		max += 1
	}

	return min, max
}

func BoolPtr(b bool) *bool {
	return &b
}

func getTimeRange(timeRange string) time.Time {
	if t, valid := ValidTimeRanges[timeRange]; valid {
		return t
	}
	log.Printf("Invalid time range: %s. Defaulting to 7d.", timeRange)

	return ValidTimeRanges["7d"]
}

func getInterval(timeRange string) string {
	var interval string

	switch timeRange {
	case "4h":
		interval = "30m"
	case "12h":
		interval = "1h"
	case "24h":
		interval = "2h"
	case "7d":
		interval = "3h"
	default:
		interval = "3h"
	}

	return interval
}
