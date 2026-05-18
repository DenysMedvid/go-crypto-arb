import { useMemo, useState } from 'react';

import { useGetTriangularArbitrageQuery } from '../api/arbApi';
import type { TriangularOpportunity } from '../api/types';
import { DataTable, type Column } from '../components/DataTable';
import { Modal } from '../components/Modal';
import { PageHeader } from '../components/PageHeader';
import { StatusBadge } from '../components/StatusBadge';
import { useAppSelector } from '../hooks/redux';
import { usePollingInterval } from '../hooks/useRefresh';
import { formatDecimal, formatPercent, profitClass } from '../utils/decimal';
import { filterProfitableOnly } from '../utils/filters';

export function TriangularPage() {
  const pollingInterval = usePollingInterval();
  const profitableOnly = useAppSelector((state) => state.settings.profitableOnly);
  const { data = [] } = useGetTriangularArbitrageQuery(undefined, { pollingInterval });
  const [selected, setSelected] = useState<TriangularOpportunity | undefined>();
  const rows = useMemo(() => filterProfitableOnly(data, profitableOnly), [data, profitableOnly]);

  const columns: Column<TriangularOpportunity>[] = [
    { id: 'provider', header: 'Provider', cell: (row) => row.provider || row.exchange },
    { id: 'cycle', header: 'Cycle', cell: (row) => row.cycle.join(' → ') },
    { id: 'size', header: 'Trade size', cell: (row) => formatDecimal(row.start_amount), className: 'numeric' },
    { id: 'start', header: 'Start amount', cell: (row) => formatDecimal(row.start_amount), className: 'numeric' },
    { id: 'end', header: 'End amount', cell: (row) => formatDecimal(row.end_amount), className: 'numeric' },
    {
      id: 'profit',
      header: 'Net profit %',
      className: 'numeric',
      cell: (row) => (
        <span className={profitClass(row.net_profit_percent)}>
          {formatPercent(row.net_profit_percent)}
        </span>
      ),
    },
    {
      id: 'fill',
      header: 'Complete fill',
      cell: (row) => <StatusBadge label={row.complete_fill ? 'Yes' : 'Partial fill'} tone={row.complete_fill ? 'ok' : 'warning'} />,
    },
    { id: 'slippage', header: 'Max slippage', cell: (row) => formatPercent(row.max_slippage_percent), className: 'numeric' },
    { id: 'status', header: 'Status', cell: (row) => row.status },
    {
      id: 'detail',
      header: '',
      cell: (row) => (
        <button type="button" className="secondary" onClick={() => setSelected(row)}>
          Details
        </button>
      ),
    },
  ];

  return (
    <>
      <PageHeader
        title="Crypto Triangular Arbitrage"
        subtitle="Estimated cycle opportunities calculated by the backend from market data and order book depth."
      />
      <section className="panel">
        <DataTable
          columns={columns}
          rows={rows}
          emptyText="No triangular arbitrage results are currently available."
          getRowKey={(row, index) => `${row.provider}-${row.cycle.join('-')}-${index}`}
        />
      </section>
      {selected ? (
        <Modal title="Triangular Detail" onClose={() => setSelected(undefined)}>
          <div className="detailGrid">
            <span>Cycle</span>
            <strong>{selected.cycle.join(' → ')}</strong>
            <span>Fees and slippage</span>
            <strong>
              Worst leg {selected.worst_leg || 'n/a'}, max slippage{' '}
              {formatPercent(selected.max_slippage_percent)}
            </strong>
            <span>Partial fill info</span>
            <strong>{selected.complete_fill ? 'Complete fill' : 'Partial fill'}</strong>
          </div>
          <DataTable
            columns={[
              { id: 'from', header: 'From', cell: (row) => row.from_asset },
              { id: 'to', header: 'To', cell: (row) => row.to_asset },
              { id: 'symbol', header: 'Symbol', cell: (row) => row.symbol },
              { id: 'side', header: 'Side', cell: (row) => row.side },
              { id: 'input', header: 'Input', cell: (row) => formatDecimal(row.input_amount), className: 'numeric' },
              { id: 'output', header: 'Output', cell: (row) => formatDecimal(row.output_amount), className: 'numeric' },
              { id: 'fee', header: 'Fee', cell: (row) => formatDecimal(row.fee_amount, 6), className: 'numeric' },
              { id: 'slip', header: 'Slippage', cell: (row) => formatPercent(row.slippage_percent), className: 'numeric' },
              { id: 'fill', header: 'Fill', cell: (row) => (row.complete_fill ? 'Complete' : 'Partial') },
            ]}
            rows={selected.legs}
            emptyText="No legs were included in this opportunity."
            getRowKey={(row, index) => `${row.symbol}-${row.side}-${index}`}
          />
        </Modal>
      ) : null}
    </>
  );
}
