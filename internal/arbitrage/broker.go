package arbitrage

import (
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
)

func CalculateIBKRFXTriangular(cfg config.Config, tickers []exchange.Ticker, orderBooks []exchange.OrderBook) []TriangularOpportunityV2 {
	strategy := cfg.Strategies.IBKRFXTriangular
	if !strategy.Enabled {
		return nil
	}
	if !strategy.UseOrderBookDepth {
		orderBooks = nil
	}
	books := providerOrderBooks("ibkr", orderBookLookup(orderBooks, tickers, exchange.MarketSpot))
	fee := ibkrFXFee(cfg)
	var out []TriangularOpportunityV2
	now := time.Now()
	for _, cycle := range strategy.Cycles {
		if len(cycle) < 4 {
			continue
		}
		for _, start := range simulationTradeSizes(cfg) {
			end, legs, complete := runCycleV2(start, cycle, books, fee)
			if end.IsZero() && len(legs) == 0 {
				continue
			}
			profit := ProfitPercent(start, end)
			if belowOptionalProfitFloor(profit, strategy.MinProfitPercent.DecimalValue()) {
				continue
			}
			worst, maxSlip := worstLeg(legs)
			complete = complete && withinSlippageLimit(cfg, maxSlip)
			out = append(out, TriangularOpportunityV2{
				Provider:           "ibkr",
				Exchange:           "IBKR",
				StrategyTitle:      strategy.Title,
				AssetClass:         "fx",
				Cycle:              normalizeCycle(cycle),
				StartAsset:         exchange.NormalizeAsset(cycle[0]),
				StartAmount:        start,
				EndAmount:          end,
				NetProfitPercent:   profit,
				CompleteFill:       complete,
				WorstLeg:           worst,
				MaxSlippagePercent: maxSlip,
				Legs:               legs,
				Status:             statusFromPercent(profit, "WATCH", "NO"),
				UpdatedAt:          now,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].NetProfitPercent.GreaterThan(out[j].NetProfitPercent)
	})
	if strategy.MaxResults > 0 && len(out) > strategy.MaxResults {
		return out[:strategy.MaxResults]
	}
	return out
}

func CalculateBrokerFuturesBasis(cfg config.Config, spotTickers []exchange.Ticker, brokerTickers []exchange.Ticker, orderBooks []exchange.OrderBook) []BrokerFuturesBasisOpportunity {
	strategy := cfg.Strategies.CryptoSpotVsIBKRFutures
	if !strategy.Enabled {
		return nil
	}
	if !strategy.UseOrderBookDepth {
		orderBooks = nil
	}
	allTickers := append(append([]exchange.Ticker(nil), spotTickers...), brokerTickers...)
	books := orderBookLookup(orderBooks, allTickers, "")
	spotFeeCache := make(map[string]decimal.Decimal)
	futuresFee := ibkrFuturesFee(cfg)
	var out []BrokerFuturesBasisOpportunity
	now := time.Now()
	for _, mapping := range strategy.Instruments {
		futuresBook, ok := findBrokerBook(books, mapping.FuturesSymbol.Provider, mapping.FuturesSymbol.InstrumentID, "")
		if !ok || len(futuresBook.Bids) == 0 {
			continue
		}
		for _, spotSymbol := range mapping.SpotSymbols {
			if !strategy.CryptoSpotViaIBKR && strings.EqualFold(spotSymbol.Provider, "ibkr") {
				continue
			}
			spotBook, ok := findBrokerBook(books, spotSymbol.Provider, "", spotSymbol.Symbol)
			if !ok || len(spotBook.Asks) == 0 {
				continue
			}
			fee, ok := spotFeeCache[strings.ToLower(spotSymbol.Provider)]
			if !ok {
				fee = spotFee(cfg, spotSymbol.Provider)
				spotFeeCache[strings.ToLower(spotSymbol.Provider)] = fee
			}
			for _, tradeSize := range simulationTradeSizes(cfg) {
				spotBuy := SimulateBuyWithQuote(spotBook, tradeSize, fee)
				futuresSell := SimulateSellBase(futuresBook, spotBuy.FilledBaseQty, futuresFee)
				basis := decimal.Zero
				if !spotBuy.AveragePrice.IsZero() {
					basis = futuresSell.AveragePrice.Sub(spotBuy.AveragePrice).Div(spotBuy.AveragePrice).Mul(decimal.NewFromInt(100))
				}
				net := ProfitPercent(tradeSize, futuresSell.ReceivedQuoteValue)
				if belowOptionalProfitFloor(basis, strategy.MinBasisPercent.DecimalValue()) {
					continue
				}
				complete := spotBuy.CompleteFill && futuresSell.CompleteFill && withinSlippageLimit(cfg, spotBuy.SlippagePercent) && withinSlippageLimit(cfg, futuresSell.SlippagePercent)
				out = append(out, BrokerFuturesBasisOpportunity{
					StrategyTitle:       strategy.Title,
					Asset:               exchange.NormalizeAsset(mapping.Asset),
					SpotProvider:        strings.ToLower(spotSymbol.Provider),
					SpotSymbol:          exchange.NormalizeCanonicalSymbol(spotSymbol.Symbol),
					SpotAsk:             spotBook.Asks[0].Price,
					FuturesProvider:     strings.ToLower(mapping.FuturesSymbol.Provider),
					FuturesInstrumentID: mapping.FuturesSymbol.InstrumentID,
					FuturesDisplayName:  futuresBook.Symbol,
					FuturesBid:          futuresBook.Bids[0].Price,
					BasisPercent:        basis,
					NetEstimatePercent:  net,
					CompleteFill:        complete,
					Status:              statusFromPercent(net, "WATCH", "NO"),
					UpdatedAt:           now,
				})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].NetEstimatePercent.GreaterThan(out[j].NetEstimatePercent)
	})
	if strategy.MaxResults > 0 && len(out) > strategy.MaxResults {
		return out[:strategy.MaxResults]
	}
	return out
}

func providerOrderBooks(providerName string, books map[string]exchange.OrderBook) map[string]exchange.OrderBook {
	out := make(map[string]exchange.OrderBook)
	for key, book := range books {
		if providerMatches(book, providerName) {
			out[key] = book
		}
	}
	return out
}

func findBrokerBook(books map[string]exchange.OrderBook, providerName, instrumentID, symbol string) (exchange.OrderBook, bool) {
	for _, book := range books {
		if !providerMatches(book, providerName) {
			continue
		}
		if instrumentID != "" && strings.EqualFold(book.InstrumentID, instrumentID) {
			return book, true
		}
		if symbol != "" && strings.EqualFold(exchange.NormalizeCanonicalSymbol(book.Symbol), exchange.NormalizeCanonicalSymbol(symbol)) {
			return book, true
		}
	}
	return exchange.OrderBook{}, false
}

func providerMatches(book exchange.OrderBook, providerName string) bool {
	providerName = strings.ToLower(providerName)
	return strings.ToLower(book.Provider) == providerName ||
		strings.ToLower(book.Exchange) == providerName ||
		strings.ToLower(book.Broker) == providerName
}

func normalizeCycle(cycle []string) []string {
	out := make([]string, len(cycle))
	for i, asset := range cycle {
		out[i] = exchange.NormalizeAsset(asset)
	}
	return out
}

func ibkrFXFee(cfg config.Config) decimal.Decimal {
	if ibkr, ok := lookupProviderConfig(cfg, "ibkr"); ok {
		return ibkr.Fees.FXEstimatedTaker.DecimalValue()
	}
	return decimal.Zero
}

func ibkrFuturesFee(cfg config.Config) decimal.Decimal {
	if ibkr, ok := lookupProviderConfig(cfg, "ibkr"); ok {
		if !ibkr.Fees.FuturesEstimatedTaker.DecimalValue().IsZero() {
			return ibkr.Fees.FuturesEstimatedTaker.DecimalValue()
		}
		return ibkr.Fees.FuturesTaker.DecimalValue()
	}
	return decimal.Zero
}

func lookupProviderConfig(cfg config.Config, providerName string) (config.ProviderConfig, bool) {
	if provider, ok := cfg.Providers[providerName]; ok {
		return provider, true
	}
	for name, provider := range cfg.Providers {
		if strings.EqualFold(name, providerName) {
			return provider, true
		}
	}
	return config.ProviderConfig{}, false
}
