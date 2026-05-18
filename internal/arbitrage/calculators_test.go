package arbitrage_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"go-crypto-arb/internal/arbitrage"
	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
)

func TestApplyTakerFee(t *testing.T) {
	got := arbitrage.ApplyTakerFee(dec("100"), dec("0.001"))
	if !got.Equal(dec("99.900")) {
		t.Fatalf("expected 99.900, got %s", got)
	}
}

func TestCalculateTriangular(t *testing.T) {
	cfg := testConfig()
	tickers := []exchange.Ticker{
		ticker("Binance", "BTC/USDT", "BTC", "USDT", "99", "100", exchange.MarketSpot),
		ticker("Binance", "ETH/BTC", "ETH", "BTC", "0.049", "0.05", exchange.MarketSpot),
		ticker("Binance", "ETH/USDT", "ETH", "USDT", "6", "6.1", exchange.MarketSpot),
	}
	got := arbitrage.CalculateTriangular(cfg, tickers)
	if len(got) == 0 {
		t.Fatal("expected triangular results")
	}
	if !got[0].ProfitPercent.GreaterThan(dec("19")) {
		t.Fatalf("expected profitable cycle, got %s", got[0].ProfitPercent)
	}
}

func TestCalculateCrossExchange(t *testing.T) {
	cfg := testConfig()
	tickers := []exchange.Ticker{
		ticker("Binance", "BTC/USDT", "BTC", "USDT", "101", "102", exchange.MarketSpot),
		ticker("Kraken", "BTC/USDT", "BTC", "USDT", "97", "98", exchange.MarketSpot),
	}
	got := arbitrage.CalculateCrossExchange(cfg, tickers)
	if len(got) == 0 {
		t.Fatal("expected cross-exchange results")
	}
	if got[0].BuyExchange != "Kraken" || got[0].SellExchange != "Binance" {
		t.Fatalf("unexpected route: buy %s sell %s", got[0].BuyExchange, got[0].SellExchange)
	}
	if !got[0].NetPercent.GreaterThan(dec("3")) {
		t.Fatalf("expected profit over 3%%, got %s", got[0].NetPercent)
	}
}

func TestCalculateSpotFutures(t *testing.T) {
	cfg := testConfig()
	spot := []exchange.Ticker{
		ticker("Binance", "BTC/USDT", "BTC", "USDT", "99", "100", exchange.MarketSpot),
	}
	futures := []exchange.Ticker{
		ticker("Binance", "BTC/USDT", "BTC", "USDT", "105", "106", exchange.MarketFutures),
	}
	rates := []exchange.FundingRate{{
		Exchange:  "Binance",
		Symbol:    "BTC/USDT",
		Rate:      dec("0.001"),
		UpdatedAt: time.Now(),
	}}
	got := arbitrage.CalculateSpotFutures(cfg, spot, futures, rates)
	if len(got) != 1 {
		t.Fatalf("expected one result, got %d", len(got))
	}
	if !got[0].BasisPercent.Equal(dec("5.00")) {
		t.Fatalf("expected 5%% basis, got %s", got[0].BasisPercent)
	}
	if !got[0].NetEstimate.GreaterThan(dec("5")) {
		t.Fatalf("expected funding-adjusted net over 5%%, got %s", got[0].NetEstimate)
	}
}

func testConfig() config.Config {
	return config.Config{
		Exchanges: map[string]config.ExchangeConfig{
			"Binance": {Fees: config.ExchangeFees{SpotTaker: cfgDec("0"), FuturesTaker: cfgDec("0")}},
			"Kraken":  {Fees: config.ExchangeFees{SpotTaker: cfgDec("0"), FuturesTaker: cfgDec("0")}},
		},
		Assets: config.AssetConfig{
			Preferred:   []string{"USDT", "BTC", "ETH"},
			StableBases: []string{"USDT"},
			QuoteAssets: []string{"USDT", "BTC"},
		},
		Arbitrage: config.ArbitrageConfig{
			Triangular:    config.ResultLimitConfig{Enabled: true, MaxResults: 10, UseOrderBookDepth: true},
			CrossExchange: config.ResultLimitConfig{Enabled: true, MaxResults: 10, UseOrderBookDepth: true},
			SpotFutures:   config.SpotFuturesArbConfig{Enabled: true, IncludeFundingRate: true, MaxResults: 10, UseOrderBookDepth: true},
		},
		Simulation: config.SimulationConfig{
			Enabled: true,
		},
	}
}

func ticker(exchangeName, symbol, base, quote, bid, ask string, marketType exchange.MarketType) exchange.Ticker {
	return exchange.Ticker{
		Exchange:   exchangeName,
		Symbol:     symbol,
		BaseAsset:  base,
		QuoteAsset: quote,
		MarketType: marketType,
		Bid:        dec(bid),
		Ask:        dec(ask),
		Last:       dec(bid).Add(dec(ask)).Div(dec("2")),
		UpdatedAt:  time.Now(),
	}
}

func cfgDec(value string) config.Decimal {
	return config.Decimal{Decimal: dec(value)}
}

func dec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}
