package commands

import (
	"coinpaprika-telegram-bot/internal/chart"
	"fmt"
	"github.com/coinpaprika/coinpaprika-api-go-client/v2/coinpaprika"
	"github.com/pkg/errors"
	"github.com/wcharczuk/go-chart/v2/drawing"
	"log"
	"math"
	"time"
)

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
func CommandChart(argument string) ([]byte, string, error) {
	log.Printf("processing command /c with argument :%s", argument)

	if cachedItem, found := cacheGet(argument); found {
		log.Printf("returning cached result for %s", argument)
		return cachedItem.ChartData, cachedItem.Caption, nil
	}

	c, tickers, _ := GetHistoricalTickersByQuery(argument)

	if len(tickers) <= 0 {
		return nil, fmt.Sprintf(
			"[%s \\(%s\\)](https://coinpaprika.com/coin/%s) coin is not actively traded and does not have current price \n"+
				"For more details visit [CoinPaprika](https://coinpaprika.com/coin/%s) 🌶",
			escapeMarkdownV2(*c.Name), *c.Symbol, *c.ID, *c.ID), nil
	}

	chartData, err := renderChart(c, tickers)
	if err != nil {
		return nil, "", err
	}

	cacheSet(argument, chartData, *c.Name, 5*time.Minute)

	return chartData, fmt.Sprintf(
		"%s on [CoinPaprika](https://coinpaprika.com/coin/%s) 🌶/ Use this [Bot](https://github.com/coinpaprika/telegram-bot-v2/blob/main/README.md)",
		*c.Symbol, *c.ID), nil
}

func CommandChartWithTicker(argument string) ([]byte, string, error) {
	log.Printf("processing command ticker with argument :%s", argument)

	if cachedItem, found := cacheGet(argument); found {
		log.Printf("returning cached result for %s", argument)
		return cachedItem.ChartData, cachedItem.Caption, nil
	}

	c, tickers, err := GetHistoricalTickersByQuery(argument)
	if err != nil {
		return nil, "", err
	}

	_, details, err := GetTicker(c)
	if err != nil {
		return nil, "", err
	}

	if details == nil || details.Quotes == nil {
		return nil, fmt.Sprintf(
			"[%s \\(%s\\)](https://coinpaprika.com/coin/%s) coin is not actively traded and does not have current price \n"+
				"For more details visit [CoinPaprika](https://coinpaprika.com/coin/%s) 🌶",
			escapeMarkdownV2(*c.Name), *c.Symbol, *c.ID, *c.ID), nil
	}

	usdQuote := details.Quotes["USD"]

	caption := fmt.Sprintf(
		"[%s](https://coinpaprika.com/coin/%s) \\(%s\\)\n"+
			"Price:  *$%s*\n"+
			"1h price change: *%s%%*\n"+
			"24h price change: *%s%%*\n"+
			"7d price change: *%s%%*\n"+
			"Vol:  *$%s*\n"+
			"MCap:  *$%s*\n"+
			"Circ\\. Supply:  *%s %s*\n"+
			"Total Supply:  *%s %s*\n\n"+
			"[%s on CoinPaprika](https://coinpaprika.com/coin/%s) 🌶",
		escapeMarkdownV2(*details.Name),
		*details.ID,
		*details.Symbol,
		formatPriceUS(*usdQuote.Price, true),
		escapeMarkdownV2(fmt.Sprintf("%.2f", *usdQuote.PercentChange1h)),  // Escaping percentage
		escapeMarkdownV2(fmt.Sprintf("%.2f", *usdQuote.PercentChange24h)), // Escaping percentage
		escapeMarkdownV2(fmt.Sprintf("%.2f", *usdQuote.PercentChange7d)),  // Escaping percentage
		formatPriceRoundedUS(math.Round(*usdQuote.Volume24h)),
		formatPriceRoundedUS(math.Round(*usdQuote.MarketCap)),
		func() string {
			if details.CirculatingSupply != nil {
				return formatSupplyUS(*details.CirculatingSupply)
			}
			return "N/A"
		}(),
		*details.Symbol,
		formatSupplyUS(*details.TotalSupply),
		*details.Symbol,
		escapeMarkdownV2(*details.Name),
		*details.ID,
	)

	chartData, err := renderChart(c, tickers)
	if err != nil {
		return nil, "", err
	}

	cacheSet(argument, chartData, caption, 5*time.Minute)

	return chartData, caption, nil
}

func renderChart(c *coinpaprika.Coin, tickers []*coinpaprika.TickerHistorical) ([]byte, error) {
	var times []*time.Time
	var prices []*float64

	for _, t := range tickers {
		times = append(times, t.Timestamp)
		prices = append(prices, t.Price)
	}

	priceValues := [][]float64{{}}
	for _, price := range prices {
		priceValues[0] = append(priceValues[0], *price)
	}

	xLabels := []string{}
	var lastDay string
	for _, t := range times {
		currentDay := (*t).Format("02-Jan")
		if currentDay != lastDay {
			xLabels = append(xLabels, currentDay)
			lastDay = currentDay
		} else {
			xLabels = append(xLabels, "-")
		}
	}

	if len(xLabels) != len(priceValues[0]) {
		return nil, errors.New("mismatch between number of labels and data points")
	}

	minPrice, maxPrice := getMinMax(prices)
	padding := (maxPrice - minPrice) * 0.1
	minValue := minPrice - padding
	maxValue := maxPrice + padding

	// Render chart with the specified options
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
				Theme: nil,
				Text:  fmt.Sprintf("%s 7 days price chart (%s) - CoinPaprika", *c.Name, *c.Symbol),
				Left:  "center", // Centered title
				Top:   "20px",   // Adds more space from Y-axis
			}

			opt.ValueFormatter = func(v float64) string {
				return formatPriceUS(v, false)
			}

			opt.XAxis = chart.XAxisOption{
				Data:        xLabels,
				BoundaryGap: BoolPtr(false),
				FontSize:    12,
				FontColor:   chart.Color{R: 200, G: 200, B: 200, A: 255},
				Show:        BoolPtr(true),
				StrokeColor: chart.Color{R: 0, G: 0, B: 0, A: 255}, // Set a visible stroke color for X-axis
			}

			opt.YAxisOptions = []chart.YAxisOption{
				{
					Min:           &minValue,
					Max:           &maxValue,
					FontSize:      12,
					FontColor:     chart.Color{R: 200, G: 200, B: 200, A: 255},
					Position:      "left",
					SplitLineShow: BoolPtr(true), // Horizontal lines across chart
					Show:          BoolPtr(true),
				},
			}
		},
	)

	if err != nil {
		return nil, err
	}

	buf, err := p.Bytes()
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func getMinMax(prices []*float64) (min, max float64) {
	if len(prices) == 0 {
		return 0, 1
	}

	min, max = *prices[0], *prices[0]
	for _, price := range prices {
		if *price < min {
			min = *price
		}
		if *price > max {
			max = *price
		}
	}
	return min, max
}

func getPriceFormat(_, maxPrice float64) string {
	if maxPrice >= 1000 {
		return "%.0f"
	} else if maxPrice >= 1 {
		return "%.2f"
	} else if maxPrice >= 0.01 {
		return "%.4f"
	}
	return "$%.8f"
}

func BoolPtr(b bool) *bool {
	return &b
}
