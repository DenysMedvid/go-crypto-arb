package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"go-crypto-arb/internal/alerts"
	"go-crypto-arb/internal/arbitrage"
	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
	"go-crypto-arb/internal/marketdata"
)

func TestPrometheusMetricsEndpoint(t *testing.T) {
	store := marketdata.NewStore()
	now := time.Now()
	store.UpsertSpotTickers([]exchange.Ticker{
		{
			Provider:   "binance",
			Exchange:   "Binance",
			Symbol:     "BTC/USDT",
			BaseAsset:  "BTC",
			QuoteAsset: "USDT",
			MarketType: exchange.MarketSpot,
			AssetClass: "crypto",
			Bid:        decimal.RequireFromString("100"),
			Ask:        decimal.RequireFromString("101"),
			Last:       decimal.RequireFromString("100.5"),
			UpdatedAt:  now.Add(-2 * time.Second),
		},
	})
	store.UpsertFuturesTickers([]exchange.Ticker{
		{
			Provider:   "binance",
			Exchange:   "Binance",
			Symbol:     "BTC/USDT",
			BaseAsset:  "BTC",
			QuoteAsset: "USDT",
			MarketType: exchange.MarketFutures,
			AssetClass: "crypto",
			Bid:        decimal.RequireFromString("102"),
			Ask:        decimal.RequireFromString("103"),
			UpdatedAt:  now.Add(-3 * time.Second),
		},
	})
	store.UpsertFundingRates([]exchange.FundingRate{
		{
			Exchange:        "Binance",
			Symbol:          "BTC/USDT",
			Rate:            decimal.RequireFromString("0.0001"),
			NextFundingTime: now.Add(time.Hour),
			UpdatedAt:       now.Add(-time.Minute),
		},
	})
	store.UpsertOrderBooks([]exchange.OrderBook{
		{
			Provider:   "binance",
			Exchange:   "Binance",
			Symbol:     "BTC/USDT",
			MarketType: exchange.MarketSpot,
			AssetClass: "crypto",
			Bids:       []exchange.OrderBookLevel{{Price: decimal.RequireFromString("99.5"), Quantity: decimal.RequireFromString("2")}},
			Asks:       []exchange.OrderBookLevel{{Price: decimal.RequireFromString("101.5"), Quantity: decimal.RequireFromString("3")}},
			UpdatedAt:  now.Add(-4 * time.Second),
		},
	})
	store.SetMarkets([]exchange.MarketInfo{
		{
			Provider:     "ibkr",
			Exchange:     "IBKR",
			Broker:       "IBKR",
			Symbol:       "EUR/USD",
			InstrumentID: "EUR_USD",
			DisplayName:  "EUR/USD",
			AssetClass:   "fx",
			MarketType:   exchange.MarketSpot,
			Active:       true,
		},
	})
	store.SetCalculations(
		[]arbitrage.TriangularOpportunityV2{{
			Provider:           "binance",
			Exchange:           "Binance",
			StrategyTitle:      "Crypto Triangular",
			Cycle:              []string{"BTC", "ETH", "USDT"},
			StartAmount:        decimal.RequireFromString("100"),
			EndAmount:          decimal.RequireFromString("101"),
			NetProfitPercent:   decimal.RequireFromString("1"),
			CompleteFill:       true,
			MaxSlippagePercent: decimal.RequireFromString("0.2"),
			Legs: []arbitrage.LegSimulation{{
				FromAsset:       "BTC",
				ToAsset:         "ETH",
				Symbol:          "BTC/ETH",
				Side:            arbitrage.TradeSell,
				InputAmount:     decimal.RequireFromString("1"),
				OutputAmount:    decimal.RequireFromString("14"),
				AveragePrice:    decimal.RequireFromString("14"),
				FeeAmount:       decimal.RequireFromString("0.01"),
				SlippagePercent: decimal.RequireFromString("0.1"),
				CompleteFill:    true,
			}},
			UpdatedAt: now.Add(-5 * time.Second),
		}},
		[]arbitrage.CrossExchangeOpportunityV2{{
			StrategyTitle:       "Cross",
			Symbol:              "BTC/USDT",
			BuyProvider:         "binance",
			SellProvider:        "kraken",
			BuyExchange:         "Binance",
			SellExchange:        "Kraken",
			TradeSize:           decimal.RequireFromString("1000"),
			BuyAveragePrice:     decimal.RequireFromString("100"),
			SellAveragePrice:    decimal.RequireFromString("102"),
			BuySlippagePercent:  decimal.RequireFromString("0.1"),
			SellSlippagePercent: decimal.RequireFromString("0.2"),
			BuyFeeAmount:        decimal.RequireFromString("1"),
			SellFeeAmount:       decimal.RequireFromString("1.5"),
			NetProfitPercent:    decimal.RequireFromString("0.5"),
			CompleteFill:        true,
			UpdatedAt:           now.Add(-6 * time.Second),
		}},
		[]arbitrage.SpotFuturesOpportunityV2{{
			StrategyTitle:           "Spot Futures",
			Provider:                "binance",
			Exchange:                "Binance",
			Symbol:                  "BTC/USDT",
			TradeSize:               decimal.RequireFromString("500"),
			SpotAverageBuyPrice:     decimal.RequireFromString("100"),
			FuturesAverageSellPrice: decimal.RequireFromString("103"),
			BasisPercent:            decimal.RequireFromString("3"),
			FundingRate:             decimal.RequireFromString("0.0001"),
			NetEstimatePercent:      decimal.RequireFromString("2.5"),
			CompleteFill:            true,
			UpdatedAt:               now.Add(-7 * time.Second),
		}},
		[]arbitrage.RelatedAssetGroupSignal{{
			Group:        "BTC ecosystem",
			GroupAverage: decimal.RequireFromString("1.2"),
			CalculatedAt: now.Add(-8 * time.Second),
			Assets: []arbitrage.RelatedAssetSignal{{
				Symbol:            "BTC/USDT",
				Asset:             "BTC",
				Exchange:          "Binance",
				ChangePercent:     decimal.RequireFromString("1.1"),
				DivergencePercent: decimal.RequireFromString("0.2"),
			}},
		}},
	)
	store.SetBrokerCalculations(
		[]arbitrage.TriangularOpportunityV2{{
			Provider:         "ibkr",
			Exchange:         "IBKR",
			StrategyTitle:    "IBKR FX",
			Cycle:            []string{"EUR", "USD", "GBP"},
			StartAmount:      decimal.RequireFromString("100"),
			EndAmount:        decimal.RequireFromString("100.5"),
			NetProfitPercent: decimal.RequireFromString("0.5"),
			CompleteFill:     true,
			UpdatedAt:        now.Add(-9 * time.Second),
		}},
		[]arbitrage.BrokerFuturesBasisOpportunity{{
			StrategyTitle:       "IBKR Basis",
			Asset:               "BTC",
			SpotProvider:        "binance",
			SpotSymbol:          "BTC/USDT",
			SpotAsk:             decimal.RequireFromString("101"),
			FuturesProvider:     "ibkr",
			FuturesInstrumentID: "CME_MICRO_BTC",
			FuturesBid:          decimal.RequireFromString("105"),
			BasisPercent:        decimal.RequireFromString("4"),
			NetEstimatePercent:  decimal.RequireFromString("3.5"),
			CompleteFill:        true,
			UpdatedAt:           now.Add(-10 * time.Second),
		}},
	)
	store.SetAlerts([]alerts.Alert{{
		Type:        alerts.AlertCrossExchange,
		Severity:    alerts.AlertWarning,
		Symbol:      "BTC/USDT",
		Value:       decimal.RequireFromString("0.5"),
		Threshold:   decimal.RequireFromString("0.2"),
		UpdatedAt:   now.Add(-11 * time.Second),
		RepeatCount: 2,
		Status:      "active",
	}})
	store.SetExchangeHealth([]exchange.ExchangeHealth{{
		Provider:            "binance",
		Exchange:            "Binance",
		Enabled:             true,
		SpotEnabled:         true,
		FuturesEnabled:      true,
		WebSocketEnabled:    true,
		WebSocketConnected:  true,
		RestFallbackActive:  false,
		DataFresh:           true,
		ReconnectCount:      1,
		StaleTickerCount:    2,
		StaleOrderBookCount: 3,
		Score:               95,
		Status:              "ok",
		LastMessageTime:     now.Add(-12 * time.Second),
	}})
	cfg := config.Config{
		Metrics: config.MetricsConfig{
			PrometheusEnabled: true,
			PrometheusPath:    "/metrics",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	NewServer(cfg, store, "secret", nil).Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	wants := []string{
		`go_crypto_arb_price_bid{market="spot",provider="binance",symbol="BTC/USDT"} 100`,
		`go_crypto_arb_price_spread{market="spot",provider="binance",symbol="BTC/USDT"} 1`,
		`go_crypto_arb_funding_rate{exchange="binance",symbol="BTC/USDT"} 0.0001`,
		`go_crypto_arb_market_active{asset_class="fx",broker="ibkr",display_name="EUR/USD",exchange="ibkr",instrument_id="EUR_USD",market="spot",provider="ibkr",symbol="EUR/USD"} 1`,
		`go_crypto_arb_arbitrage_trade_size{buy_exchange="binance",buy_provider="binance",exchange="",provider="",sell_exchange="kraken",sell_provider="kraken",strategy="Cross",symbol="BTC/USDT",type="cross_exchange"} 1000`,
		`go_crypto_arb_arbitrage_leg_input_amount`,
		`go_crypto_arb_arbitrage_basis_percent`,
		`go_crypto_arb_related_asset_change_percent{asset="BTC",exchange="binance",group="BTC ecosystem",symbol="BTC/USDT"} 1.1`,
		`go_crypto_arb_order_book_best_bid{exchange="binance",market="spot",provider="binance",symbol="BTC/USDT"} 99.5`,
		`go_crypto_arb_alert_repeat_count{exchange="",severity="warning",status="active",symbol="BTC/USDT",type="cross_exchange_arbitrage"} 2`,
		`go_crypto_arb_stale_order_book_total{exchange="binance",provider="binance",status="ok"} 3`,
	}
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Fatalf("expected metrics body to contain %q, got:\n%s", want, body)
		}
	}
}

func TestMetricsSnapshotEndpointRemoved(t *testing.T) {
	cfg := config.Config{
		Metrics: config.MetricsConfig{
			PrometheusEnabled: true,
			PrometheusPath:    "/metrics",
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/snapshot", nil)
	req.Header.Set("X-API-Key", "secret")
	rr := httptest.NewRecorder()

	NewServer(cfg, marketdata.NewStore(), "secret", nil).Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected removed metrics snapshot endpoint to return 404, got %d", rr.Code)
	}
}
