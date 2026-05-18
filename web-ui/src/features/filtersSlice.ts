import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

export interface FiltersState {
  prices: {
    provider: string;
    marketType: string;
    symbol: string;
    staleOnly: boolean;
  };
  alerts: {
    severity: string;
    type: string;
    provider: string;
  };
}

const initialState: FiltersState = {
  prices: {
    provider: '',
    marketType: '',
    symbol: '',
    staleOnly: false,
  },
  alerts: {
    severity: '',
    type: '',
    provider: '',
  },
};

const filtersSlice = createSlice({
  name: 'filters',
  initialState,
  reducers: {
    setPriceProvider(state, action: PayloadAction<string>) {
      state.prices.provider = action.payload;
    },
    setPriceMarketType(state, action: PayloadAction<string>) {
      state.prices.marketType = action.payload;
    },
    setPriceSymbol(state, action: PayloadAction<string>) {
      state.prices.symbol = action.payload;
    },
    setPriceStaleOnly(state, action: PayloadAction<boolean>) {
      state.prices.staleOnly = action.payload;
    },
    setAlertSeverity(state, action: PayloadAction<string>) {
      state.alerts.severity = action.payload;
    },
    setAlertType(state, action: PayloadAction<string>) {
      state.alerts.type = action.payload;
    },
    setAlertProvider(state, action: PayloadAction<string>) {
      state.alerts.provider = action.payload;
    },
  },
});

export const {
  setAlertProvider,
  setAlertSeverity,
  setAlertType,
  setPriceMarketType,
  setPriceProvider,
  setPriceStaleOnly,
  setPriceSymbol,
} = filtersSlice.actions;

export const filtersReducer = filtersSlice.reducer;
