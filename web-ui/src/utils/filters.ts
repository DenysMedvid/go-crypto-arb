import type {
  CrossExchangeOpportunity,
  Decimal,
  SpotFuturesOpportunity,
  Ticker,
  TriangularOpportunity,
} from '../api/types';
import { decimalToNumber } from './decimal';
import { priceStatus, providerLabel } from './status';

export interface PriceFilters {
  provider: string;
  marketType: string;
  symbol: string;
  staleOnly: boolean;
}

export interface PriceRow extends Ticker {
  providerLabel: string;
  status: 'ok' | 'stale';
}

export function flattenPrices(prices: Record<string, Ticker[]> | undefined): PriceRow[] {
  if (!prices) {
    return [];
  }
  return Object.values(prices)
    .flat()
    .map((ticker) => ({
      ...ticker,
      providerLabel: providerLabel(ticker),
      status: priceStatus(ticker),
    }))
    .sort((left, right) => {
      const providerCompare = left.providerLabel.localeCompare(right.providerLabel);
      return providerCompare === 0 ? left.symbol.localeCompare(right.symbol) : providerCompare;
    });
}

export function applyPriceFilters(rows: PriceRow[], filters: PriceFilters): PriceRow[] {
  const symbolSearch = filters.symbol.trim().toLowerCase();
  return rows.filter((row) => {
    if (filters.provider && row.providerLabel !== filters.provider) {
      return false;
    }
    if (filters.marketType && row.market_type !== filters.marketType) {
      return false;
    }
    if (symbolSearch && !row.symbol.toLowerCase().includes(symbolSearch)) {
      return false;
    }
    if (filters.staleOnly && row.status !== 'stale') {
      return false;
    }
    return true;
  });
}

export type AnyOpportunity =
  | TriangularOpportunity
  | CrossExchangeOpportunity
  | SpotFuturesOpportunity
  | { net_estimate_percent: Decimal; basis_percent?: Decimal };

export function opportunityProfit(value: AnyOpportunity): number {
  if ('net_profit_percent' in value) {
    return decimalToNumber(value.net_profit_percent);
  }
  return decimalToNumber(value.net_estimate_percent);
}

export function filterProfitableOnly<T extends AnyOpportunity>(items: T[], enabled: boolean): T[] {
  if (!enabled) {
    return items;
  }
  return items.filter((item) => opportunityProfit(item) > 0);
}

export function sortCrossExchangeByPotentialProfit(
  items: CrossExchangeOpportunity[],
): CrossExchangeOpportunity[] {
  return [...items].sort((left, right) => {
    const profitDifference = opportunityProfit(right) - opportunityProfit(left);
    if (profitDifference !== 0) {
      return profitDifference;
    }

    const leftSpread =
      decimalToNumber(left.sell_average_price) - decimalToNumber(left.buy_average_price);
    const rightSpread =
      decimalToNumber(right.sell_average_price) - decimalToNumber(right.buy_average_price);
    if (rightSpread !== leftSpread) {
      return rightSpread - leftSpread;
    }

    return left.symbol.localeCompare(right.symbol);
  });
}
