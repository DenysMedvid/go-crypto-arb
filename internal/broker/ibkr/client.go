package ibkr

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
	"go-crypto-arb/internal/instrument"
	"go-crypto-arb/internal/provider"
)

const (
	Name = "IBKR"
	name = "IBKR"
)

type Client struct {
	cfg         config.ProviderConfig
	instruments []config.InstrumentConfig
	logger      *slog.Logger

	mu      sync.RWMutex
	markets map[string]exchange.MarketInfo
	tickers map[string]exchange.Ticker
	books   map[string]exchange.OrderBook
	health  exchange.ExchangeHealth
}

func New(cfg config.ProviderConfig, instruments []config.InstrumentConfig, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	client := &Client{
		cfg:         cfg,
		instruments: append([]config.InstrumentConfig(nil), instruments...),
		logger:      logger.With("provider", name),
		markets:     make(map[string]exchange.MarketInfo),
		tickers:     make(map[string]exchange.Ticker),
		books:       make(map[string]exchange.OrderBook),
		health: exchange.ExchangeHealth{
			Provider:          strings.ToLower(name),
			ProviderType:      provider.TypeBroker,
			Broker:            name,
			Exchange:          name,
			Enabled:           cfg.Enabled,
			FuturesEnabled:    cfg.FuturesEnabled,
			MarketDataEnabled: cfg.MarketDataEnabled,
			TradingEnabled:    cfg.TradingEnabled,
			Status:            "starting",
		},
	}
	client.loadConfiguredMarkets()
	return client
}

func (c *Client) Name() string { return name }

func (c *Client) Type() string { return provider.TypeBroker }

func (c *Client) Start(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.cfg.Enabled {
		c.health.Status = "disabled"
		return nil
	}
	c.health.LastMessageTime = time.Now()
	c.health.LastMessageAt = c.health.LastMessageTime
	c.health.GatewayConnected = false
	c.health.MarketDataOK = false
	c.health.WebSocketConnected = false
	c.health.RestFallbackActive = false
	c.health.DataFresh = false
	c.health.Status = "disconnected"
	c.health.LastError = "IBKR TWS Gateway market-data transport is not implemented in v2.1; configured instruments are exposed for validation only"
	if c.cfg.TradingEnabled {
		c.health.LastError = "IBKR trading_enabled=true is unsupported in v2.1; no trading code path is available"
	}
	return nil
}

func (c *Client) Stop(context.Context) error { return nil }

func (c *Client) Health() exchange.ExchangeHealth {
	c.mu.RLock()
	defer c.mu.RUnlock()
	h := c.health
	h.Provider = strings.ToLower(name)
	h.ProviderType = provider.TypeBroker
	h.Broker = name
	h.Exchange = name
	h.MarketDataEnabled = c.cfg.MarketDataEnabled
	h.TradingEnabled = c.cfg.TradingEnabled
	h.LastMessageAt = h.LastMessageTime
	if !c.cfg.Enabled {
		h.Status = "disabled"
		return h
	}
	if !h.GatewayConnected {
		h.Status = "disconnected"
		h.DataFresh = false
	}
	return h
}

func (c *Client) GetLatestTickers() []exchange.Ticker {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]exchange.Ticker, 0, len(c.tickers))
	for _, ticker := range c.tickers {
		out = append(out, ticker)
	}
	return out
}

func (c *Client) GetLatestOrderBooks() []exchange.OrderBook {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]exchange.OrderBook, 0, len(c.books))
	for _, book := range c.books {
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

func (c *Client) GetAccountSummary(context.Context) (provider.AccountSummary, error) {
	return provider.AccountSummary{
		Provider:  strings.ToLower(name),
		Status:    "not_implemented",
		Message:   "IBKR account summary is intentionally not implemented in v2.1",
		Supported: false,
	}, provider.ErrNotImplemented
}

func (c *Client) GetPortfolio(context.Context) (provider.PortfolioSnapshot, error) {
	return provider.PortfolioSnapshot{
		Provider:  strings.ToLower(name),
		Status:    "not_implemented",
		Message:   "IBKR portfolio snapshot is intentionally not implemented in v2.1",
		Supported: false,
	}, provider.ErrNotImplemented
}

func (c *Client) ConfiguredInstruments() []config.InstrumentConfig {
	return append([]config.InstrumentConfig(nil), c.instruments...)
}

func (c *Client) loadConfiguredMarkets() {
	for _, item := range c.instruments {
		if isCryptoSpot(item) && !c.cfg.CryptoSpotEnabled {
			continue
		}
		symbol := instrument.DisplaySymbol(item)
		base, quote := inferBaseQuote(item)
		marketType := instrument.MarketType(item.MarketType)
		assetClass := strings.ToLower(strings.TrimSpace(item.AssetClass))
		if assetClass == "" {
			assetClass = inferAssetClass(item)
		}
		market := exchange.MarketInfo{
			Provider:     strings.ToLower(name),
			Broker:       name,
			Exchange:     name,
			Symbol:       symbol,
			InstrumentID: item.ID,
			DisplayName:  displayName(item),
			BaseAsset:    base,
			QuoteAsset:   quote,
			AssetClass:   assetClass,
			MarketType:   marketType,
			Active:       true,
		}
		c.markets[marketKey(market)] = market
	}
}

func marketKey(m exchange.MarketInfo) string {
	return string(m.MarketType) + "|" + m.InstrumentID + "|" + m.Symbol
}

func displayName(item config.InstrumentConfig) string {
	if item.DisplayName != "" {
		return item.DisplayName
	}
	if item.ID != "" {
		return item.ID
	}
	return instrument.DisplaySymbol(item)
}

func inferBaseQuote(item config.InstrumentConfig) (string, string) {
	symbol := instrument.DisplaySymbol(item)
	if strings.Contains(symbol, "/") {
		parts := strings.Split(exchange.NormalizeCanonicalSymbol(symbol), "/")
		if len(parts) == 2 {
			return exchange.NormalizeAsset(parts[0]), exchange.NormalizeAsset(parts[1])
		}
	}
	currency := exchange.NormalizeAsset(item.IBKR.Currency)
	if currency == "" {
		currency = "USD"
	}
	if item.ID != "" {
		return exchange.NormalizeAsset(item.ID), currency
	}
	return exchange.NormalizeAsset(symbol), currency
}

func inferAssetClass(item config.InstrumentConfig) string {
	secType := strings.ToUpper(strings.TrimSpace(item.IBKR.SecType))
	switch secType {
	case "FUT":
		return instrument.AssetClassFutures
	case "STK":
		return instrument.AssetClassStock
	case "CASH":
		return instrument.AssetClassFX
	default:
		return strings.ToLower(secType)
	}
}

func isCryptoSpot(item config.InstrumentConfig) bool {
	return strings.EqualFold(item.AssetClass, instrument.AssetClassCrypto) && strings.EqualFold(item.MarketType, string(exchange.MarketSpot))
}

func ValidateInstrument(item config.InstrumentConfig) error {
	if item.ID == "" {
		return fmt.Errorf("instrument id is required")
	}
	if item.Symbol == "" && item.IBKR.Symbol == "" {
		return fmt.Errorf("%s symbol is required", item.ID)
	}
	if item.IBKR.Symbol != "" {
		if item.IBKR.SecType == "" {
			return fmt.Errorf("%s ibkr.sec_type is required", item.ID)
		}
		if item.IBKR.Exchange == "" {
			return fmt.Errorf("%s ibkr.exchange is required", item.ID)
		}
		if item.IBKR.Currency == "" {
			return fmt.Errorf("%s ibkr.currency is required", item.ID)
		}
	}
	return nil
}
