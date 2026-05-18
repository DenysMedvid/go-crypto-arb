# Arbitrage Engine

The arbitrage engine lives in `internal/arbitrage`. It consumes normalized market data and produces estimate-based opportunities. It does not execute trades.

## Decimal Arithmetic

All financial calculations use `shopspring/decimal.Decimal`. This avoids binary floating-point rounding errors in prices, quantities, fees, and percentages.

## Tickers vs Order Books

Tickers contain best bid/ask/last values. Order books contain depth:

- Bids sorted descending by price
- Asks sorted ascending by price
- Level price and quantity

When order book depth is available, v2 calculators use depth simulation. When depth is unavailable, calculators may synthesize level-1 books from tickers and mark `LimitedDepth`.

## Best Bid/Ask Limitation

Best bid/ask can be too optimistic:

- The top level may not have enough quantity.
- The book may move before execution.
- Some providers expose only top-of-book.

Order book simulation reduces false positives but cannot eliminate them.

## Execution Simulation

Implemented in `internal/arbitrage/simulation.go`.

Functions:

- `SimulateBuyWithQuote(orderBook, quoteAmount, feePercent)`
- `SimulateSellBase(orderBook, baseAmount, feePercent)`

Rules:

- Buy consumes asks from lowest ask upward.
- Sell consumes bids from highest bid downward.
- Average execution price is calculated from consumed levels.
- Slippage is calculated relative to best visible price.
- Fees are applied to the filled amount/value.
- Incomplete liquidity sets `CompleteFill=false`.
- Empty books return a clear status/error.

## Trade Direction Rules

For a pair `BASE/QUOTE`:

- Converting `QUOTE -> BASE` means buying base and consumes asks.
- Converting `BASE -> QUOTE` means selling base and consumes bids.
- Fees are applied per leg.
- Partial fills reduce reliability and mark opportunities incomplete.

## Fees

Fee helpers live in `internal/arbitrage/fee.go` and config fee lookup helpers in calculators.

Fee sources:

- `ProviderConfig.Fees.SpotTaker`
- `ProviderConfig.Fees.FuturesTaker`
- `ProviderConfig.Fees.FXEstimatedTaker`
- `ProviderConfig.Fees.FuturesEstimatedTaker`
- Backward-compatible exchange fee config

## Strategy Outputs

Important output types in `internal/arbitrage/model.go`:

- `TriangularOpportunityV2`
- `CrossExchangeOpportunityV2`
- `SpotFuturesOpportunityV2`
- `BrokerFuturesBasisOpportunity`
- `RelatedAssetGroupSignal`

## Crypto Triangular Arbitrage

Function:

- `CalculateTriangularV2`

Inputs:

- Configured stable bases and preferred assets
- Spot tickers
- Spot order books
- Spot taker fees
- Trade sizes

Method:

1. Build cycles such as `USDT -> BTC -> ETH -> USDT`.
2. Simulate each leg using order books when enabled.
3. Apply fee per leg.
4. Track worst leg and maximum slippage.
5. Sort by net profit descending.

Output:

- Provider/exchange
- Cycle
- Start/end amount
- Net profit percent
- Complete fill
- Worst leg
- Max slippage
- Per-leg simulations

Limitations:

- Generated cycles depend on configured asset universe.
- Provider filters in target config are not enforced in every legacy path.
- Assumes simultaneous execution is possible, which is not guaranteed.

False positive risks:

- Stale order books
- Partial books
- Fast-moving pairs
- Exchange order constraints

## Cross-Exchange Arbitrage

Function:

- `CalculateCrossExchangeV2`

Inputs:

- Spot tickers/order books across exchanges
- Trade sizes
- Spot fees

Method:

1. Buy on provider with ask-side liquidity.
2. Sell on provider with bid-side liquidity.
3. Apply fees on both sides.
4. Calculate net profit percent.
5. Mark incomplete if either side lacks liquidity.

Limitations:

- Does not model withdrawals, deposits, or transfer time.
- Does not require balances on both venues.

False positive risks:

- Transfer latency
- Withdrawal/deposit fees
- Venue downtime
- Different quote assets or settlement rules

## Crypto Spot-Futures Arbitrage

Function:

- `CalculateSpotFuturesV2`

Inputs:

- Spot tickers/order books
- Futures tickers/order books
- Funding rates
- Spot/futures fees

Method:

1. Simulate buying spot with configured quote size.
2. Simulate selling futures with filled base quantity.
3. Calculate basis percent.
4. Include funding rate when configured.
5. Mark incomplete on partial fills.

Limitations:

- Does not model contract sizing, margin, liquidation, or expiry.
- Funding is an estimate and can change.

## IBKR FX Triangular Arbitrage

Function:

- `CalculateIBKRFXTriangular`

Inputs:

- IBKR FX tickers/order books
- Configured cycles
- Estimated FX fees
- Trade sizes

Method:

1. Use only provider `ibkr`.
2. Simulate configured FX cycles.
3. Apply estimated fees.
4. Return separate `TriangularOpportunityV2` results with `AssetClass: "fx"`.

Limitations:

- Depends on live IBKR market data, which is partial/planned.
- FX order book depth availability depends on subscription and API support.

## Crypto Spot vs IBKR Futures Basis

Function:

- `CalculateBrokerFuturesBasis`

Inputs:

- Crypto spot books from configured crypto exchange providers
- IBKR futures books
- Configured instrument mappings
- Spot and estimated futures fees

Method:

1. Simulate buying crypto spot from configured spot providers.
2. Simulate selling broker futures by configured futures instrument ID.
3. Calculate basis and net estimate.
4. Label as basis monitoring.

Limitations:

- Not guaranteed arbitrage.
- Does not model contract multiplier, expiry basis, margin, or hedge ratio.
- IBKR futures market data is partial/planned.

## Related Asset Signals

Implemented by `SignalEngine` in `internal/arbitrage/signals.go`.

Inputs:

- Spot tickers
- Related asset groups from config

Method:

- Tracks previous prices.
- Calculates change percent and divergence within configured groups.

Limitations:

- This is a signal, not arbitrage.
- Uses in-memory history only.

## Engine Gaps

- No balance or position awareness.
- No latency modeling.
- No exchange-specific min order size/tick size validation.
- No withdrawal/deposit fee modeling.
- Some provider filters from v2.1 config are not yet uniformly applied across every strategy path.
