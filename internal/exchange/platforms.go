package exchange

import (
	"sort"
	"strings"
)

var supportedCryptoPlatforms = map[string]string{
	"okx":      "OKX",
	"bybit":    "Bybit",
	"binance":  "Binance",
	"kraken":   "Kraken",
	"coinbase": "Coinbase",
	"gateio":   "Gate.io",
	"bitget":   "Bitget",
}

func SupportedCryptoPlatforms() []string {
	out := make([]string, 0, len(supportedCryptoPlatforms))
	for name := range supportedCryptoPlatforms {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func IsSupportedCryptoPlatform(name string) bool {
	_, ok := supportedCryptoPlatforms[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

func DisplayName(name string) string {
	if display, ok := supportedCryptoPlatforms[strings.ToLower(strings.TrimSpace(name))]; ok {
		return display
	}
	return name
}
