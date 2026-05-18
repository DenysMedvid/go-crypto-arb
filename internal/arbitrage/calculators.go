package arbitrage

import (
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
)

var defaultStartAmount = decimal.NewFromInt(1000)

func CalculateTriangular(cfg config.Config, tickers []exchange.Ticker) []TriangularOpportunity {
	if !cfg.Arbitrage.Triangular.Enabled {
		return nil
	}
	byExchange := make(map[string][]exchange.Ticker)
	for _, ticker := range tickers {
		if validTicker(ticker) && ticker.MarketType == exchange.MarketSpot {
			byExchange[ticker.Exchange] = append(byExchange[ticker.Exchange], ticker)
		}
	}

	var out []TriangularOpportunity
	now := time.Now()
	for exchangeName, exchangeTickers := range byExchange {
		fee := spotFee(cfg, exchangeName)
		for _, stable := range cfg.Assets.StableBases {
			stable = exchange.NormalizeAsset(stable)
			for _, first := range cfg.Assets.Preferred {
				first = exchange.NormalizeAsset(first)
				if first == stable {
					continue
				}
				for _, second := range cfg.Assets.Preferred {
					second = exchange.NormalizeAsset(second)
					if second == stable || second == first {
						continue
					}
					cycle := []string{stable, first, second, stable}
					end, ok := runCycle(defaultStartAmount, cycle, exchangeTickers, fee)
					if !ok {
						continue
					}
					profit := ProfitPercent(defaultStartAmount, end)
					if belowOptionalProfitFloor(profit, cfg.Arbitrage.Triangular.MinProfitPercent.DecimalValue()) {
						continue
					}
					out = append(out, TriangularOpportunity{
						Exchange:      exchangeName,
						Cycle:         cycle,
						StartAsset:    stable,
						StartAmount:   defaultStartAmount,
						EndAmount:     end,
						ProfitPercent: profit,
						Status:        statusFromPercent(profit, "PROFIT", "LOSS"),
						CalculatedAt:  now,
					})
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ProfitPercent.GreaterThan(out[j].ProfitPercent)
	})
	return limitTriangular(out, cfg.Arbitrage.Triangular.MaxResults)
}

func CalculateCrossExchange(cfg config.Config, tickers []exchange.Ticker) []CrossExchangeOpportunity {
	if !cfg.Arbitrage.CrossExchange.Enabled {
		return nil
	}
	bySymbol := make(map[string][]exchange.Ticker)
	for _, ticker := range tickers {
		if validTicker(ticker) && ticker.MarketType == exchange.MarketSpot {
			bySymbol[ticker.Symbol] = append(bySymbol[ticker.Symbol], ticker)
		}
	}

	var out []CrossExchangeOpportunity
	now := time.Now()
	for symbol, candidates := range bySymbol {
		for _, buy := range candidates {
			for _, sell := range candidates {
				if buy.Exchange == sell.Exchange {
					continue
				}
				buyFee := spotFee(cfg, buy.Exchange)
				sellFee := spotFee(cfg, sell.Exchange)
				baseAmount := ApplyTakerFee(defaultStartAmount.Div(buy.Ask), buyFee)
				endAmount := ApplyTakerFee(baseAmount.Mul(sell.Bid), sellFee)
				net := ProfitPercent(defaultStartAmount, endAmount)
				if belowOptionalProfitFloor(net, cfg.Arbitrage.CrossExchange.MinProfitPercent.DecimalValue()) {
					continue
				}
				out = append(out, CrossExchangeOpportunity{
					Symbol:       symbol,
					BuyExchange:  buy.Exchange,
					SellExchange: sell.Exchange,
					BuyAsk:       buy.Ask,
					SellBid:      sell.Bid,
					BuyFee:       buyFee,
					SellFee:      sellFee,
					NetPercent:   net,
					Status:       statusFromPercent(net, "PROFIT", "LOSS"),
					CalculatedAt: now,
				})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].NetPercent.GreaterThan(out[j].NetPercent)
	})
	return limitCross(out, cfg.Arbitrage.CrossExchange.MaxResults)
}

func CalculateSpotFutures(cfg config.Config, spotTickers []exchange.Ticker, futuresTickers []exchange.Ticker, fundingRates []exchange.FundingRate) []SpotFuturesOpportunity {
	if !cfg.Arbitrage.SpotFutures.Enabled {
		return nil
	}
	spotByExchangeSymbol := make(map[string]exchange.Ticker)
	for _, ticker := range spotTickers {
		if validTicker(ticker) && ticker.MarketType == exchange.MarketSpot {
			spotByExchangeSymbol[ticker.Exchange+"|"+ticker.Symbol] = ticker
		}
	}
	fundingByExchangeSymbol := make(map[string]exchange.FundingRate)
	for _, rate := range fundingRates {
		fundingByExchangeSymbol[rate.Exchange+"|"+rate.Symbol] = rate
	}

	var out []SpotFuturesOpportunity
	now := time.Now()
	for _, futures := range futuresTickers {
		if !validTicker(futures) || futures.MarketType != exchange.MarketFutures {
			continue
		}
		spot, ok := spotByExchangeSymbol[futures.Exchange+"|"+futures.Symbol]
		if !ok {
			continue
		}
		basis := futures.Bid.Sub(spot.Ask).Div(spot.Ask).Mul(decimal.NewFromInt(100))
		if belowOptionalProfitFloor(basis, cfg.Arbitrage.SpotFutures.MinBasisPercent.DecimalValue()) {
			continue
		}
		spotFee := spotFee(cfg, futures.Exchange)
		futuresFee := futuresFee(cfg, futures.Exchange)
		baseAmount := ApplyTakerFee(defaultStartAmount.Div(spot.Ask), spotFee)
		endAmount := ApplyTakerFee(baseAmount.Mul(futures.Bid), futuresFee)
		net := ProfitPercent(defaultStartAmount, endAmount)
		funding := decimal.Zero
		if rate, ok := fundingByExchangeSymbol[futures.Exchange+"|"+futures.Symbol]; ok {
			funding = rate.Rate
			if cfg.Arbitrage.SpotFutures.IncludeFundingRate {
				net = net.Add(funding.Mul(decimal.NewFromInt(100)))
			}
		}
		out = append(out, SpotFuturesOpportunity{
			Symbol:       futures.Symbol,
			Exchange:     futures.Exchange,
			SpotAsk:      spot.Ask,
			FuturesBid:   futures.Bid,
			BasisPercent: basis,
			FundingRate:  funding,
			NetEstimate:  net,
			Status:       statusFromPercent(net, "WATCH", "NO"),
			CalculatedAt: now,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].NetEstimate.GreaterThan(out[j].NetEstimate)
	})
	return limitSpotFutures(out, cfg.Arbitrage.SpotFutures.MaxResults)
}

func runCycle(start decimal.Decimal, cycle []string, tickers []exchange.Ticker, fee decimal.Decimal) (decimal.Decimal, bool) {
	amount := start
	for i := 0; i < len(cycle)-1; i++ {
		var ok bool
		// Direction matters: quote->base buys at ask; base->quote sells at bid.
		amount, ok = convert(amount, cycle[i], cycle[i+1], tickers, fee)
		if !ok {
			return decimal.Zero, false
		}
	}
	return amount, true
}

func convert(amount decimal.Decimal, from, to string, tickers []exchange.Ticker, fee decimal.Decimal) (decimal.Decimal, bool) {
	from = exchange.NormalizeAsset(from)
	to = exchange.NormalizeAsset(to)
	for _, ticker := range tickers {
		if ticker.BaseAsset == to && ticker.QuoteAsset == from && !ticker.Ask.IsZero() {
			return ApplyTakerFee(amount.Div(ticker.Ask), fee), true
		}
		if ticker.BaseAsset == from && ticker.QuoteAsset == to && !ticker.Bid.IsZero() {
			return ApplyTakerFee(amount.Mul(ticker.Bid), fee), true
		}
	}
	return decimal.Zero, false
}

func validTicker(t exchange.Ticker) bool {
	return exchange.ValidBidAsk(t.Bid, t.Ask)
}

func spotFee(cfg config.Config, exchangeName string) decimal.Decimal {
	if ex, ok := lookupExchangeConfig(cfg, exchangeName); ok {
		return ex.Fees.SpotTaker.DecimalValue()
	}
	return decimal.Zero
}

func futuresFee(cfg config.Config, exchangeName string) decimal.Decimal {
	if ex, ok := lookupExchangeConfig(cfg, exchangeName); ok {
		return ex.Fees.FuturesTaker.DecimalValue()
	}
	return decimal.Zero
}

func lookupExchangeConfig(cfg config.Config, exchangeName string) (config.ExchangeConfig, bool) {
	if ex, ok := cfg.Exchanges[exchangeName]; ok {
		return ex, true
	}
	for name, ex := range cfg.Exchanges {
		if strings.EqualFold(name, exchangeName) {
			return ex, true
		}
	}
	return config.ExchangeConfig{}, false
}

func belowOptionalProfitFloor(value decimal.Decimal, floor decimal.Decimal) bool {
	return floor.GreaterThan(decimal.Zero) && value.LessThan(floor)
}

func limitTriangular(in []TriangularOpportunity, max int) []TriangularOpportunity {
	if max <= 0 || len(in) <= max {
		return in
	}
	return in[:max]
}

func limitCross(in []CrossExchangeOpportunity, max int) []CrossExchangeOpportunity {
	if max <= 0 || len(in) <= max {
		return in
	}
	return in[:max]
}

func limitSpotFutures(in []SpotFuturesOpportunity, max int) []SpotFuturesOpportunity {
	if max <= 0 || len(in) <= max {
		return in
	}
	return in[:max]
}
