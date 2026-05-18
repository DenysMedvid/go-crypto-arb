# IBKR Design

IBKR support is intentionally modeled as broker market data, not crypto exchange trading.

## Scope

Current v2.1 scope:

- Broker provider skeleton
- Configured instrument metadata
- Health/status reporting
- IBKR TUI tab
- IBKR FX triangular strategy model
- Crypto spot vs IBKR futures basis strategy model

Not in scope:

- IBKR order placement
- IBKR crypto spot trading
- Account trading
- Portfolio execution
- Position management

## Package

IBKR code lives in:

```text
internal/broker/ibkr
```

Main type:

- `Client`

Key methods:

- `Start`
- `Stop`
- `Health`
- `GetLatestTickers`
- `GetLatestOrderBooks`
- `GetMarkets`
- `DiscoverMarkets`
- `GetAccountSummary`
- `GetPortfolio`

`GetAccountSummary` and `GetPortfolio` return explicit `not_implemented` results.

## Broker Provider, Not Exchange Provider

IBKR differs from crypto exchange adapters:

- Instruments are contracts, not simple exchange pairs.
- Contract metadata includes `symbol`, `sec_type`, `exchange`, `currency`, and optional `con_id`.
- IBKR market data may require subscriptions.
- Gateway/session health matters.
- FX, futures, stocks, and ETFs can share the same broker API.

## Trading Disabled

Config must keep:

```yaml
trading_enabled: false
```

If set to `true`, config validation emits a hard error:

```text
IBKR trading_enabled must remain false in v2.1
```

No order-placement code path exists in this version.

## TWS Gateway / IB Gateway

Config supports:

```yaml
api_mode: "tws_gateway"
host: "127.0.0.1"
port: 7497
client_id: 101
```

Current implementation note: live TWS Gateway market-data transport is partial/planned. The adapter currently exposes configured instruments and clear health status.

## Configured Instruments

IBKR instruments are configured in instrument universes:

```yaml
instrument_universes:
  ibkr_futures:
    providers: ["ibkr"]
    instruments:
      - id: CME_MICRO_BTC
        display_name: "CME Micro Bitcoin Future"
        asset_class: futures
        market_type: futures
        providers: ["ibkr"]
        ibkr:
          symbol: "MBT"
          sec_type: "FUT"
          exchange: "CME"
          currency: "USD"
          con_id: null
```

## Contract Metadata

Fields:

- `symbol`: IBKR symbol.
- `sec_type`: security type, such as `FUT` or `STK`.
- `exchange`: routing/exchange, such as `CME` or `SMART`.
- `currency`: instrument currency.
- `con_id`: optional contract identifier for future resolution/storage.

## Asset Classes

Supported by configuration:

- FX instruments
- Futures instruments
- ETF/stocks instruments

Crypto spot via IBKR is disabled by default and should remain opt-in only.

## IBKR TUI Tab

The IBKR tab shows:

- Broker status
- Market-data-only mode
- Trading disabled/unsupported status
- Configured instruments
- IBKR FX triangular results
- Crypto spot vs IBKR futures basis results
- IBKR health

## Supported IBKR Strategies

### IBKR FX Triangular Arbitrage

Uses configured FX cycles such as:

```text
USD -> EUR -> JPY -> USD
```

It is monitoring only. It does not place FX orders.

### Crypto Spot vs IBKR Futures Basis

Compares crypto spot from configured crypto exchange providers against configured IBKR futures instruments.

This is basis monitoring, not guaranteed arbitrage.

## Health

IBKR health includes:

- Provider type `broker`
- Gateway connection status
- Market data status
- Last message time
- Reconnect count
- Last error
- Health score/status

Current skeleton reports disconnected/partial status until live market-data transport is implemented.

## Explicit Safety Statements

- Crypto spot via IBKR is disabled unless explicitly enabled.
- IBKR futures basis is monitoring, not guaranteed arbitrage.
- No IBKR execution module exists in this version.
- No IBKR orders can be placed by current code.

## IBKR Limitations

- Live TWS Gateway transport is partial/planned.
- Contract lookup/conId resolution is not implemented.
- Market depth support depends on IBKR subscriptions and API support.
- Account/portfolio snapshots are not implemented.
- Futures contract sizing and expiry basis are not fully modeled.
