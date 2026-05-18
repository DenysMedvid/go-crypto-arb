# API Compatibility

Swagger/OpenAPI source: `../swagger.yml`

Generated TypeScript source: `src/api/schema.ts`

The web UI uses RTK Query endpoints typed from the generated OpenAPI schema. Decimal values are kept as the JSON values returned by the backend; display helpers convert them only for rendering and filtering.

## Endpoints Used

| Endpoint | Auth | Web UI Use | Backend Status |
| --- | --- | --- | --- |
| `GET /health` | Public | Header and API status page | Implemented |
| `GET /api/v1/snapshot` | `X-API-Key` | Crypto dashboard summary | Implemented |
| `GET /api/v1/prices` | `X-API-Key` | Prices page | Implemented |
| `GET /api/v1/order-books` | `X-API-Key` | API layer; future detail use | Implemented |
| `GET /api/v1/providers` | `X-API-Key` | API layer; future settings/status use | Implemented |
| `GET /api/v1/providers/health` | `X-API-Key` | Provider health and IBKR status | Implemented |
| `GET /api/v1/exchanges/health` | `X-API-Key` | API layer compatibility | Implemented, legacy overlap |
| `GET /api/v1/arbitrage/triangular` | `X-API-Key` | Crypto triangular page | Implemented |
| `GET /api/v1/arbitrage/cross-exchange` | `X-API-Key` | Cross-exchange page | Implemented |
| `GET /api/v1/arbitrage/spot-futures` | `X-API-Key` | Spot-futures page | Implemented |
| `GET /api/v1/signals/related-assets` | `X-API-Key` | Related asset signals page | Implemented |
| `GET /api/v1/alerts` | `X-API-Key` | Alerts page | Implemented |
| `GET /api/v1/ibkr/instruments` | `X-API-Key` | IBKR instruments table | Implemented / partial live data |
| `GET /api/v1/ibkr/fx-triangular` | `X-API-Key` | IBKR FX table | Implemented / partial live data |
| `GET /api/v1/ibkr/crypto-futures-basis` | `X-API-Key` | IBKR basis table | Implemented / partial live data |
| `GET /metrics` | Public | Documented only | Conditional on config |

## Missing Or Partial API Support

- `GET /api/v1/metrics/snapshot` is not implemented. `internal/api/server_test.go` explicitly asserts that this endpoint returns `404`.
- IBKR live TWS Gateway market data is partial/planned. The UI displays configured instruments, health, and strategy results when the backend returns them, and otherwise shows empty states.
- OKX, Bybit, Coinbase, Gate.io, and Bitget are spot public-REST adapters in this backend version. Futures/funding/WebSocket support is documented as not implemented for those adapters.
- `GET /metrics` is public but only registered when `metrics.prometheus_enabled=true`.
- `/api/v1/exchanges/health` and `/api/v1/providers/health` currently overlap because both read the same store health map.

## Discrepancies Found Before Coding

- `swagger.yml` exists at the repository root, while `docs/api-design.md` and `docs/limitations.md` still say that no OpenAPI spec is generated. The practical interpretation is that a checked-in Swagger/OpenAPI file exists, but it is not generated automatically from Go code.
- The task’s expected endpoint list included `GET /api/v1/metrics/snapshot`; the implementation and tests show it has been removed.
- The OpenAPI `Decimal` schema allows string or number. The Go backend uses `shopspring/decimal.Decimal`; the default JSON behavior returns precision-safe strings. The UI does not coerce API data in state.
- The OpenAPI health status enum omits an `unknown` value. The backend health models use the documented statuses, while the TUI locally falls back to `unknown` when no IBKR health payload exists. The web UI treats missing IBKR health as “Not reported” instead of inventing an API status.

## Error Handling

- `401` responses are shown as API authentication failures.
- fetch/network failures are shown as backend unavailable.
- RTK Query keeps cached successful data when refetches fail, so the UI can show stale or partial data clearly.
- Browser preflight `OPTIONS` requests are supported by the backend CORS middleware. Loopback web UI origins are allowed automatically; deployed origins should be listed in `api.cors_allowed_origins`.
