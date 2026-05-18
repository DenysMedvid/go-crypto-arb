package provider

import (
	"context"
	"errors"

	"go-crypto-arb/internal/exchange"
)

const (
	TypeCryptoExchange = "crypto_exchange"
	TypeBroker         = "broker"
)

var ErrNotImplemented = errors.New("not implemented")

type MarketDataProvider interface {
	Name() string
	Type() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health() exchange.ExchangeHealth
	GetLatestTickers() []exchange.Ticker
	GetLatestOrderBooks() []exchange.OrderBook
	DiscoverMarkets(ctx context.Context) ([]exchange.MarketInfo, error)
}

type CryptoExchangeProvider interface {
	MarketDataProvider
	GetFundingRates() []exchange.FundingRate
}

type BrokerProvider interface {
	MarketDataProvider
	GetAccountSummary(ctx context.Context) (AccountSummary, error)
	GetPortfolio(ctx context.Context) (PortfolioSnapshot, error)
}

type AccountSummary struct {
	Provider  string `json:"provider"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Supported bool   `json:"supported"`
}

type PortfolioSnapshot struct {
	Provider  string              `json:"provider"`
	Positions []PortfolioPosition `json:"positions"`
	Status    string              `json:"status"`
	Message   string              `json:"message,omitempty"`
	Supported bool                `json:"supported"`
}

type PortfolioPosition struct {
	InstrumentID string `json:"instrument_id"`
	Symbol       string `json:"symbol"`
	Quantity     string `json:"quantity"`
}
