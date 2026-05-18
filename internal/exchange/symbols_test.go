package exchange

import "testing"

func TestNormalizeCanonicalSymbol(t *testing.T) {
	got := NormalizeCanonicalSymbol("XBT/USD")
	if got != "BTC/USD" {
		t.Fatalf("expected BTC/USD, got %s", got)
	}
}

func TestSplitJoinedSymbol(t *testing.T) {
	base, quote, canonical, ok := SplitJoinedSymbol("BTCUSDT", []string{"BTC", "USDT", "USD"})
	if !ok {
		t.Fatal("expected split to succeed")
	}
	if base != "BTC" || quote != "USDT" || canonical != "BTC/USDT" {
		t.Fatalf("unexpected split: %s %s %s", base, quote, canonical)
	}
}
