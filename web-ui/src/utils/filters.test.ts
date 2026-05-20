import { describe, expect, it } from 'vitest';

import type { CrossExchangeOpportunity, Ticker } from '../api/types';
import {
  applyPriceFilters,
  filterProfitableOnly,
  flattenPrices,
  sortCrossExchangeByPotentialProfit,
} from './filters';

const freshTime = new Date().toISOString();
const staleTime = new Date(Date.now() - 60_000).toISOString();

function ticker(overrides: Partial<Ticker>): Ticker {
  return {
    exchange: 'Binance',
    symbol: 'BTC/USDT',
    base_asset: 'BTC',
    quote_asset: 'USDT',
    market_type: 'spot',
    asset_class: 'crypto',
    bid: '100',
    ask: '101',
    last: '100.5',
    updated_at: freshTime,
    ...overrides,
  };
}

function crossOpportunity(overrides: Partial<CrossExchangeOpportunity>): CrossExchangeOpportunity {
  return {
    strategy_title: 'Cross',
    symbol: 'BTC/USDT',
    buy_provider: 'binance',
    sell_provider: 'kraken',
    buy_exchange: 'Binance',
    sell_exchange: 'Kraken',
    trade_size: '1000',
    buy_average_price: '100',
    sell_average_price: '102',
    buy_slippage_percent: '0',
    sell_slippage_percent: '0',
    buy_fee_amount: '1',
    sell_fee_amount: '1',
    net_profit_percent: '0.2',
    complete_fill: true,
    status: 'watch',
    updated_at: freshTime,
    ...overrides,
  };
}

describe('price filters', () => {
  it('flattens prices and marks stale rows', () => {
    const rows = flattenPrices({
      Binance: [
        ticker({ provider: 'binance' }),
        ticker({ symbol: 'ETH/USDT', updated_at: staleTime }),
      ],
    });

    expect(rows).toHaveLength(2);
    expect(rows.find((row) => row.symbol === 'ETH/USDT')?.status).toBe('stale');
  });

  it('applies provider, market, symbol, and stale-only filters', () => {
    const rows = flattenPrices({
      Binance: [ticker({ provider: 'binance' })],
      Kraken: [ticker({ exchange: 'Kraken', symbol: 'ETH/USDT', updated_at: staleTime })],
    });

    const filtered = applyPriceFilters(rows, {
      provider: 'Kraken',
      marketType: 'spot',
      symbol: 'ETH',
      staleOnly: true,
    });

    expect(filtered.map((row) => row.symbol)).toEqual(['ETH/USDT']);
  });
});

describe('profitable-only filtering', () => {
  it('keeps only positive opportunity estimates', () => {
    const opportunities: CrossExchangeOpportunity[] = [
      {
        strategy_title: 'Cross',
        symbol: 'BTC/USDT',
        buy_provider: 'binance',
        sell_provider: 'kraken',
        buy_exchange: 'Binance',
        sell_exchange: 'Kraken',
        trade_size: '1000',
        buy_average_price: '100',
        sell_average_price: '102',
        buy_slippage_percent: '0',
        sell_slippage_percent: '0',
        buy_fee_amount: '1',
        sell_fee_amount: '1',
        net_profit_percent: '0.2',
        complete_fill: true,
        status: 'watch',
        updated_at: freshTime,
      },
      {
        strategy_title: 'Cross',
        symbol: 'ETH/USDT',
        buy_provider: 'binance',
        sell_provider: 'kraken',
        buy_exchange: 'Binance',
        sell_exchange: 'Kraken',
        trade_size: '1000',
        buy_average_price: '100',
        sell_average_price: '99',
        buy_slippage_percent: '0',
        sell_slippage_percent: '0',
        buy_fee_amount: '1',
        sell_fee_amount: '1',
        net_profit_percent: '-0.2',
        complete_fill: true,
        status: 'no',
        updated_at: freshTime,
      },
    ];

    expect(filterProfitableOnly(opportunities, true).map((item) => item.symbol)).toEqual([
      'BTC/USDT',
    ]);
  });
});

describe('cross-exchange sorting', () => {
  it('puts the highest potential profit first', () => {
    const opportunities: CrossExchangeOpportunity[] = [
      crossOpportunity({
        symbol: 'ETH/USDT',
        sell_average_price: '103',
        net_profit_percent: '0.25',
      }),
      crossOpportunity({
        symbol: 'BTC/USDT',
        sell_average_price: '105',
        net_profit_percent: '1.25',
      }),
      crossOpportunity({
        symbol: 'SOL/USDT',
        sell_average_price: '104',
        net_profit_percent: '0.75',
      }),
    ];

    expect(sortCrossExchangeByPotentialProfit(opportunities).map((item) => item.symbol)).toEqual([
      'BTC/USDT',
      'SOL/USDT',
      'ETH/USDT',
    ]);
  });
});
