package publicrest

import (
	"testing"

	"github.com/shopspring/decimal"

	"go-crypto-arb/internal/exchange"
)

var knownAssets = []string{"BTC", "ETH", "USDT", "USD"}

func TestParseVenueTickers(t *testing.T) {
	tests := []struct {
		name  string
		parse func([]byte, []string) ([]tickerUpdate, error)
		data  string
	}{
		{
			name:  "okx",
			parse: parseOKXTickers,
			data:  `{"data":[{"instId":"BTC-USDT","bidPx":"100","askPx":"101","last":"100.5"},{"instId":"DOGE-USDT","bidPx":"1","askPx":"2","last":"1.5"}]}`,
		},
		{
			name:  "bybit",
			parse: parseBybitTickers,
			data:  `{"result":{"list":[{"symbol":"BTCUSDT","bid1Price":"100","ask1Price":"101","lastPrice":"100.5"}]}}`,
		},
		{
			name:  "gateio",
			parse: parseGateIOTickers,
			data:  `[{"currency_pair":"BTC_USDT","highest_bid":"100","lowest_ask":"101","last":"100.5"}]`,
		},
		{
			name:  "bitget",
			parse: parseBitgetTickers,
			data:  `{"data":[{"symbol":"BTCUSDT","bidPr":"100","askPr":"101","lastPr":"100.5"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updates, err := tt.parse([]byte(tt.data), knownAssets)
			if err != nil {
				t.Fatalf("parse tickers: %v", err)
			}
			if len(updates) != 1 {
				t.Fatalf("expected one filtered ticker, got %d", len(updates))
			}
			assertTicker(t, updates[0])
		})
	}
}

func TestParseCoinbaseProductsAndTicker(t *testing.T) {
	products, err := parseCoinbaseProducts([]byte(`[
		{"id":"BTC-USDT","base_currency":"BTC","quote_currency":"USDT","status":"online","trading_disabled":false},
		{"id":"ETH-USD","base_currency":"ETH","quote_currency":"USD","status":"delisted","trading_disabled":false}
	]`), knownAssets)
	if err != nil {
		t.Fatalf("parse products: %v", err)
	}
	if len(products) != 1 || products[0] != "BTC/USDT" {
		t.Fatalf("unexpected products %#v", products)
	}

	update, ok, err := parseCoinbaseTicker([]byte(`{"bid":"100","ask":"101","price":"100.5"}`), "BTC/USDT")
	if err != nil {
		t.Fatalf("parse ticker: %v", err)
	}
	if !ok {
		t.Fatal("expected valid ticker")
	}
	assertTicker(t, update)
}

func TestParseVenueOrderBooks(t *testing.T) {
	tests := []struct {
		name  string
		parse func([]byte) ([]exchange.OrderBookLevel, []exchange.OrderBookLevel, error)
		data  string
	}{
		{
			name:  "okx",
			parse: parseOKXOrderBook,
			data:  `{"data":[{"bids":[["100","1","0","1"]],"asks":[["101","2","0","1"]]}]}`,
		},
		{
			name:  "bybit",
			parse: parseBybitOrderBook,
			data:  `{"result":{"b":[["100","1"]],"a":[["101","2"]]}}`,
		},
		{
			name:  "coinbase",
			parse: parseCoinbaseOrderBook,
			data:  `{"bids":[["100","1",1]],"asks":[["101","2",1]]}`,
		},
		{
			name:  "gateio",
			parse: parseGateIOOrderBook,
			data:  `{"bids":[["100","1"]],"asks":[["101","2"]]}`,
		},
		{
			name:  "bitget",
			parse: parseBitgetOrderBook,
			data:  `{"data":{"bids":[["100","1"]],"asks":[["101","2"]]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bids, asks, err := tt.parse([]byte(tt.data))
			if err != nil {
				t.Fatalf("parse book: %v", err)
			}
			if len(bids) != 1 || len(asks) != 1 {
				t.Fatalf("expected one bid/ask, got bids=%d asks=%d", len(bids), len(asks))
			}
			if !bids[0].Price.Equal(decimal.RequireFromString("100")) || !asks[0].Quantity.Equal(decimal.RequireFromString("2")) {
				t.Fatalf("unexpected levels bids=%#v asks=%#v", bids, asks)
			}
		})
	}
}

func assertTicker(t *testing.T, update tickerUpdate) {
	t.Helper()
	if update.Symbol != "BTC/USDT" {
		t.Fatalf("unexpected symbol %q", update.Symbol)
	}
	if !update.Bid.Equal(decimal.RequireFromString("100")) {
		t.Fatalf("unexpected bid %s", update.Bid)
	}
	if !update.Ask.Equal(decimal.RequireFromString("101")) {
		t.Fatalf("unexpected ask %s", update.Ask)
	}
	if !update.Last.Equal(decimal.RequireFromString("100.5")) {
		t.Fatalf("unexpected last %s", update.Last)
	}
}
