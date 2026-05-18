package marketdata

import (
	"sort"
	"strconv"
	"sync"
	"time"

	"go-crypto-arb/internal/alerts"
	"go-crypto-arb/internal/arbitrage"
	"go-crypto-arb/internal/exchange"
)

type Store struct {
	mu sync.RWMutex

	spotTickers    map[string]exchange.Ticker
	futuresTickers map[string]exchange.Ticker
	orderBooks     map[string]exchange.OrderBook
	fundingRates   map[string]exchange.FundingRate
	exchangeHealth map[string]exchange.ExchangeHealth
	markets        map[string]exchange.MarketInfo

	triangular    []arbitrage.TriangularOpportunityV2
	crossExchange []arbitrage.CrossExchangeOpportunityV2
	spotFutures   []arbitrage.SpotFuturesOpportunityV2
	ibkrFX        []arbitrage.TriangularOpportunityV2
	brokerBasis   []arbitrage.BrokerFuturesBasisOpportunity
	related       []arbitrage.RelatedAssetGroupSignal
	alerts        []alerts.Alert
}

type Snapshot struct {
	Version                string                                    `json:"version"`
	Prices                 map[string][]exchange.Ticker              `json:"prices"`
	FuturesPrices          map[string][]exchange.Ticker              `json:"futures_prices"`
	OrderBooks             []exchange.OrderBook                      `json:"order_books"`
	OrderBookSummary       []OrderBookSummary                        `json:"order_book_summary"`
	FundingRates           []exchange.FundingRate                    `json:"funding_rates"`
	Markets                []exchange.MarketInfo                     `json:"markets"`
	TriangularArbitrage    []arbitrage.TriangularOpportunityV2       `json:"triangular_arbitrage"`
	CrossExchangeArbitrage []arbitrage.CrossExchangeOpportunityV2    `json:"cross_exchange_arbitrage"`
	SpotFuturesArbitrage   []arbitrage.SpotFuturesOpportunityV2      `json:"spot_futures_arbitrage"`
	RelatedAssetSignals    []arbitrage.RelatedAssetGroupSignal       `json:"related_asset_signals"`
	IBKRInstruments        []exchange.MarketInfo                     `json:"ibkr_instruments"`
	IBKRFXTriangular       []arbitrage.TriangularOpportunityV2       `json:"ibkr_fx_triangular"`
	CryptoSpotVsIBKRBasis  []arbitrage.BrokerFuturesBasisOpportunity `json:"crypto_spot_vs_ibkr_futures_basis"`
	Alerts                 []alerts.Alert                            `json:"alerts"`
	ExchangeHealth         map[string]exchange.ExchangeHealth        `json:"exchange_health"`
	ProviderHealth         map[string]exchange.ExchangeHealth        `json:"provider_health"`
}

type OrderBookSummary struct {
	Provider   string              `json:"provider"`
	Exchange   string              `json:"exchange"`
	Broker     string              `json:"broker,omitempty"`
	Symbol     string              `json:"symbol"`
	MarketType exchange.MarketType `json:"market_type"`
	BestBid    string              `json:"best_bid"`
	BestAsk    string              `json:"best_ask"`
	BidLevels  int                 `json:"bid_levels"`
	AskLevels  int                 `json:"ask_levels"`
	AgeSeconds string              `json:"age_seconds"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

func NewStore() *Store {
	return &Store{
		spotTickers:    make(map[string]exchange.Ticker),
		futuresTickers: make(map[string]exchange.Ticker),
		orderBooks:     make(map[string]exchange.OrderBook),
		fundingRates:   make(map[string]exchange.FundingRate),
		exchangeHealth: make(map[string]exchange.ExchangeHealth),
		markets:        make(map[string]exchange.MarketInfo),
	}
}

func (s *Store) UpsertSpotTickers(tickers []exchange.Ticker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ticker := range tickers {
		if !exchange.ValidBidAsk(ticker.Bid, ticker.Ask) {
			continue
		}
		s.spotTickers[tickerKey(ticker)] = ticker
	}
}

func (s *Store) UpsertFuturesTickers(tickers []exchange.Ticker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ticker := range tickers {
		if !exchange.ValidBidAsk(ticker.Bid, ticker.Ask) {
			continue
		}
		s.futuresTickers[tickerKey(ticker)] = ticker
	}
}

func (s *Store) UpsertFundingRates(rates []exchange.FundingRate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rate := range rates {
		s.fundingRates[rate.Exchange+"|"+rate.Symbol] = rate
	}
}

func (s *Store) UpsertOrderBooks(orderBooks []exchange.OrderBook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, book := range orderBooks {
		s.orderBooks[orderBookKey(book)] = book
	}
}

func (s *Store) SetMarkets(markets []exchange.MarketInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, market := range markets {
		s.markets[string(market.MarketType)+"|"+market.Exchange+"|"+market.Symbol] = market
	}
}

func (s *Store) SetExchangeHealth(health []exchange.ExchangeHealth) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range health {
		s.exchangeHealth[item.Exchange] = item
	}
}

func (s *Store) SetCalculations(tri []arbitrage.TriangularOpportunityV2, cross []arbitrage.CrossExchangeOpportunityV2, sf []arbitrage.SpotFuturesOpportunityV2, related []arbitrage.RelatedAssetGroupSignal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.triangular = tri
	s.crossExchange = cross
	s.spotFutures = sf
	s.related = related
}

func (s *Store) SetBrokerCalculations(ibkrFX []arbitrage.TriangularOpportunityV2, brokerBasis []arbitrage.BrokerFuturesBasisOpportunity) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ibkrFX = ibkrFX
	s.brokerBasis = brokerBasis
}

func (s *Store) SetAlerts(items []alerts.Alert) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts = items
}

func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return Snapshot{
		Version:                "v2.1.0",
		Prices:                 groupTickers(s.spotTickers),
		FuturesPrices:          groupTickers(s.futuresTickers),
		OrderBooks:             sortedOrderBooks(s.orderBooks),
		OrderBookSummary:       summarizeOrderBooks(s.orderBooks),
		FundingRates:           sortedFundingRates(s.fundingRates),
		Markets:                sortedMarkets(s.markets),
		TriangularArbitrage:    append([]arbitrage.TriangularOpportunityV2(nil), s.triangular...),
		CrossExchangeArbitrage: append([]arbitrage.CrossExchangeOpportunityV2(nil), s.crossExchange...),
		SpotFuturesArbitrage:   append([]arbitrage.SpotFuturesOpportunityV2(nil), s.spotFutures...),
		RelatedAssetSignals:    append([]arbitrage.RelatedAssetGroupSignal(nil), s.related...),
		IBKRInstruments:        filterIBKRMarkets(s.markets),
		IBKRFXTriangular:       append([]arbitrage.TriangularOpportunityV2(nil), s.ibkrFX...),
		CryptoSpotVsIBKRBasis:  append([]arbitrage.BrokerFuturesBasisOpportunity(nil), s.brokerBasis...),
		Alerts:                 append([]alerts.Alert(nil), s.alerts...),
		ExchangeHealth:         cloneHealth(s.exchangeHealth),
		ProviderHealth:         cloneHealth(s.exchangeHealth),
	}
}

func tickerKey(t exchange.Ticker) string {
	return string(t.MarketType) + "|" + t.Exchange + "|" + t.Symbol
}

func orderBookKey(book exchange.OrderBook) string {
	return string(book.MarketType) + "|" + book.Exchange + "|" + book.Symbol
}

func groupTickers(in map[string]exchange.Ticker) map[string][]exchange.Ticker {
	out := make(map[string][]exchange.Ticker)
	for _, ticker := range in {
		out[ticker.Exchange] = append(out[ticker.Exchange], ticker)
	}
	for exchangeName := range out {
		sort.Slice(out[exchangeName], func(i, j int) bool {
			return out[exchangeName][i].Symbol < out[exchangeName][j].Symbol
		})
	}
	return out
}

func sortedFundingRates(in map[string]exchange.FundingRate) []exchange.FundingRate {
	out := make([]exchange.FundingRate, 0, len(in))
	for _, rate := range in {
		out = append(out, rate)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Exchange == out[j].Exchange {
			return out[i].Symbol < out[j].Symbol
		}
		return out[i].Exchange < out[j].Exchange
	})
	return out
}

func sortedOrderBooks(in map[string]exchange.OrderBook) []exchange.OrderBook {
	out := make([]exchange.OrderBook, 0, len(in))
	for _, book := range in {
		out = append(out, book)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Exchange == out[j].Exchange {
			if out[i].MarketType == out[j].MarketType {
				return out[i].Symbol < out[j].Symbol
			}
			return out[i].MarketType < out[j].MarketType
		}
		return out[i].Exchange < out[j].Exchange
	})
	return out
}

func summarizeOrderBooks(in map[string]exchange.OrderBook) []OrderBookSummary {
	books := sortedOrderBooks(in)
	out := make([]OrderBookSummary, 0, len(books))
	now := time.Now()
	for _, book := range books {
		summary := OrderBookSummary{
			Provider:   firstNonEmpty(book.Provider, book.Exchange, book.Broker),
			Exchange:   book.Exchange,
			Broker:     book.Broker,
			Symbol:     book.Symbol,
			MarketType: book.MarketType,
			BidLevels:  len(book.Bids),
			AskLevels:  len(book.Asks),
			UpdatedAt:  book.UpdatedAt,
		}
		if len(book.Bids) > 0 {
			summary.BestBid = book.Bids[0].Price.String()
		}
		if len(book.Asks) > 0 {
			summary.BestAsk = book.Asks[0].Price.String()
		}
		if !book.UpdatedAt.IsZero() {
			summary.AgeSeconds = strconv.FormatFloat(now.Sub(book.UpdatedAt).Seconds(), 'f', 3, 64)
		}
		out = append(out, summary)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func sortedMarkets(in map[string]exchange.MarketInfo) []exchange.MarketInfo {
	out := make([]exchange.MarketInfo, 0, len(in))
	for _, market := range in {
		out = append(out, market)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Exchange == out[j].Exchange {
			if out[i].MarketType == out[j].MarketType {
				return out[i].Symbol < out[j].Symbol
			}
			return out[i].MarketType < out[j].MarketType
		}
		return out[i].Exchange < out[j].Exchange
	})
	return out
}

func cloneHealth(in map[string]exchange.ExchangeHealth) map[string]exchange.ExchangeHealth {
	out := make(map[string]exchange.ExchangeHealth, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func filterIBKRMarkets(in map[string]exchange.MarketInfo) []exchange.MarketInfo {
	var out []exchange.MarketInfo
	for _, market := range in {
		if market.Provider == "ibkr" || market.Broker == "IBKR" || market.Exchange == "IBKR" {
			out = append(out, market)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].AssetClass == out[j].AssetClass {
			return out[i].DisplayName < out[j].DisplayName
		}
		return out[i].AssetClass < out[j].AssetClass
	})
	return out
}
