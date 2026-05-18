package publicrest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
	"go-crypto-arb/internal/health"
)

type exchangeSpec struct {
	name             string
	provider         string
	tickersURL       string
	productsURL      string
	tickerURL        func(string) string
	orderBookURL     func(string, int) string
	parseTickers     func([]byte, []string) ([]tickerUpdate, error)
	parseTicker      func([]byte, string) (tickerUpdate, bool, error)
	parseProducts    func([]byte, []string) ([]string, error)
	parseOrderBook   func([]byte) ([]exchange.OrderBookLevel, []exchange.OrderBookLevel, error)
	futuresSupported bool
}

type tickerUpdate struct {
	Symbol string
	Bid    decimal.Decimal
	Ask    decimal.Decimal
	Last   decimal.Decimal
}

type Client struct {
	cfg            config.ExchangeConfig
	spec           exchangeSpec
	knownAssets    []string
	staleAfter     time.Duration
	bookStaleAfter time.Duration
	orderBookDepth int
	logger         *slog.Logger
	httpClient     *http.Client

	cancel context.CancelFunc

	mu          sync.RWMutex
	spotTickers map[string]exchange.Ticker
	orderBooks  map[string]exchange.OrderBook
	markets     map[string]exchange.MarketInfo
	health      exchange.ExchangeHealth
}

func New(platform string, cfg config.ExchangeConfig, knownAssets []string, staleAfter, bookStaleAfter time.Duration, orderBookDepth int, logger *slog.Logger) (*Client, bool) {
	spec, ok := specs()[strings.ToLower(platform)]
	if !ok {
		return nil, false
	}
	if logger == nil {
		logger = slog.Default()
	}
	if orderBookDepth <= 0 {
		orderBookDepth = 20
	}
	return &Client{
		cfg:            cfg,
		spec:           spec,
		knownAssets:    knownAssets,
		staleAfter:     staleAfter,
		bookStaleAfter: bookStaleAfter,
		orderBookDepth: orderBookDepth,
		logger:         logger.With("exchange", spec.name),
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		spotTickers:    make(map[string]exchange.Ticker),
		orderBooks:     make(map[string]exchange.OrderBook),
		markets:        make(map[string]exchange.MarketInfo),
		health: exchange.ExchangeHealth{
			Provider:         spec.provider,
			ProviderType:     "crypto_exchange",
			Exchange:         spec.name,
			Enabled:          cfg.Enabled,
			SpotEnabled:      cfg.SpotEnabled,
			FuturesEnabled:   cfg.FuturesEnabled,
			WebSocketEnabled: cfg.WebsocketEnabled,
			PartialSupport:   (cfg.FuturesEnabled && !spec.futuresSupported) || cfg.WebsocketEnabled,
			Status:           "starting",
		},
	}, true
}

func (c *Client) Name() string { return c.spec.name }

func (c *Client) Type() string { return "crypto_exchange" }

func (c *Client) Start(ctx context.Context) error {
	if !c.cfg.Enabled {
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	if c.cfg.FuturesEnabled && !c.spec.futuresSupported {
		c.setError(fmt.Errorf("%s futures support is not implemented; spot market data only", c.spec.name))
	}
	if c.cfg.WebsocketEnabled {
		c.setError(fmt.Errorf("%s websocket support is not implemented; REST polling is used", c.spec.name))
	}
	if c.cfg.SpotEnabled {
		if err := c.pollSpot(runCtx); err != nil {
			c.setError(err)
		}
		if err := c.pollOrderBooks(runCtx); err != nil {
			c.setError(err)
		}
		go c.pollLoop(runCtx, c.pollSpot)
		go c.pollLoop(runCtx, c.pollOrderBooks)
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
	h.RestFallbackActive = true
	h.DataFresh = health.Fresh(h.LastMessageTime, c.staleAfter)
	h.LastMessageAt = h.LastMessageTime
	h.StaleTickerCount = countStaleTickers(c.spotTickers, c.staleAfter)
	h.StaleOrderBookCount = countStaleOrderBooks(c.orderBooks, c.bookStaleAfter)
	h.PartialSupport = h.PartialSupport || (c.cfg.FuturesEnabled && !c.spec.futuresSupported) || c.cfg.WebsocketEnabled
	switch {
	case !h.Enabled:
		h.Status = "disabled"
	case !h.DataFresh:
		h.Status = "stale"
	case h.PartialSupport:
		h.Status = "partial"
	default:
		h.Status = "rest-fallback"
	}
	return h
}

func (c *Client) GetLatestTickers() []exchange.Ticker {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return cloneTickers(c.spotTickers)
}

func (c *Client) GetLatestFuturesTickers() []exchange.Ticker { return nil }

func (c *Client) GetFundingRates() []exchange.FundingRate { return nil }

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
	var updates []tickerUpdate
	var err error
	if c.spec.tickersURL != "" {
		var data []byte
		data, err = c.get(ctx, c.spec.tickersURL)
		if err == nil {
			updates, err = c.spec.parseTickers(data, c.knownAssets)
		}
	} else {
		updates, err = c.pollProductTickers(ctx)
	}
	if err != nil {
		return err
	}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, update := range updates {
		base, quote := splitCanonical(update.Symbol)
		if base == "" || quote == "" || !exchange.ValidBidAsk(update.Bid, update.Ask) {
			continue
		}
		last := update.Last
		if last.IsZero() {
			last = update.Bid.Add(update.Ask).Div(decimal.NewFromInt(2))
		}
		c.spotTickers[update.Symbol] = exchange.Ticker{
			Provider:   c.spec.provider,
			Exchange:   c.spec.name,
			Symbol:     update.Symbol,
			BaseAsset:  base,
			QuoteAsset: quote,
			MarketType: exchange.MarketSpot,
			AssetClass: "crypto",
			Bid:        update.Bid,
			Ask:        update.Ask,
			Last:       last,
			UpdatedAt:  now,
		}
		c.markets[string(exchange.MarketSpot)+"|"+update.Symbol] = exchange.MarketInfo{
			Provider:   c.spec.provider,
			Exchange:   c.spec.name,
			Symbol:     update.Symbol,
			BaseAsset:  base,
			QuoteAsset: quote,
			AssetClass: "crypto",
			MarketType: exchange.MarketSpot,
			Active:     true,
		}
	}
	if len(updates) > 0 {
		c.health.LastMessageTime = now
		c.health.LastError = ""
	}
	return nil
}

func (c *Client) pollProductTickers(ctx context.Context) ([]tickerUpdate, error) {
	data, err := c.get(ctx, c.spec.productsURL)
	if err != nil {
		return nil, err
	}
	symbols, err := c.spec.parseProducts(data, c.knownAssets)
	if err != nil {
		return nil, err
	}
	var updates []tickerUpdate
	var lastErr error
	for _, symbol := range symbols {
		if ctx.Err() != nil {
			return updates, ctx.Err()
		}
		data, err := c.get(ctx, c.spec.tickerURL(symbol))
		if err != nil {
			lastErr = err
			continue
		}
		update, ok, err := c.spec.parseTicker(data, symbol)
		if err != nil {
			lastErr = err
			continue
		}
		if ok {
			updates = append(updates, update)
		}
	}
	if len(updates) == 0 && lastErr != nil {
		return nil, lastErr
	}
	return updates, nil
}

func (c *Client) pollOrderBooks(ctx context.Context) error {
	tickers := c.GetLatestTickers()
	for _, ticker := range tickers {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := c.pollOrderBook(ctx, ticker); err != nil {
			c.setError(err)
		}
	}
	return nil
}

func (c *Client) pollOrderBook(ctx context.Context, ticker exchange.Ticker) error {
	data, err := c.get(ctx, c.spec.orderBookURL(ticker.Symbol, c.orderBookDepth))
	if err != nil {
		return err
	}
	bids, asks, err := c.spec.parseOrderBook(data)
	if err != nil {
		return err
	}
	book := exchange.OrderBook{
		Provider:   c.spec.provider,
		Exchange:   c.spec.name,
		Symbol:     ticker.Symbol,
		BaseAsset:  ticker.BaseAsset,
		QuoteAsset: ticker.QuoteAsset,
		MarketType: exchange.MarketSpot,
		AssetClass: "crypto",
		Bids:       bids,
		Asks:       asks,
		UpdatedAt:  time.Now(),
	}
	c.mu.Lock()
	c.orderBooks[string(book.MarketType)+"|"+book.Symbol] = exchange.NormalizeOrderBook(book, c.orderBookDepth)
	c.mu.Unlock()
	return nil
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "go-crypto-arb")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %s: %s", url, resp.Status)
	}
	return data, nil
}

func (c *Client) setError(err error) {
	if err == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.health.LastError = err.Error()
}

func specs() map[string]exchangeSpec {
	return map[string]exchangeSpec{
		"okx": {
			name:       "OKX",
			provider:   "okx",
			tickersURL: "https://www.okx.com/api/v5/market/tickers?instType=SPOT",
			orderBookURL: func(symbol string, depth int) string {
				return fmt.Sprintf("https://www.okx.com/api/v5/market/books?instId=%s&sz=%d", strings.ReplaceAll(symbol, "/", "-"), depth)
			},
			parseTickers:   parseOKXTickers,
			parseOrderBook: parseOKXOrderBook,
		},
		"bybit": {
			name:       "Bybit",
			provider:   "bybit",
			tickersURL: "https://api.bybit.com/v5/market/tickers?category=spot",
			orderBookURL: func(symbol string, depth int) string {
				return fmt.Sprintf("https://api.bybit.com/v5/market/orderbook?category=spot&symbol=%s&limit=%d", exchange.JoinedSymbol(symbol), depth)
			},
			parseTickers:   parseBybitTickers,
			parseOrderBook: parseBybitOrderBook,
		},
		"coinbase": {
			name:        "Coinbase",
			provider:    "coinbase",
			productsURL: "https://api.exchange.coinbase.com/products",
			tickerURL: func(symbol string) string {
				return "https://api.exchange.coinbase.com/products/" + strings.ReplaceAll(symbol, "/", "-") + "/ticker"
			},
			orderBookURL: func(symbol string, depth int) string {
				return "https://api.exchange.coinbase.com/products/" + strings.ReplaceAll(symbol, "/", "-") + "/book?level=2"
			},
			parseProducts:  parseCoinbaseProducts,
			parseTicker:    parseCoinbaseTicker,
			parseOrderBook: parseCoinbaseOrderBook,
		},
		"gateio": {
			name:       "Gate.io",
			provider:   "gateio",
			tickersURL: "https://api.gateio.ws/api/v4/spot/tickers",
			orderBookURL: func(symbol string, depth int) string {
				return fmt.Sprintf("https://api.gateio.ws/api/v4/spot/order_book?currency_pair=%s&limit=%d", strings.ReplaceAll(symbol, "/", "_"), depth)
			},
			parseTickers:   parseGateIOTickers,
			parseOrderBook: parseGateIOOrderBook,
		},
		"bitget": {
			name:       "Bitget",
			provider:   "bitget",
			tickersURL: "https://api.bitget.com/api/v2/spot/market/tickers",
			orderBookURL: func(symbol string, depth int) string {
				return fmt.Sprintf("https://api.bitget.com/api/v2/spot/market/orderbook?symbol=%s&type=step0&limit=%d", exchange.JoinedSymbol(symbol), depth)
			},
			parseTickers:   parseBitgetTickers,
			parseOrderBook: parseBitgetOrderBook,
		},
	}
}

func parseOKXTickers(data []byte, knownAssets []string) ([]tickerUpdate, error) {
	var payload struct {
		Data []struct {
			InstID string `json:"instId"`
			Bid    string `json:"bidPx"`
			Ask    string `json:"askPx"`
			Last   string `json:"last"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return parseTickerRows(payload.Data, knownAssets, func(row struct {
		InstID string `json:"instId"`
		Bid    string `json:"bidPx"`
		Ask    string `json:"askPx"`
		Last   string `json:"last"`
	}) (string, string, string, string) {
		return row.InstID, row.Bid, row.Ask, row.Last
	}), nil
}

func parseBybitTickers(data []byte, knownAssets []string) ([]tickerUpdate, error) {
	var payload struct {
		Result struct {
			List []struct {
				Symbol string `json:"symbol"`
				Bid    string `json:"bid1Price"`
				Ask    string `json:"ask1Price"`
				Last   string `json:"lastPrice"`
			} `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return parseTickerRows(payload.Result.List, knownAssets, func(row struct {
		Symbol string `json:"symbol"`
		Bid    string `json:"bid1Price"`
		Ask    string `json:"ask1Price"`
		Last   string `json:"lastPrice"`
	}) (string, string, string, string) {
		return row.Symbol, row.Bid, row.Ask, row.Last
	}), nil
}

func parseCoinbaseProducts(data []byte, knownAssets []string) ([]string, error) {
	var products []struct {
		ID              string `json:"id"`
		BaseCurrency    string `json:"base_currency"`
		QuoteCurrency   string `json:"quote_currency"`
		Status          string `json:"status"`
		TradingDisabled bool   `json:"trading_disabled"`
	}
	if err := json.Unmarshal(data, &products); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	var out []string
	for _, product := range products {
		if product.TradingDisabled || product.Status != "online" {
			continue
		}
		base := exchange.NormalizeAsset(product.BaseCurrency)
		quote := exchange.NormalizeAsset(product.QuoteCurrency)
		if !knownAsset(knownAssets, base) || !knownAsset(knownAssets, quote) {
			continue
		}
		symbol := exchange.CanonicalSymbol(base, quote)
		if _, ok := seen[symbol]; ok {
			continue
		}
		seen[symbol] = struct{}{}
		out = append(out, symbol)
	}
	sort.Strings(out)
	return out, nil
}

func parseCoinbaseTicker(data []byte, symbol string) (tickerUpdate, bool, error) {
	var payload struct {
		Bid  string `json:"bid"`
		Ask  string `json:"ask"`
		Last string `json:"price"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return tickerUpdate{}, false, err
	}
	update, ok := buildTickerUpdate(symbol, payload.Bid, payload.Ask, payload.Last, nil)
	return update, ok, nil
}

func parseGateIOTickers(data []byte, knownAssets []string) ([]tickerUpdate, error) {
	var payload []struct {
		Symbol string `json:"currency_pair"`
		Bid    string `json:"highest_bid"`
		Ask    string `json:"lowest_ask"`
		Last   string `json:"last"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return parseTickerRows(payload, knownAssets, func(row struct {
		Symbol string `json:"currency_pair"`
		Bid    string `json:"highest_bid"`
		Ask    string `json:"lowest_ask"`
		Last   string `json:"last"`
	}) (string, string, string, string) {
		return strings.ReplaceAll(row.Symbol, "_", "/"), row.Bid, row.Ask, row.Last
	}), nil
}

func parseBitgetTickers(data []byte, knownAssets []string) ([]tickerUpdate, error) {
	var payload struct {
		Data []struct {
			Symbol string `json:"symbol"`
			Bid    string `json:"bidPr"`
			Ask    string `json:"askPr"`
			Last   string `json:"lastPr"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return parseTickerRows(payload.Data, knownAssets, func(row struct {
		Symbol string `json:"symbol"`
		Bid    string `json:"bidPr"`
		Ask    string `json:"askPr"`
		Last   string `json:"lastPr"`
	}) (string, string, string, string) {
		return row.Symbol, row.Bid, row.Ask, row.Last
	}), nil
}

func parseOKXOrderBook(data []byte) ([]exchange.OrderBookLevel, []exchange.OrderBookLevel, error) {
	var payload struct {
		Data []struct {
			Asks [][]string `json:"asks"`
			Bids [][]string `json:"bids"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, nil, err
	}
	if len(payload.Data) == 0 {
		return nil, nil, nil
	}
	return parseStringLevels(payload.Data[0].Bids), parseStringLevels(payload.Data[0].Asks), nil
}

func parseBybitOrderBook(data []byte) ([]exchange.OrderBookLevel, []exchange.OrderBookLevel, error) {
	var payload struct {
		Result struct {
			Asks [][]string `json:"a"`
			Bids [][]string `json:"b"`
		} `json:"result"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, nil, err
	}
	return parseStringLevels(payload.Result.Bids), parseStringLevels(payload.Result.Asks), nil
}

func parseCoinbaseOrderBook(data []byte) ([]exchange.OrderBookLevel, []exchange.OrderBookLevel, error) {
	var payload struct {
		Asks [][]json.RawMessage `json:"asks"`
		Bids [][]json.RawMessage `json:"bids"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, nil, err
	}
	return parseRawLevels(payload.Bids), parseRawLevels(payload.Asks), nil
}

func parseGateIOOrderBook(data []byte) ([]exchange.OrderBookLevel, []exchange.OrderBookLevel, error) {
	var payload struct {
		Asks [][]string `json:"asks"`
		Bids [][]string `json:"bids"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, nil, err
	}
	return parseStringLevels(payload.Bids), parseStringLevels(payload.Asks), nil
}

func parseBitgetOrderBook(data []byte) ([]exchange.OrderBookLevel, []exchange.OrderBookLevel, error) {
	var payload struct {
		Data struct {
			Asks [][]string `json:"asks"`
			Bids [][]string `json:"bids"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, nil, err
	}
	return parseStringLevels(payload.Data.Bids), parseStringLevels(payload.Data.Asks), nil
}

func parseTickerRows[T any](rows []T, knownAssets []string, fields func(T) (symbol, bid, ask, last string)) []tickerUpdate {
	out := make([]tickerUpdate, 0, len(rows))
	for _, row := range rows {
		symbol, bidRaw, askRaw, lastRaw := fields(row)
		update, ok := buildTickerUpdate(symbol, bidRaw, askRaw, lastRaw, knownAssets)
		if ok {
			out = append(out, update)
		}
	}
	return out
}

func buildTickerUpdate(symbol, bidRaw, askRaw, lastRaw string, knownAssets []string) (tickerUpdate, bool) {
	canonical, ok := normalizeSymbol(symbol, knownAssets)
	if !ok {
		return tickerUpdate{}, false
	}
	bid, bidOK := exchange.DecimalFromString(bidRaw)
	ask, askOK := exchange.DecimalFromString(askRaw)
	last, _ := exchange.DecimalFromString(lastRaw)
	if !bidOK || !askOK || !exchange.ValidBidAsk(bid, ask) {
		return tickerUpdate{}, false
	}
	return tickerUpdate{Symbol: canonical, Bid: bid, Ask: ask, Last: last}, true
}

func normalizeSymbol(symbol string, knownAssets []string) (string, bool) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if strings.Contains(symbol, "/") || strings.Contains(symbol, "-") || strings.Contains(symbol, "_") {
		symbol = strings.ReplaceAll(symbol, "_", "/")
		canonical := exchange.NormalizeCanonicalSymbol(symbol)
		base, quote := splitCanonical(canonical)
		if base == "" || quote == "" {
			return "", false
		}
		if len(knownAssets) > 0 && (!knownAsset(knownAssets, base) || !knownAsset(knownAssets, quote)) {
			return "", false
		}
		return canonical, true
	}
	base, quote, canonical, ok := exchange.SplitJoinedSymbol(symbol, knownAssets)
	if !ok || base == "" || quote == "" {
		return "", false
	}
	return canonical, true
}

func parseStringLevels(levels [][]string) []exchange.OrderBookLevel {
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

func parseRawLevels(levels [][]json.RawMessage) []exchange.OrderBookLevel {
	out := make([]exchange.OrderBookLevel, 0, len(levels))
	for _, row := range levels {
		if len(row) < 2 {
			continue
		}
		price, priceOK := decimalFromRaw(row[0])
		qty, qtyOK := decimalFromRaw(row[1])
		if !priceOK || !qtyOK || price.LessThanOrEqual(decimal.Zero) || qty.LessThanOrEqual(decimal.Zero) {
			continue
		}
		out = append(out, exchange.OrderBookLevel{Price: price, Quantity: qty})
	}
	return out
}

func decimalFromRaw(value json.RawMessage) (decimal.Decimal, bool) {
	var asString string
	if err := json.Unmarshal(value, &asString); err == nil {
		return exchange.DecimalFromString(asString)
	}
	return exchange.DecimalFromString(string(value))
}

func splitCanonical(symbol string) (string, string) {
	parts := strings.Split(exchange.NormalizeCanonicalSymbol(symbol), "/")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func knownAsset(knownAssets []string, target string) bool {
	if len(knownAssets) == 0 {
		return true
	}
	target = exchange.NormalizeAsset(target)
	for _, asset := range knownAssets {
		if exchange.NormalizeAsset(asset) == target {
			return true
		}
	}
	return false
}

func countStaleTickers(tickers map[string]exchange.Ticker, staleAfter time.Duration) int {
	var count int
	for _, ticker := range tickers {
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
