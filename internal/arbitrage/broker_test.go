package arbitrage_test

import (
	"testing"

	"go-crypto-arb/internal/arbitrage"
	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
)

func TestCalculateIBKRFXTriangular(t *testing.T) {
	cfg := testConfig()
	cfg.Simulation.TradeSizes = []config.Decimal{cfgDec("1000")}
	cfg.Providers = map[string]config.ProviderConfig{
		"ibkr": {Fees: config.ProviderFees{FXEstimatedTaker: cfgDec("0")}},
	}
	cfg.Strategies.IBKRFXTriangular = config.FXTriangularStrategyConfig{
		Enabled:           true,
		Title:             "IBKR FX Triangular Arbitrage",
		Cycles:            [][]string{{"USD", "EUR", "JPY", "USD"}},
		MaxResults:        10,
		UseOrderBookDepth: true,
	}
	books := []exchange.OrderBook{
		ibkrBook("EUR/USD", "", exchange.MarketSpot, [][2]string{{"0.91", "100000"}}, [][2]string{{"0.92", "100000"}}, "fx"),
		ibkrBook("EUR/JPY", "", exchange.MarketSpot, [][2]string{{"170", "100000"}}, [][2]string{{"171", "100000"}}, "fx"),
		ibkrBook("USD/JPY", "", exchange.MarketSpot, [][2]string{{"150", "100000"}}, [][2]string{{"151", "100000"}}, "fx"),
	}
	got := arbitrage.CalculateIBKRFXTriangular(cfg, nil, books)
	if len(got) == 0 {
		t.Fatal("expected IBKR FX triangular result")
	}
	if got[0].Provider != "ibkr" || got[0].AssetClass != "fx" {
		t.Fatalf("expected IBKR FX result, got %#v", got[0])
	}
}

func TestCalculateBrokerFuturesBasis(t *testing.T) {
	cfg := testConfig()
	cfg.Simulation.TradeSizes = []config.Decimal{cfgDec("1000")}
	cfg.Providers = map[string]config.ProviderConfig{
		"ibkr": {Fees: config.ProviderFees{FuturesEstimatedTaker: cfgDec("0")}},
	}
	cfg.Strategies.CryptoSpotVsIBKRFutures = config.BrokerFuturesBasisStrategy{
		Enabled:           true,
		Title:             "Crypto Spot vs IBKR Futures Basis",
		MaxResults:        10,
		UseOrderBookDepth: true,
		Instruments: []config.BrokerFuturesBasisInstrument{{
			ID:    "BTC_SPOT_VS_CME_MBT",
			Asset: "BTC",
			SpotSymbols: []config.ProviderSymbol{{
				Provider: "binance",
				Symbol:   "BTC/USDT",
			}},
			FuturesSymbol: config.ProviderInstrumentSymbol{
				Provider:     "ibkr",
				InstrumentID: "CME_MICRO_BTC",
			},
		}},
	}
	books := []exchange.OrderBook{
		book("Binance", "BTC/USDT", exchange.MarketSpot, [][2]string{{"99", "20"}}, [][2]string{{"100", "20"}}),
		ibkrBook("MBT", "CME_MICRO_BTC", exchange.MarketFutures, [][2]string{{"105", "20"}}, [][2]string{{"106", "20"}}, "futures"),
	}
	got := arbitrage.CalculateBrokerFuturesBasis(cfg, nil, nil, books)
	if len(got) != 1 {
		t.Fatalf("expected one basis result, got %d", len(got))
	}
	if !got[0].BasisPercent.GreaterThan(dec("4")) {
		t.Fatalf("expected positive basis, got %s", got[0].BasisPercent)
	}
}

func ibkrBook(symbol, instrumentID string, marketType exchange.MarketType, bids [][2]string, asks [][2]string, assetClass string) exchange.OrderBook {
	out := book("IBKR", symbol, marketType, bids, asks)
	out.Provider = "ibkr"
	out.Broker = "IBKR"
	out.InstrumentID = instrumentID
	out.AssetClass = assetClass
	return exchange.NormalizeOrderBook(out, 0)
}
