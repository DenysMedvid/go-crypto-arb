import type { ExchangeHealth } from '../api/types';
import { StatusBadge } from './StatusBadge';

interface IBKRTradingStatusProps {
  health?: ExchangeHealth;
}

export function IBKRTradingStatus({ health }: IBKRTradingStatusProps) {
  const enabled = Boolean(health?.trading_enabled);
  return (
    <div className="inlineFacts">
      <span>Market data mode</span>
      <StatusBadge
        label={health?.market_data_enabled === false ? 'Unavailable' : 'Market data only'}
        tone={health?.market_data_ok ? 'ok' : 'warning'}
      />
      <span>Trading</span>
      <StatusBadge label={enabled ? 'UNSUPPORTED' : 'DISABLED'} tone={enabled ? 'error' : 'ok'} />
    </div>
  );
}
