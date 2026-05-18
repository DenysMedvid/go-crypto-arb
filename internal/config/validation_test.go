package config

import (
	"os"
	"path/filepath"
	"testing"

	"go-crypto-arb/internal/exchange"
)

func TestParseV2ConfigFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := []byte(`
app:
  refresh_interval: 2s
api:
  http_addr: ":8080"
market_data:
  order_book_depth: 50
  order_book_stale_after: 10s
  ticker_stale_after: 15s
simulation:
  enabled: true
  quote_asset: USDT
  trade_sizes: [100, 500]
  max_slippage_percent: 0.3
exchanges:
  binance:
    enabled: true
    fees:
      spot_taker: 0.001
      futures_taker: 0.0005
assets:
  preferred: [BTC, USDT]
  stable_bases: [USDT]
  quote_assets: [USDT]
alerts:
  enabled: true
  cooldown: 5m
metrics:
  prometheus_enabled: true
  prometheus_path: /metrics
health:
  scoring_enabled: true
  stale_penalty: 20
tui:
  use_emoji: true
  tabs:
    ibkr:
      enabled: true
      title: "IBKR Monitor"
providers:
  ibkr:
    enabled: true
    type: broker
    market_data_enabled: true
    trading_enabled: false
    crypto_spot_enabled: false
    host: "127.0.0.1"
    port: 7497
    client_id: 101
instrument_universes:
  ibkr_fx:
    title: "IBKR FX Pairs"
    providers: ["ibkr"]
    instruments:
      - id: EUR_USD
        display_name: "EUR/USD"
        symbol: "EUR/USD"
        asset_class: fx
        market_type: spot
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MarketData.OrderBookDepth != 50 {
		t.Fatalf("expected depth 50, got %d", cfg.MarketData.OrderBookDepth)
	}
	if len(cfg.Simulation.TradeSizes) != 2 {
		t.Fatalf("expected two trade sizes, got %d", len(cfg.Simulation.TradeSizes))
	}
	if !cfg.Metrics.PrometheusEnabled {
		t.Fatal("expected prometheus enabled")
	}
	if !cfg.TUI.UseEmoji {
		t.Fatal("expected emoji enabled")
	}
	if cfg.TUI.Tabs["ibkr"].Title != "IBKR Monitor" {
		t.Fatalf("expected custom IBKR tab title, got %q", cfg.TUI.Tabs["ibkr"].Title)
	}
	if cfg.Providers["ibkr"].TradingEnabled {
		t.Fatal("expected IBKR trading disabled")
	}
}

func TestValidateMissingAPIKey(t *testing.T) {
	cfg := Config{
		Exchanges: map[string]ExchangeConfig{"binance": {Enabled: true, Fees: ExchangeFees{SpotTaker: MustDecimal("0.001")}}},
		Assets:    AssetConfig{Preferred: []string{"BTC", "USDT"}, StableBases: []string{"USDT"}, QuoteAssets: []string{"USDT"}},
	}
	messages := Validate(cfg, Env{}, nil)
	if !HasValidationErrors(messages) {
		t.Fatal("expected validation error")
	}
}

func TestValidateIBKRSafety(t *testing.T) {
	cfg := Config{
		Providers: map[string]ProviderConfig{
			"ibkr": {
				Enabled:           true,
				Type:              "broker",
				MarketDataEnabled: true,
				TradingEnabled:    true,
				CryptoSpotEnabled: false,
				Host:              "127.0.0.1",
				Port:              7497,
				ClientID:          101,
			},
		},
		InstrumentUniverses: map[string]InstrumentUniverse{
			"ibkr_fx": {
				Providers: []string{"ibkr"},
				Instruments: []InstrumentConfig{{
					ID:          "EUR_USD",
					DisplayName: "EUR/USD",
					Symbol:      "EUR/USD",
					AssetClass:  "fx",
					MarketType:  "spot",
				}},
			},
		},
		Assets: AssetConfig{Preferred: []string{"USD", "EUR"}, StableBases: []string{"USD"}, QuoteAssets: []string{"USD", "EUR"}},
	}
	messages := Validate(cfg, Env{APIKey: "secret"}, nil)
	foundTradingError := false
	foundCryptoSpotOK := false
	for _, message := range messages {
		if message.Level == ValidationError && message.Message == "IBKR trading_enabled must remain false in v2.1" {
			foundTradingError = true
		}
		if message.Level == ValidationOK && message.Message == "IBKR crypto spot is disabled" {
			foundCryptoSpotOK = true
		}
	}
	if !foundTradingError || !foundCryptoSpotOK {
		t.Fatalf("expected IBKR safety messages, got %#v", messages)
	}
}

func TestValidateMissingMarkets(t *testing.T) {
	cfg := Config{
		Exchanges: map[string]ExchangeConfig{"binance": {Enabled: true, FuturesEnabled: true, Fees: ExchangeFees{SpotTaker: MustDecimal("0.001"), FuturesTaker: MustDecimal("0.0005")}}},
		Assets:    AssetConfig{Preferred: []string{"BTC", "USDT"}, StableBases: []string{"USDT"}, QuoteAssets: []string{"USDT"}},
	}
	markets := []exchange.MarketInfo{{Exchange: "Binance", Symbol: "BTC/USDT", BaseAsset: "BTC", QuoteAsset: "USDT", MarketType: exchange.MarketSpot, Active: true}}
	messages := Validate(cfg, Env{APIKey: "secret"}, markets)
	found := false
	for _, message := range messages {
		if message.Level == ValidationWarn && message.Message == "Binance futures BTC/USDT is unavailable" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected missing futures warning, got %#v", messages)
	}
}

func TestValidateSupportedCryptoPlatforms(t *testing.T) {
	providers := make(map[string]ProviderConfig)
	for _, name := range []string{"okx", "bybit", "binance", "kraken", "coinbase", "gateio", "bitget"} {
		providers[name] = ProviderConfig{
			Enabled:     true,
			Type:        "crypto_exchange",
			SpotEnabled: true,
			Fees: ProviderFees{
				SpotTaker: MustDecimal("0.001"),
			},
		}
	}
	cfg := Config{
		Providers: providers,
		Assets: AssetConfig{
			Preferred:   []string{"BTC", "USDT"},
			StableBases: []string{"USDT"},
			QuoteAssets: []string{"USDT"},
		},
	}
	messages := Validate(cfg, Env{APIKey: "secret"}, nil)
	for _, message := range messages {
		if message.Level == ValidationError {
			t.Fatalf("expected supported platforms to validate without errors, got %#v", messages)
		}
	}
}
