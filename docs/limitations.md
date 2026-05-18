# Limitations

This project is intentionally monitoring-only. The following limitations are current design constraints, not bugs.

## Trading and Execution

- Monitoring only.
- No real trading.
- No order execution.
- No buy/sell automation.
- No IBKR orders.
- No paper trading module in this version.
- No execution risk engine.
- No position tracking.

## Account and Balance Data

- No private exchange account balances unless implemented later.
- No IBKR account/portfolio snapshot beyond explicit `not_implemented` placeholders.
- No balance-aware opportunity filtering.
- No margin or collateral modeling.

## Fees and Transfer Modeling

- No withdrawal/deposit fee modeling.
- No transfer-time modeling.
- No chain congestion modeling.
- No venue-specific min order size/tick size validation.
- Taker fee modeling is configurable but incomplete compared with real fee schedules.

## Market Data and Latency

- No latency/race-condition guarantee.
- No guaranteed executable arbitrage.
- Order book simulation improves realism but does not guarantee execution.
- Order book snapshots can become stale before an order could execute.
- Some providers may expose only limited depth.
- Some exchange futures endpoints may be partial.
- OKX, Bybit, Coinbase, Gate.io, and Bitget currently support spot public REST only; futures, funding rates, and WebSocket ingestion are not implemented for those adapters yet.

## IBKR

- IBKR support is market-data only.
- IBKR live TWS Gateway market-data transport is partial/planned.
- IBKR market data may require subscriptions.
- IBKR crypto spot is disabled by default.
- IBKR futures basis is not guaranteed arbitrage.
- IBKR contract lookup/conId resolution is not implemented.
- No IBKR execution module exists.

## State and Storage

- In-memory state only.
- No historical database in the app.
- State is lost on restart.
- Related asset signals use in-memory previous observations only.

## API and UI

- TUI uses polling rather than real-time streaming.
- Web UI uses RTK Query polling rather than real-time streaming.
- `/api/v1/snapshot` can grow as more data is added.
- `swagger.yml` is checked in, but it is not generated automatically from Go handlers.
- No authentication roles or user management beyond a single API key.
- Browser deployments require CORS configuration for non-loopback origins.
- Optional web UI API key persistence uses browser local storage and should be treated as a local convenience, not high-security secret storage.

## Metrics and Alerts

- Prometheus metrics are rendered manually in `internal/api`.
- Alert state is in memory only.
- No Telegram/email/webhook notifier implementation yet.
- No alert acknowledgement or silence model.

## Architecture Gaps

- Provider management is embedded in `internal/app`.
- `internal/provider` is the target abstraction, but crypto adapters still primarily implement `exchange.Exchange`.
- No dedicated `internal/metrics` package.
- Config validation does not perform live provider discovery.
