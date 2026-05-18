package exchange

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestValidBidAskRejectsCrossedBookTicker(t *testing.T) {
	if ValidBidAsk(decimal.RequireFromString("0.97413"), decimal.RequireFromString("0.44122")) {
		t.Fatal("expected crossed bid/ask to be invalid")
	}
}

func TestValidBidAskAcceptsPositiveOrderedTicker(t *testing.T) {
	if !ValidBidAsk(decimal.RequireFromString("100"), decimal.RequireFromString("101")) {
		t.Fatal("expected ordered positive bid/ask to be valid")
	}
}
