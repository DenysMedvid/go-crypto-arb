import type { ReactNode } from 'react';
import { NavLink } from 'react-router-dom';

import { arbApi } from '../api/arbApi';
import { useAppDispatch, useAppSelector } from '../hooks/redux';
import { useRefreshCountdown } from '../hooks/useRefresh';
import { setAutoRefresh } from '../features/settingsSlice';
import { formatDateTime } from '../utils/time';

interface LayoutProps {
  children: ReactNode;
}

const navItems = [
  { to: '/', label: 'Crypto Dashboard', icon: '📊' },
  { to: '/prices', label: 'Prices', icon: '💰' },
  { to: '/triangular', label: 'Crypto Triangular', icon: '🔺' },
  { to: '/cross-exchange', label: 'Cross-Exchange', icon: '🔁' },
  { to: '/spot-futures', label: 'Crypto Spot-Futures', icon: '📈' },
  { to: '/signals', label: 'Related Signals', icon: '🧭' },
  { to: '/alerts', label: 'Alerts', icon: '🚨' },
  { to: '/provider-health', label: 'Provider Health', icon: '🩺' },
  { to: '/ibkr', label: 'IBKR Monitor', icon: '🏦' },
  { to: '/status', label: 'API Status', icon: '✅' },
  { to: '/settings', label: 'Settings', icon: '⚙️' },
];

const refreshTags = [
  'Alerts',
  'Arbitrage',
  'Health',
  'IBKR',
  'Prices',
  'Providers',
  'Signals',
  'Snapshot',
] as const;

export function Layout({ children }: LayoutProps) {
  const dispatch = useAppDispatch();
  const settings = useAppSelector((state) => state.settings);
  const apiStatus = useAppSelector((state) => state.apiStatus);
  const countdown = useRefreshCountdown();

  const icon = (value: string) => (settings.emojiEnabled ? <span aria-hidden="true">{value}</span> : null);
  const refreshNow = () => {
    dispatch(arbApi.util.invalidateTags([...refreshTags]));
  };

  return (
    <div className={`appShell ${settings.compactMode ? 'compact' : ''} theme-${settings.themeMode}`}>
      <aside className="sidebar">
        <div className="brand">
          <strong>go-crypto-arb</strong>
          <span>Web UI</span>
        </div>
        <nav aria-label="Main navigation">
          {navItems.map((item) => (
            <NavLink key={item.to} to={item.to} end={item.to === '/'}>
              {icon(item.icon)}
              <span>{item.label}</span>
            </NavLink>
          ))}
        </nav>
      </aside>
      <div className="contentColumn">
        <header className="topbar">
          <div>
            <div className="eyebrow">Backend</div>
            <strong>{apiStatus.backendUnavailable ? 'Unavailable' : 'Monitoring'}</strong>
          </div>
          <div className="topbarFacts">
            <span>URL: {settings.apiBaseUrl}</span>
            <span>Latency: {apiStatus.latencyMs ?? 'n/a'} ms</span>
            <span>Refresh: {countdown}</span>
            <span>Last OK: {formatDateTime(apiStatus.lastSuccessfulRequest?.at)}</span>
          </div>
          <div className="topbarActions">
            <button type="button" onClick={refreshNow}>
              Refresh
            </button>
            <button
              type="button"
              className={settings.autoRefresh ? 'secondary' : 'warningButton'}
              onClick={() => dispatch(setAutoRefresh(!settings.autoRefresh))}
            >
              {settings.autoRefresh ? 'Pause' : 'Resume'}
            </button>
          </div>
        </header>
        {apiStatus.lastError ? (
          <div className="errorBanner" role="alert">
            <strong>{apiStatus.authFailed ? 'Authentication failed' : 'Request failed'}</strong>
            <span>{apiStatus.lastError}</span>
            {apiStatus.lastSuccessfulRequest ? <em>Showing cached data where available.</em> : null}
          </div>
        ) : null}
        <main>{children}</main>
      </div>
    </div>
  );
}
