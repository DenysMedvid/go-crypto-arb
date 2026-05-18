# Alerting and Health

Alerting and health turn raw market/provider state into operational signals.

## Alert Engine

Package:

- `internal/alerts`

Main types:

- `Engine`
- `Alert`
- `AlertSeverity`
- `AlertType`
- `Notifier`

Main function:

- `Engine.Evaluate`

## Alert Fields

`Alert` contains:

- `ID`
- `DedupKey`
- `Severity`
- `Type`
- `Message`
- `Exchange`
- `Symbol`
- `Value`
- `Threshold`
- `CreatedAt`
- `UpdatedAt`
- `RepeatCount`
- `Status`

## Deduplication

Each alert gets a stable deduplication key based on alert type and opportunity/provider identity.

Examples:

- Triangular: type + exchange + cycle + trade size
- Cross-exchange: type + symbol + buy provider + sell provider + trade size
- Spot-futures: type + exchange + symbol + trade size
- IBKR FX: type + provider + cycle + trade size
- Broker basis: type + asset + spot provider + futures instrument
- Health: type + provider/exchange

The same event does not create a fresh alert every refresh.

## Cooldown

Configured in:

```yaml
alerts:
  cooldown: 5m
```

If an alert is repeated during cooldown, it is suppressed unless the configured change threshold is met.

## Repeat Threshold

Configured in:

```yaml
alerts:
  repeat_if_profit_changes_by_percent: 0.1
```

If a repeated opportunity changes enough, the alert is updated and `RepeatCount` increments.

## Severity Levels

```go
const (
    AlertInfo     AlertSeverity = "info"
    AlertWarning  AlertSeverity = "warning"
    AlertCritical AlertSeverity = "critical"
)
```

Current behavior:

- Opportunity alerts are usually `info` or `warning` depending on threshold multiple.
- Provider disconnected and low health score are `critical`.
- Stale market data is `warning`.

## Alert Types

Implemented alert types include:

- `triangular_arbitrage`
- `cross_exchange_arbitrage`
- `spot_futures_basis`
- `ibkr_fx_triangular_arbitrage`
- `crypto_spot_vs_ibkr_futures_basis`
- `exchange_data_stale`
- `exchange_disconnected`
- `health_score_low`

## Notifier Extension Point

`Notifier` is defined but no notifiers are implemented:

```go
type Notifier interface {
    Notify(ctx context.Context, alert Alert) error
}
```

Telegram/email/webhook are planned extension points.

## Health Scoring

Package:

- `internal/health`

Main functions:

- `Fresh`
- `Score`
- `ClampScore`

Health inputs come from `exchange.ExchangeHealth`.

## Health Score Pseudocode

```text
score = 100

if broker and enabled and gateway disconnected:
    score -= disconnected_penalty

if websocket enabled and websocket disconnected and no rest fallback:
    score -= disconnected_penalty

if rest fallback active:
    score -= rest_fallback_penalty

if data not fresh or stale ticker/order book count > 0:
    score -= stale_penalty

if last_error present:
    score -= stale_penalty / 2

score -= reconnect_count * reconnect_penalty

if partial support:
    score -= rest_fallback_penalty

score = clamp(score, 0, 100)
```

## Health Status

Status values:

- `ok`
- `degraded`
- `stale`
- `disconnected`

Rules:

- Score >= 90 with fresh data -> `ok`
- Broker gateway disconnected -> `disconnected`
- WebSocket disconnected with no fallback -> `disconnected`
- Not fresh -> `stale`
- Otherwise -> `degraded`

## Stale Detection

Provider adapters count:

- stale tickers
- stale order books

Freshness uses configured durations:

- `market_data.ticker_stale_after`
- `market_data.order_book_stale_after`
- `app.stale_price_after`

## WebSocket Disconnects and REST Fallback

Binance/Kraken track WebSocket connection count. If WebSocket is disconnected but REST polling is still active, health may indicate REST fallback rather than full outage. OKX, Bybit, Coinbase, Gate.io, and Bitget are REST-only in this version, so REST fallback is expected when they are enabled.

## IBKR Gateway Health

IBKR health includes:

- `GatewayConnected`
- `MarketDataOK`
- `MarketDataEnabled`
- `TradingEnabled`
- `LastError`

Current skeleton marks gateway disconnected until live transport exists.

## Alerting / Health Gaps

- Alert state is in memory only.
- No external notifiers are implemented.
- No alert acknowledgement/silencing model.
- Health scoring is deterministic but coarse.
- REST fallback is status reporting, not automatic circuit breaking.
