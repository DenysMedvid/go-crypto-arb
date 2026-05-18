package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-crypto-arb/internal/arbitrage"
	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
	"go-crypto-arb/internal/marketdata"
)

type Server struct {
	cfg    config.Config
	store  *marketdata.Store
	apiKey string
	logger *slog.Logger
}

func NewServer(cfg config.Config, store *marketdata.Store, apiKey string, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{cfg: cfg, store: store, apiKey: apiKey, logger: logger}
}

func (s *Server) Handler() http.Handler {
	root := http.NewServeMux()
	root.HandleFunc("GET /health", s.handleHealth)
	if s.cfg.Metrics.PrometheusEnabled {
		root.HandleFunc("GET "+s.cfg.Metrics.PrometheusPath, s.handlePrometheus)
	}

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("GET /api/v1/prices", s.handlePrices)
	apiMux.HandleFunc("GET /api/v1/order-books", s.handleOrderBooks)
	apiMux.HandleFunc("GET /api/v1/providers", s.handleProviders)
	apiMux.HandleFunc("GET /api/v1/providers/health", s.handleProviderHealth)
	apiMux.HandleFunc("GET /api/v1/arbitrage/triangular", s.handleTriangular)
	apiMux.HandleFunc("GET /api/v1/arbitrage/cross-exchange", s.handleCrossExchange)
	apiMux.HandleFunc("GET /api/v1/arbitrage/spot-futures", s.handleSpotFutures)
	apiMux.HandleFunc("GET /api/v1/ibkr/instruments", s.handleIBKRInstruments)
	apiMux.HandleFunc("GET /api/v1/ibkr/fx-triangular", s.handleIBKRFXTriangular)
	apiMux.HandleFunc("GET /api/v1/ibkr/crypto-futures-basis", s.handleIBKRBasis)
	apiMux.HandleFunc("GET /api/v1/signals/related-assets", s.handleRelatedAssets)
	apiMux.HandleFunc("GET /api/v1/alerts", s.handleAlerts)
	apiMux.HandleFunc("GET /api/v1/exchanges/health", s.handleExchangeHealth)
	apiMux.HandleFunc("GET /api/v1/snapshot", s.handleSnapshot)
	root.Handle("/api/v1/", RequireAPIKey(s.apiKey)(apiMux))
	return loggingMiddleware(s.logger, CORSMiddleware(s.cfg.API.CORSAllowedOrigins)(root))
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": s.cfg.App.Version,
		"time":    time.Now().UTC(),
	})
}

func (s *Server) handlePrices(w http.ResponseWriter, _ *http.Request) {
	snapshot := s.store.Snapshot()
	writeJSON(w, http.StatusOK, map[string]any{
		"prices":         snapshot.Prices,
		"futures_prices": snapshot.FuturesPrices,
		"funding_rates":  snapshot.FundingRates,
	})
}

func (s *Server) handleOrderBooks(w http.ResponseWriter, r *http.Request) {
	snapshot := s.store.Snapshot()
	exchangeFilter := strings.ToLower(r.URL.Query().Get("exchange"))
	providerFilter := strings.ToLower(r.URL.Query().Get("provider"))
	symbolFilter := strings.ToUpper(r.URL.Query().Get("symbol"))
	marketFilter := strings.ToLower(r.URL.Query().Get("market"))
	var out []exchange.OrderBook
	for _, book := range snapshot.OrderBooks {
		bookProvider := strings.ToLower(firstNonEmpty(book.Provider, book.Exchange, book.Broker))
		if providerFilter != "" && bookProvider != providerFilter {
			continue
		}
		if exchangeFilter != "" && strings.ToLower(book.Exchange) != exchangeFilter {
			continue
		}
		if symbolFilter != "" && strings.ToUpper(book.Symbol) != symbolFilter {
			continue
		}
		if marketFilter != "" && string(book.MarketType) != marketFilter {
			continue
		}
		out = append(out, book)
	}
	if exchangeFilter == "" && providerFilter == "" && symbolFilter == "" && marketFilter == "" {
		writeJSON(w, http.StatusOK, snapshot.OrderBookSummary)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleProviders(w http.ResponseWriter, _ *http.Request) {
	type providerResponse struct {
		Name    string                `json:"name"`
		Type    string                `json:"type"`
		Enabled bool                  `json:"enabled"`
		Config  config.ProviderConfig `json:"config"`
	}
	var out []providerResponse
	for name, providerCfg := range s.cfg.Providers {
		out = append(out, providerResponse{Name: name, Type: providerCfg.Type, Enabled: providerCfg.Enabled, Config: providerCfg})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleProviderHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Snapshot().ProviderHealth)
}

func (s *Server) handleTriangular(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Snapshot().TriangularArbitrage)
}

func (s *Server) handleCrossExchange(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Snapshot().CrossExchangeArbitrage)
}

func (s *Server) handleSpotFutures(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Snapshot().SpotFuturesArbitrage)
}

func (s *Server) handleIBKRInstruments(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Snapshot().IBKRInstruments)
}

func (s *Server) handleIBKRFXTriangular(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Snapshot().IBKRFXTriangular)
}

func (s *Server) handleIBKRBasis(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Snapshot().CryptoSpotVsIBKRBasis)
}

func (s *Server) handleRelatedAssets(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Snapshot().RelatedAssetSignals)
}

func (s *Server) handleAlerts(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Snapshot().Alerts)
}

func (s *Server) handleExchangeHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Snapshot().ExchangeHealth)
}

func (s *Server) handleSnapshot(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Snapshot())
}

func (s *Server) handlePrometheus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	snapshot := s.store.Snapshot()
	now := time.Now()
	for _, group := range snapshot.Prices {
		for _, ticker := range group {
			writeTickerMetrics(w, ticker, now)
		}
	}
	for _, group := range snapshot.FuturesPrices {
		for _, ticker := range group {
			writeTickerMetrics(w, ticker, now)
		}
	}
	for _, rate := range snapshot.FundingRates {
		labels := map[string]string{
			"exchange": strings.ToLower(rate.Exchange),
			"symbol":   rate.Symbol,
		}
		writeMetric(w, "go_crypto_arb_funding_rate", labels, rate.Rate.String())
		writeMetric(w, "go_crypto_arb_funding_age_seconds", labels, ageSeconds(rate.UpdatedAt, now))
		if !rate.NextFundingTime.IsZero() {
			writeMetric(w, "go_crypto_arb_funding_next_time_seconds", labels, strconv.FormatInt(rate.NextFundingTime.Unix(), 10))
		}
	}
	for _, market := range snapshot.Markets {
		labels := marketLabels(market)
		writeMetric(w, "go_crypto_arb_market_active", labels, boolValue(market.Active))
	}
	for _, item := range snapshot.CrossExchangeArbitrage {
		labels := opportunityLabels("cross_exchange", item.StrategyTitle, item.Symbol, "", "", item.BuyProvider, item.SellProvider, item.BuyExchange, item.SellExchange)
		writeMetric(w, "go_crypto_arb_arbitrage_profit_percent", labels, item.NetProfitPercent.String())
		writeMetric(w, "go_crypto_arb_arbitrage_trade_size", labels, item.TradeSize.String())
		writeMetric(w, "go_crypto_arb_arbitrage_buy_average_price", labels, item.BuyAveragePrice.String())
		writeMetric(w, "go_crypto_arb_arbitrage_sell_average_price", labels, item.SellAveragePrice.String())
		writeMetric(w, "go_crypto_arb_arbitrage_buy_slippage_percent", labels, item.BuySlippagePercent.String())
		writeMetric(w, "go_crypto_arb_arbitrage_sell_slippage_percent", labels, item.SellSlippagePercent.String())
		writeMetric(w, "go_crypto_arb_arbitrage_buy_fee_amount", labels, item.BuyFeeAmount.String())
		writeMetric(w, "go_crypto_arb_arbitrage_sell_fee_amount", labels, item.SellFeeAmount.String())
		writeMetric(w, "go_crypto_arb_arbitrage_complete_fill", labels, boolValue(item.CompleteFill))
		writeMetric(w, "go_crypto_arb_arbitrage_age_seconds", labels, ageSeconds(item.UpdatedAt, now))
	}
	for _, item := range snapshot.TriangularArbitrage {
		cycle := strings.Join(item.Cycle, ">")
		labels := opportunityLabels("triangular", item.StrategyTitle, cycle, item.Provider, item.Exchange, "", "", "", "")
		writeMetric(w, "go_crypto_arb_arbitrage_profit_percent", labels, item.NetProfitPercent.String())
		writeMetric(w, "go_crypto_arb_arbitrage_start_amount", labels, item.StartAmount.String())
		writeMetric(w, "go_crypto_arb_arbitrage_end_amount", labels, item.EndAmount.String())
		writeMetric(w, "go_crypto_arb_arbitrage_max_slippage_percent", labels, item.MaxSlippagePercent.String())
		writeMetric(w, "go_crypto_arb_arbitrage_complete_fill", labels, boolValue(item.CompleteFill))
		writeMetric(w, "go_crypto_arb_arbitrage_age_seconds", labels, ageSeconds(item.UpdatedAt, now))
		writeLegMetrics(w, labels, item.Legs)
	}
	for _, item := range snapshot.SpotFuturesArbitrage {
		labels := opportunityLabels("spot_futures", item.StrategyTitle, item.Symbol, item.Provider, item.Exchange, "", "", "", "")
		writeMetric(w, "go_crypto_arb_arbitrage_profit_percent", labels, item.NetEstimatePercent.String())
		writeMetric(w, "go_crypto_arb_arbitrage_trade_size", labels, item.TradeSize.String())
		writeMetric(w, "go_crypto_arb_arbitrage_spot_average_buy_price", labels, item.SpotAverageBuyPrice.String())
		writeMetric(w, "go_crypto_arb_arbitrage_futures_average_sell_price", labels, item.FuturesAverageSellPrice.String())
		writeMetric(w, "go_crypto_arb_arbitrage_spot_slippage_percent", labels, item.SpotSlippagePercent.String())
		writeMetric(w, "go_crypto_arb_arbitrage_futures_slippage_percent", labels, item.FuturesSlippagePercent.String())
		writeMetric(w, "go_crypto_arb_arbitrage_spot_fee_amount", labels, item.SpotFeeAmount.String())
		writeMetric(w, "go_crypto_arb_arbitrage_futures_fee_amount", labels, item.FuturesFeeAmount.String())
		writeMetric(w, "go_crypto_arb_arbitrage_basis_percent", labels, item.BasisPercent.String())
		writeMetric(w, "go_crypto_arb_arbitrage_funding_rate", labels, item.FundingRate.String())
		writeMetric(w, "go_crypto_arb_arbitrage_net_estimate_percent", labels, item.NetEstimatePercent.String())
		writeMetric(w, "go_crypto_arb_arbitrage_complete_fill", labels, boolValue(item.CompleteFill))
		writeMetric(w, "go_crypto_arb_arbitrage_age_seconds", labels, ageSeconds(item.UpdatedAt, now))
	}
	for _, item := range snapshot.IBKRFXTriangular {
		cycle := strings.Join(item.Cycle, ">")
		labels := opportunityLabels("ibkr_fx_triangular", item.StrategyTitle, cycle, item.Provider, item.Exchange, "", "", "", "")
		writeMetric(w, "go_crypto_arb_arbitrage_profit_percent", labels, item.NetProfitPercent.String())
		writeMetric(w, "go_crypto_arb_arbitrage_start_amount", labels, item.StartAmount.String())
		writeMetric(w, "go_crypto_arb_arbitrage_end_amount", labels, item.EndAmount.String())
		writeMetric(w, "go_crypto_arb_arbitrage_max_slippage_percent", labels, item.MaxSlippagePercent.String())
		writeMetric(w, "go_crypto_arb_arbitrage_complete_fill", labels, boolValue(item.CompleteFill))
		writeMetric(w, "go_crypto_arb_arbitrage_age_seconds", labels, ageSeconds(item.UpdatedAt, now))
		writeLegMetrics(w, labels, item.Legs)
	}
	for _, item := range snapshot.CryptoSpotVsIBKRBasis {
		labels := opportunityLabels("crypto_spot_vs_ibkr_futures", item.StrategyTitle, item.Asset, item.FuturesProvider, "", item.SpotProvider, item.FuturesProvider, "", "")
		labels["spot_symbol"] = item.SpotSymbol
		labels["futures_instrument_id"] = item.FuturesInstrumentID
		writeMetric(w, "go_crypto_arb_arbitrage_profit_percent", labels, item.NetEstimatePercent.String())
		writeMetric(w, "go_crypto_arb_arbitrage_spot_ask", labels, item.SpotAsk.String())
		writeMetric(w, "go_crypto_arb_arbitrage_futures_bid", labels, item.FuturesBid.String())
		writeMetric(w, "go_crypto_arb_arbitrage_basis_percent", labels, item.BasisPercent.String())
		writeMetric(w, "go_crypto_arb_arbitrage_net_estimate_percent", labels, item.NetEstimatePercent.String())
		writeMetric(w, "go_crypto_arb_arbitrage_complete_fill", labels, boolValue(item.CompleteFill))
		writeMetric(w, "go_crypto_arb_arbitrage_age_seconds", labels, ageSeconds(item.UpdatedAt, now))
	}
	for _, group := range snapshot.RelatedAssetSignals {
		groupLabels := map[string]string{"group": group.Group}
		writeMetric(w, "go_crypto_arb_related_asset_group_average_percent", groupLabels, group.GroupAverage.String())
		writeMetric(w, "go_crypto_arb_related_asset_group_age_seconds", groupLabels, ageSeconds(group.CalculatedAt, now))
		for _, signal := range group.Assets {
			labels := map[string]string{
				"group":    group.Group,
				"asset":    signal.Asset,
				"symbol":   signal.Symbol,
				"exchange": strings.ToLower(signal.Exchange),
			}
			writeMetric(w, "go_crypto_arb_related_asset_change_percent", labels, signal.ChangePercent.String())
			writeMetric(w, "go_crypto_arb_related_asset_divergence_percent", labels, signal.DivergencePercent.String())
		}
	}
	for _, book := range snapshot.OrderBooks {
		providerName := strings.ToLower(firstNonEmpty(book.Provider, book.Exchange, book.Broker))
		labels := map[string]string{"provider": providerName, "exchange": strings.ToLower(book.Exchange), "symbol": book.Symbol, "market": string(book.MarketType)}
		if len(book.Bids) > 0 {
			writeMetric(w, "go_crypto_arb_order_book_best_bid", labels, book.Bids[0].Price.String())
		}
		if len(book.Asks) > 0 {
			writeMetric(w, "go_crypto_arb_order_book_best_ask", labels, book.Asks[0].Price.String())
		}
		writeMetric(w, "go_crypto_arb_order_book_bid_levels", labels, strconv.Itoa(len(book.Bids)))
		writeMetric(w, "go_crypto_arb_order_book_ask_levels", labels, strconv.Itoa(len(book.Asks)))
		writeMetric(w, "go_crypto_arb_order_book_limited_depth", labels, boolValue(book.LimitedDepth))
		writeMetric(w, "go_crypto_arb_order_book_age_seconds", labels, ageSeconds(book.UpdatedAt, now))
	}
	for _, alert := range snapshot.Alerts {
		labels := map[string]string{
			"type":     string(alert.Type),
			"severity": string(alert.Severity),
			"status":   alert.Status,
			"exchange": strings.ToLower(alert.Exchange),
			"symbol":   alert.Symbol,
		}
		writeMetric(w, "go_crypto_arb_alert_active", labels, "1")
		writeMetric(w, "go_crypto_arb_alert_value", labels, alert.Value.String())
		writeMetric(w, "go_crypto_arb_alert_threshold", labels, alert.Threshold.String())
		writeMetric(w, "go_crypto_arb_alert_repeat_count", labels, strconv.Itoa(alert.RepeatCount))
		writeMetric(w, "go_crypto_arb_alert_age_seconds", labels, ageSeconds(alert.UpdatedAt, now))
	}
	for _, h := range snapshot.ExchangeHealth {
		connected := "0"
		if h.WebSocketConnected || h.RestFallbackActive || h.GatewayConnected {
			connected = "1"
		}
		providerName := strings.ToLower(firstNonEmpty(h.Provider, h.Exchange, h.Broker))
		labels := map[string]string{"provider": providerName, "exchange": strings.ToLower(h.Exchange), "status": h.Status}
		writeMetric(w, "go_crypto_arb_provider_connected", map[string]string{"provider": providerName}, connected)
		writeMetric(w, "go_crypto_arb_provider_enabled", labels, boolValue(h.Enabled))
		writeMetric(w, "go_crypto_arb_provider_spot_enabled", labels, boolValue(h.SpotEnabled))
		writeMetric(w, "go_crypto_arb_provider_futures_enabled", labels, boolValue(h.FuturesEnabled))
		writeMetric(w, "go_crypto_arb_provider_market_data_enabled", labels, boolValue(h.MarketDataEnabled))
		writeMetric(w, "go_crypto_arb_provider_trading_enabled", labels, boolValue(h.TradingEnabled))
		writeMetric(w, "go_crypto_arb_provider_websocket_enabled", labels, boolValue(h.WebSocketEnabled))
		writeMetric(w, "go_crypto_arb_provider_websocket_connected", labels, boolValue(h.WebSocketConnected))
		writeMetric(w, "go_crypto_arb_provider_gateway_connected", labels, boolValue(h.GatewayConnected))
		writeMetric(w, "go_crypto_arb_provider_market_data_ok", labels, boolValue(h.MarketDataOK))
		writeMetric(w, "go_crypto_arb_provider_rest_fallback_active", labels, boolValue(h.RestFallbackActive))
		writeMetric(w, "go_crypto_arb_provider_data_fresh", labels, boolValue(h.DataFresh))
		writeMetric(w, "go_crypto_arb_provider_partial_support", labels, boolValue(h.PartialSupport))
		writeMetric(w, "go_crypto_arb_provider_last_message_age_seconds", labels, ageSeconds(firstNonZeroTime(h.LastMessageAt, h.LastMessageTime), now))
		writeMetric(w, "go_crypto_arb_ws_reconnect_total", map[string]string{"provider": providerName}, strconv.Itoa(h.ReconnectCount))
		writeMetric(w, "go_crypto_arb_stale_price_total", map[string]string{"provider": providerName}, strconv.Itoa(h.StaleTickerCount))
		writeMetric(w, "go_crypto_arb_stale_order_book_total", labels, strconv.Itoa(h.StaleOrderBookCount))
		writeMetric(w, "go_crypto_arb_health_score", map[string]string{"provider": providerName}, strconv.Itoa(h.Score))
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Debug("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}

func writeMetric(w http.ResponseWriter, name string, labels map[string]string, value string) {
	fmt.Fprintf(w, "%s{%s} %s\n", name, labelsString(labels), value)
}

func writeTickerMetrics(w http.ResponseWriter, ticker exchange.Ticker, now time.Time) {
	providerName := strings.ToLower(firstNonEmpty(ticker.Provider, ticker.Exchange, ticker.Broker))
	labels := map[string]string{
		"provider": providerName,
		"symbol":   ticker.Symbol,
		"market":   string(ticker.MarketType),
	}
	writeMetric(w, "go_crypto_arb_price_bid", labels, ticker.Bid.String())
	writeMetric(w, "go_crypto_arb_price_ask", labels, ticker.Ask.String())
	writeMetric(w, "go_crypto_arb_price_last", labels, ticker.Last.String())
	writeMetric(w, "go_crypto_arb_price_spread", labels, ticker.Ask.Sub(ticker.Bid).String())
	writeMetric(w, "go_crypto_arb_price_age_seconds", labels, ageSeconds(ticker.UpdatedAt, now))
}

func writeLegMetrics(w http.ResponseWriter, baseLabels map[string]string, legs []arbitrage.LegSimulation) {
	for index, leg := range legs {
		labels := cloneLabels(baseLabels)
		labels["leg"] = strconv.Itoa(index + 1)
		labels["from_asset"] = leg.FromAsset
		labels["to_asset"] = leg.ToAsset
		labels["leg_symbol"] = leg.Symbol
		labels["side"] = string(leg.Side)
		writeMetric(w, "go_crypto_arb_arbitrage_leg_input_amount", labels, leg.InputAmount.String())
		writeMetric(w, "go_crypto_arb_arbitrage_leg_output_amount", labels, leg.OutputAmount.String())
		writeMetric(w, "go_crypto_arb_arbitrage_leg_average_price", labels, leg.AveragePrice.String())
		writeMetric(w, "go_crypto_arb_arbitrage_leg_fee_amount", labels, leg.FeeAmount.String())
		writeMetric(w, "go_crypto_arb_arbitrage_leg_slippage_percent", labels, leg.SlippagePercent.String())
		writeMetric(w, "go_crypto_arb_arbitrage_leg_complete_fill", labels, boolValue(leg.CompleteFill))
	}
}

func opportunityLabels(kind, strategy, symbol, provider, exchangeName, buyProvider, sellProvider, buyExchange, sellExchange string) map[string]string {
	return map[string]string{
		"type":          kind,
		"strategy":      strategy,
		"symbol":        symbol,
		"provider":      strings.ToLower(provider),
		"exchange":      strings.ToLower(exchangeName),
		"buy_provider":  strings.ToLower(buyProvider),
		"sell_provider": strings.ToLower(sellProvider),
		"buy_exchange":  strings.ToLower(buyExchange),
		"sell_exchange": strings.ToLower(sellExchange),
	}
}

func marketLabels(market exchange.MarketInfo) map[string]string {
	return map[string]string{
		"provider":      strings.ToLower(firstNonEmpty(market.Provider, market.Exchange, market.Broker)),
		"exchange":      strings.ToLower(market.Exchange),
		"broker":        strings.ToLower(market.Broker),
		"symbol":        market.Symbol,
		"instrument_id": market.InstrumentID,
		"display_name":  market.DisplayName,
		"asset_class":   market.AssetClass,
		"market":        string(market.MarketType),
	}
}

func ageSeconds(t time.Time, now time.Time) string {
	if t.IsZero() {
		return "0"
	}
	return strconv.FormatFloat(now.Sub(t).Seconds(), 'f', 3, 64)
}

func boolValue(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func cloneLabels(labels map[string]string) map[string]string {
	out := make(map[string]string, len(labels))
	for key, value := range labels {
		out[key] = value
	}
	return out
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func labelsString(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", key, labels[key]))
	}
	return strings.Join(parts, ",")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
