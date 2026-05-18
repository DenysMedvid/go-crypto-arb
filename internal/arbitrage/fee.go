package arbitrage

import "github.com/shopspring/decimal"

func ApplyTakerFee(amount decimal.Decimal, fee decimal.Decimal) decimal.Decimal {
	if amount.IsZero() {
		return amount
	}
	return amount.Mul(decimal.NewFromInt(1).Sub(fee))
}

func ProfitPercent(start, end decimal.Decimal) decimal.Decimal {
	if start.IsZero() {
		return decimal.Zero
	}
	return end.Sub(start).Div(start).Mul(decimal.NewFromInt(100))
}

func statusFromPercent(value decimal.Decimal, positive string, nonPositive string) string {
	if value.GreaterThan(decimal.Zero) {
		return positive
	}
	return nonPositive
}
