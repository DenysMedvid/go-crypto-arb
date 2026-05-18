import type { Decimal } from '../api/types';
import type { PriceRow } from './filters';
import { decimalToNumber } from './decimal';

export type PriceHighlight = 'none' | 'best' | 'worst';
export type PriceSide = 'bid' | 'ask';

interface PriceExtrema {
  best: number;
  worst: number;
  hasValue: boolean;
  hasSpread: boolean;
}

export interface PriceHighlightStats {
  bidHighlight: (row: PriceRow) => PriceHighlight;
  askHighlight: (row: PriceRow) => PriceHighlight;
}

export function buildPriceHighlightStats(rows: PriceRow[]): PriceHighlightStats {
  const bids = new Map<string, PriceExtrema>();
  const asks = new Map<string, PriceExtrema>();

  for (const row of rows) {
    const key = priceBucketKey(row);
    updatePriceExtrema(bids, key, row.bid, (left, right) => left > right, (left, right) => left < right);
    updatePriceExtrema(asks, key, row.ask, (left, right) => left < right, (left, right) => left > right);
  }

  finalizePriceExtrema(bids);
  finalizePriceExtrema(asks);

  return {
    bidHighlight: (row) => priceHighlightFor(bids, priceBucketKey(row), row.bid),
    askHighlight: (row) => priceHighlightFor(asks, priceBucketKey(row), row.ask),
  };
}

export function priceHighlightClass(
  highlight: PriceHighlight,
  fallback: 'priceStale' | '' = '',
): string {
  if (highlight === 'best') {
    return 'priceBest';
  }
  if (highlight === 'worst') {
    return 'priceWorst';
  }
  return fallback;
}

export function priceBucketKey(row: Pick<PriceRow, 'market_type' | 'symbol'>): string {
  return `${row.market_type}|${normalizeSymbol(row.symbol)}`;
}

function updatePriceExtrema(
  extrema: Map<string, PriceExtrema>,
  key: string,
  value: Decimal | undefined,
  better: (left: number, right: number) => boolean,
  worse: (left: number, right: number) => boolean,
): void {
  const parsed = decimalToNumber(value);
  if (!key || parsed <= 0) {
    return;
  }
  const current = extrema.get(key);
  if (!current) {
    extrema.set(key, { best: parsed, worst: parsed, hasValue: true, hasSpread: false });
    return;
  }
  if (better(parsed, current.best)) {
    current.best = parsed;
  }
  if (worse(parsed, current.worst)) {
    current.worst = parsed;
  }
}

function finalizePriceExtrema(extrema: Map<string, PriceExtrema>): void {
  for (const current of extrema.values()) {
    current.hasSpread = current.hasValue && current.best !== current.worst;
  }
}

function priceHighlightFor(
  extrema: Map<string, PriceExtrema>,
  key: string,
  value: Decimal | undefined,
): PriceHighlight {
  const parsed = decimalToNumber(value);
  if (parsed <= 0) {
    return 'none';
  }
  const current = extrema.get(key);
  if (!current?.hasSpread) {
    return 'none';
  }
  if (parsed === current.best) {
    return 'best';
  }
  if (parsed === current.worst) {
    return 'worst';
  }
  return 'none';
}

function normalizeSymbol(symbol: string): string {
  return symbol.trim().toUpperCase().replace(/[-_]/g, '/');
}
