# Provider Design

Providers are external market-data sources normalized into internal market models.

## Target Interfaces

`internal/provider/provider.go` defines:

```go
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
```

`CryptoExchangeProvider` adds:

- `GetFundingRates`

`BrokerProvider` adds:

- `GetAccountSummary`
- `GetPortfolio`

Current implementation note: crypto exchange adapters primarily implement `exchange.Exchange`; the provider interfaces are the broader target abstraction.

## Normalized Models

Shared models live in `internal/exchange/model.go`:

- `Ticker`
- `OrderBook`
- `OrderBookLevel`
- `FundingRate`
- `MarketInfo`
- `ExchangeHealth`

These models include provider-aware fields such as `Provider`, `Broker`, `InstrumentID`, and `AssetClass`.

## Binance Adapter

Package: `internal/exchange/binance`

Capabilities:

- Spot ticker polling
- Futures ticker polling
- Order book depth polling
- Funding rate polling
- WebSocket book ticker updates
- Health reporting
- Market metadata from observed symbols

Transport:

- REST: Binance spot and futures APIs
- WebSocket: Binance stream endpoints

## Kraken Adapter

Package: `internal/exchange/kraken`

Capabilities:

- Spot asset pair discovery
- Spot ticker polling
- Spot order book depth polling
- Spot WebSocket ticker updates
- Partial futures ticker/funding support
- Health reporting

Notes:

- Kraken futures support is partial.
- Futures order books may fall back to top-of-book limited-depth data.

## Public REST Spot Adapters

Package: `internal/exchange/publicrest`

Supported platforms:

- OKX
- Bybit
- Coinbase
- Gate.io
- Bitget

Capabilities:

- Spot ticker polling
- Spot order book depth polling
- Market metadata from observed symbols
- Health reporting with REST fallback active

Notes:

- These adapters are disabled by default in `configs/config.yaml` and `configs/config.example.yaml`.
- Futures, funding rates, and WebSocket ingestion are not implemented for these adapters yet.
- If futures are enabled for one of these platforms, validation emits a warning and provider health marks partial support.

## IBKR Adapter

Package: `internal/broker/ibkr`

Capabilities:

- Loads configured instruments.
- Exposes `MarketInfo` for configured IBKR instruments.
- Reports broker health/status.
- Account and portfolio methods return explicit `not_implemented`.

Partial/planned:

- Live TWS Gateway market-data transport.
- Contract lookup / conId resolution.
- Live bid/ask/order book updates.

## Symbol Normalization

Package: `internal/exchange`

Important functions:

- `CanonicalSymbol`
- `NormalizeAsset`
- `NormalizeCanonicalSymbol`
- `SplitJoinedSymbol`
- `JoinedSymbol`

Examples:

- `BTCUSDT` -> `BTC/USDT`
- `XBT` -> `BTC`
- `BTC-USDT` -> `BTC/USDT`

## Contract Normalization

IBKR instruments are configured as contract-like instruments:

```yaml
ibkr:
  symbol: "MBT"
  sec_type: "FUT"
  exchange: "CME"
  currency: "USD"
  con_id: null
```

The adapter maps these into `exchange.MarketInfo`. The optional `con_id` is reserved for future resolved contract identifiers.

## Market Discovery

Current:

- Binance, Kraken, and public-REST spot adapters expose observed markets through `GetMarkets`.
- Kraken spot asset pair discovery is used internally.
- IBKR exposes configured instruments as markets.

Partial:

- `DiscoverMarkets` methods exist for target provider compatibility.
- `validate-config` does not perform live provider discovery.

## Health Reporting

Provider health includes:

- WebSocket connected
- REST fallback active
- Last message time
- Reconnect count
- Last error
- Stale ticker/order book counts
- Partial support
- Score/status after `internal/health.Score`
- IBKR gateway/market-data status fields

## REST Fallback and WebSocket Handling

Binance/Kraken can use WebSockets for timely book ticker updates and REST polling as fallback/reference. Public-REST spot adapters use REST polling only, so `RestFallbackActive` is expected for OKX, Bybit, Coinbase, Gate.io, and Bitget.

## Why IBKR Is Separate

IBKR is separate because:

- It is a broker, not a crypto exchange.
- It uses contract metadata and gateway sessions.
- It supports FX/futures/stocks/ETFs.
- Crypto spot through IBKR is disabled by default.
- Trading is unsupported.

## Provider Design Gaps

- No full provider manager package.
- Crypto adapters do not yet fully use `internal/provider` interfaces as their primary interface.
- IBKR live market data is partial/planned.
- Contract resolution and persistent conId storage are not implemented.
