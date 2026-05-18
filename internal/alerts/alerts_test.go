package alerts

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"go-crypto-arb/internal/arbitrage"
	"go-crypto-arb/internal/config"
)

func TestAlertDeduplicationCooldown(t *testing.T) {
	engine := NewEngine()
	cfg := alertConfig()
	item := arbitrage.CrossExchangeOpportunityV2{
		Symbol:           "BTC/USDT",
		BuyExchange:      "Kraken",
		SellExchange:     "Binance",
		TradeSize:        dec("1000"),
		NetProfitPercent: dec("0.3"),
		CompleteFill:     true,
	}
	first := engine.Evaluate(cfg, nil, []arbitrage.CrossExchangeOpportunityV2{item}, nil, nil, nil, nil)
	second := engine.Evaluate(cfg, nil, []arbitrage.CrossExchangeOpportunityV2{item}, nil, nil, nil, nil)
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("expected one alert, got %d/%d", len(first), len(second))
	}
	if second[0].RepeatCount != 0 || !second[0].UpdatedAt.Equal(first[0].UpdatedAt) {
		t.Fatalf("expected deduped alert without repeat: %#v", second[0])
	}
}

func TestAlertRepeatsWhenProfitChangesEnough(t *testing.T) {
	engine := NewEngine()
	cfg := alertConfig()
	item := arbitrage.CrossExchangeOpportunityV2{
		Symbol:           "BTC/USDT",
		BuyExchange:      "Kraken",
		SellExchange:     "Binance",
		TradeSize:        dec("1000"),
		NetProfitPercent: dec("0.3"),
		CompleteFill:     true,
	}
	_ = engine.Evaluate(cfg, nil, []arbitrage.CrossExchangeOpportunityV2{item}, nil, nil, nil, nil)
	item.NetProfitPercent = dec("0.5")
	got := engine.Evaluate(cfg, nil, []arbitrage.CrossExchangeOpportunityV2{item}, nil, nil, nil, nil)
	if got[0].RepeatCount != 1 {
		t.Fatalf("expected repeat count 1, got %d", got[0].RepeatCount)
	}
}

func alertConfig() config.Config {
	return config.Config{
		Alerts: config.AlertsConfig{
			Enabled:                        true,
			MinProfitPercent:               config.Decimal{Decimal: dec("0.2")},
			MinBasisPercent:                config.Decimal{Decimal: dec("0.2")},
			Cooldown:                       config.Duration{Duration: 5 * time.Minute},
			RepeatIfProfitChangesByPercent: config.Decimal{Decimal: dec("0.1")},
			MaxResults:                     50,
		},
	}
}

func dec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}
