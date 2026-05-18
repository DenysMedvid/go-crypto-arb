package arbitrage

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
)

type SignalEngine struct {
	mu         sync.Mutex
	maxPoints  int
	priceByKey map[string][]pricePoint
}

type pricePoint struct {
	price decimal.Decimal
	at    time.Time
}

func NewSignalEngine(maxPoints int) *SignalEngine {
	if maxPoints <= 1 {
		maxPoints = 30
	}
	return &SignalEngine{
		maxPoints:  maxPoints,
		priceByKey: make(map[string][]pricePoint),
	}
}

func (e *SignalEngine) Update(cfg config.Config, tickers []exchange.Ticker) []RelatedAssetGroupSignal {
	if !cfg.Signals.RelatedAssets.Enabled {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	preferred := pickPreferredStableQuotes(cfg)
	var out []RelatedAssetGroupSignal
	for _, group := range cfg.Signals.RelatedAssets.Groups {
		var signals []RelatedAssetSignal
		var total decimal.Decimal
		for _, asset := range group.Assets {
			ticker, ok := findAssetTicker(asset, preferred, tickers)
			if !ok {
				continue
			}
			key := group.Name + "|" + ticker.Exchange + "|" + ticker.Symbol
			mid := ticker.Bid.Add(ticker.Ask).Div(decimal.NewFromInt(2))
			e.priceByKey[key] = append(e.priceByKey[key], pricePoint{price: mid, at: now})
			if len(e.priceByKey[key]) > e.maxPoints {
				e.priceByKey[key] = e.priceByKey[key][len(e.priceByKey[key])-e.maxPoints:]
			}
			points := e.priceByKey[key]
			if len(points) < 2 || points[0].price.IsZero() {
				continue
			}
			change := points[len(points)-1].price.Sub(points[0].price).Div(points[0].price).Mul(decimal.NewFromInt(100))
			signals = append(signals, RelatedAssetSignal{
				Symbol:        ticker.Symbol,
				Asset:         ticker.BaseAsset,
				Exchange:      ticker.Exchange,
				ChangePercent: change,
			})
			total = total.Add(change)
		}
		if len(signals) == 0 {
			continue
		}
		average := total.Div(decimal.NewFromInt(int64(len(signals))))
		for i := range signals {
			signals[i].DivergencePercent = signals[i].ChangePercent.Sub(average)
		}
		sort.Slice(signals, func(i, j int) bool {
			return absDecimal(signals[i].DivergencePercent).GreaterThan(absDecimal(signals[j].DivergencePercent))
		})
		out = append(out, RelatedAssetGroupSignal{
			Group:        group.Name,
			Assets:       limitSignals(signals, cfg.Signals.RelatedAssets.MaxResults),
			GroupAverage: average,
			CalculatedAt: now,
		})
	}
	return out
}

func findAssetTicker(asset string, preferredQuotes []string, tickers []exchange.Ticker) (exchange.Ticker, bool) {
	asset = exchange.NormalizeAsset(asset)
	for _, quote := range preferredQuotes {
		for _, ticker := range tickers {
			if ticker.MarketType == exchange.MarketSpot && ticker.BaseAsset == asset && ticker.QuoteAsset == quote && validTicker(ticker) {
				return ticker, true
			}
		}
	}
	return exchange.Ticker{}, false
}

func pickPreferredStableQuotes(cfg config.Config) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, list := range [][]string{cfg.Assets.StableBases, cfg.Assets.QuoteAssets} {
		for _, asset := range list {
			asset = exchange.NormalizeAsset(asset)
			if _, ok := seen[asset]; ok {
				continue
			}
			seen[asset] = struct{}{}
			out = append(out, asset)
		}
	}
	return out
}

func absDecimal(value decimal.Decimal) decimal.Decimal {
	f, _ := value.Float64()
	if math.Signbit(f) {
		return value.Neg()
	}
	return value
}

func limitSignals(in []RelatedAssetSignal, max int) []RelatedAssetSignal {
	if max <= 0 || len(in) <= max {
		return in
	}
	return in[:max]
}
