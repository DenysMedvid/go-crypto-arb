# Web UI Design

The web UI lives in `web-ui/` as a separate React application. It is an alternative read-only client for the backend REST API and does not replace the Bubble Tea terminal UI.

The web UI follows the same boundary as the TUI:

- The backend is the only process that talks to exchanges and IBKR.
- Arbitrage, alerting, health scoring, and snapshot assembly stay in Go.
- The web UI reads REST API responses and renders them.
- The web UI does not place orders, submit trades, collect exchange secrets, or manage IBKR orders.

## Folder Choice

The folder is named `web-ui/` to keep the browser application isolated from backend Go packages while making its purpose obvious at the repository root.

Important files:

- `web-ui/package.json`: npm scripts and dependency list
- `web-ui/vite.config.ts`: Vite and Vitest config
- `web-ui/src/api/arbApi.ts`: RTK Query API layer
- `web-ui/src/api/schema.ts`: generated OpenAPI TypeScript schema
- `web-ui/src/app/store.ts`: Redux store
- `web-ui/src/features`: settings, filters, and API status slices
- `web-ui/src/pages`: routed screens
- `web-ui/src/components`: shared display components
- `web-ui/src/utils`: formatting, filters, stale-data, API error, and price-highlight helpers

## Stack

The web UI uses:

- Vite
- React
- TypeScript with strict settings
- Redux Toolkit
- RTK Query
- React Router
- Vitest
- React Testing Library
- ESLint
- Prettier

No UI component library is currently used. Styling is implemented with focused CSS in `src/styles.css` to keep the dependency surface small.

## API Contract

The API type source is the repository root `swagger.yml`. Generated types are written with:

```bash
cd web-ui
npm run generate:api
```

This produces `src/api/schema.ts`, and `src/api/types.ts` provides friendly aliases for the generated schemas.

The web UI uses the documented REST endpoints:

- `GET /health`
- `GET /api/v1/snapshot`
- `GET /api/v1/prices`
- `GET /api/v1/order-books`
- `GET /api/v1/providers`
- `GET /api/v1/providers/health`
- `GET /api/v1/exchanges/health`
- `GET /api/v1/arbitrage/triangular`
- `GET /api/v1/arbitrage/cross-exchange`
- `GET /api/v1/arbitrage/spot-futures`
- `GET /api/v1/signals/related-assets`
- `GET /api/v1/alerts`
- `GET /api/v1/ibkr/instruments`
- `GET /api/v1/ibkr/fx-triangular`
- `GET /api/v1/ibkr/crypto-futures-basis`

`/health` is public. Protected `/api/v1/*` requests include `X-API-Key` when a key is configured.

Financial values are displayed from API decimal fields. The UI converts decimals to numbers only for local rendering decisions such as sorting, filtering, and color classification; it does not rewrite API cache state.

## CORS

Browser clients need CORS because the Vite dev server and API server run on different origins. The backend CORS middleware allows loopback origins such as `http://localhost:<port>` and `http://127.0.0.1:<port>` automatically for local development.

Deployed web UI origins should be configured explicitly:

```yaml
api:
  cors_allowed_origins:
    - "https://arb.example.com"
```

Preflight `OPTIONS` requests are answered before API-key middleware. The real protected request still needs `X-API-Key`.

## Routing

React Router maps the required screens:

| Route | Screen |
| --- | --- |
| `/` | Crypto Dashboard |
| `/prices` | Prices |
| `/triangular` | Crypto Triangular Arbitrage |
| `/cross-exchange` | Cross-Exchange Arbitrage |
| `/spot-futures` | Crypto Spot-Futures Arbitrage |
| `/signals` | Related Asset Signals |
| `/alerts` | Alerts |
| `/provider-health` | Provider / Exchange Health |
| `/ibkr` | IBKR Monitor |
| `/status` | API / Backend Status |
| `/settings` | Settings |

The sidebar is the primary navigation. The top bar shows backend status, base URL, last success time, latency, refresh countdown, and refresh controls.

## State Management

Redux state is intentionally small:

- `arbApi`: RTK Query server cache
- `settings`: API base URL, API key source, refresh interval, compact mode, emoji toggle, profitable-only filter, theme mode
- `filters`: page filters for prices and alerts
- `apiStatus`: last successful request, last failed request, latency, auth failure, backend unavailable, last error

The UI does not duplicate API response data in custom slices. RTK Query owns the API cache and keeps the last successful data visible during refetch failures.

## Settings Persistence

Persisted browser settings:

- API base URL
- refresh interval
- auto-refresh toggle
- compact mode
- emoji/icons toggle
- profitable-only filter
- theme mode

The API key is persisted only when the user explicitly saves it in Settings. The web UI documentation warns that browser local storage is readable by scripts running on the same origin. Environment configuration through `VITE_API_KEY` is preferred for local development when practical.

## Refresh Behavior

Each page uses RTK Query polling with the configured interval. The default is `2000ms`, matching the TUI/default config behavior when no backend-specific interval is available.

Controls:

- manual refresh invalidates active RTK Query tags
- pause/resume disables or enables polling
- the header shows refresh countdown or paused state
- failed refetches show a banner while cached data remains visible

## Error Handling

`src/utils/apiErrors.ts` maps API failures into user-facing states:

- `401`: authentication failed
- `FETCH_ERROR`: backend unavailable or blocked by browser/network
- `PARSING_ERROR`: backend returned an unexpected response format
- other HTTP statuses: backend returned an error

The UI shows stale and partial data clearly rather than clearing the last good state.

## Price Coloring

The web UI mirrors the TUI price-coloring logic:

- highest bid for a visible market/symbol is green
- lowest bid for a visible market/symbol is red
- lowest ask for a visible market/symbol is green
- highest ask for a visible market/symbol is red
- tied prices are not highlighted
- stale rows remain visibly marked

The implementation lives in `src/utils/priceHighlights.ts`. It compares prices within `market_type + symbol` buckets so spot and futures prices are not mixed.

## Screen Design

### Crypto Dashboard

The dashboard summarizes:

- spot price count
- active alert count
- health summary
- IBKR summary
- top estimated opportunities
- price cards grouped by provider/exchange

It uses “estimated,” “watch,” and “not guaranteed executable” language to preserve monitoring-only semantics.

### Prices

The prices page shows spot and futures tickers with filters for provider, market type, symbol search, and stale-only rows.

### Arbitrage Pages

The triangular, cross-exchange, and spot-futures pages render backend estimates. They show fill status, slippage/basis/funding context, and no execution controls.

### Related Asset Signals

Signals are grouped by configured backend signal groups and show change, group average, divergence, and a derived display label.

### Alerts

The alerts page shows current in-memory alerts and filters by severity, type, and provider/symbol.

### Provider / Exchange Health

The health page shows status, score, WebSocket state, REST fallback, last message age, reconnects, last error, and stale data counts.

### IBKR Monitor

IBKR is kept separate from crypto exchange pages. The page shows:

- IBKR status
- market-data mode
- `Trading: DISABLED` when disabled
- configured IBKR instruments
- IBKR FX triangular estimates
- crypto spot vs IBKR futures basis estimates
- IBKR health

The UI does not mix IBKR instruments into crypto triangular arbitrage.

### API / Backend Status

The status page shows:

- `/health`
- API base URL
- API key status without exposing the full key
- last successful request
- last failed request
- latency
- error details
- endpoint support notes

### Settings

Settings include:

- API base URL
- API key input
- auto-refresh interval
- auto-refresh toggle
- compact mode
- emoji/icons toggle
- profitable-only filter
- theme mode

## Testing

The web UI test suite covers:

- decimal display helpers
- price filtering
- TUI-equivalent price highlighting
- API error classification
- settings persistence
- stale data display
- profitable-only filtering
- IBKR `Trading: DISABLED` display
- basic summary card rendering

Commands:

```bash
cd web-ui
npm run typecheck
npm run lint
npm run test
npm run build
```

## Deployment

Development:

```bash
cd web-ui
npm install
npm run generate:api
npm run dev
```

Production:

```bash
cd web-ui
npm run build
```

The static build output is `web-ui/dist/`. It can be served by Nginx, another static host, or a future backend static-file mode. The current backend does not need to serve the web assets.

## Current Limitations

- The web UI is polling-based, like the TUI.
- Browser local storage API-key persistence is optional and should be used carefully.
- Historical charts are not built into the web UI; Prometheus/Grafana remain the history path.
- IBKR live market data depends on backend support, which is partial/planned.
- `GET /api/v1/metrics/snapshot` is not implemented; the backend intentionally returns `404`.
