package binance

import (
	"encoding/json"
	"testing"
)

func TestBookTickerPayloadKeepsPricesWhenQuantitiesPresent(t *testing.T) {
	data := []byte(`{
		"s": "BTCUSDT",
		"b": "76888.52000000",
		"B": "2.85900000",
		"a": "76888.53000000",
		"A": "4.25031000"
	}`)

	var payload bookTickerPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if payload.bidPrice() != "76888.52000000" {
		t.Fatalf("expected bid price, got %q", payload.bidPrice())
	}
	if payload.askPrice() != "76888.53000000" {
		t.Fatalf("expected ask price, got %q", payload.askPrice())
	}
}
