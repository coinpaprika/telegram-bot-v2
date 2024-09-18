package commands

import (
	"fmt"
	"github.com/coinpaprika/coinpaprika-api-go-client/v2/coinpaprika"
	"github.com/pkg/errors"
	"github.com/vicanso/go-charts/v2"
	"log"
	"time"
)

func init() {
	// Define CoinPaprika's red-centric color scheme for series
	coinPaprikaSeriesColors := []charts.Color{
		{R: 211, G: 47, B: 47, A: 255}, // #D32F2F -> Primary Red
		{R: 255, G: 82, B: 82, A: 255}, // #FF5252 -> Lighter Red
		{R: 183, G: 28, B: 28, A: 255}, // #B71C1C -> Dark Red
		{R: 239, G: 83, B: 80, A: 255}, // #EF5350 -> Red with some pinkish tone
	}

	// Add a new "coinpaprika" theme to the charts package
	charts.AddTheme(
		"coinpaprika",
		charts.ThemeOption{
			IsDarkMode: false, // Light mode theme
			AxisStrokeColor: charts.Color{
				R: 0, G: 0, B: 0, A: 255, // Sharp black for axis lines
			},
			AxisSplitLineColor: charts.Color{
				R: 200, G: 200, B: 200, A: 255, // Light grey for grid/split lines
			},
			BackgroundColor: charts.Color{
				R: 255, G: 255, B: 255, A: 255, // Pure white background
			},
			TextColor: charts.Color{
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

	chartData, err := renderChart(tickers)
	if err != nil {
		return nil, "", err
	}

	cacheSet(argument, chartData, *c.Name, 5*time.Minute)

	return chartData, *c.Name, nil
}

func CommandChartWithTicker(argument string) ([]byte, string, error) {
	log.Printf("processing command ticker with argument :%s", argument)

	if cachedItem, found := cacheGet(argument); found {
		log.Printf("returning cached result for %s", argument)
		return cachedItem.ChartData, cachedItem.Caption, nil
	}

	c, tickers, _ := GetHistoricalTickersByQuery(argument)
	details, err := GetTicker(c)
	if err != nil {
		return nil, "", err
	}

	if details == nil || details.Quotes == nil {
		return nil, "", errors.New("missing ticker data")
	}

	usdQuote := details.Quotes["USD"]
	btcQuote := details.Quotes["BTC"]
	ethQuote := details.Quotes["ETH"]

	caption := fmt.Sprintf(
		"*%s Overview:*\n\n"+
			"▫️*Price:* \n  `%s` *USD* \n  `%s` *BTC* \n  `%s` *ETH*\n\n"+
			"▫️*Price Changes:*\n  *1h*: `%.2f%%` \n  *24h*: `%.2f%%` \n  *7d*: `%.2f%%`\n\n"+
			"▫️*Volume \\(24h\\):*\n  `%s` *USD*\n\n"+
			"▫️*Market Cap:*\n  `%s` *USD*\n\n"+
			"▫️*Circulating Supply:*\n  `%s` *%s*\n\n"+
			"▫️*Total Supply:*\n  `%s` *%s*\n\n"+
			"[See %s\\(%s\\) on CoinPaprika 🌶](https://coinpaprika.com/coin/%s)",
		*details.Name,
		formatPriceUS(*usdQuote.Price),
		formatPriceUS(*btcQuote.Price),
		formatPriceUS(*ethQuote.Price),
		*usdQuote.PercentChange1h,
		*usdQuote.PercentChange24h,
		*usdQuote.PercentChange7d,
		formatPriceUS(*usdQuote.Volume24h),
		formatPriceUS(*usdQuote.MarketCap),
		func() string {
			if details.CirculatingSupply != nil {
				return formatSupplyUS(*details.CirculatingSupply)
			}
			return "N/A"
		}(),
		*details.Symbol,
		formatSupplyUS(*details.TotalSupply),
		*details.Symbol,
		*details.Name,
		*details.Symbol,
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

	// Determine the interval (1d, 1h, or other) based on the time difference between two points
	var interval string
	if len(times) > 1 {
		timeDiff := times[1].Sub(*times[0])
		if timeDiff.Hours() >= 24 {
			interval = "1d"
		} else if timeDiff.Hours() >= 1 {
			interval = "1h"
		} else {
			interval = "other"
		}
	} else {
		interval = "default"
	}

	// Create labels for the X-axis based on the interval
	xLabels := []string{}
	for _, t := range times {
		switch interval {
		case "1d":
			xLabels = append(xLabels, (*t).Format("02-Jan"))
		case "1h":
			xLabels = append(xLabels, (*t).Format("15:04"))
		default:
			xLabels = append(xLabels, (*t).Format(time.RFC822))
		}
	}

	// Extract prices and create price value slices for chart rendering
	priceValues := [][]float64{{}}
	for _, price := range prices {
		priceValues[0] = append(priceValues[0], *price)
	}

	// Calculate the min and max prices and add a small padding for better visualization
	minPrice, maxPrice := getMinMax(prices)
	padding := (maxPrice - minPrice) * 0.05 // 5% padding
	minValue := minPrice - padding
	maxValue := maxPrice + padding

	// Set the price format dynamically based on the price range
	priceFormat := getPriceFormat(minPrice, maxPrice)

	p, err := charts.LineRender(
		priceValues,
		charts.TitleTextOptionFunc("Price over Time"),
		charts.XAxisDataOptionFunc(xLabels),
		charts.ThemeOptionFunc("coinpaprika"),
		charts.WidthOptionFunc(1200),
		charts.LegendLabelsOptionFunc([]string{"price"}),
		// Customize the Y-axis options with dynamically calculated min and max values
		func(opt *charts.ChartOption) {
			opt.FillArea = true
			opt.LineStrokeWidth = 2.0

			showSymbol := true
			opt.SymbolShow = &showSymbol

			opt.ValueFormatter = func(v float64) string {
				return fmt.Sprintf(priceFormat, v)
			}

			opt.YAxisOptions = []charts.YAxisOption{
				{
					Min:           &minValue,
					Max:           &maxValue,
					FontSize:      12,
					FontColor:     charts.Color{R: 0, G: 0, B: 0, A: 255},
					Position:      "left",
					SplitLineShow: BoolPtr(true),
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
