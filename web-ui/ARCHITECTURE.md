# Architecture

The web UI is a separate Vite React app under `web-ui/`. It is a read-only client for the existing Go backend.

## Structure

```text
web-ui/
├── src/api
│   ├── arbApi.ts        RTK Query endpoints and API key header handling
│   ├── schema.ts        Generated from ../swagger.yml
│   └── types.ts         Friendly aliases for generated schemas
├── src/app
│   └── store.ts         Redux store setup
├── src/components       Shared layout, tables, status badges, cards, modal
├── src/features         Redux slices for settings, filters, API status
├── src/hooks            Typed Redux hooks and refresh helpers
├── src/pages            Routed screen implementations
└── src/utils            Formatting, stale-data, filters, API error helpers
```

## Redux Store

The store contains:

- `arbApi`: RTK Query cache and request state
- `settings`: API base URL, API key, refresh interval, compact mode, emoji mode, profitable-only filter, theme mode
- `filters`: prices and alerts filters
- `apiStatus`: last successful request, last failed request, latency, auth failure, backend unavailable, last error

API response data is not copied into custom slices. RTK Query owns server cache.

## RTK Query API Layer

`src/api/arbApi.ts` uses a dynamic base query so requests always read the latest API base URL and API key from Redux settings.

Rules:

- `/health` is public and receives no API key header.
- `/api/v1/*` requests receive `X-API-Key` when a key is configured.
- request latency and failures are recorded in `apiStatus`.
- generated OpenAPI types from `src/api/schema.ts` define response shapes.

## Routing

React Router maps the required screens:

- `/` Crypto Dashboard
- `/prices`
- `/triangular`
- `/cross-exchange`
- `/spot-futures`
- `/signals`
- `/alerts`
- `/provider-health`
- `/ibkr`
- `/status`
- `/settings`

The sidebar is the primary navigation. The layout keeps backend status, refresh controls, latency, last success time, and error banners visible.

## Refresh Strategy

Each page uses RTK Query polling with the configured interval. The default is `2000ms`, matching the TUI/default config behavior when no backend-specific interval is available.

Manual refresh invalidates all active RTK Query tags. Pause/resume sets polling to `0` or the configured interval. The header displays a countdown state.

When a refetch fails, RTK Query keeps the last successful cache entry. The UI shows an error banner and leaves existing table/card data visible as cached data.

## Settings Persistence

Safe UI settings are persisted to browser local storage:

- API base URL
- refresh interval
- auto-refresh toggle
- compact mode
- emoji/icons toggle
- profitable-only filter
- theme mode

The API key is persisted only when the user explicitly saves it in Settings. Documentation and UI copy warn that browser local storage is readable by scripts on the same origin.

## Component Structure

- `Layout` owns sidebar navigation, header status, refresh actions, and global error banner.
- `DataTable` is the reusable typed table component.
- `SummaryCard`, `StatusBadge`, `PriceTable`, and `IBKRTradingStatus` provide focused display primitives.
- Page components handle query selection, filters, and table column definitions.

## Error Handling

`utils/apiErrors.ts` maps RTK Query errors into user-facing states:

- `401`: authentication failed
- `FETCH_ERROR`: backend unavailable
- `PARSING_ERROR`: backend response shape problem
- other HTTP statuses: backend returned an error

The UI avoids fake schemas or fake data. Missing endpoints and partial backend support are documented in `API_COMPATIBILITY.md` and rendered as empty states.

## Safety

The web UI is monitoring-only:

- no trading buttons
- no buy/sell execution controls
- no order forms
- no exchange secret inputs
- no IBKR order placement

Opportunity wording uses “estimated profit,” “basis,” “watch,” “partial fill,” and “not guaranteed executable.”
