# go-crypto-arb Web UI

This folder contains a separate browser UI for the existing `go-crypto-arb` backend. It is an alternative client to the Bubble Tea terminal UI; it does not replace the TUI and it does not connect directly to exchanges or IBKR.

The folder is named `web-ui/` to keep the React application isolated from the Go backend while making its purpose obvious at the repository root.

## Stack

- Vite
- React
- TypeScript with strict compiler settings
- Redux Toolkit
- RTK Query
- React Router
- Vitest
- React Testing Library
- ESLint
- Prettier

No component library is used. The UI uses focused React components and CSS so the dependency surface stays small.

## Configure

Create a local Vite env file when needed:

```env
VITE_API_BASE_URL=http://localhost:8080
VITE_API_KEY=change-me
```

`/health` is called without an API key. Protected `/api/v1/*` requests include:

```text
X-API-Key: <configured key>
```

You can also set the API key in Settings. If you save it there, the key is stored in browser local storage for this origin. This is convenient for a local dashboard but is less safe than environment configuration because scripts running on the same origin can read local storage.

## Install

```bash
cd web-ui
npm install
npm run generate:api
```

`generate:api` reads the repository root `swagger.yml` and writes `src/api/schema.ts`.

## Run

Start the Go backend separately:

```bash
go run ./cmd/api
```

Start the web UI dev server:

```bash
cd web-ui
npm run dev
```

The default Vite URL is `http://127.0.0.1:5173`.

The backend allows loopback browser origins such as `http://127.0.0.1:5173`
and `http://127.0.0.1:5174` for local development. If the web UI is deployed
somewhere else, add that origin to backend config:

```yaml
api:
  cors_allowed_origins:
    - "https://arb.example.com"
```

## Build

```bash
cd web-ui
npm run build
```

Production assets are written to `web-ui/dist/`. They can be served by Nginx, a static file host, or a future backend static-file mode.

## Validate

```bash
npm run typecheck
npm run lint
npm run test
```

## Relationship To The TUI

The web UI follows the same boundary as the TUI:

- The backend owns market data, arbitrage calculations, alerts, health, and IBKR provider state.
- The UI reads backend REST endpoints only.
- The UI is monitoring-only and has no trading buttons, order forms, exchange secret inputs, or IBKR order placement.
- IBKR is displayed separately from crypto exchange arbitrage views.

The web UI adds browser affordances such as routed pages, filters, responsive tables, persisted display settings, API status visibility, and clearer stale/error states.
