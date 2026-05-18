package arbitrage_test

import (
	"testing"

	"go-crypto-arb/internal/arbitrage"
	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
)

func TestCalculateTriangularV2FullLiquidity(t *testing.T) {
	cfg := testConfig()
	cfg.Simulation.TradeSizes = []config.Decimal{cfgDec("1000")}
	books := []exchange.OrderBook{
		book("Binance", "BTC/USDT", exchange.MarketSpot, [][2]string{{"99", "20"}}, [][2]string{{"100", "20"}}),
		book("Binance", "ETH/BTC", exchange.MarketSpot, [][2]string{{"0.049", "1000"}}, [][2]string{{"0.05", "1000"}}),
		book("Binance", "ETH/USDT", exchange.MarketSpot, [][2]string{{"6", "10000"}}, [][2]string{{"6.1", "10000"}}),
	}
	got := arbitrage.CalculateTriangularV2(cfg, nil, books)
	if len(got) == 0 {
		t.Fatal("expected triangular opportunity")
	}
	if !got[0].CompleteFill {
		t.Fatal("expected complete fill")
	}
	if !got[0].NetProfitPercent.GreaterThan(dec("19")) {
		t.Fatalf("expected profitable cycle, got %s", got[0].NetProfitPercent)
	}
}

func TestCalculateTriangularV2PartialLiquidity(t *testing.T) {
	cfg := testConfig()
	cfg.Simulation.TradeSizes = []config.Decimal{cfgDec("1000")}
	books := []exchange.OrderBook{
		book("Binance", "BTC/USDT", exchange.MarketSpot, [][2]string{{"99", "1"}}, [][2]string{{"100", "1"}}),
		book("Binance", "ETH/BTC", exchange.MarketSpot, [][2]string{{"0.049", "1"}}, [][2]string{{"0.05", "1"}}),
		book("Binance", "ETH/USDT", exchange.MarketSpot, [][2]string{{"6", "1"}}, [][2]string{{"6.1", "1"}}),
	}
	got := arbitrage.CalculateTriangularV2(cfg, nil, books)
	if len(got) == 0 {
		t.Fatal("expected triangular opportunity")
	}
	if got[0].CompleteFill {
		t.Fatal("expected partial liquidity")
	}
}

func TestCalculateCrossExchangeV2FullAndPartialLiquidity(t *testing.T) {
	cfg := testConfig()
	cfg.Simulation.TradeSizes = []config.Decimal{cfgDec("1000")}
	full := []exchange.OrderBook{
		book("Kraken", "BTC/USDT", exchange.MarketSpot, [][2]string{{"98", "20"}}, [][2]string{{"99", "20"}}),
		book("Binance", "BTC/USDT", exchange.MarketSpot, [][2]string{{"110", "20"}}, [][2]string{{"111", "20"}}),
	}
	got := arbitrage.CalculateCrossExchangeV2(cfg, nil, full)
	if len(got) == 0 || !got[0].CompleteFill {
		t.Fatalf("expected complete cross-exchange result: %#v", got)
	}
	partial := []exchange.OrderBook{
		book("Kraken", "BTC/USDT", exchange.MarketSpot, [][2]string{{"98", "1"}}, [][2]string{{"99", "1"}}),
		book("Binance", "BTC/USDT", exchange.MarketSpot, [][2]string{{"110", "0.1"}}, [][2]string{{"111", "1"}}),
	}
	got = arbitrage.CalculateCrossExchangeV2(cfg, nil, partial)
	if len(got) == 0 || got[0].CompleteFill {
		t.Fatalf("expected partial cross-exchange result: %#v", got)
	}
}

func TestCalculateSpotFuturesV2WithFunding(t *testing.T) {
	cfg := testConfig()
	cfg.Simulation.TradeSizes = []config.Decimal{cfgDec("1000")}
	books := []exchange.OrderBook{
		book("Binance", "BTC/USDT", exchange.MarketSpot, [][2]string{{"99", "20"}}, [][2]string{{"100", "20"}}),
		book("Binance", "BTC/USDT", exchange.MarketFutures, [][2]string{{"105", "20"}}, [][2]string{{"106", "20"}}),
	}
	rates := []exchange.FundingRate{{Exchange: "Binance", Symbol: "BTC/USDT", Rate: dec("0.001")}}
	got := arbitrage.CalculateSpotFuturesV2(cfg, nil, nil, books, rates)
	if len(got) != 1 {
		t.Fatalf("expected one spot-futures result, got %d", len(got))
	}
	if !got[0].NetEstimatePercent.GreaterThan(dec("5")) {
		t.Fatalf("expected funding-adjusted net over 5%%, got %s", got[0].NetEstimatePercent)
	}
}
