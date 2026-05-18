package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/shopspring/decimal"

	"go-crypto-arb/internal/exchange"
)

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(text string) string {
	return ansiEscapePattern.ReplaceAllString(text, "")
}

func TestPriceHighlightStatsRanksBidAndAskAcrossExchanges(t *testing.T) {
	stats := Model{}.priceHighlightStats(map[string][]exchange.Ticker{
		"binance": {
			{Symbol: "BTC/USDT", Bid: decimal.RequireFromString("100"), Ask: decimal.RequireFromString("101")},
		},
		"kraken": {
			{Symbol: "BTC/USDT", Bid: decimal.RequireFromString("99"), Ask: decimal.RequireFromString("102")},
		},
	})

	symbol := exchange.NormalizeCanonicalSymbol("BTC/USDT")
	if got := stats.bidHighlight(symbol, decimal.RequireFromString("100")); got != priceHighlightBest {
		t.Fatalf("expected highest bid to be best, got %v", got)
	}
	if got := stats.bidHighlight(symbol, decimal.RequireFromString("99")); got != priceHighlightWorst {
		t.Fatalf("expected lowest bid to be worst, got %v", got)
	}
	if got := stats.askHighlight(symbol, decimal.RequireFromString("101")); got != priceHighlightBest {
		t.Fatalf("expected lowest ask to be best, got %v", got)
	}
	if got := stats.askHighlight(symbol, decimal.RequireFromString("102")); got != priceHighlightWorst {
		t.Fatalf("expected highest ask to be worst, got %v", got)
	}
}

func TestPriceHighlightStatsDoesNotHighlightFlatPrices(t *testing.T) {
	stats := Model{}.priceHighlightStats(map[string][]exchange.Ticker{
		"binance": {
			{Symbol: "ETH/USDT", Bid: decimal.RequireFromString("2000"), Ask: decimal.RequireFromString("2001")},
		},
		"kraken": {
			{Symbol: "ETH/USDT", Bid: decimal.RequireFromString("2000"), Ask: decimal.RequireFromString("2001")},
		},
	})

	symbol := exchange.NormalizeCanonicalSymbol("ETH/USDT")
	if got := stats.bidHighlight(symbol, decimal.RequireFromString("2000")); got != priceHighlightNone {
		t.Fatalf("expected tied bids to have no highlight, got %v", got)
	}
	if got := stats.askHighlight(symbol, decimal.RequireFromString("2001")); got != priceHighlightNone {
		t.Fatalf("expected tied asks to have no highlight, got %v", got)
	}
}

func TestRenderHealthRowKeepsVisibleColumnsAligned(t *testing.T) {
	header := stripANSI(renderHealthHeader())
	row := stripANSI(renderHealthRow(exchange.ExchangeHealth{
		Exchange:           "binance",
		SpotEnabled:        true,
		FuturesEnabled:     true,
		WebSocketEnabled:   true,
		WebSocketConnected: true,
		DataFresh:          false,
		ReconnectCount:     3,
		Score:              95,
	}))

	expectedHeader := fmt.Sprintf("%-10s %-7s %-8s %-7s %-13s %-10s %-10s %-6s", "Exchange", "Spot", "Futures", "WS", "REST Fallback", "Last Msg", "Reconnects", "Score")
	expectedRow := fmt.Sprintf("%-10s %-7s %-8s %-7s %-13s %-10s %-10d %-6s", "binance", "WARN", "WARN", "OK", "No", "n/a", 3, "95")
	if header != expectedHeader {
		t.Fatalf("expected header %q, got %q", expectedHeader, header)
	}
	if row != expectedRow {
		t.Fatalf("expected row %q, got %q", expectedRow, row)
	}
}

func TestFixedCardWidthDoesNotStretchPastPreferredWidth(t *testing.T) {
	rendered := card("Test", "body", fixedCardWidth(180, priceCardWidth))

	if got := lipgloss.Width(rendered); got != priceCardWidth {
		t.Fatalf("expected card width %d, got %d", priceCardWidth, got)
	}
}

func TestFlowCardsPacksCardsBesideEachOtherAndWraps(t *testing.T) {
	firstCard := card("One", "body", 40)
	secondCard := card("Two", "body", 40)
	thirdCard := card("Three", "body", 40)

	wide := stripANSI(flowCards(90, firstCard, secondCard))
	if got, want := len(strings.Split(wide, "\n")), lipgloss.Height(firstCard); got != want {
		t.Fatalf("expected cards to share one row with %d lines, got %d lines in %q", want, got, wide)
	}

	wrapped := stripANSI(flowCards(90, firstCard, secondCard, thirdCard))
	if got, want := len(strings.Split(wrapped, "\n")), lipgloss.Height(firstCard)*2; got != want {
		t.Fatalf("expected third card to wrap to a second row with %d lines, got %d lines in %q", want, got, wrapped)
	}
}
