import { useMemo } from 'react';

import { useGetCrossExchangeArbitrageQuery } from '../api/arbApi';
import type { CrossExchangeOpportunity } from '../api/types';
import { DataTable, type Column } from '../components/DataTable';
import { PageHeader } from '../components/PageHeader';
import { StatusBadge } from '../components/StatusBadge';
import { useAppSelector } from '../hooks/redux';
import { usePollingInterval } from '../hooks/useRefresh';
import { formatDecimal, formatPercent, profitClass } from '../utils/decimal';
import { filterProfitableOnly } from '../utils/filters';

const columns: Column<CrossExchangeOpportunity>[] = [
  { id: 'symbol', header: 'Symbol', cell: (row) => row.symbol },
  { id: 'buyProvider', header: 'Buy provider', cell: (row) => row.buy_provider },
  { id: 'sellProvider', header: 'Sell provider', cell: (row) => row.sell_provider },
  { id: 'tradeSize', header: 'Trade size', cell: (row) => formatDecimal(row.trade_size), className: 'numeric' },
  { id: 'buyAvg', header: 'Buy average price', cell: (row) => formatDecimal(row.buy_average_price), className: 'numeric' },
  { id: 'sellAvg', header: 'Sell average price', cell: (row) => formatDecimal(row.sell_average_price), className: 'numeric' },
  {
    id: 'profit',
    header: 'Net profit %',
    cell: (row) => (
      <span className={profitClass(row.net_profit_percent)}>
        {formatPercent(row.net_profit_percent)}
      </span>
    ),
    className: 'numeric',
  },
  {
    id: 'fill',
    header: 'Complete fill',
    cell: (row) => <StatusBadge label={row.complete_fill ? 'Yes' : 'Partial fill'} tone={row.complete_fill ? 'ok' : 'warning'} />,
  },
  { id: 'status', header: 'Status', cell: (row) => row.status },
];

export function CrossExchangePage() {
  const pollingInterval = usePollingInterval();
  const profitableOnly = useAppSelector((state) => state.settings.profitableOnly);
  const { data = [] } = useGetCrossExchangeArbitrageQuery(undefined, { pollingInterval });
  const rows = useMemo(() => filterProfitableOnly(data, profitableOnly), [data, profitableOnly]);

  return (
    <>
      <PageHeader
        title="Cross-Exchange Arbitrage"
        subtitle="Estimated venue-to-venue opportunities. Transfer costs and latency are not modeled."
      />
      <section className="panel">
        <DataTable
          columns={columns}
          rows={rows}
          emptyText="No cross-exchange arbitrage results are currently available."
          getRowKey={(row, index) => `${row.symbol}-${row.buy_provider}-${row.sell_provider}-${index}`}
        />
      </section>
    </>
  );
}
