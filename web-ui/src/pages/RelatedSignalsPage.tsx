import { useMemo } from 'react';

import { useGetRelatedAssetSignalsQuery } from '../api/arbApi';
import type { Decimal } from '../api/types';
import { DataTable, type Column } from '../components/DataTable';
import { PageHeader } from '../components/PageHeader';
import { usePollingInterval } from '../hooks/useRefresh';
import { decimalToNumber, formatPercent, profitClass } from '../utils/decimal';

interface SignalRow {
  group: string;
  symbol: string;
  changePercent: Decimal;
  groupAverage: Decimal;
  divergencePercent: Decimal;
  signal: string;
}

const columns: Column<SignalRow>[] = [
  { id: 'group', header: 'Group', cell: (row) => row.group },
  { id: 'symbol', header: 'Symbol', cell: (row) => row.symbol },
  {
    id: 'change',
    header: 'Change %',
    cell: (row) => <span className={profitClass(row.changePercent)}>{formatPercent(row.changePercent)}</span>,
    className: 'numeric',
  },
  {
    id: 'avg',
    header: 'Group average %',
    cell: (row) => formatPercent(row.groupAverage),
    className: 'numeric',
  },
  {
    id: 'divergence',
    header: 'Divergence %',
    cell: (row) => (
      <span className={profitClass(row.divergencePercent)}>
        {formatPercent(row.divergencePercent)}
      </span>
    ),
    className: 'numeric',
  },
  { id: 'signal', header: 'Signal', cell: (row) => row.signal },
];

export function RelatedSignalsPage() {
  const pollingInterval = usePollingInterval();
  const { data = [] } = useGetRelatedAssetSignalsQuery(undefined, { pollingInterval });
  const rows = useMemo<SignalRow[]>(
    () =>
      data.flatMap((group) =>
        group.assets.map((asset) => {
          const divergence = decimalToNumber(asset.divergence_percent);
          return {
            group: group.group,
            symbol: asset.symbol,
            changePercent: asset.change_percent,
            groupAverage: group.group_average,
            divergencePercent: asset.divergence_percent,
            signal: Math.abs(divergence) < 0.05 ? 'In line' : divergence > 0 ? 'Outperforming' : 'Lagging',
          };
        }),
      ),
    [data],
  );

  return (
    <>
      <PageHeader
        title="Related Asset Signals"
        subtitle="Group divergence signals from configured related-asset sets."
      />
      <section className="panel">
        <DataTable
          columns={columns}
          rows={rows}
          emptyText="No related asset signals are currently available."
          getRowKey={(row) => `${row.group}-${row.symbol}`}
        />
      </section>
    </>
  );
}
