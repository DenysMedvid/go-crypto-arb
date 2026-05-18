import type { components } from './schema';

export type Decimal = components['schemas']['Decimal'];
export type HealthResponse = components['schemas']['HealthResponse'];
export type PricesResponse = components['schemas']['PricesResponse'];
export type Ticker = components['schemas']['Ticker'];
export type FundingRate = components['schemas']['FundingRate'];
export type OrderBook = components['schemas']['OrderBook'];
export type OrderBookSummary = components['schemas']['OrderBookSummary'];
export type MarketInfo = components['schemas']['MarketInfo'];
export type ExchangeHealth = components['schemas']['ExchangeHealth'];
export type HealthMap = components['schemas']['HealthMap'];
export type ProviderResponse = components['schemas']['ProviderResponse'];
export type LegSimulation = components['schemas']['LegSimulation'];
export type TriangularOpportunity = components['schemas']['TriangularOpportunityV2'];
export type CrossExchangeOpportunity = components['schemas']['CrossExchangeOpportunityV2'];
export type SpotFuturesOpportunity = components['schemas']['SpotFuturesOpportunityV2'];
export type BrokerFuturesBasisOpportunity =
  components['schemas']['BrokerFuturesBasisOpportunity'];
export type RelatedAssetGroupSignal = components['schemas']['RelatedAssetGroupSignal'];
export type RelatedAssetSignal = components['schemas']['RelatedAssetSignal'];
export type Alert = components['schemas']['Alert'];
export type Snapshot = components['schemas']['Snapshot'];
export type MarketType = components['schemas']['MarketType'];

export type OrderBooksResponse = OrderBookSummary[] | OrderBook[];
