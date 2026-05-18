import { useGetHealthQuery } from '../api/arbApi';
import { DataTable, type Column } from '../components/DataTable';
import { PageHeader } from '../components/PageHeader';
import { StatusBadge } from '../components/StatusBadge';
import { SummaryCard } from '../components/SummaryCard';
import { useAppSelector } from '../hooks/redux';
import { usePollingInterval } from '../hooks/useRefresh';
import { formatDateTime } from '../utils/time';

interface EndpointRow {
  endpoint: string;
  auth: string;
  support: string;
  note: string;
}

const endpointRows: EndpointRow[] = [
  { endpoint: '/health', auth: 'None', support: 'Implemented', note: 'Public liveness endpoint' },
  { endpoint: '/api/v1/snapshot', auth: 'X-API-Key', support: 'Implemented', note: 'Aggregate TUI snapshot' },
  { endpoint: '/api/v1/prices', auth: 'X-API-Key', support: 'Implemented', note: 'Spot, futures, funding' },
  { endpoint: '/api/v1/providers/health', auth: 'X-API-Key', support: 'Implemented', note: 'Provider health map' },
  { endpoint: '/api/v1/metrics/snapshot', auth: 'X-API-Key', support: 'Removed', note: 'Go test asserts 404' },
  { endpoint: '/metrics', auth: 'None', support: 'Conditional', note: 'Only registered when Prometheus is enabled' },
];

const columns: Column<EndpointRow>[] = [
  { id: 'endpoint', header: 'Endpoint', cell: (row) => row.endpoint },
  { id: 'auth', header: 'Auth', cell: (row) => row.auth },
  { id: 'support', header: 'Support', cell: (row) => row.support },
  { id: 'note', header: 'Note', cell: (row) => row.note },
];

function apiKeyStatus(apiKey: string, source: string): string {
  if (!apiKey) {
    return 'Missing';
  }
  return `${source}, ${apiKey.length} characters`;
}

export function ApiStatusPage() {
  const pollingInterval = usePollingInterval();
  const { data } = useGetHealthQuery(undefined, { pollingInterval });
  const settings = useAppSelector((state) => state.settings);
  const apiStatus = useAppSelector((state) => state.apiStatus);

  return (
    <>
      <PageHeader title="API / Backend Status" subtitle="Backend connectivity, authentication, and endpoint support." />
      <section className="summaryGrid">
        <SummaryCard
          title="/health"
          value={data?.status ?? 'Unknown'}
          detail={data?.version ? `Version ${data.version}` : 'No health response yet'}
          tone={data?.status === 'ok' ? 'ok' : 'warning'}
        />
        <SummaryCard title="API base URL" value={settings.apiBaseUrl} />
        <SummaryCard title="API key status" value={apiKeyStatus(settings.apiKey, settings.apiKeySource)} />
        <SummaryCard title="Latency" value={`${apiStatus.latencyMs ?? 'n/a'} ms`} />
      </section>
      <section className="panel">
        <div className="sectionHeader">
          <h2>Request State</h2>
          <StatusBadge
            label={apiStatus.authFailed ? 'Auth failed' : apiStatus.backendUnavailable ? 'Unavailable' : 'OK'}
            tone={apiStatus.authFailed || apiStatus.backendUnavailable ? 'error' : 'ok'}
          />
        </div>
        <div className="detailGrid">
          <span>Last successful request</span>
          <strong>{formatDateTime(apiStatus.lastSuccessfulRequest?.at)}</strong>
          <span>Last failed request</span>
          <strong>{formatDateTime(apiStatus.lastFailedRequest?.at)}</strong>
          <span>Error details</span>
          <strong>{apiStatus.lastError || 'none'}</strong>
        </div>
      </section>
      <section className="panel">
        <div className="sectionHeader">
          <h2>Endpoint Compatibility</h2>
          <span>Based on docs, Swagger, and `internal/api/server.go`.</span>
        </div>
        <DataTable
          columns={columns}
          rows={endpointRows}
          emptyText="No endpoint metadata is available."
          getRowKey={(row) => row.endpoint}
        />
      </section>
    </>
  );
}
