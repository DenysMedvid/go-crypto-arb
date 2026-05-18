package kraken

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
)

func TestDepthResponseAcceptsNumericTimestamp(t *testing.T) {
	data := []byte(`{
		"error": [],
		"result": {
			"XXBTZUSD": {
				"asks": [["76860.10000", "0.25", 1779187000]],
				"bids": [["76859.90000", "0.5", 1779187000]]
			}
		}
	}`)

	var payload depthResponse
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal depth response: %v", err)
	}

	depth := payload.Result["XXBTZUSD"]
	asks := parseDepthLevels(depth.Asks)
	bids := parseDepthLevels(depth.Bids)
	if len(asks) != 1 || len(bids) != 1 {
		t.Fatalf("expected one ask and one bid, got asks=%d bids=%d", len(asks), len(bids))
	}
	if !asks[0].Price.Equal(decimal.RequireFromString("76860.10000")) {
		t.Fatalf("unexpected ask price %s", asks[0].Price)
	}
	if !bids[0].Quantity.Equal(decimal.RequireFromString("0.5")) {
		t.Fatalf("unexpected bid quantity %s", bids[0].Quantity)
	}
}

func TestDepthResponseAcceptsNumericPriceAndQuantity(t *testing.T) {
	data := []byte(`{
		"error": [],
		"result": {
			"XXBTZUSD": {
				"asks": [[76860.1, 0.25, 1779187000]],
				"bids": [[76859.9, 0.5, 1779187000]]
			}
		}
	}`)

	var payload depthResponse
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal depth response: %v", err)
	}

	asks := parseDepthLevels(payload.Result["XXBTZUSD"].Asks)
	if len(asks) != 1 {
		t.Fatalf("expected one ask, got %d", len(asks))
	}
	if !asks[0].Price.Equal(decimal.RequireFromString("76860.1")) {
		t.Fatalf("unexpected ask price %s", asks[0].Price)
	}
}

func TestFuturesTickerResponseAcceptsNumericMarketFields(t *testing.T) {
	data := []byte(`{
		"tickers": [{
			"symbol": "PF_XBTUSD",
			"pair": "XBT:USD",
			"bid": 100.1,
			"ask": 100.2,
			"last": 100.15,
			"fundingRate": -0.00001
		}]
	}`)

	var payload futuresTickerResponse
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal futures ticker response: %v", err)
	}
	if len(payload.Tickers) != 1 {
		t.Fatalf("expected one ticker, got %d", len(payload.Tickers))
	}
	ticker := payload.Tickers[0]
	if string(ticker.Bid) != "100.1" {
		t.Fatalf("unexpected bid %q", ticker.Bid)
	}
	if string(ticker.Ask) != "100.2" {
		t.Fatalf("unexpected ask %q", ticker.Ask)
	}
	if string(ticker.Last) != "100.15" {
		t.Fatalf("unexpected last %q", ticker.Last)
	}
	if string(ticker.FundingRate) != "-0.00001" {
		t.Fatalf("unexpected funding rate %q", ticker.FundingRate)
	}
}

func TestFuturesTickerResponseAcceptsLastAlias(t *testing.T) {
	data := []byte(`{
		"tickers": [{
			"symbol": "PF_XBTUSD",
			"pair": "XBT:USD",
			"bid": "100.1",
			"ask": "100.2",
			"la": 100.15
		}]
	}`)

	var payload futuresTickerResponse
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal futures ticker response: %v", err)
	}
	if got := firstNonEmpty(string(payload.Tickers[0].Last), string(payload.Tickers[0].LastAlias)); got != "100.15" {
		t.Fatalf("unexpected last alias %q", got)
	}
}
