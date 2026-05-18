import { describe, expect, it } from 'vitest';

import type { PriceRow } from './filters';
import { buildPriceHighlightStats, priceHighlightClass } from './priceHighlights';

function priceRow(overrides: Partial<PriceRow>): PriceRow {
  return {
    providerLabel: 'binance',
    status: 'ok',
    provider: 'binance',
    exchange: 'Binance',
    symbol: 'BTC/USDT',
    base_asset: 'BTC',
    quote_asset: 'USDT',
    market_type: 'spot',
    asset_class: 'crypto',
    bid: '100',
    ask: '101',
    last: '100.5',
    updated_at: new Date().toISOString(),
    ...overrides,
  };
}

describe('price highlight stats', () => {
  it('matches TUI color logic for bid and ask extrema', () => {
    const rows = [
      priceRow({ providerLabel: 'binance', bid: '100', ask: '102' }),
      priceRow({ providerLabel: 'kraken', exchange: 'Kraken', bid: '99', ask: '103' }),
    ];
    const highlights = buildPriceHighlightStats(rows);

    expect(highlights.bidHighlight(rows[0])).toBe('best');
    expect(highlights.bidHighlight(rows[1])).toBe('worst');
    expect(highlights.askHighlight(rows[0])).toBe('best');
    expect(highlights.askHighlight(rows[1])).toBe('worst');
  });

  it('does not highlight tied prices', () => {
    const rows = [
      priceRow({ providerLabel: 'binance', bid: '100', ask: '101' }),
      priceRow({ providerLabel: 'kraken', exchange: 'Kraken', bid: '100', ask: '101' }),
    ];
    const highlights = buildPriceHighlightStats(rows);

    expect(highlights.bidHighlight(rows[0])).toBe('none');
    expect(highlights.askHighlight(rows[1])).toBe('none');
  });

  it('keeps spot and futures in separate comparison buckets', () => {
    const spot = priceRow({ market_type: 'spot', bid: '100', ask: '101' });
    const futures = priceRow({ market_type: 'futures', bid: '110', ask: '111' });
    const highlights = buildPriceHighlightStats([spot, futures]);

    expect(highlights.bidHighlight(spot)).toBe('none');
    expect(highlights.bidHighlight(futures)).toBe('none');
  });

  it('maps highlights to CSS classes with stale fallback', () => {
    expect(priceHighlightClass('best', 'priceStale')).toBe('priceBest');
    expect(priceHighlightClass('worst', 'priceStale')).toBe('priceWorst');
    expect(priceHighlightClass('none', 'priceStale')).toBe('priceStale');
  });
});
