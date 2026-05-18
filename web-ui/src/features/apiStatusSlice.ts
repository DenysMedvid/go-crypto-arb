import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

export interface RequestStatusPayload {
  url: string;
  latencyMs: number;
  at: string;
  status?: string;
  message?: string;
  authFailed?: boolean;
  backendUnavailable?: boolean;
}

export interface ApiStatusState {
  lastSuccessfulRequest?: RequestStatusPayload;
  lastFailedRequest?: RequestStatusPayload;
  latencyMs?: number;
  authFailed: boolean;
  backendUnavailable: boolean;
  lastError?: string;
}

const initialState: ApiStatusState = {
  authFailed: false,
  backendUnavailable: false,
};

const apiStatusSlice = createSlice({
  name: 'apiStatus',
  initialState,
  reducers: {
    requestSucceeded(state, action: PayloadAction<RequestStatusPayload>) {
      state.lastSuccessfulRequest = action.payload;
      state.latencyMs = action.payload.latencyMs;
      state.authFailed = false;
      state.backendUnavailable = false;
      state.lastError = undefined;
    },
    requestFailed(state, action: PayloadAction<RequestStatusPayload>) {
      state.lastFailedRequest = action.payload;
      state.latencyMs = action.payload.latencyMs;
      state.authFailed = Boolean(action.payload.authFailed);
      state.backendUnavailable = Boolean(action.payload.backendUnavailable);
      state.lastError = action.payload.message;
    },
  },
});

export const { requestFailed, requestSucceeded } = apiStatusSlice.actions;
export const apiStatusReducer = apiStatusSlice.reducer;
