# Modules

This document maps the actual repository packages and binaries.

## `cmd/api`

Responsibility:

- API service entrypoint.
- Loads environment and config.
- Runs `validate-config`.
- Starts `internal/app.App`.

Important functions:

- `main`
- `runValidateConfig`

Inputs:

- `.env`
- `configs/config.yaml`
- OS signals

Outputs:

- HTTP API service
- Config validation output

Dependencies:

- `internal/app`
- `internal/config`

Maintainer notes:

- `validate-config` currently performs structural validation and safety checks. It does not start providers for live discovery.

## `cmd/tui`

Responsibility:

- Terminal UI entrypoint.
- Loads config/env.
- Starts Bubble Tea model.

Important functions:

- `main`

Inputs:

- `.env`
- `config.yaml`

Outputs:

- TUI process

Dependencies:

- `internal/config`
- `internal/tui`

Maintainer notes:

- The TUI must remain a backend API client only.

## `web-ui`

Responsibility:

- Browser UI client.
- Renders routed dashboard screens.
- Calls backend REST endpoints through RTK Query.
- Stores safe UI preferences in browser local storage.
- Generates TypeScript API types from `swagger.yml`.

Important files:

- `package.json`
- `src/api/arbApi.ts`
- `src/api/schema.ts`
- `src/app/store.ts`
- `src/features/settingsSlice.ts`
- `src/features/filtersSlice.ts`
- `src/features/apiStatusSlice.ts`
- `src/pages/*`
- `src/components/*`
- `src/utils/*`

Inputs:

- `VITE_API_BASE_URL`
- `VITE_API_KEY`
- browser local storage settings
- backend REST API responses

Outputs:

- Static production assets in `web-ui/dist`
- Browser dashboard at the Vite/dev-server or static-host origin

Dependencies:

- React
- Redux Toolkit
- RTK Query
- React Router
- Vite
- Vitest / React Testing Library
- generated OpenAPI types from `swagger.yml`

Maintainer notes:

- The web UI must remain monitoring-only.
- The web UI must not connect directly to exchanges or IBKR.
- API key storage in browser local storage is optional and should be documented as a local convenience.
- Run `npm run generate:api` after changing `swagger.yml`.

## `internal/app`

Responsibility:

- Application orchestration.
- Builds provider adapters.
- Starts provider loops.
- Runs periodic calculation loop.
- Writes latest state into `marketdata.Store`.

Important types:

- `App`

Important functions:

- `New`
- `Run`
- `calculate`

Inputs:

- `config.Config`
- `config.Env`
- Provider snapshots

Outputs:

- Updated market data store
- HTTP server lifecycle

Dependencies:

- `internal/api`
- `internal/arbitrage`
- `internal/alerts`
- `internal/exchange/binance`
- `internal/exchange/kraken`
- `internal/exchange/publicrest`
- `internal/broker/ibkr`
- `internal/health`
- `internal/marketdata`

Maintainer notes:

- Provider management is embedded here. A future `provider manager` package could reduce wiring growth.

## `internal/api`

Responsibility:

- REST API handlers.
- API key middleware.
- Prometheus text output.

Important types:

- `Server`

Important files:

- `server.go`
- `middleware.go`
- `websocket.go`

Inputs:

- `marketdata.Store`
- API key
- HTTP requests

Outputs:

- JSON API responses
- Prometheus text metrics

Dependencies:

- `internal/config`
- `internal/exchange`
- `internal/marketdata`

Maintainer notes:

- There is no separate `internal/metrics` package; metrics are currently rendered in `server.go`.
- `websocket.go` defines a `SnapshotBroadcaster` interface but streaming snapshots are not the main implemented API.
- CORS middleware supports loopback web UI development origins and configured deployed origins.

## `internal/config`

Responsibility:

- YAML config structures.
- `.env` loading.
- Defaults and environment overrides.
- Decimal/duration parsing.
- Config validation.

Important types:

- `Config`
- `Env`
- `ProviderConfig`
- `InstrumentUniverse`
- `InstrumentConfig`
- `StrategiesConfig`
- `TUIConfig`
- `ValidationMessage`
- `Decimal`
- `Duration`

Important functions:

- `LoadEnv`
- `Load`
- `ApplyEnv`
- `Validate`
- `HasValidationErrors`

Inputs:

- `.env`
- YAML config

Outputs:

- Runtime config
- Validation messages

Dependencies:

- `github.com/joho/godotenv`
- `gopkg.in/yaml.v3`
- `github.com/shopspring/decimal`

Maintainer notes:

- Config supports both older `exchanges`/`arbitrage` and newer `providers`/`strategies` shapes.

## `internal/provider`

Responsibility:

- Target provider abstraction.
- Shared broker account/portfolio placeholder models.

Important interfaces:

- `MarketDataProvider`
- `CryptoExchangeProvider`
- `BrokerProvider`

Important types:

- `AccountSummary`
- `PortfolioSnapshot`
- `PortfolioPosition`

Maintainer notes:

- Crypto adapters still primarily implement `exchange.Exchange`; this package documents the broader provider target.

## `internal/exchange`

Responsibility:

- Shared crypto exchange models.
- Market type constants.
- Symbol parsing and normalization.
- Order book normalization.

Important types:

- `Exchange`
- `MarketType`
- `Ticker`
- `FundingRate`
- `OrderBookLevel`
- `OrderBook`
- `MarketInfo`
- `ExchangeHealth`

Important functions:

- `CanonicalSymbol`
- `NormalizeAsset`
- `NormalizeCanonicalSymbol`
- `SplitJoinedSymbol`
- `NormalizeOrderBook`
- Decimal parse helpers in `parse.go`

Maintainer notes:

- This package now contains provider-aware fields as well as exchange-oriented names.

## `internal/exchange/binance`

Responsibility:

- Binance market data adapter.
- Spot and futures ticker polling.
- Depth polling.
- Funding rate polling.
- WebSocket book ticker updates.

Important type:

- `Client`

Important methods:

- `Start`
- `Stop`
- `Health`
- `GetLatestTickers`
- `GetLatestFuturesTickers`
- `GetFundingRates`
- `GetLatestOrderBooks`
- `GetMarkets`
- `DiscoverMarkets`

Inputs:

- Binance REST and WebSocket public APIs.
- Known assets from config.

Outputs:

- Normalized tickers, order books, funding rates, markets, health.

Maintainer notes:

- Order book polling loops over known tickers, so configured assets influence breadth.

## `internal/exchange/kraken`

Responsibility:

- Kraken market data adapter.
- Asset pair discovery.
- Spot ticker/depth polling.
- Spot WebSocket ticker updates.
- Partial futures ticker/funding support.

Important type:

- `Client`

Maintainer notes:

- Kraken futures support is partial and uses top-of-book fallback for depth where full depth is unavailable.

## `internal/exchange/publicrest`

Responsibility:

- Shared spot public-REST adapter for OKX, Bybit, Coinbase, Gate.io, and Bitget.
- Spot ticker polling.
- Spot order book depth polling.
- Exchange-specific response parsing and symbol normalization.
- Health reporting for REST-only providers.

Important type:

- `Client`

Maintainer notes:

- The five public-REST platforms are disabled by default in config.
- Futures, funding rates, and WebSocket support are not implemented for these adapters yet.
- Keep platform-specific parsing covered by fixture tests because each venue uses a different response envelope.

## `internal/broker/ibkr`

Responsibility:

- IBKR broker provider skeleton.
- Configured instrument loading.
- IBKR health/status.
- Account/portfolio methods returning `not_implemented`.

Important type:

- `Client`

Important functions:

- `New`
- `ValidateInstrument`

Inputs:

- `config.ProviderConfig`
- `config.InstrumentConfig`

Outputs:

- Configured `exchange.MarketInfo`
- Broker health status

Maintainer notes:

- Live TWS Gateway market-data transport is not implemented.
- No trading code path exists.

## `internal/instrument`

Responsibility:

- Instrument universe helpers.
- IBKR instrument selection.
- Market type and display symbol helpers.

Important functions:

- `IBKRInstruments`
- `UniverseInstruments`
- `MarketType`
- `DisplaySymbol`

## `internal/marketdata`

Responsibility:

- Thread-safe latest-state repository.
- Snapshot assembly for API, TUI, and web UI.

Important types:

- `Store`
- `Snapshot`
- `OrderBookSummary`

Important methods:

- `UpsertSpotTickers`
- `UpsertFuturesTickers`
- `UpsertFundingRates`
- `UpsertOrderBooks`
- `SetMarkets`
- `SetExchangeHealth`
- `SetCalculations`
- `SetBrokerCalculations`
- `SetAlerts`
- `Snapshot`

Maintainer notes:

- Store keys are still exchange/symbol oriented in places. Provider-aware naming is present in snapshot models.

## `internal/arbitrage`

Responsibility:

- Fee helpers.
- Legacy ticker-only calculators.
- v2 order-book-aware calculators.
- Execution simulation.
- IBKR/broker basis calculators.
- Related asset signals.

Important types:

- `ExecutionSimulation`
- `LegSimulation`
- `TriangularOpportunityV2`
- `CrossExchangeOpportunityV2`
- `SpotFuturesOpportunityV2`
- `BrokerFuturesBasisOpportunity`
- `SignalEngine`

Important functions:

- `SimulateBuyWithQuote`
- `SimulateSellBase`
- `CalculateTriangularV2`
- `CalculateCrossExchangeV2`
- `CalculateSpotFuturesV2`
- `CalculateIBKRFXTriangular`
- `CalculateBrokerFuturesBasis`
- `ProfitPercent`
- `ApplyTakerFee`

## `internal/alerts`

Responsibility:

- Alert generation.
- Deduplication.
- Cooldown and repeat thresholds.
- Severity ordering.

Important types:

- `Alert`
- `AlertSeverity`
- `AlertType`
- `Notifier`
- `Engine`

Important functions:

- `NewEngine`
- `Evaluate`

Maintainer notes:

- `Notifier` exists as an extension point; Telegram/email/webhook are not implemented.

## `internal/health`

Responsibility:

- Provider health scoring.
- Freshness checks.
- Score clamping.

Important functions:

- `Fresh`
- `Score`
- `ClampScore`

## `internal/tui`

Responsibility:

- Backend API client.
- Bubble Tea model/update/view loop.
- Lip Gloss rendering.
- Tabs, detail panels, selection, icon/emoji layer.

Important types:

- `Client`
- `Model`
- `IconSet`

Important functions:

- `NewClient`
- `Client.Snapshot`
- `NewModel`
- `NewIconSet`

Maintainer notes:

- The TUI renders backend snapshots only. It must not import or call provider adapters.

## Tests

Tests currently cover:

- API key middleware
- Decimal/order book symbol helpers
- Execution simulation
- v1 and v2 arbitrage calculators
- IBKR/broker strategy calculations
- Alerts deduplication/repeat behavior
- Config parsing/validation
- Health scoring
- TUI icon helper and basic rendering
- Web UI utility, settings, stale display, API error, price highlighting, and IBKR safety rendering tests

## Naming Differences

Expected package `internal/metrics` is not present. Metrics are currently implemented in `internal/api/server.go`.
