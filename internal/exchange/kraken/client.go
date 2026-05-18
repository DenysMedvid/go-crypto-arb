package kraken

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"

	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
	"go-crypto-arb/internal/health"
)

const (
	name            = "Kraken"
	spotRESTBase    = "https://api.kraken.com/0/public"
	futuresRESTBase = "https://futures.kraken.com/derivatives/api/v3"
	spotWSURL       = "wss://ws.kraken.com/v2"
)

type Client struct {
	cfg            config.ExchangeConfig
	knownAssets    []string
	staleAfter     time.Duration
	bookStaleAfter time.Duration
	orderBookDepth int
	logger         *slog.Logger
	httpClient     *http.Client

	cancel context.CancelFunc

	mu              sync.RWMutex
	restToCanonical map[string]string
	canonicalToRest map[string]string
	spotTickers     map[string]exchange.Ticker
	futuresTickers  map[string]exchange.Ticker
	orderBooks      map[string]exchange.OrderBook
	fundingRates    map[string]exchange.FundingRate
	markets         map[string]exchange.MarketInfo
	health          exchange.ExchangeHealth
	wsConnections   int
}

func New(cfg config.ExchangeConfig, knownAssets []string, staleAfter time.Duration, bookStaleAfter time.Duration, orderBookDepth int, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	if orderBookDepth <= 0 {
		orderBookDepth = 20
	}
	return &Client{
		cfg:             cfg,
		knownAssets:     knownAssets,
		staleAfter:      staleAfter,
		bookStaleAfter:  bookStaleAfter,
		orderBookDepth:  orderBookDepth,
		logger:          logger.With("exchange", name),
		httpClient:      &http.Client{Timeout: 10 * time.Second},
		restToCanonical: make(map[string]string),
		canonicalToRest: make(map[string]string),
		spotTickers:     make(map[string]exchange.Ticker),
		futuresTickers:  make(map[string]exchange.Ticker),
		orderBooks:      make(map[string]exchange.OrderBook),
		fundingRates:    make(map[string]exchange.FundingRate),
		markets:         make(map[string]exchange.MarketInfo),
		health: exchange.ExchangeHealth{
			Provider:         strings.ToLower(name),
			ProviderType:     "crypto_exchange",
			Exchange:         name,
			Enabled:          cfg.Enabled,
			SpotEnabled:      cfg.SpotEnabled,
			FuturesEnabled:   cfg.FuturesEnabled,
			WebSocketEnabled: cfg.WebsocketEnabled,
			Status:           "starting",
		},
	}
}

func (c *Client) Name() string { return name }

func (c *Client) Type() string { return "crypto_exchange" }

func (c *Client) Start(ctx context.Context) error {
	if !c.cfg.Enabled {
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	if c.cfg.SpotEnabled {
		if err := c.pollAssetPairs(runCtx); err != nil {
			c.setError(err)
		}
		if err := c.pollSpot(runCtx); err != nil {
			c.setError(err)
		}
		if err := c.pollSpotOrderBooks(runCtx); err != nil {
			c.setError(err)
		}
		go c.pollLoop(runCtx, c.pollAssetPairs)
		go c.pollLoop(runCtx, c.pollSpot)
		go c.pollLoop(runCtx, c.pollSpotOrderBooks)
		if c.cfg.WebsocketEnabled {
			go c.websocketLoop(runCtx)
		}
	}
	if c.cfg.FuturesEnabled {
		if err := c.pollFutures(runCtx); err != nil {
			c.setError(err)
		}
		go c.pollLoop(runCtx, c.pollFutures)
	}
	return nil
}

func (c *Client) Stop(context.Context) error {
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

func (c *Client) Health() exchange.ExchangeHealth {
	c.mu.RLock()
	defer c.mu.RUnlock()
	h := c.health
	h.WebSocketConnected = c.wsConnections > 0
	h.RestFallbackActive = !h.WebSocketConnected
	h.DataFresh = health.Fresh(h.LastMessageTime, c.staleAfter)
	h.LastMessageAt = h.LastMessageTime
	h.StaleTickerCount = countStaleTickers(c.spotTickers, c.futuresTickers, c.staleAfter)
	h.StaleOrderBookCount = countStaleOrderBooks(c.orderBooks, c.bookStaleAfter)
	h.PartialSupport = c.cfg.FuturesEnabled
	switch {
	case !h.DataFresh:
		h.Status = "stale"
	case h.WebSocketConnected:
		h.Status = "ok"
	case h.RestFallbackActive:
		h.Status = "rest-fallback"
	default:
		h.Status = "warn"
	}
	return h
}

func (c *Client) GetLatestTickers() []exchange.Ticker {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return cloneTickers(c.spotTickers)
}

func (c *Client) GetLatestFuturesTickers() []exchange.Ticker {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return cloneTickers(c.futuresTickers)
}

func (c *Client) GetFundingRates() []exchange.FundingRate {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]exchange.FundingRate, 0, len(c.fundingRates))
	for _, rate := range c.fundingRates {
		out = append(out, rate)
	}
	return out
}

func (c *Client) GetLatestOrderBooks() []exchange.OrderBook {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]exchange.OrderBook, 0, len(c.orderBooks))
	for _, book := range c.orderBooks {
		out = append(out, book)
	}
	return out
}

func (c *Client) GetMarkets() []exchange.MarketInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]exchange.MarketInfo, 0, len(c.markets))
	for _, market := range c.markets {
		out = append(out, market)
	}
	return out
}

func (c *Client) DiscoverMarkets(context.Context) ([]exchange.MarketInfo, error) {
	return c.GetMarkets(), nil
}

func (c *Client) pollLoop(ctx context.Context, fn func(context.Context) error) {
	interval := c.cfg.RestPollInterval.Duration
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if err := fn(ctx); err != nil {
		c.setError(err)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := fn(ctx); err != nil {
				c.setError(err)
			}
		}
	}
}

func (c *Client) pollAssetPairs(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spotRESTBase+"/AssetPairs", nil)
	if err != nil {
		return err
	}
	var payload assetPairsResponse
	if err := c.doJSON(req, &payload); err != nil {
		return err
	}
	if len(payload.Errors) > 0 {
		return fmt.Errorf("kraken asset pairs: %s", strings.Join(payload.Errors, ", "))
	}
	mapping := make(map[string]string)
	inverse := make(map[string]string)
	for restName, pair := range payload.Result {
		if pair.WSName == "" {
			continue
		}
		canonical := exchange.NormalizeCanonicalSymbol(pair.WSName)
		if c.knownPair(canonical) {
			mapping[restName] = canonical
			inverse[canonical] = restName
		}
	}
	c.mu.Lock()
	c.restToCanonical = mapping
	c.canonicalToRest = inverse
	c.mu.Unlock()
	return nil
}

func (c *Client) pollSpot(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spotRESTBase+"/Ticker", nil)
	if err != nil {
		return err
	}
	var payload tickerResponse
	if err := c.doJSON(req, &payload); err != nil {
		return err
	}
	if len(payload.Errors) > 0 {
		return fmt.Errorf("kraken ticker: %s", strings.Join(payload.Errors, ", "))
	}

	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for restName, item := range payload.Result {
		canonical, ok := c.restToCanonical[restName]
		if !ok {
			continue
		}
		bid, ask, last, ok := parseKrakenTicker(item)
		if !ok {
			continue
		}
		base, quote := splitCanonical(canonical)
		c.spotTickers[canonical] = exchange.Ticker{
			Provider:   strings.ToLower(name),
			Exchange:   name,
			Symbol:     canonical,
			BaseAsset:  base,
			QuoteAsset: quote,
			MarketType: exchange.MarketSpot,
			AssetClass: "crypto",
			Bid:        bid,
			Ask:        ask,
			Last:       last,
			UpdatedAt:  now,
		}
		c.markets[string(exchange.MarketSpot)+"|"+canonical] = exchange.MarketInfo{
			Provider:   strings.ToLower(name),
			Exchange:   name,
			Symbol:     canonical,
			BaseAsset:  base,
			QuoteAsset: quote,
			AssetClass: "crypto",
			MarketType: exchange.MarketSpot,
			Active:     true,
		}
	}
	c.health.LastMessageTime = now
	c.health.LastError = ""
	return nil
}

func (c *Client) pollFutures(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, futuresRESTBase+"/tickers", nil)
	if err != nil {
		return err
	}
	var payload futuresTickerResponse
	if err := c.doJSON(req, &payload); err != nil {
		return err
	}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, item := range payload.Tickers {
		canonical, ok := canonicalFromKrakenFutures(item)
		if !ok || !c.knownPair(canonical) {
			continue
		}
		bid, bidOK := exchange.DecimalFromString(firstNonEmpty(string(item.Bid), string(item.BestBid)))
		ask, askOK := exchange.DecimalFromString(firstNonEmpty(string(item.Ask), string(item.BestAsk)))
		if !bidOK || !askOK || !exchange.ValidBidAsk(bid, ask) {
			continue
		}
		last, _ := exchange.DecimalFromString(firstNonEmpty(string(item.Last), string(item.LastAlias)))
		base, quote := splitCanonical(canonical)
		c.futuresTickers[canonical] = exchange.Ticker{
			Provider:   strings.ToLower(name),
			Exchange:   name,
			Symbol:     canonical,
			BaseAsset:  base,
			QuoteAsset: quote,
			MarketType: exchange.MarketFutures,
			AssetClass: "crypto",
			Bid:        bid,
			Ask:        ask,
			Last:       last,
			UpdatedAt:  now,
		}
		c.markets[string(exchange.MarketFutures)+"|"+canonical] = exchange.MarketInfo{
			Provider:   strings.ToLower(name),
			Exchange:   name,
			Symbol:     canonical,
			BaseAsset:  base,
			QuoteAsset: quote,
			AssetClass: "crypto",
			MarketType: exchange.MarketFutures,
			Active:     true,
		}
		c.orderBooks[orderBookKey(exchange.OrderBook{MarketType: exchange.MarketFutures, Symbol: canonical})] = exchange.NormalizeOrderBook(exchange.OrderBook{
			Provider:     strings.ToLower(name),
			Exchange:     name,
			Symbol:       canonical,
			BaseAsset:    base,
			QuoteAsset:   quote,
			MarketType:   exchange.MarketFutures,
			AssetClass:   "crypto",
			Bids:         []exchange.OrderBookLevel{{Price: bid, Quantity: decimal.NewFromInt(1)}},
			Asks:         []exchange.OrderBookLevel{{Price: ask, Quantity: decimal.NewFromInt(1)}},
			UpdatedAt:    now,
			LimitedDepth: true,
		}, c.orderBookDepth)
		if rate, ok := exchange.DecimalFromString(string(item.FundingRate)); ok {
			c.fundingRates[canonical] = exchange.FundingRate{
				Exchange:  name,
				Symbol:    canonical,
				Rate:      rate,
				UpdatedAt: now,
			}
		}
	}
	// TODO: Enrich Kraken futures funding next time from the dedicated historical/real-time funding endpoints.
	c.health.LastMessageTime = now
	c.health.LastError = ""
	return nil
}

func (c *Client) pollSpotOrderBooks(ctx context.Context) error {
	pairs := c.spotPairs()
	for restName, canonical := range pairs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := c.pollSpotOrderBook(ctx, restName, canonical); err != nil {
			c.setError(err)
		}
	}
	return nil
}

func (c *Client) pollSpotOrderBook(ctx context.Context, restName, canonical string) error {
	url := fmt.Sprintf("%s/Depth?pair=%s&count=%d", spotRESTBase, restName, c.orderBookDepth)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	var payload depthResponse
	if err := c.doJSON(req, &payload); err != nil {
		return err
	}
	if len(payload.Errors) > 0 {
		return fmt.Errorf("kraken depth: %s", strings.Join(payload.Errors, ", "))
	}
	var depth krakenDepthPayload
	for _, item := range payload.Result {
		depth = item
		break
	}
	base, quote := splitCanonical(canonical)
	book := exchange.OrderBook{
		Provider:   strings.ToLower(name),
		Exchange:   name,
		Symbol:     canonical,
		BaseAsset:  base,
		QuoteAsset: quote,
		MarketType: exchange.MarketSpot,
		AssetClass: "crypto",
		Bids:       parseDepthLevels(depth.Bids),
		Asks:       parseDepthLevels(depth.Asks),
		UpdatedAt:  time.Now(),
	}
	c.mu.Lock()
	c.orderBooks[orderBookKey(book)] = exchange.NormalizeOrderBook(book, c.orderBookDepth)
	c.mu.Unlock()
	return nil
}

func (c *Client) websocketLoop(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		if err := c.runWebSocket(ctx); err != nil {
			c.setError(err)
			c.recordReconnect()
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}
	}
}

func (c *Client) runWebSocket(ctx context.Context) error {
	symbols := c.websocketSymbols()
	if len(symbols) == 0 {
		return fmt.Errorf("no kraken websocket symbols configured")
	}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, spotWSURL, nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}
	defer conn.Close()
	c.markWSConnected()
	defer c.markWSDisconnected()

	subscribe := map[string]any{
		"method": "subscribe",
		"params": map[string]any{
			"channel":  "ticker",
			"symbol":   symbols,
			"snapshot": true,
		},
	}
	if err := conn.WriteJSON(subscribe); err != nil {
		return fmt.Errorf("websocket subscribe: %w", err)
	}
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("websocket read: %w", err)
		}
		c.applyWebSocketMessage(data)
	}
}

func (c *Client) applyWebSocketMessage(data []byte) {
	var msg krakenWSMessage
	if err := json.Unmarshal(data, &msg); err != nil || msg.Channel != "ticker" {
		return
	}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, row := range msg.Data {
		canonical := exchange.NormalizeCanonicalSymbol(row.Symbol)
		if !c.knownPair(canonical) {
			continue
		}
		bid, bidOK := exchange.DecimalFromAny(row.Bid)
		ask, askOK := exchange.DecimalFromAny(row.Ask)
		last, _ := exchange.DecimalFromAny(row.Last)
		if !bidOK || !askOK || !exchange.ValidBidAsk(bid, ask) {
			continue
		}
		base, quote := splitCanonical(canonical)
		c.spotTickers[canonical] = exchange.Ticker{
			Provider:   strings.ToLower(name),
			Exchange:   name,
			Symbol:     canonical,
			BaseAsset:  base,
			QuoteAsset: quote,
			MarketType: exchange.MarketSpot,
			AssetClass: "crypto",
			Bid:        bid,
			Ask:        ask,
			Last:       last,
			UpdatedAt:  now,
		}
		c.markets[string(exchange.MarketSpot)+"|"+canonical] = exchange.MarketInfo{
			Provider:   strings.ToLower(name),
			Exchange:   name,
			Symbol:     canonical,
			BaseAsset:  base,
			QuoteAsset: quote,
			AssetClass: "crypto",
			MarketType: exchange.MarketSpot,
			Active:     true,
		}
	}
	c.health.LastMessageTime = now
	c.health.LastError = ""
}

func (c *Client) doJSON(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("http %s: %s", req.URL.String(), resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) knownPair(symbol string) bool {
	base, quote := splitCanonical(symbol)
	if base == "" || quote == "" {
		return false
	}
	return containsAsset(c.knownAssets, base) && containsAsset(c.knownAssets, quote)
}

func (c *Client) websocketSymbols() []string {
	c.mu.RLock()
	var mapped []string
	seenMapped := make(map[string]struct{})
	for _, symbol := range c.restToCanonical {
		if _, ok := seenMapped[symbol]; ok {
			continue
		}
		seenMapped[symbol] = struct{}{}
		mapped = append(mapped, symbol)
	}
	c.mu.RUnlock()
	if len(mapped) > 0 {
		sort.Strings(mapped)
		if len(mapped) > 50 {
			return mapped[:50]
		}
		return mapped
	}
	var symbols []string
	seen := make(map[string]struct{})
	for _, base := range c.knownAssets {
		for _, quote := range c.knownAssets {
			if base == quote {
				continue
			}
			if quote != "USD" && quote != "USDT" && quote != "USDC" && quote != "EUR" && quote != "BTC" && quote != "ETH" {
				continue
			}
			symbol := exchange.CanonicalSymbol(base, quote)
			if _, ok := seen[symbol]; ok {
				continue
			}
			seen[symbol] = struct{}{}
			symbols = append(symbols, symbol)
		}
	}
	sort.Strings(symbols)
	if len(symbols) > 50 {
		return symbols[:50]
	}
	return symbols
}

func (c *Client) setError(err error) {
	if err == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.health.LastError = err.Error()
}

func (c *Client) markWSConnected() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.wsConnections++
	c.health.LastMessageTime = time.Now()
}

func (c *Client) markWSDisconnected() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.wsConnections > 0 {
		c.wsConnections--
	}
}

func (c *Client) recordReconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.health.ReconnectCount++
}

type assetPairsResponse struct {
	Errors []string                    `json:"error"`
	Result map[string]assetPairPayload `json:"result"`
}

type assetPairPayload struct {
	WSName string `json:"wsname"`
}

type tickerResponse struct {
	Errors []string                       `json:"error"`
	Result map[string]krakenTickerPayload `json:"result"`
}

type depthResponse struct {
	Errors []string                      `json:"error"`
	Result map[string]krakenDepthPayload `json:"result"`
}

type krakenDepthPayload struct {
	Asks [][]json.RawMessage `json:"asks"`
	Bids [][]json.RawMessage `json:"bids"`
}

type krakenTickerPayload struct {
	Ask   []string `json:"a"`
	Bid   []string `json:"b"`
	Close []string `json:"c"`
}

type futuresTickerResponse struct {
	Tickers []krakenFuturesTicker `json:"tickers"`
}

type krakenFuturesTicker struct {
	Symbol      string               `json:"symbol"`
	Pair        string               `json:"pair"`
	Bid         krakenFlexibleString `json:"bid"`
	Ask         krakenFlexibleString `json:"ask"`
	BestBid     krakenFlexibleString `json:"bestBid"`
	BestAsk     krakenFlexibleString `json:"bestAsk"`
	Last        krakenFlexibleString `json:"last"`
	LastAlias   krakenFlexibleString `json:"la"`
	FundingRate krakenFlexibleString `json:"fundingRate"`
}

type krakenFlexibleString string

func (v *krakenFlexibleString) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*v = ""
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*v = krakenFlexibleString(text)
		return nil
	}
	var number json.Number
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&number); err != nil {
		return err
	}
	*v = krakenFlexibleString(number.String())
	return nil
}

type krakenWSMessage struct {
	Channel string          `json:"channel"`
	Data    []krakenWSQuote `json:"data"`
}

type krakenWSQuote struct {
	Symbol string `json:"symbol"`
	Bid    any    `json:"bid"`
	Ask    any    `json:"ask"`
	Last   any    `json:"last"`
}

func parseKrakenTicker(item krakenTickerPayload) (bid decimal.Decimal, ask decimal.Decimal, last decimal.Decimal, ok bool) {
	if len(item.Bid) == 0 || len(item.Ask) == 0 {
		return decimal.Zero, decimal.Zero, decimal.Zero, false
	}
	bid, bidOK := exchange.DecimalFromString(item.Bid[0])
	ask, askOK := exchange.DecimalFromString(item.Ask[0])
	if len(item.Close) > 0 {
		last, _ = exchange.DecimalFromString(item.Close[0])
	}
	return bid, ask, last, bidOK && askOK && exchange.ValidBidAsk(bid, ask)
}

func canonicalFromKrakenFutures(item krakenFuturesTicker) (string, bool) {
	if item.Pair != "" {
		return exchange.NormalizeCanonicalSymbol(strings.ReplaceAll(item.Pair, ":", "/")), true
	}
	symbol := strings.ToUpper(item.Symbol)
	symbol = strings.TrimPrefix(symbol, "PI_")
	symbol = strings.TrimPrefix(symbol, "PF_")
	base, quote, canonical, ok := exchange.SplitJoinedSymbol(symbol, []string{"XBT", "BTC", "ETH", "SOL", "USDT", "USDC", "USD", "EUR"})
	if ok {
		return exchange.CanonicalSymbol(base, quote), true
	}
	return canonical, ok
}

func splitCanonical(symbol string) (string, string) {
	parts := strings.Split(exchange.NormalizeCanonicalSymbol(symbol), "/")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func parseDepthLevels(levels [][]json.RawMessage) []exchange.OrderBookLevel {
	out := make([]exchange.OrderBookLevel, 0, len(levels))
	for _, row := range levels {
		if len(row) < 2 {
			continue
		}
		price, priceOK := decimalFromDepthValue(row[0])
		qty, qtyOK := decimalFromDepthValue(row[1])
		if !priceOK || !qtyOK || price.LessThanOrEqual(decimal.Zero) || qty.LessThanOrEqual(decimal.Zero) {
			continue
		}
		out = append(out, exchange.OrderBookLevel{Price: price, Quantity: qty})
	}
	return out
}

func decimalFromDepthValue(value json.RawMessage) (decimal.Decimal, bool) {
	var asString string
	if err := json.Unmarshal(value, &asString); err == nil {
		return exchange.DecimalFromString(asString)
	}
	return exchange.DecimalFromString(string(value))
}

func (c *Client) spotPairs() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]string, len(c.restToCanonical))
	for restName, canonical := range c.restToCanonical {
		out[restName] = canonical
	}
	return out
}

func orderBookKey(book exchange.OrderBook) string {
	return string(book.MarketType) + "|" + book.Symbol
}

func countStaleTickers(spot map[string]exchange.Ticker, futures map[string]exchange.Ticker, staleAfter time.Duration) int {
	var count int
	for _, ticker := range spot {
		if !health.Fresh(ticker.UpdatedAt, staleAfter) {
			count++
		}
	}
	for _, ticker := range futures {
		if !health.Fresh(ticker.UpdatedAt, staleAfter) {
			count++
		}
	}
	return count
}

func countStaleOrderBooks(orderBooks map[string]exchange.OrderBook, staleAfter time.Duration) int {
	var count int
	for _, book := range orderBooks {
		if !health.Fresh(book.UpdatedAt, staleAfter) {
			count++
		}
	}
	return count
}

func containsAsset(assets []string, target string) bool {
	target = exchange.NormalizeAsset(target)
	for _, asset := range assets {
		if exchange.NormalizeAsset(asset) == target {
			return true
		}
	}
	return false
}

func cloneTickers(in map[string]exchange.Ticker) []exchange.Ticker {
	out := make([]exchange.Ticker, 0, len(in))
	for _, ticker := range in {
		out = append(out, ticker)
	}
	return out
}
