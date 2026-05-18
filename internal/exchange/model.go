package exchange

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type MarketType string

const (
	MarketSpot    MarketType = "spot"
	MarketFutures MarketType = "futures"
)

type Exchange interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health() ExchangeHealth
	GetLatestTickers() []Ticker
	GetLatestFuturesTickers() []Ticker
	GetFundingRates() []FundingRate
	GetLatestOrderBooks() []OrderBook
	GetMarkets() []MarketInfo
}

type Ticker struct {
	Provider     string          `json:"provider,omitempty"`
	Exchange     string          `json:"exchange"`
	Broker       string          `json:"broker,omitempty"`
	InstrumentID string          `json:"instrument_id,omitempty"`
	DisplayName  string          `json:"display_name,omitempty"`
	Symbol       string          `json:"symbol"`
	BaseAsset    string          `json:"base_asset"`
	QuoteAsset   string          `json:"quote_asset"`
	MarketType   MarketType      `json:"market_type"`
	AssetClass   string          `json:"asset_class"`
	Bid          decimal.Decimal `json:"bid"`
	Ask          decimal.Decimal `json:"ask"`
	Last         decimal.Decimal `json:"last"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type FundingRate struct {
	Exchange        string          `json:"exchange"`
	Symbol          string          `json:"symbol"`
	Rate            decimal.Decimal `json:"rate"`
	NextFundingTime time.Time       `json:"next_funding_time"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type OrderBookLevel struct {
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity"`
}

type OrderBook struct {
	Provider     string           `json:"provider"`
	Exchange     string           `json:"exchange"`
	Broker       string           `json:"broker,omitempty"`
	Symbol       string           `json:"symbol"`
	InstrumentID string           `json:"instrument_id,omitempty"`
	BaseAsset    string           `json:"base_asset"`
	QuoteAsset   string           `json:"quote_asset"`
	MarketType   MarketType       `json:"market_type"`
	AssetClass   string           `json:"asset_class"`
	Bids         []OrderBookLevel `json:"bids"`
	Asks         []OrderBookLevel `json:"asks"`
	UpdatedAt    time.Time        `json:"updated_at"`
	LimitedDepth bool             `json:"limited_depth,omitempty"`
}

type MarketInfo struct {
	Provider     string     `json:"provider,omitempty"`
	Exchange     string     `json:"exchange"`
	Broker       string     `json:"broker,omitempty"`
	Symbol       string     `json:"symbol"`
	InstrumentID string     `json:"instrument_id,omitempty"`
	DisplayName  string     `json:"display_name,omitempty"`
	BaseAsset    string     `json:"base_asset"`
	QuoteAsset   string     `json:"quote_asset"`
	AssetClass   string     `json:"asset_class"`
	MarketType   MarketType `json:"market_type"`
	Active       bool       `json:"active"`
}

type ExchangeHealth struct {
	Provider            string    `json:"provider,omitempty"`
	ProviderType        string    `json:"provider_type,omitempty"`
	Exchange            string    `json:"exchange"`
	Broker              string    `json:"broker,omitempty"`
	Enabled             bool      `json:"enabled"`
	SpotEnabled         bool      `json:"spot_enabled"`
	FuturesEnabled      bool      `json:"futures_enabled"`
	MarketDataEnabled   bool      `json:"market_data_enabled,omitempty"`
	TradingEnabled      bool      `json:"trading_enabled"`
	WebSocketEnabled    bool      `json:"websocket_enabled"`
	WebSocketConnected  bool      `json:"websocket_connected"`
	GatewayConnected    bool      `json:"gateway_connected,omitempty"`
	MarketDataOK        bool      `json:"market_data_ok,omitempty"`
	LastMessageTime     time.Time `json:"last_message_time"`
	LastMessageAt       time.Time `json:"last_message_at"`
	RestFallbackActive  bool      `json:"rest_fallback_active"`
	ReconnectCount      int       `json:"reconnect_count"`
	LastError           string    `json:"last_error,omitempty"`
	DataFresh           bool      `json:"data_fresh"`
	StaleTickerCount    int       `json:"stale_ticker_count"`
	StaleOrderBookCount int       `json:"stale_order_book_count"`
	PartialSupport      bool      `json:"partial_support"`
	Score               int       `json:"score"`
	Status              string    `json:"status"`
}
