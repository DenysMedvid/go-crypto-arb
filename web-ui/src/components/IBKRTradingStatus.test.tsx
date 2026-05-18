import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { IBKRTradingStatus } from './IBKRTradingStatus';

describe('IBKRTradingStatus', () => {
  it('clearly displays Trading: DISABLED when backend reports trading disabled', () => {
    render(
      <IBKRTradingStatus
        health={{
          provider: 'ibkr',
          provider_type: 'broker',
          exchange: 'IBKR',
          broker: 'IBKR',
          enabled: true,
          spot_enabled: false,
          futures_enabled: true,
          market_data_enabled: true,
          trading_enabled: false,
          websocket_enabled: false,
          websocket_connected: false,
          gateway_connected: false,
          market_data_ok: false,
          last_message_time: new Date().toISOString(),
          last_message_at: new Date().toISOString(),
          rest_fallback_active: false,
          reconnect_count: 0,
          data_fresh: false,
          stale_ticker_count: 0,
          stale_order_book_count: 0,
          partial_support: true,
          score: 40,
          status: 'disconnected',
        }}
      />,
    );

    expect(screen.getByText('Trading')).toBeInTheDocument();
    expect(screen.getByText('DISABLED')).toBeInTheDocument();
  });
});
