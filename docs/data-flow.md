# Data Flow

This document describes how data moves through the running system.

## Summary

```text
Provider REST/WebSocket
  -> Provider Adapter
  -> Normalized Ticker / OrderBook / FundingRate
  -> In-Memory Store
  -> Arbitrage Engine
  -> Alert Engine
  -> REST API
  -> TUI Client / Web UI Client
```

The backend is the producer and owner of state. The TUI and web UI are polling read-only clients.

## Startup Flow

1. `cmd/api/main.go` loads `.env` through `config.LoadEnv`.
2. `cmd/api/main.go` loads YAML config through `config.Load`.
3. Environment overrides are applied with `cfg.ApplyEnv`.
4. `app.New` constructs the in-memory store, enabled crypto exchange adapters, optional IBKR broker adapter, signal engine, and alert engine.
5. `App.Run` starts configured providers.
6. `App.calculate` runs once to seed derived state.
7. A calculation loop starts using `app.refresh_interval`.
8. `internal/api.Server` starts HTTP routes.

```mermaid
sequenceDiagram
    participant Main as cmd/api
    participant Config as internal/config
    participant App as internal/app
    participant Providers as Provider Adapters
    participant API as internal/api

    Main->>Config: LoadEnv()
    Main->>Config: Load(config.yaml)
    Main->>Config: ApplyEnv(env)
    Main->>App: New(cfg, env, logger)
    App->>Providers: construct enabled crypto exchanges / IBKR
    Main->>App: Run(ctx)
    App->>Providers: Start(ctx)
    App->>App: calculate()
    App->>API: NewServer(...).Handler()
    API-->>Main: ListenAndServe()
```

## Provider Startup

Binance and Kraken adapters start REST polling loops and WebSocket loops when configured. OKX, Bybit, Coinbase, Gate.io, and Bitget use spot public-REST polling when enabled. Adapters update internal maps protected by mutexes. `internal/app` periodically pulls snapshots out of each adapter.

IBKR currently loads configured instruments and reports health. Live TWS Gateway market-data transport is partial/planned.

## Market Data Ingestion

Provider adapters normalize provider-specific payloads:

- Binance book ticker/depth payloads
- Binance funding rates
- Kraken asset pairs/ticker/depth payloads
- Kraken futures ticker payloads
- OKX, Bybit, Coinbase, Gate.io, and Bitget spot ticker/depth payloads
- IBKR configured instrument metadata

Normalized output types live in `internal/exchange/model.go`:

- `Ticker`
- `OrderBook`
- `FundingRate`
- `MarketInfo`
- `ExchangeHealth`

```mermaid
sequenceDiagram
    participant Provider as External Provider
    participant Adapter as Provider Adapter
    participant App as internal/app
    participant Store as marketdata.Store
    participant Health as health.Score

    Provider-->>Adapter: REST poll or WebSocket message
    Adapter->>Adapter: parse provider payload
    Adapter->>Adapter: normalize symbols and decimals
    App->>Adapter: GetLatestTickers / GetLatestOrderBooks / GetFundingRates
    Adapter-->>App: normalized latest state
    App->>Store: Upsert tickers, order books, funding
    App->>Adapter: Health()
    Adapter-->>App: raw health
    App->>Health: Score(config, health)
    Health-->>App: scored health
    App->>Store: SetExchangeHealth()
```

## Store Update

`internal/app.App.calculate` collects latest provider state and writes it into `internal/marketdata.Store`:

- `UpsertSpotTickers`
- `UpsertFuturesTickers`
- `UpsertFundingRates`
- `UpsertOrderBooks`
- `SetMarkets`
- `SetExchangeHealth`

The store is latest-state only. It does not persist historical changes.

## Arbitrage Calculation

After updating raw market data, `App.calculate` runs:

- `arbitrage.CalculateTriangularV2`
- `arbitrage.CalculateCrossExchangeV2`
- `arbitrage.CalculateSpotFuturesV2`
- `arbitrage.CalculateIBKRFXTriangular`
- `arbitrage.CalculateBrokerFuturesBasis`
- `SignalEngine.Update`

Results are written through:

- `Store.SetCalculations`
- `Store.SetBrokerCalculations`

```mermaid
sequenceDiagram
    participant App as internal/app
    participant Store as marketdata.Store
    participant Arb as internal/arbitrage
    participant Alerts as alerts.Engine

    App->>Store: Upsert latest market data
    App->>Arb: CalculateTriangularV2
    Arb-->>App: triangular opportunities
    App->>Arb: CalculateCrossExchangeV2
    Arb-->>App: cross-exchange opportunities
    App->>Arb: CalculateSpotFuturesV2
    Arb-->>App: spot-futures opportunities
    App->>Arb: CalculateIBKRFXTriangular
    Arb-->>App: IBKR FX opportunities
    App->>Arb: CalculateBrokerFuturesBasis
    Arb-->>App: broker basis opportunities
    App->>Store: SetCalculations and SetBrokerCalculations
    App->>Alerts: Evaluate
    Alerts-->>App: deduplicated alerts
    App->>Store: SetAlerts
```

## Alert Generation

`alerts.Engine.Evaluate` consumes arbitrage results and health snapshots. It applies:

- Threshold checks
- Deduplication keys
- Cooldown
- Repeat-if-value-changed threshold
- Severity selection

The final alert list is written through `Store.SetAlerts`.

## Health Scoring

Each provider adapter exposes a health snapshot. `internal/health.Score` applies deterministic penalties and writes `Score` and `Status`.

Health status is exposed through:

- `/api/v1/exchanges/health`
- `/api/v1/providers/health`
- `/api/v1/snapshot`
- `/metrics`

## REST API Response Flow

API handlers call `store.Snapshot()` and serialize current state. Protected endpoints require `X-API-Key`.

Important snapshot endpoints:

- `/api/v1/snapshot`
- `/metrics`

Browser clients also rely on CORS preflight handling. The backend answers `OPTIONS` requests before API-key middleware, then the actual protected `/api/v1/*` request must include `X-API-Key`.

## TUI Refresh Flow

The TUI model in `internal/tui/model.go` periodically runs `fetchSnapshot`, which calls `Client.Snapshot` in `internal/tui/client.go`. The response is stored in the Bubble Tea model and rendered by `internal/tui/render.go`.

```mermaid
sequenceDiagram
    participant TUI as TUI Model
    participant Client as internal/tui.Client
    participant API as Backend API
    participant Store as marketdata.Store

    TUI->>TUI: tick / manual refresh
    TUI->>Client: Snapshot(ctx)
    Client->>API: GET /api/v1/snapshot + X-API-Key
    API->>Store: Snapshot()
    Store-->>API: marketdata.Snapshot
    API-->>Client: JSON snapshot
    Client-->>TUI: snapshotMsg
    TUI->>TUI: update model
    TUI->>TUI: render View()
```

## Web UI Refresh Flow

The web UI in `web-ui/` uses RTK Query polling from React pages. The API layer reads the current API base URL and API key from Redux settings, adds `X-API-Key` to protected `/api/v1/*` requests, and records latency and errors in `apiStatus`.

```mermaid
sequenceDiagram
    participant Browser as Browser / React Page
    participant Store as Redux Store
    participant Query as RTK Query API Layer
    participant API as Backend API
    participant State as marketdata.Store

    Browser->>Store: read settings and filters
    Browser->>Query: subscribe to endpoint query
    Query->>API: OPTIONS preflight when needed
    API-->>Query: CORS headers
    Query->>API: GET endpoint + X-API-Key
    API->>State: Snapshot or focused state read
    State-->>API: latest state
    API-->>Query: JSON response
    Query->>Store: cache response and request status
    Store-->>Browser: render cards, filters, tables, stale/error state
```

Manual refresh invalidates RTK Query tags. Pausing refresh sets polling to zero. Failed refetches keep the last successful cache entry visible.

## Additional Diagrams

- [data-flow.mmd](diagrams/data-flow.mmd)
- [sequence-market-data.mmd](diagrams/sequence-market-data.mmd)
- [sequence-arbitrage-calculation.mmd](diagrams/sequence-arbitrage-calculation.mmd)
- [sequence-tui-refresh.mmd](diagrams/sequence-tui-refresh.mmd)
- [sequence-web-ui-refresh.mmd](diagrams/sequence-web-ui-refresh.mmd)

## Architecture Gaps

- Market data updates are pulled from adapters during `calculate`; there is no central event bus or durable queue.
- WebSocket snapshot broadcasting is not a primary implemented client flow.
- Live IBKR market-data ingestion is partial/planned.
- The web UI is polling-based and does not yet use a server-push snapshot stream.
