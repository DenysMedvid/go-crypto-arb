import { useMemo } from 'react';

import { useGetSnapshotQuery } from '../api/arbApi';
import { DataTable, type Column } from '../components/DataTable';
import { EmptyState } from '../components/EmptyState';
import { PageHeader } from '../components/PageHeader';
import { StatusBadge } from '../components/StatusBadge';
import { SummaryCard } from '../components/SummaryCard';
import { usePollingInterval } from '../hooks/useRefresh';
import { decimalToNumber, formatDecimal, formatPercent, profitClass } from '../utils/decimal';
import { flattenPrices, type PriceRow } from '../utils/filters';
import { buildPriceHighlightStats, priceHighlightClass } from '../utils/priceHighlights';
import { formatAge } from '../utils/time';

interface DashboardOpportunity {
  kind: string;
  label: string;
  provider: string;
  profit: number;
  fill: boolean;
  status: string;
}

const opportunityColumns: Column<DashboardOpportunity>[] = [
  { id: 'kind', header: 'Type', cell: (row) => row.kind },
  { id: 'label', header: 'Opportunity', cell: (row) => row.label },
  { id: 'provider', header: 'Provider', cell: (row) => row.provider },
  {
    id: 'profit',
    header: 'Estimated profit',
    cell: (row) => <span className={profitClass(row.profit)}>{formatPercent(row.profit)}</span>,
    className: 'numeric',
  },
  {
    id: 'fill',
    header: 'Fill',
    cell: (row) => (
      <StatusBadge label={row.fill ? 'Complete' : 'Partial fill'} tone={row.fill ? 'ok' : 'warning'} />
    ),
  },
  { id: 'status', header: 'Status', cell: (row) => row.status || 'watch' },
];

export function DashboardPage() {
  const pollingInterval = usePollingInterval();
  const { data } = useGetSnapshotQuery(undefined, { pollingInterval });

  const opportunities = useMemo<DashboardOpportunity[]>(() => {
    if (!data) {
      return [];
    }
    return [
      ...(data.triangular_arbitrage ?? []).map((item) => ({
        kind: 'Triangular',
        label: item.cycle.join(' → '),
        provider: item.provider || item.exchange,
        profit: decimalToNumber(item.net_profit_percent),
        fill: item.complete_fill,
        status: item.status,
      })),
      ...(data.cross_exchange_arbitrage ?? []).map((item) => ({
        kind: 'Cross-exchange',
        label: item.symbol,
        provider: `${item.buy_provider} → ${item.sell_provider}`,
        profit: decimalToNumber(item.net_profit_percent),
        fill: item.complete_fill,
        status: item.status,
      })),
      ...(data.spot_futures_arbitrage ?? []).map((item) => ({
        kind: 'Spot-futures',
        label: item.symbol,
        provider: item.provider || item.exchange,
        profit: decimalToNumber(item.net_estimate_percent),
        fill: item.complete_fill,
        status: item.status,
      })),
    ]
      .sort((left, right) => right.profit - left.profit)
      .slice(0, 8);
  }, [data]);

  const healthItems = Object.values(data?.provider_health ?? data?.exchange_health ?? {});
  const unhealthy = healthItems.filter((item) => item.status !== 'ok').length;
  const spotCount = Object.values(data?.prices ?? {}).reduce((count, rows) => count + rows.length, 0);
  const alertCount = data?.alerts?.length ?? 0;
  const priceRows = useMemo(() => flattenPrices(data?.prices), [data]);
  const priceHighlights = useMemo(() => buildPriceHighlightStats(priceRows), [priceRows]);
  const priceRowsByProvider = useMemo(() => {
    const grouped = new Map<string, PriceRow[]>();
    for (const row of priceRows) {
      const group = grouped.get(row.exchange) ?? [];
      group.push(row);
      grouped.set(row.exchange, group);
    }
    return [...grouped.entries()].sort(([left], [right]) => left.localeCompare(right));
  }, [priceRows]);
  const ibkrHealth = healthItems.find(
    (item) => item.provider === 'ibkr' || item.broker === 'IBKR' || item.exchange === 'IBKR',
  );

  return (
    <>
      <PageHeader
        title="Crypto Dashboard"
        subtitle="Monitoring-only snapshot view. Estimated opportunities are not guaranteed executable."
      />
      <section className="summaryGrid">
        <SummaryCard title="Spot prices" value={spotCount} detail="Grouped by provider" />
        <SummaryCard
          title="Alerts"
          value={alertCount}
          detail={alertCount > 0 ? 'Review active alerts' : 'No active alerts'}
          tone={alertCount > 0 ? 'warning' : 'ok'}
        />
        <SummaryCard
          title="Health"
          value={unhealthy === 0 ? 'OK' : `${unhealthy} degraded`}
          detail={`${healthItems.length} providers reported`}
          tone={unhealthy === 0 ? 'ok' : 'warning'}
        />
        <SummaryCard
          title="IBKR"
          value={ibkrHealth?.status ?? 'Not reported'}
          detail={ibkrHealth?.trading_enabled ? 'Trading unsupported' : 'Trading: DISABLED'}
          tone={ibkrHealth?.status === 'ok' ? 'ok' : 'warning'}
        />
      </section>

      <section className="panel">
        <div className="sectionHeader">
          <h2>Top Opportunities</h2>
          <span>Watch estimates with stale and partial-fill context.</span>
        </div>
        <DataTable
          columns={opportunityColumns}
          rows={opportunities}
          emptyText="No arbitrage opportunities are currently reported."
          getRowKey={(row, index) => `${row.kind}-${row.label}-${index}`}
        />
      </section>

      <section className="dashboardGrid">
        {priceRowsByProvider.map(([provider, rows]) => (
          <section className="panel" key={provider}>
            <div className="sectionHeader">
              <h2>{provider} Prices</h2>
              <span>{rows.length} symbols</span>
            </div>
            <div className="miniRows">
              {rows.slice(0, 8).map((row) => (
                <div className="miniRow" key={`${provider}-${row.symbol}`}>
                  <strong className={row.status === 'stale' ? 'priceStale' : ''}>{row.symbol}</strong>
                  <span
                    className={priceHighlightClass(
                      priceHighlights.bidHighlight(row),
                      row.status === 'stale' ? 'priceStale' : '',
                    )}
                  >
                    {formatDecimal(row.bid)} bid
                  </span>
                  <span
                    className={priceHighlightClass(
                      priceHighlights.askHighlight(row),
                      row.status === 'stale' ? 'priceStale' : '',
                    )}
                  >
                    {formatDecimal(row.ask)} ask
                  </span>
                  <span className={row.status === 'stale' ? 'priceStale' : ''}>
                    {formatAge(row.updated_at)}
                  </span>
                </div>
              ))}
            </div>
          </section>
        ))}
        {spotCount === 0 ? (
          <EmptyState
            title="No prices yet"
            message="The backend snapshot is reachable but has not reported spot prices."
          />
        ) : null}
      </section>
    </>
  );
}
