# TUI Design

The terminal UI is implemented with Bubble Tea and Lip Gloss in `internal/tui`.

## Architecture

Important files:

- `client.go`: backend API client
- `model.go`: Bubble Tea model, update loop, commands, key handling
- `render.go`: Lip Gloss rendering, tabs, tables, detail panels
- `icons.go`: emoji/ASCII icon layer

The TUI fetches `/api/v1/snapshot` from the backend. It does not connect directly to exchanges or IBKR.

## Bubble Tea Model / Update / View

`Model` stores:

- API client
- refresh interval
- latest `marketdata.Snapshot`
- terminal dimensions
- pause/profitable-only/help/detail state
- selected row
- current tab/view
- icon set
- tab titles

Main message types:

- `snapshotMsg`
- `tickMsg`
- Bubble Tea window/key messages

Flow:

1. `Init` fetches a snapshot and schedules the next tick.
2. `Update` handles keys, snapshot responses, and ticks.
3. `View` renders the active tab and optional overlays/detail panels.

## Backend API Client

`Client.Snapshot` sends:

```text
GET /api/v1/snapshot
X-API-Key: <API_KEY>
```

The TUI relies on backend aggregation and calculation. It does not duplicate arbitrage logic.

## Dashboard Grid Layout

The main crypto dashboard renders:

- Header
- Tab bar
- Prices grouped by exchange
- Crypto triangular arbitrage
- Cross-exchange arbitrage
- Crypto spot-futures arbitrage
- Related asset signals
- Alerts
- Health
- Footer

When the terminal is wide enough, some sections render side-by-side.

## Tabs

Keyboard tabs:

```text
1  Crypto Dashboard
2  Crypto Triangular
3  Cross-Exchange
4  Crypto Spot-Futures
5  Signals
6  Alerts
7  Health
8  IBKR Monitor
```

Tab labels can be customized through `tui.tabs` in config.

## IBKR Tab

The IBKR tab renders:

- IBKR header/status
- Configured IBKR instruments
- IBKR FX triangular arbitrage
- Crypto spot vs IBKR futures basis
- IBKR health

IBKR trading status is shown as disabled/unsupported. No execution controls exist.

## Detail Panels

Rows are selectable with up/down. `Enter` opens a detail panel. `Esc` closes it.

Details include:

- Triangular per-leg simulations
- Cross-exchange buy/sell averages, fees, slippage
- Spot-futures basis/funding/net estimate
- IBKR instruments
- IBKR futures basis

## Keyboard Controls

```text
q or Ctrl+C  quit
r            manual refresh
p            pause/resume auto-refresh
f            toggle profitable-only filter
↑/↓          select row
Enter        open detail panel
Esc          close detail panel
?            help

1            Crypto Dashboard
2            Crypto Triangular
3            Cross-Exchange
4            Crypto Spot-Futures
5            Signals
6            Alerts
7            Health
8            IBKR Monitor
```

## Emoji / Icon System

`IconSet` in `internal/tui/icons.go` provides two sets:

- Emoji icons when `tui.use_emoji=true`
- ASCII-safe labels when `tui.use_emoji=false`

Icons are visual markers only. Text labels such as `OK`, `WARN`, `ERROR`, `WATCH`, `NO`, and `PARTIAL` remain visible.

## Responsive Behavior

Rendering uses:

- Current terminal width from `tea.WindowSizeMsg`
- Truncation through `lipgloss.Width`
- Side-by-side cards only on wider terminals
- Minimum card widths for narrow terminals

## Stale Data and Alerts

Stale prices are highlighted with warning style. Health tables show score and WebSocket/fallback status. Alerts show severity coloring and repeat counts.

## ASCII Mockup

```text
┌ go-crypto-arb ──────────────────────────────────────────────┐
│ Backend: OK | API: http://localhost:8080 | Last refresh: ... │
└──────────────────────────────────────────────────────────────┘
┌ Tabs ────────────────────────────────────────────────────────┐
│ [1 Crypto Dashboard]  2 Crypto Triangular  ...  8 IBKR       │
└──────────────────────────────────────────────────────────────┘
┌ Prices: Binance ─────────────────────────────────────────────┐
│ Symbol              Bid          Ask    Age                  │
│ BTC/USDT       67210.20     67212.10   2s                   │
└──────────────────────────────────────────────────────────────┘
┌ Cross-Exchange ──────────────────────────────────────────────┐
│ > BTC/USDT  Kraken  67200.00  Binance  67240.00  +0.020%    │
└──────────────────────────────────────────────────────────────┘
```

IBKR tab:

```text
┌ IBKR Monitor ────────────────────────────────────────────────┐
│ Status: ERROR | Market Data Only | Trading: DISABLED | Age n/a│
└──────────────────────────────────────────────────────────────┘
┌ IBKR Instruments ────────────────────────────────────────────┐
│ Title                     Type      Symbol      Bid Ask Age  │
│ CME Micro Bitcoin Future  FUTURES   MBT         n/a n/a n/a  │
└──────────────────────────────────────────────────────────────┘
```

## TUI Design Gaps

- No real screenshot fixtures are checked in.
- No virtualization for very large tables.
- TUI uses polling rather than server-push streaming.
- IBKR live data display depends on future IBKR market-data transport.
