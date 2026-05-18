import { useMemo } from 'react';

import { useGetSpotFuturesArbitrageQuery } from '../api/arbApi';
import type { SpotFuturesOpportunity } from '../api/types';
import { DataTable, type Column } from '../components/DataTable';
import { PageHeader } from '../components/PageHeader';
import { StatusBadge } from '../components/StatusBadge';
import { useAppSelector } from '../hooks/redux';
import { usePollingInterval } from '../hooks/useRefresh';
import { formatDecimal, formatPercent, profitClass } from '../utils/decimal';
import { filterProfitableOnly } from '../utils/filters';

const columns: Column<SpotFuturesOpportunity>[] = [
  { id: 'provider', header: 'Provider', cell: (row) => row.provider || row.exchange },
  { id: 'symbol', header: 'Symbol', cell: (row) => row.symbol },
  { id: 'tradeSize', header: 'Trade size', cell: (row) => formatDecimal(row.trade_size), className: 'numeric' },
  { id: 'spotAvg', header: 'Spot average buy price', cell: (row) => formatDecimal(row.spot_average_buy_price), className: 'numeric' },
  { id: 'futuresAvg', header: 'Futures average sell price', cell: (row) => formatDecimal(row.futures_average_sell_price), className: 'numeric' },
  { id: 'basis', header: 'Basis %', cell: (row) => formatPercent(row.basis_percent), className: 'numeric' },
  { id: 'funding', header: 'Funding rate', cell: (row) => formatDecimal(row.funding_rate, 6), className: 'numeric' },
  {
    id: 'net',
    header: 'Net estimate %',
    cell: (row) => (
      <span className={profitClass(row.net_estimate_percent)}>
        {formatPercent(row.net_estimate_percent)}
      </span>
    ),
    className: 'numeric',
  },
  {
    id: 'fill',
    header: 'Complete fill',
    cell: (row) => <StatusBadge label={row.complete_fill ? 'Yes' : 'Partial fill'} tone={row.complete_fill ? 'ok' : 'warning'} />,
  },
];

export function SpotFuturesPage() {
  const pollingInterval = usePollingInterval();
  const profitableOnly = useAppSelector((state) => state.settings.profitableOnly);
  const { data = [] } = useGetSpotFuturesArbitrageQuery(undefined, { pollingInterval });
  const rows = useMemo(() => filterProfitableOnly(data, profitableOnly), [data, profitableOnly]);

  return (
    <>
      <PageHeader
        title="Crypto Spot-Futures Arbitrage"
        subtitle="Basis monitoring using backend spot, futures, funding, fee, and order book estimates."
      />
      <section className="panel">
        <DataTable
          columns={columns}
          rows={rows}
          emptyText="No spot-futures basis results are currently available."
          getRowKey={(row, index) => `${row.provider}-${row.symbol}-${index}`}
        />
      </section>
    </>
  );
}
