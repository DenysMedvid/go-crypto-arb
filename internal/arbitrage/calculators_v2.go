package arbitrage

import (
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
)

func CalculateTriangularV2(cfg config.Config, tickers []exchange.Ticker, orderBooks []exchange.OrderBook) []TriangularOpportunityV2 {
	if !cfg.Arbitrage.Triangular.Enabled {
		return nil
	}
	if !cfg.Arbitrage.Triangular.UseOrderBookDepth {
		orderBooks = nil
	}
	books := orderBooksByExchange(orderBooks, tickers, exchange.MarketSpot)
	var out []TriangularOpportunityV2
	now := time.Now()
	for exchangeName, exchangeBooks := range books {
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
					for _, start := range simulationTradeSizes(cfg) {
						end, legs, complete := runCycleV2(start, cycle, exchangeBooks, fee)
						if end.IsZero() && len(legs) == 0 {
							continue
						}
						profit := ProfitPercent(start, end)
						if belowOptionalProfitFloor(profit, cfg.Arbitrage.Triangular.MinProfitPercent.DecimalValue()) {
							continue
						}
						worst, maxSlip := worstLeg(legs)
						complete = complete && withinSlippageLimit(cfg, maxSlip)
						out = append(out, TriangularOpportunityV2{
							Provider:           strings.ToLower(exchangeName),
							Exchange:           exchangeName,
							StrategyTitle:      cfg.Strategies.CryptoTriangular.Title,
							AssetClass:         "crypto",
							Cycle:              cycle,
							StartAsset:         stable,
							StartAmount:        start,
							EndAmount:          end,
							NetProfitPercent:   profit,
							CompleteFill:       complete,
							WorstLeg:           worst,
							MaxSlippagePercent: maxSlip,
							Legs:               legs,
							Status:             statusFromPercent(profit, "PROFIT", "LOSS"),
							UpdatedAt:          now,
						})
					}
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].NetProfitPercent.GreaterThan(out[j].NetProfitPercent)
	})
	return limitTriangularV2(out, cfg.Arbitrage.Triangular.MaxResults)
}

func CalculateCrossExchangeV2(cfg config.Config, tickers []exchange.Ticker, orderBooks []exchange.OrderBook) []CrossExchangeOpportunityV2 {
	if !cfg.Arbitrage.CrossExchange.Enabled {
		return nil
	}
	if !cfg.Arbitrage.CrossExchange.UseOrderBookDepth {
		orderBooks = nil
	}
	books := orderBookLookup(orderBooks, tickers, exchange.MarketSpot)
	bySymbol := make(map[string][]exchange.OrderBook)
	for _, book := range books {
		if book.MarketType == exchange.MarketSpot && len(book.Asks) > 0 && len(book.Bids) > 0 {
			bySymbol[book.Symbol] = append(bySymbol[book.Symbol], book)
		}
	}
	var out []CrossExchangeOpportunityV2
	now := time.Now()
	for symbol, candidates := range bySymbol {
		for _, buyBook := range candidates {
			for _, sellBook := range candidates {
				if buyBook.Exchange == sellBook.Exchange {
					continue
				}
				for _, tradeSize := range simulationTradeSizes(cfg) {
					buy := SimulateBuyWithQuote(buyBook, tradeSize, spotFee(cfg, buyBook.Exchange))
					sell := SimulateSellBase(sellBook, buy.FilledBaseQty, spotFee(cfg, sellBook.Exchange))
					net := ProfitPercent(tradeSize, sell.ReceivedQuoteValue)
					if belowOptionalProfitFloor(net, cfg.Arbitrage.CrossExchange.MinProfitPercent.DecimalValue()) {
						continue
					}
					complete := buy.CompleteFill && sell.CompleteFill && withinSlippageLimit(cfg, buy.SlippagePercent) && withinSlippageLimit(cfg, sell.SlippagePercent)
					out = append(out, CrossExchangeOpportunityV2{
						StrategyTitle:       cfg.Strategies.CrossExchange.Title,
						Symbol:              symbol,
						BuyProvider:         strings.ToLower(buyBook.Exchange),
						SellProvider:        strings.ToLower(sellBook.Exchange),
						BuyExchange:         buyBook.Exchange,
						SellExchange:        sellBook.Exchange,
						TradeSize:           tradeSize,
						BuyAveragePrice:     buy.AveragePrice,
						SellAveragePrice:    sell.AveragePrice,
						BuySlippagePercent:  buy.SlippagePercent,
						SellSlippagePercent: sell.SlippagePercent,
						BuyFeeAmount:        buy.FeeAmount,
						SellFeeAmount:       sell.FeeAmount,
						NetProfitPercent:    net,
						CompleteFill:        complete,
						Status:              statusFromPercent(net, "PROFIT", "LOSS"),
						UpdatedAt:           now,
					})
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].NetProfitPercent.GreaterThan(out[j].NetProfitPercent)
	})
	return limitCrossV2(out, cfg.Arbitrage.CrossExchange.MaxResults)
}

func CalculateSpotFuturesV2(cfg config.Config, spotTickers []exchange.Ticker, futuresTickers []exchange.Ticker, orderBooks []exchange.OrderBook, fundingRates []exchange.FundingRate) []SpotFuturesOpportunityV2 {
	if !cfg.Arbitrage.SpotFutures.Enabled {
		return nil
	}
	if !cfg.Arbitrage.SpotFutures.UseOrderBookDepth {
		orderBooks = nil
	}
	books := orderBookLookup(orderBooks, append(append([]exchange.Ticker(nil), spotTickers...), futuresTickers...), "")
	fundingByExchangeSymbol := make(map[string]exchange.FundingRate)
	for _, rate := range fundingRates {
		fundingByExchangeSymbol[rate.Exchange+"|"+rate.Symbol] = rate
	}
	var out []SpotFuturesOpportunityV2
	now := time.Now()
	for key, spotBook := range books {
		if spotBook.MarketType != exchange.MarketSpot {
			continue
		}
		futuresBook, ok := books["futures|"+spotBook.Exchange+"|"+spotBook.Symbol]
		if !ok {
			_ = key
			continue
		}
		for _, tradeSize := range simulationTradeSizes(cfg) {
			spotBuy := SimulateBuyWithQuote(spotBook, tradeSize, spotFee(cfg, spotBook.Exchange))
			futuresSell := SimulateSellBase(futuresBook, spotBuy.FilledBaseQty, futuresFee(cfg, spotBook.Exchange))
			basis := decimal.Zero
			if !spotBuy.AveragePrice.IsZero() {
				basis = futuresSell.AveragePrice.Sub(spotBuy.AveragePrice).Div(spotBuy.AveragePrice).Mul(decimal.NewFromInt(100))
			}
			net := ProfitPercent(tradeSize, futuresSell.ReceivedQuoteValue)
			funding := decimal.Zero
			if rate, ok := fundingByExchangeSymbol[spotBook.Exchange+"|"+spotBook.Symbol]; ok {
				funding = rate.Rate
				if cfg.Arbitrage.SpotFutures.IncludeFundingRate {
					net = net.Add(funding.Mul(decimal.NewFromInt(100)))
				}
			}
			if belowOptionalProfitFloor(basis, cfg.Arbitrage.SpotFutures.MinBasisPercent.DecimalValue()) {
				continue
			}
			complete := spotBuy.CompleteFill && futuresSell.CompleteFill && withinSlippageLimit(cfg, spotBuy.SlippagePercent) && withinSlippageLimit(cfg, futuresSell.SlippagePercent)
			out = append(out, SpotFuturesOpportunityV2{
				StrategyTitle:           cfg.Strategies.CryptoSpotFutures.Title,
				Provider:                strings.ToLower(spotBook.Exchange),
				Exchange:                spotBook.Exchange,
				Symbol:                  spotBook.Symbol,
				TradeSize:               tradeSize,
				SpotAverageBuyPrice:     spotBuy.AveragePrice,
				FuturesAverageSellPrice: futuresSell.AveragePrice,
				SpotSlippagePercent:     spotBuy.SlippagePercent,
				FuturesSlippagePercent:  futuresSell.SlippagePercent,
				SpotFeeAmount:           spotBuy.FeeAmount,
				FuturesFeeAmount:        futuresSell.FeeAmount,
				BasisPercent:            basis,
				FundingRate:             funding,
				NetEstimatePercent:      net,
				CompleteFill:            complete,
				Status:                  statusFromPercent(net, "WATCH", "NO"),
				UpdatedAt:               now,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].NetEstimatePercent.GreaterThan(out[j].NetEstimatePercent)
	})
	return limitSpotFuturesV2(out, cfg.Arbitrage.SpotFutures.MaxResults)
}

func runCycleV2(start decimal.Decimal, cycle []string, books map[string]exchange.OrderBook, fee decimal.Decimal) (decimal.Decimal, []LegSimulation, bool) {
	amount := start
	complete := true
	legs := make([]LegSimulation, 0, len(cycle)-1)
	for i := 0; i < len(cycle)-1; i++ {
		leg, output, ok := simulateConversion(amount, cycle[i], cycle[i+1], books, fee)
		if !ok {
			return decimal.Zero, legs, false
		}
		if !leg.CompleteFill {
			complete = false
		}
		legs = append(legs, leg)
		amount = output
	}
	return amount, legs, complete
}

func simulateConversion(amount decimal.Decimal, from, to string, books map[string]exchange.OrderBook, fee decimal.Decimal) (LegSimulation, decimal.Decimal, bool) {
	from = exchange.NormalizeAsset(from)
	to = exchange.NormalizeAsset(to)
	if book, ok := findBook(books, from, to); ok {
		sim := SimulateSellBase(book, amount, fee)
		leg := LegSimulation{
			FromAsset:       from,
			ToAsset:         to,
			Symbol:          book.Symbol,
			Side:            TradeSell,
			InputAmount:     amount,
			OutputAmount:    sim.ReceivedQuoteValue,
			AveragePrice:    sim.AveragePrice,
			FeeAmount:       sim.FeeAmount,
			SlippagePercent: sim.SlippagePercent,
			CompleteFill:    sim.CompleteFill,
		}
		return leg, sim.ReceivedQuoteValue, true
	}
	if book, ok := findBook(books, to, from); ok {
		sim := SimulateBuyWithQuote(book, amount, fee)
		leg := LegSimulation{
			FromAsset:       from,
			ToAsset:         to,
			Symbol:          book.Symbol,
			Side:            TradeBuy,
			InputAmount:     amount,
			OutputAmount:    sim.FilledBaseQty,
			AveragePrice:    sim.AveragePrice,
			FeeAmount:       sim.FeeAmount,
			SlippagePercent: sim.SlippagePercent,
			CompleteFill:    sim.CompleteFill,
		}
		return leg, sim.FilledBaseQty, true
	}
	return LegSimulation{}, decimal.Zero, false
}

func findBook(books map[string]exchange.OrderBook, base, quote string) (exchange.OrderBook, bool) {
	for _, book := range books {
		if book.BaseAsset == base && book.QuoteAsset == quote && len(book.Bids) > 0 && len(book.Asks) > 0 {
			return book, true
		}
	}
	return exchange.OrderBook{}, false
}

func orderBooksByExchange(orderBooks []exchange.OrderBook, tickers []exchange.Ticker, marketType exchange.MarketType) map[string]map[string]exchange.OrderBook {
	lookup := orderBookLookup(orderBooks, tickers, marketType)
	out := make(map[string]map[string]exchange.OrderBook)
	for _, book := range lookup {
		if marketType != "" && book.MarketType != marketType {
			continue
		}
		if _, ok := out[book.Exchange]; !ok {
			out[book.Exchange] = make(map[string]exchange.OrderBook)
		}
		out[book.Exchange][book.Symbol] = book
	}
	return out
}

func orderBookLookup(orderBooks []exchange.OrderBook, tickers []exchange.Ticker, marketType exchange.MarketType) map[string]exchange.OrderBook {
	out := make(map[string]exchange.OrderBook)
	for _, book := range orderBooks {
		if marketType != "" && book.MarketType != marketType {
			continue
		}
		if len(book.Bids) == 0 && len(book.Asks) == 0 {
			continue
		}
		out[string(book.MarketType)+"|"+book.Exchange+"|"+book.Symbol] = exchange.NormalizeOrderBook(book, 0)
	}
	for _, ticker := range tickers {
		if marketType != "" && ticker.MarketType != marketType {
			continue
		}
		key := string(ticker.MarketType) + "|" + ticker.Exchange + "|" + ticker.Symbol
		if _, ok := out[key]; ok {
			continue
		}
		if !validTicker(ticker) {
			continue
		}
		out[key] = exchange.OrderBook{
			Provider:     providerName(ticker),
			Exchange:     ticker.Exchange,
			Broker:       ticker.Broker,
			Symbol:       ticker.Symbol,
			InstrumentID: ticker.InstrumentID,
			BaseAsset:    ticker.BaseAsset,
			QuoteAsset:   ticker.QuoteAsset,
			MarketType:   ticker.MarketType,
			AssetClass:   ticker.AssetClass,
			Bids:         []exchange.OrderBookLevel{{Price: ticker.Bid, Quantity: decimal.NewFromInt(1)}},
			Asks:         []exchange.OrderBookLevel{{Price: ticker.Ask, Quantity: decimal.NewFromInt(1)}},
			UpdatedAt:    ticker.UpdatedAt,
			LimitedDepth: true,
		}
	}
	return out
}

func providerName(ticker exchange.Ticker) string {
	if ticker.Provider != "" {
		return ticker.Provider
	}
	if ticker.Exchange != "" {
		return ticker.Exchange
	}
	return ticker.Broker
}

func simulationTradeSizes(cfg config.Config) []decimal.Decimal {
	if !cfg.Simulation.Enabled {
		return []decimal.Decimal{defaultStartAmount}
	}
	out := make([]decimal.Decimal, 0, len(cfg.Simulation.TradeSizes))
	for _, size := range cfg.Simulation.TradeSizes {
		if size.DecimalValue().GreaterThan(decimal.Zero) {
			out = append(out, size.DecimalValue())
		}
	}
	if len(out) == 0 {
		return []decimal.Decimal{defaultStartAmount}
	}
	return out
}

func worstLeg(legs []LegSimulation) (string, decimal.Decimal) {
	var worst string
	var maxSlip decimal.Decimal
	for _, leg := range legs {
		if !leg.CompleteFill && worst == "" {
			worst = leg.FromAsset + "->" + leg.ToAsset
		}
		if leg.SlippagePercent.GreaterThan(maxSlip) {
			maxSlip = leg.SlippagePercent
			worst = leg.FromAsset + "->" + leg.ToAsset
		}
	}
	return worst, maxSlip
}

func withinSlippageLimit(cfg config.Config, value decimal.Decimal) bool {
	limit := cfg.Simulation.MaxSlippagePercent.DecimalValue()
	return limit.IsZero() || value.LessThanOrEqual(limit)
}

func limitTriangularV2(in []TriangularOpportunityV2, max int) []TriangularOpportunityV2 {
	if max <= 0 || len(in) <= max {
		return in
	}
	return in[:max]
}

func limitCrossV2(in []CrossExchangeOpportunityV2, max int) []CrossExchangeOpportunityV2 {
	if max <= 0 || len(in) <= max {
		return in
	}
	return in[:max]
}

func limitSpotFuturesV2(in []SpotFuturesOpportunityV2, max int) []SpotFuturesOpportunityV2 {
	if max <= 0 || len(in) <= max {
		return in
	}
	return in[:max]
}
