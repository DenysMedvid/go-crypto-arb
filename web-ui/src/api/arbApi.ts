import {
  createApi,
  fetchBaseQuery,
  type BaseQueryFn,
  type FetchArgs,
  type FetchBaseQueryError,
} from '@reduxjs/toolkit/query/react';

import type {
  Alert,
  BrokerFuturesBasisOpportunity,
  CrossExchangeOpportunity,
  HealthMap,
  HealthResponse,
  MarketInfo,
  OrderBooksResponse,
  PricesResponse,
  ProviderResponse,
  RelatedAssetGroupSignal,
  Snapshot,
  SpotFuturesOpportunity,
  TriangularOpportunity,
} from './types';
import type { RootState } from '../app/store';
import { requestFailed, requestSucceeded } from '../features/apiStatusSlice';
import { describeApiError } from '../utils/apiErrors';

type ApiTag =
  | 'Alerts'
  | 'Arbitrage'
  | 'Health'
  | 'IBKR'
  | 'Prices'
  | 'Providers'
  | 'Signals'
  | 'Snapshot';

const tagTypes: ApiTag[] = [
  'Alerts',
  'Arbitrage',
  'Health',
  'IBKR',
  'Prices',
  'Providers',
  'Signals',
  'Snapshot',
];

const dynamicBaseQuery: BaseQueryFn<string | FetchArgs, unknown, FetchBaseQueryError> = async (
  args,
  api,
  extraOptions,
) => {
  const state = api.getState() as RootState;
  const baseUrl = state.settings.apiBaseUrl.replace(/\/+$/, '');
  const apiKey = state.settings.apiKey;
  const url = typeof args === 'string' ? args : args.url;
  const started = performance.now();
  const rawBaseQuery = fetchBaseQuery({
    baseUrl,
    prepareHeaders: (headers) => {
      if (url.startsWith('/api/v1/') && apiKey) {
        headers.set('X-API-Key', apiKey);
      }
      return headers;
    },
  });

  const result = await rawBaseQuery(args, api, extraOptions);
  const latencyMs = Math.max(0, Math.round(performance.now() - started));
  const at = new Date().toISOString();

  if (result.error) {
    const view = describeApiError(result.error);
    api.dispatch(
      requestFailed({
        url,
        latencyMs,
        at,
        status: view.status,
        message: view.message,
        authFailed: view.authFailed,
        backendUnavailable: view.backendUnavailable,
      }),
    );
  } else {
    api.dispatch(
      requestSucceeded({
        url,
        latencyMs,
        at,
      }),
    );
  }

  return result;
};

export const arbApi = createApi({
  reducerPath: 'arbApi',
  baseQuery: dynamicBaseQuery,
  tagTypes,
  endpoints: (builder) => ({
    getHealth: builder.query<HealthResponse, void>({
      query: () => '/health',
      providesTags: ['Health'],
    }),
    getSnapshot: builder.query<Snapshot, void>({
      query: () => '/api/v1/snapshot',
      providesTags: ['Snapshot'],
    }),
    getPrices: builder.query<PricesResponse, void>({
      query: () => '/api/v1/prices',
      providesTags: ['Prices'],
    }),
    getOrderBooks: builder.query<OrderBooksResponse, void>({
      query: () => '/api/v1/order-books',
      providesTags: ['Prices'],
    }),
    getProviders: builder.query<ProviderResponse[], void>({
      query: () => '/api/v1/providers',
      providesTags: ['Providers'],
    }),
    getProviderHealth: builder.query<HealthMap, void>({
      query: () => '/api/v1/providers/health',
      providesTags: ['Providers', 'Health'],
    }),
    getTriangularArbitrage: builder.query<TriangularOpportunity[], void>({
      query: () => '/api/v1/arbitrage/triangular',
      providesTags: ['Arbitrage'],
    }),
    getCrossExchangeArbitrage: builder.query<CrossExchangeOpportunity[], void>({
      query: () => '/api/v1/arbitrage/cross-exchange',
      providesTags: ['Arbitrage'],
    }),
    getSpotFuturesArbitrage: builder.query<SpotFuturesOpportunity[], void>({
      query: () => '/api/v1/arbitrage/spot-futures',
      providesTags: ['Arbitrage'],
    }),
    getIBKRInstruments: builder.query<MarketInfo[], void>({
      query: () => '/api/v1/ibkr/instruments',
      providesTags: ['IBKR'],
    }),
    getIBKRFXTriangular: builder.query<TriangularOpportunity[], void>({
      query: () => '/api/v1/ibkr/fx-triangular',
      providesTags: ['IBKR', 'Arbitrage'],
    }),
    getIBKRCryptoFuturesBasis: builder.query<BrokerFuturesBasisOpportunity[], void>({
      query: () => '/api/v1/ibkr/crypto-futures-basis',
      providesTags: ['IBKR', 'Arbitrage'],
    }),
    getRelatedAssetSignals: builder.query<RelatedAssetGroupSignal[], void>({
      query: () => '/api/v1/signals/related-assets',
      providesTags: ['Signals'],
    }),
    getAlerts: builder.query<Alert[], void>({
      query: () => '/api/v1/alerts',
      providesTags: ['Alerts'],
    }),
    getExchangeHealth: builder.query<HealthMap, void>({
      query: () => '/api/v1/exchanges/health',
      providesTags: ['Health'],
    }),
  }),
});

export const {
  useGetAlertsQuery,
  useGetCrossExchangeArbitrageQuery,
  useGetExchangeHealthQuery,
  useGetHealthQuery,
  useGetIBKRCryptoFuturesBasisQuery,
  useGetIBKRFXTriangularQuery,
  useGetIBKRInstrumentsQuery,
  useGetOrderBooksQuery,
  useGetPricesQuery,
  useGetProviderHealthQuery,
  useGetProvidersQuery,
  useGetRelatedAssetSignalsQuery,
  useGetSnapshotQuery,
  useGetSpotFuturesArbitrageQuery,
  useGetTriangularArbitrageQuery,
} = arbApi;
