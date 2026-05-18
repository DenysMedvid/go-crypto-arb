import { useEffect, useMemo, useState } from 'react';

import { useAppSelector } from './redux';

export function usePollingInterval(): number {
  return useAppSelector((state) =>
    state.settings.autoRefresh ? state.settings.refreshIntervalMs : 0,
  );
}

export function useRefreshCountdown(): string {
  const autoRefresh = useAppSelector((state) => state.settings.autoRefresh);
  const refreshIntervalMs = useAppSelector((state) => state.settings.refreshIntervalMs);
  const lastCompletedAt = useAppSelector(
    (state) =>
      state.apiStatus.lastSuccessfulRequest?.at || state.apiStatus.lastFailedRequest?.at || '',
  );
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 250);
    return () => window.clearInterval(id);
  }, []);

  return useMemo(() => {
    if (!autoRefresh) {
      return 'paused';
    }
    const parsed = Date.parse(lastCompletedAt);
    if (!Number.isFinite(parsed)) {
      return 'pending';
    }
    const elapsed = Math.max(0, now - parsed);
    const remaining = Math.max(0, refreshIntervalMs - (elapsed % refreshIntervalMs));
    return `${Math.ceil(remaining / 1000)}s`;
  }, [autoRefresh, lastCompletedAt, now, refreshIntervalMs]);
}
