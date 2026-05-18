# API Design

The backend exposes current state through a REST API implemented in `internal/api/server.go`.

## Authentication

Protected API endpoints require:

```text
X-API-Key: <API_KEY>
```

`API_KEY` is loaded from `.env` through `config.LoadEnv`.

Public endpoints:

- `GET /health`
- `GET /metrics` when enabled

Protected endpoints:

- All `/api/v1/*` routes

If the API key is missing or invalid, middleware in `internal/api/middleware.go` returns `401 Unauthorized`.

## CORS

The API includes CORS handling for browser clients. Loopback origins such as
`http://localhost:<port>` and `http://127.0.0.1:<port>` are allowed by default
for local web UI development, including Vite ports such as `5173` or `5174`.

For deployed web UIs, configure explicit origins:

```yaml
api:
  cors_allowed_origins:
    - "https://arb.example.com"
```

Preflight `OPTIONS` requests are answered before API-key middleware. Protected
`/api/v1/*` requests still require `X-API-Key`.

## Endpoint Status

| Endpoint | Status | Notes |
| --- | --- | --- |
| `GET /health` | Implemented | Public status/version/time |
| `GET /api/v1/prices` | Implemented | Spot, futures, funding rates |
| `GET /api/v1/order-books` | Implemented | Summary by default; filters supported |
| `GET /api/v1/arbitrage/triangular` | Implemented | Crypto triangular v2 results |
| `GET /api/v1/arbitrage/cross-exchange` | Implemented | Cross-exchange v2 results |
| `GET /api/v1/arbitrage/spot-futures` | Implemented | Crypto spot-futures v2 results |
| `GET /api/v1/signals/related-assets` | Implemented | Related asset signals |
| `GET /api/v1/alerts` | Implemented | Latest in-memory alerts |
| `GET /api/v1/exchanges/health` | Implemented | Exchange/provider health map, legacy name |
| `GET /api/v1/providers` | Implemented | Configured providers |
| `GET /api/v1/providers/health` | Implemented | Provider health map |
| `GET /api/v1/ibkr/instruments` | Implemented / partial | Configured IBKR instruments; live data partial |
| `GET /api/v1/ibkr/fx-triangular` | Implemented / partial | Strategy endpoint; depends on IBKR market data availability |
| `GET /api/v1/ibkr/crypto-futures-basis` | Implemented / partial | Strategy endpoint; depends on IBKR market data availability |
| `GET /api/v1/snapshot` | Implemented | Aggregated snapshot consumed by TUI |
| `GET /metrics` | Implemented | Public Prometheus text when enabled |

## Error Response Format

Errors use simple JSON:

```json
{
  "error": "message"
}
```

Authentication failure is handled by middleware.

## Decimal Serialization

Financial values use `shopspring/decimal.Decimal`. JSON endpoints serialize decimals through decimal's JSON behavior.

## Endpoint Details

### `GET /health`

Public.

Returns:

- `status`
- `version`
- `time`

### `GET /api/v1/prices`

Protected.

Returns:

- `prices`
- `futures_prices`
- `funding_rates`

### `GET /api/v1/order-books`

Protected.

By default returns order book summaries to avoid large payloads.

Query parameters:

```text
provider=binance
exchange=Binance
symbol=BTC/USDT
market=spot
```

Supported crypto provider identities are `okx`, `bybit`, `binance`, `kraken`, `coinbase`, `gateio`, and `bitget`. The additional public-REST platforms are only present in responses when enabled and returning market data.

When filters are supplied, full matching `exchange.OrderBook` values are returned.

### Arbitrage Endpoints

Protected:

- `/api/v1/arbitrage/triangular`
- `/api/v1/arbitrage/cross-exchange`
- `/api/v1/arbitrage/spot-futures`

These expose order-book-aware v2 strategy results from `internal/arbitrage`.

### IBKR Endpoints

Protected:

- `/api/v1/ibkr/instruments`
- `/api/v1/ibkr/fx-triangular`
- `/api/v1/ibkr/crypto-futures-basis`

IBKR live market-data transport is partial. Instrument metadata is config-derived. Strategy endpoints return calculated results only when relevant broker tickers/order books are available.

### `GET /api/v1/alerts`

Protected.

Returns current alerts from `alerts.Engine`, including:

- `id`
- `dedup_key`
- `severity`
- `type`
- `message`
- `value`
- `threshold`
- `repeat_count`
- timestamps

### Health Endpoints

Protected:

- `/api/v1/exchanges/health`
- `/api/v1/providers/health`

Both currently expose health maps from the same store data. `/api/v1/exchanges/health` is retained for compatibility.

### `GET /api/v1/snapshot`

Protected.

Returns `marketdata.Snapshot`, including:

- prices
- futures prices
- order books
- order book summary
- funding rates
- markets
- crypto strategy outputs
- related asset signals
- IBKR instruments
- IBKR strategy outputs
- alerts
- exchange/provider health

### `GET /metrics`

Public when `metrics.prometheus_enabled=true`.

Rendered manually as Prometheus text in `internal/api/server.go`. The project does not currently use the Prometheus Go client library.

Metric families mirror the latest snapshot consumed by the TUI:

- `go_crypto_arb_price_bid`
- `go_crypto_arb_price_ask`
- `go_crypto_arb_price_last`
- `go_crypto_arb_price_spread`
- `go_crypto_arb_price_age_seconds`
- `go_crypto_arb_funding_rate`
- `go_crypto_arb_funding_age_seconds`
- `go_crypto_arb_funding_next_time_seconds`
- `go_crypto_arb_order_book_best_bid`
- `go_crypto_arb_order_book_best_ask`
- `go_crypto_arb_order_book_bid_levels`
- `go_crypto_arb_order_book_ask_levels`
- `go_crypto_arb_order_book_limited_depth`
- `go_crypto_arb_order_book_age_seconds`
- `go_crypto_arb_market_active`
- `go_crypto_arb_arbitrage_profit_percent`
- `go_crypto_arb_arbitrage_trade_size`
- `go_crypto_arb_arbitrage_start_amount`
- `go_crypto_arb_arbitrage_end_amount`
- `go_crypto_arb_arbitrage_basis_percent`
- `go_crypto_arb_arbitrage_net_estimate_percent`
- `go_crypto_arb_arbitrage_complete_fill`
- `go_crypto_arb_arbitrage_age_seconds`
- `go_crypto_arb_arbitrage_leg_*`
- `go_crypto_arb_related_asset_*`
- `go_crypto_arb_alert_*`
- `go_crypto_arb_provider_connected`
- `go_crypto_arb_provider_*`
- `go_crypto_arb_ws_reconnect_total`
- `go_crypto_arb_stale_price_total`
- `go_crypto_arb_stale_order_book_total`
- `go_crypto_arb_health_score`

## API Design Gaps

- No OpenAPI spec is currently generated.
- No pagination for large snapshots.
- No streaming snapshot endpoint is wired as the main TUI path.
- `/api/v1/exchanges/health` and `/api/v1/providers/health` overlap.
- Metrics are rendered in API code rather than a dedicated metrics module.
