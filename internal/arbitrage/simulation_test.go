package arbitrage_test

import (
	"testing"
	"time"

	"go-crypto-arb/internal/arbitrage"
	"go-crypto-arb/internal/exchange"
)

func TestSimulateBuyWithQuoteSingleLevelFullFill(t *testing.T) {
	book := book("Binance", "BTC/USDT", exchange.MarketSpot, [][2]string{{"100", "2"}}, [][2]string{{"101", "2"}})
	got := arbitrage.SimulateBuyWithQuote(book, dec("101"), dec("0"))
	if !got.CompleteFill {
		t.Fatal("expected complete fill")
	}
	if !got.FilledBaseQty.Equal(dec("1")) {
		t.Fatalf("expected 1 BTC, got %s", got.FilledBaseQty)
	}
	if !got.AveragePrice.Equal(dec("101")) {
		t.Fatalf("expected avg 101, got %s", got.AveragePrice)
	}
}

func TestSimulateBuyWithQuoteMultiLevelFullFillAndSlippage(t *testing.T) {
	book := book("Binance", "BTC/USDT", exchange.MarketSpot, [][2]string{{"99", "2"}}, [][2]string{{"100", "1"}, {"110", "1"}})
	got := arbitrage.SimulateBuyWithQuote(book, dec("210"), dec("0"))
	if !got.CompleteFill {
		t.Fatal("expected complete fill")
	}
	if !got.FilledBaseQty.Equal(dec("2")) {
		t.Fatalf("expected 2 base, got %s", got.FilledBaseQty)
	}
	if !got.AveragePrice.Equal(dec("105")) {
		t.Fatalf("expected avg 105, got %s", got.AveragePrice)
	}
	if !got.SlippagePercent.Equal(dec("5.00")) {
		t.Fatalf("expected 5%% slippage, got %s", got.SlippagePercent)
	}
}

func TestSimulateBuyPartialLiquidity(t *testing.T) {
	book := book("Binance", "BTC/USDT", exchange.MarketSpot, nil, [][2]string{{"100", "1"}})
	got := arbitrage.SimulateBuyWithQuote(book, dec("500"), dec("0"))
	if got.CompleteFill {
		t.Fatal("expected partial fill")
	}
	if !got.FilledBaseQty.Equal(dec("1")) {
		t.Fatalf("expected 1 filled, got %s", got.FilledBaseQty)
	}
}

func TestSimulateEmptyOrderBook(t *testing.T) {
	got := arbitrage.SimulateSellBase(exchange.OrderBook{Symbol: "BTC/USDT"}, dec("1"), dec("0"))
	if got.CompleteFill {
		t.Fatal("expected incomplete fill")
	}
	if got.Status != "empty_order_book" {
		t.Fatalf("expected empty_order_book, got %s", got.Status)
	}
}

func TestSimulateSellFeeCalculation(t *testing.T) {
	book := book("Binance", "BTC/USDT", exchange.MarketSpot, [][2]string{{"100", "2"}}, nil)
	got := arbitrage.SimulateSellBase(book, dec("1"), dec("0.01"))
	if !got.ReceivedQuoteValue.Equal(dec("99.00")) {
		t.Fatalf("expected 99 quote after fee, got %s", got.ReceivedQuoteValue)
	}
	if !got.FeeAmount.Equal(dec("1.00")) {
		t.Fatalf("expected quote fee 1, got %s", got.FeeAmount)
	}
}

func book(exchangeName, symbol string, marketType exchange.MarketType, bids [][2]string, asks [][2]string) exchange.OrderBook {
	base, quote := split(symbol)
	out := exchange.OrderBook{
		Exchange:   exchangeName,
		Symbol:     symbol,
		BaseAsset:  base,
		QuoteAsset: quote,
		MarketType: marketType,
		UpdatedAt:  time.Now(),
	}
	for _, row := range bids {
		out.Bids = append(out.Bids, exchange.OrderBookLevel{Price: dec(row[0]), Quantity: dec(row[1])})
	}
	for _, row := range asks {
		out.Asks = append(out.Asks, exchange.OrderBookLevel{Price: dec(row[0]), Quantity: dec(row[1])})
	}
	return exchange.NormalizeOrderBook(out, 0)
}

func split(symbol string) (string, string) {
	for i := 0; i < len(symbol); i++ {
		if symbol[i] == '/' {
			return symbol[:i], symbol[i+1:]
		}
	}
	return symbol, ""
}
