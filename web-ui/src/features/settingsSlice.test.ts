import { describe, expect, it } from 'vitest';

import {
  createInitialSettings,
  readPersistedSettings,
  SETTINGS_STORAGE_KEY,
  writePersistedSettings,
  type SettingsState,
} from './settingsSlice';

describe('settings persistence', () => {
  it('loads persisted refresh and API base URL settings', () => {
    const settings = createInitialSettings({
      apiBaseUrl: 'http://example.test:8080',
      refreshIntervalMs: 5000,
      autoRefresh: false,
      compactMode: true,
    });

    expect(settings.apiBaseUrl).toBe('http://example.test:8080');
    expect(settings.refreshIntervalMs).toBe(5000);
    expect(settings.autoRefresh).toBe(false);
    expect(settings.compactMode).toBe(true);
  });

  it('stores browser-configured API keys only when the source is localStorage', () => {
    const storage = window.localStorage;
    storage.clear();
    const base: SettingsState = {
      apiBaseUrl: 'http://localhost:8080',
      apiKey: 'secret',
      apiKeySource: 'env',
      autoRefresh: true,
      refreshIntervalMs: 2000,
      compactMode: false,
      emojiEnabled: true,
      profitableOnly: false,
      themeMode: 'system',
    };

    writePersistedSettings(base, storage);
    expect(storage.getItem(SETTINGS_STORAGE_KEY)).not.toContain('secret');

    writePersistedSettings({ ...base, apiKeySource: 'localStorage' }, storage);
    expect(readPersistedSettings(storage)?.apiKey).toBe('secret');
  });
});
