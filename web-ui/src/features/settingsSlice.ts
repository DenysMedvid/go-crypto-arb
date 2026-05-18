import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

export type ThemeMode = 'system' | 'light' | 'dark';
export type ApiKeySource = 'env' | 'localStorage' | 'none';

export interface SettingsState {
  apiBaseUrl: string;
  apiKey: string;
  apiKeySource: ApiKeySource;
  autoRefresh: boolean;
  refreshIntervalMs: number;
  compactMode: boolean;
  emojiEnabled: boolean;
  profitableOnly: boolean;
  themeMode: ThemeMode;
}

export const SETTINGS_STORAGE_KEY = 'go-crypto-arb.web-ui.settings';

export interface PersistedSettings {
  apiBaseUrl?: string;
  apiKey?: string;
  autoRefresh?: boolean;
  refreshIntervalMs?: number;
  compactMode?: boolean;
  emojiEnabled?: boolean;
  profitableOnly?: boolean;
  themeMode?: ThemeMode;
}

const envApiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';
const envApiKey = import.meta.env.VITE_API_KEY || '';

export function readPersistedSettings(storage: Storage | undefined = safeLocalStorage()):
  | PersistedSettings
  | undefined {
  if (!storage) {
    return undefined;
  }
  const raw = storage.getItem(SETTINGS_STORAGE_KEY);
  if (!raw) {
    return undefined;
  }
  try {
    return JSON.parse(raw) as PersistedSettings;
  } catch {
    return undefined;
  }
}

export function writePersistedSettings(
  settings: SettingsState,
  storage: Storage | undefined = safeLocalStorage(),
): void {
  if (!storage) {
    return;
  }
  const persisted: PersistedSettings = {
    apiBaseUrl: settings.apiBaseUrl,
    autoRefresh: settings.autoRefresh,
    refreshIntervalMs: settings.refreshIntervalMs,
    compactMode: settings.compactMode,
    emojiEnabled: settings.emojiEnabled,
    profitableOnly: settings.profitableOnly,
    themeMode: settings.themeMode,
  };
  if (settings.apiKeySource === 'localStorage' && settings.apiKey) {
    persisted.apiKey = settings.apiKey;
  }
  storage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(persisted));
}

export function createInitialSettings(
  persisted: PersistedSettings | undefined = readPersistedSettings(),
): SettingsState {
  const storedKey = persisted?.apiKey || '';
  const apiKey = storedKey || envApiKey;
  return {
    apiBaseUrl: persisted?.apiBaseUrl || envApiBaseUrl,
    apiKey,
    apiKeySource: storedKey ? 'localStorage' : envApiKey ? 'env' : 'none',
    autoRefresh: persisted?.autoRefresh ?? true,
    refreshIntervalMs: clampRefreshInterval(persisted?.refreshIntervalMs ?? 2000),
    compactMode: persisted?.compactMode ?? false,
    emojiEnabled: persisted?.emojiEnabled ?? true,
    profitableOnly: persisted?.profitableOnly ?? false,
    themeMode: persisted?.themeMode ?? 'system',
  };
}

function safeLocalStorage(): Storage | undefined {
  if (typeof window === 'undefined') {
    return undefined;
  }
  return window.localStorage;
}

function clampRefreshInterval(value: number): number {
  if (!Number.isFinite(value)) {
    return 2000;
  }
  return Math.min(60_000, Math.max(1000, Math.round(value)));
}

const settingsSlice = createSlice({
  name: 'settings',
  initialState: createInitialSettings(),
  reducers: {
    setApiBaseUrl(state, action: PayloadAction<string>) {
      state.apiBaseUrl = action.payload.trim().replace(/\/+$/, '') || envApiBaseUrl;
    },
    setApiKey(state, action: PayloadAction<string>) {
      state.apiKey = action.payload;
      state.apiKeySource = action.payload ? 'localStorage' : envApiKey ? 'env' : 'none';
    },
    clearApiKey(state) {
      state.apiKey = envApiKey;
      state.apiKeySource = envApiKey ? 'env' : 'none';
    },
    setAutoRefresh(state, action: PayloadAction<boolean>) {
      state.autoRefresh = action.payload;
    },
    setRefreshIntervalMs(state, action: PayloadAction<number>) {
      state.refreshIntervalMs = clampRefreshInterval(action.payload);
    },
    setCompactMode(state, action: PayloadAction<boolean>) {
      state.compactMode = action.payload;
    },
    setEmojiEnabled(state, action: PayloadAction<boolean>) {
      state.emojiEnabled = action.payload;
    },
    setProfitableOnly(state, action: PayloadAction<boolean>) {
      state.profitableOnly = action.payload;
    },
    setThemeMode(state, action: PayloadAction<ThemeMode>) {
      state.themeMode = action.payload;
    },
  },
});

export const {
  clearApiKey,
  setApiBaseUrl,
  setApiKey,
  setAutoRefresh,
  setCompactMode,
  setEmojiEnabled,
  setProfitableOnly,
  setRefreshIntervalMs,
  setThemeMode,
} = settingsSlice.actions;

export const settingsReducer = settingsSlice.reducer;
