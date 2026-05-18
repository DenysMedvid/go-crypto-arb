# go-crypto-arb Documentation

This directory contains the technical documentation for `go-crypto-arb`: architecture, design decisions, runtime data flow, package responsibilities, extension points, and current limitations.

## Start Here

| Document | Purpose |
| --- | --- |
| [Architecture](architecture.md) | High-level system architecture, backend/UI split, provider model, and architecture gaps. |
| [Design](design.md) | Design decisions, tradeoffs, monitoring-only scope, decimal arithmetic, and depth simulation rationale. |
| [Modules](modules.md) | Package-by-package guide to the actual repository structure. |
| [Data Flow](data-flow.md) | Startup, provider ingestion, calculation, alerting, API, TUI, and web UI refresh flows. |

## Core System Areas

| Document | Purpose |
| --- | --- |
| [Provider Design](provider-design.md) | Market data provider architecture, exchange adapters, IBKR broker provider, and discovery/health behavior. |
| [Arbitrage Engine](arbitrage-engine.md) | Execution simulation, order book depth, fee/slippage rules, and strategy calculation details. |
| [API Design](api-design.md) | REST API authentication, endpoints, snapshot format, metrics, and endpoint status. |
| [API History and Grafana](api-history-and-grafana.md) | How to use API and Prometheus metrics to store history and render it in Grafana. |
| [Docker Deployment](docker-deployment.md) | How to build, configure, run, and monitor the API with Docker or Docker Compose. |
| [TUI Design](tui-design.md) | Bubble Tea model/update/view flow, dashboard layout, tabs, detail panels, and icon support. |
| [Web UI Design](web-ui-design.md) | React/Redux browser UI architecture, routes, API integration, state, refresh, and safety rules. |
| [Config Design](config-design.md) | `.env` vs `config.yaml`, providers, instruments, strategies, validation, and custom titles. |
| [Alerting and Health](alerting-and-health.md) | Alert deduplication/cooldown, severity, health scoring, stale data, and provider status. |
| [IBKR Design](ibkr-design.md) | IBKR market-data-only broker design, instrument config, safety guarantees, and limitations. |

## Maintenance and Evolution

| Document | Purpose |
| --- | --- |
| [Patterns](patterns.md) | Architectural/design patterns used in the codebase and their tradeoffs. |
| [Extension Guide](extension-guide.md) | How to add exchanges, brokers, strategies, TUI tabs, notifiers, metrics, and storage. |
| [Limitations](limitations.md) | Explicit current limitations and non-goals. |

## Diagrams

Mermaid diagram sources live in [diagrams](diagrams/).

| Diagram | Purpose |
| --- | --- |
| [System Context](diagrams/system-context.mmd) | External actors and systems around the backend and UI clients. |
| [Container Diagram](diagrams/container-diagram.mmd) | Runtime binaries, config files, state store, and external providers. |
| [Component Diagram](diagrams/component-diagram.mmd) | Backend components and internal responsibilities. |
| [Data Flow](diagrams/data-flow.mmd) | Market data movement through adapters, store, calculators, API, TUI, and web UI. |
| [Market Data Sequence](diagrams/sequence-market-data.mmd) | Provider update and store write sequence. |
| [Arbitrage Calculation Sequence](diagrams/sequence-arbitrage-calculation.mmd) | Calculation and alert generation sequence. |
| [TUI Refresh Sequence](diagrams/sequence-tui-refresh.mmd) | TUI polling and render sequence. |
| [Web UI Refresh Sequence](diagrams/sequence-web-ui-refresh.mmd) | Browser RTK Query polling, CORS, cache, and render sequence. |

## Current Implementation Notes

- The project is monitoring-only and does not implement real trading or order execution.
- OKX, Bybit, Binance, Kraken, Coinbase, Gate.io, and Bitget are supported crypto exchange platforms.
- OKX, Bybit, Coinbase, Gate.io, and Bitget are disabled by default and currently use spot public REST only.
- IBKR is currently a broker market-data skeleton with configured instruments and health/status reporting.
- Metrics are currently rendered from `internal/api`, not a dedicated `internal/metrics` package.
- State is latest-only and in memory.
- `web-ui/` is a separate React/TypeScript browser client for the same backend API.

## Suggested Reading Paths

For new maintainers:

1. [Architecture](architecture.md)
2. [Modules](modules.md)
3. [Data Flow](data-flow.md)
4. [Extension Guide](extension-guide.md)

For strategy work:

1. [Arbitrage Engine](arbitrage-engine.md)
2. [Provider Design](provider-design.md)
3. [Config Design](config-design.md)
4. [Alerting and Health](alerting-and-health.md)

For UI work:

1. [TUI Design](tui-design.md)
2. [Web UI Design](web-ui-design.md)
3. [API Design](api-design.md)
4. [Data Flow](data-flow.md)

For IBKR work:

1. [IBKR Design](ibkr-design.md)
2. [Provider Design](provider-design.md)
3. [Config Design](config-design.md)
4. [Limitations](limitations.md)
