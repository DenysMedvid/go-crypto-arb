import type { ExchangeHealth, Ticker } from '../api/types';
import { isStale } from './time';

export function providerLabel(item: {
  provider?: string;
  exchange?: string;
  broker?: string;
}): string {
  return item.provider || item.exchange || item.broker || 'unknown';
}

export function priceStatus(ticker: Ticker): 'ok' | 'stale' {
  return isStale(ticker.updated_at) ? 'stale' : 'ok';
}

export function healthTone(health: ExchangeHealth | undefined): 'ok' | 'warning' | 'error' | 'muted' {
  if (!health) {
    return 'muted';
  }
  if (health.status === 'ok') {
    return 'ok';
  }
  if (health.status === 'disconnected') {
    return 'error';
  }
  return 'warning';
}

export function boolLabel(value: boolean | undefined): string {
  return value ? 'Yes' : 'No';
}
