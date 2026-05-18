import { useMemo } from 'react';

import {
  useGetIBKRCryptoFuturesBasisQuery,
  useGetIBKRFXTriangularQuery,
  useGetIBKRInstrumentsQuery,
  useGetProviderHealthQuery,
} from '../api/arbApi';
import type { BrokerFuturesBasisOpportunity, ExchangeHealth, MarketInfo, TriangularOpportunity } from '../api/types';
import { DataTable, type Column } from '../components/DataTable';
import { IBKRTradingStatus } from '../components/IBKRTradingStatus';
import { PageHeader } from '../components/PageHeader';
import { StatusBadge } from '../components/StatusBadge';
import { SummaryCard } from '../components/SummaryCard';
import { usePollingInterval } from '../hooks/useRefresh';
import { formatDecimal, formatPercent, profitClass } from '../utils/decimal';
import { healthTone, providerLabel } from '../utils/status';

function findIBKRHealth(values: ExchangeHealth[]): ExchangeHealth | undefined {
  return values.find(
    (item) =>
      item.provider?.toLowerCase() === 'ibkr' ||
      item.broker?.toLowerCase() === 'ibkr' ||
      item.exchange?.toLowerCase() === 'ibkr',
  );
}

const instrumentColumns: Column<MarketInfo>[] = [
  { id: 'name', header: 'Instrument', cell: (row) => row.display_name || row.instrument_id || row.symbol },
  { id: 'assetClass', header: 'Asset class', cell: (row) => row.asset_class },
  { id: 'market', header: 'Market', cell: (row) => row.market_type },
  { id: 'symbol', header: 'Symbol', cell: (row) => row.symbol },
  { id: 'exchange', header: 'Exchange', cell: (row) => row.exchange },
  { id: 'active', header: 'Active', cell: (row) => (row.active ? 'Yes' : 'No') },
];

const fxColumns: Column<TriangularOpportunity>[] = [
  { id: 'cycle', header: 'Cycle', cell: (row) => row.cycle.join(' → ') },
  { id: 'size', header: 'Trade size', cell: (row) => formatDecimal(row.start_amount), className: 'numeric' },
  { id: 'end', header: 'End amount', cell: (row) => formatDecimal(row.end_amount), className: 'numeric' },
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
  { id: 'status', header: 'Status', cell: (row) => row.status },
];

const basisColumns: Column<BrokerFuturesBasisOpportunity>[] = [
  { id: 'asset', header: 'Asset', cell: (row) => row.asset },
  { id: 'spotProvider', header: 'Spot provider', cell: (row) => row.spot_provider },
  { id: 'spotAsk', header: 'Spot ask', cell: (row) => formatDecimal(row.spot_ask), className: 'numeric' },
  { id: 'future', header: 'IBKR future', cell: (row) => row.futures_display_name || row.futures_instrument_id },
  { id: 'futureBid', header: 'Futures bid', cell: (row) => formatDecimal(row.futures_bid), className: 'numeric' },
  { id: 'basis', header: 'Basis %', cell: (row) => formatPercent(row.basis_percent), className: 'numeric' },
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
  { id: 'fill', header: 'Complete fill', cell: (row) => (row.complete_fill ? 'Yes' : 'Partial fill') },
];

export function IBKRPage() {
  const pollingInterval = usePollingInterval();
  const { data: healthMap } = useGetProviderHealthQuery(undefined, { pollingInterval });
  const { data: instruments = [] } = useGetIBKRInstrumentsQuery(undefined, { pollingInterval });
  const { data: fx = [] } = useGetIBKRFXTriangularQuery(undefined, { pollingInterval });
  const { data: basis = [] } = useGetIBKRCryptoFuturesBasisQuery(undefined, { pollingInterval });
  const healthRows = useMemo(() => Object.values(healthMap ?? {}), [healthMap]);
  const ibkrHealth = findIBKRHealth(healthRows);

  return (
    <>
      <PageHeader
        title="IBKR Monitor"
        subtitle="Broker market-data monitoring kept separate from crypto exchange strategy views."
      />
      <section className="summaryGrid">
        <SummaryCard
          title="IBKR status"
          value={ibkrHealth?.status ?? 'Not reported'}
          detail={ibkrHealth ? providerLabel(ibkrHealth) : 'No IBKR health payload'}
          tone={healthTone(ibkrHealth) === 'error' ? 'error' : healthTone(ibkrHealth) === 'ok' ? 'ok' : 'warning'}
        />
        <SummaryCard
          title="Trading"
          value={ibkrHealth?.trading_enabled ? 'UNSUPPORTED' : 'DISABLED'}
          detail="No order placement UI exists."
          tone={ibkrHealth?.trading_enabled ? 'error' : 'ok'}
        />
        <SummaryCard title="Instruments" value={instruments.length} detail="Configured IBKR instruments" />
        <SummaryCard
          title="IBKR health"
          value={ibkrHealth?.score ?? 'n/a'}
          detail={ibkrHealth?.last_error || 'No last error reported'}
          tone={healthTone(ibkrHealth) === 'error' ? 'error' : 'neutral'}
        />
      </section>
      <section className="panel">
        <div className="sectionHeader">
          <h2>IBKR Status</h2>
          <StatusBadge label={ibkrHealth?.status ?? 'unknown'} tone={healthTone(ibkrHealth)} />
        </div>
        <IBKRTradingStatus health={ibkrHealth} />
      </section>
      <section className="panel">
        <div className="sectionHeader">
          <h2>IBKR Instruments</h2>
          <span>Configured, not mixed into crypto triangular arbitrage.</span>
        </div>
        <DataTable
          columns={instrumentColumns}
          rows={instruments}
          emptyText="No IBKR instruments are configured or returned by the backend."
          getRowKey={(row, index) => row.instrument_id || `${row.symbol}-${index}`}
        />
      </section>
      <section className="panel">
        <div className="sectionHeader">
          <h2>IBKR FX Triangular Arbitrage</h2>
          <span>Monitoring-only FX estimates.</span>
        </div>
        <DataTable
          columns={fxColumns}
          rows={fx}
          emptyText="No IBKR FX triangular results are currently available."
          getRowKey={(row, index) => `${row.cycle.join('-')}-${index}`}
        />
      </section>
      <section className="panel">
        <div className="sectionHeader">
          <h2>Crypto Spot vs IBKR Futures Basis</h2>
          <span>Basis monitoring, not guaranteed arbitrage.</span>
        </div>
        <DataTable
          columns={basisColumns}
          rows={basis}
          emptyText="No crypto spot vs IBKR futures basis results are currently available."
          getRowKey={(row, index) => `${row.asset}-${row.futures_instrument_id}-${index}`}
        />
      </section>
    </>
  );
}
