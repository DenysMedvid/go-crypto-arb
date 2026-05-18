# Config Design

Configuration is split between `.env` and YAML runtime config.

## `.env`

`.env` is for secrets and deployment overrides:

```env
API_KEY=change-me
CONFIG_PATH=./configs/config.yaml
HTTP_ADDR=:8080

IBKR_ENABLED=true
IBKR_HOST=127.0.0.1
IBKR_PORT=7497
IBKR_CLIENT_ID=101
```

Secrets should not be committed. `.env.example` documents expected keys.

## `config.yaml`

YAML config controls runtime behavior:

- App refresh and stale thresholds
- Provider enablement
- Market data depth
- Simulation trade sizes
- Assets and quote assets
- Instrument universes
- Strategy configuration
- Alerts
- Health scoring
- Metrics
- TUI titles/icons/tabs

The example lives in `configs/config.example.yaml`.

## Config Types

Key structs live in `internal/config/config.go`:

- `Config`
- `AppConfig`
- `APIConfig`
- `ProviderConfig`
- `MarketDataConfig`
- `SimulationConfig`
- `AssetConfig`
- `InstrumentUniverse`
- `InstrumentConfig`
- `StrategiesConfig`
- `AlertsConfig`
- `MetricsConfig`
- `HealthConfig`
- `TUIConfig`

## API Config

```yaml
api:
  http_addr: ":8080"
  cors_allowed_origins:
    - "https://arb.example.com"
```

Loopback origins such as `http://localhost:<port>` and
`http://127.0.0.1:<port>` are allowed automatically for local web UI
development. Add deployed web UI origins to `cors_allowed_origins`.

## Provider Config

Crypto exchange example:

```yaml
providers:
  binance:
    enabled: true
    type: crypto_exchange
    spot_enabled: true
    futures_enabled: true
    websocket_enabled: true
    rest_poll_interval: 5s
    fees:
      spot_taker: 0.001
      futures_taker: 0.0005
```

Broker example:

```yaml
providers:
  ibkr:
    enabled: true
    type: broker
    market_data_enabled: true
    trading_enabled: false
    crypto_spot_enabled: false
    api_mode: "tws_gateway"
    host: "127.0.0.1"
    port: 7497
    client_id: 101
```

Additional supported crypto exchange providers are present but disabled by default:

```yaml
providers:
  okx:
    enabled: false
    type: crypto_exchange
    spot_enabled: true
    futures_enabled: false
  bybit:
    enabled: false
    type: crypto_exchange
    spot_enabled: true
    futures_enabled: false
  coinbase:
    enabled: false
    type: crypto_exchange
    spot_enabled: true
    futures_enabled: false
  gateio:
    enabled: false
    type: crypto_exchange
    spot_enabled: true
    futures_enabled: false
  bitget:
    enabled: false
    type: crypto_exchange
    spot_enabled: true
    futures_enabled: false
```

OKX, Bybit, Coinbase, Gate.io, and Bitget currently use public spot REST adapters. Futures, funding rates, and WebSocket ingestion are not implemented for those adapters yet.

## Instrument Universes

Instrument universes group configured instruments by use case:

```yaml
instrument_universes:
  ibkr_futures:
    title: "IBKR Futures"
    providers: ["ibkr"]
    instruments:
      - id: CME_MICRO_BTC
        display_name: "CME Micro Bitcoin Future"
        asset_class: futures
        market_type: futures
        ibkr:
          symbol: "MBT"
          sec_type: "FUT"
          exchange: "CME"
          currency: "USD"
          con_id: null
```

## Strategy Config

Strategy config controls enablement, titles, provider filters, thresholds, result limits, and order book depth use.

```yaml
strategies:
  cross_exchange:
    enabled: true
    title: "Cross-Exchange Arbitrage"
    providers: ["binance", "kraken"]
    universe: "crypto_spot"
    min_profit_percent: 0.0
    max_results: 20
    use_order_book_depth: true
```

The code also keeps older `arbitrage` config fields for compatibility.

## TUI Config

```yaml
tui:
  backend_url: "http://localhost:8080"
  refresh_interval: 2s
  default_view: "crypto_dashboard"
  use_emoji: true
  use_ascii_fallback: true
  tabs:
    ibkr:
      enabled: true
      title: "IBKR Monitor"
```

## Custom Titles

Custom titles exist because provider symbols are often not user-friendly:

- TUI tab labels
- Strategy display names
- Instrument display names
- Universe names
- Alert/signal block names

They let users map provider-specific names to operational names without changing code.

## Alerts Config

```yaml
alerts:
  enabled: true
  min_profit_percent: 0.2
  min_basis_percent: 0.2
  cooldown: 5m
  repeat_if_profit_changes_by_percent: 0.1
  max_results: 50
```

## Health Config

```yaml
health:
  scoring_enabled: true
  stale_penalty: 20
  disconnected_penalty: 40
  rest_fallback_penalty: 10
  reconnect_penalty: 2
```

## Metrics Config

```yaml
metrics:
  prometheus_enabled: true
  prometheus_path: /metrics
```

## Validation Rules

Implemented in `internal/config/validation.go`.

Checks include:

- Missing `API_KEY`
- Unsupported enabled exchanges/providers
- Enabled supported crypto exchange providers: OKX, Bybit, Binance, Kraken, Coinbase, Gate.io, and Bitget
- Missing fees
- Missing preferred assets/stable bases
- Missing market availability when discovered markets are provided
- Invalid strategy provider references
- IBKR host/port/client ID shape
- IBKR instruments configured enough to attempt market data
- IBKR trading must remain disabled
- IBKR crypto spot disabled unless explicitly enabled
- Kraken futures partial-support warning
- Public-REST adapter futures partial-support warning for OKX, Bybit, Coinbase, Gate.io, and Bitget

## Config Design Gaps

- Validation does not currently connect to providers for live discovery.
- Some target strategy provider filters are documented/configured but not enforced uniformly in every calculator yet.
- Backward compatibility between `exchanges`/`arbitrage` and `providers`/`strategies` adds complexity.
