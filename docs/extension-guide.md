# Extension Guide

This guide explains how to add new capabilities while preserving the monitoring-only architecture.

## Add a New Crypto Exchange

The built-in crypto exchange platform list is OKX, Bybit, Binance, Kraken, Coinbase, Gate.io, and Bitget. Binance and Kraken have dedicated adapters. OKX, Bybit, Coinbase, Gate.io, and Bitget share the `internal/exchange/publicrest` spot adapter and are disabled by default in config.

1. Create an adapter package:

   ```text
   internal/exchange/<name>
   ```

2. Implement the exchange/provider methods:

   - `Name`
   - `Type`
   - `Start`
   - `Stop`
   - `Health`
   - `GetLatestTickers`
   - `GetLatestFuturesTickers`
   - `GetFundingRates`
   - `GetLatestOrderBooks`
   - `GetMarkets`
   - `DiscoverMarkets`

3. Normalize symbols into `BASE/QUOTE` using helpers in `internal/exchange`.

4. Normalize market data into:

   - `exchange.Ticker`
   - `exchange.OrderBook`
   - `exchange.FundingRate`
   - `exchange.MarketInfo`

5. Add config under `providers`.

6. Wire the adapter in `internal/app.New`.

7. Add health reporting:

   - WebSocket connected
   - REST fallback
   - last message time
   - reconnect count
   - stale counts
   - last error

8. Add unit tests for:

   - symbol normalization
   - parser behavior
   - order book normalization
   - health edge cases

## Add a New Broker

1. Create a broker adapter:

   ```text
   internal/broker/<name>
   ```

2. Implement `provider.BrokerProvider`.

3. Define broker-specific instrument mapping.

4. Add config fields under `ProviderConfig` only if they are generally useful; otherwise keep broker-specific fields nested in instruments.

5. Add instrument universe support if needed.

6. Wire the broker in `internal/app.New`.

7. Add TUI sections only through backend snapshot/API data.

8. Keep trading disabled unless a future version explicitly introduces a paper or execution module.

## Add a New Arbitrage Strategy

1. Define strategy config in `internal/config`.

2. Define output model in `internal/arbitrage/model.go`.

3. Identify inputs:

   - tickers
   - order books
   - funding rates
   - provider/instrument metadata
   - fees

4. Add calculation function in `internal/arbitrage`.

5. Use `decimal.Decimal` for all financial math.

6. Use `SimulateBuyWithQuote` and `SimulateSellBase` for depth-aware execution estimates.

7. Wire calculation in `internal/app.App.calculate`.

8. Store results in `internal/marketdata.Store`.

9. Add API endpoint in `internal/api.Server`.

10. Add TUI rendering in `internal/tui/render.go`.

11. Add unit tests:

   - full liquidity
   - partial liquidity
   - fee behavior
   - slippage behavior
   - threshold/result sorting

## Add a New TUI Tab

1. Add tab config in `TUIConfig` and `configs/config.example.yaml`.

2. Ensure backend snapshot/API includes required data.

3. Add tab key handling in `Model.Update`.

4. Add rendering method in `render.go`.

5. Add detail panel support if rows are selectable.

6. Add help/footer labels.

7. Add basic render tests if possible.

## Add a New Web UI Page

1. Ensure the backend API or `marketdata.Snapshot` exposes the required read-only data.

2. Update `swagger.yml` if the API contract changes.

3. Regenerate web UI types:

   ```bash
   cd web-ui
   npm run generate:api
   ```

4. Add or extend the RTK Query endpoint in `web-ui/src/api/arbApi.ts`.

5. Add a routed page under `web-ui/src/pages`.

6. Add navigation in `web-ui/src/components/Layout.tsx`.

7. Keep API data in RTK Query cache; add Redux slice state only for UI settings, filters, or client request status.

8. Add focused tests for formatting, filtering, stale/error display, or safety labels.

9. Run:

   ```bash
   cd web-ui
   npm run typecheck
   npm run lint
   npm run test
   npm run build
   ```

## Add a New Notifier

1. Implement:

   ```go
   type Notifier interface {
       Notify(ctx context.Context, alert Alert) error
   }
   ```

2. Add notifier config.

3. Wire the notifier into `alerts.Engine` or an alert dispatch layer.

4. Preserve deduplication and cooldown semantics.

5. Add tests for:

   - notification on first alert
   - no notification during cooldown
   - repeat when value changes enough
   - notifier failure handling

## Add Historical Storage

Recommended approach:

- Keep `marketdata.Store` latest-state only.
- Use Prometheus to scrape `/metrics` and retain history outside the application.
- Avoid blocking provider ingestion or API responses on database writes.

## Add Metrics

Current metrics live in `internal/api/server.go`. For larger metrics work:

1. Create `internal/metrics`.
2. Move metric formatting/collection there.
3. Consider Prometheus Go client if label cardinality is controlled.
4. Keep `/metrics` public unless config says otherwise.

## Safety Rules for Extensions

- Do not add real trading to monitoring-only packages.
- Do not let the TUI or web UI connect directly to providers.
- Do not use `float64` for financial calculations.
- Do not mix broker instruments into crypto exchange strategies unless explicitly configured.
- Mark partial provider support clearly in health and docs.
- Do not add browser execution controls, order forms, private key handling, or exchange secret inputs to the web UI.
