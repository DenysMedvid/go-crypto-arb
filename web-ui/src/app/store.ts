import { configureStore } from '@reduxjs/toolkit';
import { setupListeners } from '@reduxjs/toolkit/query';

import { arbApi } from '../api/arbApi';
import { apiStatusReducer } from '../features/apiStatusSlice';
import { filtersReducer } from '../features/filtersSlice';
import { settingsReducer, writePersistedSettings } from '../features/settingsSlice';

export const store = configureStore({
  reducer: {
    apiStatus: apiStatusReducer,
    filters: filtersReducer,
    settings: settingsReducer,
    [arbApi.reducerPath]: arbApi.reducer,
  },
  middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(arbApi.middleware),
});

setupListeners(store.dispatch);

let previousPersisted = '';
store.subscribe(() => {
  const settings = store.getState().settings;
  const serialized = JSON.stringify(settings);
  if (serialized !== previousPersisted) {
    previousPersisted = serialized;
    writePersistedSettings(settings);
  }
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
