import { useMemo } from 'react';

import { useGetPricesQuery } from '../api/arbApi';
import type { Ticker } from '../api/types';
import { PageHeader } from '../components/PageHeader';
import { PriceTable } from '../components/PriceTable';
import {
  setPriceMarketType,
  setPriceProvider,
  setPriceStaleOnly,
  setPriceSymbol,
} from '../features/filtersSlice';
import { useAppDispatch, useAppSelector } from '../hooks/redux';
import { usePollingInterval } from '../hooks/useRefresh';
import { applyPriceFilters, flattenPrices } from '../utils/filters';

function mergePriceMaps(
  spot: Record<string, Ticker[]> | undefined,
  futures: Record<string, Ticker[]> | undefined,
): Record<string, Ticker[]> {
  const out: Record<string, Ticker[]> = {};
  for (const [provider, rows] of Object.entries(spot ?? {})) {
    out[provider] = [...rows];
  }
  for (const [provider, rows] of Object.entries(futures ?? {})) {
    out[provider] = [...(out[provider] ?? []), ...rows];
  }
  return out;
}

export function PricesPage() {
  const dispatch = useAppDispatch();
  const filters = useAppSelector((state) => state.filters.prices);
  const pollingInterval = usePollingInterval();
  const { data } = useGetPricesQuery(undefined, { pollingInterval });

  const allRows = useMemo(
    () => flattenPrices(mergePriceMaps(data?.prices, data?.futures_prices)),
    [data],
  );
  const rows = useMemo(() => applyPriceFilters(allRows, filters), [allRows, filters]);
  const providers = [...new Set(allRows.map((row) => row.providerLabel))].sort();

  return (
    <>
      <PageHeader title="Prices" subtitle="Spot and futures tickers from the backend API." />
      <section className="filters">
        <label>
          Provider
          <select
            value={filters.provider}
            onChange={(event) => dispatch(setPriceProvider(event.target.value))}
          >
            <option value="">All</option>
            {providers.map((provider) => (
              <option key={provider} value={provider}>
                {provider}
              </option>
            ))}
          </select>
        </label>
        <label>
          Market type
          <select
            value={filters.marketType}
            onChange={(event) => dispatch(setPriceMarketType(event.target.value))}
          >
            <option value="">All</option>
            <option value="spot">Spot</option>
            <option value="futures">Futures</option>
          </select>
        </label>
        <label>
          Symbol
          <input
            value={filters.symbol}
            onChange={(event) => dispatch(setPriceSymbol(event.target.value))}
            placeholder="BTC/USDT"
          />
        </label>
        <label className="checkLabel">
          <input
            type="checkbox"
            checked={filters.staleOnly}
            onChange={(event) => dispatch(setPriceStaleOnly(event.target.checked))}
          />
          Stale only
        </label>
      </section>
      <section className="panel">
        <PriceTable rows={rows} />
      </section>
    </>
  );
}
