import { useMemo } from 'react';

import { useGetAlertsQuery } from '../api/arbApi';
import type { Alert } from '../api/types';
import { DataTable, type Column } from '../components/DataTable';
import { PageHeader } from '../components/PageHeader';
import { StatusBadge } from '../components/StatusBadge';
import { setAlertProvider, setAlertSeverity, setAlertType } from '../features/filtersSlice';
import { useAppDispatch, useAppSelector } from '../hooks/redux';
import { usePollingInterval } from '../hooks/useRefresh';
import { formatDecimal } from '../utils/decimal';
import { formatDateTime } from '../utils/time';

const columns: Column<Alert>[] = [
  { id: 'time', header: 'Time', cell: (row) => formatDateTime(row.updated_at) },
  {
    id: 'severity',
    header: 'Severity',
    cell: (row) => (
      <StatusBadge
        label={row.severity}
        tone={row.severity === 'critical' ? 'error' : row.severity === 'warning' ? 'warning' : 'muted'}
      />
    ),
  },
  { id: 'type', header: 'Type', cell: (row) => row.type },
  { id: 'message', header: 'Message', cell: (row) => row.message },
  { id: 'value', header: 'Value', cell: (row) => formatDecimal(row.value), className: 'numeric' },
  { id: 'threshold', header: 'Threshold', cell: (row) => formatDecimal(row.threshold), className: 'numeric' },
  { id: 'repeat', header: 'Repeat count', cell: (row) => row.repeat_count, className: 'numeric' },
];

export function AlertsPage() {
  const dispatch = useAppDispatch();
  const filters = useAppSelector((state) => state.filters.alerts);
  const pollingInterval = usePollingInterval();
  const { data = [] } = useGetAlertsQuery(undefined, { pollingInterval });
  const severities = [...new Set(data.map((row) => row.severity))].sort();
  const types = [...new Set(data.map((row) => row.type))].sort();
  const providers = [...new Set(data.map((row) => row.exchange || row.symbol).filter(Boolean))].sort();
  const rows = useMemo(
    () =>
      data.filter((row) => {
        if (filters.severity && row.severity !== filters.severity) {
          return false;
        }
        if (filters.type && row.type !== filters.type) {
          return false;
        }
        if (filters.provider && row.exchange !== filters.provider && row.symbol !== filters.provider) {
          return false;
        }
        return true;
      }),
    [data, filters],
  );

  return (
    <>
      <PageHeader title="Alerts" subtitle="Current in-memory alerts from the backend alert engine." />
      <section className="filters">
        <label>
          Severity
          <select
            value={filters.severity}
            onChange={(event) => dispatch(setAlertSeverity(event.target.value))}
          >
            <option value="">All</option>
            {severities.map((severity) => (
              <option key={severity} value={severity}>
                {severity}
              </option>
            ))}
          </select>
        </label>
        <label>
          Type
          <select value={filters.type} onChange={(event) => dispatch(setAlertType(event.target.value))}>
            <option value="">All</option>
            {types.map((type) => (
              <option key={type} value={type}>
                {type}
              </option>
            ))}
          </select>
        </label>
        <label>
          Provider / Symbol
          <select
            value={filters.provider}
            onChange={(event) => dispatch(setAlertProvider(event.target.value))}
          >
            <option value="">All</option>
            {providers.map((provider) => (
              <option key={provider} value={provider}>
                {provider}
              </option>
            ))}
          </select>
        </label>
      </section>
      <section className="panel">
        <DataTable
          columns={columns}
          rows={rows}
          emptyText="No alerts match the selected filters."
          getRowKey={(row, index) => row.id || `${row.type}-${index}`}
        />
      </section>
    </>
  );
}
