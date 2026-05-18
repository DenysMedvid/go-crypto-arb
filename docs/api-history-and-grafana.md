# API History and Grafana

`go-crypto-arb` keeps only the latest market state in memory. The TUI polls the backend and renders that current snapshot; it does not keep historical candles, charts, or a local database.

To keep history and render it in Grafana, run the API service as the producer and let Prometheus scrape the metrics endpoint:

```text
Providers -> Backend API -> /api/v1/snapshot -> TUI current view
                       \-> /metrics -> Prometheus -> Grafana history
```

## Runtime Requirements

Set an API key in `.env`:

```env
API_KEY=change-me
CONFIG_PATH=./configs/config.yaml
HTTP_ADDR=:8080
```

Enable Prometheus metrics in `configs/config.yaml`:

```yaml
metrics:
  prometheus_enabled: true
  prometheus_path: /metrics
```

Run the backend:

```bash
go run ./cmd/api
```

Run the TUI in another terminal:

```bash
go run ./cmd/tui
```

The TUI uses the same `API_KEY` from `.env` and calls:

```http
GET /api/v1/snapshot
X-API-Key: change-me
```

## API Endpoints

Useful endpoints for dashboards and current-state inspection:

| Endpoint | Auth | Use |
| --- | --- | --- |
| `GET /health` | Public | Backend liveness, version, current UTC time. |
| `GET /api/v1/snapshot` | `X-API-Key` | Full latest-state snapshot used by UI clients. |
| `GET /api/v1/prices` | `X-API-Key` | Spot, futures, and funding rate data. |
| `GET /api/v1/order-books` | `X-API-Key` | Compact order book summary by default, full books with filters. |
| `GET /api/v1/arbitrage/cross-exchange` | `X-API-Key` | Latest cross-exchange estimates. |
| `GET /api/v1/arbitrage/triangular` | `X-API-Key` | Latest triangular estimates. |
| `GET /api/v1/arbitrage/spot-futures` | `X-API-Key` | Latest spot-futures estimates. |
| `GET /api/v1/alerts` | `X-API-Key` | Current in-memory alerts. |
| `GET /api/v1/providers/health` | `X-API-Key` | Provider health map. |
| `GET /metrics` | Public | Prometheus text exposition when enabled. |

Examples:

```bash
curl http://localhost:8080/health
curl -H "X-API-Key: change-me" http://localhost:8080/api/v1/snapshot
curl http://localhost:8080/metrics
```

## Prometheus Path

Prometheus is the supported history path. The backend exposes a public text exposition endpoint when `metrics.prometheus_enabled=true`.

Example scrape config:

```yaml
global:
  scrape_interval: 2s

scrape_configs:
  - job_name: go-crypto-arb
    metrics_path: /metrics
    static_configs:
      - targets:
          - localhost:8080
```

For a remote deployment, keep `/metrics` behind a private network, firewall, or reverse proxy if the data should not be public.

Metric families:

| Metric | Labels | Meaning |
| --- | --- | --- |
| `go_crypto_arb_price_bid` | `provider`, `symbol`, `market` | Latest bid. |
| `go_crypto_arb_price_ask` | `provider`, `symbol`, `market` | Latest ask. |
| `go_crypto_arb_price_last` / `go_crypto_arb_price_spread` / `go_crypto_arb_price_age_seconds` | `provider`, `symbol`, `market` | Ticker last, spread, and data age. |
| `go_crypto_arb_funding_*` | `exchange`, `symbol` | Funding rate, funding data age, and next funding timestamp. |
| `go_crypto_arb_order_book_*` | `provider`, `exchange`, `symbol`, `market` | Best bid/ask, depth levels, limited-depth flag, and book age. |
| `go_crypto_arb_market_active` | `provider`, `exchange`, `broker`, `symbol`, `instrument_id`, `asset_class`, `market` | Configured market or IBKR instrument active flag. |
| `go_crypto_arb_arbitrage_*` | strategy labels | Profit, sizing, prices, fees, slippage, basis, fill, age, and leg-level values. |
| `go_crypto_arb_related_asset_*` | `group`, `asset`, `symbol`, `exchange` | Related asset group average, change, and divergence. |
| `go_crypto_arb_alert_*` | `type`, `severity`, `status`, `exchange`, `symbol` | Active alert marker, value, threshold, repeat count, and age. |
| `go_crypto_arb_provider_connected` | `provider` | `1` when connected or fallback is active, otherwise `0`. |
| `go_crypto_arb_provider_*` | `provider`, `exchange`, `status` | Provider capability and connectivity flags. |
| `go_crypto_arb_ws_reconnect_total` | `provider` | WebSocket reconnect count. |
| `go_crypto_arb_stale_price_total` | `provider` | Stale ticker count. |
| `go_crypto_arb_stale_order_book_total` | `provider`, `exchange`, `status` | Stale order book count. |
| `go_crypto_arb_health_score` | `provider` | Provider health score from `0` to `100`. |

Useful Grafana PromQL examples:

```promql
go_crypto_arb_price_bid{provider="binance", symbol="BTC/USDT", market="spot"}
go_crypto_arb_price_ask{provider="binance", symbol="BTC/USDT", market="spot"}
go_crypto_arb_arbitrage_profit_percent{type="cross_exchange"}
go_crypto_arb_health_score
go_crypto_arb_provider_connected
go_crypto_arb_order_book_age_seconds
```

Provider labels may include any enabled supported crypto platform: `okx`, `bybit`, `binance`, `kraken`, `coinbase`, `gateio`, or `bitget`.

For longer history, configure Prometheus retention explicitly:

```bash
prometheus --config.file=prometheus.yml --storage.tsdb.retention.time=30d
```

## Dashboard Panels

Good first Grafana panels:

- Time series: bid and ask by `provider`, `symbol`, and `market`.
- Time series: `go_crypto_arb_arbitrage_profit_percent` by strategy type.
- Stat: provider connected state.
- Gauge: provider health score.
- Table: current alerts by severity.
- Time series: order book age to spot stale data.

## TUI History Notes

The TUI is intentionally a live display client. If you want historical charts, Grafana should read from Prometheus. If a future TUI history view is added, keep the same boundary: the TUI should call the backend or a time-series API, not exchanges, IBKR, or other providers directly.

## Operational Notes

- Keep `marketdata.Store` latest-state only; do not block provider ingestion on history collection.
- Use `/metrics` for Prometheus scraping.
- Keep label cardinality controlled. `provider`, `market`, `symbol`, `type`, and `severity` are useful. Avoid putting alert messages into labels.
- The API service is read-only and monitoring-only. Historical storage does not add order execution.
