# Patterns

This project uses several common architecture and design patterns. Some are fully implemented, while others are emerging as the provider architecture evolves.

## 1. Hexagonal Architecture / Ports and Adapters

Where it appears:

- Provider interfaces in `internal/provider/provider.go` are target ports.
- `exchange.Exchange` in `internal/exchange/model.go` is the current crypto exchange port.
- Binance, Kraken, shared public-REST exchange, and IBKR packages are adapters.
- REST API, TUI, and web UI are external boundaries.

Why it helps:

- Provider-specific transport and parsing stay out of arbitrage calculations.
- The TUI and web UI depend on REST models, not provider SDKs.
- Tests can feed normalized tickers/order books directly into calculators.

Drawbacks:

- The codebase is in transition: crypto adapters still use `exchange.Exchange` directly, while `internal/provider` defines the broader target abstraction.

## 2. Strategy Pattern

Where it appears:

- `CalculateTriangularV2`
- `CalculateCrossExchangeV2`
- `CalculateSpotFuturesV2`
- `CalculateIBKRFXTriangular`
- `CalculateBrokerFuturesBasis`
- `SignalEngine.Update`

Why it helps:

- Each strategy has independent inputs, rules, outputs, and tests.
- New strategies can be added without changing provider adapters.

Drawbacks:

- Shared concerns such as fees, slippage limits, and provider filters must be kept consistent across strategy functions.

## 3. Adapter Pattern

Where it appears:

- Binance converts REST/WebSocket payloads into `exchange.Ticker`, `exchange.OrderBook`, and `exchange.FundingRate`.
- Kraken converts REST/WebSocket payloads and Kraken symbols into canonical symbols.
- IBKR converts configured contracts/instruments into `exchange.MarketInfo`.

Why it helps:

- Provider-specific symbol formats do not leak into the arbitrage engine.
- The rest of the system can rely on canonical symbols such as `BTC/USDT`.

Drawbacks:

- Symbol normalization is easy to get subtly wrong for assets like `XBT`/`BTC` or broker contracts without clear base/quote semantics.

## 4. Repository / State Store Pattern

Where it appears:

- `internal/marketdata.Store`

Why it helps:

- Centralizes latest-state reads/writes.
- Gives API handlers one snapshot source.
- Avoids passing many provider-specific maps through the application.

Drawbacks:

- In-memory state is lost on restart.
- There is no built-in historical query model.

## 5. Worker / Producer-Consumer Pattern

Where it appears:

- Binance/Kraken REST polling loops, WebSocket loops, and public-REST spot polling loops produce market data.
- `internal/app.App.calculate` consumes latest provider state and produces derived results.

Why it helps:

- Provider ingestion can run independently from calculation cadence.
- REST fallback and WebSocket updates can coexist.

Drawbacks:

- Without a queue, the store observes latest values only; intermediate updates can be overwritten.

## 6. Facade Pattern

Where it appears:

- `internal/api.Server` exposes simplified current-state endpoints.
- `/api/v1/snapshot` aggregates prices, order books, opportunities, alerts, health, and IBKR data.
- `web-ui/src/api/arbApi.ts` exposes a typed RTK Query facade for browser components.

Why it helps:

- TUI, web UI, and external tools do not need to know internal package boundaries.
- The API stabilizes the backend/client contract.

Drawbacks:

- Large snapshot responses can grow as more data is added.

## 7. Dependency Injection

Where it appears:

- `app.New(cfg, env, logger)`
- `api.NewServer(cfg, store, apiKey, logger)`
- Provider constructors receive config, known assets, stale durations, and logger.
- `tui.NewModel(cfg, env)` receives TUI settings and API key.
- The web UI Redux store injects the current API base URL and API key into RTK Query requests.

Why it helps:

- Makes dependencies visible.
- Simplifies testing with explicit config and fixtures.

Drawbacks:

- Manual wiring in `internal/app` grows as more providers are added.

## 8. Observer-like / Polling Refresh Pattern

Where it appears:

- The TUI periodically calls `/api/v1/snapshot`.
- Manual refresh and pause/resume are implemented in the Bubble Tea update loop.
- The web UI polls focused API endpoints through RTK Query and supports manual refresh and pause/resume.

Why it helps:

- Simple, robust, and easy to debug.
- Avoids terminal/browser UI complexity around streaming events.

Drawbacks:

- Polling can miss short-lived changes and adds periodic HTTP overhead.

## 9. Circuit Breaker-like Health Handling

Where it appears:

- `internal/health.Score`
- Provider health fields such as `WebSocketConnected`, `RestFallbackActive`, `StaleTickerCount`, and `ReconnectCount`.

Why it helps:

- Operators can see degraded/stale/disconnected providers.
- Alerts can be generated for stale data or low health scores.

Drawbacks:

- This is not a real circuit breaker that stops traffic; it is health scoring and status reporting.

## 10. Null Object / Partial Implementation

Where it appears:

- IBKR account/portfolio methods return clear `not_implemented` results.
- IBKR starts and reports a clear disconnected/partial status when live market-data transport is unavailable.
- Kraken futures support is marked partial.
- OKX, Bybit, Coinbase, Gate.io, and Bitget mark partial support if futures are enabled because their current adapters are spot public REST only.

Why it helps:

- The project compiles and exposes safe status even before full provider support exists.
- Partial capabilities are explicit rather than hidden behind panics.

Drawbacks:

- Users must read health/status and docs to understand which provider features are incomplete.
