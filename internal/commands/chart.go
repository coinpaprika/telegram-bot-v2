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
	// Define CoinPaprika's red-centric color scheme for series
	coinPaprikaSeriesColors := []drawing.Color{
		{R: 211, G: 47, B: 47, A: 255}, // #D32F2F -> Primary Red
		{R: 255, G: 82, B: 82, A: 255}, // #FF5252 -> Lighter Red
		{R: 183, G: 28, B: 28, A: 255}, // #B71C1C -> Dark Red
		{R: 239, G: 83, B: 80, A: 255}, // #EF5350 -> Red with some pinkish tone
	}

	// Add a new "coinpaprika" theme to the charts package
	chart.AddTheme(
		"coinpaprika",
		chart.ThemeOption{
			IsDarkMode: false, // Light mode theme
			AxisStrokeColor: chart.Color{
				R: 0, G: 0, B: 0, A: 255, // Sharp black for axis lines
			},
			AxisSplitLineColor: chart.Color{
				R: 200, G: 200, B: 200, A: 255, // Light grey for grid/split lines
			},
			BackgroundColor: chart.Color{
				R: 255, G: 255, B: 255, A: 255, // Pure white background
			},
			TextColor: chart.Color{
				R: 0, G: 0, B: 0, A: 255, // Sharp black for text
			},
			SeriesColors: coinPaprikaSeriesColors, // Apply the defined series colors
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
		return nil, fmt.Sprintf("This coin is not actively traded and doesn't have current price \n"+
			"For more details visit [CoinPaprika]https://coinpaprika\\.com/coin/%s🌶", *c.ID), nil
	}

	chartData, err := renderChart(tickers)
	if err != nil {
		return nil, "", err
	}

	cacheSet(argument, chartData, *c.Name, 5*time.Minute)

	return chartData, fmt.Sprintf(
		"%s on [CoinPaprika](https://coinpaprika.com/coin/%s)🌶/ Use this [Bot](https://github.com/coinpaprika/telegram-bot-v2/blob/main/README.md)",
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
		return nil, fmt.Sprintf("This coin is not actively traded and does not have current price \n"+
			"For more details visit [CoinPaprika](https://coinpaprika.com/coin/%s)🌶", *c.ID), nil
	}

	usdQuote := details.Quotes["USD"]

	caption := fmt.Sprintf(
		"*%s*\n"+
			"Price:  $%s\n"+
			"1h price change: %s%%\n"+
			"24h price change: %s%%\n"+
			"7d price change: %s%%\n"+
			"Vol:  $%s \n"+
			"MCap:  $%s\n"+
			"Circ\\. Supply:  %s *%s*\n"+
			"Total Supply:  %s *%s*\n"+
			"%s on [CoinPaprika](https://coinpaprika.com/coin/%s)🌶",
		escapeMarkdownV2(*details.Name),
		formatPriceUS(*usdQuote.Price),
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

	chartData, err := renderChart(tickers)
	if err != nil {
		return nil, "", err
	}

	cacheSet(argument, chartData, caption, 5*time.Minute)

	return chartData, caption, nil
}

func renderChart(tickers []*coinpaprika.TickerHistorical) ([]byte, error) {
	var times []*time.Time
	var prices []*float64

	// Extract timestamps and prices from the tickers
	for _, t := range tickers {
		times = append(times, t.Timestamp)
		prices = append(prices, t.Price)
	}

	// Create price value slices for chart rendering
	priceValues := [][]float64{{}}
	for _, price := range prices {
		priceValues[0] = append(priceValues[0], *price)
	}

	// Generate x-axis labels for each unique day
	xLabels := []string{}
	var lastDay string
	for _, t := range times {
		currentDay := (*t).Format("02-Jan")
		if currentDay != lastDay {
			xLabels = append(xLabels, currentDay)
			lastDay = currentDay
		} else {
			xLabels = append(xLabels, "-") // Use an empty string for same-day points
		}
	}

	// Validate that the number of xLabels and price values match
	if len(xLabels) != len(priceValues[0]) {
		return nil, errors.New("mismatch between number of labels and data points")
	}

	// Calculate the min and max prices and add padding
	minPrice, maxPrice := getMinMax(prices)
	padding := (maxPrice - minPrice) * 0.1
	minValue := minPrice - padding
	maxValue := maxPrice + padding

	// Set the price format dynamically based on the price range
	priceFormat := getPriceFormat(minPrice, maxPrice)

	// Create the line chart with the labels and price values
	p, err := chart.LineRender(
		priceValues,
		chart.TitleTextOptionFunc("CoinPaprika"),
		chart.ThemeOptionFunc("coinpaprika"),
		chart.WidthOptionFunc(1200),
		chart.LegendLabelsOptionFunc([]string{""}),

		func(opt *chart.ChartOption) {
			opt.FillArea = true
			opt.LineStrokeWidth = 2.0

			showSymbol := true
			opt.SymbolShow = &showSymbol

			opt.ValueFormatter = func(v float64) string {
				return fmt.Sprintf(priceFormat, v)
			}

			// Configure the X-axis with tighter control over boundary gap and alignment
			opt.XAxis = chart.XAxisOption{
				Data:        xLabels,
				BoundaryGap: BoolPtr(false), // Set BoundaryGap to false for tight alignment
				FontSize:    12,             // Keep font size readable
				FontColor:   drawing.Color{R: 0, G: 0, B: 0, A: 255},
				Show:        BoolPtr(true),
				LabelOffset: chart.Box{
					Top:   15, // Increase offset to push x-axis labels down
					Left:  20, // Add padding to the left
					Right: 20, // Add padding to the right
				},
			}

			// Customize the Y-axis for dynamic price range
			opt.YAxisOptions = []chart.YAxisOption{
				{
					Min:           &minValue,
					Max:           &maxValue,
					FontSize:      12,
					FontColor:     chart.Color{R: 0, G: 0, B: 0, A: 255},
					Position:      "left",
					SplitLineShow: BoolPtr(true),
				},
			}
		},
	)

	if err != nil {
		return nil, err
	}

	// Render the chart as a byte array
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
	if maxPrice >= 1 {
		return "$%.2f"
	}
	if maxPrice >= 0.01 {
		return "$%.4f"
	}
	return "$%.8f"
}

func BoolPtr(b bool) *bool {
	return &b
}
