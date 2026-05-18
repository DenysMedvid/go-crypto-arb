import { useState } from 'react';

import { arbApi } from '../api/arbApi';
import { PageHeader } from '../components/PageHeader';
import {
  clearApiKey,
  setApiBaseUrl,
  setApiKey,
  setAutoRefresh,
  setCompactMode,
  setEmojiEnabled,
  setProfitableOnly,
  setRefreshIntervalMs,
  setThemeMode,
  type ThemeMode,
} from '../features/settingsSlice';
import { useAppDispatch, useAppSelector } from '../hooks/redux';

export function SettingsPage() {
  const dispatch = useAppDispatch();
  const settings = useAppSelector((state) => state.settings);
  const [apiKeyDraft, setApiKeyDraft] = useState('');

  const saveKey = () => {
    dispatch(setApiKey(apiKeyDraft));
    setApiKeyDraft('');
    dispatch(arbApi.util.resetApiState());
  };

  return (
    <>
      <PageHeader title="Settings" subtitle="Client-side preferences for this browser." />
      <section className="panel settingsPanel">
        <label>
          API base URL
          <input
            value={settings.apiBaseUrl}
            onChange={(event) => dispatch(setApiBaseUrl(event.target.value))}
            placeholder="http://localhost:8080"
          />
        </label>
        <label>
          API key
          <input
            type="password"
            value={apiKeyDraft}
            onChange={(event) => setApiKeyDraft(event.target.value)}
            placeholder={settings.apiKey ? 'Configured' : 'change-me'}
          />
        </label>
        <div className="buttonRow">
          <button type="button" onClick={saveKey}>
            Save API key in this browser
          </button>
          <button type="button" className="secondary" onClick={() => dispatch(clearApiKey())}>
            Clear browser key
          </button>
        </div>
        <p className="warningText">
          Browser local storage is convenient for local dashboards, but it is readable by scripts
          running on this origin. Prefer `VITE_API_KEY` for local development when possible.
        </p>
      </section>
      <section className="panel settingsPanel">
        <label>
          Auto-refresh interval
          <input
            type="number"
            min={1000}
            step={1000}
            value={settings.refreshIntervalMs}
            onChange={(event) => dispatch(setRefreshIntervalMs(Number(event.target.value)))}
          />
        </label>
        <label className="checkLabel">
          <input
            type="checkbox"
            checked={settings.autoRefresh}
            onChange={(event) => dispatch(setAutoRefresh(event.target.checked))}
          />
          Auto-refresh enabled
        </label>
        <label className="checkLabel">
          <input
            type="checkbox"
            checked={settings.compactMode}
            onChange={(event) => dispatch(setCompactMode(event.target.checked))}
          />
          Compact mode
        </label>
        <label className="checkLabel">
          <input
            type="checkbox"
            checked={settings.emojiEnabled}
            onChange={(event) => dispatch(setEmojiEnabled(event.target.checked))}
          />
          Emoji/icons enabled
        </label>
        <label className="checkLabel">
          <input
            type="checkbox"
            checked={settings.profitableOnly}
            onChange={(event) => dispatch(setProfitableOnly(event.target.checked))}
          />
          Profitable-only filter
        </label>
        <label>
          Theme mode
          <select
            value={settings.themeMode}
            onChange={(event) => dispatch(setThemeMode(event.target.value as ThemeMode))}
          >
            <option value="system">System</option>
            <option value="light">Light</option>
            <option value="dark">Dark</option>
          </select>
        </label>
      </section>
    </>
  );
}
