package arbitrage

import (
	"github.com/shopspring/decimal"

	"go-crypto-arb/internal/exchange"
)

func SimulateBuyWithQuote(orderBook exchange.OrderBook, quoteAmount decimal.Decimal, feePercent decimal.Decimal) ExecutionSimulation {
	result := ExecutionSimulation{
		Symbol:              orderBook.Symbol,
		Side:                TradeBuy,
		RequestedQuoteValue: quoteAmount,
		FeePercent:          feePercent,
		CompleteFill:        true,
		LimitedDepth:        orderBook.LimitedDepth,
	}
	if quoteAmount.LessThanOrEqual(decimal.Zero) {
		result.Status = "empty_request"
		result.Error = "requested quote amount must be positive"
		return result
	}
	if len(orderBook.Asks) == 0 {
		result.CompleteFill = false
		result.Status = "empty_order_book"
		result.Error = "order book has no asks"
		return result
	}
	result.BestPrice = orderBook.Asks[0].Price
	remainingQuote := quoteAmount
	for _, level := range orderBook.Asks {
		if remainingQuote.LessThanOrEqual(decimal.Zero) {
			break
		}
		if level.Price.LessThanOrEqual(decimal.Zero) || level.Quantity.LessThanOrEqual(decimal.Zero) {
			continue
		}
		levelQuote := level.Price.Mul(level.Quantity)
		quoteAtLevel := decimal.Min(remainingQuote, levelQuote)
		baseAtLevel := quoteAtLevel.Div(level.Price)
		result.FilledBaseQty = result.FilledBaseQty.Add(baseAtLevel)
		result.SpentQuoteValue = result.SpentQuoteValue.Add(quoteAtLevel)
		remainingQuote = remainingQuote.Sub(quoteAtLevel)
	}
	if remainingQuote.GreaterThan(decimal.Zero) {
		result.CompleteFill = false
		result.Status = "partial_fill"
		if result.LimitedDepth {
			result.Error = "limited depth could not fill requested quote amount"
		}
	}
	if result.FilledBaseQty.GreaterThan(decimal.Zero) {
		result.AveragePrice = result.SpentQuoteValue.Div(result.FilledBaseQty)
		result.FeeAmount = result.FilledBaseQty.Mul(feePercent)
		result.FilledBaseQty = result.FilledBaseQty.Sub(result.FeeAmount)
		result.SlippagePercent = percentDiff(result.AveragePrice, result.BestPrice)
	}
	if result.Status == "" {
		result.Status = "ok"
	}
	return result
}

func SimulateSellBase(orderBook exchange.OrderBook, baseAmount decimal.Decimal, feePercent decimal.Decimal) ExecutionSimulation {
	result := ExecutionSimulation{
		Symbol:           orderBook.Symbol,
		Side:             TradeSell,
		RequestedBaseQty: baseAmount,
		FeePercent:       feePercent,
		CompleteFill:     true,
		LimitedDepth:     orderBook.LimitedDepth,
	}
	if baseAmount.LessThanOrEqual(decimal.Zero) {
		result.Status = "empty_request"
		result.Error = "requested base quantity must be positive"
		return result
	}
	if len(orderBook.Bids) == 0 {
		result.CompleteFill = false
		result.Status = "empty_order_book"
		result.Error = "order book has no bids"
		return result
	}
	result.BestPrice = orderBook.Bids[0].Price
	remainingBase := baseAmount
	for _, level := range orderBook.Bids {
		if remainingBase.LessThanOrEqual(decimal.Zero) {
			break
		}
		if level.Price.LessThanOrEqual(decimal.Zero) || level.Quantity.LessThanOrEqual(decimal.Zero) {
			continue
		}
		baseAtLevel := decimal.Min(remainingBase, level.Quantity)
		quoteAtLevel := baseAtLevel.Mul(level.Price)
		result.FilledBaseQty = result.FilledBaseQty.Add(baseAtLevel)
		result.ReceivedQuoteValue = result.ReceivedQuoteValue.Add(quoteAtLevel)
		remainingBase = remainingBase.Sub(baseAtLevel)
	}
	if remainingBase.GreaterThan(decimal.Zero) {
		result.CompleteFill = false
		result.Status = "partial_fill"
		if result.LimitedDepth {
			result.Error = "limited depth could not fill requested base quantity"
		}
	}
	if result.FilledBaseQty.GreaterThan(decimal.Zero) {
		result.AveragePrice = result.ReceivedQuoteValue.Div(result.FilledBaseQty)
		result.FeeAmount = result.ReceivedQuoteValue.Mul(feePercent)
		result.ReceivedQuoteValue = result.ReceivedQuoteValue.Sub(result.FeeAmount)
		result.SlippagePercent = percentDiff(result.BestPrice, result.AveragePrice)
	}
	if result.Status == "" {
		result.Status = "ok"
	}
	return result
}

func percentDiff(a, b decimal.Decimal) decimal.Decimal {
	if b.IsZero() {
		return decimal.Zero
	}
	return a.Sub(b).Div(b).Mul(decimal.NewFromInt(100)).Abs()
}
