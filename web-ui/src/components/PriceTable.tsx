import type { PriceRow } from '../utils/filters';
import { formatDecimal } from '../utils/decimal';
import { buildPriceHighlightStats, priceHighlightClass } from '../utils/priceHighlights';
import { formatAge } from '../utils/time';
import { DataTable, type Column } from './DataTable';
import { StatusBadge } from './StatusBadge';

interface PriceTableProps {
  rows: PriceRow[];
}

export function PriceTable({ rows }: PriceTableProps) {
  const highlights = buildPriceHighlightStats(rows);
  const columns: Column<PriceRow>[] = [
    { id: 'provider', header: 'Provider', cell: (row) => row.providerLabel },
    { id: 'market', header: 'Market', cell: (row) => row.market_type },
    {
      id: 'symbol',
      header: 'Symbol',
      cell: (row) => <span className={row.status === 'stale' ? 'priceStale' : ''}>{row.symbol}</span>,
    },
    {
      id: 'bid',
      header: 'Bid',
      cell: (row) => (
        <span
          className={priceHighlightClass(
            highlights.bidHighlight(row),
            row.status === 'stale' ? 'priceStale' : '',
          )}
        >
          {formatDecimal(row.bid)}
        </span>
      ),
      className: 'numeric',
    },
    {
      id: 'ask',
      header: 'Ask',
      cell: (row) => (
        <span
          className={priceHighlightClass(
            highlights.askHighlight(row),
            row.status === 'stale' ? 'priceStale' : '',
          )}
        >
          {formatDecimal(row.ask)}
        </span>
      ),
      className: 'numeric',
    },
    {
      id: 'last',
      header: 'Last',
      cell: (row) => (
        <span className={row.status === 'stale' ? 'priceStale' : ''}>{formatDecimal(row.last)}</span>
      ),
      className: 'numeric',
    },
    {
      id: 'age',
      header: 'Age',
      cell: (row) => (
        <span className={row.status === 'stale' ? 'priceStale' : ''}>{formatAge(row.updated_at)}</span>
      ),
      className: 'numeric',
    },
    {
      id: 'status',
      header: 'Status',
      cell: (row) => (
        <StatusBadge
          label={row.status === 'stale' ? 'Stale data' : 'OK'}
          tone={row.status === 'stale' ? 'warning' : 'ok'}
        />
      ),
    },
  ];

  return (
    <DataTable
      columns={columns}
      rows={rows}
      emptyText="No price data is available for the selected filters."
      getRowKey={(row) => `${row.providerLabel}-${row.market_type}-${row.symbol}`}
    />
  );
}
