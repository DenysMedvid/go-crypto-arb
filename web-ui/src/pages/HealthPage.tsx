import { useMemo } from 'react';

import { useGetProviderHealthQuery } from '../api/arbApi';
import type { ExchangeHealth } from '../api/types';
import { DataTable, type Column } from '../components/DataTable';
import { PageHeader } from '../components/PageHeader';
import { StatusBadge } from '../components/StatusBadge';
import { SummaryCard } from '../components/SummaryCard';
import { usePollingInterval } from '../hooks/useRefresh';
import { boolLabel, healthTone, providerLabel } from '../utils/status';
import { formatAge } from '../utils/time';

const columns: Column<ExchangeHealth>[] = [
  { id: 'provider', header: 'Provider', cell: (row) => providerLabel(row) },
  { id: 'type', header: 'Type', cell: (row) => row.provider_type || 'crypto_exchange' },
  {
    id: 'status',
    header: 'Status',
    cell: (row) => <StatusBadge label={row.status} tone={healthTone(row)} />,
  },
  { id: 'score', header: 'Score', cell: (row) => row.score, className: 'numeric' },
  { id: 'ws', header: 'WebSocket status', cell: (row) => boolLabel(row.websocket_connected) },
  { id: 'rest', header: 'REST fallback', cell: (row) => boolLabel(row.rest_fallback_active) },
  { id: 'last', header: 'Last message time', cell: (row) => formatAge(row.last_message_at || row.last_message_time) },
  { id: 'reconnects', header: 'Reconnect count', cell: (row) => row.reconnect_count, className: 'numeric' },
  { id: 'lastError', header: 'Last error', cell: (row) => row.last_error || 'none' },
  { id: 'staleTicker', header: 'Stale ticker count', cell: (row) => row.stale_ticker_count, className: 'numeric' },
  { id: 'staleBook', header: 'Stale order book count', cell: (row) => row.stale_order_book_count, className: 'numeric' },
];

export function HealthPage() {
  const pollingInterval = usePollingInterval();
  const { data } = useGetProviderHealthQuery(undefined, { pollingInterval });
  const rows = useMemo(
    () => Object.values(data ?? {}).sort((left, right) => providerLabel(left).localeCompare(providerLabel(right))),
    [data],
  );
  const okCount = rows.filter((row) => row.status === 'ok').length;
  const disconnected = rows.filter((row) => row.status === 'disconnected').length;

  return (
    <>
      <PageHeader
        title="Provider / Exchange Health"
        subtitle="Health score, connectivity, stale data, and fallback status by provider."
      />
      <section className="summaryGrid">
        <SummaryCard title="Providers" value={rows.length} />
        <SummaryCard title="Healthy" value={okCount} tone="ok" />
        <SummaryCard title="Disconnected" value={disconnected} tone={disconnected > 0 ? 'error' : 'ok'} />
      </section>
      <section className="panel">
        <DataTable
          columns={columns}
          rows={rows}
          emptyText="No provider health data is currently available."
          getRowKey={(row) => providerLabel(row)}
        />
      </section>
    </>
  );
}
