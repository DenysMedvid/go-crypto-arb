import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { PriceRow } from '../utils/filters';
import { PriceTable } from './PriceTable';

describe('PriceTable', () => {
  it('shows stale data status without hiding the last known values', () => {
    const row: PriceRow = {
      providerLabel: 'binance',
      status: 'stale',
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
      updated_at: new Date(Date.now() - 60_000).toISOString(),
    };

    render(<PriceTable rows={[row]} />);

    expect(screen.getByText('BTC/USDT')).toBeInTheDocument();
    expect(screen.getByText('Stale data')).toBeInTheDocument();
    expect(screen.getByText('100.00')).toBeInTheDocument();
    expect(screen.getByText('100.00')).toHaveClass('priceStale');
  });

  it('colors best and worst bid/ask values like the TUI', () => {
    const rows: PriceRow[] = [
      {
        providerLabel: 'binance',
        status: 'ok',
        provider: 'binance',
        exchange: 'Binance',
        symbol: 'BTC/USDT',
        base_asset: 'BTC',
        quote_asset: 'USDT',
        market_type: 'spot',
        asset_class: 'crypto',
        bid: '101',
        ask: '102',
        last: '101.5',
        updated_at: new Date().toISOString(),
      },
      {
        providerLabel: 'kraken',
        status: 'ok',
        provider: 'kraken',
        exchange: 'Kraken',
        symbol: 'BTC/USDT',
        base_asset: 'BTC',
        quote_asset: 'USDT',
        market_type: 'spot',
        asset_class: 'crypto',
        bid: '100',
        ask: '103',
        last: '101.5',
        updated_at: new Date().toISOString(),
      },
    ];

    render(<PriceTable rows={rows} />);

    expect(screen.getByText('101.00')).toHaveClass('priceBest');
    expect(screen.getByText('100.00')).toHaveClass('priceWorst');
    expect(screen.getByText('102.00')).toHaveClass('priceBest');
    expect(screen.getByText('103.00')).toHaveClass('priceWorst');
  });
});
