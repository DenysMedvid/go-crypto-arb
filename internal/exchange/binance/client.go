package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
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
	name            = "Binance"
	spotRESTBase    = "https://api.binance.com"
	futuresRESTBase = "https://fapi.binance.com"
	spotWSBase      = "wss://stream.binance.com:9443/stream?streams="
	futuresWSBase   = "wss://fstream.binance.com/stream?streams="
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

	mu             sync.RWMutex
	spotTickers    map[string]exchange.Ticker
	futuresTickers map[string]exchange.Ticker
	orderBooks     map[string]exchange.OrderBook
	fundingRates   map[string]exchange.FundingRate
	markets        map[string]exchange.MarketInfo
	health         exchange.ExchangeHealth
	wsConnections  int
}

func New(cfg config.ExchangeConfig, knownAssets []string, staleAfter time.Duration, bookStaleAfter time.Duration, orderBookDepth int, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	if orderBookDepth <= 0 {
		orderBookDepth = 20
	}
	return &Client{
		cfg:            cfg,
		knownAssets:    knownAssets,
		staleAfter:     staleAfter,
		bookStaleAfter: bookStaleAfter,
		orderBookDepth: orderBookDepth,
		logger:         logger.With("exchange", name),
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		spotTickers:    make(map[string]exchange.Ticker),
		futuresTickers: make(map[string]exchange.Ticker),
		orderBooks:     make(map[string]exchange.OrderBook),
		fundingRates:   make(map[string]exchange.FundingRate),
		markets:        make(map[string]exchange.MarketInfo),
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
		if err := c.pollSpot(runCtx); err != nil {
			c.setError(err)
		}
		if err := c.pollOrderBooks(runCtx, exchange.MarketSpot); err != nil {
			c.setError(err)
		}
		go c.pollLoop(runCtx, c.pollSpot)
		go c.pollLoop(runCtx, func(ctx context.Context) error { return c.pollOrderBooks(ctx, exchange.MarketSpot) })
		if c.cfg.WebsocketEnabled {
			go c.websocketLoop(runCtx, c.bookTickerStreamURL(spotWSBase, exchange.MarketSpot), exchange.MarketSpot)
		}
	}
	if c.cfg.FuturesEnabled {
		if err := c.pollFutures(runCtx); err != nil {
			c.setError(err)
		}
		if err := c.pollOrderBooks(runCtx, exchange.MarketFutures); err != nil {
			c.setError(err)
		}
		go c.pollLoop(runCtx, c.pollFutures)
		go c.pollLoop(runCtx, func(ctx context.Context) error { return c.pollOrderBooks(ctx, exchange.MarketFutures) })
		go c.pollLoop(runCtx, c.pollFundingRates)
		if c.cfg.WebsocketEnabled {
			go c.websocketLoop(runCtx, c.bookTickerStreamURL(futuresWSBase, exchange.MarketFutures), exchange.MarketFutures)
		}
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

func (c *Client) pollSpot(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spotRESTBase+"/api/v3/ticker/bookTicker", nil)
	if err != nil {
		return err
	}
	var payload []bookTickerPayload
	if err := c.doJSON(req, &payload); err != nil {
		return err
	}
	c.applyBookTickers(payload, exchange.MarketSpot)
	return nil
}

func (c *Client) pollFutures(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, futuresRESTBase+"/fapi/v1/ticker/bookTicker", nil)
	if err != nil {
		return err
	}
	var payload []bookTickerPayload
	if err := c.doJSON(req, &payload); err != nil {
		return err
	}
	c.applyBookTickers(payload, exchange.MarketFutures)
	return nil
}

func (c *Client) pollFundingRates(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, futuresRESTBase+"/fapi/v1/premiumIndex", nil)
	if err != nil {
		return err
	}
	var payload []premiumIndexPayload
	if err := c.doJSON(req, &payload); err != nil {
		return err
	}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, item := range payload {
		_, _, canonical, ok := exchange.SplitJoinedSymbol(item.Symbol, c.knownAssets)
		if !ok {
			continue
		}
		rate, ok := exchange.DecimalFromString(item.LastFundingRate)
		if !ok {
			continue
		}
		c.fundingRates[canonical] = exchange.FundingRate{
			Exchange:        name,
			Symbol:          canonical,
			Rate:            rate,
			NextFundingTime: time.UnixMilli(item.NextFundingTime),
			UpdatedAt:       now,
		}
	}
	c.health.LastMessageTime = now
	c.health.LastError = ""
	return nil
}

func (c *Client) pollOrderBooks(ctx context.Context, marketType exchange.MarketType) error {
	tickers := c.tickersForMarket(marketType)
	for _, ticker := range tickers {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := c.pollOrderBook(ctx, ticker, marketType); err != nil {
			c.setError(err)
		}
	}
	return nil
}

func (c *Client) pollOrderBook(ctx context.Context, ticker exchange.Ticker, marketType exchange.MarketType) error {
	baseURL := spotRESTBase + "/api/v3/depth"
	if marketType == exchange.MarketFutures {
		baseURL = futuresRESTBase + "/fapi/v1/depth"
	}
	url := fmt.Sprintf("%s?symbol=%s&limit=%d", baseURL, exchange.JoinedSymbol(ticker.Symbol), c.orderBookDepth)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	var payload depthPayload
	if err := c.doJSON(req, &payload); err != nil {
		return err
	}
	book := exchange.OrderBook{
		Provider:   strings.ToLower(name),
		Exchange:   name,
		Symbol:     ticker.Symbol,
		BaseAsset:  ticker.BaseAsset,
		QuoteAsset: ticker.QuoteAsset,
		MarketType: marketType,
		AssetClass: "crypto",
		Bids:       parseDepthLevels(payload.Bids),
		Asks:       parseDepthLevels(payload.Asks),
		UpdatedAt:  time.Now(),
	}
	c.mu.Lock()
	c.orderBooks[orderBookKey(book)] = exchange.NormalizeOrderBook(book, c.orderBookDepth)
	c.mu.Unlock()
	return nil
}

func (c *Client) websocketLoop(ctx context.Context, endpoint string, marketType exchange.MarketType) {
	for {
		if ctx.Err() != nil {
			return
		}
		if err := c.runWebSocket(ctx, endpoint, marketType); err != nil {
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

func (c *Client) runWebSocket(ctx context.Context, endpoint string, marketType exchange.MarketType) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}
	defer conn.Close()
	c.markWSConnected()
	defer c.markWSDisconnected()

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("websocket read: %w", err)
		}
		var combined combinedBookTickerPayload
		if err := json.Unmarshal(data, &combined); err == nil && combined.Data.symbol() != "" {
			c.applyBookTickers([]bookTickerPayload{combined.Data}, marketType)
			continue
		}
		var one bookTickerPayload
		if err := json.Unmarshal(data, &one); err == nil && one.symbol() != "" {
			c.applyBookTickers([]bookTickerPayload{one}, marketType)
			continue
		}
		var many []bookTickerPayload
		if err := json.Unmarshal(data, &many); err == nil {
			c.applyBookTickers(many, marketType)
		}
	}
}

func (c *Client) bookTickerStreamURL(baseURL string, marketType exchange.MarketType) string {
	var streams []string
	c.mu.RLock()
	source := c.spotTickers
	if marketType == exchange.MarketFutures {
		source = c.futuresTickers
	}
	for symbol := range source {
		streams = append(streams, strings.ToLower(exchange.JoinedSymbol(symbol))+"@bookTicker")
	}
	c.mu.RUnlock()
	if len(streams) > 0 {
		return baseURL + strings.Join(streams, "/")
	}
	for _, base := range c.knownAssets {
		for _, quote := range c.knownAssets {
			if base == quote {
				continue
			}
			symbol := strings.ToLower(exchange.JoinedSymbol(exchange.CanonicalSymbol(base, quote)))
			streams = append(streams, symbol+"@bookTicker")
		}
	}
	if len(streams) == 0 {
		return baseURL
	}
	return baseURL + strings.Join(streams, "/")
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

func (c *Client) applyBookTickers(payload []bookTickerPayload, marketType exchange.MarketType) {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, item := range payload {
		base, quote, canonical, ok := exchange.SplitJoinedSymbol(item.symbol(), c.knownAssets)
		if !ok {
			continue
		}
		bid, bidOK := exchange.DecimalFromString(item.bidPrice())
		ask, askOK := exchange.DecimalFromString(item.askPrice())
		if !bidOK || !askOK || !exchange.ValidBidAsk(bid, ask) {
			continue
		}
		ticker := exchange.Ticker{
			Provider:   strings.ToLower(name),
			Exchange:   name,
			Symbol:     canonical,
			BaseAsset:  base,
			QuoteAsset: quote,
			MarketType: marketType,
			AssetClass: "crypto",
			Bid:        bid,
			Ask:        ask,
			Last:       bid.Add(ask).Div(decimal.NewFromInt(2)),
			UpdatedAt:  now,
		}
		if marketType == exchange.MarketSpot {
			c.spotTickers[canonical] = ticker
		} else {
			c.futuresTickers[canonical] = ticker
		}
		c.markets[string(marketType)+"|"+canonical] = exchange.MarketInfo{
			Provider:   strings.ToLower(name),
			Exchange:   name,
			Symbol:     canonical,
			BaseAsset:  base,
			QuoteAsset: quote,
			MarketType: marketType,
			AssetClass: "crypto",
			Active:     true,
		}
	}
	c.health.LastMessageTime = now
	c.health.LastError = ""
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

type bookTickerPayload struct {
	Symbol       string `json:"s"`
	RESTSymbol   string `json:"symbol"`
	BidPrice     string `json:"b"`
	RESTBidPrice string `json:"bidPrice"`
	BidQty       string `json:"B"`
	RESTBidQty   string `json:"bidQty"`
	AskPrice     string `json:"a"`
	RESTAskPrice string `json:"askPrice"`
	AskQty       string `json:"A"`
	RESTAskQty   string `json:"askQty"`
}

type combinedBookTickerPayload struct {
	Data bookTickerPayload `json:"data"`
}

func (p bookTickerPayload) symbol() string {
	if p.Symbol != "" {
		return p.Symbol
	}
	return p.RESTSymbol
}

func (p bookTickerPayload) bidPrice() string {
	if p.BidPrice != "" {
		return p.BidPrice
	}
	return p.RESTBidPrice
}

func (p bookTickerPayload) askPrice() string {
	if p.AskPrice != "" {
		return p.AskPrice
	}
	return p.RESTAskPrice
}

type premiumIndexPayload struct {
	Symbol          string `json:"symbol"`
	LastFundingRate string `json:"lastFundingRate"`
	NextFundingTime int64  `json:"nextFundingTime"`
}

type depthPayload struct {
	Bids [][]string `json:"bids"`
	Asks [][]string `json:"asks"`
}

func parseDepthLevels(levels [][]string) []exchange.OrderBookLevel {
	out := make([]exchange.OrderBookLevel, 0, len(levels))
	for _, row := range levels {
		if len(row) < 2 {
			continue
		}
		price, priceOK := exchange.DecimalFromString(row[0])
		qty, qtyOK := exchange.DecimalFromString(row[1])
		if !priceOK || !qtyOK || price.LessThanOrEqual(decimal.Zero) || qty.LessThanOrEqual(decimal.Zero) {
			continue
		}
		out = append(out, exchange.OrderBookLevel{Price: price, Quantity: qty})
	}
	return out
}

func (c *Client) tickersForMarket(marketType exchange.MarketType) []exchange.Ticker {
	c.mu.RLock()
	defer c.mu.RUnlock()
	source := c.spotTickers
	if marketType == exchange.MarketFutures {
		source = c.futuresTickers
	}
	return cloneTickers(source)
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

func cloneTickers(in map[string]exchange.Ticker) []exchange.Ticker {
	out := make([]exchange.Ticker, 0, len(in))
	for _, ticker := range in {
		out = append(out, ticker)
	}
	return out
}
