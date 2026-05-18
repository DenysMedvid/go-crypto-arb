# Design

This document explains the main design choices in `go-crypto-arb`.

## Why Go

Go fits this project because it has:

- Simple concurrency primitives for polling and WebSocket workers.
- A strong standard HTTP stack for REST APIs.
- Static binaries that are easy to deploy through Docker or systemd.
- Clear package boundaries for adapters, calculators, and UI clients.
- Good support for table-driven tests around financial calculation logic.

The code uses explicit dependencies rather than a heavy framework. That keeps the backend easy to reason about and keeps UI clients independent.

## Provider Interfaces

Provider interfaces in `internal/provider/provider.go` describe the target architecture:

- `MarketDataProvider`
- `CryptoExchangeProvider`
- `BrokerProvider`

The crypto adapters currently implement `exchange.Exchange` from `internal/exchange/model.go`. Binance and Kraken have dedicated packages; OKX, Bybit, Coinbase, Gate.io, and Bitget share the `internal/exchange/publicrest` spot adapter. IBKR is separate under `internal/broker/ibkr`. This reflects a transition from exchange-only architecture to provider-based architecture.

Interfaces are useful here because each provider has different transport, symbol format, health signals, and supported market types, while the rest of the system wants normalized tickers, order books, markets, and health.

## `.env` vs `config.yaml`

`.env` contains deployment-specific secrets and overrides:

- `API_KEY`
- `CONFIG_PATH`
- `HTTP_ADDR`
- IBKR connection overrides

`config.yaml` contains runtime behavior:

- Provider enablement
- Fees
- Instrument universes
- Strategy settings
- TUI titles and icons
- Web UI CORS origins
- Alerts
- Health scoring
- Metrics

This split lets operators keep secrets out of checked-in config examples while keeping strategy behavior auditable and versioned.

## Decimal Arithmetic

Financial calculations must not use binary floating point. The project uses `github.com/shopspring/decimal` directly and through `config.Decimal`.

Decimal arithmetic is used for:

- Bid/ask/last prices
- Order book levels
- Trade sizes
- Slippage
- Fees
- Profit and basis percentages
- Alert thresholds

## Why Best Bid/Ask Is Not Enough

Best bid/ask only describes the top visible level. It does not answer whether the configured trade size can actually fill at or near that price.

Example:

- Best ask: `100`
- Quantity at best ask: `0.01`
- Requested quote size: `5,000`

The estimate must consume deeper ask levels. Without depth simulation, the system can overstate profit and understate slippage.

## Order Book Depth Simulation

`internal/arbitrage/simulation.go` implements:

- `SimulateBuyWithQuote`
- `SimulateSellBase`

Rules:

- Quote-to-base conversion consumes asks.
- Base-to-quote conversion consumes bids.
- Fees are applied to the simulated fill.
- Partial fills set `CompleteFill=false`.
- Limited depth is flagged when the book only represents top-of-book or partial depth.

## IBKR as Broker Provider

IBKR is not modeled as a crypto exchange because:

- IBKR instruments are contracts.
- Contract metadata includes `sec_type`, `exchange`, `currency`, and optional `con_id`.
- IBKR can serve FX, futures, stocks, and ETFs.
- IBKR crypto spot is disabled by default.
- Trading is unsupported in this version.

The IBKR adapter is intentionally separate from `internal/exchange/binance` and `internal/exchange/kraken`.

## Monitoring-Only Design

The system estimates opportunities and surfaces them to humans and external monitoring systems. It does not attempt execution.

Reasons:

- Execution needs risk management, balances, positions, order state, and retry semantics.
- Depth snapshots can become stale before an order reaches a venue.
- Transfer times and withdrawal fees can dominate cross-exchange opportunities.
- Broker and exchange trading APIs should be introduced only with explicit safety boundaries.

## Estimates, Not Guarantees

All arbitrage outputs are estimates. They can be false positives because of:

- Stale data
- Partial order book depth
- Hidden liquidity
- Latency
- Funding changes
- Trading fees not fully modeled
- Withdrawal/deposit costs not modeled
- Transfer-time risk
- Exchange-specific order constraints

## Design Tradeoffs

### Simplicity vs Realism

The system adds realism through order book simulation but avoids full exchange execution modeling. This keeps calculations testable and fast, but does not guarantee executable opportunities.

### In-Memory State vs Database

`internal/marketdata.Store` keeps latest state only. This is simple and low-latency. The tradeoff is that the app itself cannot query history; Prometheus must scrape `/metrics` when history is needed.

### REST API First vs WebSocket First

REST snapshots are simple for the TUI and external tools. The tradeoff is polling overhead and less real-time behavior than a streaming API. `internal/api/websocket.go` contains a `SnapshotBroadcaster` interface, but WebSocket snapshot broadcasting is not currently wired as a primary API.

### TUI Dashboard vs Web UI

The TUI is lightweight, scriptable, and useful over SSH. The tradeoff is limited layout and visualization compared with a browser UI.

The web UI provides routed pages, browser tables, filters, settings persistence, API status visibility, and responsive layout. The tradeoff is a Node/Vite toolchain for development and a browser/CORS deployment surface. The web UI stays read-only and uses the backend REST API instead of provider SDKs.

### Redux / RTK Query vs Manual Fetching

The web UI uses Redux Toolkit and RTK Query so API cache ownership, polling, refetch state, and stale/error display are centralized. This avoids copying backend snapshots into multiple custom slices. The tradeoff is a larger frontend dependency footprint than a small hand-written fetch client.

### OpenAPI Types vs Hand-Written API Models

The web UI generates TypeScript types from `swagger.yml`. This keeps client models aligned with documented API schemas and reduces drift. The tradeoff is that maintainers must rerun `npm run generate:api` after changing the OpenAPI file.

### Browser Settings vs Backend Config

The web UI stores display preferences such as refresh interval, compact mode, emoji mode, and filters in browser local storage. Backend strategy behavior, provider enablement, API keys for the Go service, and safety settings remain in `.env` and `config.yaml`. If a user saves a web UI API key in the browser, it is stored only for that browser origin and should be treated as less secure than environment configuration.

### Config-Driven Instruments vs Auto-Discovery

Config-driven instruments make behavior explicit and reproducible. Auto-discovery exists partially through provider market metadata, but validation and strategies still rely heavily on configured assets/instruments.

## Architecture Gaps

- Live IBKR market-data transport is planned/partial.
- A dedicated provider manager package is not present.
- Metrics are implemented in `internal/api` rather than `internal/metrics`.
- Live market discovery is not performed by the config validation command.
