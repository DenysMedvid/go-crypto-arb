package exchange

import (
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"
)

func DecimalFromString(value string) (decimal.Decimal, bool) {
	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, false
	}
	return parsed, true
}

func DecimalFromAny(value any) (decimal.Decimal, bool) {
	switch v := value.(type) {
	case string:
		return DecimalFromString(v)
	case json.Number:
		return DecimalFromString(v.String())
	case int:
		return decimal.NewFromInt(int64(v)), true
	case int64:
		return decimal.NewFromInt(v), true
	default:
		return DecimalFromString(fmt.Sprint(value))
	}
}

func ValidBidAsk(bid, ask decimal.Decimal) bool {
	return bid.IsPositive() && ask.IsPositive() && bid.LessThanOrEqual(ask)
}
